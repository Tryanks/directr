//go:build windows

package directr

import "github.com/Tryanks/directr/internal/uiautomation"

type Element struct {
	raw *uiautomation.Element
}

func wrapElement(raw *uiautomation.Element) *Element {
	if raw == nil {
		return nil
	}
	return &Element{raw: raw}
}

func (e *Element) AutomationId() string {
	if e == nil || e.raw == nil {
		return ""
	}
	return e.raw.CurrentAutomationId
}

func (e *Element) ClassName() string {
	if e == nil || e.raw == nil {
		return ""
	}
	return e.raw.CurrentClassName
}

func (e *Element) ControlType() ControlTypeId {
	if e == nil || e.raw == nil {
		return 0
	}
	return ControlTypeId(e.raw.CurrentControlType)
}

func (e *Element) Name() string {
	if e == nil || e.raw == nil {
		return ""
	}
	return e.raw.CurrentName
}

func (e *Element) ProcessId() int32 {
	if e == nil || e.raw == nil {
		return 0
	}
	return e.raw.CurrentProcessId
}

func (e *Element) IsContentElement() bool {
	if e == nil || e.raw == nil {
		return false
	}
	return e.raw.CurrentIsContentElement != 0
}

func (e *Element) IsControlElement() bool {
	if e == nil || e.raw == nil {
		return false
	}
	return e.raw.CurrentIsControlElement != 0
}

func (e *Element) IsKeyboardFocusable() bool {
	if e == nil || e.raw == nil {
		return false
	}
	return e.raw.CurrentIsKeyboardFocusable != 0
}

func (e *Element) IsEnabled() bool {
	if e == nil || e.raw == nil {
		return false
	}
	return e.raw.CurrentIsEnabled != 0
}

func (e *Element) IsOffscreen() bool {
	if e == nil || e.raw == nil {
		return false
	}
	return e.raw.CurrentIsOffscreen != 0
}

func (e *Element) HasKeyboardFocus() bool {
	if e == nil || e.raw == nil {
		return false
	}
	return e.raw.CurrentHasKeyboardFocus != 0
}

func (e *Element) Children() []*Element {
	if e == nil || e.raw == nil || len(e.raw.Child) == 0 {
		return nil
	}
	children := make([]*Element, len(e.raw.Child))
	for i, child := range e.raw.Child {
		children[i] = wrapElement(child)
	}
	return children
}

func (e *Element) uiElement() *uiautomation.IUIAutomationElement {
	if e == nil || e.raw == nil {
		return nil
	}
	return e.raw.UIAutoElement
}

func (e *Element) rawElement() *uiautomation.Element {
	if e == nil {
		return nil
	}
	return e.raw
}
