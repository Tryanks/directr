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
	"slices"
	"strings"

	directr "github.com/Tryanks/directr"
)

type command struct {
	run  func(args []string) error
	desc string
}

var commands = map[string]command{
	"open":           {run: runOpen, desc: "Select a window and save it to the session"},
	"close":          {run: runClose, desc: "Close the selected window (WM_CLOSE)"},
	"list":           {run: runList, desc: "List top-level visible windows"},
	"snapshot":       {run: runSnapshot, desc: "Capture UI Automation tree and emit Playwright-like YAML"},
	"click":          {run: func(args []string) error { return runClick(args, false) }, desc: "Invoke or click a target element"},
	"dblclick":       {run: func(args []string) error { return runClick(args, true) }, desc: "Double-click a target element"},
	"hover":          {run: runHover, desc: "Move mouse to a target element"},
	"fill":           {run: runFill, desc: "Set value on a target element"},
	"type":           {run: runType, desc: "Type text into the focused element"},
	"press":          {run: runPress, desc: "Press a key (e.g., Enter, Ctrl+A)"},
	"drag":           {run: runDrag, desc: "Drag from one element to another"},
	"select":         {run: runSelect, desc: "Select an option in a list/combo"},
	"check":          {run: func(args []string) error { return runCheck(args, true) }, desc: "Ensure a toggleable element is On"},
	"uncheck":        {run: func(args []string) error { return runCheck(args, false) }, desc: "Ensure a toggleable element is Off"},
	"upload":         {run: runUpload, desc: "Best-effort file upload (ValuePattern if available)"},
	"focus":          {run: runFocus, desc: "Focus a target element"},
	"property":       {run: runProperty, desc: "Print properties for a target element"},
	"value":          {run: runValue, desc: "Read ValuePattern from a target element"},
	"tree":           {run: runTree, desc: "Dump a verbose UI Automation tree"},
	"invoke":         {run: runInvoke, desc: "Force InvokePattern on a target element"},
	"toggle":         {run: runToggle, desc: "Toggle a target element"},
	"session-delete": {run: runSessionDelete, desc: "Delete the session file"},
	"version": {run: func(args []string) error {
		fmt.Printf("directr-cli version %s\n", Version)
		return nil
	}, desc: "Print version information"},
}

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(2)
	}

	cmdName := os.Args[1]
	if cmdName == "help" || cmdName == "-h" || cmdName == "--help" {
		printUsage()
		return
	}
	if cmdName == "version" || cmdName == "-v" || cmdName == "--version" {
		fmt.Printf("directr-cli version %s\n", Version)
		return
	}

	cmd, ok := commands[cmdName]
	if !ok {
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", cmdName)
		printUsage()
		os.Exit(2)
	}

	if err := cmd.run(os.Args[2:]); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

var (
	refPattern = regexp.MustCompile(`^e\d+$`)
	Version    = "0.1.0"
)

func printUsage() {
	fmt.Println("directr-cli (windows)")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  directr-cli <command> [options]")
	fmt.Println()
	fmt.Println("Commands:")

	// Sort commands for consistent output
	names := make([]string, 0, len(commands))
	for name := range commands {
		names = append(names, name)
	}
	slices.Sort(names)

	for _, name := range names {
		fmt.Printf("  %-12s %s\n", name, commands[name].desc)
	}
	fmt.Println()
	fmt.Println("Common flags (available for most commands):")
	fmt.Println("  --class           Window class name")
	fmt.Println("  --title           Window title")
	fmt.Println("  --hwnd            Window handle (hex or decimal)")
	fmt.Println("  --session         Session file path")
	fmt.Println("  --json            Output in JSON format")
	fmt.Println("  --snapshot        Snapshot mode: auto (default), off, stdout")
	fmt.Println("  --snapshot-dir    Directory to save snapshots (default: .directr-cli)")
	fmt.Println("  --out             Output to file instead of stdout")
	fmt.Println()
	fmt.Println("Common selector flags:")
	fmt.Println("  --ref             Snapshot ref (e1, e2, ...)")
	fmt.Println("  --name            Element name")
	fmt.Println("  --automation-id   UIA AutomationId")
	fmt.Println("  --class-name      UIA ClassName")
	fmt.Println("  --control-type    Control type")
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

type commonFlags struct {
	fs           *flag.FlagSet
	window       *windowFlagRefs
	selector     *selectorFlagRefs
	sessionPath  *string
	json         *bool
	out          *string
	maxDepth     *int
	maxNodes     *int
	snapshotMode *string
	snapshotDir  *string
}

func (f *commonFlags) parse(args []string) error {
	return f.fs.Parse(args)
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

func bindCommonFlags(name string, flags int) *commonFlags {
	fs := newFlagSet(name)
	cf := &commonFlags{fs: fs}

	if flags&flagWindow != 0 {
		cf.window = bindWindowFlags(fs)
	}
	if flags&flagSelector != 0 {
		cf.selector = bindSelectorFlags(fs)
	}
	if flags&flagSession != 0 {
		cf.sessionPath = fs.String("session", directr.DefaultPath(), "session file path")
	}
	if flags&flagJSON != 0 {
		cf.json = fs.Bool("json", false, "output in JSON format")
	}
	if flags&flagOut != 0 {
		cf.out = fs.String("out", "", "output file path")
	}
	if flags&flagDepth != 0 {
		cf.maxDepth = fs.Int("max-depth", directr.DefaultMaxDepth, "max traversal depth")
		cf.maxNodes = fs.Int("max-nodes", directr.DefaultMaxNodes, "max nodes")
	}
	if flags&flagSnapshot != 0 {
		cf.snapshotMode = fs.String("snapshot", "auto", "snapshot mode: auto, off, stdout")
		cf.snapshotDir = fs.String("snapshot-dir", ".directr-cli", "directory to save snapshots")
	}

	return cf
}

const (
	flagWindow = 1 << iota
	flagSelector
	flagSession
	flagJSON
	flagOut
	flagDepth
	flagSnapshot
)

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
