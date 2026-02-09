//go:build windows
// +build windows

package directr

import (
	"fmt"
	"time"
)

// BatchAction describes a single UI action to execute as part of a batch.
type BatchAction struct {
	Action       string `json:"action"`                 // click, dblclick, fill, type, press, hover, invoke, toggle, check, uncheck, select, drag, sleep
	Ref          string `json:"ref,omitempty"`           // snapshot ref (e1, e2, ...)
	AutomationId string `json:"automationId,omitempty"` // UIA AutomationId
	Name         string `json:"name,omitempty"`          // element name
	ClassName    string `json:"className,omitempty"`     // UIA ClassName
	ControlType  string `json:"controlType,omitempty"`   // control type
	Value        string `json:"value,omitempty"`         // value for fill/type/press/select/sleep (milliseconds for sleep)
	ToRef        string `json:"toRef,omitempty"`         // drag target ref
	ToAutoId     string `json:"toAutomationId,omitempty"` // drag target automationId
	ToName       string `json:"toName,omitempty"`        // drag target name
}

// BatchResult reports the outcome of a single action in a batch.
type BatchResult struct {
	Index  int    `json:"index"`
	Action string `json:"action"`
	OK     bool   `json:"ok"`
	Error  string `json:"error,omitempty"`
}

// ExecuteBatch runs a sequence of actions against the given root element inside
// a single UIA session.  Each action is executed independently; a failure in one
// action does not stop later actions.
func ExecuteBatch(root *Element, actions []BatchAction, sessionData Data) []BatchResult {
	results := make([]BatchResult, len(actions))
	for i, act := range actions {
		results[i] = BatchResult{Index: i, Action: act.Action}
		err := executeSingleAction(root, act, sessionData)
		if err != nil {
			results[i].Error = err.Error()
		} else {
			results[i].OK = true
		}
	}
	return results
}

func selectorFromBatchAction(act BatchAction) *ElementSelector {
	return &ElementSelector{
		Ref:          act.Ref,
		AutomationId: act.AutomationId,
		Name:         act.Name,
		ClassName:    act.ClassName,
		ControlType:  act.ControlType,
	}
}

func executeSingleAction(root *Element, act BatchAction, sessionData Data) error {
	switch act.Action {
	case "click":
		return executeBatchClick(root, act, sessionData, false)
	case "dblclick":
		return executeBatchClick(root, act, sessionData, true)
	case "fill":
		return executeBatchFill(root, act, sessionData)
	case "type":
		return executeBatchType(root, act, sessionData)
	case "press":
		return executeBatchPress(root, act, sessionData)
	case "hover":
		return executeBatchHover(root, act, sessionData)
	case "invoke":
		return executeBatchInvoke(root, act, sessionData)
	case "toggle":
		return executeBatchToggle(root, act, sessionData)
	case "check":
		return executeBatchCheck(root, act, sessionData, true)
	case "uncheck":
		return executeBatchCheck(root, act, sessionData, false)
	case "select":
		return executeBatchSelect(root, act, sessionData)
	case "drag":
		return executeBatchDrag(root, act, sessionData)
	case "sleep":
		return executeBatchSleep(act)
	default:
		return fmt.Errorf("unknown action %q", act.Action)
	}
}

func executeBatchClick(root *Element, act BatchAction, sessionData Data, double bool) error {
	sel := selectorFromBatchAction(act)
	element, err := ResolveElement(root, sel, sessionData)
	if err != nil {
		return err
	}
	if !double && TryInvoke(element) {
		return nil
	}
	point, err := ElementCenter(element)
	if err != nil {
		return err
	}
	if double {
		return MouseDoubleClick(point.X, point.Y)
	}
	return MouseClick(point.X, point.Y)
}

func executeBatchFill(root *Element, act BatchAction, sessionData Data) error {
	if act.Value == "" {
		return fmt.Errorf("fill requires a value")
	}
	sel := selectorFromBatchAction(act)
	element, err := ResolveElement(root, sel, sessionData)
	if err != nil {
		return err
	}
	if err := SetValue(element, act.Value); err == nil {
		return nil
	}
	if err := Focus(element); err != nil {
		return err
	}
	if err := SendKeyChord("Ctrl+A"); err != nil {
		return err
	}
	return TypeText(act.Value)
}

func executeBatchType(root *Element, act BatchAction, sessionData Data) error {
	if act.Value == "" {
		return fmt.Errorf("type requires a value")
	}
	sel := selectorFromBatchAction(act)
	if sel.Ref != "" || sel.AutomationId != "" || sel.Name != "" || sel.ClassName != "" || sel.ControlType != "" {
		element, err := ResolveElement(root, sel, sessionData)
		if err != nil {
			return err
		}
		if err := Focus(element); err == nil {
			return TypeText(act.Value)
		}
		point, err := ElementCenter(element)
		if err != nil {
			return err
		}
		if err := MouseClick(point.X, point.Y); err != nil {
			return err
		}
	}
	return TypeText(act.Value)
}

func executeBatchPress(root *Element, act BatchAction, sessionData Data) error {
	if act.Value == "" {
		return fmt.Errorf("press requires a key")
	}
	sel := selectorFromBatchAction(act)
	if sel.Ref != "" || sel.AutomationId != "" || sel.Name != "" || sel.ClassName != "" || sel.ControlType != "" {
		element, err := ResolveElement(root, sel, sessionData)
		if err != nil {
			return err
		}
		if err := Focus(element); err != nil {
			return err
		}
	}
	return SendKeyChord(act.Value)
}

func executeBatchHover(root *Element, act BatchAction, sessionData Data) error {
	sel := selectorFromBatchAction(act)
	element, err := ResolveElement(root, sel, sessionData)
	if err != nil {
		return err
	}
	point, err := ElementCenter(element)
	if err != nil {
		return err
	}
	return SetCursor(point.X, point.Y)
}

func executeBatchInvoke(root *Element, act BatchAction, sessionData Data) error {
	sel := selectorFromBatchAction(act)
	element, err := ResolveElement(root, sel, sessionData)
	if err != nil {
		return err
	}
	return Invoke(element)
}

func executeBatchToggle(root *Element, act BatchAction, sessionData Data) error {
	sel := selectorFromBatchAction(act)
	element, err := ResolveElement(root, sel, sessionData)
	if err != nil {
		return err
	}
	return Toggle(element)
}

func executeBatchCheck(root *Element, act BatchAction, sessionData Data, wantOn bool) error {
	sel := selectorFromBatchAction(act)
	element, err := ResolveElement(root, sel, sessionData)
	if err != nil {
		return err
	}
	if err := ToggleTo(element, wantOn); err == nil {
		return nil
	}
	return Invoke(element)
}

func executeBatchSelect(root *Element, act BatchAction, sessionData Data) error {
	if act.Value == "" {
		return fmt.Errorf("select requires an option value")
	}
	sel := selectorFromBatchAction(act)
	target, err := ResolveElement(root, sel, sessionData)
	if err != nil {
		return err
	}
	ExpandBestEffort(target)
	optionEl := FindFirst(target, func(el *Element) bool {
		return el.Name() == act.Value
	})
	if optionEl == nil {
		return fmt.Errorf("option %q not found", act.Value)
	}
	return SelectItem(optionEl)
}

func executeBatchDrag(root *Element, act BatchAction, sessionData Data) error {
	fromSel := selectorFromBatchAction(act)
	fromEl, err := ResolveElement(root, fromSel, sessionData)
	if err != nil {
		return fmt.Errorf("drag source: %w", err)
	}

	toSel := &ElementSelector{
		Ref:          act.ToRef,
		AutomationId: act.ToAutoId,
		Name:         act.ToName,
	}
	toEl, err := ResolveElement(root, toSel, sessionData)
	if err != nil {
		return fmt.Errorf("drag target: %w", err)
	}

	fromPoint, err := ElementCenter(fromEl)
	if err != nil {
		return err
	}
	toPoint, err := ElementCenter(toEl)
	if err != nil {
		return err
	}

	// Small delay between actions for UI responsiveness
	time.Sleep(50 * time.Millisecond)
	return MouseDrag(fromPoint.X, fromPoint.Y, toPoint.X, toPoint.Y)
}

func executeBatchSleep(act BatchAction) error {
	if act.Value == "" {
		return fmt.Errorf("sleep requires a duration value in milliseconds")
	}
	var durationMs int
	if _, err := fmt.Sscanf(act.Value, "%d", &durationMs); err != nil {
		return fmt.Errorf("sleep duration must be a number in milliseconds: %w", err)
	}
	if durationMs < 0 {
		return fmt.Errorf("sleep duration must be non-negative")
	}
	time.Sleep(time.Duration(durationMs) * time.Millisecond)
	return nil
}
