package chargen_test

import (
	"flag"
	"os"
	"path/filepath"
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
	{"merchants-captain", 145, traveller.Merchants, true, chargen.DefaultPolicy()},
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
	{"other-crisis-survived", 56, traveller.Other, true, chargen.DefaultPolicy()},
	// The same crisis, failed. E008 reads a failed saving throw as death.
	{"other-crisis-died", 17, traveller.Other, true, chargen.DefaultPolicy()},
	{"other-death", 5, traveller.Other, true, chargen.DefaultPolicy()},
	{"other-title", 4, traveller.Other, true, chargen.DefaultPolicy()},
	// Killed in the Scouts, which is where the service-wide grant of p. 23
	// is carried by a character who never lived to muster out.
	{
		"scouts-died", 1, traveller.Scouts, true,
		chargen.Policy{Career: chargen.CareerRetire, Skills: chargen.SkillsService, Muster: chargen.MusterSpartan},
	},
	// The scout ship rolled twice. P. 23: "Only one scout ship may be
	// acquired by a character, and throws resulting in additional ships are
	// lost."
	{
		"scouts-second-ship", 55, traveller.Scouts, true,
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

// Every fixture's JSON, sheet and transcript, pinned.
func TestGoldens(t *testing.T) {
	t.Parallel()

	for _, f := range fixtures {
		character := f.generate(t)

		encoded, err := render.JSON(character)
		if err != nil {
			t.Fatalf("%s: %v", f.name, err)
		}

		golden(t, f.name+".json", encoded)
		golden(t, f.name+".sheet.md", []byte(render.Sheet(character)))
		golden(t, f.name+".transcript.md", []byte(render.Transcript(character)))
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
// services it could not hold. E012 is the one reading stamped on no record
// by design - a spelling is a transcription, not a reading, and nothing
// about a character changes with the choice.
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

	scripted, err := chargen.GenerateWith(
		fixtures[0].inputs(), &scripted{twelves: 80}, chargen.DefaultPolicy(),
	)
	if err != nil {
		t.Fatalf("the scripted career: %v", err)
	}

	for _, erratum := range scripted.Errata {
		stamped[erratum] = true
	}

	for _, erratum := range traveller.Errata {
		if erratum == traveller.E012 {
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

	mark(!character.Served, "civilian")
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
		// for four of fourteen.
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
