package chargen_test

import (
	"testing"

	"github.com/philoserf/ctchargen/chargen"
	"github.com/philoserf/ctchargen/traveller"
)

// scripted is a roller that answers every two-dice throw with a 12 for as
// long as it is told to, and with a 2 thereafter. One die is always a 1.
//
// It exists because two readings cannot be reached by any seed. Past term 7
// only a 12 on the reenlistment throw grants another term (pp. 6-7), so a
// career long enough to leave the Aging Table's last printed column needs
// eight consecutive 12s and then more - which is what E003's recurrence and
// E014's terminal band both turn on. This is the scripted die the PRD asks
// for so that a particular path can be walked deliberately.
type scripted struct {
	twelves int
	drawn   int
}

func (s *scripted) Die() int {
	s.drawn++

	return 1
}

func (s *scripted) TwoDice() (int, int) {
	s.drawn += 2

	if s.twelves > 0 {
		s.twelves--

		return 6, 6
	}

	return 1, 1
}

// E003 reads p. 7's "an additional term" as recurring, and E014 reads the
// Aging Table's last column as terminal. Both are unreachable by a seed, and
// both are load-bearing: without the first the career stops at term 8, and
// without the second there is no row to read for term 15.
func TestACareerPastTheTablesLastColumn(t *testing.T) {
	t.Parallel()

	const enoughTwelves = 80

	inputs := chargen.Inputs{
		Seed: 0, Service: traveller.Other, Forced: true,
		Career: chargen.CareerServe, Skills: chargen.SkillsAdvanced, Muster: chargen.MusterCash,
	}

	character, err := chargen.GenerateWith(inputs, &scripted{twelves: enoughTwelves}, chargen.DefaultPolicy())
	if err != nil {
		t.Fatalf("generating: %v", err)
	}

	const pastTheTable = 15

	if character.Terms < pastTheTable {
		t.Fatalf("served %d terms; the Aging Table's last printed term is 14, so this reaches nothing new",
			character.Terms)
	}

	stamped := map[traveller.Erratum]bool{}
	for _, erratum := range character.Errata {
		stamped[erratum] = true
	}

	for _, want := range []traveller.Erratum{traveller.E003, traveller.E014} {
		if !stamped[want] {
			t.Errorf("a career of %d terms did not stamp %v", character.Terms, want)
		}
	}

	// Age is 18 plus four years a term, whatever ends the service (p. 5,
	// p. 9's note, E004).
	const startingAge = 18

	if want := startingAge + traveller.Years*character.Terms; character.Age.Years() != want {
		t.Errorf("age %v after %d terms, want %d", character.Age, character.Terms, want)
	}
}

// A voluntary career stops at seven terms. P. 7: "A character may serve up
// to 7 terms voluntarily, and retire any time after the end of the 5th
// term", and "Service beyond the seventh term is normally impossible".
func TestVoluntaryServiceStopsAtSeven(t *testing.T) {
	t.Parallel()

	inputs := chargen.Inputs{
		Seed: 0, Service: traveller.Other, Forced: true,
		Career: chargen.CareerServe, Skills: chargen.SkillsAdvanced, Muster: chargen.MusterCash,
	}

	// Twelves for enlistment and the first term's survival, then throws that
	// make every reenlistment without ever rolling the 12 that forces one.
	character, err := chargen.GenerateWith(inputs, &alwaysNine{}, chargen.DefaultPolicy())
	if err != nil {
		t.Fatalf("generating: %v", err)
	}

	const voluntaryCap = 7

	if character.Terms != voluntaryCap {
		t.Errorf("served %d terms voluntarily, want %d", character.Terms, voluntaryCap)
	}

	if _, retired := character.Departure.(traveller.Retired); !retired {
		t.Errorf("left after seven terms as %T; p. 21 makes that retirement", character.Departure)
	}
}

// alwaysNine makes every throw a 9: enough for every target Other prints,
// and never the 12 that forces another term.
type alwaysNine struct{}

func (alwaysNine) Die() int            { return 1 }
func (alwaysNine) TwoDice() (int, int) { return 4, 5 }
