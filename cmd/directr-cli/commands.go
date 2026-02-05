//go:build windows
// +build windows

package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	directr "github.com/Tryanks/directr"
)

func runOpen(args []string) error {
	fs := newFlagSet("open")
	className := fs.String("class", "", "window class name")
	title := fs.String("title", "", "window title")
	hwndStr := fs.String("hwnd", "", "window handle in hex or decimal")
	sessionPath := fs.String("session", directr.DefaultPath(), "session file path")

	if err := fs.Parse(args); err != nil {
		return err
	}

	if *className == "" && *title == "" && *hwndStr == "" {
		return errors.New("open requires --class, --title, or --hwnd")
	}

	hwnd, err := directr.ResolveWindowHandle(*hwndStr, *className, *title)
	if err != nil {
		return err
	}

	data := directr.Data{
		Window:  directr.WindowRef{Hwnd: directr.FormatHwnd(hwnd), Class: *className, Title: *title},
		Updated: time.Now().Format(time.RFC3339),
	}
	return directr.Save(*sessionPath, data)
}

func runClose(args []string) error {
	fs := newFlagSet("close")
	className := fs.String("class", "", "window class name")
	title := fs.String("title", "", "window title")
	hwndStr := fs.String("hwnd", "", "window handle in hex or decimal")
	sessionPath := fs.String("session", directr.DefaultPath(), "session file path")

	if err := fs.Parse(args); err != nil {
		return err
	}

	hwnd, sessionData, err := directr.ResolveWindowFromFlagsOrSession(*hwndStr, *className, *title, *sessionPath)
	if err != nil {
		return err
	}

	if err := directr.PostClose(hwnd); err != nil {
		return err
	}

	if sessionData.Window.Hwnd != "" {
		directr.Touch(&sessionData)
		return directr.Save(*sessionPath, sessionData)
	}
	return nil
}

func runList(args []string) error {
	fs := newFlagSet("list")
	if err := fs.Parse(args); err != nil {
		return err
	}

	windows, err := directr.ListWindows()
	if err != nil {
		return err
	}
	for _, win := range windows {
		fmt.Printf("%s class=\"%s\" title=\"%s\"\n", win.Hwnd, win.Class, win.Title)
	}
	return nil
}

func runSnapshot(args []string) error {
	fs := newFlagSet("snapshot")
	className := fs.String("class", "", "window class name")
	title := fs.String("title", "", "window title")
	hwndStr := fs.String("hwnd", "", "window handle in hex or decimal")
	outPath := fs.String("out", "", "output file path")
	maxDepth := fs.Int("max-depth", directr.DefaultMaxDepth, "max traversal depth")
	maxNodes := fs.Int("max-nodes", directr.DefaultMaxNodes, "max nodes")
	sessionPath := fs.String("session", directr.DefaultPath(), "session file path")

	if err := fs.Parse(args); err != nil {
		return err
	}

	hwnd, sessionData, err := directr.ResolveWindowFromFlagsOrSession(*hwndStr, *className, *title, *sessionPath)
	if err != nil {
		return err
	}

	var output string
	var state directr.SnapshotState
	if err := directr.WithUIAutomation(hwnd, func(rootElement *directr.Element) error {
		output, state = directr.SnapshotTree(rootElement, *maxDepth, *maxNodes)
		return nil
	}); err != nil {
		return err
	}

	if err := writeOutput(*outPath, output); err != nil {
		return err
	}

	sessionData.Window.Hwnd = directr.FormatHwnd(hwnd)
	sessionData.Window.Class = *className
	sessionData.Window.Title = *title
	sessionData.Snapshot = state
	sessionData.Snapshot.Captured = time.Now().Format(time.RFC3339)
	directr.Touch(&sessionData)
	return directr.Save(*sessionPath, sessionData)
}

func runClick(args []string, double bool) error {
	fs := newFlagSet("click")
	windowFlags := bindWindowFlags(fs)
	sessionPath := fs.String("session", directr.DefaultPath(), "session file path")
	selectorFlags := bindSelectorFlags(fs)

	if err := fs.Parse(args); err != nil {
		return err
	}

	selector, err := selectorFromFlags(selectorFlags, fs.Args())
	if err != nil {
		return errors.New("click requires a selector")
	}

	hwnd, sessionData, err := directr.ResolveWindowFromFlagsOrSession(*windowFlags.hwnd, *windowFlags.class, *windowFlags.title, *sessionPath)
	if err != nil {
		return err
	}

	return directr.WithUIAutomation(hwnd, func(rootElement *directr.Element) error {
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
	})
}

func runHover(args []string) error {
	fs := newFlagSet("hover")
	windowFlags := bindWindowFlags(fs)
	sessionPath := fs.String("session", directr.DefaultPath(), "session file path")
	selectorFlags := bindSelectorFlags(fs)

	if err := fs.Parse(args); err != nil {
		return err
	}

	selector, err := selectorFromFlags(selectorFlags, fs.Args())
	if err != nil {
		return errors.New("hover requires a selector")
	}

	hwnd, sessionData, err := directr.ResolveWindowFromFlagsOrSession(*windowFlags.hwnd, *windowFlags.class, *windowFlags.title, *sessionPath)
	if err != nil {
		return err
	}

	return directr.WithUIAutomation(hwnd, func(rootElement *directr.Element) error {
		element, err := directr.ResolveElement(rootElement, selector, sessionData)
		if err != nil {
			return err
		}
		point, err := directr.ElementCenter(element)
		if err != nil {
			return err
		}
		return directr.SetCursor(point.X, point.Y)
	})
}

func runFill(args []string) error {
	fs := newFlagSet("fill")
	windowFlags := bindWindowFlags(fs)
	sessionPath := fs.String("session", directr.DefaultPath(), "session file path")
	selectorFlags := bindSelectorFlags(fs)
	value := fs.String("value", "", "value to set")

	if err := fs.Parse(args); err != nil {
		return err
	}

	selector, err := parseSelectorAndValue(fs.Args(), selectorFlags, value)
	if err != nil {
		return err
	}
	if *value == "" {
		return errors.New("fill requires a value")
	}

	hwnd, sessionData, err := directr.ResolveWindowFromFlagsOrSession(*windowFlags.hwnd, *windowFlags.class, *windowFlags.title, *sessionPath)
	if err != nil {
		return err
	}

	return directr.WithUIAutomation(hwnd, func(rootElement *directr.Element) error {
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
	})
}

func runType(args []string) error {
	fs := newFlagSet("type")
	windowFlags := bindWindowFlags(fs)
	sessionPath := fs.String("session", directr.DefaultPath(), "session file path")
	selectorFlags := bindSelectorFlags(fs)
	value := fs.String("text", "", "text to type")

	if err := fs.Parse(args); err != nil {
		return err
	}

	args = fs.Args()
	selector := optionalSelectorFromFlags(selectorFlags, args)
	if *value == "" && len(args) > 0 {
		if selector != nil && refPattern.MatchString(args[0]) && len(args) > 1 {
			*value = args[1]
		} else if !refPattern.MatchString(args[0]) {
			*value = args[0]
		}
	}
	if *value == "" {
		return errors.New("type requires text")
	}

	hwnd, sessionData, err := directr.ResolveWindowFromFlagsOrSession(*windowFlags.hwnd, *windowFlags.class, *windowFlags.title, *sessionPath)
	if err != nil {
		return err
	}

	return directr.WithUIAutomation(hwnd, func(rootElement *directr.Element) error {
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
	})
}

func runPress(args []string) error {
	fs := newFlagSet("press")
	windowFlags := bindWindowFlags(fs)
	sessionPath := fs.String("session", directr.DefaultPath(), "session file path")
	selectorFlags := bindSelectorFlags(fs)
	key := fs.String("key", "", "key to press")

	if err := fs.Parse(args); err != nil {
		return err
	}

	args = fs.Args()
	selector := optionalSelectorFromFlags(selectorFlags, args)
	if *key == "" && len(args) > 0 {
		if selector != nil && refPattern.MatchString(args[0]) && len(args) > 1 {
			*key = args[1]
		} else if !refPattern.MatchString(args[0]) {
			*key = args[0]
		}
	}
	if *key == "" {
		return errors.New("press requires a key")
	}

	hwnd, sessionData, err := directr.ResolveWindowFromFlagsOrSession(*windowFlags.hwnd, *windowFlags.class, *windowFlags.title, *sessionPath)
	if err != nil {
		return err
	}

	return directr.WithUIAutomation(hwnd, func(rootElement *directr.Element) error {
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
	})
}

func runDrag(args []string) error {
	fs := newFlagSet("drag")
	windowFlags := bindWindowFlags(fs)
	sessionPath := fs.String("session", directr.DefaultPath(), "session file path")
	fromRef := fs.String("from-ref", "", "source ref")
	toRef := fs.String("to-ref", "", "target ref")

	if err := fs.Parse(args); err != nil {
		return err
	}

	remaining := fs.Args()
	if *fromRef == "" && len(remaining) > 0 && refPattern.MatchString(remaining[0]) {
		*fromRef = remaining[0]
	}
	if *toRef == "" && len(remaining) > 1 && refPattern.MatchString(remaining[1]) {
		*toRef = remaining[1]
	}
	if *fromRef == "" || *toRef == "" {
		return errors.New("drag requires --from-ref and --to-ref (or positional refs)")
	}

	hwnd, sessionData, err := directr.ResolveWindowFromFlagsOrSession(*windowFlags.hwnd, *windowFlags.class, *windowFlags.title, *sessionPath)
	if err != nil {
		return err
	}

	return directr.WithUIAutomation(hwnd, func(rootElement *directr.Element) error {
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
	})
}

func runSelect(args []string) error {
	fs := newFlagSet("select")
	windowFlags := bindWindowFlags(fs)
	sessionPath := fs.String("session", directr.DefaultPath(), "session file path")
	selectorFlags := bindSelectorFlags(fs)
	option := fs.String("option", "", "option text to select")

	if err := fs.Parse(args); err != nil {
		return err
	}

	selector, err := parseSelectorAndValue(fs.Args(), selectorFlags, option)
	if err != nil {
		return err
	}
	if *option == "" {
		return errors.New("select requires an option value")
	}

	hwnd, sessionData, err := directr.ResolveWindowFromFlagsOrSession(*windowFlags.hwnd, *windowFlags.class, *windowFlags.title, *sessionPath)
	if err != nil {
		return err
	}

	return directr.WithUIAutomation(hwnd, func(rootElement *directr.Element) error {
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
	})
}

func runCheck(args []string, wantOn bool) error {
	fs := newFlagSet("check")
	windowFlags := bindWindowFlags(fs)
	sessionPath := fs.String("session", directr.DefaultPath(), "session file path")
	selectorFlags := bindSelectorFlags(fs)

	if err := fs.Parse(args); err != nil {
		return err
	}

	selector, err := selectorFromFlags(selectorFlags, fs.Args())
	if err != nil {
		return errors.New("check/uncheck requires a selector")
	}

	hwnd, sessionData, err := directr.ResolveWindowFromFlagsOrSession(*windowFlags.hwnd, *windowFlags.class, *windowFlags.title, *sessionPath)
	if err != nil {
		return err
	}

	return directr.WithUIAutomation(hwnd, func(rootElement *directr.Element) error {
		element, err := directr.ResolveElement(rootElement, selector, sessionData)
		if err != nil {
			return err
		}

		if err := directr.ToggleTo(element, wantOn); err == nil {
			return nil
		}
		return directr.Invoke(element)
	})
}

func runUpload(args []string) error {
	fs := newFlagSet("upload")
	windowFlags := bindWindowFlags(fs)
	sessionPath := fs.String("session", directr.DefaultPath(), "session file path")
	selectorFlags := bindSelectorFlags(fs)
	path := fs.String("file", "", "file path to upload")

	if err := fs.Parse(args); err != nil {
		return err
	}

	selector, err := parseSelectorAndValue(fs.Args(), selectorFlags, path)
	if err != nil {
		return err
	}
	if *path == "" {
		return errors.New("upload requires a file path")
	}

	hwnd, sessionData, err := directr.ResolveWindowFromFlagsOrSession(*windowFlags.hwnd, *windowFlags.class, *windowFlags.title, *sessionPath)
	if err != nil {
		return err
	}

	return directr.WithUIAutomation(hwnd, func(rootElement *directr.Element) error {
		element, err := directr.ResolveElement(rootElement, selector, sessionData)
		if err != nil {
			return err
		}
		return directr.SetValue(element, *path)
	})
}

func runFocus(args []string) error {
	fs := newFlagSet("focus")
	windowFlags := bindWindowFlags(fs)
	sessionPath := fs.String("session", directr.DefaultPath(), "session file path")
	selectorFlags := bindSelectorFlags(fs)

	if err := fs.Parse(args); err != nil {
		return err
	}

	selector, err := selectorFromFlags(selectorFlags, fs.Args())
	if err != nil {
		return errors.New("focus requires a selector")
	}

	hwnd, sessionData, err := directr.ResolveWindowFromFlagsOrSession(*windowFlags.hwnd, *windowFlags.class, *windowFlags.title, *sessionPath)
	if err != nil {
		return err
	}

	return directr.WithUIAutomation(hwnd, func(rootElement *directr.Element) error {
		element, err := directr.ResolveElement(rootElement, selector, sessionData)
		if err != nil {
			return err
		}
		return directr.Focus(element)
	})
}

func runProperty(args []string) error {
	fs := newFlagSet("property")
	windowFlags := bindWindowFlags(fs)
	sessionPath := fs.String("session", directr.DefaultPath(), "session file path")
	selectorFlags := bindSelectorFlags(fs)

	if err := fs.Parse(args); err != nil {
		return err
	}

	selector, err := selectorFromFlags(selectorFlags, fs.Args())
	if err != nil {
		return errors.New("property requires a selector")
	}

	hwnd, sessionData, err := directr.ResolveWindowFromFlagsOrSession(*windowFlags.hwnd, *windowFlags.class, *windowFlags.title, *sessionPath)
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
		encoded, err := json.MarshalIndent(props, "", "  ")
		if err != nil {
			return err
		}
		fmt.Println(string(encoded))
		return nil
	})
}

func runValue(args []string) error {
	fs := newFlagSet("value")
	windowFlags := bindWindowFlags(fs)
	sessionPath := fs.String("session", directr.DefaultPath(), "session file path")
	selectorFlags := bindSelectorFlags(fs)

	if err := fs.Parse(args); err != nil {
		return err
	}

	selector, err := selectorFromFlags(selectorFlags, fs.Args())
	if err != nil {
		return errors.New("value requires a selector")
	}

	hwnd, sessionData, err := directr.ResolveWindowFromFlagsOrSession(*windowFlags.hwnd, *windowFlags.class, *windowFlags.title, *sessionPath)
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
		fmt.Println(val)
		return nil
	})
}

func runTree(args []string) error {
	fs := newFlagSet("tree")
	className := fs.String("class", "", "window class name")
	title := fs.String("title", "", "window title")
	hwndStr := fs.String("hwnd", "", "window handle in hex or decimal")
	outPath := fs.String("out", "", "output file path")
	maxDepth := fs.Int("max-depth", directr.DefaultMaxDepth, "max traversal depth")
	maxNodes := fs.Int("max-nodes", directr.DefaultMaxNodes, "max nodes")
	sessionPath := fs.String("session", directr.DefaultPath(), "session file path")

	if err := fs.Parse(args); err != nil {
		return err
	}

	hwnd, sessionData, err := directr.ResolveWindowFromFlagsOrSession(*hwndStr, *className, *title, *sessionPath)
	if err != nil {
		return err
	}

	var output string
	if err := directr.WithUIAutomation(hwnd, func(rootElement *directr.Element) error {
		output = directr.VerboseTree(rootElement, *maxDepth, *maxNodes)
		return nil
	}); err != nil {
		return err
	}

	if err := writeOutput(*outPath, output); err != nil {
		return err
	}

	sessionData.Window.Hwnd = directr.FormatHwnd(hwnd)
	sessionData.Window.Class = *className
	sessionData.Window.Title = *title
	directr.Touch(&sessionData)
	return directr.Save(*sessionPath, sessionData)
}

func runInvoke(args []string) error {
	fs := newFlagSet("invoke")
	windowFlags := bindWindowFlags(fs)
	sessionPath := fs.String("session", directr.DefaultPath(), "session file path")
	selectorFlags := bindSelectorFlags(fs)

	if err := fs.Parse(args); err != nil {
		return err
	}

	selector, err := selectorFromFlags(selectorFlags, fs.Args())
	if err != nil {
		return errors.New("invoke requires a selector")
	}

	hwnd, sessionData, err := directr.ResolveWindowFromFlagsOrSession(*windowFlags.hwnd, *windowFlags.class, *windowFlags.title, *sessionPath)
	if err != nil {
		return err
	}

	return directr.WithUIAutomation(hwnd, func(rootElement *directr.Element) error {
		element, err := directr.ResolveElement(rootElement, selector, sessionData)
		if err != nil {
			return err
		}
		return directr.Invoke(element)
	})
}

func runToggle(args []string) error {
	fs := newFlagSet("toggle")
	windowFlags := bindWindowFlags(fs)
	sessionPath := fs.String("session", directr.DefaultPath(), "session file path")
	selectorFlags := bindSelectorFlags(fs)

	if err := fs.Parse(args); err != nil {
		return err
	}

	selector, err := selectorFromFlags(selectorFlags, fs.Args())
	if err != nil {
		return errors.New("toggle requires a selector")
	}

	hwnd, sessionData, err := directr.ResolveWindowFromFlagsOrSession(*windowFlags.hwnd, *windowFlags.class, *windowFlags.title, *sessionPath)
	if err != nil {
		return err
	}

	return directr.WithUIAutomation(hwnd, func(rootElement *directr.Element) error {
		element, err := directr.ResolveElement(rootElement, selector, sessionData)
		if err != nil {
			return err
		}
		return directr.Toggle(element)
	})
}

func runSessionDelete(args []string) error {
	fs := newFlagSet("session-delete")
	sessionPath := fs.String("session", directr.DefaultPath(), "session file path")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *sessionPath == "" {
		return errors.New("session-delete requires --session path")
	}
	if err := os.Remove(*sessionPath); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
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
