//go:build windows
// +build windows

package flow

import directr "github.com/Tryanks/directr"

func (s Selector) toDirectr() *directr.ElementSelector {
	return &directr.ElementSelector{
		Ref:          s.Ref,
		Name:         s.Name,
		AutomationId: s.AutomationID,
		ClassName:    s.ClassName,
		ControlType:  s.ControlType,
	}
}
