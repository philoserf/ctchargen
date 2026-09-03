package chargen

import "github.com/philoserf/ctchargen/traveller"

// Roller is the source of the procedure's dice, declared here because this
// is where they are consumed. *dice.Stream satisfies it, and so does the
// scripted roller the tests use to walk a particular path.
type Roller interface {
	// Die returns one six-sided die, for the draft (p. 5), the skills
	// tables (p. 11), a mustering out roll (p. 9), and the months a medical
	// crisis recovery adds (pp. 7-8).
	Die() int

	// TwoDice returns the two dice of a 2D throw, first die then second.
	// P. 10: "All rolls except draft are two-die throws."
	TwoDice() (int, int)
}

// Throw is one 2D throw against a target, with its modifier.
type throw struct {
	dice      [2]int
	modifier  int
	target    traveller.Target
	succeeded bool
}

func (t throw) total() int { return t.dice[0] + t.dice[1] + t.modifier }

// roll makes a 2D throw against a target and reports whether it was met.
func roll(r Roller, target traveller.Target, modifier int) throw {
	first, second := r.TwoDice()
	t := throw{dice: [2]int{first, second}, modifier: modifier, target: target}

	t.succeeded = target.Satisfied(t.total())

	return t
}
