package creds

import "testing"

func TestLoadRuntimeStateMissingFile(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	state, err := LoadRuntimeState()
	if err != nil {
		t.Fatalf("LoadRuntimeState() returned error: %v", err)
	}
	if state.LastCheckAt != "" {
		t.Fatalf("LoadRuntimeState() returned unexpected data: %#v", state)
	}
}
