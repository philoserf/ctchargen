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
