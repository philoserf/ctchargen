package main

import (
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

// printedCommand matches the command a rendering offers, which the render
// package writes between backticks.
var printedCommand = regexp.MustCompile("`(ctchargen [^`]*)`")

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

		printed := printedCommand.FindStringSubmatch(first.String())
		if printed == nil {
			t.Errorf("%s: no command in\n%s", name, first.String())

			continue
		}

		again, err := shellFields(printed[1])
		if err != nil {
			t.Errorf("%s: %v", name, err)

			continue
		}

		if len(again) == 0 || again[0] != "ctchargen" {
			t.Errorf("%s: the line does not name the tool: %q", name, printed[1])

			continue
		}

		var second strings.Builder

		err = run(again[1:], nil, &second, io.Discard)
		if err != nil {
			t.Errorf("%s: running %q: %v", name, printed[1], err)

			continue
		}

		if first.String() != second.String() {
			t.Errorf("%s: %q did not reproduce what printed it:\n--- first\n%s\n--- again\n%s",
				name, printed[1], first.String(), second.String())
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

	printed := printedCommand.FindStringSubmatch(out.String())
	if printed == nil {
		t.Fatalf("no command in\n%s", out.String())
	}

	for _, want := range []string{
		flagCareer, "serve", flagSkills, "advanced", flagMuster, "cash",
	} {
		if !strings.Contains(printed[1], want) {
			t.Errorf("%q does not name %q", printed[1], want)
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

// shellFields splits a printed command the way a shell would.
//
// strings.Fields would do for every argument but one: a name is printed
// quoted because it can hold a space, and splitting on whitespace would make
// it two arguments and a stray quote. Reading the quotes here is what lets
// the test above notice if the printing stopped writing them.
func shellFields(line string) ([]string, error) {
	var fields []string

	for rest := strings.TrimSpace(line); rest != ""; rest = strings.TrimSpace(rest) {
		if strings.HasPrefix(rest, `"`) {
			quoted, err := strconv.QuotedPrefix(rest)
			if err != nil {
				return nil, fmt.Errorf("unbalanced quote in %q: %w", line, err)
			}

			value, err := strconv.Unquote(quoted)
			if err != nil {
				return nil, fmt.Errorf("reading %s in %q: %w", quoted, line, err)
			}

			fields = append(fields, value)
			rest = rest[len(quoted):]

			continue
		}

		field, remainder, _ := strings.Cut(rest, " ")

		fields = append(fields, field)
		rest = remainder
	}

	return fields, nil
}
