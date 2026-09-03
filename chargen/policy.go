package chargen

import (
	"fmt"
	"slices"

	"github.com/philoserf/ctchargen/traveller"
)

// Policy is the auto decider of docs/POLICY.md. It is total - it answers
// every method of Decider - and deterministic, tie-breaking by first-listed
// order in Book 1.
//
// Every strategy is a pure function of what it is handed. Where a strategy
// needs to know that an option is legal, the engine supplies that by what it
// puts in the offered set, never by the strategy reaching into the
// character.
type Policy struct {
	Career string
	Skills string
	Muster string
}

// The named strategies of docs/POLICY.md.
const (
	CareerServe   = "serve"
	CareerRetire  = "retire"
	CareerOneTerm = "oneterm"

	SkillsAdvanced = "advanced"
	SkillsService  = "service"
	SkillsPersonal = "personal"

	MusterCash    = "cash"
	MusterGoods   = "goods"
	MusterSpartan = "spartan"
)

// Strategies is every selectable strategy, by flag, in the order POLICY.md
// lists them - the first of each is the default.
//
//nolint:gochecknoglobals // an immutable table, and Go has no const map.
var Strategies = map[string][]string{
	"career": {CareerServe, CareerRetire, CareerOneTerm},
	"skills": {SkillsAdvanced, SkillsService, SkillsPersonal},
	"muster": {MusterCash, MusterGoods, MusterSpartan},
}

// DefaultPolicy is what a bare --auto run applies. A record names its
// strategies either way, so no record is silent about the policy that made
// it.
func DefaultPolicy() Policy {
	return Policy{Career: CareerServe, Skills: SkillsAdvanced, Muster: MusterCash}
}

// Validate reports a strategy name that no row of POLICY.md carries.
func (p Policy) Validate() error {
	for flag, chosen := range map[string]string{
		"career": p.Career, "skills": p.Skills, "muster": p.Muster,
	} {
		if !slices.Contains(Strategies[flag], chosen) {
			return fmt.Errorf("%w: --%s %q; POLICY.md carries %v",
				errNoSuchStrategy, flag, chosen, Strategies[flag])
		}
	}

	return nil
}

// prefer picks the first of a ranked preference that is on offer, and falls
// back to the first thing offered - which is book order, because that is the
// order the engine builds an offered set in.
func prefer[T comparable](offered, ranked []T) T {
	for _, want := range ranked {
		if slices.Contains(offered, want) {
			return want
		}
	}

	return offered[0]
}

// Service takes the offer whose throw is likeliest to succeed, ties going to
// the first listed - which is the Prior Service Table's column order (p. 10).
func (p Policy) Service(from []traveller.EnlistmentOffer) (traveller.ServiceName, error) {
	best := from[0]
	bestOdds := odds(best)

	for _, offer := range from[1:] {
		if o := odds(offer); o > bestOdds {
			best, bestOdds = offer, o
		}
	}

	return best.Service, nil
}

// odds is the chance of meeting an offer's target on two dice after its
// modifier. It is computed from the printed target and the printed DM; it is
// not a table of its own and it is not a rule.
func odds(offer traveller.EnlistmentOffer) float64 {
	ways := 0

	const faces = 6

	for first := 1; first <= faces; first++ {
		for second := 1; second <= faces; second++ {
			if offer.Target.Satisfied(first + second + offer.DM) {
				ways++
			}
		}
	}

	return float64(ways) / float64(faces*faces)
}

// SubmitToDraft is answered no only by oneterm, which is what reaches the civilian
// record at all (E001).
func (p Policy) SubmitToDraft() (bool, error) { return p.Career != CareerOneTerm, nil }

// AttemptCommission is always yes: nothing in the rules costs a character a failed
// attempt, and a commission carries a skill eligibility (p. 6).
func (p Policy) AttemptCommission() (bool, error) { return true, nil }

// AttemptPromotion is always yes, for the same reason (p. 6).
func (p Policy) AttemptPromotion() (bool, error) { return true, nil }

// AssumeTitle is always yes: every strategy assumes the title.
func (p Policy) AssumeTitle(traveller.Title) (bool, error) { return true, nil }

// ReenlistIntent ranks the three by career strategy.
func (p Policy) ReenlistIntent(from []traveller.Intent) (traveller.Intent, error) {
	ranked := map[string][]traveller.Intent{
		CareerServe:   {traveller.Continue, traveller.Retire, traveller.Discharge},
		CareerRetire:  {traveller.Retire, traveller.Continue, traveller.Discharge},
		CareerOneTerm: {traveller.Discharge, traveller.Retire, traveller.Continue},
	}[p.Career]

	return prefer(from, ranked), nil
}

// SkillTable ranks the four by skills strategy. advanced is the default
// because it is the one ranking that makes the Education 8+ gate visible in
// a default run: it takes the fourth table the instant it opens.
func (p Policy) SkillTable(from []traveller.SkillTable) (traveller.SkillTable, error) {
	ranked := map[string][]traveller.SkillTable{
		SkillsAdvanced: {
			traveller.AdvancedEducationEight, traveller.AdvancedEducation,
			traveller.ServiceSkills, traveller.PersonalDevelopment,
		},
		SkillsService: {
			traveller.ServiceSkills, traveller.AdvancedEducationEight,
			traveller.AdvancedEducation, traveller.PersonalDevelopment,
		},
		SkillsPersonal: {
			traveller.PersonalDevelopment, traveller.ServiceSkills,
			traveller.AdvancedEducationEight, traveller.AdvancedEducation,
		},
	}[p.Skills]

	return prefer(from, ranked), nil
}

// Weapon takes the first name on the category's printed list - Dagger for
// blades, Body Pistol for guns. Repeats take it again, which raises its
// level (p. 12) and is the branch that produces a level above 1.
func (p Policy) Weapon(
	_ traveller.WeaponCategory, from []traveller.WeaponName,
) (traveller.WeaponName, error) {
	return from[0], nil
}

// MusterTable is ranked by muster strategy. cash reaches the three-roll cap, which is the only way the
// cap is exercised; goods is what reaches the ships and the dash rows.
func (p Policy) MusterTable(from []traveller.MusterTable) (traveller.MusterTable, error) {
	if p.Muster == MusterCash {
		return prefer(from, []traveller.MusterTable{traveller.TableTwo, traveller.TableOne}), nil
	}

	return prefer(from, []traveller.MusterTable{traveller.TableOne, traveller.TableTwo}), nil
}

// MusterTable1DM is declined only by spartan, which declines both, so that the
// declined branch of each is reachable by a generated golden.
func (p Policy) MusterTable1DM() (bool, error) { return p.Muster != MusterSpartan, nil }

// MusterTable2DM is declined only by spartan, for the same reason (p. 9).
func (p Policy) MusterTable2DM() (bool, error) { return p.Muster != MusterSpartan, nil }

// MusterWeapon converts a repeat into expertise, except under spartan, which
// diversifies - the branch that fills a benefit list with distinct weapons.
func (p Policy) MusterWeapon(
	_ traveller.WeaponCategory, from, received []traveller.WeaponName,
) (traveller.WeaponBenefit, error) {
	if p.Muster != MusterSpartan && len(received) > 0 {
		return traveller.TakeExpertise{Weapon: received[0]}, nil
	}

	if p.Muster == MusterSpartan {
		for _, name := range from {
			if !slices.Contains(received, name) {
				return traveller.TakeWeapon{Weapon: name}, nil
			}
		}
	}

	return traveller.TakeWeapon{Weapon: from[0]}, nil
}
