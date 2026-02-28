package flow

import (
	"errors"
	"fmt"
	"maps"
	"slices"
	"time"
)

type Definition struct {
	Name     string           `json:"name,omitzero"`
	Initial  string           `json:"initial,omitzero"`
	Target   string           `json:"target"`
	MaxSteps int              `json:"maxSteps,omitzero"`
	States   map[string]State `json:"states"`
}

type State struct {
	Name        string       `json:"name,omitzero"`
	Scene       Scene        `json:"scene"`
	Transitions []Transition `json:"transitions,omitzero"`
}

type Scene struct {
	Required  []Selector         `json:"required,omitzero"`
	Forbidden []Selector         `json:"forbidden,omitzero"`
	Optional  []WeightedSelector `json:"optional,omitzero"`
	MinScore  float64            `json:"minScore,omitzero"`
}

type WeightedSelector struct {
	Selector Selector `json:"selector"`
	Weight   float64  `json:"weight,omitzero"`
}

type Selector struct {
	Ref          string `json:"ref,omitzero"`
	Name         string `json:"name,omitzero"`
	AutomationID string `json:"automationId,omitzero"`
	ClassName    string `json:"className,omitzero"`
	ControlType  string `json:"controlType,omitzero"`
}

type ActionType string

const (
	ActionNone        ActionType = "none"
	ActionClick       ActionType = "click"
	ActionDoubleClick ActionType = "double_click"
	ActionInvoke      ActionType = "invoke"
	ActionFill        ActionType = "fill"
	ActionFocus       ActionType = "focus"
	ActionToggle      ActionType = "toggle"
	ActionToggleTo    ActionType = "toggle_to"
	ActionSelect      ActionType = "select"
)

type Action struct {
	Type     ActionType `json:"type"`
	Selector Selector   `json:"selector,omitzero"`
	Value    string     `json:"value,omitzero"`
	WantOn   bool       `json:"wantOn,omitzero"`
}

type Transition struct {
	To     string `json:"to"`
	Action Action `json:"action"`
	WaitMS int    `json:"waitMs,omitzero"`
}

type Options struct {
	MaxSteps      int           `json:"maxSteps,omitzero"`
	PollInterval  time.Duration `json:"pollInterval,omitzero"`
	SnapshotDepth int           `json:"snapshotDepth,omitzero"`
	SnapshotNodes int           `json:"snapshotNodes,omitzero"`
}

type StepRecord struct {
	Index        int       `json:"index"`
	At           time.Time `json:"at,omitzero"`
	Detected     string    `json:"detected,omitzero"`
	Score        float64   `json:"score,omitzero"`
	Action       string    `json:"action,omitzero"`
	TransitionTo string    `json:"transitionTo,omitzero"`
	Error        string    `json:"error,omitzero"`
}

type Result struct {
	Flow      string       `json:"flow,omitzero"`
	Success   bool         `json:"success"`
	StartedAt time.Time    `json:"startedAt,omitzero"`
	EndedAt   time.Time    `json:"endedAt,omitzero"`
	LastState string       `json:"lastState,omitzero"`
	Steps     []StepRecord `json:"steps,omitzero"`
}

func (d *Definition) Normalize() {
	if d == nil {
		return
	}
	if d.Name == "" {
		d.Name = "flow"
	}
	if len(d.States) == 0 {
		return
	}
	for name := range maps.Keys(d.States) {
		state := d.States[name]
		if state.Name == "" {
			state.Name = name
			d.States[name] = state
		}
	}
	if d.Initial == "" {
		names := slices.Sorted(maps.Keys(d.States))
		d.Initial = names[0]
	}
}

func (d Definition) Validate() error {
	if len(d.States) == 0 {
		return errors.New("states is required")
	}
	if d.Initial == "" {
		return errors.New("initial state is required")
	}
	if d.Target == "" {
		return errors.New("target state is required")
	}
	if _, ok := d.States[d.Initial]; !ok {
		return fmt.Errorf("initial state %q not found", d.Initial)
	}
	if _, ok := d.States[d.Target]; !ok {
		return fmt.Errorf("target state %q not found", d.Target)
	}

	errList := make([]error, 0)
	for name := range maps.Keys(d.States) {
		state := d.States[name]
		if err := state.validate(name, d.States); err != nil {
			errList = append(errList, err)
		}
	}
	return errors.Join(errList...)
}

func (s State) validate(name string, stateMap map[string]State) error {
	errList := make([]error, 0)
	if err := s.Scene.validate(name); err != nil {
		errList = append(errList, err)
	}
	for idx, t := range s.Transitions {
		if t.To == "" {
			errList = append(errList, fmt.Errorf("state %q transition[%d] missing to", name, idx))
		}
		if _, ok := stateMap[t.To]; !ok {
			errList = append(errList, fmt.Errorf("state %q transition[%d] target %q not found", name, idx, t.To))
		}
		if err := t.Action.validate(name, idx); err != nil {
			errList = append(errList, err)
		}
	}
	return errors.Join(errList...)
}

func (s Scene) validate(stateName string) error {
	errList := make([]error, 0)
	for idx, selector := range s.Required {
		if err := selector.validate(); err != nil {
			errList = append(errList, fmt.Errorf("state %q required[%d]: %w", stateName, idx, err))
		}
	}
	for idx, selector := range s.Forbidden {
		if err := selector.validate(); err != nil {
			errList = append(errList, fmt.Errorf("state %q forbidden[%d]: %w", stateName, idx, err))
		}
	}
	for idx, optional := range s.Optional {
		if err := optional.Selector.validate(); err != nil {
			errList = append(errList, fmt.Errorf("state %q optional[%d]: %w", stateName, idx, err))
		}
	}
	return errors.Join(errList...)
}

func (s Selector) validate() error {
	if s.Ref == "" && s.Name == "" && s.AutomationID == "" && s.ClassName == "" && s.ControlType == "" {
		return errors.New("selector must include at least one field")
	}
	return nil
}

func (a Action) validate(stateName string, transitionIndex int) error {
	if a.Type == "" {
		return fmt.Errorf("state %q transition[%d] action type is required", stateName, transitionIndex)
	}
	if a.Type == ActionNone {
		return nil
	}
	if err := a.Selector.validate(); err != nil {
		return fmt.Errorf("state %q transition[%d] action selector: %w", stateName, transitionIndex, err)
	}
	if a.Type == ActionFill && a.Value == "" {
		return fmt.Errorf("state %q transition[%d] fill action requires value", stateName, transitionIndex)
	}
	return nil
}
