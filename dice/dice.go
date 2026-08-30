// Package dice is the rules' die-roll mechanics (Book 1 pp. 2-3): a seeded
// stream of six-sided dice, two-die throws with cumulative DMs against
// N+/N-/exact targets, and the single-die rolls the procedure uses. The
// stream is the only randomness in the program; every roll is consumed
// from it in procedure order, which makes that order load-bearing for
// replay (docs/PRD.md, Replay and provenance contract).
package dice

import (
	"errors"
	"fmt"
	"math/rand/v2"
	"strconv"
)

// Algorithm names the RNG in every character record's provenance block.
// Changing the algorithm (or how the seed expands into PCG state) is an
// engine version bump.
const Algorithm = "go-math-rand-v2-pcg"

// Stream is a seeded source of six-sided dice. It does not keep the seed:
// the record carries it (chargen.RNG), and one copy of a value replay
// depends on is better than two that could disagree.
type Stream struct {
	rng *rand.Rand
}

// New returns a stream seeded with the given value. The single recorded
// seed fills both words of the PCG state, so the record's one seed field
// reproduces the stream exactly.
func New(seed uint64) *Stream {
	// The deterministic seeded stream is the contract (FR9; replay depends
	// on it), so the "weak" generator is the required one, not an oversight.
	return &Stream{rng: rand.New(rand.NewPCG(seed, seed))} // #nosec G404
}

// One rolls a single die (1-6).
func (s *Stream) One() int { return s.rng.IntN(6) + 1 }

// Two rolls two dice in order and reports them individually; throws
// record the dice, not just the sum (FR10).
func (s *Stream) Two() (int, int) { return s.One(), s.One() }

// Mode is how a target reads: 8+ means the total must equal or exceed 8,
// 8- means equal or less, and an exact target must match (pp. 2-3).
type Mode int

// Target modes, in the notation of pp. 2-3.
const (
	Plus Mode = iota
	Minus
	Exact
)

// Target is a throw's required result.
type Target struct {
	Value int
	Mode  Mode
}

// Met reports whether a throw total (dice plus DMs) satisfies the target.
func (t Target) Met(total int) bool {
	switch t.Mode {
	case Plus:
		return total >= t.Value
	case Minus:
		return total <= t.Value
	case Exact:
		return total == t.Value
	}

	return false
}

// ErrBadTarget reports throw-target notation that does not parse.
var ErrBadTarget = errors.New("bad throw target")

// ParseTarget reads the book's notation: "8+", "8-", or an exact "12".
// It is the inverse of String, used by the data files' load-time
// validation.
func ParseTarget(s string) (Target, error) {
	if s == "" {
		return Target{}, fmt.Errorf("%w: empty", ErrBadTarget)
	}

	mode := Exact
	digits := s

	switch s[len(s)-1] {
	case '+':
		mode, digits = Plus, s[:len(s)-1]
	case '-':
		mode, digits = Minus, s[:len(s)-1]
	}

	value, err := strconv.Atoi(digits)
	if err != nil || value < 2 || value > 12 {
		return Target{}, fmt.Errorf("%w: %q, want a two-die value 2-12 with optional +/-", ErrBadTarget, s)
	}

	return Target{Value: value, Mode: mode}, nil
}

// String renders the target in the book's notation: "8+", "8-", "8".
func (t Target) String() string {
	switch t.Mode {
	case Plus:
		return strconv.Itoa(t.Value) + "+"
	case Minus:
		return strconv.Itoa(t.Value) + "-"
	case Exact:
		return strconv.Itoa(t.Value)
	default:
		return strconv.Itoa(t.Value)
	}
}
