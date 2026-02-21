package cmd

import (
	"strings"
	"testing"
)

func TestEnsureKnownConvoyStatus(t *testing.T) {
	t.Parallel()

	if err := ensureKnownConvoyStatus("open"); err != nil {
		t.Fatalf("expected open to be accepted: %v", err)
	}
	if err := ensureKnownConvoyStatus(" closed "); err != nil {
		t.Fatalf("expected closed to be accepted: %v", err)
	}
	if err := ensureKnownConvoyStatus("in_progress"); err == nil {
		t.Fatal("expected unknown status to be rejected")
	}
}

func TestValidateConvoyStatusTransition(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		current string
		target  string
		wantErr bool
	}{
		{name: "open to closed", current: "open", target: "closed", wantErr: false},
		{name: "closed to open", current: "closed", target: "open", wantErr: false},
		{name: "same open", current: "open", target: "open", wantErr: false},
		{name: "same closed", current: "closed", target: "closed", wantErr: false},
		{name: "unknown current", current: "in_progress", target: "closed", wantErr: true},
		{name: "unknown target", current: "open", target: "archived", wantErr: true},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := validateConvoyStatusTransition(tc.current, tc.target)
			if tc.wantErr && err == nil {
				t.Fatalf("expected error for transition %q -> %q", tc.current, tc.target)
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("expected transition %q -> %q to pass, got %v", tc.current, tc.target, err)
			}
		})
	}
}

func TestEnsureKnownConvoyStatus_Staged(t *testing.T) {
	t.Parallel()

	// staged:ready should be accepted
	if err := ensureKnownConvoyStatus("staged:ready"); err != nil {
		t.Fatalf("expected staged:ready to be accepted: %v", err)
	}

	// staged:warnings should be accepted
	if err := ensureKnownConvoyStatus("staged:warnings"); err != nil {
		t.Fatalf("expected staged:warnings to be accepted: %v", err)
	}

	// staged:unknown should be rejected
	if err := ensureKnownConvoyStatus("staged:unknown"); err == nil {
		t.Fatal("expected staged:unknown to be rejected")
	}

	// STAGED:READY (uppercase) should be accepted via normalization
	if err := ensureKnownConvoyStatus("STAGED:READY"); err != nil {
		t.Fatalf("expected STAGED:READY to be accepted (normalization): %v", err)
	}

	// Verify error message includes all valid statuses
	err := ensureKnownConvoyStatus("bogus")
	if err == nil {
		t.Fatal("expected bogus to be rejected")
	}
	msg := err.Error()
	for _, want := range []string{"open", "closed", "staged:ready", "staged:warnings"} {
		if !strings.Contains(msg, want) {
			t.Errorf("error message %q should mention %q", msg, want)
		}
	}
}

func TestValidateConvoyStatusTransition_Staged(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		current string
		target  string
		wantErr bool
	}{
		// staged → open (launch)
		{name: "staged:ready to open", current: "staged:ready", target: "open", wantErr: false},
		{name: "staged:warnings to open", current: "staged:warnings", target: "open", wantErr: false},

		// staged → closed (cancel)
		{name: "staged:ready to closed", current: "staged:ready", target: "closed", wantErr: false},
		{name: "staged:warnings to closed", current: "staged:warnings", target: "closed", wantErr: false},

		// staged identity
		{name: "staged:ready to staged:ready", current: "staged:ready", target: "staged:ready", wantErr: false},
		{name: "staged:warnings to staged:warnings", current: "staged:warnings", target: "staged:warnings", wantErr: false},

		// staged ↔ staged (re-stage)
		{name: "staged:ready to staged:warnings", current: "staged:ready", target: "staged:warnings", wantErr: false},
		{name: "staged:warnings to staged:ready", current: "staged:warnings", target: "staged:ready", wantErr: false},

		// REJECTED: open → staged:*
		{name: "open to staged:ready rejected", current: "open", target: "staged:ready", wantErr: true},
		{name: "open to staged:warnings rejected", current: "open", target: "staged:warnings", wantErr: true},

		// REJECTED: closed → staged:*
		{name: "closed to staged:ready rejected", current: "closed", target: "staged:ready", wantErr: true},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := validateConvoyStatusTransition(tc.current, tc.target)
			if tc.wantErr && err == nil {
				t.Fatalf("expected error for transition %q -> %q", tc.current, tc.target)
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("expected transition %q -> %q to pass, got %v", tc.current, tc.target, err)
			}
		})
	}
}
