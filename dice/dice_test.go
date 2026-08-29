package dice_test

import (
	"testing"

	"github.com/philoserf/ctchargen/dice"
)

func TestOneStaysInBounds(t *testing.T) {
	s := dice.New(1)
	for range 10_000 {
		if v := s.One(); v < 1 || v > 6 {
			t.Fatalf("One() = %d, want 1-6", v)
		}
	}
}

func TestSameSeedSameStream(t *testing.T) {
	a, b := dice.New(42), dice.New(42)
	for i := range 1_000 {
		if av, bv := a.One(), b.One(); av != bv {
			t.Fatalf("roll %d: streams diverge (%d != %d) for identical seed", i, av, bv)
		}
	}
}

func TestDifferentSeedsDiverge(t *testing.T) {
	a, b := dice.New(1), dice.New(2)
	for range 1_000 {
		if a.One() != b.One() {
			return
		}
	}

	t.Fatal("streams for seeds 1 and 2 identical over 1000 rolls")
}

// Consumption order is load-bearing for replay: interleaving another roll
// must change what the stream hands out next.
func TestStreamOrderMatters(t *testing.T) {
	a, b := dice.New(7), dice.New(7)

	first, _ := a.Two()
	if first != b.One() {
		t.Fatal("first die of Two() differs from One() on an identical stream")
	}
}

func TestTargetMet(t *testing.T) {
	tests := []struct {
		target dice.Target
		total  int
		want   bool
	}{
		{dice.Target{Value: 8, Mode: dice.Plus}, 8, true},
		{dice.Target{Value: 8, Mode: dice.Plus}, 7, false},
		{dice.Target{Value: 8, Mode: dice.Plus}, 12, true},
		{dice.Target{Value: 5, Mode: dice.Minus}, 5, true},
		{dice.Target{Value: 5, Mode: dice.Minus}, 6, false},
		{dice.Target{Value: 12, Mode: dice.Exact}, 12, true},
		{dice.Target{Value: 12, Mode: dice.Exact}, 11, false},
	}
	for _, tt := range tests {
		if got := tt.target.Met(tt.total); got != tt.want {
			t.Errorf("%s.Met(%d) = %v, want %v", tt.target, tt.total, got, tt.want)
		}
	}
}

func TestParseTarget(t *testing.T) {
	good := []string{"3+", "5-", "12", "2+"}
	for _, s := range good {
		target, err := dice.ParseTarget(s)
		if err != nil {
			t.Errorf("ParseTarget(%q) error: %v", s, err)

			continue
		}

		if got := target.String(); got != s {
			t.Errorf("ParseTarget(%q).String() = %q, want round-trip", s, got)
		}
	}

	bad := []string{"", "+", "13+", "1+", "3a", "3a+", "x"}
	for _, s := range bad {
		if _, err := dice.ParseTarget(s); err == nil {
			t.Errorf("ParseTarget(%q) succeeded, want error", s)
		}
	}
}

func TestTargetString(t *testing.T) {
	tests := []struct {
		target dice.Target
		want   string
	}{
		{dice.Target{Value: 8, Mode: dice.Plus}, "8+"},
		{dice.Target{Value: 5, Mode: dice.Minus}, "5-"},
		{dice.Target{Value: 12, Mode: dice.Exact}, "12"},
	}
	for _, tt := range tests {
		if got := tt.target.String(); got != tt.want {
			t.Errorf("Target%v.String() = %q, want %q", tt.target, got, tt.want)
		}
	}
}
