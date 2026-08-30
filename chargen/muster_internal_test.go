package chargen

import (
	"errors"
	"testing"

	"github.com/philoserf/ctchargen/service"
)

// wantReceipts is asserted on every accepted throw: Receipts means
// "times received" in both classes, so a repeat of either kind counts,
// even where — as for the scout — nothing is derived from the count.
type shipBenefitCase struct {
	name         string
	held         *Ship
	rolled       string
	wantReceipts int
	wantErr      bool
}

func shipBenefitCases() []shipBenefitCase {
	return []shipBenefitCase{
		{name: "no ship yet, receives a scout", rolled: "scout", wantReceipts: 1},
		{name: "no ship yet, receives a free trader", rolled: "free_trader", wantReceipts: 1},
		{
			name:         "holds a scout, receives another",
			held:         &Ship{Class: classScout, Receipts: 1, ConstructivePossession: true},
			rolled:       "scout",
			wantReceipts: 2,
		},
		{
			name:         "holds a free trader, receives another",
			held:         &Ship{Class: classFreeTrader, Receipts: 1, PaymentYearsRemaining: 40},
			rolled:       "free_trader",
			wantReceipts: 2,
		},
		{
			name:    "holds a scout, receives a free trader",
			held:    &Ship{Class: classScout, Receipts: 1, ConstructivePossession: true},
			rolled:  "free_trader",
			wantErr: true,
		},
		{
			name:    "holds a free trader, receives a scout",
			held:    &Ship{Class: classFreeTrader, Receipts: 1, PaymentYearsRemaining: 40},
			rolled:  "scout",
			wantErr: true,
		},
	}
}

// The cross-kind ship case is unreachable through Generate — no service
// offers both ships, and validateOneShipKind now forbids one from doing
// so — which is exactly why it needs an internal test: the guard exists
// to turn a silent record corruption into a loud failure if that ever
// changes.
func TestShipBenefitRejectsTheOtherKind(t *testing.T) {
	for _, tt := range shipBenefitCases() {
		t.Run(tt.name, func(t *testing.T) {
			g := &generator{char: &Character{Benefits: Benefits{Ship: tt.held}}}

			err := g.shipBenefit("muster-out", tt.rolled, 0)
			if tt.wantErr != (err != nil) {
				t.Fatalf("wantErr %v, got %v", tt.wantErr, err)
			}

			if !tt.wantErr {
				if got := g.char.Benefits.Ship.Receipts; got != tt.wantReceipts {
					t.Errorf("receipts = %d, want %d", got, tt.wantReceipts)
				}

				return
			}

			if !errors.Is(err, ErrBadDecision) {
				t.Errorf("want ErrBadDecision, got %v", err)
			}

			// The point of the guard: the held ship is left alone rather
			// than mutated into a hybrid.
			if got := g.char.Benefits.Ship; got != tt.held {
				t.Errorf("held ship replaced: %+v", got)
			}

			if tt.held.Receipts != 1 {
				t.Errorf("held ship mutated: receipts now %d", tt.held.Receipts)
			}
		})
	}
}

// Holding the same weapon twice is unreachable under the auto policy —
// it always takes the expertise option once one exists, so Benefits.Weapons
// never repeats within a category — which is why the fixtures cannot cover
// this and an internal test must. A repeat receipt may take the same
// weapon again (p. 22), and offering its expertise twice is one option the
// player cannot tell from the other, recorded twice in the event's account
// of what was on the table.
func TestWeaponBenefitOffersEachExpertiseOnce(t *testing.T) {
	reg, err := service.Load()
	if err != nil {
		t.Fatal(err)
	}

	g := &generator{
		reg:     reg,
		decider: AutoPolicy{},
		char: &Character{
			Skills:   []Skill{},
			Benefits: Benefits{Weapons: []string{"Dagger", "Dagger", "Cutlass"}},
		},
	}

	if err := g.weaponBenefit("muster-out", "blade", 0); err != nil {
		t.Fatalf("weaponBenefit: %v", err)
	}

	var choice *Event

	for i := range g.char.Events {
		if g.char.Events[i].Kind == "choice" {
			choice = &g.char.Events[i]

			break
		}
	}

	if choice == nil {
		t.Fatal("no choice event recorded")
	}

	counts := map[string]int{}
	for _, option := range choice.Options {
		counts[option]++
	}

	for _, weapon := range []string{"Dagger", "Cutlass"} {
		if got := counts[ExpertisePrefix+weapon]; got != 1 {
			t.Errorf("%s offered %d times, want once: %v", ExpertisePrefix+weapon, got, choice.Options)
		}
	}
}

// The event log records what was offered. If it aliased the caller's
// slice, a later write through that slice would rewrite history — and the
// skill-table options are a view onto the package-level
// service.TableNames.
func TestChooseDoesNotAliasCallerOptions(t *testing.T) {
	g := &generator{char: &Character{Events: []Event{}}, decider: AutoPolicy{}}

	options := []string{"personal_development", "service_skills"}

	if _, err := g.choose(Choice{Step: "term-1", Label: ChoiceSkillTable, Options: options}); err != nil {
		t.Fatal(err)
	}

	options[0] = "rewritten"

	if got := g.char.Events[0].Options[0]; got != "personal_development" {
		t.Errorf("recorded option followed the caller's slice: %q", got)
	}
}

// A Decider handed no options has nothing to pick; AutoPolicy and the
// prompter would both index past the end of the slice.
func TestChooseRejectsEmptyOptions(t *testing.T) {
	g := &generator{char: &Character{}, decider: AutoPolicy{}}

	_, err := g.choose(Choice{Step: "enlistment", Label: ChoiceService})
	if err == nil {
		t.Fatal("want an error for a choice with no options, got nil")
	}

	if !errors.Is(err, ErrBadDecision) {
		t.Errorf("want ErrBadDecision, got %v", err)
	}
}
