//go:build windows

package main

import (
	"fmt"
	"os"
	"syscall"
	"time"
	"unsafe"

	"github.com/Tryanks/directr/internal/uiautomation"
)

var (
	user32                    = syscall.NewLazyDLL("user32.dll")
	procSetForegroundWindow   = user32.NewProc("SetForegroundWindow")
	procShowWindow            = user32.NewProc("ShowWindow")
	procIsIconic              = user32.NewProc("IsIconic")
	procGetForegroundWindow   = user32.NewProc("GetForegroundWindow")
	procGetWindowThreadProcId = user32.NewProc("GetWindowThreadProcessId")
	procAttachThreadInput     = user32.NewProc("AttachThreadInput")
	procBringWindowToTop      = user32.NewProc("BringWindowToTop")
	procSetWindowPos          = user32.NewProc("SetWindowPos")
	procKeybdEvent            = user32.NewProc("keybd_event")

	kernel32               = syscall.NewLazyDLL("kernel32.dll")
	procGetCurrentThreadId = kernel32.NewProc("GetCurrentThreadId")
)

const (
	swRestore       = 9
	swShow          = 5
	vkMenu          = 0x12
	keyeventfKeyup  = 0x0002
	hwndTopmost     = ^uintptr(0) // -1
	hwndNotopmost   = ^uintptr(1) // -2
	swpNomove       = 0x0002
	swpNosize       = 0x0001
	swpShowwindow   = 0x0040
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: focus-test <window-title>")
		os.Exit(1)
	}
	title := os.Args[1]

	hwnd, err := uiautomation.GetWindowForString("", title)
	if err != nil {
		fmt.Printf("找不到窗口 %q: %v\n", title, err)
		os.Exit(1)
	}
	fmt.Printf("找到窗口: hwnd=0x%X, title=%q\n", hwnd, title)

	// 等2秒让用户看到当前状态
	fmt.Println("2秒后将尝试置顶窗口...")
	time.Sleep(2 * time.Second)

	// 方法1: 如果最小化则恢复
	ret, _, _ := procIsIconic.Call(hwnd)
	if ret != 0 {
		fmt.Println("窗口已最小化，正在恢复...")
		procShowWindow.Call(hwnd, swRestore)
	} else {
		procShowWindow.Call(hwnd, swShow)
	}

	// 方法2: Alt key trick using keybd_event
	fmt.Println("尝试 Alt key trick...")
	procKeybdEvent.Call(uintptr(vkMenu), 0, 0, 0)
	procKeybdEvent.Call(uintptr(vkMenu), 0, uintptr(keyeventfKeyup), 0)

	// 方法3: SetForegroundWindow
	ret, _, err2 := procSetForegroundWindow.Call(hwnd)
	fmt.Printf("SetForegroundWindow result: %d, err: %v\n", ret, err2)

	// 方法4: BringWindowToTop
	ret, _, err2 = procBringWindowToTop.Call(hwnd)
	fmt.Printf("BringWindowToTop result: %d, err: %v\n", ret, err2)

	// 方法5: SetWindowPos TOPMOST then NOTOPMOST
	fmt.Println("尝试 SetWindowPos TOPMOST/NOTOPMOST...")
	ret, _, err2 = procSetWindowPos.Call(hwnd, hwndTopmost, 0, 0, 0, 0, swpNomove|swpNosize|swpShowwindow)
	fmt.Printf("SetWindowPos(TOPMOST) result: %d, err: %v\n", ret, err2)
	ret, _, err2 = procSetWindowPos.Call(hwnd, hwndNotopmost, 0, 0, 0, 0, swpNomove|swpNosize|swpShowwindow)
	fmt.Printf("SetWindowPos(NOTOPMOST) result: %d, err: %v\n", ret, err2)

	// 方法6: AttachThreadInput
	fgHwnd, _, _ := procGetForegroundWindow.Call()
	fmt.Printf("当前前台窗口: 0x%X\n", fgHwnd)

	if fgHwnd != 0 && fgHwnd != hwnd {
		ourThread, _, _ := procGetCurrentThreadId.Call()
		var fgPid uint32
		fgThread, _, _ := procGetWindowThreadProcId.Call(fgHwnd, uintptr(unsafe.Pointer(&fgPid)))
		fmt.Printf("我们的线程: %d, 前台线程: %d\n", ourThread, fgThread)

		if ourThread != fgThread {
			procAttachThreadInput.Call(ourThread, fgThread, 1)
			ret, _, _ = procSetForegroundWindow.Call(hwnd)
			fmt.Printf("AttachThreadInput + SetForegroundWindow result: %d\n", ret)
			procAttachThreadInput.Call(ourThread, fgThread, 0)
		}
	}

	// 最后再检查一下前台窗口
	time.Sleep(200 * time.Millisecond)
	fgHwnd, _, _ = procGetForegroundWindow.Call()
	if fgHwnd == hwnd {
		fmt.Println("✓ 成功! 窗口已在前台。")
	} else {
		fmt.Printf("✗ 失败。当前前台窗口: 0x%X，目标: 0x%X\n", fgHwnd, hwnd)
	}
}
