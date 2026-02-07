//go:build windows
// +build windows

package main

import (
	"sync"

	directr "github.com/Tryanks/directr"
)

// serverState holds the in-memory state for the MCP server.
// All fields are protected by mu.
type serverState struct {
	mu      sync.Mutex
	hwnd    uintptr      // current window handle (0 = not connected)
	session directr.Data // session data including snapshot refs
}

// SetWindow stores the current window handle and resets the snapshot state.
func (s *serverState) SetWindow(hwnd uintptr, class, title string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.hwnd = hwnd
	s.session = directr.Data{
		Window: directr.WindowRef{
			Hwnd:  directr.FormatHwnd(hwnd),
			Class: class,
			Title: title,
		},
	}
}

// SetSnapshot updates the stored snapshot state.
func (s *serverState) SetSnapshot(snap directr.SnapshotState) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.session.Snapshot = snap
}

// Get returns a copy of the current hwnd and session data.
func (s *serverState) Get() (uintptr, directr.Data) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.hwnd, s.session
}

// Hwnd returns the current window handle.
func (s *serverState) Hwnd() uintptr {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.hwnd
}
