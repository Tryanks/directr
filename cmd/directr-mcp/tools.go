//go:build windows

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	directr "github.com/Tryanks/directr"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// ── Tool input types ────────────────────────────────────────────────────────

type listWindowsArgs struct{}

type connectArgs struct {
	Title string `json:"title,omitempty" jsonschema:"Window title (substring match)"`
	Class string `json:"class,omitempty" jsonschema:"Window class name"`
	Hwnd  string `json:"hwnd,omitempty"  jsonschema:"Window handle (hex or decimal)"`
}

type snapshotArgs struct {
	Format string `json:"format,omitempty" jsonschema:"Snapshot format: compact (default) or full"`
}

type actionArgs struct {
	Actions  []directr.BatchAction `json:"actions"    jsonschema:"Array of actions to execute"`
	Snapshot bool                  `json:"snapshot,omitempty" jsonschema:"Return a compact snapshot after all actions"`
}

type readArgs struct {
	Ref          string `json:"ref,omitempty"          jsonschema:"Snapshot ref (e1 e2 ...)"`
	AutomationId string `json:"automationId,omitempty" jsonschema:"UIA AutomationId"`
	Name         string `json:"name,omitempty"         jsonschema:"Element name"`
	What         string `json:"what,omitempty"         jsonschema:"What to read: value (default), name, or properties"`
}

// ── Handlers ────────────────────────────────────────────────────────────────

func handleListWindows(_ context.Context, _ *mcp.CallToolRequest, _ listWindowsArgs) (*mcp.CallToolResult, any, error) {
	windows, err := directr.ListWindows()
	if err != nil {
		return errResult(err), nil, nil
	}

	var sb strings.Builder
	for _, w := range windows {
		sb.WriteString(fmt.Sprintf("%s class=%q title=%q\n", w.Hwnd, w.Class, w.Title))
	}
	return textResult(sb.String()), nil, nil
}

func handleConnect(_ context.Context, _ *mcp.CallToolRequest, args connectArgs) (*mcp.CallToolResult, any, error) {
	if args.Title == "" && args.Class == "" && args.Hwnd == "" {
		return errResult(fmt.Errorf("at least one of title, class, or hwnd is required")), nil, nil
	}

	hwnd, err := directr.ResolveWindowHandle(args.Hwnd, args.Class, args.Title)
	if err != nil {
		return errResult(err), nil, nil
	}

	state.SetWindow(hwnd, args.Class, args.Title)

	// Bring window to foreground
	_ = directr.FocusWindow(hwnd)

	// Capture compact snapshot
	var output string
	err = directr.WithUIAutomation(hwnd, func(root *directr.Element) error {
		var snap directr.SnapshotState
		output, snap = directr.CompactSnapshotTree(root, directr.DefaultMaxDepth, directr.DefaultMaxNodes)
		state.SetSnapshot(snap)
		return nil
	})
	if err != nil {
		return errResult(err), nil, nil
	}

	return textResult(output), nil, nil
}

func handleSnapshot(_ context.Context, _ *mcp.CallToolRequest, args snapshotArgs) (*mcp.CallToolResult, any, error) {
	hwnd := state.Hwnd()
	if hwnd == 0 {
		return errResult(fmt.Errorf("no window connected; call ui_connect first")), nil, nil
	}

	format := args.Format
	if format == "" {
		format = "compact"
	}

	var output string
	err := directr.WithUIAutomation(hwnd, func(root *directr.Element) error {
		var snap directr.SnapshotState
		if format == "compact" {
			output, snap = directr.CompactSnapshotTree(root, directr.DefaultMaxDepth, directr.DefaultMaxNodes)
		} else {
			output, snap = directr.SnapshotTree(root, directr.DefaultMaxDepth, directr.DefaultMaxNodes)
		}
		state.SetSnapshot(snap)
		return nil
	})
	if err != nil {
		return errResult(err), nil, nil
	}

	return textResult(output), nil, nil
}

func handleAction(_ context.Context, _ *mcp.CallToolRequest, args actionArgs) (*mcp.CallToolResult, any, error) {
	hwnd, sessionData := state.Get()
	if hwnd == 0 {
		return errResult(fmt.Errorf("no window connected; call ui_connect first")), nil, nil
	}

	if len(args.Actions) == 0 {
		return errResult(fmt.Errorf("actions array is empty")), nil, nil
	}

	// Bring window to foreground before executing actions
	_ = directr.FocusWindow(hwnd)

	var results []directr.BatchResult
	var snapOutput string

	err := directr.WithUIAutomation(hwnd, func(root *directr.Element) error {
		results = directr.ExecuteBatch(root, args.Actions, sessionData)
		return nil
	})
	if err != nil {
		return errResult(err), nil, nil
	}

	// Optionally capture a snapshot after actions
	if args.Snapshot {
		err = directr.WithUIAutomation(hwnd, func(root *directr.Element) error {
			var snap directr.SnapshotState
			snapOutput, snap = directr.CompactSnapshotTree(root, directr.DefaultMaxDepth, directr.DefaultMaxNodes)
			state.SetSnapshot(snap)
			return nil
		})
		if err != nil {
			return errResult(err), nil, nil
		}
	}

	// Build output
	resultJSON, _ := json.Marshal(results)
	output := string(resultJSON)
	if snapOutput != "" {
		output += "\n\n" + snapOutput
	}

	// Check if any action failed
	hasError := false
	for _, r := range results {
		if !r.OK {
			hasError = true
			break
		}
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: output}},
		IsError: hasError,
	}, nil, nil
}

func handleRead(_ context.Context, _ *mcp.CallToolRequest, args readArgs) (*mcp.CallToolResult, any, error) {
	hwnd, sessionData := state.Get()
	if hwnd == 0 {
		return errResult(fmt.Errorf("no window connected; call ui_connect first")), nil, nil
	}

	sel := &directr.ElementSelector{
		Ref:          args.Ref,
		AutomationId: args.AutomationId,
		Name:         args.Name,
	}

	what := args.What
	if what == "" {
		what = "value"
	}

	var output string
	err := directr.WithUIAutomation(hwnd, func(root *directr.Element) error {
		element, err := directr.ResolveElement(root, sel, sessionData)
		if err != nil {
			return err
		}

		switch what {
		case "value":
			val, err := directr.GetValue(element)
			if err != nil {
				// Fallback: return the element name (useful for text elements)
				output = element.Name()
				return nil
			}
			output = val
		case "name":
			output = element.Name()
		case "properties":
			props, err := directr.Properties(element)
			if err != nil {
				return err
			}
			encoded, err := json.MarshalIndent(props, "", "  ")
			if err != nil {
				return err
			}
			output = string(encoded)
		default:
			return fmt.Errorf("unknown what=%q; use value, name, or properties", what)
		}
		return nil
	})
	if err != nil {
		return errResult(err), nil, nil
	}

	return textResult(output), nil, nil
}

// ── Helpers ─────────────────────────────────────────────────────────────────

func textResult(text string) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: text}},
	}
}

func errResult(err error) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: "error: " + err.Error()}},
		IsError: true,
	}
}
