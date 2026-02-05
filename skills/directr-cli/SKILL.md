---
name: directr-cli
description: Automates Windows UI Automation for desktop app testing, form filling, snapshot capture, and element interaction. Use when working on the directr-cli CLI: adding/changing commands, snapshot YAML output, selector/ref resolution, session handling, or UIA-backed behaviors in the Go codebase.
---

# Windows UI Automation with directr-cli

## Quick start

```bash
directr-cli list
directr-cli open --class "Chrome_WidgetWin_1" --title "My App"
directr-cli snapshot
directr-cli click e15
directr-cli type "search query"
directr-cli press Enter
```

## Core workflow

1. Select a window: `directr-cli open --class ... --title ...`
2. Snapshot: `directr-cli snapshot`
3. Interact using refs from the snapshot
4. Re-snapshot after significant UI changes

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
directr-cli snapshot

directr-cli fill e1 "user@example.com"
directr-cli fill e2 "password123"
directr-cli click e3
directr-cli snapshot
```

## Example: Debugging with properties

```bash
directr-cli open --class "Chrome_WidgetWin_1" --title "Login"
directr-cli snapshot
directr-cli property e5
directr-cli value e5
```

## Snapshot/selector rules

- Snapshot output is Playwright-like YAML.
- Snapshot refs map to tree paths stored in session JSON.
- Selector resolution order:
  - `ref` from snapshot session
  - `automationId`/`name`/`class`/`controlType`
