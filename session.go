package directr

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

type Data struct {
	Window   WindowRef     `json:"window"`
	Snapshot SnapshotState `json:"snapshot,omitempty"`
	Updated  string        `json:"updated"`
}

type WindowRef struct {
	Hwnd  string `json:"hwnd"`
	Class string `json:"class,omitempty"`
	Title string `json:"title,omitempty"`
}

type SnapshotState struct {
	Refs      map[string]SnapshotRef `json:"refs"`
	Captured  string                 `json:"captured"`
	MaxDepth  int                    `json:"maxDepth"`
	MaxNodes  int                    `json:"maxNodes"`
	RootTitle string                 `json:"rootTitle,omitempty"`
}

type SnapshotRef struct {
	Path         []int  `json:"path"`
	Name         string `json:"name,omitempty"`
	AutomationId string `json:"automationId,omitempty"`
	ClassName    string `json:"className,omitempty"`
	ControlType  string `json:"controlType,omitempty"`
}

func DefaultPath() string {
	return filepath.Join(".directr", "session.json")
}

func Load(path string) (Data, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Data{}, err
	}

	var session Data
	if err := json.Unmarshal(data, &session); err != nil {
		return Data{}, err
	}
	return session, nil
}

func Save(path string, session Data) error {
	if path == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create session dir: %w", err)
	}
	encoded, err := json.MarshalIndent(session, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, encoded, 0o644)
}

func Touch(session *Data) {
	if session == nil {
		return
	}
	session.Updated = time.Now().Format(time.RFC3339)
}
