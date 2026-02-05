//go:build windows
// +build windows

package directr

import (
	"testing"

	"github.com/Tryanks/directr/internal/uiautomation"
)

func TestSnapshotTree(t *testing.T) {
	root := &uiautomation.Element{
		CurrentControlType: uiautomation.UIA_PaneControlTypeId,
		CurrentName:        "Root",
		Child: []*uiautomation.Element{
			{
				CurrentControlType: uiautomation.UIA_TextControlTypeId,
				CurrentName:        "Hello",
			},
		},
	}

	out, state := SnapshotTree(wrapElement(root), 10, 10)

	want := "- generic \"Root\" [ref=e1]:\n  - text \"Hello\" [ref=e2]: Hello\n"
	if out != want {
		t.Fatalf("unexpected snapshot output:\nwant:\n%s\n\n got:\n%s", want, out)
	}
	if ref, ok := state.Refs["e1"]; !ok || len(ref.Path) != 0 {
		t.Fatalf("expected ref e1 at root path, got %+v", ref)
	}
	if ref, ok := state.Refs["e2"]; !ok || len(ref.Path) != 1 || ref.Path[0] != 0 {
		t.Fatalf("expected ref e2 at child path [0], got %+v", ref)
	}
}

func TestSnapshotTreeDepthAndNodesLimits(t *testing.T) {
	root := &uiautomation.Element{
		CurrentControlType: uiautomation.UIA_WindowControlTypeId,
		CurrentName:        "Root",
		Child: []*uiautomation.Element{
			{
				CurrentControlType: uiautomation.UIA_PaneControlTypeId,
				CurrentName:        "Child",
				Child: []*uiautomation.Element{
					{
						CurrentControlType: uiautomation.UIA_TextControlTypeId,
						CurrentName:        "Grandchild",
					},
				},
			},
		},
	}

	out, state := SnapshotTree(wrapElement(root), 1, 10)
	want := "- window \"Root\" [ref=e1]:\n  - generic \"Child\" [ref=e2]\n"
	if out != want {
		t.Fatalf("unexpected snapshot output with depth limit:\nwant:\n%s\n\n got:\n%s", want, out)
	}
	if ref, ok := state.Refs["e2"]; !ok || len(ref.Path) != 1 || ref.Path[0] != 0 {
		t.Fatalf("expected ref e2 at child path [0], got %+v", ref)
	}

	out, state = SnapshotTree(wrapElement(root), 10, 1)
	want = "- window \"Root\" [ref=e1]:\n"
	if out != want {
		t.Fatalf("unexpected snapshot output with node limit:\nwant:\n%s\n\n got:\n%s", want, out)
	}
	if len(state.Refs) != 1 {
		t.Fatalf("expected only 1 ref with node limit, got %d", len(state.Refs))
	}
}

func TestVerboseTreeOutput(t *testing.T) {
	root := &uiautomation.Element{
		CurrentControlType:  uiautomation.UIA_WindowControlTypeId,
		CurrentName:         "Root",
		CurrentClassName:    "MainWin",
		CurrentAutomationId: "root-1",
		CurrentIsEnabled:    1,
		Child: []*uiautomation.Element{
			{
				CurrentControlType:  uiautomation.UIA_TextControlTypeId,
				CurrentName:         "Hello",
				CurrentClassName:    "",
				CurrentAutomationId: "child-1",
				CurrentIsEnabled:    0,
			},
		},
	}

	out := VerboseTree(wrapElement(root), 10, 10)
	want := "- window name=\"Root\" class=\"MainWin\" automationId=\"root-1\" enabled=true\n" +
		"  - text name=\"Hello\" class=\"\" automationId=\"child-1\" enabled=false\n"
	if out != want {
		t.Fatalf("unexpected verbose output:\nwant:\n%s\n\n got:\n%s", want, out)
	}
}

func TestQuoteIfNeeded(t *testing.T) {
	if got := quoteIfNeeded(""); got != "\"\"" {
		t.Fatalf("expected empty string to be quoted, got %q", got)
	}
	if got := quoteIfNeeded("hello"); got != "\"hello\"" {
		t.Fatalf("expected quoted string, got %q", got)
	}
	if got := quoteIfNeeded("a\"b\\c"); got != "\"a\\\"b\\\\c\"" {
		t.Fatalf("expected escaped string, got %q", got)
	}
}
