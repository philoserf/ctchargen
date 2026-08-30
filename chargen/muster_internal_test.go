package chargen

import (
	"errors"
	"testing"
)

// The cross-kind ship case is unreachable through Generate — no service
// offers both ships, and validateOneShipKind now forbids one from doing
// so — which is exactly why it needs an internal test: the guard exists
// to turn a silent record corruption into a loud failure if that ever
// changes.
func TestShipBenefitRejectsTheOtherKind(t *testing.T) {
	tests := []struct {
		name    string
		held    *Ship
		rolled  string
		wantErr bool
	}{
		{name: "no ship yet, receives a scout", rolled: "scout"},
		{name: "no ship yet, receives a free trader", rolled: "free_trader"},
		{
			name:   "holds a scout, receives another",
			held:   &Ship{Class: classScout, Receipts: 1, ConstructivePossession: true},
			rolled: "scout",
		},
		{
			name:   "holds a free trader, receives another",
			held:   &Ship{Class: classFreeTrader, Receipts: 1, PaymentYearsRemaining: 40},
			rolled: "free_trader",
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

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := &generator{char: &Character{Benefits: Benefits{Ship: tt.held}}}

			err := g.shipBenefit("muster-out", tt.rolled, 0)
			if tt.wantErr != (err != nil) {
				t.Fatalf("wantErr %v, got %v", tt.wantErr, err)
			}

			if !tt.wantErr {
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
