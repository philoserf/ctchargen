package chargen

import (
	"slices"
	"testing"

	"github.com/philoserf/ctchargen/dice"
	"github.com/philoserf/ctchargen/service"
)

// A 12 on the reenlistment throw past the voluntary cap is reading E009,
// and a reading has to be stamped or it is being applied silently. No
// golden reaches it — it needs a 12 exactly (1 in 36) in the seventh term
// or later — so the branch is driven directly. Seed 11 throws 6+6, chosen
// by inspection of the stream the way internal/fixture's seeds are.
func TestReenlistmentStampsE009OnlyPastTheCap(t *testing.T) {
	svc := &service.Service{Reenlist: service.ThrowSpec{Target: "6+"}}

	tests := []struct {
		name  string
		term  int
		stamp bool
	}{
		// Below the cap the character still had a say, and the 12 only
		// overrides the answer he gave — the printed rule, not a reading.
		{name: "term 6, still within the voluntary cap", term: 6, stamp: false},
		{name: "term 7, the last voluntary term", term: 7, stamp: true},
		{name: "term 8, already serving on a previous 12", term: 8, stamp: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := &generator{
				stream:  dice.New(11),
				decider: AutoPolicy{},
				char:    &Character{Errata: []string{}, Events: []Event{}},
			}

			stays, err := g.reenlistment(svc, "term", tt.term)
			if err != nil {
				t.Fatalf("reenlistment: %v", err)
			}

			// The 12 forces another term either way; only the stamp differs.
			if !stays {
				t.Fatal("a 12 exactly did not force another term (pp. 6-7)")
			}

			if got := slices.Contains(g.char.Errata, "E009"); got != tt.stamp {
				t.Errorf("E009 stamped = %t, want %t (errata %v)", got, tt.stamp, g.char.Errata)
			}
		})
	}
}

// Past the cap the character is not asked whether he wants to stay — the
// question is closed, and offering it would put a choice event in the
// record that the rules never gave him (p. 7).
func TestReenlistmentStopsAskingPastTheCap(t *testing.T) {
	svc := &service.Service{Reenlist: service.ThrowSpec{Target: "6+"}}

	for _, term := range []int{7, 8} {
		g := &generator{
			stream:  dice.New(11),
			decider: AutoPolicy{},
			char:    &Character{Errata: []string{}, Events: []Event{}},
		}

		if _, err := g.reenlistment(svc, "term", term); err != nil {
			t.Fatalf("term %d: %v", term, err)
		}

		for _, ev := range g.char.Events {
			if ev.Kind == "choice" && ev.Label == ChoiceReenlist {
				t.Errorf("term %d offered a reenlistment choice past the cap", term)
			}
		}
	}
}
