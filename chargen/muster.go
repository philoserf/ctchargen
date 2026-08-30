package chargen

import (
	"fmt"
	"strings"

	"github.com/philoserf/ctchargen/service"
)

// aging applies the Aging Table's rounds for every 4-year threshold the
// character's age has crossed (34, 38, ...; pp. 7, 9) — per term, not
// batched at muster out (the Jamison example batches "for simplicity" and
// says so, p. 25). Position in the dice stream is reading E005: at the
// very end of the term, after the reenlistment throw. A characteristic
// reduced to zero is a medical crisis (pp. 7-8).
func (g *generator) aging(step string, term int) error {
	for threshold := g.nextAging; threshold <= g.char.Age; threshold += 4 {
		g.nextAging = threshold + 4

		round := g.agingTable.roundFor(threshold)
		if round == nil {
			continue
		}

		g.stampErratum("E005")
		g.outcome(step, fmt.Sprintf("aging: turning %d (pp. 7, 9; E005)", threshold), 0)

		for _, throw := range round.Throws {
			died, err := g.agingThrow(step, term, throw)
			if err != nil || died {
				return err
			}
		}
	}

	return nil
}

func (g *generator) agingThrow(step string, term int, throw AgingThrow) (bool, error) {
	spec := service.ThrowSpec{Target: throw.Save}

	_, ok, seq, err := g.targetThrow(step, "aging "+throw.Characteristic+" save", spec)
	if err != nil {
		return false, err
	}

	if ok {
		return false, nil
	}

	before, after := g.char.Characteristics.applyAging(throw.Characteristic, throw.Loss)
	g.outcome(step, fmt.Sprintf("-%d %s (%d → %d), aging (p. 9)", throw.Loss, throw.Characteristic, before, after), seq)

	if after == 0 {
		return g.medicalCrisis(step, term, throw.Characteristic)
	}

	return false, nil
}

// medicalCrisis is a characteristic at zero (pp. 7-8): saving throw 8+
// with no DM (E007: the printed medical-expertise DM has no referent in
// solo generation). Survival is immediate recovery to 1 plus 1D months of
// age; the page never states the failed throw's outcome — reading E006:
// the character does not survive.
func (g *generator) medicalCrisis(step string, term int, characteristic string) (bool, error) {
	g.stampErratum("E007")
	g.outcome(step, characteristic+" reduced to zero: medical crisis (pp. 7-8)", 0)

	spec := service.ThrowSpec{Target: "8+"}

	_, ok, seq, err := g.targetThrow(step, "medical crisis save", spec)
	if err != nil {
		return false, err
	}

	if !ok {
		g.stampErratum("E006")

		cause := "medical crisis: failed the 8+ saving throw after " + characteristic + " fell to zero"
		g.char.Death = &Death{Term: term, Cause: cause}
		g.outcome(step, fmt.Sprintf("died of the medical crisis, term %d (E006)", term), seq)

		return true, nil
	}

	g.char.Characteristics.Apply(characteristic, 1) // recovery is immediate, the zero becomes one (p. 8)

	months, monthsSeq := g.plainRoll(step, "recovery months")
	g.char.AddAgeMonths(months)
	text := fmt.Sprintf("recovered: %s to 1, aged %d months (now %d years %d months; pp. 7-8)",
		characteristic, months, g.char.Age, g.char.AgeMonths)
	g.outcome(step, text, monthsSeq)

	return false, nil
}

// musterOut is FR6 (pp. 7, 9, 21-23): rolls equal to terms served plus 1
// for rank 1-2 or 2 for rank 3+, at most 3 on Table 2 (cash), the table
// designated before each roll, +1 DM available on Table 1 at rank 5-6 and
// on Table 2 with gambling skill.
func (g *generator) musterOut(svc *service.Service) error {
	step := "muster-out"

	extra := 0

	switch {
	case g.char.Rank >= 3:
		extra = 2
	case g.char.Rank >= 1:
		extra = 1
	}

	rolls := g.char.Terms + extra
	g.step(step, fmt.Sprintf("mustering out: %d rolls (%d terms + %d for rank), at most 3 on the cash table (pp. 7, 9)",
		rolls, g.char.Terms, extra))

	cashRolls := 0

	for range rolls {
		options := []string{"benefits"}
		if cashRolls < 3 {
			options = append(options, "cash")
		}

		table, err := g.choose(Choice{Step: step, Label: ChoiceMusterTable, Options: options})
		if err != nil {
			return err
		}

		if table == "cash" {
			cashRolls++

			if err := g.cashRoll(svc, step); err != nil {
				return err
			}

			continue
		}

		if err := g.benefitRoll(svc, step); err != nil {
			return err
		}
	}

	return nil
}

func (g *generator) cashRoll(svc *service.Service, step string) error {
	dm := 0

	if g.hasSkill(service.Gambling) {
		pick, err := g.choose(Choice{Step: step, Label: ChoiceCashDM, Options: []string{Yes, No}})
		if err != nil {
			return err
		}

		if pick == Yes {
			dm = 1
		}
	}

	row, seq := g.musterRoll(step, "cash table", "gambling skill", dm)
	amount := svc.Muster.Cash[row-1]
	g.char.Benefits.Cash += amount
	g.outcome(step, fmt.Sprintf("CR %d cash (Table 2 row %d; p. 9)", amount, row), seq)

	return nil
}

func (g *generator) benefitRoll(svc *service.Service, step string) error {
	dm := 0

	if g.char.Rank >= 5 {
		pick, err := g.choose(Choice{Step: step, Label: ChoiceBenefitDM, Options: []string{Yes, No}})
		if err != nil {
			return err
		}

		if pick == Yes {
			dm = 1
		}
	}

	row, seq := g.musterRoll(step, "benefits table", "rank 5-6", dm)

	return g.applyBenefit(step, svc.Muster.Benefits[row-1], seq)
}

// musterRoll is one die with an optional +1 DM already granted by a
// choice event; the DM can reach row 7 (p. 9).
func (g *generator) musterRoll(step, label, dmSource string, dm int) (int, int) {
	v := g.stream.One()
	seq := g.next()

	var dms []EventDM
	if dm != 0 {
		dms = []EventDM{{Source: dmSource, Value: dm}}
	}

	g.char.Events = append(g.char.Events, Event{
		Seq: seq, Kind: "throw", Step: step, Label: label,
		Dice: []int{v}, DMs: dms, Total: v + dm,
	})

	return v + dm, seq
}

func (g *generator) applyBenefit(step string, b service.Benefit, ref int) error {
	switch {
	case b.Passage != "":
		g.addPassage(step, b.Passage, ref)
	case b.Characteristic != "":
		before, after := g.char.Characteristics.Apply(b.Characteristic, b.Delta)
		text := fmt.Sprintf("%+d %s (%d → %d), applied immediately (p. 23)", b.Delta, b.Characteristic, before, after)
		g.outcome(step, text, ref)
	case b.Weapon != "":
		return g.weaponBenefit(step, b.Weapon, ref)
	case b.TravellersAid:
		if g.char.Benefits.TravellersAid {
			g.outcome(step, "Travellers' Aid membership may be achieved only once; the roll is wasted (p. 22)", ref)

			return nil
		}

		g.char.Benefits.TravellersAid = true
		g.outcome(step, "Travellers' Aid Society membership, for life (p. 22)", ref)
	case b.Ship != "":
		return g.shipBenefit(step, b.Ship, ref)
	default:
		g.outcome(step, "no benefit (the table's blank row)", ref)
	}

	return nil
}

func (g *generator) addPassage(step, class string, ref int) {
	switch class {
	case "high":
		g.char.Benefits.Passages.High++
		g.outcome(step, "high passage (CR 10,000, sellable at 90%; pp. 21-22)", ref)
	case "middle":
		g.char.Benefits.Passages.Middle++
		g.outcome(step, "middle passage (CR 8,000, sellable at 90%; pp. 21-22)", ref)
	default:
		g.char.Benefits.Passages.Low++
		g.outcome(step, "low passage (CR 1,000, sellable at 90%; pp. 21-22)", ref)
	}
}

// weaponBenefit chooses the specific weapon immediately (p. 9 note,
// p. 22); a repeat receipt may take the same weapon, a different one, or
// +1 expertise in a weapon already received as a benefit.
func (g *generator) weaponBenefit(step, category string, ref int) error {
	options, err := g.reg.Weapons(category)
	if err != nil {
		return fmt.Errorf("weapon benefit: %w", err)
	}

	categoryWeapons := map[string]bool{}
	for _, w := range options {
		categoryWeapons[w] = true
	}

	for _, received := range g.char.Benefits.Weapons {
		if categoryWeapons[received] {
			options = append(options, ExpertisePrefix+received)
		}
	}

	pick, err := g.choose(Choice{Step: step, Label: ChoiceMusterWeapon, Category: category, Options: options})
	if err != nil {
		return err
	}

	if weapon, isExpertise := strings.CutPrefix(pick, ExpertisePrefix); isExpertise {
		level := g.char.AddSkill(weapon, category)
		g.outcome(step, fmt.Sprintf("+1 expertise in lieu of another weapon: %s-%d (p. 22)", weapon, level), ref)

		return nil
	}

	g.char.Benefits.Weapons = append(g.char.Benefits.Weapons, pick)
	g.outcome(step, fmt.Sprintf("weapon benefit: %s (type declared immediately; p. 9, p. 22)", pick), ref)

	return nil
}

// The two ship classes, named where they are written and where a repeat
// receipt is recognised, so the two cannot drift apart.
const (
	classScout      = "Scout (Type S)"
	classFreeTrader = "Free Trader (Type A)"
)

// shipBenefit confers a ship, or recognises a repeat receipt of the one
// already held. Each branch tests the held ship's class rather than
// merely whether a ship exists: a character cannot receive both kinds,
// because validateOneShipKind forbids a service from offering both, so
// the cross-kind case is impossible state rather than a rule to apply.
func (g *generator) shipBenefit(step, kind string, ref int) error {
	held := g.char.Benefits.Ship

	if kind == "scout" {
		if held != nil && held.Class != classScout {
			return g.crossKindShip(held, kind)
		}

		if held != nil {
			g.outcome(step, "only one scout ship may be acquired; additional throws are lost (p. 23)", ref)

			return nil
		}

		g.char.Benefits.Ship = &Ship{
			Class:                  classScout,
			Book2Page:              18,
			Receipts:               1,
			ConstructivePossession: true,
		}
		g.outcome(step,
			"scout ship: Type S in constructive possession — no title, no sale, no mortgage (p. 23; Book 2 p. 18)", ref)

		return nil
	}

	if held != nil && held.Class != classFreeTrader {
		return g.crossKindShip(held, kind)
	}

	if held == nil {
		g.char.Benefits.Ship = &Ship{
			Class:                 classFreeTrader,
			Book2Page:             19,
			Receipts:              1,
			PaymentYearsRemaining: 40,
		}
		g.outcome(step, "Free Trader: Type A, owing 40 years of monthly payments (pp. 22-23; Book 2 p. 19)", ref)

		return nil
	}

	held.Receipts++
	held.AgeYears = 10 * (held.Receipts - 1)
	held.PaymentYearsRemaining = max(40-10*(held.Receipts-1), 0)
	text := fmt.Sprintf("Free Trader again: 10 years paid off, ship 10 years older — now %d years old, %d payment years remaining (pp. 22-23)", //nolint:lll // one sentence, one cite
		held.AgeYears, held.PaymentYearsRemaining)
	g.outcome(step, text, ref)

	return nil
}

func (g *generator) crossKindShip(held *Ship, kind string) error {
	return fmt.Errorf("%w: holds a %s and rolled a %q ship; validated muster data cannot offer both",
		ErrBadDecision, held.Class, kind)
}

// retirement pay: a 5th-term-or-later departure from a service that
// grants it — CR 4,000 per year at 5 terms, +2,000 per further term
// (pp. 7, 21).
func (g *generator) retirement(svc *service.Service, step string) {
	if !svc.RetirementPay || g.char.Terms < 5 {
		return
	}

	g.char.RetirementPay = 4000 + 2000*(g.char.Terms-5)
	text := fmt.Sprintf("retired after %d terms: CR %d per year retirement pay (pp. 7, 21)",
		g.char.Terms, g.char.RetirementPay)
	g.outcome(step, text, 0)
}

// title is FR7: Social Standing 11+ may assume the hereditary title
// (p. 4; Book 3 p. 22). The record stores the eligibility and the choice.
func (g *generator) title(step string) error {
	social := g.char.Characteristics.SocialStanding
	if social < 11 {
		return nil
	}

	name := g.nobility.titleFor(social)

	pick, err := g.choose(Choice{Step: step, Label: ChoiceTitle, Options: []string{Yes, No}})
	if err != nil {
		return err
	}

	g.char.Title = &Title{Title: name, Assumed: pick == Yes}

	if pick == Yes {
		text := fmt.Sprintf("assumes the hereditary title %s (Social Standing %d; p. 4, Book 3 p. 22)", name, social)
		g.outcome(step, text, 0)
	} else {
		g.outcome(step, fmt.Sprintf("eligible for the hereditary title %s but does not assume it (p. 4)", name), 0)
	}

	return nil
}

func (g *generator) hasSkill(name string) bool {
	for _, s := range g.char.Skills {
		if s.Name == name && s.Level > 0 {
			return true
		}
	}

	return false
}
