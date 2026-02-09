//go:build windows

package directr

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"
	"unsafe"
)

var (
	procGetWindowThreadProcessId = user32.NewProc("GetWindowThreadProcessId")
	kernel32                     = syscall.NewLazyDLL("kernel32.dll")
	procOpenProcess              = kernel32.NewProc("OpenProcess")
	procCloseHandle              = kernel32.NewProc("CloseHandle")
	procGetProcessTimes          = kernel32.NewProc("GetProcessTimes")
	procQueryFullProcessImage    = kernel32.NewProc("QueryFullProcessImageNameW")
)

var debugWindowTest = strings.TrimSpace(os.Getenv("DIRECTR_DEBUG_WINDOW")) != ""
var stepDelay = readStepDelay()

func readStepDelay() time.Duration {
	raw := strings.TrimSpace(os.Getenv("DIRECTR_STEP_DELAY_MS"))
	if raw == "" {
		return 800 * time.Millisecond
	}
	if v, err := strconv.Atoi(raw); err == nil && v >= 0 {
		return time.Duration(v) * time.Millisecond
	}
	return 800 * time.Millisecond
}

func stepPause(t *testing.T, label string) {
	t.Logf("[step] %s (pause=%s)", label, stepDelay)
	time.Sleep(stepDelay)
}

func waitForWindowByTitleAndClass(titleSubstr string, className string, timeout time.Duration) (uintptr, bool) {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		var found uintptr
		_ = enumWindows(func(hwnd uintptr) bool {
			if !isWindowVisible(hwnd) {
				return true
			}
			if className != "" && !strings.EqualFold(strings.TrimSpace(getClassName(hwnd)), className) {
				return true
			}
			title := strings.TrimSpace(getWindowText(hwnd))
			if titleSubstr != "" && !strings.Contains(title, titleSubstr) {
				return true
			}
			found = hwnd
			return false
		})
		if found != 0 {
			return found, true
		}
		time.Sleep(100 * time.Millisecond)
	}
	return 0, false
}

func findFirstByControlTypeAndName(root *Element, wantType string, wantNames []string) *Element {
	if root == nil {
		return nil
	}
	if ControlTypeToName(root.ControlType()) == wantType {
		name := strings.TrimSpace(root.Name())
		for _, want := range wantNames {
			if want == "" {
				continue
			}
			if strings.Contains(name, want) {
				return root
			}
		}
	}
	for _, child := range root.Children() {
		if found := findFirstByControlTypeAndName(child, wantType, wantNames); found != nil {
			return found
		}
	}
	return nil
}

func clickElementBestEffort(el *Element) error {
	if el == nil {
		return errors.New("element is nil")
	}
	if TryInvoke(el) {
		return nil
	}
	if err := Invoke(el); err == nil {
		return nil
	}
	point, err := ElementCenter(el)
	if err != nil {
		return err
	}
	return MouseClick(point.X, point.Y)
}

func dismissNotepadSavePrompt(t *testing.T, timeout time.Duration) {
	hwnd, ok := waitForWindowByTitleAndClass("记事本", "#32770", timeout)
	if !ok {
		// Try English title fallback.
		hwnd, ok = waitForWindowByTitleAndClass("Notepad", "#32770", timeout)
		if !ok {
			t.Logf("[step] save prompt not found")
			return
		}
	}

	_ = WithUIAutomation(hwnd, func(rootElement *Element) error {
		btn := findFirstByControlTypeAndName(rootElement, "button", []string{
			"不保存", "Don't Save", "Do not save", "Not Save",
		})
		if btn == nil {
			return fmt.Errorf("save prompt button not found")
		}
		if err := clickElementBestEffort(btn); err != nil {
			return err
		}
		return nil
	})
	stepPause(t, "dismissed save prompt")
}

func findFirstByControlType(root *Element, want string) *Element {
	if root == nil {
		return nil
	}
	if ControlTypeToName(root.ControlType()) == want {
		return root
	}
	for _, child := range root.Children() {
		if found := findFirstByControlType(child, want); found != nil {
			return found
		}
	}
	return nil
}

func findFirstEditable(root *Element) *Element {
	if root == nil {
		return nil
	}
	if el := findFirstByControlType(root, "document"); el != nil {
		return el
	}
	if el := findFirstByControlType(root, "edit"); el != nil {
		return el
	}
	if el := findFirstByControlType(root, "textbox"); el != nil {
		return el
	}
	return nil
}

func TestSnapshotTreeRealNotepad(t *testing.T) {
	existing := snapshotVisibleWindows()
	stepPause(t, "captured existing windows")
	cmd := exec.Command("notepad.exe")
	started := time.Now()
	if err := cmd.Start(); err != nil {
		t.Fatalf("start notepad: %v", err)
	}
	if cmd.Process == nil {
		t.Fatalf("notepad process missing")
	}

	hwnd, err := waitForNotepadWindow(cmd.Process.Pid, started, 10*time.Second, existing)
	if err != nil {
		debugDumpWindows("notepad-timeout", existing)
		_ = cmd.Process.Kill()
		t.Fatalf("find notepad window: %v", err)
	}

	defer func() {
		_ = PostClose(hwnd)
		dismissNotepadSavePrompt(t, 4*time.Second)
		_, _ = cmd.Process.Wait()
	}()

	t.Logf("notepad hwnd=%s pid=%d", FormatHwnd(hwnd), cmd.Process.Pid)
	stepPause(t, "notepad window located")

	if windows, err := ListWindows(); err == nil {
		t.Logf("[step] visible windows:")
		for _, win := range windows {
			t.Logf("  %s class=%q title=%q", win.Hwnd, win.Class, win.Title)
		}
	} else {
		t.Logf("list windows failed: %v", err)
	}
	stepPause(t, "printed visible windows")

	var output string
	var state SnapshotState
	done := make(chan error, 1)
	go func() {
		done <- WithUIAutomation(hwnd, func(rootElement *Element) error {
			output, state = SnapshotTree(rootElement, 20, 2000)
			return nil
		})
	}()

	select {
	case err = <-done:
	case <-time.After(20 * time.Second):
		debugDumpWindows("uiautomation-timeout", existing)
		t.Fatalf("snapshot notepad window: timeout after 20s")
	}
	if err != nil {
		t.Fatalf("snapshot notepad window: %v", err)
	}

	if strings.TrimSpace(output) == "" {
		t.Fatalf("expected non-empty snapshot output")
	}
	if !strings.HasPrefix(output, "- window") {
		t.Fatalf("expected snapshot to start with window root, got:\n%s", output)
	}
	t.Logf("[step] initial snapshot:\n%s", output)
	stepPause(t, "printed initial snapshot")

	if err := WithUIAutomation(hwnd, func(rootElement *Element) error {
		editable := findFirstEditable(rootElement)
		if editable == nil {
			return fmt.Errorf("no editable element found")
		}
		if err := Focus(editable); err != nil {
			t.Logf("focus editable failed: %v", err)
		}
		if point, err := ElementCenter(editable); err == nil {
			_ = MouseClick(point.X, point.Y)
		}
		if err := SetValue(editable, "directr test input"); err == nil {
			return nil
		}
		if err := SendKeyChord("Ctrl+A"); err != nil {
			return err
		}
		return TypeText("directr test input")
	}); err != nil {
		t.Logf("input step failed: %v", err)
	}
	stepPause(t, "typed text into notepad")

	var outputAfter string
	if err := WithUIAutomation(hwnd, func(rootElement *Element) error {
		outputAfter, _ = SnapshotTree(rootElement, 20, 2000)
		return nil
	}); err != nil {
		t.Fatalf("snapshot after input: %v", err)
	}
	t.Logf("[step] snapshot after input:\n%s", outputAfter)
	stepPause(t, "printed snapshot after input")

	if state.RootTitle == "" {
		t.Fatalf("expected non-empty root title")
	}
	if ref, ok := state.Refs["e1"]; !ok || len(ref.Path) != 0 {
		t.Fatalf("expected ref e1 at root path, got %+v", ref)
	}
}

func waitForNotepadWindow(pid int, started time.Time, timeout time.Duration, existing map[uintptr]struct{}) (uintptr, error) {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if hwnd, ok := findWindowByPID(pid, existing); ok {
			return hwnd, nil
		}
		time.Sleep(100 * time.Millisecond)
	}
	for time.Now().Before(deadline) {
		if hwnd, ok := findNotepadWindowStartedAfter(started.Add(-2*time.Second), existing); ok {
			return hwnd, nil
		}
		time.Sleep(100 * time.Millisecond)
	}
	return 0, fmt.Errorf("window not found for pid %d within %s", pid, timeout)
}

func findWindowByPID(pid int, existing map[uintptr]struct{}) (uintptr, bool) {
	var found uintptr
	err := enumWindows(func(hwnd uintptr) bool {
		if !isWindowVisible(hwnd) {
			return true
		}
		if _, ok := existing[hwnd]; ok {
			return true
		}
		if getWindowPID(hwnd) != pid {
			return true
		}
		if !windowLooksLikeNotepad(hwnd) {
			return true
		}
		found = hwnd
		return false
	})
	if err != nil || found == 0 {
		return 0, false
	}
	return found, true
}

func findNotepadWindowStartedAfter(started time.Time, existing map[uintptr]struct{}) (uintptr, bool) {
	var found uintptr
	_ = enumWindows(func(hwnd uintptr) bool {
		if !isWindowVisible(hwnd) {
			return true
		}
		if _, ok := existing[hwnd]; ok {
			return true
		}
		if !windowLooksLikeNotepad(hwnd) {
			return true
		}
		pid := getWindowPID(hwnd)
		if pid != 0 {
			if path, err := getProcessImagePath(pid); err == nil {
				if strings.Contains(strings.ToLower(path), "notepad") {
					if createdAt, err := getProcessCreationTime(pid); err == nil && createdAt.Before(started) {
						return true
					}
				}
			}
		}
		found = hwnd
		return false
	})
	if found == 0 {
		return 0, false
	}
	return found, true
}

func snapshotVisibleWindows() map[uintptr]struct{} {
	existing := make(map[uintptr]struct{})
	_ = enumWindows(func(hwnd uintptr) bool {
		if !isWindowVisible(hwnd) {
			return true
		}
		existing[hwnd] = struct{}{}
		return true
	})
	return existing
}

func windowLooksLikeNotepad(hwnd uintptr) bool {
	className := strings.TrimSpace(getClassName(hwnd))
	if strings.EqualFold(className, "Notepad") {
		return true
	}
	title := strings.TrimSpace(getWindowText(hwnd))
	if strings.Contains(strings.ToLower(title), "notepad") {
		return true
	}
	pid := getWindowPID(hwnd)
	if pid == 0 {
		return false
	}
	path, err := getProcessImagePath(pid)
	if err != nil {
		return false
	}
	return strings.Contains(strings.ToLower(path), "notepad")
}

func debugDumpWindows(tag string, existing map[uintptr]struct{}) {
	if !debugWindowTest {
		return
	}
	fmt.Fprintf(os.Stderr, "[directr] window dump (%s)\n", tag)
	_ = enumWindows(func(hwnd uintptr) bool {
		visible := isWindowVisible(hwnd)
		className := strings.TrimSpace(getClassName(hwnd))
		title := strings.TrimSpace(getWindowText(hwnd))
		pid := getWindowPID(hwnd)
		path := ""
		if pid != 0 {
			if p, err := getProcessImagePath(pid); err == nil {
				path = p
			}
		}
		_, wasExisting := existing[hwnd]
		fmt.Fprintf(
			os.Stderr,
			"[directr] hwnd=%s visible=%v existing=%v class=%q title=%q pid=%d path=%q\n",
			FormatHwnd(hwnd),
			visible,
			wasExisting,
			className,
			title,
			pid,
			path,
		)
		return true
	})
}

func getWindowPID(hwnd uintptr) int {
	var pid uint32
	ret, _, _ := procGetWindowThreadProcessId.Call(hwnd, uintptr(unsafe.Pointer(&pid)))
	if ret == 0 {
		return 0
	}
	return int(pid)
}

func openProcessForQuery(pid int) (syscall.Handle, error) {
	handle, _, err := procOpenProcess.Call(
		uintptr(0x1000), // PROCESS_QUERY_LIMITED_INFORMATION
		uintptr(0),
		uintptr(pid),
	)
	if handle == 0 {
		return 0, err
	}
	return syscall.Handle(handle), nil
}

func getProcessImagePath(pid int) (string, error) {
	handle, err := openProcessForQuery(pid)
	if err != nil {
		return "", err
	}
	defer procCloseHandle.Call(uintptr(handle))

	buf := make([]uint16, 1024)
	size := uint32(len(buf))
	ret, _, err := procQueryFullProcessImage.Call(
		uintptr(handle),
		uintptr(0),
		uintptr(unsafe.Pointer(&buf[0])),
		uintptr(unsafe.Pointer(&size)),
	)
	if ret == 0 {
		return "", err
	}
	return syscall.UTF16ToString(buf[:size]), nil
}

type filetime struct {
	lowDateTime  uint32
	highDateTime uint32
}

func filetimeToTime(ft filetime) time.Time {
	const ticksPerSecond = 10000000
	const epochDiff = 11644473600
	ticks := (uint64(ft.highDateTime) << 32) | uint64(ft.lowDateTime)
	sec := int64(ticks / ticksPerSecond)
	nsec := int64(ticks%ticksPerSecond) * 100
	return time.Unix(sec-epochDiff, nsec)
}

func getProcessCreationTime(pid int) (time.Time, error) {
	handle, err := openProcessForQuery(pid)
	if err != nil {
		return time.Time{}, err
	}
	defer procCloseHandle.Call(uintptr(handle))

	var created, exited, kernel, user filetime
	ret, _, err := procGetProcessTimes.Call(
		uintptr(handle),
		uintptr(unsafe.Pointer(&created)),
		uintptr(unsafe.Pointer(&exited)),
		uintptr(unsafe.Pointer(&kernel)),
		uintptr(unsafe.Pointer(&user)),
	)
	if ret == 0 {
		return time.Time{}, err
	}
	return filetimeToTime(created), nil
}
