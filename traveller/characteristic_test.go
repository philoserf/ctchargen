package traveller_test

import (
	"fmt"
	"testing"

	"github.com/philoserf/ctchargen/traveller"
)

func profile(values ...int) traveller.Profile {
	var p traveller.Profile
	copy(p[:], values)

	return p
}

// P. 4: values "may never exceed 15, and do not go below 1 except for
// calamitous injury or aging." Alter is every alteration that is not aging.
func TestAlterHoldsTheOrdinaryFloorAndTheCeiling(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name  string
		start int
		delta int
		want  int
	}{
		{"ordinary rise", 7, 1, 8},
		{"ordinary fall", 7, -1, 6},
		{"floors at 1, not 0", 1, -1, 1},
		{"floors at 1 from a big fall", 3, -9, 1},
		{"caps at 15", 15, 1, 15},
		{"caps at 15 from a big rise", 14, 4, 15},
		{"reaches 15 exactly", 13, 2, 15},
	} {
		got := profile(0, 0, 0, 0, 0, tc.start).Alter(traveller.SocialStanding, tc.delta)
		if got[traveller.SocialStanding] != tc.want {
			t.Errorf("%s: %d %+d = %d, want %d",
				tc.name, tc.start, tc.delta, got[traveller.SocialStanding], tc.want)
		}
	}
}

// E010: aging alone may carry a characteristic to 0, which is a medical
// crisis (pp. 7-8). A profile that floored at 1 here would make the crisis
// unreachable, and with it the one path that puts months on an age.
func TestAgeReduceFloorsAtZero(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name  string
		start int
		by    int
		want  int
	}{
		{"ordinary reduction", 7, 1, 6},
		{"reaches zero", 1, 1, 0},
		{"a 2 taken off a 1 stops at zero", 1, 2, 0},
		{"a 2 taken off a 2 reaches zero", 2, 2, 0},
		{"never goes below zero", 1, 9, 0},
	} {
		got := profile(tc.start).AgeReduce(traveller.Strength, tc.by)
		if got[traveller.Strength] != tc.want {
			t.Errorf("%s: %d less %d = %d, want %d",
				tc.name, tc.start, tc.by, got[traveller.Strength], tc.want)
		}
	}
}

// The two floors are the whole of E010, so they must differ.
func TestTheTwoFloorsDiffer(t *testing.T) {
	t.Parallel()

	start := profile(1)
	if got := start.Alter(traveller.Strength, -1)[traveller.Strength]; got != 1 {
		t.Errorf("an ordinary alteration took a 1 to %d; E010 floors it at 1", got)
	}
	if got := start.AgeReduce(traveller.Strength, 1)[traveller.Strength]; got != 0 {
		t.Errorf("an aging reduction took a 1 to %d; E010 floors it at 0", got)
	}
}

// Altering one characteristic must not disturb another, and a Profile is a
// value: altering a copy must leave the original alone.
func TestAlterTouchesOneCharacteristicAndCopies(t *testing.T) {
	t.Parallel()

	start := profile(7, 7, 7, 7, 7, 7)
	got := start.Alter(traveller.Education, 1)

	if start[traveller.Education] != 7 {
		t.Error("Alter mutated the profile it was called on")
	}
	for _, c := range traveller.Characteristics {
		want := 7
		if c == traveller.Education {
			want = 8
		}
		if got[c] != want {
			t.Errorf("%v = %d, want %d", c, got[c], want)
		}
	}
}

// P. 8: the UPP is "a string of 6 digits, in the order originally rolled",
// hexadecimal, "the digits 10 through 15 are represented by the letters A
// through F."
func TestUPP(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name string
		p    traveller.Profile
		want string
	}{
		{"totally average", profile(7, 7, 7, 7, 7, 7), "777777"},
		{"the page's own example", profile(7, 7, 7, 11, 7, 7), "777B77"},
		{"Jamison as rolled", profile(6, 8, 8, 12, 8, 9), "688C89"},
		{"the full hex range", profile(10, 11, 12, 13, 14, 15), "ABCDEF"},
		{"a characteristic at zero", profile(0, 7, 7, 7, 7, 7), "077777"},
	} {
		if got := tc.p.UPP(); got != tc.want {
			t.Errorf("%s: UPP() = %q, want %q", tc.name, got, tc.want)
		}
	}
}

func TestEnumsNameAnUnknownValue(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		got  string
		want string
	}{
		{traveller.Characteristic(99).String(), "Characteristic(99)"},
		{traveller.ServiceName(99).String(), "ServiceName(99)"},
		{traveller.SkillTable(99).String(), "SkillTable(99)"},
		{traveller.WeaponCategory(99).String(), "WeaponCategory(99)"},
		{traveller.PassageClass(99).String(), "PassageClass(99)"},
		{traveller.ShipKind(99).String(), "ShipKind(99)"},
		{traveller.Title(99).String(), "Title(99)"},
		{traveller.Intent(99).String(), "Intent(99)"},
		{traveller.MusterTable(99).String(), "MusterTable(99)"},
		{traveller.ChoicePoint(99).String(), "ChoicePoint(99)"},
		{traveller.DecidedBy(99).String(), "DecidedBy(99)"},
	} {
		if tc.got != tc.want {
			t.Errorf("String() = %q, want %q", tc.got, tc.want)
		}
	}
}

// Every value of every closed alphabet must name itself, or the event log
// prints a blank where a reader needs a word.
func TestEveryEnumValueNamesItself(t *testing.T) {
	t.Parallel()

	named := map[string][]string{}
	for _, c := range traveller.Characteristics {
		named["Characteristic"] = append(named["Characteristic"], c.String())
	}
	for _, s := range traveller.ServiceNames {
		named["ServiceName"] = append(named["ServiceName"], s.String())
	}
	for _, s := range traveller.SkillTables {
		named["SkillTable"] = append(named["SkillTable"], s.String())
	}
	for _, w := range traveller.WeaponCategories {
		named["WeaponCategory"] = append(named["WeaponCategory"], w.String())
	}
	for _, c := range traveller.ChoicePoints {
		named["ChoicePoint"] = append(named["ChoicePoint"], c.String())
	}
	for _, e := range traveller.Errata {
		named["Erratum"] = append(named["Erratum"], e.String())
	}
	for _, p := range traveller.PassageClasses {
		named["PassageClass"] = append(named["PassageClass"], p.String())
	}
	for _, s := range traveller.ShipKinds {
		named["ShipKind"] = append(named["ShipKind"], s.String())
	}
	for _, title := range traveller.Titles {
		named["Title"] = append(named["Title"], title.String())
	}
	for _, i := range traveller.Intents {
		named["Intent"] = append(named["Intent"], i.String())
	}
	for _, m := range traveller.MusterTables {
		named["MusterTable"] = append(named["MusterTable"], m.String())
	}
	for _, d := range traveller.DecidedBys {
		named["DecidedBy"] = append(named["DecidedBy"], d.String())
	}

	for kind, names := range named {
		seen := map[string]bool{}
		for _, name := range names {
			switch {
			case name == "":
				t.Errorf("%s: a value named itself with an empty string", kind)
			case seen[name]:
				t.Errorf("%s: two values both name themselves %q", kind, name)
			}
			seen[name] = true
		}
	}
}

// alphabet is a closed set of int constants that names its own values.
type alphabet interface {
	~int
	fmt.Stringer
}

// checkAlphabet holds one alphabet's iteration array to its constants, in
// both directions.
//
// Element i must be the constant whose value is i, which catches an array
// that reorders, repeats or omits. And the value one past the end must have
// no name of its own — a named constant there is a value the array stops
// short of.
func checkAlphabet[T alphabet](t *testing.T, kind string, values []T) {
	t.Helper()

	for i, v := range values {
		if int(v) != i {
			t.Errorf("%s: element %d is %v, whose value is %d; the array must hold the constants in declaration order",
				kind, i, v, int(v))
		}
	}

	past := T(len(values))
	unnamed := fmt.Sprintf("%s(%d)", kind, len(values))
	if got := past.String(); got != unnamed {
		t.Errorf("%s: the value %d names itself %q, so a constant exists past the end of an array of %d",
			kind, len(values), got, len(values))
	}
}

// A closed alphabet is two declarations — the constants and the array that
// iterates them — and Go checks neither against the other. The exhaustive
// linter guards the switch; until this test, nothing guarded the array.
// Adding a constant and
// forgetting the array compiles, passes every other test in this package,
// and silently drops the value from every loop that iterates the alphabet:
// a service never offered, a title never conferred, a table never rolled on.
//
// Erratum is absent because it numbers from 1 and has no unknown-value form;
// the gate in internal/docsgate holds it to ERRATA.md instead. Modality is
// absent because it has no array to hold: Exactly names itself with the
// empty string, which is the book's notation for a throw with no sign.
func TestEveryAlphabetHoldsEveryConstant(t *testing.T) {
	t.Parallel()

	checkAlphabet(t, "Characteristic", traveller.Characteristics[:])
	checkAlphabet(t, "ServiceName", traveller.ServiceNames[:])
	checkAlphabet(t, "SkillTable", traveller.SkillTables[:])
	checkAlphabet(t, "WeaponCategory", traveller.WeaponCategories[:])
	checkAlphabet(t, "PassageClass", traveller.PassageClasses[:])
	checkAlphabet(t, "ShipKind", traveller.ShipKinds[:])
	checkAlphabet(t, "Title", traveller.Titles[:])
	checkAlphabet(t, "Intent", traveller.Intents[:])
	checkAlphabet(t, "MusterTable", traveller.MusterTables[:])
	checkAlphabet(t, "DecidedBy", traveller.DecidedBys[:])
	checkAlphabet(t, "ChoicePoint", traveller.ChoicePoints[:])
}
