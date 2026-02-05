//go:build windows
// +build windows

package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"os"
	"regexp"
	"strings"

	directr "github.com/Tryanks/directr"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(2)
	}

	switch os.Args[1] {
	case "open":
		if err := runOpen(os.Args[2:]); err != nil {
			fmt.Fprintln(os.Stderr, "error:", err)
			os.Exit(1)
		}
	case "close":
		if err := runClose(os.Args[2:]); err != nil {
			fmt.Fprintln(os.Stderr, "error:", err)
			os.Exit(1)
		}
	case "list":
		if err := runList(os.Args[2:]); err != nil {
			fmt.Fprintln(os.Stderr, "error:", err)
			os.Exit(1)
		}
	case "snapshot":
		if err := runSnapshot(os.Args[2:]); err != nil {
			fmt.Fprintln(os.Stderr, "error:", err)
			os.Exit(1)
		}
	case "click":
		if err := runClick(os.Args[2:], false); err != nil {
			fmt.Fprintln(os.Stderr, "error:", err)
			os.Exit(1)
		}
	case "dblclick":
		if err := runClick(os.Args[2:], true); err != nil {
			fmt.Fprintln(os.Stderr, "error:", err)
			os.Exit(1)
		}
	case "hover":
		if err := runHover(os.Args[2:]); err != nil {
			fmt.Fprintln(os.Stderr, "error:", err)
			os.Exit(1)
		}
	case "fill":
		if err := runFill(os.Args[2:]); err != nil {
			fmt.Fprintln(os.Stderr, "error:", err)
			os.Exit(1)
		}
	case "type":
		if err := runType(os.Args[2:]); err != nil {
			fmt.Fprintln(os.Stderr, "error:", err)
			os.Exit(1)
		}
	case "press":
		if err := runPress(os.Args[2:]); err != nil {
			fmt.Fprintln(os.Stderr, "error:", err)
			os.Exit(1)
		}
	case "drag":
		if err := runDrag(os.Args[2:]); err != nil {
			fmt.Fprintln(os.Stderr, "error:", err)
			os.Exit(1)
		}
	case "select":
		if err := runSelect(os.Args[2:]); err != nil {
			fmt.Fprintln(os.Stderr, "error:", err)
			os.Exit(1)
		}
	case "check":
		if err := runCheck(os.Args[2:], true); err != nil {
			fmt.Fprintln(os.Stderr, "error:", err)
			os.Exit(1)
		}
	case "uncheck":
		if err := runCheck(os.Args[2:], false); err != nil {
			fmt.Fprintln(os.Stderr, "error:", err)
			os.Exit(1)
		}
	case "upload":
		if err := runUpload(os.Args[2:]); err != nil {
			fmt.Fprintln(os.Stderr, "error:", err)
			os.Exit(1)
		}
	case "session-delete":
		if err := runSessionDelete(os.Args[2:]); err != nil {
			fmt.Fprintln(os.Stderr, "error:", err)
			os.Exit(1)
		}
	case "focus":
		if err := runFocus(os.Args[2:]); err != nil {
			fmt.Fprintln(os.Stderr, "error:", err)
			os.Exit(1)
		}
	case "property":
		if err := runProperty(os.Args[2:]); err != nil {
			fmt.Fprintln(os.Stderr, "error:", err)
			os.Exit(1)
		}
	case "value":
		if err := runValue(os.Args[2:]); err != nil {
			fmt.Fprintln(os.Stderr, "error:", err)
			os.Exit(1)
		}
	case "tree":
		if err := runTree(os.Args[2:]); err != nil {
			fmt.Fprintln(os.Stderr, "error:", err)
			os.Exit(1)
		}
	case "invoke":
		if err := runInvoke(os.Args[2:]); err != nil {
			fmt.Fprintln(os.Stderr, "error:", err)
			os.Exit(1)
		}
	case "toggle":
		if err := runToggle(os.Args[2:]); err != nil {
			fmt.Fprintln(os.Stderr, "error:", err)
			os.Exit(1)
		}
	case "help", "-h", "--help":
		printUsage()
	default:
		printUsage()
		os.Exit(2)
	}
}

var refPattern = regexp.MustCompile(`^e\d+$`)

func printUsage() {
	fmt.Println("directr-cli (windows)")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  directr-cli <command> [options]")
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println("  open       Select a window and save it to the session")
	fmt.Println("  close      Close the selected window (WM_CLOSE)")
	fmt.Println("  list       List top-level visible windows")
	fmt.Println("  snapshot   Capture UI Automation tree and emit Playwright-like YAML")
	fmt.Println("  click      Invoke or click a target element")
	fmt.Println("  dblclick   Double-click a target element")
	fmt.Println("  hover      Move mouse to a target element")
	fmt.Println("  fill       Set value on a target element")
	fmt.Println("  type       Type text into the focused element")
	fmt.Println("  press      Press a key (e.g., Enter, Ctrl+A)")
	fmt.Println("  drag       Drag from one element to another")
	fmt.Println("  select     Select an option in a list/combo")
	fmt.Println("  check      Ensure a toggleable element is On")
	fmt.Println("  uncheck    Ensure a toggleable element is Off")
	fmt.Println("  upload     Best-effort file upload (ValuePattern if available)")
	fmt.Println("  focus      Focus a target element")
	fmt.Println("  property   Print properties for a target element")
	fmt.Println("  value      Read ValuePattern from a target element")
	fmt.Println("  tree       Dump a verbose UI Automation tree")
	fmt.Println("  invoke     Force InvokePattern on a target element")
	fmt.Println("  toggle     Toggle a target element")
	fmt.Println("  session-delete   Delete the session file")
	fmt.Println()
	fmt.Println("Common window flags:")
	fmt.Println("  --class       Window class name")
	fmt.Println("  --title       Window title")
	fmt.Println("  --hwnd        Window handle (hex like 0x1234 or decimal)")
	fmt.Println("  --session     Session file (default: .directr/session.json)")
	fmt.Println()
	fmt.Println("Common selector flags:")
	fmt.Println("  --ref         Snapshot ref (e1, e2, ...) or provide as positional argument")
	fmt.Println("  --name        Element name")
	fmt.Println("  --automation-id  UIA AutomationId")
	fmt.Println("  --class-name  UIA ClassName")
	fmt.Println("  --control-type  Control type (button, textbox, checkbox, ...)")
}

type windowFlagRefs struct {
	class *string
	title *string
	hwnd  *string
}

type selectorFlagRefs struct {
	ref          *string
	name         *string
	automationId *string
	className    *string
	controlType  *string
}

func bindWindowFlags(fs *flag.FlagSet) *windowFlagRefs {
	className := fs.String("class", "", "window class name")
	title := fs.String("title", "", "window title")
	hwndStr := fs.String("hwnd", "", "window handle in hex or decimal")
	return &windowFlagRefs{class: className, title: title, hwnd: hwndStr}
}

func bindSelectorFlags(fs *flag.FlagSet) *selectorFlagRefs {
	ref := fs.String("ref", "", "snapshot ref")
	name := fs.String("name", "", "element name")
	automationId := fs.String("automation-id", "", "automation id")
	className := fs.String("class-name", "", "class name")
	controlType := fs.String("control-type", "", "control type")
	return &selectorFlagRefs{ref: ref, name: name, automationId: automationId, className: className, controlType: controlType}
}

func selectorFromFlags(flags *selectorFlagRefs, args []string) (*directr.ElementSelector, error) {
	selector := &directr.ElementSelector{}
	selector.Ref = strings.TrimSpace(*flags.ref)
	selector.Name = strings.TrimSpace(*flags.name)
	selector.AutomationId = strings.TrimSpace(*flags.automationId)
	selector.ClassName = strings.TrimSpace(*flags.className)
	selector.ControlType = strings.TrimSpace(*flags.controlType)

	if selector.Ref == "" && len(args) > 0 && refPattern.MatchString(args[0]) {
		selector.Ref = args[0]
	}

	if selector.Ref == "" && selector.Name == "" && selector.AutomationId == "" && selector.ClassName == "" && selector.ControlType == "" {
		return nil, errors.New("selector not provided")
	}
	return selector, nil
}

func optionalSelectorFromFlags(flags *selectorFlagRefs, args []string) *directr.ElementSelector {
	selector, err := selectorFromFlags(flags, args)
	if err != nil {
		return nil
	}
	return selector
}

func parseSelectorAndValue(args []string, flags *selectorFlagRefs, value *string) (*directr.ElementSelector, error) {
	if len(args) >= 2 && refPattern.MatchString(args[0]) {
		if *value == "" {
			*value = args[1]
		}
		return &directr.ElementSelector{Ref: args[0]}, nil
	}

	selector, err := selectorFromFlags(flags, args)
	if err != nil {
		return nil, err
	}
	if *value == "" && len(args) > 0 {
		*value = args[0]
	}
	return selector, nil
}

func newFlagSet(name string) *flag.FlagSet {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.SetOutput(new(bytes.Buffer))
	return fs
}
