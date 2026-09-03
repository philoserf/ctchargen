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
	// A characteristic reduced to zero, and the crisis survived - the one
	// path that puts months on an age (pp. 7-8).
	{"other-crisis-survived", 56, traveller.Other, true, chargen.DefaultPolicy()},
	// The same crisis, failed. E008 reads a failed saving throw as death.
	{"other-crisis-died", 17, traveller.Other, true, chargen.DefaultPolicy()},
	{"other-death", 5, traveller.Other, true, chargen.DefaultPolicy()},
	{"other-title", 4, traveller.Other, true, chargen.DefaultPolicy()},
	{
		"scouts-retire", 1, traveller.Scouts, true,
		chargen.Policy{Career: chargen.CareerRetire, Skills: chargen.SkillsService, Muster: chargen.MusterSpartan},
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

// The roster must reach every path it claims to, or a fixture that stopped
// reaching one would read exactly like a fixture that still does.
func TestTheRosterReachesItsPaths(t *testing.T) {
	t.Parallel()

	reached := map[string]bool{}

	for _, f := range fixtures {
		character := f.generate(t)

		if !character.Served {
			reached["civilian"] = true
		}

		if _, drafted := character.Enlistment.(traveller.Drafted); drafted {
			reached["draftee"] = true
		}

		if _, dead := character.Departure.(traveller.KilledBySurvivalThrow); dead {
			reached["death"] = true
		}

		if character.Title.Eligible {
			reached["title"] = true
		}

		for _, erratum := range character.Errata {
			// A slice and not a map keyed by Erratum: a map keyed by the
			// enum claims to be a complete table of every reading, and this
			// looks for three of fourteen.
			for _, watched := range []struct {
				erratum traveller.Erratum
				path    string
			}{
				{traveller.E005, "service grant"},
				{traveller.E006, "aging"},
				{traveller.E008, "medical crisis"},
			} {
				if erratum == watched.erratum {
					reached[watched.path] = true
				}
			}
		}

		if character.Age.Months() > 0 {
			reached["months on an age"] = true
		}

		// The two ways a repeat weapon row can be taken (p. 22). Both are
		// end-to-end paths, not just policy rows: without a fixture each,
		// MusterWeapon and the fold that records what it returned are
		// reached by nothing but a unit test of the policy.
		if len(character.Benefits.Weapons) > 0 {
			reached["a weapon benefit"] = true
		}

		if len(character.Benefits.Weapons) > 1 {
			reached["a second weapon benefit"] = true
		}
	}

	for _, path := range []string{
		"civilian", "draftee", "death", "title", "service grant",
		"medical crisis", "aging", "months on an age",
		"a weapon benefit", "a second weapon benefit",
	} {
		if !reached[path] {
			t.Errorf("no fixture reaches %q", path)
		}
	}
}
