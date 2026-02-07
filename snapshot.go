//go:build windows
// +build windows

package directr

import (
	"bytes"
	"fmt"
	"strings"
)

const (
	DefaultMaxDepth = 200
	DefaultMaxNodes = 50000
)

type refGen struct {
	n int
}

func (r *refGen) next() string {
	r.n++
	return fmt.Sprintf("e%d", r.n)
}

func SnapshotTree(root *Element, maxDepth, maxNodes int) (string, SnapshotState) {
	buf := &bytes.Buffer{}
	state := SnapshotState{Refs: map[string]SnapshotRef{}, MaxDepth: maxDepth, MaxNodes: maxNodes}
	gen := &refGen{}
	remaining := maxNodes
	state.RootTitle = root.Name()
	writeSnapshotNode(buf, root, 0, gen, maxDepth, &remaining, nil, &state)
	return buf.String(), state
}

func VerboseTree(root *Element, maxDepth, maxNodes int) string {
	buf := &bytes.Buffer{}
	remaining := maxNodes
	writeVerboseTree(buf, root, 0, maxDepth, &remaining)
	return buf.String()
}

func writeSnapshotNode(buf *bytes.Buffer, node *Element, depth int, gen *refGen, maxDepth int, remaining *int, path []int, state *SnapshotState) {
	if node == nil || *remaining <= 0 || depth > maxDepth {
		return
	}
	*remaining = *remaining - 1

	typeName := ControlTypeToName(node.ControlType())
	if typeName == "" {
		typeName = "generic"
	}

	name := strings.TrimSpace(node.Name())
	ref := gen.next()
	indent := strings.Repeat("  ", depth)

	line := indent + "- " + typeName
	if name != "" {
		line += " " + quoteIfNeeded(name)
	}
	line += " [ref=" + ref + "]"

	pathCopy := append([]int(nil), path...)
	state.Refs[ref] = SnapshotRef{
		Path:         pathCopy,
		Name:         node.Name(),
		AutomationId: node.AutomationId(),
		ClassName:    node.ClassName(),
		ControlType:  typeName,
	}

	children := node.Children()
	hasChildren := len(children) > 0 && depth < maxDepth
	if hasChildren {
		line += ":"
		buf.WriteString(line + "\n")
		for i, child := range children {
			childPath := append(pathCopy, i)
			writeSnapshotNode(buf, child, depth+1, gen, maxDepth, remaining, childPath, state)
			if *remaining <= 0 {
				return
			}
		}
		return
	}

	if name != "" && (typeName == "text" || typeName == "paragraph") {
		line += ": " + name
	}
	buf.WriteString(line + "\n")
}

func writeVerboseTree(buf *bytes.Buffer, node *Element, depth int, maxDepth int, remaining *int) {
	if node == nil || *remaining <= 0 || depth > maxDepth {
		return
	}
	*remaining = *remaining - 1

	typeName := ControlTypeToName(node.ControlType())
	if typeName == "" {
		typeName = "generic"
	}

	indent := strings.Repeat("  ", depth)
	line := fmt.Sprintf(
		"%s- %s name=%s class=%s automationId=%s enabled=%t",
		indent,
		typeName,
		quoteIfNeeded(strings.TrimSpace(node.Name())),
		quoteIfNeeded(strings.TrimSpace(node.ClassName())),
		quoteIfNeeded(strings.TrimSpace(node.AutomationId())),
		node.IsEnabled(),
	)
	buf.WriteString(line + "\n")

	if depth >= maxDepth {
		return
	}
	for _, child := range node.Children() {
		writeVerboseTree(buf, child, depth+1, maxDepth, remaining)
		if *remaining <= 0 {
			return
		}
	}
}

// CompactSnapshotTree generates a compact snapshot that only includes actionable
// elements in a flat format.  All elements still get a ref assigned (so the
// returned SnapshotState is identical to SnapshotTree), but only elements whose
// control type is in the actionable set are printed.
func CompactSnapshotTree(root *Element, maxDepth, maxNodes int) (string, SnapshotState) {
	state := SnapshotState{Refs: map[string]SnapshotRef{}, MaxDepth: maxDepth, MaxNodes: maxNodes}
	gen := &refGen{}
	remaining := maxNodes
	state.RootTitle = root.Name()

	// Collect actionable lines while walking the full tree.
	var lines []string
	collectCompactNodes(root, 0, gen, maxDepth, &remaining, nil, &state, &lines)

	buf := &bytes.Buffer{}
	buf.WriteString("window: " + quoteIfNeeded(strings.TrimSpace(root.Name())) + "\n---\n")
	for _, l := range lines {
		buf.WriteString(l + "\n")
	}
	return buf.String(), state
}

// collectCompactNodes walks the full tree, assigns refs to every node, stores
// them in state, and appends a line to *lines only for actionable elements.
func collectCompactNodes(node *Element, depth int, gen *refGen, maxDepth int, remaining *int, path []int, state *SnapshotState, lines *[]string) {
	if node == nil || *remaining <= 0 || depth > maxDepth {
		return
	}
	*remaining--

	typeName := ControlTypeToName(node.ControlType())
	if typeName == "" {
		typeName = "generic"
	}

	name := strings.TrimSpace(node.Name())
	ref := gen.next()

	pathCopy := append([]int(nil), path...)
	autoId := node.AutomationId()
	state.Refs[ref] = SnapshotRef{
		Path:         pathCopy,
		Name:         node.Name(),
		AutomationId: autoId,
		ClassName:    node.ClassName(),
		ControlType:  typeName,
	}

	if IsActionable(typeName) && name != "" {
		line := "[" + ref + "] " + typeName + " " + quoteIfNeeded(name)
		if autoId != "" {
			line += "  id=" + autoId
		}
		// For text/paragraph elements, append current value
		if typeName == "text" || typeName == "paragraph" {
			line += ": " + name
		}
		*lines = append(*lines, line)
	}

	if depth < maxDepth {
		for i, child := range node.Children() {
			childPath := append(pathCopy, i)
			collectCompactNodes(child, depth+1, gen, maxDepth, remaining, childPath, state, lines)
			if *remaining <= 0 {
				return
			}
		}
	}
}

// IsActionable returns true for control types that are typically interactive.
func IsActionable(typeName string) bool {
	switch typeName {
	case "button", "textbox", "text", "paragraph",
		"checkbox", "radio", "combobox", "slider",
		"listitem", "menuitem", "tab", "treeitem",
		"link", "spinbutton", "splitbutton":
		return true
	}
	return false
}

func quoteIfNeeded(value string) string {
	if value == "" {
		return "\"\""
	}
	replacer := strings.NewReplacer("\\", "\\\\", "\"", "\\\"")
	escaped := replacer.Replace(value)
	return "\"" + escaped + "\""
}
