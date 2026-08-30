package chargen

import (
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/philoserf/ctchargen/dice"
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
// slice, a later write through that slice would rewrite history. The
// callers now hand over slices they own — service.TableNames returns a
// fresh one — so this guards the other direction: nothing stops a Decider
// writing to the elements it was shown.
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

// The crisis save's two branches, neither reachable through the golden
// roster: the one crisis fixture is a death, so the recovery path — the
// zero becoming one, the 1D months of added age, the outcome that reports
// them (pp. 7-8) — runs in no other test, and render says the same of its
// own goldens (render.TestSheetCarriesRecoveryMonths). Seeds are chosen by
// inspection of the stream, the way internal/fixture's are: seed 4 throws
// 5+5=10 and then a 3 for the months, seed 1 throws 6+1=7 against the 8+.
//
// medicalCrisis reads only the stream and the record — no registry, no
// Decider — so the generator can be built by hand.
func TestMedicalCrisis(t *testing.T) {
	tests := []struct {
		name       string
		seed       uint64
		wantDied   bool
		wantValue  int
		wantMonths int
		wantErrata []string
	}{
		{
			name: "survives the 8+ and recovers", seed: 4,
			wantDied: false, wantValue: 1, wantMonths: 3,
			wantErrata: []string{"E007"},
		},
		{
			// Stamped in the order crossed, not sorted: E007 on entering the
			// crisis at all, E006 only once the save is failed.
			name: "fails the 8+ and dies (E006)", seed: 1,
			wantDied: true, wantValue: 0, wantMonths: 0,
			wantErrata: []string{"E007", "E006"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := &generator{
				stream: dice.New(tt.seed),
				char: &Character{
					Age:             42,
					Characteristics: Characteristics{Strength: 0, Endurance: 7},
					Errata:          []string{},
					Events:          []Event{},
				},
			}

			died, err := g.medicalCrisis("term-6", 6, service.Strength)
			if err != nil {
				t.Fatalf("medicalCrisis: %v", err)
			}

			if died != tt.wantDied {
				t.Errorf("died = %t, want %t", died, tt.wantDied)
			}

			if got := g.char.Characteristics.Strength; got != tt.wantValue {
				t.Errorf("strength = %d, want %d", got, tt.wantValue)
			}

			// Recovery ages the character by whole months, and 1D never
			// carries a year, so the years must stand still.
			if got := g.char.AgeMonths; got != tt.wantMonths {
				t.Errorf("age months = %d, want %d", got, tt.wantMonths)
			}

			if g.char.Age != 42 {
				t.Errorf("age = %d, want 42 unchanged", g.char.Age)
			}

			if !slices.Equal(g.char.Errata, tt.wantErrata) {
				t.Errorf("errata = %v, want %v", g.char.Errata, tt.wantErrata)
			}

			assertCrisisOutcome(t, g.char, tt.wantDied)
		})
	}
}

// The record has to say which way the crisis went: a death carries its
// term and cause (FR8), a recovery reports the months it cost.
func assertCrisisOutcome(t *testing.T, char *Character, died bool) {
	t.Helper()

	if !died {
		if char.Death != nil {
			t.Errorf("survivor carries a death: %+v", char.Death)
		}

		if !hasOutcomeContaining(char, "recovered") {
			t.Errorf("no recovery outcome recorded: %+v", char.Events)
		}

		return
	}

	if char.Death == nil {
		t.Fatal("a failed crisis save recorded no death")
	}

	if char.Death.Term != 6 {
		t.Errorf("death term = %d, want 6", char.Death.Term)
	}

	if !strings.Contains(char.Death.Cause, "medical crisis") {
		t.Errorf("death cause %q does not name the medical crisis", char.Death.Cause)
	}
}

func hasOutcomeContaining(char *Character, text string) bool {
	for _, ev := range char.Events {
		if ev.Kind == "outcome" && strings.Contains(ev.Text, text) {
			return true
		}
	}

	return false
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
