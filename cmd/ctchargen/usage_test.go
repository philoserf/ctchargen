package main

import (
	"errors"
	"io"
	"strings"
	"testing"
)

// The help's own words, and the flags each subcommand offers, typed out here
// rather than read back off the flag set the printer walks. Reading the set
// would agree with the printer however wrong both were; a second list is
// what notices a flag the printer stopped writing.
const (
	flagHelp  = "--help"
	flagShort = "-h"
	cmdHelp   = "help"
	wantTypo  = "extra-arg"
)

// The three flag lists, in no order, because the help prints them sorted and
// this is about which flags are there.
var (
	newFlags = []string{
		flagSeed, flagAuto, flagService, flagName, flagCareer, flagSkills,
		flagMuster, flagSheet, flagHistory, flagOutput, flagForce,
	}
	batchFlags = []string{
		flagCount, flagSeed, flagAuto, flagService, flagName, flagCareer,
		flagSkills, flagMuster, flagOutput, flagForce,
	}
	renderFlags = []string{flagHistory, flagOutput, flagForce}
)

// Help is not an error: it exits 0, and it names every flag the command has.
//
// Both spellings, because the referee who reported this tried `new --help`
// and `batch -h` and got the same "help requested" line from each.
func TestHelpNamesEveryFlag(t *testing.T) {
	t.Parallel()

	for name, tc := range map[string]struct {
		command string
		flags   []string
	}{
		cmdNew:    {cmdNew, newFlags},
		cmdBatch:  {cmdBatch, batchFlags},
		cmdRender: {cmdRender, renderFlags},
	} {
		for _, asked := range []string{flagHelp, flagShort} {
			var out strings.Builder

			err := run([]string{tc.command, asked}, nil, &out, io.Discard)
			if err != nil {
				t.Errorf("%s %s: %v", name, asked, err)

				continue
			}

			help := out.String()

			if !strings.Contains(help, tc.command+" [flags]") &&
				!strings.HasPrefix(help, "usage: ctchargen "+tc.command) {
				t.Errorf("%s %s: no usage line: %q", name, asked, help)
			}

			for _, flag := range tc.flags {
				if !strings.Contains(help, flag+" ") {
					t.Errorf("%s %s: the help does not name %s", name, asked, flag)
				}
			}
		}
	}
}

// The values a strategy flag takes, and the one it takes when left alone.
//
// This is where a `go install` user learns them: the rejection message names
// POLICY.md, which is a file he does not have, and the release notes he read
// document three strategies that were renamed under him.
func TestHelpNamesTheStrategiesAndTheirDefaults(t *testing.T) {
	t.Parallel()

	var out strings.Builder

	err := run([]string{cmdNew, flagHelp}, nil, &out, io.Discard)
	if err != nil {
		t.Fatalf("new --help: %v", err)
	}

	for _, want := range []string{
		"serve, retire, oneterm", "advanced, service, personal",
		"cash, goods, spartan", "(default serve)", "(default advanced)",
		"(default cash)", merchants, navy,
	} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("the help does not mention %q", want)
		}
	}
}

// Help asked for generates nothing. `new` with no --auto and nothing to read
// from otherwise reports that the input ended, so a help that reached the
// procedure would say so.
func TestHelpDoesNotGenerate(t *testing.T) {
	t.Parallel()

	var out, asking strings.Builder

	err := run([]string{cmdNew, flagHelp}, nil, &out, &asking)
	if err != nil {
		t.Fatalf("new --help: %v", err)
	}

	if asking.Len() > 0 {
		t.Errorf("new --help asked a question: %q", asking.String())
	}

	if strings.Contains(out.String(), "UPP") {
		t.Errorf("new --help rolled a character: %q", out.String())
	}
}

// --help before a subcommand is named is a request, not a mistyped command.
func TestTopLevelHelpNamesEverySubcommand(t *testing.T) {
	t.Parallel()

	for _, asked := range []string{flagHelp, flagShort, cmdHelp} {
		var out strings.Builder

		err := run([]string{asked}, nil, &out, io.Discard)
		if err != nil {
			t.Errorf("ctchargen %s: %v", asked, err)

			continue
		}

		for _, command := range []string{cmdNew, cmdBatch, cmdRender, cmdVersion} {
			if !strings.Contains(out.String(), command) {
				t.Errorf("ctchargen %s: does not name %s", asked, command)
			}
		}
	}
}

// A positional argument is refused rather than ignored. `new extra-arg` used
// to start an interactive session on a drawn seed, so a typo became a
// session; and because the flag package stops at the first non-flag word,
// `new foo --auto` was an interactive session too.
func TestAPositionalArgumentIsRefused(t *testing.T) {
	t.Parallel()

	for name, args := range map[string][]string{
		cmdNew:            {cmdNew, wantTypo},
		"new before auto": {cmdNew, wantTypo, flagAuto},
		cmdBatch:          {cmdBatch, flagAuto, flagCount, "1", wantTypo},
	} {
		err := run(args, nil, io.Discard, io.Discard)
		if err == nil {
			t.Errorf("%s: run(%v) was accepted", name, args)

			continue
		}

		if !strings.Contains(err.Error(), wantTypo) {
			t.Errorf("%s: error %q does not name the argument", name, err)
		}
	}
}

// The two messages that list the subcommands read the same const, so they
// cannot disagree about the order again.
func TestTheSubcommandListsAgree(t *testing.T) {
	t.Parallel()

	for name, args := range map[string][]string{
		"no command": nil,
		wantUnknown:  {"generate"},
	} {
		err := run(args, nil, io.Discard, io.Discard)
		if err == nil {
			t.Errorf("%s: run(%v) was accepted", name, args)

			continue
		}

		if !strings.Contains(err.Error(), commandList) {
			t.Errorf("%s: error %q does not list %q", name, err, commandList)
		}
	}
}

// haltingPipe lets one write through and fails after, so that the flag list
// meets a closed pipe the usage line got through.
type haltingPipe struct{ wrote bool }

func (h *haltingPipe) Write(text []byte) (int, error) {
	if h.wrote {
		return 0, errClosedPipe
	}

	h.wrote = true

	return len(text), nil
}

// A help nobody received is not a help that was given. Exiting 0 on a write
// that failed would tell a script the flags had been printed, and the whole
// point of this issue is a reader who cannot see them.
func TestHelpReportsAWriteThatFailed(t *testing.T) {
	t.Parallel()

	for name, tc := range map[string]struct {
		args []string
		out  io.Writer
	}{
		"top level":       {[]string{flagHelp}, brokenPipe{}},
		cmdNew:            {[]string{cmdNew, flagHelp}, brokenPipe{}},
		"the flags alone": {[]string{cmdNew, flagHelp}, &haltingPipe{}},
	} {
		err := run(tc.args, nil, tc.out, io.Discard)

		switch {
		case err == nil:
			t.Errorf("%s: a help nobody received was reported as written", name)
		case !errors.Is(err, errClosedPipe):
			t.Errorf("%s: error %q does not carry the write failure", name, err)
		}
	}
}
