//go:build windows
// +build windows

package main

import (
	"context"
	"log"
	"os"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// state is the global server state shared across all tool handlers.
var state = &serverState{}

func main() {
	server := mcp.NewServer(
		&mcp.Implementation{
			Name:    "directr",
			Version: "0.1.0",
		},
		nil,
	)

	// ── Register tools ─────────────────────────────────────────────────

	mcp.AddTool(server, &mcp.Tool{
		Name:        "ui_list_windows",
		Description: "List all visible top-level windows with their handle, class, and title.",
	}, handleListWindows)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "ui_connect",
		Description: "Connect to a window by title, class, or handle. Returns a compact UI snapshot of all actionable elements (buttons, inputs, text, etc.) with refs and automationIds for subsequent actions.",
	}, handleConnect)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "ui_snapshot",
		Description: "Capture the current UI state. Default format is 'compact' (flat list of actionable elements only). Use format='full' for a complete YAML tree.",
	}, handleSnapshot)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "ui_action",
		Description: "Execute one or more UI actions in a single batch. Supported actions: click, dblclick, fill, type, press, hover, invoke, toggle, check, uncheck, select, drag. Each action needs a selector (ref, automationId, or name) and optionally a value. Set snapshot=true to get a compact UI snapshot after all actions.",
	}, handleAction)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "ui_read",
		Description: "Read the value, name, or properties of a UI element. Specify a selector (ref, automationId, or name) and what to read (value, name, or properties).",
	}, handleRead)

	// ── Run on stdio ───────────────────────────────────────────────────

	ctx := context.Background()
	if err := server.Run(ctx, &mcp.StdioTransport{}); err != nil {
		log.SetOutput(os.Stderr)
		log.Fatalf("MCP server error: %v", err)
	}
}
