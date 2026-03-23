package creds

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const runtimeFile = "runtime.json"

// RuntimeState stores non-secret health metadata from the login loop.
type RuntimeState struct {
	LastCheckAt         string `json:"last_check_at"`
	LastSuccessAt       string `json:"last_success_at"`
	LastFailureAt       string `json:"last_failure_at"`
	LastError           string `json:"last_error"`
	LastStatus          string `json:"last_status"`
	LastMessage         string `json:"last_message"`
	ConsecutiveFailures int    `json:"consecutive_failures"`
}

func LoadRuntimeState() (RuntimeState, error) {
	dir, err := DataDir()
	if err != nil {
		return RuntimeState{}, fmt.Errorf("data dir: %w", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, runtimeFile))
	if err != nil {
		if os.IsNotExist(err) {
			return RuntimeState{}, nil
		}
		return RuntimeState{}, fmt.Errorf("read runtime state: %w", err)
	}

	var state RuntimeState
	if err := json.Unmarshal(data, &state); err != nil {
		return RuntimeState{}, fmt.Errorf("parse runtime state: %w", err)
	}
	return state, nil
}

func SaveRuntimeState(state RuntimeState) error {
	dir, err := DataDir()
	if err != nil {
		return fmt.Errorf("data dir: %w", err)
	}
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("create data dir: %w", err)
	}

	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal runtime state: %w", err)
	}

	return os.WriteFile(filepath.Join(dir, runtimeFile), data, 0600)
}

func UpdateRuntimeState(success bool, status, message string, when time.Time) error {
	state, err := LoadRuntimeState()
	if err != nil {
		return err
	}

	ts := when.Format(time.RFC3339)
	state.LastCheckAt = ts
	state.LastStatus = strings.TrimSpace(status)
	state.LastMessage = strings.TrimSpace(message)

	if success {
		state.LastSuccessAt = ts
		state.LastError = ""
		state.ConsecutiveFailures = 0
	} else {
		state.LastFailureAt = ts
		state.LastError = strings.TrimSpace(message)
		state.ConsecutiveFailures++
	}

	return SaveRuntimeState(state)
}
