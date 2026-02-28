//go:build windows
// +build windows

package flow

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	directr "github.com/Tryanks/directr"
)

type RunRequest struct {
	Definition     Definition `json:"-"`
	Script         string     `json:"script,omitzero"`
	ScriptPath     string     `json:"scriptPath,omitzero"`
	WindowHwnd     string     `json:"windowHwnd,omitzero"`
	WindowClass    string     `json:"windowClass,omitzero"`
	WindowTitle    string     `json:"windowTitle,omitzero"`
	TimeoutMS      int        `json:"timeoutMs,omitzero"`
	PollIntervalMS int        `json:"pollIntervalMs,omitzero"`
	SnapshotDepth  int        `json:"snapshotDepth,omitzero"`
	SnapshotNodes  int        `json:"snapshotNodes,omitzero"`
	MaxSteps       int        `json:"maxSteps,omitzero"`
}

type RunResponse struct {
	Result Result `json:"result,omitzero"`
	Error  string `json:"error,omitzero"`
}

func ParseRunRequestJSON(raw []byte) (RunRequest, error) {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	var request RunRequest
	if err := decoder.Decode(&request); err != nil {
		return RunRequest{}, err
	}
	if request.Script == "" && request.ScriptPath == "" {
		return RunRequest{}, fmt.Errorf("script or scriptPath is required")
	}
	if request.Script != "" && request.ScriptPath != "" {
		return RunRequest{}, fmt.Errorf("script and scriptPath are mutually exclusive")
	}
	if request.ScriptPath != "" {
		rawScript, err := os.ReadFile(request.ScriptPath)
		if err != nil {
			return RunRequest{}, fmt.Errorf("read script path: %w", err)
		}
		request.Definition, err = ParseDefinitionDSL(rawScript)
		if err != nil {
			return RunRequest{}, fmt.Errorf("parse script file: %w", err)
		}
	}
	if request.Script != "" {
		definition, err := ParseDefinitionDSL([]byte(request.Script))
		if err != nil {
			return RunRequest{}, fmt.Errorf("parse script: %w", err)
		}
		request.Definition = definition
	}
	request.Definition.Normalize()
	if err := request.Definition.Validate(); err != nil {
		return RunRequest{}, err
	}
	return request, nil
}

func RunFromRequest(ctx context.Context, request RunRequest) (Result, error) {
	hwnd, err := directr.ResolveWindowHandle(request.WindowHwnd, request.WindowClass, request.WindowTitle)
	if err != nil {
		return Result{}, err
	}

	if request.TimeoutMS > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, time.Duration(request.TimeoutMS)*time.Millisecond)
		defer cancel()
	}

	runner := Runner{Options: Options{
		MaxSteps:      request.MaxSteps,
		PollInterval:  time.Duration(request.PollIntervalMS) * time.Millisecond,
		SnapshotDepth: request.SnapshotDepth,
		SnapshotNodes: request.SnapshotNodes,
	}}
	return runner.Run(ctx, hwnd, request.Definition)
}

func RunFromRequestJSON(ctx context.Context, raw []byte) ([]byte, error) {
	request, err := ParseRunRequestJSON(raw)
	if err != nil {
		response := RunResponse{Error: err.Error()}
		return json.Marshal(response)
	}
	result, err := RunFromRequest(ctx, request)
	if err != nil {
		response := RunResponse{Result: result, Error: err.Error()}
		return json.Marshal(response)
	}
	response := RunResponse{Result: result}
	encoded, marshalErr := json.Marshal(response)
	if marshalErr != nil {
		return nil, fmt.Errorf("marshal run response: %w", marshalErr)
	}
	return encoded, nil
}
