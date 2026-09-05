package traveller_test

import (
	"fmt"
	"testing"

	"github.com/philoserf/ctchargen/traveller"
)

// No two values of an alphabet share a spelling.
//
// This is the assumption chargen's FR9 gate rests on. logging.record refuses
// an answer that was not offered by comparing rendered names, not values: the
// decorator holds both the offered set and the answer, and checking there
// leaves one definition of "answered outside the offer" rather than one per
// application site. That check is sound exactly as long as String() is
// injective on every alphabet the engine puts in an offered set.
//
// It was true when the gate was written and stated nowhere (#48). Two values
// that spelled the same would let a decider answer with one and have the
// other recorded, and nothing anywhere would notice.
//
// Every alphabet is here, not only the ones a Decider offers today, because
// what makes one reachable is a method signature - and a method that starts
// offering PassageClasses should not also have to remember this file.
func TestNoTwoValuesOfAnAlphabetShareASpelling(t *testing.T) {
	t.Parallel()

	spelledOnce(t, "Characteristic", traveller.Characteristics[:])
	spelledOnce(t, "Intent", traveller.Intents[:])
	spelledOnce(t, "MusterTable", traveller.MusterTables[:])
	spelledOnce(t, "ChoicePoint", traveller.ChoicePoints[:])
	spelledOnce(t, "Erratum", traveller.Errata[:])
	spelledOnce(t, "DecidedBy", traveller.DecidedBys[:])
	spelledOnce(t, "SkillTable", traveller.SkillTables[:])
	spelledOnce(t, "WeaponCategory", traveller.WeaponCategories[:])
	spelledOnce(t, "ServiceName", traveller.ServiceNames[:])
	spelledOnce(t, "PassageClass", traveller.PassageClasses[:])
	spelledOnce(t, "ShipKind", traveller.ShipKinds[:])
	spelledOnce(t, "Title", traveller.Titles[:])
}

// spelledOnce reports two values of one alphabet that render the same.
//
// The values are named by position rather than by %v, which would call the
// String() this is checking and print the same word twice.
func spelledOnce[T fmt.Stringer](t *testing.T, alphabet string, all []T) {
	t.Helper()

	if len(all) == 0 {
		t.Errorf("%s: no values, so this checks nothing", alphabet)

		return
	}

	seen := map[string]int{}

	for i, value := range all {
		spelled := value.String()

		first, taken := seen[spelled]
		if taken {
			t.Errorf("%s: values %d and %d both spell %q; the FR9 gate compares spellings",
				alphabet, first, i, spelled)

			continue
		}

		seen[spelled] = i
	}
}
