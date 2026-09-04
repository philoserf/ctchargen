package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"maps"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"
)

// printedCommand finds the command a rendering offers.
//
// It matches the prose around the span rather than the span's own fence,
// because the fence widens to hold a command carrying a backtick and Go's
// regexp has no backreference to match an opening run against its close.
var printedCommand = regexp.MustCompile(`(?m)^Regenerate with (.+?)(?:, on .+)?\.$`)

// commandIn pulls the command out of a rendering, unwrapping the code span.
func commandIn(t *testing.T, rendering string) (string, bool) {
	t.Helper()

	found := printedCommand.FindStringSubmatch(rendering)
	if found == nil {
		return "", false
	}

	return strings.TrimSpace(strings.Trim(found[1], "`")), true
}

// The command a rendering prints reproduces that rendering, byte for byte.
//
// This is what pays for the render package knowing how this command line is
// spelled. Nothing in the compiler holds those flag names to these flag sets;
// this does, by running what render wrote. It is a fixpoint - the regenerated
// output carries the same line - so byte-identity is the assertion, not a
// prefix.
//
// The name is here because it is the one value that is not a bare token. A
// footer that printed it unquoted would come back as two arguments, and the
// character would lose his surname.
func TestAPrintedCommandReproducesWhatPrintedIt(t *testing.T) {
	t.Parallel()

	for name, args := range map[string][]string{
		"sheet":            {cmdNew, flagAuto, flagSeed, "145", flagService, merchants, flagSheet},
		"transcript":       {cmdNew, flagAuto, flagSeed, "145", flagService, merchants, flagHistory},
		"unforced service": {cmdNew, flagAuto, flagSeed, "7", flagSheet},
		"a name with a space": {
			cmdNew, flagAuto, flagSeed, "4", flagService, other,
			flagName, "Alexander Jamison", flagSheet,
		},
		"strategies that are not the defaults": {
			cmdNew, flagAuto, flagSeed, "4", flagService, navy,
			"--career", "oneterm", "--skills", "personal", "--muster", "spartan", flagSheet,
		},
	} {
		var first strings.Builder

		err := run(args, nil, &first, io.Discard)
		if err != nil {
			t.Errorf("%s: %v", name, err)

			continue
		}

		printed, found := commandIn(t, first.String())
		if !found {
			t.Errorf("%s: no command in\n%s", name, first.String())

			continue
		}

		again, err := shellFields(printed)
		if err != nil {
			t.Errorf("%s: %v", name, err)

			continue
		}

		if len(again) == 0 || again[0] != "ctchargen" {
			t.Errorf("%s: the line does not name the tool: %q", name, printed)

			continue
		}

		var second strings.Builder

		err = run(again[1:], nil, &second, io.Discard)
		if err != nil {
			t.Errorf("%s: running %q: %v", name, printed, err)

			continue
		}

		if first.String() != second.String() {
			t.Errorf("%s: %q did not reproduce what printed it:\n--- first\n%s\n--- again\n%s",
				name, printed, first.String(), second.String())
		}
	}
}

// The three strategies are named whether or not they are the defaults, so
// that a line kept on paper still reproduces its character the day a default
// moves. That day is not hypothetical: the alpha.2 release notes document a
// --skills default that has since changed.
func TestAPrintedCommandNamesEveryStrategy(t *testing.T) {
	t.Parallel()

	var out strings.Builder

	err := run([]string{cmdNew, flagAuto, flagSeed, "145", flagService, merchants, flagSheet},
		nil, &out, io.Discard)
	if err != nil {
		t.Fatalf("generating: %v", err)
	}

	printed, found := commandIn(t, out.String())
	if !found {
		t.Fatalf("no command in\n%s", out.String())
	}

	for _, want := range []string{
		flagCareer, "serve", flagSkills, "advanced", flagMuster, "cash",
	} {
		if !strings.Contains(printed, want) {
			t.Errorf("%q does not name %q", printed, want)
		}
	}
}

// A character the player answered for offers no command, because none would
// work: the seed replays the dice, not the answers.
func TestAnAnsweredCharacterIsNotOfferedACommand(t *testing.T) {
	t.Parallel()

	var out strings.Builder

	err := run([]string{cmdNew, flagSeed, "145", flagService, other, flagSheet},
		answers("1", 300), &out, io.Discard)
	if err != nil {
		t.Fatalf("generating: %v", err)
	}

	if printedCommand.MatchString(out.String()) {
		t.Errorf("a sheet the player answered for offers a command:\n%s", out.String())
	}

	for _, want := range []string{"Seed 145", "player", "does not bring this character back"} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("the footer does not mention %q:\n%s", want, out.String())
		}
	}
}

// errUnbalanced reports a printed command whose quoting does not close.
var errUnbalanced = errors.New("unbalanced quote")

// shellFields splits a printed command the way a shell would.
//
// It reads single quotes, because that is what the line is written with: a
// shell expands a backtick inside double quotes and expands nothing at all
// inside single ones, so a line meant to be pasted has to use the latter.
// A quote within them is closed, escaped and reopened - 'O'\”Brien' - which
// means a token is a run of quoted and unquoted pieces rather than one span,
// and that is why this scans rather than cutting on spaces.
func shellFields(line string) ([]string, error) {
	var (
		fields  []string
		token   strings.Builder
		started bool
	)

	for i := 0; i < len(line); {
		switch char := line[i]; {
		case char == ' ':
			if started {
				fields = append(fields, token.String())
				token.Reset()

				started = false
			}

			i++
		case char == '\'':
			end := strings.IndexByte(line[i+1:], '\'')
			if end < 0 {
				return nil, fmt.Errorf("%w in %q", errUnbalanced, line)
			}

			token.WriteString(line[i+1 : i+1+end])

			started = true

			i += end + 2
		case char == '\\' && i+1 < len(line):
			token.WriteByte(line[i+1])

			started = true

			i += 2
		default:
			token.WriteByte(char)

			started = true

			i++
		}
	}

	if started {
		fields = append(fields, token.String())
	}

	return fields, nil
}

// A record cannot put words in the operator's shell.
//
// render reads whatever it is handed - PRERELEASE.md says so deliberately,
// and it is right for a value that is only displayed. This line is different:
// the tool tells the reader to paste it. Character records exist to be
// shared, so a referee rendering one a player sent him is the ordinary path,
// not an exotic one.
//
// The assertion is that each hostile value arrives as exactly ONE argument.
// That is the property that matters: a value which reaches the tool whole is
// then refused by the tool's own validation, which is the right place for it
// to fail.
func TestARecordCannotSmuggleArgumentsIntoTheLine(t *testing.T) {
	t.Parallel()

	const (
		smuggledFlags = "other --force -o /etc/hosts"
		substitution  = "serve`touch /tmp/ctchargen-should-not-exist`"
		anApostrophe  = "O'Brien"
	)

	path := recordWithInputs(t, map[string]any{
		"service": smuggledFlags,
		"career":  substitution,
		"name":    anApostrophe,
	})

	var out strings.Builder

	err := run([]string{cmdRender, path}, nil, &out, io.Discard)
	if err != nil {
		t.Fatalf("rendering: %v", err)
	}

	printed, found := commandIn(t, out.String())
	if !found {
		t.Fatalf("no command in\n%s", out.String())
	}

	got, err := shellFields(printed)
	if err != nil {
		t.Fatalf("%v", err)
	}

	// Each value whole, and in the position its flag put it.
	for flag, want := range map[string]string{
		flagService: smuggledFlags,
		flagCareer:  substitution,
		flagName:    anApostrophe,
	} {
		at := slices.Index(got, flag)
		if at < 0 || at+1 >= len(got) {
			t.Errorf("%s is not in the line: %q", flag, printed)

			continue
		}

		if got[at+1] != want {
			t.Errorf("%s arrived as %q, want the whole of %q", flag, got[at+1], want)
		}
	}
}

// recordWithInputs writes a real record and then edits its inputs, which is
// how a hostile one would reach the tool: nothing here writes these values,
// and render reads what it is handed.
func recordWithInputs(t *testing.T, doctored map[string]any) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "hostile.json")

	err := run([]string{cmdNew, flagAuto, flagSeed, "7", flagService, other, flagOutput, path},
		nil, io.Discard, io.Discard)
	if err != nil {
		t.Fatalf("writing a record: %v", err)
	}

	record := map[string]any{}
	read(t, path, &record)

	inputs, ok := record["inputs"].(map[string]any)
	if !ok {
		t.Fatalf("the record carries no inputs: %v", record["inputs"])
	}

	maps.Copy(inputs, doctored)

	write(t, path, record)

	return path
}

func read(t *testing.T, path string, into any) {
	t.Helper()

	text, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading %s: %v", path, err)
	}

	err = json.Unmarshal(text, into)
	if err != nil {
		t.Fatalf("reading %s: %v", path, err)
	}
}

func write(t *testing.T, path string, from any) {
	t.Helper()

	text, err := json.Marshal(from)
	if err != nil {
		t.Fatalf("writing %s: %v", path, err)
	}

	err = os.WriteFile(path, text, 0o600)
	if err != nil {
		t.Fatalf("writing %s: %v", path, err)
	}
}
