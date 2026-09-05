package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
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
// The run the report opened with wrote its corpses and printed nothing at all
// (#33). The count goes to the operator's channel and never to standard
// output, where it would be a line of JSONL that is not JSON.
//
// What it should say is read off the records themselves rather than typed in.
// A literal here would be a golden maintained by hand, which the dice-stream
// order is free to invalidate at any time; counting the roster instead also
// holds this file's reading of which departures are fatal to the one the
// record carries, so the two transcriptions of that rule are checked against
// each other rather than only against themselves.
func TestABatchSaysHowManyDied(t *testing.T) {
	t.Parallel()

	const wanted = 100

	var out, asking strings.Builder

	err := run([]string{
		cmdBatch, flagCount, strconv.Itoa(wanted), flagAuto, flagSeed, "1000", flagService, scouts,
	}, nil, &out, &asking)
	if err != nil {
		t.Fatalf("generating: %v", err)
	}

	written := livingCount(t, out.String())
	if written.members != wanted || written.dead == 0 {
		t.Fatalf("%d written, %d of them dead; the run this reads is not the one it means",
			written.members, written.dead)
	}

	want := fmt.Sprintf("%d written, %d died", written.members, written.dead)
	if got := strings.TrimSpace(asking.String()); got != want {
		t.Errorf("the closing line is %q, want %q", got, want)
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

	seeds := seedsWritten(t, batched.String())
	if len(seeds) != len(lines) || len(seeds) < 2 {
		t.Fatalf("%d members written", len(seeds))
	}

	for i, line := range lines {
		var alone strings.Builder

		err = run([]string{
			cmdNew, flagAuto, flagSeed, strconv.FormatUint(seeds[i], 10), flagService, scouts,
		}, nil, &alone, io.Discard)
		if err != nil {
			t.Fatalf("regenerating seed %d: %v", seeds[i], err)
		}

		// Compared as records and not as text: batch writes JSONL and `new`
		// writes indented JSON, so the bytes differ by formatting alone.
		// The first version of this compared the strings and failed, which
		// was the test being wrong about what it meant, not the tool.
		if !sameRecord(t, line, alone.String()) {
			t.Errorf("seed %d does not regenerate the member the batch wrote", seeds[i])
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

type census struct{ members, dead int }

func livingCount(t *testing.T, jsonl string) census {
	t.Helper()

	var found census

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

// Which departures count as death, stated once and checked.
//
// --survivors rests entirely on this classification, and until now nothing
// asserted it directly: the flag was checked by counting fatal records in a
// roster, which passes just as well if a case is misread in a way the seeds
// happen not to reach. Fold makes a new case a compile error here; this makes
// a wrong answer for an existing one a test failure.
func TestOnlyTheTwoDeathsCountAsDying(t *testing.T) {
	t.Parallel()

	for name, tc := range map[string]struct {
		departure traveller.Departure
		fatal     bool
	}{
		"discharged": {traveller.Discharged{}, false},
		"forced out": {traveller.ForcedOut{}, false},
		"retired":    {traveller.Retired{}, false},
		"survival":   {traveller.KilledBySurvivalThrow{}, true},
		"medical":    {traveller.KilledByMedicalCrisis{Characteristic: traveller.Endurance}, true},
		"no service": {nil, false},
	} {
		if got := died(&chargen.Character{Departure: tc.departure}); got != tc.fatal {
			t.Errorf("%s: died is %v, want %v", name, got, tc.fatal)
		}
	}
}

// A refused batch under --survivors keys on the seeds it will actually write.
//
// The check used to be arithmetic - the seeds base through base+count-1 - and
// under --survivors that set is neither what the run writes nor a superset of
// it. Both halves of the difference matter, so both are here: a file named
// for a seed the run writes must refuse the batch, and a file named for a
// seed the run only drew and passed over must not. The old arithmetic gets
// the first right for the wrong reason - O_EXCL catches it halfway through
// the directory, which is the very thing the check exists to prevent - and
// the second wrong outright, so a one-sided test would not have caught it.
func TestARefusedSurvivorBatchKeysOnTheSeedsItWrites(t *testing.T) {
	t.Parallel()

	const (
		base   = uint64(1000)
		wanted = 5
	)

	live, passedOver := survivorRun(t, base, wanted)

	for name, tc := range map[string]struct {
		plant   uint64
		refuses bool
		left    int
	}{
		"a seed it writes":      {live[len(live)-1], true, 1},
		"a seed it passes over": {passedOver, false, len(live) + 1},
	} {
		dir := t.TempDir()
		planted := memberPath(dir, tc.plant)

		err := os.WriteFile(planted, []byte("{}\n"), 0o600)
		if err != nil {
			t.Fatalf("%s: planting the collision: %v", name, err)
		}

		var out strings.Builder

		err = run([]string{
			cmdBatch, flagCount, strconv.Itoa(wanted), flagAuto,
			flagSeed, strconv.FormatUint(base, 10), flagService, scouts,
			flagSurvivors, flagOutput, dir,
		}, nil, &out, io.Discard)

		switch {
		case tc.refuses && err == nil:
			t.Errorf("%s: the batch replaced a member that was already there", name)
		case !tc.refuses && err != nil:
			t.Errorf("%s: the batch was refused over a seed it never writes: %v", name, err)
		}

		found, err := filepath.Glob(filepath.Join(dir, "*.json"))
		if err != nil {
			t.Fatalf("%s: looking for the members: %v", name, err)
		}

		if len(found) != tc.left {
			t.Errorf("%s: the run left %d files behind, want %d", name, len(found), tc.left)
		}

		// The planted file is never the run's to replace, either way.
		kept, err := os.ReadFile(planted)
		if err != nil {
			t.Fatalf("%s: reading the planted file: %v", name, err)
		}

		if string(kept) != "{}\n" {
			t.Errorf("%s: the planted file was written over", name)
		}
	}
}

// survivorRun reports what one --survivors batch writes: the seeds it wrote,
// and one it drew and passed over.
//
// Both are read off a run rather than typed in, because which seeds live is
// the dice-stream order's to change and a literal here would be a golden kept
// by hand.
func survivorRun(t *testing.T, base uint64, count int) ([]uint64, uint64) {
	t.Helper()

	var out strings.Builder

	err := run([]string{
		cmdBatch, flagCount, strconv.Itoa(count), flagAuto,
		flagSeed, strconv.FormatUint(base, 10), flagService, scouts, flagSurvivors,
	}, nil, &out, io.Discard)
	if err != nil {
		t.Fatalf("learning which seeds live: %v", err)
	}

	live := seedsWritten(t, out.String())
	if len(live) < 2 {
		t.Fatalf("%d members written", len(live))
	}

	written := make(map[uint64]bool, len(live))
	for _, seed := range live {
		written[seed] = true
	}

	// Every seed from the base up to the last member written was drawn, so
	// those not written are the ones passed over. The scan starts at the base
	// and not at the first member, because a seed below the first member is
	// exactly the one the old arithmetic would have looked at.
	for seed := base; seed <= live[len(live)-1]; seed++ {
		if !written[seed] {
			return live, seed
		}
	}

	t.Fatal("this batch passed nobody over, so there is nothing here to check")

	return nil, 0
}

// seedsWritten reads the seed off every member of a JSONL batch.
func seedsWritten(t *testing.T, jsonl string) []uint64 {
	t.Helper()

	seeds := []uint64{}

	for line := range strings.SplitSeq(strings.TrimSpace(jsonl), "\n") {
		var member struct {
			Inputs struct {
				Seed uint64 `json:"seed"`
			} `json:"inputs"`
		}

		err := json.Unmarshal([]byte(line), &member)
		if err != nil {
			t.Fatalf("reading a member: %v", err)
		}

		seeds = append(seeds, member.Inputs.Seed)
	}

	return seeds
}
