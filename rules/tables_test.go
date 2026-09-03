package rules_test

import (
	"fmt"
	"testing"

	"github.com/philoserf/ctchargen/rules"
	"github.com/philoserf/ctchargen/traveller"
)

// The rest of the second transcription: the tables that are not a grid of
// six services.

// The Aging Table (p. 9). Its three column groups are read off the
// Intelligence row, whose "no effect before age 66" ends where age 66 - the
// term 12 column - begins. Education and Social Standing carry "unaffected
// by aging" across the whole width.
func TestAgingTable(t *testing.T) {
	t.Parallel()

	r := load(t)

	for _, tc := range []struct {
		terms []traveller.Term
		want  []string
	}{
		{
			terms: []traveller.Term{4, 5, 6, 7},
			want: []string{
				"Strength -1 8+", "Dexterity -1 7+", "Endurance -1 8+",
			},
		},
		{
			terms: []traveller.Term{8, 9, 10, 11},
			want: []string{
				"Strength -1 9+", "Dexterity -1 8+", "Endurance -1 9+",
			},
		},
		{
			// The last column is terminal (E014): the Age row's 74+ is the
			// only cell in either header carrying a plus.
			terms: []traveller.Term{12, 13, 14, 15, 40},
			want: []string{
				"Strength -2 9+", "Dexterity -2 9+", "Endurance -2 9+", "Intelligence -1 9+",
			},
		},
	} {
		for _, term := range tc.terms {
			var got []string
			for _, e := range r.Aging.At(term) {
				got = append(got, fmt.Sprintf("%v -%d %v", e.Characteristic, e.Reduction, e.Saving))
			}
			if fmt.Sprint(got) != fmt.Sprint(tc.want) {
				t.Errorf("term %d: %v, want %v", term, got, tc.want)
			}
		}
	}

	// No round is run before the fourth term (p. 7).
	for term := traveller.Term(1); term < 4; term++ {
		if got := r.Aging.At(term); len(got) != 0 {
			t.Errorf("term %d: %v, want no aging", term, got)
		}
	}
	if got := r.Aging.FirstTerm(); got != 4 {
		t.Errorf("aging first applies at term %d, want 4", got)
	}

	// Education and Social Standing are unaffected at every band.
	for term := traveller.Term(4); term <= 40; term++ {
		for _, e := range r.Aging.At(term) {
			if e.Characteristic == traveller.Education || e.Characteristic == traveller.SocialStanding {
				t.Errorf("term %d reduces %v, which the table says is unaffected by aging", term, e.Characteristic)
			}
		}
	}
}

// The medical crisis (pp. 7-8): "A basic saving throw of 8+ applies ... The
// characteristic which was reduced to zero automatically becomes one. The
// character ages (one die equals the number of months in added age)."
func TestMedicalCrisis(t *testing.T) {
	t.Parallel()

	crisis := load(t).Aging.Crisis

	if got := crisis.Saving.String(); got != "8+" {
		t.Errorf("crisis saving throw is %s, want 8+", got)
	}
	if crisis.RecoversTo != 1 {
		t.Errorf("crisis recovers to %d, want 1", crisis.RecoversTo)
	}
	if crisis.MonthsDice != 1 {
		t.Errorf("crisis adds %dD months, want 1D", crisis.MonthsDice)
	}
}

// The weapon lists (pp. 12-13), read as the boxes are set: the left column,
// then the right.
func TestWeaponLists(t *testing.T) {
	t.Parallel()

	r := load(t)

	for _, tc := range []struct {
		category    traveller.WeaponCategory
		left, right []string
	}{
		{
			category: traveller.Blade,
			left:     []string{"Dagger", "Blade", "Foil", "Cutlass", "Sword", "Broadsword"},
			right:    []string{"Spear", "Halberd", "Pike", "Cudgel", "Bayonet"},
		},
		{
			category: traveller.Gun,
			left:     []string{"Body Pistol", "Automatic Pistol", "Revolver", "Carbine", "Rifle"},
			right:    []string{"Laser Carbine", "Laser Rifle", "Automatic Rifle", "Submachine Gun", "Shotgun"},
		},
	} {
		var want []traveller.WeaponName
		for _, name := range append(append([]string{}, tc.left...), tc.right...) {
			want = append(want, traveller.WeaponName(name))
		}

		got := r.Weapons(tc.category)
		if fmt.Sprint(got) != fmt.Sprint(want) {
			t.Errorf("%v:\n got %v\nwant %v", tc.category, got, want)
		}
	}
}

// The Rank and Service Skills box (p. 23). Rank 0 means the grant is by
// virtue of the service itself, which E005 reads as granted once on
// entering it.
func TestRankAndServiceSkills(t *testing.T) {
	t.Parallel()

	r := load(t)

	onEntering := map[traveller.ServiceName][]string{
		traveller.Navy:      nil,
		traveller.Marines:   {"Cutlass"},
		traveller.Army:      {"Rifle"},
		traveller.Scouts:    {"Pilot"},
		traveller.Merchants: nil,
		traveller.Other:     nil,
	}
	for service, want := range onEntering {
		var got []string
		for _, result := range r.GrantsOnEntering(service) {
			got = append(got, describe(t, result))
		}
		if fmt.Sprint(got) != fmt.Sprint(want) {
			t.Errorf("%v on entering: %v, want %v", service, got, want)
		}
	}

	byRank := []struct {
		service traveller.ServiceName
		rank    traveller.Rank
		want    []string
	}{
		{traveller.Navy, 5, []string{"+1 Social Standing"}},
		{traveller.Navy, 6, []string{"+1 Social Standing"}},
		{traveller.Marines, 1, []string{"Revolver"}},
		{traveller.Army, 1, []string{"Submachine Gun"}},
		{traveller.Merchants, 4, []string{"Pilot"}},
		{traveller.Navy, 1, nil},
		{traveller.Merchants, 5, nil},
	}
	for _, tc := range byRank {
		var got []string
		for _, result := range r.GrantsAtRank(tc.service, tc.rank) {
			got = append(got, describe(t, result))
		}
		if fmt.Sprint(got) != fmt.Sprint(tc.want) {
			t.Errorf("%v rank %d: %v, want %v", tc.service, tc.rank, got, tc.want)
		}
	}
}

// Annual Retirement Pay (p. 21), and who draws it.
func TestRetirementPay(t *testing.T) {
	t.Parallel()

	r := load(t)

	for terms, want := range map[int]traveller.Credits{
		5: 4000, 6: 6000, 7: 8000, 8: 10000,
		// "Service beyond 8 terms adds CR 2000 per additional term."
		9: 12000, 10: 14000, 12: 18000,
	} {
		if got := r.Retirement.Pay(terms); got != want {
			t.Errorf("%d terms: %v, want %v", terms, got, want)
		}
	}

	// "Retirement pay is not available to characters serving in the Scout or
	// the Other service."
	for _, name := range traveller.ServiceNames {
		want := name != traveller.Scouts && name != traveller.Other
		if got := r.Service(name).PaysPension; got != want {
			t.Errorf("%v pays a pension: %v, want %v", name, got, want)
		}
	}
}

// The Basic Skill Eligibility box (p. 6) and the Education 8+ gate (p. 11).
func TestEligibility(t *testing.T) {
	t.Parallel()

	r := load(t)

	want := rules.Eligibility{InitialTerm: 2, PerSubsequentTerm: 1, OnCommission: 1, OnPromotion: 1}
	if r.Eligibility != want {
		t.Errorf("eligibility %+v, want %+v", r.Eligibility, want)
	}

	if r.Education.Table != traveller.AdvancedEducationEight {
		t.Errorf("the education gate opens %v", r.Education.Table)
	}
	for education, open := range map[int]bool{6: false, 7: false, 8: true, 9: true, 15: true} {
		var p traveller.Profile
		p[traveller.Education] = education
		if got := r.Education.Open(p); got != open {
			t.Errorf("education %d opens the fourth table: %v, want %v", education, got, open)
		}
	}
}

// The mustering out notes (pp. 7, 9) and the passages (pp. 21-22).
func TestMusterRollsAndPassages(t *testing.T) {
	t.Parallel()

	r := load(t)

	// "One benefit roll is allowed for each term of service served ... rank
	// 1 or 2 is allowed one extra roll, rank 3 or higher is allowed two
	// extra rolls." The page's own example: an uncommissioned character who
	// has served 4 terms is eligible for 4 benefits.
	for _, tc := range []struct {
		terms int
		rank  traveller.Rank
		want  int
	}{
		{4, 0, 4}, {4, 1, 5}, {4, 2, 5}, {4, 3, 6}, {4, 6, 6}, {1, 0, 1},
		// Jamison musters out at 5 terms and rank 5, entitled to two extra
		// rolls, "in addition to the 5 rolls (for 5 terms of service)".
		{5, 5, 7},
	} {
		if got := r.Muster.Rolls(tc.terms, tc.rank); got != tc.want {
			t.Errorf("%d terms at rank %d: %d rolls, want %d", tc.terms, tc.rank, got, tc.want)
		}
	}

	if r.Muster.MaxOnTable2 != 3 {
		t.Errorf("max rolls on table 2 is %d, want 3", r.Muster.MaxOnTable2)
	}
	if r.Muster.Table1DMFromRank5or6 != 1 || r.Muster.Table2DMFromGambling != 1 {
		t.Errorf("the two optional DMs are %d and %d, want 1 and 1",
			r.Muster.Table1DMFromRank5or6, r.Muster.Table2DMFromGambling)
	}

	for class, want := range map[traveller.PassageClass]traveller.Credits{
		traveller.HighPassage: 10000, traveller.MiddlePassage: 8000, traveller.LowPassage: 1000,
	} {
		if got := r.Passage(class); got != want {
			t.Errorf("%v costs %v, want %v", class, got, want)
		}
	}
	if r.Muster.ResalePercent != 90 {
		t.Errorf("passages resell at %d%%, want 90%%", r.Muster.ResalePercent)
	}
}

// The Nobility table (Book 3 p. 22).
func TestNobility(t *testing.T) {
	t.Parallel()

	r := load(t)

	for social, want := range map[int]traveller.Title{
		11: traveller.Knight, 12: traveller.Baron, 13: traveller.Marquis,
		14: traveller.Count, 15: traveller.Duke,
	} {
		got, ok := r.TitleFor(social)
		if !ok || got != want {
			t.Errorf("social standing %d: %v (%v), want %v", social, got, ok, want)
		}
	}
	for _, social := range []int{1, 7, 10} {
		if _, ok := r.TitleFor(social); ok {
			t.Errorf("social standing %d confers a title; nobility begins at 11", social)
		}
	}
}

// P. 10's own worked example of a cumulative DM: "the enlistment throw
// required for the Navy is 8+; DM of +1 allowed for intelligence of 8 or
// greater, and DM of +2 is allowed for education of 9 or greater. Assuming a
// character has intelligence of 6 and education of 10 ... he would be
// allowed a DM of +2 (for his education)." (p. 5)
func TestModifierIsCumulative(t *testing.T) {
	t.Parallel()

	r := load(t)
	navy := r.Service(traveller.Navy).Enlistment

	profile := func(intelligence, education int) traveller.Profile {
		var p traveller.Profile
		p[traveller.Intelligence] = intelligence
		p[traveller.Education] = education

		return p
	}

	for _, tc := range []struct {
		name                    string
		intelligence, education int
		want                    int
	}{
		{"the page's own example", 6, 10, 2},
		{"neither threshold", 6, 8, 0},
		{"intelligence only", 8, 8, 1},
		{"both, cumulative", 8, 9, 3},
	} {
		if got := navy.Modifier(profile(tc.intelligence, tc.education)); got != tc.want {
			t.Errorf("%s: DM %+d, want %+d", tc.name, got, tc.want)
		}
	}

	// Jamison enlists in the Merchants: 7+, "DM of +2 allowed for his
	// intelligence of greater than 6", his strength of 6 earning nothing
	// (p. 24).
	var jamison traveller.Profile
	copy(jamison[:], []int{6, 8, 8, 12, 8, 9})
	if got := r.Service(traveller.Merchants).Enlistment.Modifier(jamison); got != 2 {
		t.Errorf("Jamison's enlistment DM is %+d, want +2", got)
	}
}

// The draft is one die, and each service prints its own number (p. 5).
func TestDraft(t *testing.T) {
	t.Parallel()

	r := load(t)

	for die, want := range map[int]traveller.ServiceName{
		1: traveller.Navy, 2: traveller.Marines, 3: traveller.Army,
		4: traveller.Scouts, 5: traveller.Merchants, 6: traveller.Other,
	} {
		got, err := r.Draft(die)
		if err != nil || got != want {
			t.Errorf("draft %d: %v (%v), want %v", die, got, err, want)
		}
	}
	for _, die := range []int{0, 7} {
		if _, err := r.Draft(die); err == nil {
			t.Errorf("draft %d named a service; the draft is one die", die)
		}
	}
}

// E012's normalizations, applied by the lift.
func TestNormalization(t *testing.T) {
	t.Parallel()

	r := load(t)

	for printed, want := range map[string]traveller.SkillName{
		"Fwd Obsv":    "Forward Observer",
		"Engnrng":     "Engineer",
		"Jack-o-T":    "Jack of all Trades",
		"Admin":       "Administration",
		"Electronics": "Electronic",
		"Rifl3":       "Rifle",
		"SMG":         "Submachine Gun",
		// A name already spelled as its description heading passes through.
		"Navigation": "Navigation",
		"Gunnery":    "Gunnery",
	} {
		if got := r.Normalize(printed); got != want {
			t.Errorf("%q normalizes to %q, want %q", printed, got, want)
		}
	}
}

// A roll off the end of a table is a defect, not a missing value.
func TestRollsOffTheEndAreRefused(t *testing.T) {
	t.Parallel()

	navy := load(t).Service(traveller.Navy)

	for _, die := range []int{0, 7} {
		if _, err := navy.Result(traveller.ServiceSkills, die); err == nil {
			t.Errorf("skills table accepted a roll of %d", die)
		}
	}
	for _, row := range []int{0, 8} {
		if _, _, err := navy.Row(row); err == nil {
			t.Errorf("mustering out accepted row %d", row)
		}
	}
}
