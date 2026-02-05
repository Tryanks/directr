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

func quoteIfNeeded(value string) string {
	if value == "" {
		return "\"\""
	}
	replacer := strings.NewReplacer("\\", "\\\\", "\"", "\\\"")
	escaped := replacer.Replace(value)
	return "\"" + escaped + "\""
}
