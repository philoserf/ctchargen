package main

import (
	"encoding/json"
	"errors"
	"io"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"github.com/philoserf/ctchargen/chargen"
	"github.com/philoserf/ctchargen/traveller"
)

// scouts is the service the report used, and the one that kills most.
const (
	scouts        = "scouts"
	flagSurvivors = "--survivors"
)

// A batch says what it did, on the channel that is not the data.
//
// The run the report opened with wrote 74 corpses out of 100 and printed
// nothing at all (#33). The count goes to the operator's channel and never to
// standard output, where it would be a line of JSONL that is not JSON.
func TestABatchSaysHowManyDied(t *testing.T) {
	t.Parallel()

	var out, asking strings.Builder

	err := run([]string{
		cmdBatch, flagCount, "100", flagAuto, flagSeed, "1000", flagService, scouts,
	}, nil, &out, &asking)
	if err != nil {
		t.Fatalf("generating: %v", err)
	}

	if got := strings.TrimSpace(asking.String()); got != "100 written, 74 died" {
		t.Errorf("the closing line is %q", got)
	}

	if strings.Contains(out.String(), "written") {
		t.Error("the summary reached standard output, where it is not JSON")
	}
}

// --survivors writes the number asked for, and all of them are living.
func TestSurvivorsWritesOnlyTheLiving(t *testing.T) {
	t.Parallel()

	const wanted = 20

	var out, asking strings.Builder

	err := run([]string{
		cmdBatch, flagCount, "20", flagAuto, flagSeed, "1000", flagService, scouts,
		flagSurvivors,
	}, nil, &out, &asking)
	if err != nil {
		t.Fatalf("generating: %v", err)
	}

	written := livingCount(t, out.String())
	if written.members != wanted {
		t.Errorf("%d written, want %d", written.members, wanted)
	}

	if written.dead > 0 {
		t.Errorf("%d of the survivors are dead", written.dead)
	}

	if !strings.Contains(asking.String(), "passed over for dying") {
		t.Errorf("the closing line does not say what was passed over: %q", asking.String())
	}
}

// A survivor is still the character his own seed makes.
//
// This is the whole of what --survivors had to answer. The roster is no
// longer members 0..N-1, it is whichever of 0..M lived - so the seeds written
// are not contiguous, and every one of them has to bring its character back
// on its own. A flag that rerolled, or that renumbered, would break that.
func TestASurvivorRegeneratesFromItsOwnSeed(t *testing.T) {
	t.Parallel()

	var batched, asking strings.Builder

	err := run([]string{
		cmdBatch, flagCount, "5", flagAuto, flagSeed, "1000", flagService, scouts, flagSurvivors,
	}, nil, &batched, &asking)
	if err != nil {
		t.Fatalf("generating the batch: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(batched.String()), "\n")
	if len(lines) < 2 {
		t.Fatalf("%d members written", len(lines))
	}

	for _, line := range lines {
		var member struct {
			Inputs struct {
				Seed uint64 `json:"seed"`
			} `json:"inputs"`
		}

		err = json.Unmarshal([]byte(line), &member)
		if err != nil {
			t.Fatalf("reading a member: %v", err)
		}

		var alone strings.Builder

		err = run([]string{
			cmdNew, flagAuto, flagSeed, strconv.FormatUint(member.Inputs.Seed, 10), flagService, scouts,
		}, nil, &alone, io.Discard)
		if err != nil {
			t.Fatalf("regenerating seed %d: %v", member.Inputs.Seed, err)
		}

		// Compared as records and not as text: batch writes JSONL and `new`
		// writes indented JSON, so the bytes differ by formatting alone.
		// The first version of this compared the strings and failed, which
		// was the test being wrong about what it meant, not the tool.
		if !sameRecord(t, line, alone.String()) {
			t.Errorf("seed %d does not regenerate the member the batch wrote", member.Inputs.Seed)
		}
	}
}

// --survivors gives up rather than drawing forever.
//
// The bound is a hundred seeds per character asked for, which no measured run
// approaches. It is reachable here because eachMember takes the bound rather
// than reading the constant, so a test can ask for the living twenty times in
// twenty draws from a service that kills three in four.
func TestSurvivorsGivesUpRatherThanDrawingForever(t *testing.T) {
	t.Parallel()

	base := chargen.Inputs{
		Seed: 1000, Service: traveller.Scouts, Forced: true,
		Career: chargen.CareerServe, Skills: chargen.SkillsAdvanced, Muster: chargen.MusterCash,
	}

	_, err := eachMember(base, chargen.DefaultPolicy(), 20, true, 1,
		func(*chargen.Character) error { return nil })
	if err == nil {
		t.Fatal("a bounded search reported success without finding the living")
	}

	for _, want := range []string{"--survivors", "20 living"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("the refusal %q does not mention %q", err, want)
		}
	}
}

type roster struct{ members, dead int }

func livingCount(t *testing.T, jsonl string) roster {
	t.Helper()

	var found roster

	for line := range strings.SplitSeq(strings.TrimSpace(jsonl), "\n") {
		var member struct {
			Departure *struct {
				Fatal bool `json:"fatal"`
			} `json:"departure"`
		}

		err := json.Unmarshal([]byte(line), &member)
		if err != nil {
			t.Fatalf("reading a member: %v", err)
		}

		found.members++

		if member.Departure != nil && member.Departure.Fatal {
			found.dead++
		}
	}

	return found
}

// sameRecord reports two encodings of the same record, whatever their
// whitespace.
func sameRecord(t *testing.T, a, b string) bool {
	t.Helper()

	var first, second any

	err := json.Unmarshal([]byte(a), &first)
	if err != nil {
		t.Fatalf("reading a record: %v", err)
	}

	err = json.Unmarshal([]byte(b), &second)
	if err != nil {
		t.Fatalf("reading a record: %v", err)
	}

	return reflect.DeepEqual(first, second)
}

// A summary nobody received is not a summary that was given.
//
// The whole of #33 is that a run which said nothing was a trap. Reporting
// success when the closing line could not be written would be the same trap
// with an extra step, so the write is checked like any other.
func TestABatchReportsASummaryThatCouldNotBeWritten(t *testing.T) {
	t.Parallel()

	var out strings.Builder

	err := run([]string{
		cmdBatch, flagCount, "2", flagAuto, flagSeed, "7", flagService, other,
	}, nil, &out, brokenPipe{})
	if err == nil {
		t.Fatal("a summary nobody received was reported as written")
	}

	if !errors.Is(err, errClosedPipe) {
		t.Errorf("error %q does not carry the write failure", err)
	}
}
