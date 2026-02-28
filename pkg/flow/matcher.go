//go:build windows
// +build windows

package flow

import (
	"errors"
	"maps"
	"slices"

	directr "github.com/Tryanks/directr"
)

var ErrStateUnknown = errors.New("unable to identify current state")

func DetectState(root *directr.Element, session directr.Data, definition Definition) (string, float64, error) {
	stateNames := slices.Sorted(maps.Keys(definition.States))
	bestState := ""
	bestScore := -1.0
	for _, stateName := range stateNames {
		state := definition.States[stateName]
		score, ok := scoreScene(root, session, state.Scene)
		if !ok {
			continue
		}
		if score > bestScore {
			bestState = stateName
			bestScore = score
		}
	}
	if bestState == "" {
		return "", 0, ErrStateUnknown
	}
	return bestState, bestScore, nil
}

func scoreScene(root *directr.Element, session directr.Data, scene Scene) (float64, bool) {
	for _, selector := range scene.Forbidden {
		if matchSelector(root, session, selector) {
			return 0, false
		}
	}

	hitScore := 0.0
	totalScore := 0.0
	for _, selector := range scene.Required {
		totalScore++
		if !matchSelector(root, session, selector) {
			return 0, false
		}
		hitScore++
	}

	for _, optional := range scene.Optional {
		weight := optional.Weight
		if weight <= 0 {
			weight = 1
		}
		totalScore += weight
		if matchSelector(root, session, optional.Selector) {
			hitScore += weight
		}
	}

	if totalScore == 0 {
		return 0, false
	}

	score := hitScore / totalScore
	if scene.MinScore > 0 && score < scene.MinScore {
		return 0, false
	}
	return score, true
}

func matchSelector(root *directr.Element, session directr.Data, selector Selector) bool {
	_, err := directr.ResolveElement(root, selector.toDirectr(), session)
	return err == nil
}
