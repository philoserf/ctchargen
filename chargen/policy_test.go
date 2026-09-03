package chargen_test

import (
	"testing"

	"github.com/philoserf/ctchargen/chargen"
	"github.com/philoserf/ctchargen/traveller"
)

// Every row of POLICY.md, checked against what the document says. The gate
// in internal/docsgate holds the row set to the interface; this holds each
// row's answer to what the row promises.
func TestCareerStrategies(t *testing.T) {
	t.Parallel()

	for name, tc := range map[string]struct {
		career  string
		offered []traveller.Intent
		want    traveller.Intent
		draft   bool
	}{
		"serve continues while it can": {
			chargen.CareerServe,
			[]traveller.Intent{traveller.Continue, traveller.Retire},
			traveller.Continue, true,
		},
		"serve retires when it must": {
			chargen.CareerServe, []traveller.Intent{traveller.Retire}, traveller.Retire, true,
		},
		"retire leaves as soon as leaving is retirement": {
			chargen.CareerRetire,
			[]traveller.Intent{traveller.Continue, traveller.Retire},
			traveller.Retire, true,
		},
		"retire continues while it cannot retire": {
			chargen.CareerRetire,
			[]traveller.Intent{traveller.Continue, traveller.Discharge},
			traveller.Continue, true,
		},
		"oneterm leaves at the first opportunity": {
			chargen.CareerOneTerm,
			[]traveller.Intent{traveller.Continue, traveller.Discharge},
			traveller.Discharge, false,
		},
	} {
		policy := chargen.Policy{Career: tc.career, Skills: chargen.SkillsAdvanced, Muster: chargen.MusterCash}

		got, err := policy.ReenlistIntent(tc.offered)
		if err != nil || got != tc.want {
			t.Errorf("%s: chose %v (%v), want %v", name, got, err, tc.want)
		}

		submit, err := policy.SubmitToDraft()
		if err != nil || submit != tc.draft {
			t.Errorf("%s: submits to the draft = %v (%v), want %v", name, submit, err, tc.draft)
		}
	}
}

func TestSkillsStrategies(t *testing.T) {
	t.Parallel()

	all := traveller.SkillTables[:]
	withoutTheGate := all[:len(all)-1]

	for name, tc := range map[string]struct {
		strategy string
		offered  []traveller.SkillTable
		want     traveller.SkillTable
	}{
		"advanced takes the fourth table the instant it opens": {
			chargen.SkillsAdvanced, all, traveller.AdvancedEducationEight,
		},
		"advanced falls back when the gate is shut": {
			chargen.SkillsAdvanced, withoutTheGate, traveller.AdvancedEducation,
		},
		"service prefers the service table": {
			chargen.SkillsService, all, traveller.ServiceSkills,
		},
		"personal reaches the one negative result": {
			chargen.SkillsPersonal, all, traveller.PersonalDevelopment,
		},
	} {
		policy := chargen.Policy{Career: chargen.CareerServe, Skills: tc.strategy, Muster: chargen.MusterCash}

		got, err := policy.SkillTable(tc.offered)
		if err != nil || got != tc.want {
			t.Errorf("%s: designated %v (%v), want %v", name, got, err, tc.want)
		}
	}
}

func TestMusterStrategies(t *testing.T) {
	t.Parallel()

	both := traveller.MusterTables[:]

	for name, tc := range map[string]struct {
		strategy string
		want     traveller.MusterTable
		takesDMs bool
	}{
		"cash prefers table 2, which is what reaches the three-roll cap": {
			chargen.MusterCash, traveller.TableTwo, true,
		},
		"goods prefers table 1, which is what reaches the ships": {
			chargen.MusterGoods, traveller.TableOne, true,
		},
		"spartan prefers table 1 and declines both modifiers": {
			chargen.MusterSpartan, traveller.TableOne, false,
		},
	} {
		policy := chargen.Policy{Career: chargen.CareerServe, Skills: chargen.SkillsAdvanced, Muster: tc.strategy}

		got, err := policy.MusterTable(both)
		if err != nil || got != tc.want {
			t.Errorf("%s: designated %v (%v), want %v", name, got, err, tc.want)
		}

		one, _ := policy.MusterTable1DM()
		two, _ := policy.MusterTable2DM()

		if one != tc.takesDMs || two != tc.takesDMs {
			t.Errorf("%s: takes the modifiers = %v and %v, want %v", name, one, two, tc.takesDMs)
		}
	}
}

// P. 22 offers expertise "in lieu of receiving a second or subsequent weapon
// of exactly the same type", so it can only be taken in one already
// received. spartan is the strategy that diversifies instead.
func TestMusterWeaponStrategies(t *testing.T) {
	t.Parallel()

	list := []traveller.WeaponName{dagger, "Blade", "Foil"}
	received := []traveller.WeaponName{dagger}

	cash := chargen.Policy{Career: chargen.CareerServe, Skills: chargen.SkillsAdvanced, Muster: chargen.MusterCash}

	taken, err := cash.MusterWeapon(traveller.Blade, list, received)
	if err != nil {
		t.Fatalf("cash: %v", err)
	}

	if _, expertise := taken.(traveller.TakeExpertise); !expertise {
		t.Errorf("cash took %T on a repeat, want expertise", taken)
	}

	// A first receipt has nothing to take expertise in.
	first, err := cash.MusterWeapon(traveller.Blade, list, nil)
	if err != nil {
		t.Fatalf("cash, first receipt: %v", err)
	}

	if _, weapon := first.(traveller.TakeWeapon); !weapon {
		t.Errorf("cash took %T on a first receipt, want the weapon", first)
	}

	spartan := chargen.Policy{Career: chargen.CareerServe, Skills: chargen.SkillsAdvanced, Muster: chargen.MusterSpartan}

	diversified, err := spartan.MusterWeapon(traveller.Blade, list, received)
	if err != nil {
		t.Fatalf("spartan: %v", err)
	}

	if got, ok := diversified.(traveller.TakeWeapon); !ok || got.Weapon != "Blade" {
		t.Errorf("spartan took %v, want the first weapon not already received", diversified)
	}
}

// Service ranks by the odds of the throw, ties to book order (p. 10).
func TestServicePicksTheLikeliestEnlistment(t *testing.T) {
	t.Parallel()

	target := func(s string) traveller.Target {
		parsed, err := traveller.ParseTarget(s)
		if err != nil {
			t.Fatalf("parsing %q: %v", s, err)
		}

		return parsed
	}

	policy := chargen.DefaultPolicy()

	chosen, err := policy.Service([]traveller.EnlistmentOffer{
		{Service: traveller.Navy, Target: target("8+")},
		{Service: traveller.Other, Target: target("3+")},
		{Service: traveller.Army, Target: target("5+")},
	})
	if err != nil || chosen != traveller.Other {
		t.Errorf("chose %v (%v), want Other, whose 3+ is the likeliest", chosen, err)
	}

	// A modifier is part of the odds.
	chosen, err = policy.Service([]traveller.EnlistmentOffer{
		{Service: traveller.Navy, Target: target("8+"), DM: 6},
		{Service: traveller.Other, Target: target("3+")},
	})
	if err != nil || chosen != traveller.Navy {
		t.Errorf("chose %v (%v), want Navy, whose 8+ with a +6 is certain", chosen, err)
	}
}

func TestValidateRefusesAStrategyNoRowCarries(t *testing.T) {
	t.Parallel()

	err := chargen.DefaultPolicy().Validate()
	if err != nil {
		t.Errorf("the default policy is invalid: %v", err)
	}

	for _, invalid := range []chargen.Policy{
		{Career: "dawdle", Skills: chargen.SkillsAdvanced, Muster: chargen.MusterCash},
		{Career: chargen.CareerServe, Skills: "osmosis", Muster: chargen.MusterCash},
		{Career: chargen.CareerServe, Skills: chargen.SkillsAdvanced, Muster: "hoard"},
	} {
		err := invalid.Validate()
		if err == nil {
			t.Errorf("%+v was accepted", invalid)
		}
	}
}
