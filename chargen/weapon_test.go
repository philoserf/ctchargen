package chargen_test

import (
	"testing"

	"github.com/philoserf/ctchargen/traveller"
)

// --auto names more than one weapon.
//
// Across thirty Army characters it named Body Pistol twenty-two times and
// Dagger seventeen, and nothing else, though every name on both printed lists
// was offered (#34). It took from[0] every time, and no strategy reached it,
// so no flag could change that either.
//
// The threshold is deliberately low, and this counts the answers rather than
// weighing them. It is not a test of the draw's distribution - that is
// math/rand's business - but of the property the report was about: a roster
// that reads as copies.
func TestTheAutoPolicyNamesMoreThanOneWeapon(t *testing.T) {
	t.Parallel()

	named := map[string]int{}

	for _, fixture := range fixtures {
		for _, event := range fixture.generate(t).Events {
			choice, isChoice := event.(traveller.ChoiceEvent)
			if isChoice && choice.Point == traveller.ChoiceWeapon {
				named[choice.Chosen]++
			}
		}
	}

	if len(named) == 0 {
		t.Fatal("no fixture names a weapon, so this checks nothing")
	}

	if len(named) < 2 {
		t.Errorf("the roster names one weapon and no other: %v", named)
	}
}

// One character trains on more than one table.
//
// Under `advanced` the policy designated Advanced Education every time it was
// offered and Personal Development never - ninety times to none over thirty
// characters (#34) - so a character generated under the default never raised
// a characteristic, though p. 11's first table is how that is done.
//
// The count is per character and not over the roster. The first version of
// this test counted across all fixtures and passed under the old ranking too:
// three strategies between them reach all four tables however each one
// chooses, so the roster says nothing about what any character did.
func TestOneCharacterTrainsOnMoreThanOneTable(t *testing.T) {
	t.Parallel()

	widest, widestBy := 0, ""

	for _, fixture := range fixtures {
		designated := map[string]bool{}

		for _, event := range fixture.generate(t).Events {
			choice, isChoice := event.(traveller.ChoiceEvent)
			if isChoice && choice.Point == traveller.ChoiceSkillTable {
				designated[choice.Chosen] = true
			}
		}

		if len(designated) > widest {
			widest, widestBy = len(designated), fixture.name
		}
	}

	// Three of the four. A career long enough to be offered the fourth is
	// offered it only while Education stands at 8, which not every fixture
	// reaches; a career that designates three has stopped hammering one.
	const spread = 3

	if widest < spread {
		t.Errorf("no fixture designates more than %d tables (%s is the widest)", widest, widestBy)
	}
}
