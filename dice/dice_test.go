package dice_test

import (
	"testing"

	"github.com/philoserf/ctchargen/dice"
)

// A smoke test over the stream, and no more than that. The properties it
// would be tempting to assert here — that a die lands in 1-6, that one seed
// reproduces another's rolls, that two seeds diverge — belong to
// math/rand/v2 and its PCG source, not to this package, and asserting them
// pins nothing this repository can break. What this repository can break is
// whether the recorded seed reaches the generator at all, and that is what
// chargen's TestReplayRoundTrip proves across fourteen records and several
// hundred rolls each, byte for byte.
func TestStreamIsSeededAndOrdered(t *testing.T) {
	a, b := dice.New(42), dice.New(42)

	// Two() is two One()s in order, which is the part of the contract
	// replay depends on: consumption order is load-bearing.
	d1, d2 := a.Two()
	if d1 != b.One() || d2 != b.One() {
		t.Fatalf("Two() = %d,%d, which is not the next two One()s of an identical stream", d1, d2)
	}

	if d1 < 1 || d1 > 6 || d2 < 1 || d2 > 6 {
		t.Fatalf("dice outside 1-6: %d, %d", d1, d2)
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
