//go:build windows
// +build windows

package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	directr "github.com/Tryanks/directr"
)

func runOpen(args []string) error {
	cf := bindCommonFlags("open", flagWindow|flagSession|flagSnapshot)
	if err := cf.parse(args); err != nil {
		return err
	}

	if *cf.window.class == "" && *cf.window.title == "" && *cf.window.hwnd == "" {
		return errors.New("open requires --class, --title, or --hwnd")
	}

	hwnd, err := directr.ResolveWindowHandle(*cf.window.hwnd, *cf.window.class, *cf.window.title)
	if err != nil {
		return err
	}

	data := directr.Data{
		Window:  directr.WindowRef{Hwnd: directr.FormatHwnd(hwnd), Class: *cf.window.class, Title: *cf.window.title},
		Updated: time.Now().Format(time.RFC3339),
	}
	if err := directr.Save(*cf.sessionPath, data); err != nil {
		return err
	}
	return autoSnapshot(cf, hwnd, data)
}

func runClose(args []string) error {
	cf := bindCommonFlags("close", flagWindow|flagSession)
	if err := cf.parse(args); err != nil {
		return err
	}

	hwnd, sessionData, err := directr.ResolveWindowFromFlagsOrSession(*cf.window.hwnd, *cf.window.class, *cf.window.title, *cf.sessionPath)
	if err != nil {
		return err
	}

	if err := directr.PostClose(hwnd); err != nil {
		return err
	}

	if sessionData.Window.Hwnd != "" {
		directr.Touch(&sessionData)
		return directr.Save(*cf.sessionPath, sessionData)
	}
	return nil
}

func runList(args []string) error {
	cf := bindCommonFlags("list", flagJSON|flagOut)
	if err := cf.parse(args); err != nil {
		return err
	}

	windows, err := directr.ListWindows()
	if err != nil {
		return err
	}

	var output string
	if *cf.json {
		data, err := json.MarshalIndent(windows, "", "  ")
		if err != nil {
			return err
		}
		output = string(data)
	} else {
		var buf strings.Builder
		for _, win := range windows {
			buf.WriteString(fmt.Sprintf("%s class=\"%s\" title=\"%s\"\n", win.Hwnd, win.Class, win.Title))
		}
		output = buf.String()
	}

	if *cf.out != "" {
		return writeOutput(*cf.out, output)
	}
	fmt.Print(output)
	return nil
}

func runSnapshot(args []string) error {
	cf := bindCommonFlags("snapshot", flagWindow|flagSession|flagOut|flagDepth|flagFormat)
	if err := cf.parse(args); err != nil {
		return err
	}

	hwnd, sessionData, err := directr.ResolveWindowFromFlagsOrSession(*cf.window.hwnd, *cf.window.class, *cf.window.title, *cf.sessionPath)
	if err != nil {
		return err
	}

	format := "full"
	if cf.format != nil {
		format = *cf.format
	}
	output, state, err := captureSnapshot(hwnd, *cf.maxDepth, *cf.maxNodes, format)
	if err != nil {
		return err
	}

	if err := writeOutput(*cf.out, output); err != nil {
		return err
	}

	return updateSessionWithSnapshot(cf, hwnd, state, sessionData)
}

func runClick(args []string, double bool) error {
	cf := bindCommonFlags("click", flagWindow|flagSession|flagSelector|flagSnapshot)
	if err := cf.parse(args); err != nil {
		return err
	}

	selector, err := selectorFromFlags(cf.selector, cf.fs.Args())
	if err != nil {
		return errors.New("click requires a selector")
	}

	hwnd, sessionData, err := directr.ResolveWindowFromFlagsOrSession(*cf.window.hwnd, *cf.window.class, *cf.window.title, *cf.sessionPath)
	if err != nil {
		return err
	}

	if err := directr.WithUIAutomation(hwnd, func(rootElement *directr.Element) error {
		element, err := directr.ResolveElement(rootElement, selector, sessionData)
		if err != nil {
			return err
		}
		if !double && directr.TryInvoke(element) {
			return nil
		}
		point, err := directr.ElementCenter(element)
		if err != nil {
			return err
		}
		if double {
			return directr.MouseDoubleClick(point.X, point.Y)
		}
		return directr.MouseClick(point.X, point.Y)
	}); err != nil {
		return err
	}
	return autoSnapshot(cf, hwnd, sessionData)
}

func runHover(args []string) error {
	cf := bindCommonFlags("hover", flagWindow|flagSession|flagSelector|flagSnapshot)
	if err := cf.parse(args); err != nil {
		return err
	}

	selector, err := selectorFromFlags(cf.selector, cf.fs.Args())
	if err != nil {
		return errors.New("hover requires a selector")
	}

	hwnd, sessionData, err := directr.ResolveWindowFromFlagsOrSession(*cf.window.hwnd, *cf.window.class, *cf.window.title, *cf.sessionPath)
	if err != nil {
		return err
	}

	if err := directr.WithUIAutomation(hwnd, func(rootElement *directr.Element) error {
		element, err := directr.ResolveElement(rootElement, selector, sessionData)
		if err != nil {
			return err
		}
		point, err := directr.ElementCenter(element)
		if err != nil {
			return err
		}
		return directr.SetCursor(point.X, point.Y)
	}); err != nil {
		return err
	}
	return autoSnapshot(cf, hwnd, sessionData)
}

func runFill(args []string) error {
	cf := bindCommonFlags("fill", flagWindow|flagSession|flagSelector|flagSnapshot)
	value := cf.fs.String("value", "", "value to set")
	if err := cf.parse(args); err != nil {
		return err
	}

	selector, err := parseSelectorAndValue(cf.fs.Args(), cf.selector, value)
	if err != nil {
		return err
	}
	if *value == "" {
		return errors.New("fill requires a value")
	}

	hwnd, sessionData, err := directr.ResolveWindowFromFlagsOrSession(*cf.window.hwnd, *cf.window.class, *cf.window.title, *cf.sessionPath)
	if err != nil {
		return err
	}

	if err := directr.WithUIAutomation(hwnd, func(rootElement *directr.Element) error {
		element, err := directr.ResolveElement(rootElement, selector, sessionData)
		if err != nil {
			return err
		}

		if err := directr.SetValue(element, *value); err == nil {
			return nil
		}

		if err := directr.Focus(element); err != nil {
			return err
		}
		if err := directr.SendKeyChord("Ctrl+A"); err != nil {
			return err
		}
		return directr.TypeText(*value)
	}); err != nil {
		return err
	}
	return autoSnapshot(cf, hwnd, sessionData)
}

func runType(args []string) error {
	cf := bindCommonFlags("type", flagWindow|flagSession|flagSelector|flagSnapshot)
	value := cf.fs.String("text", "", "text to type")
	if err := cf.parse(args); err != nil {
		return err
	}

	remaining := cf.fs.Args()
	selector := optionalSelectorFromFlags(cf.selector, remaining)
	if *value == "" && len(remaining) > 0 {
		if selector != nil && refPattern.MatchString(remaining[0]) && len(remaining) > 1 {
			*value = remaining[1]
		} else if !refPattern.MatchString(remaining[0]) {
			*value = remaining[0]
		}
	}
	if *value == "" {
		return errors.New("type requires text")
	}

	hwnd, sessionData, err := directr.ResolveWindowFromFlagsOrSession(*cf.window.hwnd, *cf.window.class, *cf.window.title, *cf.sessionPath)
	if err != nil {
		return err
	}

	if err := directr.WithUIAutomation(hwnd, func(rootElement *directr.Element) error {
		if selector != nil {
			element, err := directr.ResolveElement(rootElement, selector, sessionData)
			if err != nil {
				return err
			}
			if err := directr.Focus(element); err == nil {
				return directr.TypeText(*value)
			}
			point, err := directr.ElementCenter(element)
			if err != nil {
				return err
			}
			if err := directr.MouseClick(point.X, point.Y); err != nil {
				return err
			}
		}
		return directr.TypeText(*value)
	}); err != nil {
		return err
	}
	return autoSnapshot(cf, hwnd, sessionData)
}

func runPress(args []string) error {
	cf := bindCommonFlags("press", flagWindow|flagSession|flagSelector|flagSnapshot)
	key := cf.fs.String("key", "", "key to press")
	if err := cf.parse(args); err != nil {
		return err
	}

	remaining := cf.fs.Args()
	selector := optionalSelectorFromFlags(cf.selector, remaining)
	if *key == "" && len(remaining) > 0 {
		if selector != nil && refPattern.MatchString(remaining[0]) && len(remaining) > 1 {
			*key = remaining[1]
		} else if !refPattern.MatchString(remaining[0]) {
			*key = remaining[0]
		}
	}
	if *key == "" {
		return errors.New("press requires a key")
	}

	hwnd, sessionData, err := directr.ResolveWindowFromFlagsOrSession(*cf.window.hwnd, *cf.window.class, *cf.window.title, *cf.sessionPath)
	if err != nil {
		return err
	}

	if err := directr.WithUIAutomation(hwnd, func(rootElement *directr.Element) error {
		if selector != nil {
			element, err := directr.ResolveElement(rootElement, selector, sessionData)
			if err != nil {
				return err
			}
			if err := directr.Focus(element); err != nil {
				return err
			}
		}
		return directr.SendKeyChord(*key)
	}); err != nil {
		return err
	}
	return autoSnapshot(cf, hwnd, sessionData)
}

func runDrag(args []string) error {
	cf := bindCommonFlags("drag", flagWindow|flagSession|flagSnapshot)
	fromRef := cf.fs.String("from-ref", "", "source ref")
	toRef := cf.fs.String("to-ref", "", "target ref")
	if err := cf.parse(args); err != nil {
		return err
	}

	remaining := cf.fs.Args()
	if *fromRef == "" && len(remaining) > 0 && refPattern.MatchString(remaining[0]) {
		*fromRef = remaining[0]
	}
	if *toRef == "" && len(remaining) > 1 && refPattern.MatchString(remaining[1]) {
		*toRef = remaining[1]
	}
	if *fromRef == "" || *toRef == "" {
		return errors.New("drag requires --from-ref and --to-ref (or positional refs)")
	}

	hwnd, sessionData, err := directr.ResolveWindowFromFlagsOrSession(*cf.window.hwnd, *cf.window.class, *cf.window.title, *cf.sessionPath)
	if err != nil {
		return err
	}

	if err := directr.WithUIAutomation(hwnd, func(rootElement *directr.Element) error {
		fromSel := &directr.ElementSelector{Ref: *fromRef}
		toSel := &directr.ElementSelector{Ref: *toRef}
		fromEl, err := directr.ResolveElement(rootElement, fromSel, sessionData)
		if err != nil {
			return err
		}
		toEl, err := directr.ResolveElement(rootElement, toSel, sessionData)
		if err != nil {
			return err
		}
		fromPoint, err := directr.ElementCenter(fromEl)
		if err != nil {
			return err
		}
		toPoint, err := directr.ElementCenter(toEl)
		if err != nil {
			return err
		}
		return directr.MouseDrag(fromPoint.X, fromPoint.Y, toPoint.X, toPoint.Y)
	}); err != nil {
		return err
	}
	return autoSnapshot(cf, hwnd, sessionData)
}

func runSelect(args []string) error {
	cf := bindCommonFlags("select", flagWindow|flagSession|flagSelector|flagSnapshot)
	option := cf.fs.String("option", "", "option text to select")
	if err := cf.parse(args); err != nil {
		return err
	}

	selector, err := parseSelectorAndValue(cf.fs.Args(), cf.selector, option)
	if err != nil {
		return err
	}
	if *option == "" {
		return errors.New("select requires an option value")
	}

	hwnd, sessionData, err := directr.ResolveWindowFromFlagsOrSession(*cf.window.hwnd, *cf.window.class, *cf.window.title, *cf.sessionPath)
	if err != nil {
		return err
	}

	if err := directr.WithUIAutomation(hwnd, func(rootElement *directr.Element) error {
		target, err := directr.ResolveElement(rootElement, selector, sessionData)
		if err != nil {
			return err
		}

		directr.ExpandBestEffort(target)
		optionEl := directr.FindFirst(target, func(el *directr.Element) bool {
			return el.Name() == *option
		})
		if optionEl == nil {
			return fmt.Errorf("option %q not found", *option)
		}
		return directr.SelectItem(optionEl)
	}); err != nil {
		return err
	}
	return autoSnapshot(cf, hwnd, sessionData)
}

func runCheck(args []string, wantOn bool) error {
	name := "check"
	if !wantOn {
		name = "uncheck"
	}
	cf := bindCommonFlags(name, flagWindow|flagSession|flagSelector|flagSnapshot)
	if err := cf.parse(args); err != nil {
		return err
	}

	selector, err := selectorFromFlags(cf.selector, cf.fs.Args())
	if err != nil {
		return fmt.Errorf("%s requires a selector", name)
	}

	hwnd, sessionData, err := directr.ResolveWindowFromFlagsOrSession(*cf.window.hwnd, *cf.window.class, *cf.window.title, *cf.sessionPath)
	if err != nil {
		return err
	}

	if err := directr.WithUIAutomation(hwnd, func(rootElement *directr.Element) error {
		element, err := directr.ResolveElement(rootElement, selector, sessionData)
		if err != nil {
			return err
		}

		if err := directr.ToggleTo(element, wantOn); err == nil {
			return nil
		}
		return directr.Invoke(element)
	}); err != nil {
		return err
	}
	return autoSnapshot(cf, hwnd, sessionData)
}

func runUpload(args []string) error {
	cf := bindCommonFlags("upload", flagWindow|flagSession|flagSelector|flagSnapshot)
	path := cf.fs.String("file", "", "file path to upload")
	if err := cf.parse(args); err != nil {
		return err
	}

	selector, err := parseSelectorAndValue(cf.fs.Args(), cf.selector, path)
	if err != nil {
		return err
	}
	if *path == "" {
		return errors.New("upload requires a file path")
	}

	hwnd, sessionData, err := directr.ResolveWindowFromFlagsOrSession(*cf.window.hwnd, *cf.window.class, *cf.window.title, *cf.sessionPath)
	if err != nil {
		return err
	}

	if err := directr.WithUIAutomation(hwnd, func(rootElement *directr.Element) error {
		element, err := directr.ResolveElement(rootElement, selector, sessionData)
		if err != nil {
			return err
		}
		return directr.SetValue(element, *path)
	}); err != nil {
		return err
	}
	return autoSnapshot(cf, hwnd, sessionData)
}

func runFocus(args []string) error {
	cf := bindCommonFlags("focus", flagWindow|flagSession|flagSelector|flagSnapshot)
	if err := cf.parse(args); err != nil {
		return err
	}

	selector, err := selectorFromFlags(cf.selector, cf.fs.Args())
	if err != nil {
		return errors.New("focus requires a selector")
	}

	hwnd, sessionData, err := directr.ResolveWindowFromFlagsOrSession(*cf.window.hwnd, *cf.window.class, *cf.window.title, *cf.sessionPath)
	if err != nil {
		return err
	}

	if err := directr.WithUIAutomation(hwnd, func(rootElement *directr.Element) error {
		element, err := directr.ResolveElement(rootElement, selector, sessionData)
		if err != nil {
			return err
		}
		return directr.Focus(element)
	}); err != nil {
		return err
	}
	return autoSnapshot(cf, hwnd, sessionData)
}

func runProperty(args []string) error {
	cf := bindCommonFlags("property", flagWindow|flagSession|flagSelector|flagJSON|flagOut)
	if err := cf.parse(args); err != nil {
		return err
	}

	selector, err := selectorFromFlags(cf.selector, cf.fs.Args())
	if err != nil {
		return errors.New("property requires a selector")
	}

	hwnd, sessionData, err := directr.ResolveWindowFromFlagsOrSession(*cf.window.hwnd, *cf.window.class, *cf.window.title, *cf.sessionPath)
	if err != nil {
		return err
	}

	return directr.WithUIAutomation(hwnd, func(rootElement *directr.Element) error {
		element, err := directr.ResolveElement(rootElement, selector, sessionData)
		if err != nil {
			return err
		}
		props, err := directr.Properties(element)
		if err != nil {
			return err
		}

		var output string
		if *cf.json {
			encoded, err := json.MarshalIndent(props, "", "  ")
			if err != nil {
				return err
			}
			output = string(encoded)
		} else {
			var buf strings.Builder
			// Sort keys for deterministic output
			keys := make([]string, 0, len(props))
			for k := range props {
				keys = append(keys, k)
			}
			slices.Sort(keys)
			for _, k := range keys {
				buf.WriteString(fmt.Sprintf("%s: %v\n", k, props[k]))
			}
			output = buf.String()
		}

		if *cf.out != "" {
			return writeOutput(*cf.out, output)
		}
		fmt.Print(output)
		return nil
	})
}

func runValue(args []string) error {
	cf := bindCommonFlags("value", flagWindow|flagSession|flagSelector|flagJSON|flagOut)
	if err := cf.parse(args); err != nil {
		return err
	}

	selector, err := selectorFromFlags(cf.selector, cf.fs.Args())
	if err != nil {
		return errors.New("value requires a selector")
	}

	hwnd, sessionData, err := directr.ResolveWindowFromFlagsOrSession(*cf.window.hwnd, *cf.window.class, *cf.window.title, *cf.sessionPath)
	if err != nil {
		return err
	}

	return directr.WithUIAutomation(hwnd, func(rootElement *directr.Element) error {
		element, err := directr.ResolveElement(rootElement, selector, sessionData)
		if err != nil {
			return err
		}
		val, err := directr.GetValue(element)
		if err != nil {
			return err
		}

		var output string
		if *cf.json {
			encoded, err := json.MarshalIndent(map[string]string{"value": val}, "", "  ")
			if err != nil {
				return err
			}
			output = string(encoded)
		} else {
			output = val + "\n"
		}

		if *cf.out != "" {
			return writeOutput(*cf.out, output)
		}
		fmt.Print(output)
		return nil
	})
}

func runTree(args []string) error {
	cf := bindCommonFlags("tree", flagWindow|flagSession|flagOut|flagDepth)
	if err := cf.parse(args); err != nil {
		return err
	}

	hwnd, sessionData, err := directr.ResolveWindowFromFlagsOrSession(*cf.window.hwnd, *cf.window.class, *cf.window.title, *cf.sessionPath)
	if err != nil {
		return err
	}

	var output string
	if err := directr.WithUIAutomation(hwnd, func(rootElement *directr.Element) error {
		output = directr.VerboseTree(rootElement, *cf.maxDepth, *cf.maxNodes)
		return nil
	}); err != nil {
		return err
	}

	if err := writeOutput(*cf.out, output); err != nil {
		return err
	}

	sessionData.Window.Hwnd = directr.FormatHwnd(hwnd)
	sessionData.Window.Class = *cf.window.class
	sessionData.Window.Title = *cf.window.title
	directr.Touch(&sessionData)
	return directr.Save(*cf.sessionPath, sessionData)
}

func runInvoke(args []string) error {
	cf := bindCommonFlags("invoke", flagWindow|flagSession|flagSelector|flagSnapshot)
	if err := cf.parse(args); err != nil {
		return err
	}

	selector, err := selectorFromFlags(cf.selector, cf.fs.Args())
	if err != nil {
		return errors.New("invoke requires a selector")
	}

	hwnd, sessionData, err := directr.ResolveWindowFromFlagsOrSession(*cf.window.hwnd, *cf.window.class, *cf.window.title, *cf.sessionPath)
	if err != nil {
		return err
	}

	if err := directr.WithUIAutomation(hwnd, func(rootElement *directr.Element) error {
		element, err := directr.ResolveElement(rootElement, selector, sessionData)
		if err != nil {
			return err
		}
		return directr.Invoke(element)
	}); err != nil {
		return err
	}
	return autoSnapshot(cf, hwnd, sessionData)
}

func runToggle(args []string) error {
	cf := bindCommonFlags("toggle", flagWindow|flagSession|flagSelector|flagSnapshot)
	if err := cf.parse(args); err != nil {
		return err
	}

	selector, err := selectorFromFlags(cf.selector, cf.fs.Args())
	if err != nil {
		return errors.New("toggle requires a selector")
	}

	hwnd, sessionData, err := directr.ResolveWindowFromFlagsOrSession(*cf.window.hwnd, *cf.window.class, *cf.window.title, *cf.sessionPath)
	if err != nil {
		return err
	}

	if err := directr.WithUIAutomation(hwnd, func(rootElement *directr.Element) error {
		element, err := directr.ResolveElement(rootElement, selector, sessionData)
		if err != nil {
			return err
		}
		return directr.Toggle(element)
	}); err != nil {
		return err
	}
	return autoSnapshot(cf, hwnd, sessionData)
}

func runBatch(args []string) error {
	cf := bindCommonFlags("batch", flagWindow|flagSession|flagSnapshot|flagFormat)
	actionsJSON := cf.fs.String("actions", "", "JSON array of actions")
	stdin := cf.fs.Bool("stdin", false, "read actions JSON from stdin")
	if err := cf.parse(args); err != nil {
		return err
	}

	var rawJSON string
	if *stdin {
		data, err := os.ReadFile("/dev/stdin")
		if err != nil {
			// On Windows, read from os.Stdin directly
			buf := make([]byte, 0, 64*1024)
			tmp := make([]byte, 4096)
			for {
				n, readErr := os.Stdin.Read(tmp)
				if n > 0 {
					buf = append(buf, tmp[:n]...)
				}
				if readErr != nil {
					break
				}
			}
			rawJSON = string(buf)
		} else {
			rawJSON = string(data)
		}
	} else if *actionsJSON != "" {
		rawJSON = *actionsJSON
	} else if remaining := cf.fs.Args(); len(remaining) > 0 {
		rawJSON = remaining[0]
	} else {
		return errors.New("batch requires --actions JSON or --stdin")
	}

	var actions []directr.BatchAction
	if err := json.Unmarshal([]byte(rawJSON), &actions); err != nil {
		return fmt.Errorf("parse actions JSON: %w", err)
	}
	if len(actions) == 0 {
		return errors.New("batch requires at least one action")
	}

	hwnd, sessionData, err := directr.ResolveWindowFromFlagsOrSession(*cf.window.hwnd, *cf.window.class, *cf.window.title, *cf.sessionPath)
	if err != nil {
		return err
	}

	var results []directr.BatchResult
	if err := directr.WithUIAutomation(hwnd, func(rootElement *directr.Element) error {
		results = directr.ExecuteBatch(rootElement, actions, sessionData)
		return nil
	}); err != nil {
		return err
	}

	output, err := json.Marshal(map[string]any{"results": results})
	if err != nil {
		return err
	}
	fmt.Println(string(output))

	return autoSnapshot(cf, hwnd, sessionData)
}

func runSessionDelete(args []string) error {
	cf := bindCommonFlags("session-delete", flagSession)
	if err := cf.parse(args); err != nil {
		return err
	}
	if *cf.sessionPath == "" {
		return errors.New("session-delete requires --session path")
	}
	if err := os.Remove(*cf.sessionPath); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	fmt.Printf("Deleted session file: %s\n", *cf.sessionPath)
	return nil
}

func writeOutput(path string, data string) error {
	if path == "" {
		_, err := os.Stdout.Write([]byte(data))
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create output dir: %w", err)
	}
	return os.WriteFile(path, []byte(data), 0o644)
}

func captureSnapshot(hwnd uintptr, maxDepth, maxNodes int, format string) (string, directr.SnapshotState, error) {
	var output string
	var state directr.SnapshotState
	err := directr.WithUIAutomation(hwnd, func(rootElement *directr.Element) error {
		if format == "compact" {
			output, state = directr.CompactSnapshotTree(rootElement, maxDepth, maxNodes)
		} else {
			output, state = directr.SnapshotTree(rootElement, maxDepth, maxNodes)
		}
		return nil
	})
	return output, state, err
}

func updateSessionWithSnapshot(cf *commonFlags, hwnd uintptr, state directr.SnapshotState, sessionData directr.Data) error {
	sessionData.Window.Hwnd = directr.FormatHwnd(hwnd)
	if cf.window != nil {
		if *cf.window.class != "" {
			sessionData.Window.Class = *cf.window.class
		}
		if *cf.window.title != "" {
			sessionData.Window.Title = *cf.window.title
		}
	}
	sessionData.Snapshot = state
	sessionData.Snapshot.Captured = time.Now().Format(time.RFC3339)
	directr.Touch(&sessionData)
	return directr.Save(*cf.sessionPath, sessionData)
}

func autoSnapshot(cf *commonFlags, hwnd uintptr, sessionData directr.Data) error {
	if cf.snapshotMode == nil || *cf.snapshotMode == "off" {
		return nil
	}

	maxDepth := directr.DefaultMaxDepth
	if cf.maxDepth != nil && *cf.maxDepth > 0 {
		maxDepth = *cf.maxDepth
	}
	maxNodes := directr.DefaultMaxNodes
	if cf.maxNodes != nil && *cf.maxNodes > 0 {
		maxNodes = *cf.maxNodes
	}
	format := "full"
	if cf.format != nil {
		format = *cf.format
	}

	output, state, err := captureSnapshot(hwnd, maxDepth, maxNodes, format)
	if err != nil {
		return err
	}

	switch *cf.snapshotMode {
	case "stdout":
		fmt.Print(output)
	case "auto":
		dir := *cf.snapshotDir
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
		// Format: page-2006-01-02T15-04-05-000Z.yml
		// Use UTC and replace : with - for Windows compatibility
		now := time.Now().UTC()
		timestamp := now.Format("2006-01-02T15-04-05.000Z")
		timestamp = strings.ReplaceAll(timestamp, ":", "-")
		timestamp = strings.ReplaceAll(timestamp, ".", "-")
		filename := filepath.Join(dir, fmt.Sprintf("page-%s.yml", timestamp))
		if err := os.WriteFile(filename, []byte(output), 0o644); err != nil {
			return err
		}
		fmt.Fprintf(os.Stderr, "Snapshot saved to %s\n", filename)
	}

	return updateSessionWithSnapshot(cf, hwnd, state, sessionData)
}
