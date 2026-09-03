package rules

import (
	"strings"
	"testing"
)

// A malformed table must fail loudly, and must say which cell.
//
// Every case below starts from the real data file and breaks exactly one
// thing, so what is under test is the guard rather than a hand-built fixture
// that resembles the data. The breakages are the ones the reprint's font can
// actually produce, plus the ones a careless edit can.

func mustRead[T any](t *testing.T, name string) T {
	t.Helper()

	wire, err := read[T](name)
	if err != nil {
		t.Fatalf("reading %s: %v", name, err)
	}

	return wire
}

// refuses asserts that lifting fails and that the message names what broke.
func refuses(t *testing.T, name string, err error, mentions string) {
	t.Helper()

	switch {
	case err == nil:
		t.Errorf("%s: lifted without complaint", name)
	case !strings.Contains(err.Error(), mentions):
		t.Errorf("%s: error %q does not mention %q", name, err, mentions)
	}
}

func TestMalformedServices(t *testing.T) {
	t.Parallel()

	for name, tc := range map[string]struct {
		breaks   func(*wireServices)
		mentions string
	}{
		"a missing column": {
			func(w *wireServices) { w.Services = w.Services[:5] },
			"columns",
		},
		"columns out of book order": {
			func(w *wireServices) { w.Services[0], w.Services[1] = w.Services[1], w.Services[0] },
			"want Navy",
		},
		"a row with a missing cell": {
			func(w *wireServices) { w.Survival = w.Survival[:5] },
			"survival",
		},
		"a target that will not parse": {
			func(w *wireServices) { w.Enlistment[0].Target = "eight" },
			"enlistment",
		},
		"a reenlist target that will not parse": {
			func(w *wireServices) { w.Reenlist[0] = "" },
			"reenlist",
		},
		"a commission printed for a service with no ranks": {
			func(w *wireServices) { w.Commission[3] = w.Commission[0] },
			"p. 10 makes ranks",
		},
		"a commission missing from a service with ranks": {
			func(w *wireServices) { w.Commission[0] = nil },
			"p. 10 makes ranks",
		},
	} {
		wire := mustRead[wireServices](t, "services.json")
		tc.breaks(&wire)
		refuses(t, name, (&Rules{}).liftPriorService(wire), tc.mentions)
	}
}

func TestMalformedGrantsAndRetirement(t *testing.T) {
	t.Parallel()

	base := func(t *testing.T) (*Rules, wireServices) {
		t.Helper()

		wire := mustRead[wireServices](t, "services.json")
		r := &Rules{normalize: map[string]string{}}
		if err := r.liftPriorService(wire); err != nil {
			t.Fatalf("lifting the prior service table: %v", err)
		}

		return r, wire
	}

	r, wire := base(t)
	wire.RankAndServiceSkills[0].Service = "Marine"
	refuses(t, "a grant to a service that does not exist", r.liftGrants(wire), "Marine")

	r, wire = base(t)
	wire.RankAndServiceSkills[0].Rank = 9
	refuses(t, "a grant at a rank past the table", r.liftGrants(wire), "past the service's table of ranks")

	r, wire = base(t)
	wire.RankAndServiceSkills[0].Grant = "+1 Charisma"
	refuses(t, "a grant of something that is not a result", r.liftGrants(wire), "Charisma")

	r, wire = base(t)
	wire.RetirementPay.PaidBy[0] = "Merchant"
	refuses(t, "a pension paid by a service that does not exist", r.liftRetirement(wire), "Merchant")
}

func TestMalformedSkills(t *testing.T) {
	t.Parallel()

	for name, tc := range map[string]struct {
		breaks   func(*wireSkills)
		mentions string
	}{
		"a missing table": {
			func(w *wireSkills) { delete(w.Tables, "Service Skills Table") },
			"3 tables",
		},
		"a table the type does not name": {
			func(w *wireSkills) {
				w.Tables["Basic Training"] = w.Tables["Service Skills Table"]
				delete(w.Tables, "Service Skills Table")
			},
			"Basic Training",
		},
		"a missing row": {
			func(w *wireSkills) {
				w.Tables["Service Skills Table"] = w.Tables["Service Skills Table"][:5]
			},
			"5 rows",
		},
		"a cell that alters a characteristic that does not exist": {
			func(w *wireSkills) { w.Tables["Service Skills Table"][0][0] = "+1 Charisma" },
			"Charisma",
		},
		"an education gate on a table that does not exist": {
			func(w *wireSkills) { w.EducationGate.Table = "Basic Training" },
			"education gate",
		},
		"an education gate threshold that will not parse": {
			func(w *wireSkills) { w.EducationGate.Threshold = "eight" },
			"education gate",
		},
		"an education gate on a characteristic that does not exist": {
			func(w *wireSkills) { w.EducationGate.Characteristic = "Wisdom" },
			"education gate",
		},
	} {
		wire := mustRead[wireSkills](t, "skills.json")
		tc.breaks(&wire)
		refuses(t, name, (&Rules{normalize: map[string]string{}}).liftSkills(wire), tc.mentions)
	}
}

func TestMalformedMustering(t *testing.T) {
	t.Parallel()

	for name, tc := range map[string]struct {
		breaks   func(*wireMustering)
		mentions string
	}{
		"a missing row": {
			func(w *wireMustering) { w.Table1 = w.Table1[:6] },
			"rows",
		},
		// The font trap, lifted: p. 9's dash cells extract as the digit 4.
		"a dash cell that extracted as a 4": {
			func(w *wireMustering) { w.Table1[6][3] = "4" },
			"not a benefit",
		},
		"a benefit no row prints": {
			func(w *wireMustering) { w.Table1[0][0] = "Yacht" },
			"Yacht",
		},
		"a cash row with a missing cell": {
			func(w *wireMustering) { w.Table2[0] = w.Table2[0][:5] },
			"table 2",
		},
		"a cash allowance of nothing": {
			func(w *wireMustering) { w.Table2[0][0] = 0 },
			"not a cash allowance",
		},
		"a passage with no price": {
			func(w *wireMustering) { delete(w.Passages, "Low Passage") },
			"no price for Low Passage",
		},
		"a passage price that is not a number": {
			func(w *wireMustering) { w.Passages["Low Passage"] = "cheap" },
			"is not a price",
		},
		"no resale percentage": {
			func(w *wireMustering) { delete(w.Passages, "resalePercent") },
			"resale",
		},
	} {
		wire := mustRead[wireMustering](t, "mustering.json")
		tc.breaks(&wire)
		refuses(t, name, (&Rules{normalize: map[string]string{}}).liftMustering(wire), tc.mentions)
	}
}

func TestMalformedAging(t *testing.T) {
	t.Parallel()

	valid := func(t *testing.T) wireAging { return mustRead[wireAging](t, "aging.json") }

	wire := valid(t)
	wire.Bands[0].Effects[0].Characteristic = "Wisdom"
	refuses(t, "a band reducing a characteristic that does not exist", (&Rules{}).liftAging(wire), "Wisdom")

	wire = valid(t)
	wire.Bands[0].Effects[0].Saving = "eight"
	refuses(t, "a saving throw that will not parse", (&Rules{}).liftAging(wire), "aging band")

	wire = valid(t)
	wire.Bands[0], wire.Bands[2] = wire.Bands[2], wire.Bands[0]
	refuses(t, "bands out of order", (&Rules{}).liftAging(wire), "out of order")

	wire = valid(t)
	wire.Bands = nil
	refuses(t, "no bands at all", (&Rules{}).liftAging(wire), "no bands")

	wire = valid(t)
	wire.Crisis.Saving = "eight"
	refuses(t, "a crisis saving throw that will not parse", (&Rules{}).liftAging(wire), "medical crisis")
}

func TestMalformedWeaponsAndNobility(t *testing.T) {
	t.Parallel()

	wire := mustRead[wireWeapons](t, "weapons.json")
	delete(wire.Lists, "Gun Combat")
	refuses(t, "a category with no list", (&Rules{}).liftWeapons(wire), "no list for Gun Combat")

	wire = mustRead[wireWeapons](t, "weapons.json")
	wire.Lists["Blade Combat"] = append(wire.Lists["Blade Combat"], "Dagger")
	refuses(t, "a weapon listed twice", (&Rules{}).liftWeapons(wire), "listed twice")

	nobility := mustRead[wireNobility](t, "nobility.json")
	nobility.Titles[0].Title = "archduke"
	refuses(t, "a title Book 3 does not print", (&Rules{}).liftNobility(nobility), "archduke")

	nobility = mustRead[wireNobility](t, "nobility.json")
	nobility.Titles = nobility.Titles[:4]
	refuses(t, "a nobility table with a missing rank", (&Rules{}).liftNobility(nobility), "4 rows")
}
