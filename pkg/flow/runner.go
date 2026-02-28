//go:build windows
// +build windows

package flow

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"time"

	directr "github.com/Tryanks/directr"
)

type Runner struct {
	Options Options
}

func (r Runner) Run(ctx context.Context, hwnd uintptr, definition Definition) (Result, error) {
	definition.Normalize()
	if err := definition.Validate(); err != nil {
		return Result{}, err
	}

	options := r.Options.withDefaults(definition)
	result := Result{
		Flow:      definition.Name,
		StartedAt: time.Now().UTC(),
	}

	session := directr.Data{Window: directr.WindowRef{Hwnd: directr.FormatHwnd(hwnd)}}
	for step := range options.MaxSteps {
		record := StepRecord{
			Index: step + 1,
			At:    time.Now().UTC(),
		}

		transition, detected, err := runStep(ctx, hwnd, definition, &session, options)
		record.Detected = detected.State
		record.Score = detected.Score
		if transition != nil {
			record.TransitionTo = transition.To
			record.Action = string(transition.Action.Type)
		}
		if err != nil {
			record.Error = err.Error()
			result.Steps = append(result.Steps, record)
			result.EndedAt = time.Now().UTC()
			result.LastState = record.Detected
			return result, err
		}

		result.Steps = append(result.Steps, record)
		result.LastState = record.Detected
		if detected.State == definition.Target {
			result.Success = true
			result.EndedAt = time.Now().UTC()
			return result, nil
		}

		if transition != nil && transition.WaitMS > 0 {
			if err := sleepContext(ctx, time.Duration(transition.WaitMS)*time.Millisecond); err != nil {
				result.EndedAt = time.Now().UTC()
				return result, err
			}
		}
		if options.PollInterval > 0 {
			if err := sleepContext(ctx, options.PollInterval); err != nil {
				result.EndedAt = time.Now().UTC()
				return result, err
			}
		}
	}

	result.EndedAt = time.Now().UTC()
	return result, fmt.Errorf("max steps exceeded: %d", options.MaxSteps)
}

type detection struct {
	State string
	Score float64
}

func runStep(ctx context.Context, hwnd uintptr, definition Definition, session *directr.Data, options Options) (*Transition, detection, error) {
	var stepTransition *Transition
	var detected detection

	err := directr.WithUIAutomation(hwnd, func(root *directr.Element) error {
		_, snapshot := directr.SnapshotTree(root, options.SnapshotDepth, options.SnapshotNodes)
		session.Snapshot = snapshot

		stateName, score, err := DetectState(root, *session, definition)
		if err != nil {
			return err
		}
		detected.State = stateName
		detected.Score = score

		if stateName == definition.Target {
			return nil
		}

		state := definition.States[stateName]
		if len(state.Transitions) == 0 {
			return fmt.Errorf("state %q has no transitions", stateName)
		}

		transition := state.Transitions[0]
		stepTransition = &transition
		if transition.To != "" && transition.To != stateName {
			if _, ok := definition.States[transition.To]; !ok {
				return fmt.Errorf("state %q transition target %q not found", stateName, transition.To)
			}
		}

		return runAction(ctx, root, *session, transition.Action)
	})
	return stepTransition, detected, err
}

func runAction(ctx context.Context, root *directr.Element, session directr.Data, action Action) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if action.Type == ActionNone {
		return nil
	}

	element, err := directr.ResolveElement(root, action.Selector.toDirectr(), session)
	if err != nil {
		return err
	}

	switch action.Type {
	case ActionClick:
		if directr.TryInvoke(element) {
			return nil
		}
		point, err := directr.ElementCenter(element)
		if err != nil {
			return err
		}
		return directr.MouseClick(point.X, point.Y)
	case ActionDoubleClick:
		point, err := directr.ElementCenter(element)
		if err != nil {
			return err
		}
		return directr.MouseDoubleClick(point.X, point.Y)
	case ActionInvoke:
		if directr.TryInvoke(element) {
			return nil
		}
		return errors.New("element does not support InvokePattern")
	case ActionFill:
		return directr.SetValue(element, action.Value)
	case ActionFocus:
		return directr.Focus(element)
	case ActionToggle:
		return directr.Toggle(element)
	case ActionToggleTo:
		return directr.ToggleTo(element, action.WantOn)
	case ActionSelect:
		return directr.SelectItem(element)
	default:
		return fmt.Errorf("unsupported action type %q", action.Type)
	}
}

func (o Options) withDefaults(definition Definition) Options {
	return Options{
		MaxSteps:      cmp.Or(o.MaxSteps, definition.MaxSteps, 60),
		PollInterval:  cmp.Or(o.PollInterval, 200*time.Millisecond),
		SnapshotDepth: cmp.Or(o.SnapshotDepth, directr.DefaultMaxDepth),
		SnapshotNodes: cmp.Or(o.SnapshotNodes, directr.DefaultMaxNodes),
	}
}

func sleepContext(ctx context.Context, d time.Duration) error {
	if d <= 0 {
		return nil
	}
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
