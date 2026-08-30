package main

import (
	"encoding/json"
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

// Asking for help is an answered request, not a usage error: both forms
// exit clean and both answer on stdout, so `ctchargen new -h > flags.txt`
// captures the flag list the way `ctchargen --help > usage.txt` captures
// the usage text. Neither may put a word on stderr — that stream is for
// the usage errors, checked below.
func TestHelpExitsClean(t *testing.T) {
	code, stdout, stderr := runCmd(t, "", "--help")
	if code != exitOK {
		t.Errorf("--help = %d, want %d", code, exitOK)
	}

	if !strings.Contains(stdout, "usage:") {
		t.Errorf("--help stdout = %q, want the usage text", stdout)
	}

	if stderr != "" {
		t.Errorf("--help wrote to stderr: %q", stderr)
	}

	code, stdout, stderr = runCmd(t, "", "new", "-h")
	if code != exitOK {
		t.Errorf("new -h = %d, want %d", code, exitOK)
	}

	if !strings.Contains(stdout, "-seed") {
		t.Errorf("new -h did not print the flag list to stdout: %q", tail(stdout))
	}

	if stderr != "" {
		t.Errorf("new -h wrote to stderr: %q", stderr)
	}
}

// A parse error is the other half of the split: it is a usage error, so
// it goes to stderr with the flag list after it, and stdout stays clean
// for anything being piped.
func TestUnknownFlagIsAUsageError(t *testing.T) {
	code, stdout, stderr := runCmd(t, "", "new", "--nonesuch")
	if code != exitUsage {
		t.Errorf("new --nonesuch = %d, want %d", code, exitUsage)
	}

	if !strings.Contains(stderr, "not defined: -nonesuch") {
		t.Errorf("stderr does not name the bad flag: %q", tail(stderr))
	}

	if !strings.Contains(stderr, "-seed") {
		t.Errorf("stderr does not carry the flag list: %q", tail(stderr))
	}

	if stdout != "" {
		t.Errorf("a usage error wrote to stdout: %q", stdout)
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

// An occupied -o is refused before the first prompt, not after the last
// one: the collision is knowable up front, and discovering it afterwards
// throws away a whole interactive playthrough. Empty stdin is the
// discriminator — reaching the prompter at all would fail with "standard
// input closed" instead.
func TestNewRefusesAnOccupiedOutputBeforePrompting(t *testing.T) {
	path := filepath.Join(t.TempDir(), "taken.json")
	if err := os.WriteFile(path, []byte("original\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	code, _, stderr := runCmd(t, "", "new", "--seed", "7", "-o", path)
	if code != exitError {
		t.Fatalf("new -o over an existing file = %d, want %d", code, exitError)
	}

	if !strings.Contains(stderr, "--force") {
		t.Errorf("stderr %q does not name --force", tail(stderr))
	}

	if strings.Contains(stderr, "standard input closed") {
		t.Error("generation started before the output path was checked")
	}

	if strings.Contains(stderr, "which service") {
		t.Error("the player was prompted before the output path was checked")
	}

	data, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		t.Fatal(err)
	}

	if string(data) != "original\n" {
		t.Errorf("the existing file was touched: %q", data)
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

// The policy flags select how --auto decides, so a mistyped strategy and a
// strategy named without --auto are both usage errors. Silently ignoring
// either would hand back a character built by a policy the caller did not
// ask for, and the record would say so only if the caller went looking.
func TestPolicyFlagsAreValidated(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want string
	}{
		{"an unknown skills strategy", []string{"new", "--auto", "--skills", "bogus"}, "want one of"},
		{"an unknown muster strategy", []string{"new", "--auto", "--muster", "bogus"}, "want one of"},
		{"a career beyond the 7-term cap", []string{"new", "--auto", "--career", "9"}, "want max or a term 1-7"},
		{"a career that is not a number", []string{"new", "--auto", "--career", "later"}, "want max or a term 1-7"},
		{"a strategy without --auto", []string{"new", "--skills", "rounded"}, "need --auto"},
		{"batch validates them too", []string{"batch", "--count", "2", "--auto", "--skills", "bogus"}, "want one of"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			code, stdout, stderr := runCmd(t, "", tt.args...)
			if code != exitUsage {
				t.Errorf("exit %d, want %d (%q)", code, exitUsage, tail(stderr))
			}

			if !strings.Contains(stderr, tt.want) {
				t.Errorf("stderr %q does not say what is valid", tail(stderr))
			}

			if stdout != "" {
				t.Errorf("a usage error wrote a record to stdout: %q", tail(stdout))
			}
		})
	}
}

// A selected strategy has to reach both the generated character and the
// record's account of what was asked for.
func TestPolicyFlagsReachTheRecord(t *testing.T) {
	code, stdout, stderr := runCmd(t, "", "new", "--auto", "--seed", "3", "--service", "navy", "--skills", "rounded")
	if code != exitOK {
		t.Fatalf("new --skills rounded = %d, stderr %q", code, tail(stderr))
	}

	if !strings.Contains(stdout, `"skills": "rounded"`) {
		t.Error("the record's inputs do not name the strategy that generated it")
	}

	// Term 1 goes to Personal Development under `rounded`, which is the
	// whole difference from the default.
	if !strings.Contains(stdout, `"picked": "personal_development"`) {
		t.Error("no eligibility went to personal development")
	}

	// The default must stay clean: its inputs block carries no strategy
	// keys at all, so a version 3 record reads as a version 2 one did.
	// Checked against the parsed inputs, not the text — the character's own
	// top-level "skills" array would otherwise match.
	code, plain, stderr := runCmd(t, "", "new", "--auto", "--seed", "3", "--service", "navy")
	if code != exitOK {
		t.Fatalf("new = %d, stderr %q", code, tail(stderr))
	}

	var record struct {
		Inputs map[string]any `json:"inputs"`
	}

	if err := json.Unmarshal([]byte(plain), &record); err != nil {
		t.Fatalf("parsing the default record: %v", err)
	}

	for _, key := range []string{"skills", "muster", "career_terms"} {
		if _, present := record.Inputs[key]; present {
			t.Errorf("a default record carries inputs.%s", key)
		}
	}
}

// --career is the one policy flag taking a number rather than a strategy
// name, so nothing in the shared registry checks it and only its rejection
// paths were exercised: the two statements that turn the number into a
// config field were dark, and a --career that had quietly become a no-op
// would have left the whole suite green.
func TestCareerFlagShortensTheCareer(t *testing.T) {
	code, stdout, stderr := runCmd(t, "", "new", "--auto", "--seed", "3", "--service", "navy", "--career", "2")
	if code != exitOK {
		t.Fatalf("new --career 2 = %d, stderr %q", code, tail(stderr))
	}

	var record struct {
		Inputs struct {
			CareerTerms int `json:"career_terms"`
		} `json:"inputs"`
		Events []struct {
			Step   string `json:"step"`
			Label  string `json:"label"`
			Picked string `json:"picked"`
		} `json:"events"`
	}

	if err := json.Unmarshal([]byte(stdout), &record); err != nil {
		t.Fatalf("parsing the record: %v", err)
	}

	if record.Inputs.CareerTerms != 2 {
		t.Errorf("inputs.career_terms = %d, want 2", record.Inputs.CareerTerms)
	}

	// Intent, not outcome: the throw still decides, so what is asserted is
	// the answer the policy gave, not how many terms he actually served.
	intent := map[string]string{}

	for _, e := range record.Events {
		if e.Label == "reenlist-intent" {
			intent[e.Step] = e.Picked
		}
	}

	if intent["term-1"] != "yes" || intent["term-2"] != "no" {
		t.Errorf("reenlistment intents %v, want yes through term 1 and no at term 2", intent)
	}
}

// `max` is the documented default, so naming it must be indistinguishable
// from not naming it — down to the bytes, since the record would otherwise
// carry a career_terms the character never had.
func TestCareerMaxIsTheDefault(t *testing.T) {
	code, explicit, stderr := runCmd(t, "", "new", "--auto", "--seed", "3", "--service", "navy", "--career", "max")
	if code != exitOK {
		t.Fatalf("new --career max = %d, stderr %q", code, tail(stderr))
	}

	code, plain, stderr := runCmd(t, "", "new", "--auto", "--seed", "3", "--service", "navy")
	if code != exitOK {
		t.Fatalf("new = %d, stderr %q", code, tail(stderr))
	}

	if explicit != plain {
		t.Error("--career max produced a different record from the default")
	}
}

// Everything past `render`'s argument count was dark: the file read, the
// parse, both renderers, and the write. Assertions are on substance the
// sheet and the transcript must carry — not on their wording, which would
// only make this a second copy of the render goldens.
func TestRenderReadsARecord(t *testing.T) {
	path := filepath.Join(t.TempDir(), "char.json")

	if code, _, stderr := runCmd(t, "", "new", "--auto", "--seed", "3", "--service", "navy", "-o", path); code != exitOK {
		t.Fatalf("new: %q", stderr)
	}

	code, sheet, stderr := runCmd(t, "", "render", path)
	if code != exitOK {
		t.Fatalf("render = %d, stderr %q", code, tail(stderr))
	}

	for _, want := range []string{"UPP", "Navy", "## Skills"} {
		if !strings.Contains(sheet, want) {
			t.Errorf("sheet does not carry %q:\n%s", want, sheet)
		}
	}

	code, history, stderr := runCmd(t, "", "render", "--history", path)
	if code != exitOK {
		t.Fatalf("render --history = %d, stderr %q", code, tail(stderr))
	}

	for _, want := range []string{"# Generation record", "Seed 3", "## characteristics", "- (1)"} {
		if !strings.Contains(history, want) {
			t.Errorf("history does not carry %q:\n%s", want, tail(history))
		}
	}

	if history == sheet {
		t.Error("--history rendered the sheet")
	}
}

func TestRenderRejectsBadInput(t *testing.T) {
	dir := t.TempDir()

	notARecord := filepath.Join(dir, "garbage.json")
	if err := os.WriteFile(notARecord, []byte("{\"nope\": true}\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name string
		path string
		want string
	}{
		{"a file that is not there", filepath.Join(dir, "absent.json"), "absent.json"},
		{"a file that is not a record", notARecord, "parsing character record"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			code, stdout, stderr := runCmd(t, "", "render", tt.path)
			if code != exitError {
				t.Errorf("render = %d, want %d", code, exitError)
			}

			if !strings.Contains(stderr, tt.want) {
				t.Errorf("stderr %q does not explain the failure", tail(stderr))
			}

			// A failed render must not put half a sheet on the pipe.
			if stdout != "" {
				t.Errorf("failed render wrote to stdout: %q", stdout)
			}
		})
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
