package main

import (
	"io"
	"strings"
	"testing"

	"github.com/philoserf/ctchargen/chargen"
)

// The command line's own words, so a typo in one place is a compile error
// rather than a test that quietly stops testing the flag it names.
const (
	cmdNew      = "new"
	cmdRender   = "render"
	cmdVersion  = "version"
	wantUsage   = "usage"
	wantNoRow   = "no such strategy"
	wantUnknown = "unknown command"
	flagSheet   = "--sheet"
	flagAuto    = "--auto"
	flagSeed    = "--seed"
	flagService = "--service"
	flagName    = "--name"
	flagCareer  = "--career"
	flagSkills  = "--skills"
	flagMuster  = "--muster"
	flagHistory = "--history"
	flagForce   = "--force"
	flagOutput  = "-o"
	other       = "other"
	navy        = "navy"
	merchants   = "merchants"
)

func TestRunRejectsBadCommandLines(t *testing.T) {
	t.Parallel()

	for name, tc := range map[string]struct {
		args     []string
		mentions string
	}{
		"no command":      {nil, "new|batch|render|version"},
		"unknown command": {[]string{"generate"}, wantUnknown},
		// Interactive mode with nothing to read from asks its first
		// question and reports that the input ended, rather than answering
		// it on the player's behalf.
		"no input to read": {[]string{cmdNew}, "the input ended"},
		"unknown flag":     {[]string{cmdNew, flagAuto, "--wat"}, wantUsage},
		"no such service":  {[]string{cmdNew, flagAuto, flagService, "navvy"}, "no service is called"},
		"no such strategy": {[]string{cmdNew, flagAuto, flagCareer, "dawdle"}, wantNoRow},
		// The strategies are recorded whichever mode runs, and the schema
		// restricts them to POLICY.md's rows, so an unknown one is refused
		// without --auto too rather than being written into a record.
		"no such strategy, asking": {[]string{cmdNew, flagCareer, "dawdle"}, wantNoRow},
	} {
		err := run(tc.args, nil, io.Discard, io.Discard)

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
		"json": {[]string{cmdNew, flagAuto, flagSeed, "7", flagService, other}, `"seed": 7`},
		// The command is what fills the build, so a record it writes says
		// which tool wrote it where a test-generated one says nothing.
		"json build": {[]string{cmdNew, flagAuto, flagSeed, "7", flagService, other}, `"build"`},
		"json ruleset": {
			[]string{cmdNew, flagAuto, flagSeed, "7", flagService, other},
			`"ruleset": "Classic Traveller, Books 1-3`,
		},
		"json upp":   {[]string{cmdNew, flagAuto, flagSeed, "7", flagService, other}, `"upp"`},
		"sheet":      {[]string{cmdNew, flagAuto, flagSeed, "7", flagService, other, flagSheet}, "UPP "},
		"transcript": {[]string{cmdNew, flagAuto, flagSeed, "7", flagService, other, flagHistory}, "Generation record"},
		"version":    {[]string{cmdVersion}, "ctchargen"},
	} {
		var out strings.Builder

		err := run(tc.args, nil, &out, io.Discard)
		if err != nil {
			t.Errorf("%s: %v", name, err)

			continue
		}

		if !strings.Contains(out.String(), tc.mentions) {
			t.Errorf("%s: output does not mention %q", name, tc.mentions)
		}
	}
}

// A run with no --seed draws one, and draws a different one each time, which
// is what makes an unseeded character reproducible afterwards. That the
// record then carries the seed is held by TestRunWritesEachRendering, from a
// seed the test names.
//
// It is asserted of inputsFrom rather than of a whole run on purpose. A
// character rolled from a drawn seed walks whichever mustering out rows it
// lands on, so an unseeded generation reaches a different set of statements
// every time - and the coverage profile is taken with -coverpkg=./..., under
// which those statements are counted against the chargen package by this
// binary too. The ratchet fails on a count that falls as well as one that
// rises, so an unseeded generation here makes the gate fail at random.
func TestADrawnSeedIsRecorded(t *testing.T) {
	t.Parallel()

	drawn := func() uint64 {
		t.Helper()

		in, err := inputsFrom(0, "", other,
			chargen.CareerServe, chargen.SkillsAdvanced, chargen.MusterCash, false)
		if err != nil {
			t.Fatalf("drawing a seed: %v", err)
		}

		if in.Seed >= maxDrawnSeed {
			t.Errorf("drew %d, which is above the bound a JSON reader parses exactly", in.Seed)
		}

		return in.Seed
	}

	if first, second := drawn(), drawn(); first == second {
		t.Errorf("two unseeded runs both drew %d", first)
	}
}

// A --seed given is the operator's own number, recorded as given rather than
// redrawn.
func TestAGivenSeedIsKept(t *testing.T) {
	t.Parallel()

	const given = 7

	in, err := inputsFrom(given, "", other,
		chargen.CareerServe, chargen.SkillsAdvanced, chargen.MusterCash, true)
	if err != nil {
		t.Fatalf("taking the seed: %v", err)
	}

	if in.Seed != given {
		t.Errorf("seed %d, want the %d that was given", in.Seed, given)
	}
}
