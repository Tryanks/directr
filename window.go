//go:build windows
// +build windows

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
	wmClose = 0x0010
)

var (
	user32              = syscall.NewLazyDLL("user32.dll")
	procEnumWindows     = user32.NewProc("EnumWindows")
	procGetWindowTextW  = user32.NewProc("GetWindowTextW")
	procGetClassNameW   = user32.NewProc("GetClassNameW")
	procIsWindowVisible = user32.NewProc("IsWindowVisible")
	procPostMessageW    = user32.NewProc("PostMessageW")
)

func ResolveWindowHandle(hwndStr, className, title string) (uintptr, error) {
	if hwndStr != "" {
		parsed, err := strconv.ParseUint(strings.TrimSpace(hwndStr), 0, 64)
		if err != nil {
			return 0, fmt.Errorf("parse --hwnd: %w", err)
		}
		return uintptr(parsed), nil
	}

	hwnd, err := uiautomation.GetWindowForString(className, title)
	if err != nil {
		return 0, fmt.Errorf("find window (class=%q title=%q): %w", className, title, err)
	}
	return hwnd, nil
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

func getClassName(hwnd uintptr) string {
	buf := make([]uint16, 256)
	ret, _, _ := procGetClassNameW.Call(hwnd, uintptr(unsafe.Pointer(&buf[0])), uintptr(len(buf)))
	if ret == 0 {
		return ""
	}
	return syscall.UTF16ToString(buf[:ret])
}
