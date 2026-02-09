package witness

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/steveyegge/gastown/internal/tmux"
)

func TestZombieResult_Types(t *testing.T) {
	// Verify the ZombieResult type has all expected fields
	z := ZombieResult{
		PolecatName: "nux",
		AgentState:  "working",
		HookBead:    "gt-abc123",
		Action:      "auto-nuked",
		Error:       nil,
	}

	if z.PolecatName != "nux" {
		t.Errorf("PolecatName = %q, want %q", z.PolecatName, "nux")
	}
	if z.AgentState != "working" {
		t.Errorf("AgentState = %q, want %q", z.AgentState, "working")
	}
	if z.HookBead != "gt-abc123" {
		t.Errorf("HookBead = %q, want %q", z.HookBead, "gt-abc123")
	}
	if z.Action != "auto-nuked" {
		t.Errorf("Action = %q, want %q", z.Action, "auto-nuked")
	}
}

func TestDetectZombiePolecatsResult_EmptyResult(t *testing.T) {
	result := &DetectZombiePolecatsResult{}

	if result.Checked != 0 {
		t.Errorf("Checked = %d, want 0", result.Checked)
	}
	if len(result.Zombies) != 0 {
		t.Errorf("Zombies length = %d, want 0", len(result.Zombies))
	}
}

func TestDetectZombiePolecats_NonexistentDir(t *testing.T) {
	// Should handle missing polecats directory gracefully
	result := DetectZombiePolecats("/nonexistent/path", "testrig", nil)

	if result.Checked != 0 {
		t.Errorf("Checked = %d, want 0 for nonexistent dir", result.Checked)
	}
	if len(result.Zombies) != 0 {
		t.Errorf("Zombies = %d, want 0 for nonexistent dir", len(result.Zombies))
	}
}

func TestDetectZombiePolecats_DirectoryScanning(t *testing.T) {
	// Create a temp directory structure simulating polecats
	tmpDir := t.TempDir()
	rigName := "testrig"
	polecatsDir := filepath.Join(tmpDir, rigName, "polecats")
	if err := os.MkdirAll(polecatsDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Create polecat directories
	for _, name := range []string{"alpha", "bravo", "charlie"} {
		if err := os.Mkdir(filepath.Join(polecatsDir, name), 0o755); err != nil {
			t.Fatal(err)
		}
	}

	// Create hidden dir (should be skipped)
	if err := os.Mkdir(filepath.Join(polecatsDir, ".hidden"), 0o755); err != nil {
		t.Fatal(err)
	}

	// Create a regular file (should be skipped, not a dir)
	if err := os.WriteFile(filepath.Join(polecatsDir, "notadir.txt"), []byte("test"), 0o644); err != nil {
		t.Fatal(err)
	}

	result := DetectZombiePolecats(tmpDir, rigName, nil)

	// Should have checked 3 polecat dirs (not hidden, not file)
	if result.Checked != 3 {
		t.Errorf("Checked = %d, want 3 (should skip hidden dirs and files)", result.Checked)
	}

	// No zombies because agent bead state will be empty (bd not available),
	// so isZombie stays false for all polecats
	if len(result.Zombies) != 0 {
		t.Errorf("Zombies = %d, want 0 (no agent state = not zombie)", len(result.Zombies))
	}
}

func TestDetectZombiePolecats_EmptyPolecatsDir(t *testing.T) {
	// Empty polecats directory should return 0 checked
	tmpDir := t.TempDir()
	rigName := "testrig"
	polecatsDir := filepath.Join(tmpDir, rigName, "polecats")
	if err := os.MkdirAll(polecatsDir, 0o755); err != nil {
		t.Fatal(err)
	}

	result := DetectZombiePolecats(tmpDir, rigName, nil)

	if result.Checked != 0 {
		t.Errorf("Checked = %d, want 0 for empty polecats dir", result.Checked)
	}
}

func TestGetAgentBeadState_EmptyOutput(t *testing.T) {
	// getAgentBeadState with invalid bead ID should return empty strings
	// (it calls bd which won't exist in test, so it returns empty)
	state, hook := getAgentBeadState("/nonexistent", "nonexistent-bead")

	if state != "" {
		t.Errorf("state = %q, want empty for missing bead", state)
	}
	if hook != "" {
		t.Errorf("hook = %q, want empty for missing bead", hook)
	}
}

func TestSessionRecreated_NoSession(t *testing.T) {
	// When the session doesn't exist, sessionRecreated should return false
	// (the session wasn't recreated, it's still dead)
	tm := tmux.NewTmux()
	detectedAt := time.Now()

	recreated := sessionRecreated(tm, "gt-nonexistent-session-xyz", detectedAt)
	if recreated {
		t.Error("sessionRecreated returned true for nonexistent session, want false")
	}
}

func TestSessionRecreated_DetectedAtEdgeCases(t *testing.T) {
	// Verify that sessionRecreated returns false when session is dead
	// regardless of the detectedAt timestamp
	tm := tmux.NewTmux()

	// Try with a past timestamp
	recreated := sessionRecreated(tm, "gt-test-nosession-abc", time.Now().Add(-1*time.Hour))
	if recreated {
		t.Error("sessionRecreated returned true for nonexistent session with past time")
	}

	// Try with a future timestamp
	recreated = sessionRecreated(tm, "gt-test-nosession-def", time.Now().Add(1*time.Hour))
	if recreated {
		t.Error("sessionRecreated returned true for nonexistent session with future time")
	}
}

func TestZombieClassification_SpawningState(t *testing.T) {
	// Verify that "spawning" agent state is treated as a zombie indicator.
	// This tests the classification logic inline in DetectZombiePolecats.
	// We can't easily test this via the full function without mocking,
	// so we test the boolean logic directly.
	states := map[string]bool{
		"working":  true,
		"running":  true,
		"spawning": true,
		"idle":     false,
		"done":     false,
		"":         false,
	}

	for state, wantZombie := range states {
		hookBead := ""
		isZombie := false
		if hookBead != "" {
			isZombie = true
		}
		if state == "working" || state == "running" || state == "spawning" {
			isZombie = true
		}

		if isZombie != wantZombie {
			t.Errorf("agent_state=%q: isZombie=%v, want %v", state, isZombie, wantZombie)
		}
	}
}

func TestZombieClassification_HookBeadAlwaysZombie(t *testing.T) {
	// Any polecat with a hook_bead and dead session should be classified as zombie,
	// regardless of agent_state.
	for _, state := range []string{"", "idle", "done", "working"} {
		hookBead := "gt-some-issue"
		isZombie := false
		if hookBead != "" {
			isZombie = true
		}
		if state == "working" || state == "running" || state == "spawning" {
			isZombie = true
		}

		if !isZombie {
			t.Errorf("agent_state=%q with hook_bead=%q: isZombie=false, want true", state, hookBead)
		}
	}
}

func TestZombieClassification_NoHookNoActiveState(t *testing.T) {
	// Polecats with no hook_bead and non-active agent_state should NOT be zombies.
	for _, state := range []string{"", "idle", "done", "completed"} {
		hookBead := ""
		isZombie := false
		if hookBead != "" {
			isZombie = true
		}
		if state == "working" || state == "running" || state == "spawning" {
			isZombie = true
		}

		if isZombie {
			t.Errorf("agent_state=%q with no hook_bead: isZombie=true, want false", state)
		}
	}
}

func TestFindAnyCleanupWisp_NoBdAvailable(t *testing.T) {
	// When bd is not available (test environment), findAnyCleanupWisp
	// should return empty string without panicking
	result := findAnyCleanupWisp("/nonexistent", "testpolecat")
	if result != "" {
		t.Errorf("findAnyCleanupWisp = %q, want empty when bd unavailable", result)
	}
}
