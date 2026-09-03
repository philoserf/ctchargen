// Package dice draws the die rolls Classic Traveller's character generation
// procedure calls for, from a seeded stream.
//
// It rolls and it never judges. Whether a throw meets its target is the
// business of a Target (Book 1 pp. 2-3), which lives in the traveller
// package; keeping that out of here is what lets dice depend on nothing.
//
// Two properties of this package are load-bearing, in the sense that a seed
// means nothing without them:
//
//   - The die is IntN(6) + 1. An IntN(36), or a masked Uint64, is the same
//     PCG under the same seed and an entirely different character.
//   - A 2D throw is two of those in sequence, first die then second.
//
// Changing either changes every seeded character. That is an ordinary
// change, not a breaking one, but it is never an accidental one.
package dice

import "math/rand/v2"

// Stream is a seeded source of dice. The zero value is not usable; call New.
type Stream struct {
	r     *rand.Rand
	drawn int
}

// New returns a Stream seeded by seed. The seed fills both words of the PCG
// state, so a single recorded number reproduces the whole stream.
func New(seed uint64) *Stream {
	return &Stream{r: rand.New(rand.NewPCG(seed, seed))}
}

// Die returns one six-sided die: 1 through 6.
//
// The procedure's one-die throws are the draft (Book 1 p. 5), the acquired
// skills tables (p. 11), each mustering out roll (p. 9), and the months of
// age a medical crisis recovery adds (p. 7).
func (s *Stream) Die() int {
	s.drawn++
	return s.r.IntN(6) + 1
}

// TwoDice returns the two dice of a 2D throw, first die then second.
//
// Book 1 p. 10: "All rolls except draft are two-die throws." The caller sums
// them; both are returned because the generation record logs the dice, not
// only their total.
func (s *Stream) TwoDice() (first, second int) {
	first = s.Die()
	second = s.Die()
	return first, second
}

// Drawn reports how many dice have been taken from the stream.
//
// Dice-stream consumption order is what a seed means, so a test that means
// to pin an order pins this count alongside the values: a throw the
// procedure does not make must consume nothing.
func (s *Stream) Drawn() int {
	return s.drawn
}
