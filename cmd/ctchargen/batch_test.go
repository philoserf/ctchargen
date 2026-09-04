package main

import (
	"encoding/json"
	"errors"
	"io"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// The seed arithmetic, which is what a batch means.

const (
	cmdBatch  = "batch"
	flagCount = "--count"
)

// P. of the PRD's Determinism section: "Members number from zero, so
// `batch --count 1 --seed N` produces exactly what `new --seed N` produces,
// and every seed a referee can type is reachable as a member."
//
// Asserted rather than assumed: it is the one property that makes a batch
// member no different from a character generated on its own.
func TestOneMemberIsTheSameCharacterAsNew(t *testing.T) {
	t.Parallel()

	const seed = "42"

	var alone, batched strings.Builder

	err := run([]string{cmdNew, flagAuto, flagSeed, seed, flagService, other}, nil, &alone, io.Discard)
	if err != nil {
		t.Fatalf("new: %v", err)
	}

	err = run([]string{cmdBatch, flagCount, "1", flagAuto, flagSeed, seed, flagService, other}, nil, &batched, io.Discard)
	if err != nil {
		t.Fatalf("batch: %v", err)
	}

	if compact(t, alone.String()) != compact(t, strings.TrimSpace(batched.String())) {
		t.Error("batch --count 1 --seed 42 is not the character new --seed 42 produces")
	}
}

// Each member carries its own derived seed, not the base, so that its record
// regenerates it.
func TestEachMemberCarriesItsOwnSeed(t *testing.T) {
	t.Parallel()

	const base = 1000

	members := generateBatch(t, strconv.Itoa(base), 5)

	for i, member := range members {
		want := uint64(base + i)
		if member.Inputs.Seed != want {
			t.Errorf("member %d carries seed %d, want %d", i, member.Inputs.Seed, want)
		}
	}

	// Regenerating a member from its own recorded seed reproduces it.
	var again strings.Builder

	err := run([]string{
		cmdNew, flagAuto, flagSeed, strconv.FormatUint(members[3].Inputs.Seed, 10), flagService, other,
	}, nil, &again, io.Discard)
	if err != nil {
		t.Fatalf("regenerating member 3: %v", err)
	}

	var reproduced batchMember

	err = json.Unmarshal([]byte(again.String()), &reproduced)
	if err != nil {
		t.Fatalf("reading the regenerated member: %v", err)
	}

	if reproduced.UPP != members[3].UPP {
		t.Errorf("member 3 regenerated as %s, want %s", reproduced.UPP, members[3].UPP)
	}
}

// "an explicit --seed above [2^53], and a derived base + i that passes it
// because the base landed within --count of the bound, are both written to
// the record as given, not silently re-bounded."
func TestAnExplicitSeedIsNeverReBounded(t *testing.T) {
	t.Parallel()

	const bound = uint64(1)<<53 - 1

	base := bound - 1
	members := generateBatch(t, strconv.FormatUint(base, 10), 3)

	for i, member := range members {
		want := base + uint64(i)
		if member.Inputs.Seed != want {
			t.Errorf("member %d carries seed %d, want %d: a derived seed past the bound is written as given",
				i, member.Inputs.Seed, want)
		}
	}

	// And the addition wraps rather than failing: a base at the very top of
	// the range is a seed like any other.
	wrapped := generateBatch(t, strconv.FormatUint(math.MaxUint64, 10), 2)
	if wrapped[1].Inputs.Seed != 0 {
		t.Errorf("the member after the largest seed is %d, want 0: the addition wraps",
			wrapped[1].Inputs.Seed)
	}
}

// batch writes what it rolled: the dead are written like anyone else.
func TestBatchRefusesWithoutAuto(t *testing.T) {
	t.Parallel()

	for name, args := range map[string][]string{
		"no --auto":  {cmdBatch, flagCount, "3"},
		"no --count": {cmdBatch, flagAuto},
		"zero count": {cmdBatch, flagAuto, flagCount, "0"},
	} {
		err := run(args, nil, io.Discard, io.Discard)
		if err == nil {
			t.Errorf("%s: batch accepted %v", name, args)
		}
	}
}

// -o naming a directory writes one file per character, named for its seed.
func TestBatchIntoADirectory(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	err := run([]string{
		cmdBatch, flagCount, "3", flagAuto, flagSeed, "7", flagService, other, flagOutput, dir,
	}, nil, io.Discard, io.Discard)
	if err != nil {
		t.Fatalf("batch into a directory: %v", err)
	}

	written, err := filepath.Glob(filepath.Join(dir, "*.json"))
	if err != nil {
		t.Fatalf("looking for the members: %v", err)
	}

	if len(written) != 3 {
		t.Fatalf("wrote %d files, want 3", len(written))
	}

	// A second run refuses to replace them, and --force replaces them.
	err = run([]string{
		cmdBatch, flagCount, "3", flagAuto, flagSeed, "7", flagService, other, flagOutput, dir,
	}, nil, io.Discard, io.Discard)
	if err == nil {
		t.Error("a second batch overwrote the first without --force")
	}

	err = run([]string{
		cmdBatch, flagCount, "3", flagAuto, flagSeed, "7", flagService, other, flagOutput, dir, flagForce,
	}, nil, io.Discard, io.Discard)
	if err != nil {
		t.Errorf("--force did not replace the members: %v", err)
	}
}

// batchMember is as much of a record as these tests read.
type batchMember struct {
	UPP    string `json:"upp"`
	Inputs struct {
		Seed uint64 `json:"seed"`
	} `json:"inputs"`
}

func generateBatch(t *testing.T, seed string, count int) []batchMember {
	t.Helper()

	var out strings.Builder

	err := run([]string{
		cmdBatch, flagCount, strconv.Itoa(count), flagAuto, flagSeed, seed, flagService, other,
	}, nil, &out, io.Discard)
	if err != nil {
		t.Fatalf("batch: %v", err)
	}

	var members []batchMember

	for line := range strings.SplitSeq(strings.TrimSpace(out.String()), "\n") {
		var m batchMember

		err := json.Unmarshal([]byte(line), &m)
		if err != nil {
			t.Fatalf("reading a member: %v", err)
		}

		members = append(members, m)
	}

	if len(members) != count {
		t.Fatalf("read %d members, want %d", len(members), count)
	}

	return members
}

// compact removes the formatting difference between an indented record and a
// JSONL line, which is the only way the two are allowed to differ.
func compact(t *testing.T, text string) string {
	t.Helper()

	var record any

	err := json.Unmarshal([]byte(text), &record)
	if err != nil {
		t.Fatalf("reading a record: %v", err)
	}

	encoded, err := json.Marshal(record)
	if err != nil {
		t.Fatalf("re-encoding a record: %v", err)
	}

	return string(encoded)
}

// `-o dir/` names a directory that need not already exist. Without the
// trailing separator being read as "a directory", the README's own line
// fails, and `-o characters` quietly writes one JSONL file where twenty
// records were asked for.
func TestBatchCreatesTheOutputDirectory(t *testing.T) {
	t.Parallel()

	dir := filepath.Join(t.TempDir(), "characters") + string(filepath.Separator)

	err := run([]string{
		cmdBatch, flagCount, "3", flagAuto, flagSeed, "7", flagService, other, flagOutput, dir,
	}, nil, io.Discard, io.Discard)
	if err != nil {
		t.Fatalf("batch into a directory that is not there yet: %v", err)
	}

	written, err := filepath.Glob(filepath.Join(dir, "*.json"))
	if err != nil {
		t.Fatalf("looking for the members: %v", err)
	}

	if len(written) != 3 {
		t.Fatalf("wrote %d files, want 3", len(written))
	}
}

// A collision anywhere in the batch is found before any of it is written, so
// a refused batch leaves the directory as it was.
func TestARefusedBatchWritesNothing(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	// The second member's file, already there.
	blocked := filepath.Join(dir, "00000000000000000008.json")

	err := os.WriteFile(blocked, []byte("{}\n"), 0o600)
	if err != nil {
		t.Fatalf("planting the collision: %v", err)
	}

	err = run([]string{
		cmdBatch, flagCount, "4", flagAuto, flagSeed, "7", flagService, other, flagOutput, dir,
	}, nil, io.Discard, io.Discard)
	if err == nil {
		t.Fatal("a batch overwrote an existing member without --force")
	}

	written, err := filepath.Glob(filepath.Join(dir, "*.json"))
	if err != nil {
		t.Fatalf("looking for the members: %v", err)
	}

	if len(written) != 1 {
		t.Errorf("a refused batch left %d files behind, want only the one that was there",
			len(written))
	}
}

// A batch cannot make its output directory under an existing file, and says
// so rather than reporting the first member's open.
func TestBatchReportsADirectoryItCannotMake(t *testing.T) {
	t.Parallel()

	blocking := filepath.Join(t.TempDir(), "not-a-directory")

	err := os.WriteFile(blocking, []byte("x"), 0o600)
	if err != nil {
		t.Fatalf("planting the file: %v", err)
	}

	err = run([]string{
		cmdBatch, flagCount, "2", flagAuto, flagSeed, "7", flagService, other,
		flagOutput, filepath.Join(blocking, "members") + string(filepath.Separator),
	}, nil, io.Discard, io.Discard)
	if err == nil {
		t.Fatal("a batch wrote into a directory it could not make")
	}

	if !strings.Contains(err.Error(), "creating") {
		t.Errorf("the failure reads %q, which does not name the directory", err)
	}
}

// refusingWriter fails every write, which is how the write path's failure is
// reached without filling a disk.
type refusingWriter struct{}

var errRefused = errors.New("refused")

func (refusingWriter) Write([]byte) (int, error) { return 0, errRefused }

// A write that fails is reported as the write, and closes what it opened
// rather than leaving the descriptor held.
func TestAFailedWriteIsReportedAndClosed(t *testing.T) {
	t.Parallel()

	closed := false

	where := destination{out: refusingWriter{}, close: func() error {
		closed = true

		return nil
	}}

	err := where.write("anything")
	if !errors.Is(err, errRefused) {
		t.Errorf("a failed write was reported as %v, want the write's own error", err)
	}

	if !closed {
		t.Error("a failed write left the destination open")
	}
}
