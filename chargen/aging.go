package chargen

import (
	"fmt"

	"github.com/philoserf/ctchargen/traveller"
)

// crisisFloor is what a characteristic reduced by aging may reach. E010
// reads p. 4's floor as 0 for aging alone: zero is the only value below one
// the rules give any meaning to, and pp. 7-8 give it a complete one.
const crisisFloor = 0

// agingRound runs the term's aging round, which is the last step of the term
// (E006) and is read off the table's term row rather than its age row.
//
// The saving throws are made in the table's own row order (E007), and a
// characteristic that reaches zero resolves its crisis inline, before the
// next characteristic is thrown for - p. 7 says the recovery "is made
// immediately".
func (r *run) agingRound(term traveller.Term) error {
	effects := r.tables.Aging.At(term)
	if len(effects) == 0 {
		return nil
	}

	r.log.step(fmt.Sprintf("aging, end of term %d", term), "pp. 7-9")

	for _, effect := range effects {
		throw := roll(r.roll, effect.Saving, 0)
		seq := r.log.throw("aging, "+effect.Characteristic.String(), throw)

		if throw.succeeded {
			r.log.outcomef(seq, agingErrata(term), "%v holds", effect.Characteristic)

			continue
		}

		err := r.age(seq, term, effect.Characteristic, effect.Reduction)
		if err != nil {
			return err
		}

		if r.dead {
			return nil
		}
	}

	return nil
}

// agingErrata names the readings every aging round rests on: where the round
// sits in the term, and the order its throws are made in. Past the table's
// last printed term, E014 joins them.
func agingErrata(term traveller.Term) []traveller.Erratum {
	errata := []traveller.Erratum{traveller.E006, traveller.E007}

	const lastPrintedTerm = 14

	if term > lastPrintedTerm {
		errata = append(errata, traveller.E014)
	}

	return errata
}

// age applies one reduction and resolves the crisis if it reaches zero.
func (r *run) age(
	seq int, term traveller.Term, characteristic traveller.Characteristic, reduction int,
) error {
	before := r.char.Profile[characteristic]

	r.char.Profile = r.char.Profile.AgeReduce(characteristic, reduction)

	after := r.char.Profile[characteristic]

	errata := agingErrata(term)
	if before-reduction < crisisFloor {
		// The reduction was stopped by the floor, which is the reading
		// itself rather than a consequence of it.
		errata = append(errata, traveller.E010)
	}

	r.log.outcomef(seq, errata, "%v -%d, %d to %d", characteristic, reduction, before, after)

	if after > crisisFloor {
		return nil
	}

	return r.crisis(term, characteristic)
}

// crisis resolves a characteristic reduced to zero (pp. 7-8): "A basic
// saving throw of 8+ applies (and may be modified by the expertise of
// attending medical personnel)."
//
// No modifier applies: generation has no attending personnel for the printed
// DM to refer to (E009). A failed throw is death, which the page does not
// state and E008 reads from "If the character survives".
func (r *run) crisis(term traveller.Term, characteristic traveller.Characteristic) error {
	crisis := r.tables.Aging.Crisis
	errata := []traveller.Erratum{traveller.E008, traveller.E009}

	r.log.step(fmt.Sprintf("medical crisis, %v at zero", characteristic), "pp. 7-8")

	throw := roll(r.roll, crisis.Saving, 0)
	seq := r.log.throw("medical crisis", throw)

	if !throw.succeeded {
		r.log.outcomef(seq, errata, "died of a medical crisis in term %d", term)

		r.char.Departure = traveller.KilledByMedicalCrisis{Characteristic: characteristic}
		r.dead = true

		return nil
	}

	r.char.Profile = r.char.Profile.Alter(characteristic, crisis.RecoversTo-r.char.Profile[characteristic])

	months := 0
	for range crisis.MonthsDice {
		months += r.roll.Die()
	}

	r.char.Age = r.char.Age.PlusMonths(months)

	monthSeq := r.log.die("months of recovery", months)
	r.log.outcomef(monthSeq, errata, "recovered: %v to %d, and %d months older, now aged %v",
		characteristic, crisis.RecoversTo, months, r.char.Age)

	return nil
}
