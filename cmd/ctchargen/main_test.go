package main

import (
	"io"
	"strings"
	"testing"
)

// The command line's own words, so a typo in one place is a compile error
// rather than a test that quietly stops testing the flag it names.
const (
	cmdNew      = "new"
	flagAuto    = "--auto"
	flagSeed    = "--seed"
	flagService = "--service"
	other       = "other"
)

func TestRunRejectsBadCommandLines(t *testing.T) {
	t.Parallel()

	for name, tc := range map[string]struct {
		args     []string
		mentions string
	}{
		"no command":           {nil, "new|render|version"},
		"unknown command":      {[]string{"generate"}, "unknown command"},
		"no --auto":            {[]string{cmdNew}, "interactive mode arrives at milestone 4"},
		"unknown flag":         {[]string{cmdNew, flagAuto, "--wat"}, "usage"},
		"no such service":      {[]string{cmdNew, flagAuto, flagService, "navvy"}, "no service is called"},
		"no such strategy":     {[]string{cmdNew, flagAuto, "--career", "dawdle"}, "no such strategy"},
		"a service with ranks": {[]string{cmdNew, flagAuto, flagSeed, "1", flagService, "navy"}, "milestone 2"},
	} {
		err := run(tc.args, io.Discard)

		switch {
		case err == nil:
			t.Errorf("%s: run(%v) was accepted", name, tc.args)
		case !strings.Contains(err.Error(), tc.mentions):
			t.Errorf("%s: error %q does not mention %q", name, err, tc.mentions)
		}
	}
}

// The three renderings, and that each writes something recognisable.
func TestRunWritesEachRendering(t *testing.T) {
	t.Parallel()

	for name, tc := range map[string]struct {
		args     []string
		mentions string
	}{
		"json":       {[]string{cmdNew, flagAuto, flagSeed, "7", flagService, other}, `"upp"`},
		"sheet":      {[]string{cmdNew, flagAuto, flagSeed, "7", flagService, other, "--sheet"}, "UPP "},
		"transcript": {[]string{cmdNew, flagAuto, flagSeed, "7", flagService, other, "--history"}, "Generation record"},
		"version":    {[]string{"version"}, "ctchargen"},
	} {
		var out strings.Builder

		err := run(tc.args, &out)
		if err != nil {
			t.Errorf("%s: %v", name, err)

			continue
		}

		if !strings.Contains(out.String(), tc.mentions) {
			t.Errorf("%s: output does not mention %q", name, tc.mentions)
		}
	}
}

// A run with no --seed draws one, so two runs differ. The seed is recorded
// either way, which is what makes a character reproducible.
func TestADrawnSeedIsRecorded(t *testing.T) {
	t.Parallel()

	first, second := &strings.Builder{}, &strings.Builder{}

	for _, out := range []*strings.Builder{first, second} {
		err := run([]string{cmdNew, flagAuto, flagService, other}, out)
		if err != nil {
			t.Fatalf("generating: %v", err)
		}

		if !strings.Contains(out.String(), `"seed"`) {
			t.Fatal("the record does not carry its seed")
		}
	}

	if first.String() == second.String() {
		t.Error("two unseeded runs produced the same character")
	}
}
