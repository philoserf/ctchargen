package main

import (
	"strings"
	"testing"
)

func fixedSeed() (uint64, error) { return 1, nil }

func TestRun(t *testing.T) {
	tests := []struct {
		name       string
		args       []string
		wantCode   int
		wantStderr string
	}{
		{"no args prints usage", nil, exitUsage, "usage:"},
		{"unknown command", []string{"bogus"}, exitUsage, `unknown command "bogus"`},
		{"replay not yet implemented", []string{"replay"}, exitError, "not yet implemented"},
		{"new without auto", []string{"new"}, exitError, "interactive mode is not yet implemented"},
		{"new rejects stray argument", []string{"new", "--auto", "out.json"}, exitUsage, "flags precede"},
		{"render wants a file", []string{"render"}, exitUsage, "exactly one character.json"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stdout, stderr strings.Builder
			if got := run(tt.args, fixedSeed, &stdout, &stderr); got != tt.wantCode {
				t.Errorf("run(%v) = %d, want %d", tt.args, got, tt.wantCode)
			}

			if !strings.Contains(stderr.String(), tt.wantStderr) {
				t.Errorf("run(%v) stderr = %q, want it to contain %q", tt.args, stderr.String(), tt.wantStderr)
			}
		})
	}
}

func TestNewEmitsARecord(t *testing.T) {
	var stdout, stderr strings.Builder
	if got := run([]string{"new", "--auto", "--seed", "1"}, fixedSeed, &stdout, &stderr); got != exitOK {
		t.Fatalf("new --auto --seed 1 = %d, stderr %q", got, stderr.String())
	}

	for _, want := range []string{`"schema_version"`, `"upp"`, `"events"`, `"seed": 1`} {
		if !strings.Contains(stdout.String(), want) {
			t.Errorf("record missing %s", want)
		}
	}
}

func TestVersion(t *testing.T) {
	var stdout, stderr strings.Builder
	if got := run([]string{"version"}, fixedSeed, &stdout, &stderr); got != exitOK {
		t.Fatalf("version = %d", got)
	}

	for _, want := range []string{"schema_version", "engine_version", "policy_version", "ruleset", "rng"} {
		if !strings.Contains(stdout.String(), want) {
			t.Errorf("version output missing %s", want)
		}
	}
}
