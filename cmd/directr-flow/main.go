//go:build windows
// +build windows

package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/Tryanks/directr/pkg/flow"
)

func main() {
	var requestPath string
	var pretty bool

	flag.StringVar(&requestPath, "request", "", "Path to flow run request JSON. If empty, read request from stdin.")
	flag.BoolVar(&pretty, "pretty", false, "Pretty-print JSON output")
	flag.Parse()

	requestRaw, err := readRequest(requestPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}

	responseRaw, err := flow.RunFromRequestJSON(context.Background(), requestRaw)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}

	if !pretty {
		_, _ = os.Stdout.Write(responseRaw)
		_, _ = os.Stdout.WriteString("\n")
		return
	}

	var payload any
	if err := json.Unmarshal(responseRaw, &payload); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
	formatted, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
	_, _ = os.Stdout.Write(formatted)
	_, _ = os.Stdout.WriteString("\n")
}

func readRequest(path string) ([]byte, error) {
	if path != "" {
		return os.ReadFile(path)
	}
	raw, err := io.ReadAll(os.Stdin)
	if err != nil {
		return nil, err
	}
	if len(raw) == 0 {
		return nil, fmt.Errorf("empty request JSON")
	}
	return raw, nil
}
