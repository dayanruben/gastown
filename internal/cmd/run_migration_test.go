package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestExtractCommands(t *testing.T) {
	tests := []struct {
		name        string
		description string
		townRoot    string
		wantCount   int
		wantFirst   string
	}{
		{
			name: "single bash block",
			description: `Do something.

` + "```bash" + `
gt version
` + "```" + `

Done.`,
			townRoot:  "/tmp/gt",
			wantCount: 1,
			wantFirst: "gt version",
		},
		{
			name: "multiple bash blocks",
			description: `Step one:

` + "```bash" + `
gt version
` + "```" + `

Step two:

` + "```bash" + `
gt doctor
` + "```",
			townRoot:  "/tmp/gt",
			wantCount: 2,
			wantFirst: "gt version",
		},
		{
			name: "template variable replacement",
			description: `Check rigs:

` + "```bash" + `
ls -d {{town_root}}/*/
` + "```",
			townRoot:  "/home/user/gt",
			wantCount: 1,
			wantFirst: "ls -d /home/user/gt/*/",
		},
		{
			name: "comment-only block excluded",
			description: `Explanation:

` + "```bash" + `
# This is just a comment
# Another comment
` + "```",
			townRoot:  "/tmp/gt",
			wantCount: 0,
		},
		{
			name: "multiline block preserved",
			description: `Loop:

` + "```bash" + `
for dir in {{town_root}}/*/; do
  echo "$dir"
done
` + "```",
			townRoot:  "/tmp/gt",
			wantCount: 1,
		},
		{
			name:        "no code blocks",
			description: "Just some prose instructions without any code.",
			townRoot:    "/tmp/gt",
			wantCount:   0,
		},
		{
			name: "sh block also works",
			description: `Run:

` + "```sh" + `
echo hello
` + "```",
			townRoot:  "/tmp/gt",
			wantCount: 1,
			wantFirst: "echo hello",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			commands := extractCommands(tt.description, tt.townRoot)
			if len(commands) != tt.wantCount {
				t.Errorf("got %d commands, want %d", len(commands), tt.wantCount)
				for i, c := range commands {
					t.Logf("  command[%d]: %q", i, c)
				}
			}
			if tt.wantFirst != "" && len(commands) > 0 {
				got := strings.TrimSpace(commands[0])
				if got != tt.wantFirst {
					t.Errorf("first command = %q, want %q", got, tt.wantFirst)
				}
			}
		})
	}
}

func TestIsCommentOnly(t *testing.T) {
	tests := []struct {
		block string
		want  bool
	}{
		{"# just a comment", true},
		{"# line 1\n# line 2", true},
		{"# comment\necho hello", false},
		{"echo hello", false},
		{"", true},
		{"  \n  \n  ", true},
	}

	for _, tt := range tests {
		got := isCommentOnly(tt.block)
		if got != tt.want {
			t.Errorf("isCommentOnly(%q) = %v, want %v", tt.block, got, tt.want)
		}
	}
}

func TestTruncateOutput(t *testing.T) {
	tests := []struct {
		s      string
		maxLen int
		want   string
	}{
		{"hello", 10, "hello"},
		{"hello world", 8, "hello..."},
		{"hi", 2, "hi"},
		{"abc", 3, "abc"},
	}

	for _, tt := range tests {
		got := truncateOutput(tt.s, tt.maxLen)
		if got != tt.want {
			t.Errorf("truncateOutput(%q, %d) = %q, want %q", tt.s, tt.maxLen, got, tt.want)
		}
	}
}

func TestMigrationCheckpointRoundTrip(t *testing.T) {
	tmpDir := t.TempDir()

	cp := &MigrationCheckpoint{
		FormulaVersion: 1,
		TownRoot:       tmpDir,
		Steps: map[string]StepRun{
			"detect": {
				ID:     "detect",
				Title:  "Detect current state",
				Status: "completed",
			},
			"backup": {
				ID:     "backup",
				Title:  "Backup all data",
				Status: "pending",
			},
		},
	}

	// Save
	if err := saveMigrationCheckpoint(tmpDir, cp); err != nil {
		t.Fatalf("save failed: %v", err)
	}

	// Verify file exists
	path := filepath.Join(tmpDir, migrationCheckpointFile)
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Fatal("checkpoint file not created")
	}

	// Load
	loaded, err := loadMigrationCheckpoint(tmpDir)
	if err != nil {
		t.Fatalf("load failed: %v", err)
	}

	if loaded.FormulaVersion != 1 {
		t.Errorf("formula version = %d, want 1", loaded.FormulaVersion)
	}
	if loaded.TownRoot != tmpDir {
		t.Errorf("town root = %q, want %q", loaded.TownRoot, tmpDir)
	}
	if len(loaded.Steps) != 2 {
		t.Errorf("steps count = %d, want 2", len(loaded.Steps))
	}
	if loaded.Steps["detect"].Status != "completed" {
		t.Errorf("detect status = %q, want completed", loaded.Steps["detect"].Status)
	}
	if loaded.Steps["backup"].Status != "pending" {
		t.Errorf("backup status = %q, want pending", loaded.Steps["backup"].Status)
	}
}
