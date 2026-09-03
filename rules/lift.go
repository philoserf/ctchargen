package rules

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/philoserf/ctchargen/traveller"
)

// The wire types. They are JSON shapes and nothing more: every value is a
// string or a number, and every one of them is checked on the way into a
// domain value below. Nothing outside this file may hold a wire type.

type wireThrow struct {
	Target string   `json:"target"`
	DMs    []wireDM `json:"dms"`
}

type wireDM struct {
	DM int    `json:"dm"`
	If string `json:"if"`
}

type wireServices struct {
	Services   []string     `json:"services"`
	Enlistment []*wireThrow `json:"enlistment"`
	Draft      []int        `json:"draft"`
	Survival   []*wireThrow `json:"survival"`
	Commission []*wireThrow `json:"commission"`
	Promotion  []*wireThrow `json:"promotion"`
	Reenlist   []string     `json:"reenlist"`
	Ranks      [][]string   `json:"ranks"`

	RankAndServiceSkills []struct {
		Service string `json:"service"`
		Rank    int    `json:"rank"`
		Grant   string `json:"grant"`
	} `json:"rankAndServiceSkills"`

	RetirementPay struct {
		ByTerms []struct {
			Terms int   `json:"terms"`
			Pay   int64 `json:"pay"`
		} `json:"byTerms"`
		PerTermBeyondEight int64    `json:"perTermBeyondEight"`
		PaidBy             []string `json:"paidBy"`
	} `json:"retirementPay"`
}

type wireSkills struct {
	Services    []string              `json:"services"`
	Tables      map[string][][]string `json:"tables"`
	Eligibility struct {
		InitialTerm       int `json:"initialTerm"`
		PerSubsequentTerm int `json:"perSubsequentTerm"`
		OnCommission      int `json:"onCommission"`
		OnPromotion       int `json:"onPromotion"`
	} `json:"eligibility"`
	EducationGate struct {
		Table          string `json:"table"`
		Characteristic string `json:"characteristic"`
		Threshold      string `json:"threshold"`
	} `json:"educationGate"`
	Normalizations map[string]string `json:"normalizations"`
}

type wireMustering struct {
	Services []string   `json:"services"`
	Table1   [][]string `json:"table1"`
	Table2   [][]int64  `json:"table2"`
	Rolls    struct {
		PerTerm              int `json:"perTerm"`
		ExtraForRank1or2     int `json:"extraForRank1or2"`
		ExtraForRank3Plus    int `json:"extraForRank3Plus"`
		MaxOnTable2          int `json:"maxOnTable2"`
		Table1DMFromRank5or6 int `json:"table1DMFromRank5or6"`
		Table2DMFromGambling int `json:"table2DMFromGambling"`
	} `json:"rolls"`
	Names    map[string]string `json:"names"`
	Passages map[string]any    `json:"passages"`
}

type wireAging struct {
	Bands []struct {
		FromTerm int `json:"fromTerm"`
		Effects  []struct {
			Characteristic string `json:"characteristic"`
			Reduction      int    `json:"reduction"`
			Saving         string `json:"saving"`
		} `json:"effects"`
	} `json:"bands"`
	Crisis struct {
		Saving     string `json:"saving"`
		RecoversTo int    `json:"recoversTo"`
		MonthsDice int    `json:"monthsDice"`
	} `json:"crisis"`
}

type wireWeapons struct {
	Lists map[string][]string `json:"lists"`
}

type wireNobility struct {
	Titles []struct {
		SocialStanding int    `json:"socialStanding"`
		Title          string `json:"title"`
	} `json:"titles"`
}

// withoutNotes drops the keys a data file uses for its own citations and
// commentary, which begin with an underscore and are not data.
func withoutNotes[T any](m map[string]T) map[string]T {
	kept := make(map[string]T, len(m))
	for k, v := range m {
		if !strings.HasPrefix(k, "_") {
			kept[k] = v
		}
	}

	return kept
}

func read[T any](name string) (T, error) {
	var wire T

	text, err := files.ReadFile("data/" + name)
	if err != nil {
		return wire, fmt.Errorf("reading %s: %w", name, err)
	}
	if err := json.Unmarshal(text, &wire); err != nil {
		return wire, fmt.Errorf("%s: %w", name, err)
	}

	return wire, nil
}

// load lifts every table. It runs once, on first use, so a data file that
// will not lift stops the program where the defect is rather than where a
// path happens to reach it.
func load() (*Rules, error) {
	rules := &Rules{}

	skills, err := read[wireSkills]("skills.json")
	if err != nil {
		return nil, err
	}
	rules.normalize = withoutNotes(skills.Normalizations)

	mustering, err := read[wireMustering]("mustering.json")
	if err != nil {
		return nil, err
	}

	services, err := read[wireServices]("services.json")
	if err != nil {
		return nil, err
	}
	aging, err := read[wireAging]("aging.json")
	if err != nil {
		return nil, err
	}
	weapons, err := read[wireWeapons]("weapons.json")
	if err != nil {
		return nil, err
	}
	nobility, err := read[wireNobility]("nobility.json")
	if err != nil {
		return nil, err
	}

	for _, lift := range []func() error{
		func() error { return rules.liftPriorService(services) },
		func() error { return rules.liftGrants(services) },
		func() error { return rules.liftRetirement(services) },
		func() error { return rules.liftSkills(skills) },
		func() error { return rules.liftMustering(mustering) },
		func() error { return rules.liftAging(aging) },
		func() error { return rules.liftWeapons(weapons) },
		func() error { return rules.liftNobility(nobility) },
	} {
		if err := lift(); err != nil {
			return nil, err
		}
	}

	return rules, nil
}

// eachService checks that a row of a table has exactly one cell per service,
// in the order the domain lists them, and hands each cell to fn.
func eachService(header, row []string, what string, fn func(traveller.ServiceName, string) error) error {
	if len(header) != len(traveller.ServiceNames) {
		return fmt.Errorf("%s: %d column headings, want %d", what, len(header), len(traveller.ServiceNames))
	}
	for i, name := range header {
		service, err := parseService(name)
		if err != nil {
			return fmt.Errorf("%s column %d: %w", what, i+1, err)
		}
		if service != traveller.ServiceNames[i] {
			return fmt.Errorf("%s column %d is %v, want %v: the columns are in the order p. 10 prints them",
				what, i+1, service, traveller.ServiceNames[i])
		}
	}
	if len(row) != len(header) {
		return fmt.Errorf("%s: %d cells, want %d", what, len(row), len(header))
	}
	for i, cell := range row {
		if err := fn(traveller.ServiceNames[i], cell); err != nil {
			return fmt.Errorf("%s, %v: %w", what, traveller.ServiceNames[i], err)
		}
	}

	return nil
}

func (r *Rules) liftPriorService(wire wireServices) error {
	for i, name := range wire.Services {
		service, err := parseService(name)
		if err != nil {
			return fmt.Errorf("prior service table column %d: %w", i+1, err)
		}
		if i >= len(traveller.ServiceNames) || service != traveller.ServiceNames[i] {
			return fmt.Errorf("prior service table column %d is %v, want %v",
				i+1, service, traveller.ServiceNames[i])
		}
	}
	if len(wire.Services) != len(traveller.ServiceNames) {
		return fmt.Errorf("prior service table: %d columns, want %d",
			len(wire.Services), len(traveller.ServiceNames))
	}

	rows := map[string]int{
		"enlistment": len(wire.Enlistment), "draft": len(wire.Draft),
		"survival": len(wire.Survival), "commission": len(wire.Commission),
		"promotion": len(wire.Promotion), "reenlist": len(wire.Reenlist),
		"ranks": len(wire.Ranks),
	}
	for row, n := range rows {
		if n != len(traveller.ServiceNames) {
			return fmt.Errorf("prior service table, %s: %d cells, want %d", row, n, len(traveller.ServiceNames))
		}
	}

	for i, name := range traveller.ServiceNames {
		service := Service{Name: name, Draft: wire.Draft[i], Ranks: wire.Ranks[i]}

		var err error
		if service.Enlistment, err = liftThrow(wire.Enlistment[i], "enlistment"); err != nil {
			return fmt.Errorf("%v: %w", name, err)
		}
		if service.Survival, err = liftThrow(wire.Survival[i], "survival"); err != nil {
			return fmt.Errorf("%v: %w", name, err)
		}
		if service.Reenlist, err = traveller.ParseTarget(wire.Reenlist[i]); err != nil {
			return fmt.Errorf("%v reenlist: %w", name, err)
		}

		hasRanks := len(service.Ranks) > 0
		for label, cell := range map[string]*wireThrow{
			"commission": wire.Commission[i], "promotion": wire.Promotion[i],
		} {
			if (cell != nil) != hasRanks {
				return fmt.Errorf("%v prints %s but %s ranks: p. 10 makes ranks, commissions and promotions one fact",
					name, label, map[bool]string{true: "also", false: "no"}[hasRanks])
			}
		}
		if hasRanks {
			if service.commission, err = liftThrow(wire.Commission[i], "commission"); err != nil {
				return fmt.Errorf("%v: %w", name, err)
			}
			if service.promotion, err = liftThrow(wire.Promotion[i], "promotion"); err != nil {
				return fmt.Errorf("%v: %w", name, err)
			}
		}

		r.services[i] = service
	}

	return nil
}

func liftThrow(wire *wireThrow, what string) (Throw, error) {
	if wire == nil {
		return Throw{}, fmt.Errorf("%s: no throw printed", what)
	}

	target, err := traveller.ParseTarget(wire.Target)
	if err != nil {
		return Throw{}, fmt.Errorf("%s: %w", what, err)
	}

	throw := Throw{Target: target}
	for _, dm := range wire.DMs {
		lifted, err := parseDM(dm.DM, dm.If)
		if err != nil {
			return Throw{}, fmt.Errorf("%s: %w", what, err)
		}
		throw.DMs = append(throw.DMs, lifted)
	}

	return throw, nil
}

func (r *Rules) liftGrants(wire wireServices) error {
	for _, g := range wire.RankAndServiceSkills {
		service, err := parseService(g.Service)
		if err != nil {
			return fmt.Errorf("rank and service skills: %w", err)
		}
		result, err := parseTableResult(g.Grant, r.normalize)
		if err != nil {
			return fmt.Errorf("rank and service skills, %v: %w", service, err)
		}
		if g.Rank < 0 || traveller.Rank(g.Rank) > r.services[service].MaxRank() {
			return fmt.Errorf("rank and service skills, %v: rank %d is past the service's table of ranks",
				service, g.Rank)
		}
		r.grants = append(r.grants, Grant{Service: service, Rank: traveller.Rank(g.Rank), Result: result})
	}

	return nil
}

func (r *Rules) liftRetirement(wire wireServices) error {
	r.Retirement.ByTerms = make(map[int]traveller.Credits, len(wire.RetirementPay.ByTerms))
	for _, row := range wire.RetirementPay.ByTerms {
		r.Retirement.ByTerms[row.Terms] = traveller.Credits(row.Pay)
	}
	r.Retirement.PerTermBeyondEight = traveller.Credits(wire.RetirementPay.PerTermBeyondEight)

	for _, name := range wire.RetirementPay.PaidBy {
		service, err := parseService(name)
		if err != nil {
			return fmt.Errorf("retirement pay: %w", err)
		}
		r.services[service].PaysPension = true
	}

	return nil
}

func (r *Rules) liftSkills(wire wireSkills) error {
	if len(wire.Tables) != len(traveller.SkillTables) {
		return fmt.Errorf("acquired skills: %d tables, want %d", len(wire.Tables), len(traveller.SkillTables))
	}

	for name, rows := range wire.Tables {
		table, err := parseSkillTable(name)
		if err != nil {
			return fmt.Errorf("acquired skills: %w", err)
		}
		if len(rows) != Faces {
			return fmt.Errorf("acquired skills, %v: %d rows, want %d", table, len(rows), Faces)
		}
		for die, row := range rows {
			what := fmt.Sprintf("acquired skills, %v, row %d", table, die+1)
			err := eachService(wire.Services, row, what, func(s traveller.ServiceName, cell string) error {
				result, err := parseTableResult(cell, r.normalize)
				if err != nil {
					return err
				}
				r.services[s].Skills[table][die] = result

				return nil
			})
			if err != nil {
				return err
			}
		}
	}

	r.Eligibility = Eligibility{
		InitialTerm:       wire.Eligibility.InitialTerm,
		PerSubsequentTerm: wire.Eligibility.PerSubsequentTerm,
		OnCommission:      wire.Eligibility.OnCommission,
		OnPromotion:       wire.Eligibility.OnPromotion,
	}

	table, err := parseSkillTable(wire.EducationGate.Table)
	if err != nil {
		return fmt.Errorf("education gate: %w", err)
	}
	characteristic, err := parseCharacteristic(wire.EducationGate.Characteristic)
	if err != nil {
		return fmt.Errorf("education gate: %w", err)
	}
	threshold, err := traveller.ParseTarget(wire.EducationGate.Threshold)
	if err != nil {
		return fmt.Errorf("education gate: %w", err)
	}
	r.Education = Gate{Table: table, Characteristic: characteristic, Threshold: threshold}

	return nil
}

func (r *Rules) liftMustering(wire wireMustering) error {
	names := withoutNotes(wire.Names)

	if len(wire.Table1) != MusterRows || len(wire.Table2) != MusterRows {
		return fmt.Errorf("mustering out: %d and %d rows, want %d each",
			len(wire.Table1), len(wire.Table2), MusterRows)
	}

	for n, row := range wire.Table1 {
		what := fmt.Sprintf("mustering out table 1, row %d", n+1)
		err := eachService(wire.Services, row, what, func(s traveller.ServiceName, cell string) error {
			benefit, err := parseBenefitRow(cell, names)
			if err != nil {
				return err
			}
			r.services[s].Benefits[n] = benefit

			return nil
		})
		if err != nil {
			return err
		}
	}

	for n, row := range wire.Table2 {
		if len(row) != len(traveller.ServiceNames) {
			return fmt.Errorf("mustering out table 2, row %d: %d cells, want %d",
				n+1, len(row), len(traveller.ServiceNames))
		}
		for i, cash := range row {
			if cash <= 0 {
				return fmt.Errorf("mustering out table 2, row %d, %v: %d is not a cash allowance",
					n+1, traveller.ServiceNames[i], cash)
			}
			r.services[i].Cash[n] = traveller.Credits(cash)
		}
	}

	r.Muster = Muster{
		PerTerm:              wire.Rolls.PerTerm,
		ExtraForRank1or2:     wire.Rolls.ExtraForRank1or2,
		ExtraForRank3Plus:    wire.Rolls.ExtraForRank3Plus,
		MaxOnTable2:          wire.Rolls.MaxOnTable2,
		Table1DMFromRank5or6: wire.Rolls.Table1DMFromRank5or6,
		Table2DMFromGambling: wire.Rolls.Table2DMFromGambling,
	}

	for _, class := range traveller.PassageClasses {
		price, ok := wire.Passages[class.String()]
		if !ok {
			return fmt.Errorf("passages: no price for %v", class)
		}
		amount, ok := price.(float64)
		if !ok {
			return fmt.Errorf("passages, %v: %v is not a price", class, price)
		}
		r.passages[class] = traveller.Credits(amount)
	}

	resale, ok := wire.Passages["resalePercent"].(float64)
	if !ok {
		return fmt.Errorf("passages: no resale percentage")
	}
	r.Muster.ResalePercent = int(resale)

	return nil
}

func (r *Rules) liftAging(wire wireAging) error {
	for _, band := range wire.Bands {
		lifted := agingBand{fromTerm: traveller.Term(band.FromTerm)}
		for _, e := range band.Effects {
			characteristic, err := parseCharacteristic(e.Characteristic)
			if err != nil {
				return fmt.Errorf("aging band from term %d: %w", band.FromTerm, err)
			}
			saving, err := traveller.ParseTarget(e.Saving)
			if err != nil {
				return fmt.Errorf("aging band from term %d: %w", band.FromTerm, err)
			}
			lifted.effects = append(lifted.effects, AgingEffect{
				Characteristic: characteristic, Reduction: e.Reduction, Saving: saving,
			})
		}
		if len(r.Aging.bands) > 0 && lifted.fromTerm <= r.Aging.bands[len(r.Aging.bands)-1].fromTerm {
			return fmt.Errorf("aging bands are out of order at term %d", band.FromTerm)
		}
		r.Aging.bands = append(r.Aging.bands, lifted)
	}
	if len(r.Aging.bands) == 0 {
		return fmt.Errorf("aging: no bands")
	}

	saving, err := traveller.ParseTarget(wire.Crisis.Saving)
	if err != nil {
		return fmt.Errorf("medical crisis: %w", err)
	}
	r.Aging.Crisis = Crisis{
		Saving: saving, RecoversTo: wire.Crisis.RecoversTo, MonthsDice: wire.Crisis.MonthsDice,
	}

	return nil
}

func (r *Rules) liftWeapons(wire wireWeapons) error {
	for _, category := range traveller.WeaponCategories {
		printed, ok := wire.Lists[category.String()]
		if !ok || len(printed) == 0 {
			return fmt.Errorf("weapons: no list for %v", category)
		}
		seen := make(map[string]bool, len(printed))
		for _, name := range printed {
			if seen[name] {
				return fmt.Errorf("weapons, %v: %q is listed twice", category, name)
			}
			seen[name] = true
			r.weapons[category] = append(r.weapons[category], traveller.WeaponName(name))
		}
	}

	return nil
}

func (r *Rules) liftNobility(wire wireNobility) error {
	for _, row := range wire.Titles {
		title, err := parseTitle(row.Title)
		if err != nil {
			return fmt.Errorf("nobility: %w", err)
		}
		r.nobility = append(r.nobility, Nobility{SocialStanding: row.SocialStanding, Title: title})
	}
	if len(r.nobility) != len(traveller.Titles) {
		return fmt.Errorf("nobility: %d rows, want %d", len(r.nobility), len(traveller.Titles))
	}

	return nil
}
