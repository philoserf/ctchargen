package chargen

import "testing"

// termOf is how two strategies learn the term: it reads the number out of
// the step the engine stamps ("term-4"). Every step that is not a term
// must read as 0 rather than as some term, because 0 is what turns a
// term-keyed strategy off — and the malformed cases are unreachable from
// the engine, which is exactly why they need driving directly.
func TestTermOfReadsOnlyTermSteps(t *testing.T) {
	tests := []struct {
		step string
		want int
	}{
		{"term-1", 1},
		{"term-4", 4},
		{"term-12", 12},
		{"enlistment", 0},
		{"muster-out", 0},
		{"characteristics", 0},
		{"term-", 0},
		{"term-four", 0},
		{"term-4x", 0},
	}
	for _, tt := range tests {
		t.Run(tt.step, func(t *testing.T) {
			if got := termOf(tt.step); got != tt.want {
				t.Errorf("termOf(%q) = %d, want %d", tt.step, got, tt.want)
			}
		})
	}
}
