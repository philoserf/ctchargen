package docsgate_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// twelve is the reenlistment throw p. 6 singles out, and E003 turns on.
const twelve = 12

// record is as much of the written record as the fifteen conditions need.
//
// It is decoded from the golden files rather than generated, because
// ERRATA.md states every condition over a record - "records where the
// enlistment throw failed" - and a record is what this reads.
type record struct {
	name string

	Terms           int            `json:"terms"`
	Rank            int            `json:"rank"`
	Service         string         `json:"service"`
	Characteristics map[string]int `json:"characteristics"`
	Errata          []string       `json:"errata"`

	Departure *struct {
		How   string `json:"how"`
		Fatal bool   `json:"fatal"`
	} `json:"departure"`

	Events []event `json:"events"`
}

// event is one line of the generation record. term is not on the wire: it is
// filled in below from the step events, so a throw can be asked which term it
// fell in - which E003 and E013 both need and neither can get otherwise.
type event struct {
	Kind      string `json:"kind"`
	Step      string `json:"step"`
	Total     int    `json:"total"`
	Succeeded bool   `json:"succeeded"`

	term int
}

// How a throw's outcome is read, where a condition cares.
type outcome int

const (
	either outcome = iota
	passed
	failed
)

func (o outcome) matches(succeeded bool) bool {
	switch o {
	case passed:
		return succeeded
	case failed:
		return !succeeded
	case either:
		return true
	}

	return false
}

// throws is every throw whose step begins with the given name.
//
// A prefix, because several steps carry what they were thrown for: the
// enlistment throw names its service, the aging saving throws name their
// characteristic.
func (r record) throws(step string) []event {
	var found []event

	for _, e := range r.Events {
		if e.Kind == "throw" && strings.HasPrefix(e.Step, step) {
			found = append(found, e)
		}
	}

	return found
}

// threw reports a throw of the named step with the given outcome.
func (r record) threw(step string, want outcome) bool {
	for _, e := range r.throws(step) {
		if want.matches(e.Succeeded) {
			return true
		}
	}

	return false
}

// firstTotal is the total of the first throw of a step, and whether there was
// one. E011 needs the Social Standing as p. 4 rolled it, before any aging
// moved it.
func (r record) firstTotal(step string) (int, bool) {
	found := r.throws(step)
	if len(found) == 0 {
		return 0, false
	}

	return found[0].Total, true
}

func (r record) departedBy(how string) bool {
	return r.Departure != nil && r.Departure.How == how
}

func (r record) died() bool { return r.Departure != nil && r.Departure.Fatal }

// readRecords decodes every golden record, and refuses to run on none: a
// gate over an empty roster passes by saying nothing.
func readRecords(t *testing.T) []record {
	t.Helper()

	paths, err := filepath.Glob(filepath.Join(goldens, "*.json"))
	if err != nil {
		t.Fatalf("looking for records: %v", err)
	}

	if len(paths) == 0 {
		t.Fatalf("no records in %s", goldens)
	}

	records := make([]record, 0, len(paths))

	for _, path := range paths {
		text, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("reading %s: %v", path, err)
		}

		var rec record

		err = json.Unmarshal(text, &rec)
		if err != nil {
			t.Fatalf("reading %s: %v", path, err)
		}

		rec.name = filepath.Base(path)
		withTerms(&rec)

		records = append(records, rec)
	}

	return records
}

// withTerms marks each event with the term it fell inside.
//
// The engine opens a term with a step event whose step is "term N", and
// everything after it belongs to that term until the next one. Events before
// the first - the characteristic rolls, the enlistment throw - are term 0,
// which is no term and is what the conditions want them to be.
func withTerms(rec *record) {
	current := 0

	for i, e := range rec.Events {
		if e.Kind == "step" && strings.HasPrefix(e.Step, termStep) {
			// Only "term N" opens a term. Any other step beginning with the
			// word is left alone rather than guessed at.
			n, err := strconv.Atoi(strings.TrimPrefix(e.Step, termStep))
			if err == nil {
				current = n
			}
		}

		rec.Events[i].term = current
	}
}
