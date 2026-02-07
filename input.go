//go:build windows
// +build windows

package directr

import (
	"errors"
	"fmt"
	"strings"
	"time"
	"unsafe"
)

const (
	inputKeyboard    = 1
	keyeventfKeyup   = 0x0002
	keyeventfUnicode = 0x0004
)

const (
	mouseeventfLeftDown  = 0x0002
	mouseeventfLeftUp    = 0x0004
	mouseeventfRightDown = 0x0008
	mouseeventfRightUp   = 0x0010
)

const (
	vkControl  = 0x11
	vkAlt      = 0x12
	vkShift    = 0x10
	vkWin      = 0x5B
	vkReturn   = 0x0D
	vkTab      = 0x09
	vkEscape   = 0x1B
	vkBack     = 0x08
	vkDelete   = 0x2E
	vkLeft     = 0x25
	vkRight    = 0x27
	vkUp       = 0x26
	vkDown     = 0x28
	vkHome     = 0x24
	vkEnd      = 0x23
	vkPageUp   = 0x21
	vkPageDown = 0x22
	vkSpace    = 0x20
)

var (
	procSetCursorPos = user32.NewProc("SetCursorPos")
	procMouseEvent   = user32.NewProc("mouse_event")
	procSendInput    = user32.NewProc("SendInput")
)

type Point struct {
	X int32
	Y int32
}

type keyInput struct {
	mods  []uint16
	key   uint16
	text  rune
	isKey bool
}

type keybdInput struct {
	Vk        uint16
	Scan      uint16
	Flags     uint32
	Time      uint32
	ExtraInfo uintptr
}

type winInput struct {
	Type uint32
	Ki   keybdInput
	// Padding to match Windows INPUT struct size (40 bytes on 64-bit).
	// The C union in INPUT is sized to MOUSEINPUT (32 bytes), but keybdInput
	// is only 24 bytes; this extra 8-byte pad accounts for the difference.
	_pad [8]byte
}

func ElementCenter(element *Element) (*Point, error) {
	ui := element.uiElement()
	if ui == nil {
		return nil, errors.New("element is nil")
	}
	rect := ui.Get_CurrentBoundingRectangle()
	if rect == nil {
		return nil, errors.New("no bounding rectangle")
	}
	if rect.Right <= rect.Left || rect.Bottom <= rect.Top {
		return nil, errors.New("invalid bounding rectangle")
	}
	x := (rect.Left + rect.Right) / 2
	y := (rect.Top + rect.Bottom) / 2
	return &Point{X: x, Y: y}, nil
}

func MouseClick(x, y int32) error {
	if err := SetCursor(x, y); err != nil {
		return err
	}
	procMouseEvent.Call(mouseeventfLeftDown, 0, 0, 0, 0)
	procMouseEvent.Call(mouseeventfLeftUp, 0, 0, 0, 0)
	return nil
}

func MouseDoubleClick(x, y int32) error {
	if err := MouseClick(x, y); err != nil {
		return err
	}
	time.Sleep(50 * time.Millisecond)
	return MouseClick(x, y)
}

func MouseDrag(fromX, fromY, toX, toY int32) error {
	if err := SetCursor(fromX, fromY); err != nil {
		return err
	}
	procMouseEvent.Call(mouseeventfLeftDown, 0, 0, 0, 0)
	time.Sleep(50 * time.Millisecond)
	if err := SetCursor(toX, toY); err != nil {
		return err
	}
	procMouseEvent.Call(mouseeventfLeftUp, 0, 0, 0, 0)
	return nil
}

func SetCursor(x, y int32) error {
	ret, _, err := procSetCursorPos.Call(uintptr(x), uintptr(y))
	if ret == 0 {
		return fmt.Errorf("set cursor failed: %w", err)
	}
	return nil
}

func SendKeyChord(sequence string) error {
	parsed, err := parseKeyInput(sequence)
	if err != nil {
		return err
	}
	if !parsed.isKey {
		return TypeText(string(parsed.text))
	}

	for _, mod := range parsed.mods {
		if err := sendVirtualKey(mod, false); err != nil {
			return err
		}
	}
	if err := sendVirtualKey(parsed.key, false); err != nil {
		return err
	}
	if err := sendVirtualKey(parsed.key, true); err != nil {
		return err
	}
	for i := len(parsed.mods) - 1; i >= 0; i-- {
		if err := sendVirtualKey(parsed.mods[i], true); err != nil {
			return err
		}
	}
	return nil
}

func TypeText(text string) error {
	inputs := make([]winInput, 0, len([]rune(text))*2)
	for _, r := range text {
		inputs = append(inputs, winInput{Type: inputKeyboard, Ki: keybdInput{Scan: uint16(r), Flags: keyeventfUnicode}})
		inputs = append(inputs, winInput{Type: inputKeyboard, Ki: keybdInput{Scan: uint16(r), Flags: keyeventfUnicode | keyeventfKeyup}})
	}
	return sendInput(inputs)
}

func parseKeyInput(sequence string) (*keyInput, error) {
	sequence = strings.TrimSpace(sequence)
	if sequence == "" {
		return nil, errors.New("empty key sequence")
	}

	parts := strings.Split(sequence, "+")
	if len(parts) == 1 {
		if len([]rune(sequence)) == 1 {
			r := []rune(sequence)[0]
			return &keyInput{text: r, isKey: false}, nil
		}
	}

	mods := []uint16{}
	var key uint16

	for i, part := range parts {
		p := strings.ToLower(strings.TrimSpace(part))
		switch p {
		case "ctrl", "control":
			mods = append(mods, vkControl)
		case "alt":
			mods = append(mods, vkAlt)
		case "shift":
			mods = append(mods, vkShift)
		case "meta", "win", "windows":
			mods = append(mods, vkWin)
		default:
			if i != len(parts)-1 {
				return nil, fmt.Errorf("unknown modifier %q", part)
			}
			vk, ok := keyNameToVK(p)
			if !ok {
				if len([]rune(part)) == 1 {
					r := []rune(part)[0]
					return &keyInput{mods: mods, text: r, isKey: false}, nil
				}
				return nil, fmt.Errorf("unknown key %q", part)
			}
			key = vk
		}
	}

	if key == 0 {
		return nil, errors.New("missing key")
	}
	return &keyInput{mods: mods, key: key, isKey: true}, nil
}

func keyNameToVK(name string) (uint16, bool) {
	switch name {
	case "enter", "return":
		return vkReturn, true
	case "tab":
		return vkTab, true
	case "escape", "esc":
		return vkEscape, true
	case "backspace", "back":
		return vkBack, true
	case "delete", "del":
		return vkDelete, true
	case "left":
		return vkLeft, true
	case "right":
		return vkRight, true
	case "up":
		return vkUp, true
	case "down":
		return vkDown, true
	case "home":
		return vkHome, true
	case "end":
		return vkEnd, true
	case "pageup":
		return vkPageUp, true
	case "pagedown":
		return vkPageDown, true
	case "space", "spacebar":
		return vkSpace, true
	default:
		if len(name) == 1 {
			r := []rune(strings.ToUpper(name))[0]
			if r >= 'A' && r <= 'Z' {
				return uint16(r), true
			}
			if r >= '0' && r <= '9' {
				return uint16(r), true
			}
		}
	}
	return 0, false
}

func sendVirtualKey(vk uint16, keyUp bool) error {
	flags := uint32(0)
	if keyUp {
		flags = keyeventfKeyup
	}
	input := winInput{
		Type: inputKeyboard,
		Ki:   keybdInput{Vk: vk, Scan: 0, Flags: flags},
	}
	return sendInput([]winInput{input})
}

func sendInput(inputs []winInput) error {
	if len(inputs) == 0 {
		return nil
	}
	ret, _, err := procSendInput.Call(
		uintptr(len(inputs)),
		uintptr(unsafe.Pointer(&inputs[0])),
		unsafe.Sizeof(inputs[0]),
	)
	if ret == 0 {
		return fmt.Errorf("SendInput failed: %w", err)
	}
	return nil
}
