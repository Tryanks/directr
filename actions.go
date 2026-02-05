//go:build windows
// +build windows

package directr

import (
	"errors"

	"github.com/Tryanks/directr/internal/uiautomation"
)

func TryInvoke(element *Element) bool {
	raw := element.rawElement()
	if raw == nil {
		return false
	}
	inv, err := raw.GetInvokePattern()
	if err != nil || inv == nil {
		return false
	}
	defer inv.Release()
	if err := inv.Invoke(); err != nil {
		return false
	}
	return true
}

func Invoke(element *Element) error {
	raw := element.rawElement()
	if raw == nil {
		return errors.New("element is nil")
	}
	inv, err := raw.GetInvokePattern()
	if err != nil || inv == nil {
		return errors.New("element does not support InvokePattern")
	}
	defer inv.Release()
	return inv.Invoke()
}

func Toggle(element *Element) error {
	raw := element.rawElement()
	if raw == nil {
		return errors.New("element is nil")
	}
	toggle, err := raw.GetTogglePattern()
	if err != nil || toggle == nil {
		return errors.New("element does not support TogglePattern")
	}
	defer toggle.Release()
	return toggle.Toggle()
}

func ToggleTo(element *Element, wantOn bool) error {
	raw := element.rawElement()
	if raw == nil {
		return errors.New("element is nil")
	}
	toggle, err := raw.GetTogglePattern()
	if err != nil || toggle == nil {
		return errors.New("element does not support TogglePattern")
	}
	defer toggle.Release()
	state, err := toggle.Get_CurrentToggleState()
	if err != nil {
		return err
	}
	if wantOn && state != uiautomation.ToggleState_On {
		return toggle.Toggle()
	}
	if !wantOn && state != uiautomation.ToggleState_Off {
		return toggle.Toggle()
	}
	return nil
}

func SetValue(element *Element, value string) error {
	raw := element.rawElement()
	if raw == nil {
		return errors.New("element is nil")
	}
	vp, err := raw.GetValuePattern()
	if err != nil || vp == nil {
		return errors.New("element does not support ValuePattern")
	}
	defer vp.Release()
	return vp.SetValue(value)
}

func GetValue(element *Element) (string, error) {
	raw := element.rawElement()
	if raw == nil {
		return "", errors.New("element is nil")
	}
	vp, err := raw.GetValuePattern()
	if err != nil || vp == nil {
		return "", errors.New("element does not support ValuePattern")
	}
	defer vp.Release()
	val, err := vp.Get_CurrentValue()
	if err != nil {
		return "", err
	}
	return val, nil
}

func ExpandBestEffort(element *Element) {
	raw := element.rawElement()
	if raw == nil {
		return
	}
	exp, err := raw.GetExpandCollapsePattern()
	if err != nil || exp == nil {
		return
	}
	_ = exp.Expand()
	exp.Release()
}

func SelectItem(element *Element) error {
	raw := element.rawElement()
	if raw == nil {
		return errors.New("element is nil")
	}
	sel, err := raw.GetSelectionItemPattern()
	if err != nil || sel == nil {
		return errors.New("element does not support SelectionItemPattern")
	}
	defer sel.Release()
	return sel.Select()
}

func Focus(element *Element) error {
	ui := element.uiElement()
	if ui == nil {
		return errors.New("element is nil")
	}
	return ui.SetFocus()
}
