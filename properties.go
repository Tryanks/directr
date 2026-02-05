//go:build windows
// +build windows

package directr

import "errors"

func Properties(element *Element) (map[string]any, error) {
	ui := element.uiElement()
	if ui == nil {
		return nil, errors.New("element is nil")
	}

	rect := ui.Get_CurrentBoundingRectangle()
	frameworkID, _ := ui.Get_CurrentFrameworkId()
	providerDesc, _ := ui.Get_CurrentProviderDescription()
	localizedType, _ := ui.Get_CurrentLocalizedControlType()

	props := map[string]any{
		"name":           element.Name(),
		"automationId":   element.AutomationId(),
		"className":      element.ClassName(),
		"controlType":    ControlTypeToName(element.ControlType()),
		"isEnabled":      ui.Get_CurrentIsEnabled() != 0,
		"isOffscreen":    ui.Get_CurrentIsOffscreen() != 0,
		"hasFocus":       ui.Get_CurrentHasKeyboardFocus() != 0,
		"boundingRect":   rect,
		"localizedType":  localizedType,
		"processId":      element.ProcessId(),
		"provider":       providerDesc,
		"frameworkId":    frameworkID,
		"isContent":      element.IsContentElement(),
		"isControl":      element.IsControlElement(),
		"isKeyboardable": element.IsKeyboardFocusable(),
	}

	return props, nil
}
