//go:build windows

package directr

import (
	"errors"
	"fmt"
)

type ElementSelector struct {
	Ref          string
	Name         string
	AutomationId string
	ClassName    string
	ControlType  string
}

func ResolveElement(root *Element, selector *ElementSelector, sessionData Data) (*Element, error) {
	if selector.Ref != "" {
		if sessionData.Snapshot.Refs == nil {
			return nil, errors.New("no snapshot refs in session; run snapshot first")
		}
		refInfo, ok := sessionData.Snapshot.Refs[selector.Ref]
		if !ok {
			return nil, fmt.Errorf("ref %s not found in last snapshot", selector.Ref)
		}
		element := FindByPath(root, refInfo.Path)
		if element == nil {
			return nil, fmt.Errorf("ref %s path no longer matches current UI tree", selector.Ref)
		}
		return element, nil
	}

	if selector.Name == "" && selector.AutomationId == "" && selector.ClassName == "" && selector.ControlType == "" {
		return nil, errors.New("selector did not include a matchable field")
	}

	var controlType ControlTypeId
	if selector.ControlType != "" {
		parsed, ok := ControlTypeFromName(selector.ControlType)
		if !ok {
			return nil, fmt.Errorf("unknown control type %q", selector.ControlType)
		}
		controlType = parsed
	}

	match := func(el *Element) bool {
		if selector.Name != "" && el.Name() != selector.Name {
			return false
		}
		if selector.AutomationId != "" && el.AutomationId() != selector.AutomationId {
			return false
		}
		if selector.ClassName != "" && el.ClassName() != selector.ClassName {
			return false
		}
		if selector.ControlType != "" && el.ControlType() != controlType {
			return false
		}
		return true
	}

	if element := FindFirst(root, match); element != nil {
		return element, nil
	}
	return nil, errors.New("selector did not match any element")
}

func FindByPath(root *Element, path []int) *Element {
	if root == nil {
		return nil
	}
	node := root
	for _, idx := range path {
		children := node.Children()
		if idx < 0 || idx >= len(children) {
			return nil
		}
		node = children[idx]
	}
	return node
}

func FindFirst(root *Element, predicate func(*Element) bool) *Element {
	if root == nil {
		return nil
	}
	if predicate(root) {
		return root
	}
	for _, child := range root.Children() {
		if found := FindFirst(child, predicate); found != nil {
			return found
		}
	}
	return nil
}
