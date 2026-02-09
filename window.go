//go:build windows

package directr

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"syscall"
	"unsafe"

	"github.com/Tryanks/directr/internal/uiautomation"
)

const (
	wmClose    = 0x0010
	swRestore  = 9
	swShow     = 5
)

var (
	user32                    = syscall.NewLazyDLL("user32.dll")
	procEnumWindows           = user32.NewProc("EnumWindows")
	procGetWindowTextW        = user32.NewProc("GetWindowTextW")
	procGetClassNameW         = user32.NewProc("GetClassNameW")
	procIsWindowVisible       = user32.NewProc("IsWindowVisible")
	procPostMessageW          = user32.NewProc("PostMessageW")
	procSetForegroundWindow   = user32.NewProc("SetForegroundWindow")
	procShowWindow            = user32.NewProc("ShowWindow")
	procIsIconic              = user32.NewProc("IsIconic")
	procGetForegroundWindow   = user32.NewProc("GetForegroundWindow")
	procGetWindowThreadProcId = user32.NewProc("GetWindowThreadProcessId")
	procAttachThreadInput     = user32.NewProc("AttachThreadInput")
	procBringWindowToTop      = user32.NewProc("BringWindowToTop")
	procSetWindowPos          = user32.NewProc("SetWindowPos")
	procKeybdEvent            = user32.NewProc("keybd_event")
	kernel32                  = syscall.NewLazyDLL("kernel32.dll")
	procGetCurrentThreadId    = kernel32.NewProc("GetCurrentThreadId")
)

func ResolveWindowHandle(hwndStr, className, title string) (uintptr, error) {
	if hwndStr != "" {
		parsed, err := strconv.ParseUint(strings.TrimSpace(hwndStr), 0, 64)
		if err != nil {
			return 0, fmt.Errorf("parse --hwnd: %w", err)
		}
		return uintptr(parsed), nil
	}

	// If both class and title are provided exactly, try FindWindowW first
	if className != "" && title != "" {
		hwnd, err := uiautomation.GetWindowForString(className, title)
		if err == nil {
			return hwnd, nil
		}
	}

	// Use EnumWindows for substring matching on title, which is more
	// robust when multiple windows share the same title.
	if title != "" {
		hwnd := findBestWindowByTitle(className, title)
		if hwnd != 0 {
			return hwnd, nil
		}
	}

	// Fallback to FindWindowW for exact class-only match
	if className != "" {
		hwnd, err := uiautomation.GetWindowForString(className, "")
		if err == nil {
			return hwnd, nil
		}
	}

	return 0, fmt.Errorf("find window (class=%q title=%q): not found", className, title)
}

// findBestWindowByTitle enumerates visible windows and returns the best
// match for the given title (substring match). When multiple windows match,
// it prefers the one with a larger bounding rectangle (likely the main window).
func findBestWindowByTitle(className, title string) uintptr {
	type candidate struct {
		hwnd uintptr
		area int64
	}
	var candidates []candidate
	titleLower := strings.ToLower(title)

	enumWindows(func(hwnd uintptr) bool {
		if !isWindowVisible(hwnd) {
			return true
		}
		winTitle := getWindowText(hwnd)
		if !strings.Contains(strings.ToLower(winTitle), titleLower) {
			return true
		}
		if className != "" {
			winClass := getClassName(hwnd)
			if winClass != className {
				return true
			}
		}
		// Get window rectangle to estimate which is the "main" window
		area := getWindowArea(hwnd)
		candidates = append(candidates, candidate{hwnd: hwnd, area: area})
		return true
	})

	if len(candidates) == 0 {
		return 0
	}

	// Pick the candidate with the largest area (most likely the main window)
	best := candidates[0]
	for _, c := range candidates[1:] {
		if c.area > best.area {
			best = c
		}
	}
	return best.hwnd
}

func ResolveWindowFromFlagsOrSession(hwndStr, className, title, sessionPath string) (uintptr, Data, error) {
	var sessionData Data
	if sessionPath != "" {
		sessionData, _ = Load(sessionPath)
	}

	if hwndStr == "" && className == "" && title == "" {
		if sessionData.Window.Hwnd == "" {
			return 0, Data{}, errors.New("no window specified and no window stored in session")
		}
		hwnd, err := ParseHwnd(sessionData.Window.Hwnd)
		return hwnd, sessionData, err
	}

	hwnd, err := ResolveWindowHandle(hwndStr, className, title)
	if err != nil {
		return 0, Data{}, err
	}
	return hwnd, sessionData, nil
}

func WithUIAutomation(hwnd uintptr, fn func(root *Element) error) error {
	uiautomation.CoInitialize()
	defer uiautomation.CoUninitialize()

	instance, err := uiautomation.CreateInstance(
		uiautomation.CLSID_CUIAutomation,
		uiautomation.IID_IUIAutomation,
		uiautomation.CLSCTX_INPROC_SERVER|uiautomation.CLSCTX_LOCAL_SERVER|uiautomation.CLSCTX_REMOTE_SERVER,
	)
	if err != nil {
		return fmt.Errorf("create UIAutomation instance: %w", err)
	}
	ppv := uiautomation.NewIUIAutomation(uiautomation.NewIUnKnown(instance))
	defer ppv.Release()

	root, err := uiautomation.ElementFromHandle(ppv, hwnd)
	if err != nil {
		return fmt.Errorf("get window root element: %w", err)
	}

	rootElement := traverseUIElementTreeNoCache(ppv, root)
	return fn(wrapElement(rootElement))
}

func ListWindows() ([]WindowRef, error) {
	wins := []WindowRef{}
	err := enumWindows(func(hwnd uintptr) bool {
		if !isWindowVisible(hwnd) {
			return true
		}
		title := getWindowText(hwnd)
		className := getClassName(hwnd)
		if strings.TrimSpace(title) == "" && strings.TrimSpace(className) == "" {
			return true
		}
		wins = append(wins, WindowRef{Hwnd: FormatHwnd(hwnd), Class: className, Title: title})
		return true
	})
	return wins, err
}

const (
	hwndTopmost    = ^uintptr(0)  // HWND_TOPMOST (-1)
	hwndNotopmost  = ^uintptr(1)  // HWND_NOTOPMOST (-2)
	swpNomove      = 0x0002
	swpNosize      = 0x0001
	swpShowwindow  = 0x0040
	vkMenu         = 0x12 // VK_MENU (Alt key)
	keyevtfKeyup   = 0x0002
)

// FocusWindow brings the specified window to the foreground.
// It uses multiple techniques to work around Windows' restrictions on
// SetForegroundWindow, which normally only allows the foreground process
// to change the foreground window.
func FocusWindow(hwnd uintptr) error {
	// Step 1: Restore if minimized, otherwise show
	ret, _, _ := procIsIconic.Call(hwnd)
	if ret != 0 {
		procShowWindow.Call(hwnd, swRestore)
	} else {
		procShowWindow.Call(hwnd, swShow)
	}

	// Step 2: Alt key trick — sending a fake Alt keypress via keybd_event
	// temporarily unlocks SetForegroundWindow for background processes.
	procKeybdEvent.Call(uintptr(vkMenu), 0, 0, 0)
	procKeybdEvent.Call(uintptr(vkMenu), 0, uintptr(keyevtfKeyup), 0)

	// Step 3: SetForegroundWindow
	procSetForegroundWindow.Call(hwnd)

	// Step 4: BringWindowToTop as backup
	procBringWindowToTop.Call(hwnd)

	// Step 5: SetWindowPos TOPMOST then NOTOPMOST — brings to Z-order top
	// without keeping always-on-top permanently.
	procSetWindowPos.Call(hwnd, hwndTopmost, 0, 0, 0, 0, swpNomove|swpNosize|swpShowwindow)
	procSetWindowPos.Call(hwnd, hwndNotopmost, 0, 0, 0, 0, swpNomove|swpNosize|swpShowwindow)

	// Step 6: If still not foreground, try AttachThreadInput approach
	fgHwnd, _, _ := procGetForegroundWindow.Call()
	if fgHwnd != 0 && fgHwnd != hwnd {
		ourThread, _, _ := procGetCurrentThreadId.Call()
		var fgPid uint32
		fgThread, _, _ := procGetWindowThreadProcId.Call(fgHwnd, uintptr(unsafe.Pointer(&fgPid)))
		if ourThread != fgThread {
			procAttachThreadInput.Call(ourThread, fgThread, 1)
			procSetForegroundWindow.Call(hwnd)
			procAttachThreadInput.Call(ourThread, fgThread, 0)
		}
	}

	return nil
}

func PostClose(hwnd uintptr) error {
	ret, _, err := procPostMessageW.Call(hwnd, wmClose, 0, 0)
	if ret == 0 {
		return fmt.Errorf("PostMessage(WM_CLOSE) failed: %w", err)
	}
	return nil
}

func ParseHwnd(value string) (uintptr, error) {
	if value == "" {
		return 0, errors.New("empty hwnd")
	}
	parsed, err := strconv.ParseUint(strings.TrimSpace(value), 0, 64)
	if err != nil {
		return 0, fmt.Errorf("parse hwnd: %w", err)
	}
	return uintptr(parsed), nil
}

func FormatHwnd(hwnd uintptr) string {
	return fmt.Sprintf("0x%X", hwnd)
}

func enumWindows(cb func(hwnd uintptr) bool) error {
	callback := syscall.NewCallback(func(hwnd uintptr, lparam uintptr) uintptr {
		if cb(hwnd) {
			return 1
		}
		return 0
	})

	ret, _, err := procEnumWindows.Call(callback, 0)
	if ret == 0 {
		if err == syscall.Errno(0) {
			return nil
		}
		return fmt.Errorf("EnumWindows failed: %w", err)
	}
	return nil
}

func isWindowVisible(hwnd uintptr) bool {
	ret, _, _ := procIsWindowVisible.Call(hwnd)
	return ret != 0
}

func getWindowText(hwnd uintptr) string {
	buf := make([]uint16, 512)
	ret, _, _ := procGetWindowTextW.Call(hwnd, uintptr(unsafe.Pointer(&buf[0])), uintptr(len(buf)))
	if ret == 0 {
		return ""
	}
	return syscall.UTF16ToString(buf[:ret])
}

var procGetWindowRect = user32.NewProc("GetWindowRect")

type winRect struct {
	Left, Top, Right, Bottom int32
}

func getWindowArea(hwnd uintptr) int64 {
	var rect winRect
	ret, _, _ := procGetWindowRect.Call(hwnd, uintptr(unsafe.Pointer(&rect)))
	if ret == 0 {
		return 0
	}
	w := int64(rect.Right - rect.Left)
	h := int64(rect.Bottom - rect.Top)
	if w <= 0 || h <= 0 {
		return 0
	}
	return w * h
}

func getClassName(hwnd uintptr) string {
	buf := make([]uint16, 256)
	ret, _, _ := procGetClassNameW.Call(hwnd, uintptr(unsafe.Pointer(&buf[0])), uintptr(len(buf)))
	if ret == 0 {
		return ""
	}
	return syscall.UTF16ToString(buf[:ret])
}
