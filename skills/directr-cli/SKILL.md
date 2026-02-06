---
name: directr-cli
description: Automates Windows UI Automation for desktop app testing, form filling, snapshot capture, and element interaction. Use when working on the directr-cli CLI: adding/changing commands, snapshot YAML output, selector/ref resolution, session handling, or UIA-backed behaviors in the Go codebase.
---

# Windows UI Automation with directr-cli

## Quick start

```bash
directr-cli list
directr-cli open --class "Chrome_WidgetWin_1" --title "My App"
# Each interaction (click, type, etc.) automatically saves a new snapshot to .directr-cli/ (current directory)
directr-cli click e15
directr-cli type "search query"
directr-cli press Enter
```

## Core workflow

1. Select a window: `directr-cli open --class ... --title ...`
2. Interact using refs from the snapshot.
3. **Automatic Snapshots**: Each successful action (click, fill, etc.) saves a new UI snapshot in the `.directr-cli/` directory (relative to current working directory).
4. **Session**: Window and snapshot mapping are stored in `~/.directr-cli/session.json` by default (global across different working directories).

## Global Flags

Available for most interaction commands:

- `--snapshot`: Snapshot mode: `auto` (default), `off` (disable auto-save), `stdout` (print to terminal).
- `--snapshot-dir`: Directory to save snapshots (default: `.directr-cli` in current directory).
- `--session`: Path to the session JSON (default: `~/.directr-cli/session.json`).

## Commands

### Core

```bash
directr-cli open --class "Chrome_WidgetWin_1" --title "My App"
directr-cli close
directr-cli list
directr-cli type "search query"
directr-cli click e3
directr-cli dblclick e7
directr-cli fill e5 "user@example.com"
directr-cli drag e2 e8
directr-cli hover e4
directr-cli select e9 "option-text"
directr-cli upload e10 "C:\\path\\file.pdf"
directr-cli check e12
directr-cli uncheck e12
directr-cli snapshot
```

### Keyboard

```bash
directr-cli press Enter
directr-cli press ArrowDown
```

### Mouse

```bash
directr-cli hover e4
directr-cli click e5
directr-cli dblclick e7
directr-cli drag e2 e8
```

### UIA Extras

```bash
directr-cli focus e5
directr-cli property e5
directr-cli value e5
directr-cli tree
directr-cli invoke e6
directr-cli toggle e12
```

### Sessions

```bash
directr-cli --session=mysession open --class "Chrome_WidgetWin_1" --title "My App"
directr-cli --session=mysession click e6
directr-cli session-delete
```

## Example: Form submission

```bash
directr-cli open --class "Chrome_WidgetWin_1" --title "Login"
# Initial snapshot is saved to .directr-cli/

directr-cli fill e1 "user@example.com"
# Automatic snapshot saved after fill
directr-cli fill e2 "password123"
# Automatic snapshot saved after fill
directr-cli click e3
# Automatic snapshot saved after click
```

## Example: Debugging with properties

```bash
directr-cli open --class "Chrome_WidgetWin_1" --title "Login"
# Use the auto-saved snapshot from .directr-cli/
directr-cli property e5
directr-cli value e5
```

## Snapshot/selector rules

- Snapshot output is Playwright-like YAML.
- Snapshot refs map to tree paths stored in session JSON.
- Selector resolution order:
  - `ref` from snapshot session
  - `automationId`/`name`/`class`/`controlType`
