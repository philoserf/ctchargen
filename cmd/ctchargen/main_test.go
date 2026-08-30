package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func fixedSeed() (uint64, error) { return 1, nil }

func runCmd(t *testing.T, stdin string, args ...string) (int, string, string) {
	t.Helper()

	var stdout, stderr strings.Builder

	code := run(args, fixedSeed, strings.NewReader(stdin), &stdout, &stderr)

	return code, stdout.String(), stderr.String()
}

func TestRun(t *testing.T) {
	tests := []struct {
		name       string
		args       []string
		wantCode   int
		wantStderr string
	}{
		{"no args prints usage", nil, exitUsage, "usage:"},
		{"unknown command", []string{"bogus"}, exitUsage, `unknown command "bogus"`},
		{"new rejects stray argument", []string{"new", "--auto", "out.json"}, exitUsage, "flags precede"},
		{"new rejects unknown service", []string{"new", "--auto", "--service", "pirates"}, exitError, "not available"},
		{"batch requires count and auto", []string{"batch", "--count", "3"}, exitUsage, "requires --count N"},
		{"render wants a file", []string{"render"}, exitUsage, "exactly one character.json"},
		{"replay wants a file", []string{"replay"}, exitUsage, "exactly one character.json"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			code, _, stderr := runCmd(t, "", tt.args...)
			if code != tt.wantCode {
				t.Errorf("run(%v) = %d, want %d", tt.args, code, tt.wantCode)
			}

			if !strings.Contains(stderr, tt.wantStderr) {
				t.Errorf("run(%v) stderr = %q, want it to contain %q", tt.args, stderr, tt.wantStderr)
			}
		})
	}
}

// Asking for help is an answered request, not a usage error. The two
// forms print to different places — the top level writes usage to stdout,
// while a subcommand's flag list comes from the flag package, which writes
// to the flag set's output — so the exit code is what both share.
func TestHelpExitsClean(t *testing.T) {
	code, stdout, _ := runCmd(t, "", "--help")
	if code != exitOK {
		t.Errorf("--help = %d, want %d", code, exitOK)
	}

	if !strings.Contains(stdout, "usage:") {
		t.Errorf("--help stdout = %q, want the usage text", stdout)
	}

	code, _, stderr := runCmd(t, "", "new", "-h")
	if code != exitOK {
		t.Errorf("new -h = %d, want %d", code, exitOK)
	}

	if !strings.Contains(stderr, "-seed") {
		t.Errorf("new -h did not print the flag list: %q", tail(stderr))
	}
}

func TestNewEmitsARecord(t *testing.T) {
	code, stdout, stderr := runCmd(t, "", "new", "--auto", "--seed", "1")
	if code != exitOK {
		t.Fatalf("new --auto --seed 1 = %d, stderr %q", code, stderr)
	}

	for _, want := range []string{`"schema_version"`, `"upp"`, `"events"`, `"seed": 1`} {
		if !strings.Contains(stdout, want) {
			t.Errorf("record missing %s", want)
		}
	}
}

// Interactive mode walks the procedure step by step; answering "1" to
// everything is a complete, deterministic playthrough for a fixed seed.
func TestNewInteractive(t *testing.T) {
	answers := strings.Repeat("1\n", 500)

	code, stdout, stderr := runCmd(t, answers, "new", "--seed", "7")
	if code != exitOK {
		t.Fatalf("interactive new = %d, stderr tail %q", code, tail(stderr))
	}

	if !strings.Contains(stdout, `"by": "player"`) {
		t.Error("interactive record has no player choices")
	}

	if !strings.Contains(stderr, "which service does the character attempt to enlist in?") {
		t.Error("no service prompt shown")
	}
}

func TestNewInteractiveRepromptsOnBadInput(t *testing.T) {
	answers := "bogus\n99\n" + strings.Repeat("1\n", 500)

	code, _, stderr := runCmd(t, answers, "new", "--seed", "7")
	if code != exitOK {
		t.Fatalf("interactive new = %d", code)
	}

	if !strings.Contains(stderr, "pick 1-") {
		t.Error("invalid input did not reprompt")
	}
}

func TestNewInteractiveInputClosed(t *testing.T) {
	code, _, stderr := runCmd(t, "", "new", "--seed", "7")
	if code != exitError {
		t.Fatalf("interactive new with closed stdin = %d, want %d", code, exitError)
	}

	if !strings.Contains(stderr, "standard input closed") {
		t.Errorf("stderr %q does not explain the closed input", tail(stderr))
	}
}

func TestBatchJSONLDerivesSeeds(t *testing.T) {
	code, stdout, stderr := runCmd(t, "", "batch", "--count", "3", "--auto", "--seed", "10")
	if code != exitOK {
		t.Fatalf("batch = %d, stderr %q", code, stderr)
	}

	lines := strings.Split(strings.TrimSpace(stdout), "\n")
	if len(lines) != 3 {
		t.Fatalf("%d JSONL lines, want 3", len(lines))
	}

	for i, want := range []string{`"seed":10`, `"seed":11`, `"seed":12`} {
		if !strings.Contains(lines[i], want) {
			t.Errorf("member %d missing %s", i, want)
		}
	}
}

func TestBatchToDirectory(t *testing.T) {
	dir := t.TempDir()

	code, _, stderr := runCmd(t, "", "batch", "--count", "2", "--auto", "--seed", "10", "-o", dir)
	if code != exitOK {
		t.Fatalf("batch -o dir = %d, stderr %q", code, stderr)
	}

	for _, name := range []string{"character-0000.json", "character-0001.json"} {
		if _, err := os.Stat(filepath.Join(dir, name)); err != nil {
			t.Errorf("expected %s: %v", name, err)
		}
	}

	// Existing files are never overwritten without --force.
	code, _, stderr = runCmd(t, "", "batch", "--count", "2", "--auto", "--seed", "10", "-o", dir)
	if code != exitError || !strings.Contains(stderr, "--force") {
		t.Errorf("re-run without --force = %d (%q), want refusal", code, tail(stderr))
	}
}

// A collision anywhere in the run must leave the directory as it was: the
// refusal names every colliding file, and none of the members that would
// not have collided are written.
func TestBatchToDirectoryIsAllOrNothing(t *testing.T) {
	dir := t.TempDir()

	// Seed the directory with the second and fourth members' names only.
	for _, name := range []string{"character-0001.json", "character-0003.json"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("{}\n"), 0o600); err != nil {
			t.Fatal(err)
		}
	}

	code, _, stderr := runCmd(t, "", "batch", "--count", "5", "--auto", "--seed", "10", "-o", dir)
	if code != exitError {
		t.Fatalf("batch over existing files = %d, want refusal", code)
	}

	for _, name := range []string{"character-0001.json", "character-0003.json"} {
		if !strings.Contains(stderr, name) {
			t.Errorf("refusal does not name %s: %q", name, tail(stderr))
		}
	}

	for _, name := range []string{"character-0000.json", "character-0002.json", "character-0004.json"} {
		if _, err := os.Stat(filepath.Join(dir, name)); err == nil {
			t.Errorf("%s was written despite the run being refused", name)
		}
	}
}

func TestReplayRoundTripViaCLI(t *testing.T) {
	path := filepath.Join(t.TempDir(), "char.json")

	if code, _, stderr := runCmd(t, "", "new", "--auto", "--seed", "3", "-o", path); code != exitOK {
		t.Fatalf("new: %q", stderr)
	}

	code, stdout, stderr := runCmd(t, "", "replay", path)
	if code != exitOK || !strings.Contains(stdout, "replay verified") {
		t.Fatalf("replay = %d, stdout %q, stderr %q", code, stdout, stderr)
	}

	// Tamper with a recorded total and replay must fail, naming the event.
	data, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		t.Fatal(err)
	}

	tampered := strings.Replace(string(data), `"total": `, `"total": 1`, 1)

	// The path is this test's own t.TempDir file, not tainted input.
	if err := os.WriteFile(filepath.Clean(path), []byte(tampered), 0o600); err != nil { // #nosec G703
		t.Fatal(err)
	}

	code, _, stderr = runCmd(t, "", "replay", path)
	if code != exitError || !strings.Contains(stderr, "diverge") {
		t.Errorf("tampered replay = %d (%q), want divergence", code, tail(stderr))
	}
}

func TestVersion(t *testing.T) {
	code, stdout, _ := runCmd(t, "", "version")
	if code != exitOK {
		t.Fatalf("version = %d", code)
	}

	for _, want := range []string{"schema_version", "engine_version", "policy_version", "ruleset", "rng"} {
		if !strings.Contains(stdout, want) {
			t.Errorf("version output missing %s", want)
		}
	}
}

func tail(s string) string {
	if len(s) > 300 {
		return "…" + s[len(s)-300:]
	}

	return s
}
