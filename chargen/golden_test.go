package chargen_test

import (
	"flag"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/philoserf/ctchargen/chargen"
	"github.com/philoserf/ctchargen/render"
	"github.com/philoserf/ctchargen/traveller"
)

// The golden fixtures. They are regenerated, never hand-edited: run
// `go test ./chargen -regenerate` and read the diff.
//
// The roster is what the PRD asks for, restricted to the paths this
// milestone's engine can walk - one per selectable strategy, plus the cases
// only a particular path produces: a civilian who declined the draft, a
// draftee, a death in service, and a medical crisis, which is the one path
// that puts months on an age.
//
//nolint:gochecknoglobals // the flag package has no other shape.
var regenerate = flag.Bool("regenerate", false, "rewrite the golden fixtures")

type fixture struct {
	name    string
	seed    uint64
	service traveller.ServiceName
	forced  bool
	policy  chargen.Policy
}

//nolint:gochecknoglobals // an immutable roster, and Go has no const slice.
var fixtures = []fixture{
	{"other-serve", 7, traveller.Other, true, chargen.DefaultPolicy()},

	// The four services that print a rank column (p. 10).
	{"navy-captain", 4, traveller.Navy, true, chargen.DefaultPolicy()},
	{"marines-serve", 4, traveller.Marines, true, chargen.DefaultPolicy()},
	{"army-serve", 4, traveller.Army, true, chargen.DefaultPolicy()},
	// Captain of the Merchants, which is the top of that column (E013), and
	// the only column whose Table 1 awards a Free Trader.
	{"merchants-captain", 316, traveller.Merchants, true, chargen.DefaultPolicy()},
	// Rejected by the Navy, drafted into the Merchants, and commissioned
	// there in a later term - p. 5 bars a draftee from a commission in his
	// first term only: "they do become eligible during the second and
	// subsequent terms of service if they reenlist."
	{"drafted-then-commissioned", 7, traveller.Navy, true, chargen.DefaultPolicy()},
	// Rank 5 taking the +1 the page allows on Table 1 (p. 9).
	{
		"merchants-table1-modifier", 52, traveller.Merchants, true,
		chargen.Policy{Career: chargen.CareerServe, Skills: chargen.SkillsAdvanced, Muster: chargen.MusterGoods},
	},
	// A characteristic reduced to zero, and the crisis survived - the one
	// path that puts months on an age (pp. 7-8).
	{"other-crisis-survived", 18, traveller.Other, true, chargen.DefaultPolicy()},
	// The same crisis, failed. E008 reads a failed saving throw as death.
	{"other-crisis-died", 106, traveller.Other, true, chargen.DefaultPolicy()},
	{"other-death", 5, traveller.Other, true, chargen.DefaultPolicy()},
	{"other-title", 4, traveller.Other, true, chargen.DefaultPolicy()},
	// Killed by a survival throw holding a Social Standing of 12. E011
	// assesses the dead and does not ask them, and no other fixture reaches
	// that branch: every other death in this roster ends below 11.
	{"died-a-noble", 39, traveller.Navy, true, chargen.DefaultPolicy()},
	// Rank 5 declining the +1 that p. 9 allows him on Table 1. Spartan is
	// the only strategy that declines it, and until this fixture no
	// generated character both held the rank and refused the modifier.
	{
		"navy-spartan-declines", 4, traveller.Navy, true,
		chargen.Policy{Career: chargen.CareerServe, Skills: chargen.SkillsAdvanced, Muster: chargen.MusterSpartan},
	},
	// Killed in the Scouts, which is where the service-wide grant of p. 23
	// is carried by a character who never lived to muster out.
	{
		"scouts-died", 1, traveller.Scouts, true,
		chargen.Policy{Career: chargen.CareerRetire, Skills: chargen.SkillsService, Muster: chargen.MusterSpartan},
	},
	// Every seed on this roster was chosen for a path, and #34's change to
	// the skills table moved five of them off it: two medical crises, the
	// months an age carries, a Free Trader, and a duplicate scout ship. The
	// seeds below are re-chosen for the same paths rather than the paths
	// being dropped - which is what golden_test's own path check is for,
	// and it is what caught them.
	// The scout ship rolled twice. P. 23: "Only one scout ship may be
	// acquired by a character, and throws resulting in additional ships are
	// lost."
	{
		"scouts-second-ship", 5039, traveller.Scouts, true,
		chargen.Policy{Career: chargen.CareerServe, Skills: chargen.SkillsAdvanced, Muster: chargen.MusterGoods},
	},
	// Rejected by the Navy, drafted into Other, and killed there.
	{
		"drafted-and-died", 2, traveller.Navy, true,
		chargen.Policy{Career: chargen.CareerRetire, Skills: chargen.SkillsService, Muster: chargen.MusterSpartan},
	},
	{
		"scouts-civilian", 6, traveller.Scouts, true,
		chargen.Policy{Career: chargen.CareerOneTerm, Skills: chargen.SkillsPersonal, Muster: chargen.MusterGoods},
	},
	// Weapon rows taken twice: goods converts the repeat into expertise,
	// spartan takes a different weapon instead (p. 22).
	{
		"scouts-expertise", 4, traveller.Scouts, true,
		chargen.Policy{Career: chargen.CareerServe, Skills: chargen.SkillsAdvanced, Muster: chargen.MusterGoods},
	},
	{
		"scouts-diversified", 4, traveller.Scouts, true,
		chargen.Policy{Career: chargen.CareerServe, Skills: chargen.SkillsAdvanced, Muster: chargen.MusterSpartan},
	},
	{
		"other-oneterm", 19, traveller.Other, false,
		chargen.Policy{Career: chargen.CareerOneTerm, Skills: chargen.SkillsPersonal, Muster: chargen.MusterGoods},
	},
	// Unforced AND drafted, which nothing else in this roster is. The policy
	// picked Other, the throw failed, and the draft put him in the Marines -
	// so the service he ended in is one nobody chose. Every other unforced
	// fixture got the service it asked for, which let the sheet credit the
	// policy with a draft's outcome and no golden notice.
	{"drafted-over-the-policys-pick", 150, traveller.Other, false, chargen.DefaultPolicy()},
	// Names a weapon, three times, from both lists. Until #34 no fixture on
	// this roster reached the Weapon choice point at all - eleven of the
	// twelve were exercised and that one was not - so the answer to it was
	// covered by nothing, before or after it changed.
	{"marines-armed", 309, traveller.Marines, true, chargen.DefaultPolicy()},
}

func (f fixture) inputs() chargen.Inputs {
	return chargen.Inputs{
		Seed: f.seed, Service: f.service, Forced: f.forced,
		Career: f.policy.Career, Skills: f.policy.Skills, Muster: f.policy.Muster,
	}
}

func (f fixture) generate(t *testing.T) *chargen.Character {
	t.Helper()

	character, err := chargen.Generate(f.inputs(), f.policy)
	if err != nil {
		t.Fatalf("%s: %v", f.name, err)
	}

	return character
}

func golden(t *testing.T, name string, got []byte) {
	t.Helper()

	path := filepath.Join("testdata", name)

	if *regenerate {
		err := os.WriteFile(path, got, 0o600)
		if err != nil {
			t.Fatalf("writing %s: %v", path, err)
		}

		return
	}

	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading %s: %v (run go test ./chargen -regenerate)", path, err)
	}

	if string(got) != string(want) {
		t.Errorf("%s has changed. If that was intended, regenerate it and read the diff:\n"+
			"  go test ./chargen -regenerate", path)
	}
}

// The two examples that sit beside docs/character.schema.json, written by
// the same path that writes the goldens so they cannot drift from what the
// engine emits.
//
// minimal is the civilian who declined the draft, which is the smallest
// complete record there is. complete is the fullest character the tables
// permit - and it is not complete, because no character can hold both
// Travellers' Aid and a starship: p. 9 prints the membership only in the
// Navy and Marines columns and the ships only in the Scout and Merchant
// columns, and a character serves one service. The schema says so.
//
//nolint:gochecknoglobals // an immutable pair, and Go has no const map.
var documented = map[string]string{
	"scouts-civilian":   "character.minimal.json",
	"merchants-captain": "character.complete.json",
}

// documentedExample writes or checks one of the two examples that sit beside
// the schema, by the same path that writes the goldens.
func documentedExample(t *testing.T, name string, encoded []byte) {
	t.Helper()

	path := filepath.Join("..", "docs", name)

	if *regenerate {
		err := os.WriteFile(path, encoded, 0o600)
		if err != nil {
			t.Fatalf("writing %s: %v", path, err)
		}

		return
	}

	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading %s: %v", path, err)
	}

	if string(encoded) != string(want) {
		t.Errorf("%s has drifted from the record it documents; regenerate it", path)
	}
}

// Every fixture's JSON, sheet and transcript, pinned.
func TestGoldens(t *testing.T) {
	t.Parallel()

	written := map[string]bool{}

	for _, fixture := range fixtures {
		character := fixture.generate(t)

		encoded, err := render.JSON(character)
		if err != nil {
			t.Fatalf("%s: %v", fixture.name, err)
		}

		golden(t, fixture.name+".json", encoded)

		if name, documents := documented[fixture.name]; documents {
			documentedExample(t, name, encoded)

			written[fixture.name] = true
		}

		sheet, err := render.Sheet(character)
		if err != nil {
			t.Fatalf("%s: %v", fixture.name, err)
		}

		transcript, err := render.Transcript(character)
		if err != nil {
			t.Fatalf("%s: %v", fixture.name, err)
		}

		golden(t, fixture.name+".sheet.md", []byte(sheet))
		golden(t, fixture.name+".transcript.md", []byte(transcript))
	}

	// Without this, renaming or dropping a fixture that documents an
	// example leaves the example behind: nothing regenerates it, nothing
	// compares it, and the schema's published record silently stops being
	// one the engine emits.
	for name := range documented {
		if !written[name] {
			t.Errorf("%s documents an example but is not in the fixture roster", name)
		}
	}
}

// The regeneration round-trip: every golden reproduced from its own recorded
// seed and inputs, byte for byte.
//
// This is what stands in for the replay subcommand the PRD rules out. It
// runs against records this repository controls rather than shipping a
// verifier for records it did not write.
func TestGoldensRegenerate(t *testing.T) {
	t.Parallel()

	for _, f := range fixtures {
		first, err := render.JSON(f.generate(t))
		if err != nil {
			t.Fatalf("%s: %v", f.name, err)
		}

		second, err := render.JSON(f.generate(t))
		if err != nil {
			t.Fatalf("%s: %v", f.name, err)
		}

		if string(first) != string(second) {
			t.Errorf("%s did not reproduce from its own seed and inputs", f.name)
		}
	}
}

// Every recorded reading is reachable by some path.
//
// This is the gate the PRD asks for, and until the engine walked the ranked
// services it could not hold. Two readings are stamped on no record by
// design: E012, a spelling, which is a transcription rather than a reading
// and changes nothing about a character; and E015, which settles the worked
// example against the table it illustrates, and which only the replay of
// that example depends on.
//
// E003 and E014 come from the scripted career rather than a fixture: past
// term 7 only a 12 grants another term, so no seed reaches the Aging Table's
// last printed column.
func TestEveryReadingIsReachable(t *testing.T) {
	t.Parallel()

	stamped := map[traveller.Erratum]bool{}

	for _, f := range fixtures {
		for _, erratum := range f.generate(t).Errata {
			stamped[erratum] = true
		}
	}

	scripted, err := chargen.Generate(
		fixtures[0].inputs(), chargen.DefaultPolicy(),
		chargen.WithRoller(&scripted{twelves: 80}),
	)
	if err != nil {
		t.Fatalf("the scripted career: %v", err)
	}

	for _, erratum := range scripted.Errata {
		stamped[erratum] = true
	}

	// The two readings that name no record. E012 is a spelling, which
	// changes nothing about a character; E015 settles the worked example
	// against a table, and only the replay of that example depends on it.
	namesNoRecord := []traveller.Erratum{traveller.E012, traveller.E015}

	for _, erratum := range traveller.Errata {
		if slices.Contains(namesNoRecord, erratum) {
			if stamped[erratum] {
				t.Errorf("%v is stamped on a record; ERRATA.md says it names none", erratum)
			}

			continue
		}

		if !stamped[erratum] {
			t.Errorf("%v is reachable by no path this repository tests", erratum)
		}
	}
}

// pathsReached names every path a single character walked, which is what
// the roster is checked against below.
func pathsReached(character *chargen.Character, reached map[string]bool) {
	mark := func(when bool, path string) {
		if when {
			reached[path] = true
		}
	}

	_, drafted := character.Enlistment.(traveller.Drafted)
	_, dead := character.Departure.(traveller.KilledBySurvivalThrow)

	_, served := character.ServedIn()

	mark(!served, "civilian")
	mark(drafted, "draftee")
	mark(dead, "death")
	mark(character.Title.Eligible, "title")
	mark(character.Age.Months() > 0, "months on an age")
	mark(len(character.Benefits.Weapons) > 0, "a weapon benefit")
	mark(len(character.Benefits.Weapons) > 1, "a second weapon benefit")
	mark(character.Rank.Commissioned(), "a commission")
	mark(drafted && character.Rank.Commissioned(), "a draftee commissioned later")
	mark(character.Rank > 1, "a promotion")
	mark(character.RankTitle != "", "a rank title")
	mark(character.Pension > 0, "retirement pay")

	for _, ship := range character.Benefits.Ships {
		mark(ship.Kind == traveller.FreeTrader, "a Free Trader")
		mark(ship.Kind == traveller.ScoutShip, "a scout ship")
		mark(ship.Kind == traveller.FreeTrader && ship.Years > 0, "a Free Trader received twice")
	}

	mark(character.DuplicateShips > 0, "a scout ship received twice")

	for _, erratum := range character.Errata {
		// A slice and not a map keyed by Erratum: a map keyed by the enum
		// claims to be a complete table of every reading, and this looks
		// for four of fifteen.
		for _, watched := range []struct {
			erratum traveller.Erratum
			path    string
		}{
			{traveller.E005, "service grant"},
			{traveller.E006, "aging"},
			{traveller.E008, "medical crisis"},
			{traveller.E013, "the top of a rank column"},
		} {
			mark(erratum == watched.erratum, watched.path)
		}
	}
}

// The roster must reach every path it claims to, or a fixture that stopped
// reaching one would read exactly like a fixture that still does.
func TestTheRosterReachesItsPaths(t *testing.T) {
	t.Parallel()

	reached := map[string]bool{}
	for _, f := range fixtures {
		pathsReached(f.generate(t), reached)
	}

	for _, path := range []string{
		"civilian", "draftee", "death", "title", "service grant",
		"medical crisis", "aging", "months on an age",
		"a weapon benefit", "a second weapon benefit",
		"a commission", "a draftee commissioned later", "a promotion",
		"a rank title", "retirement pay",
		"a Free Trader", "a Free Trader received twice",
		"a scout ship", "a scout ship received twice",
		"the top of a rank column",
	} {
		if !reached[path] {
			t.Errorf("no fixture reaches %q", path)
		}
	}
}
