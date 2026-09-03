package chargen

import (
	"fmt"

	"github.com/philoserf/ctchargen/traveller"
)

// log is the generation record: the ordered account of everything the
// procedure did, which serves audit - any character can be checked against
// Book 1 by walking it - and narrative.
//
// It is wired in from the first step rather than added afterwards, because a
// log that is added afterwards records what the author remembered.
type log struct {
	events []traveller.Event
	errata map[traveller.Erratum]bool
	seq    int
}

func newLog() *log {
	return &log{errata: map[traveller.Erratum]bool{}}
}

func (l *log) next() int {
	l.seq++

	return l.seq
}

// step records a procedure step being entered, with the pages that govern it.
func (l *log) step(name, pages string) {
	l.events = append(l.events, traveller.StepEvent{Seq: l.next(), Step: name, Pages: pages})
}

// throw records a throw and returns its sequence number, so the consequence
// can name what caused it.
func (l *log) throw(step string, t throw) int {
	seq := l.next()

	l.events = append(l.events, traveller.ThrowEvent{
		Seq:       seq,
		Step:      step,
		Dice:      []int{t.dice[0], t.dice[1]},
		DM:        t.modifier,
		Target:    t.target,
		Succeeded: t.succeeded,
	})

	return seq
}

// die records a one-die roll, which has no target to meet.
func (l *log) die(step string, face int) int {
	return l.dice(step, face)
}

// dieWithModifier records a one-die roll taken with a die modifier, so the
// record carries the row that was consulted and not only the face thrown.
//
// P. 9's two mustering out modifiers are the only ones a one-die throw takes,
// and without the modifier the record misreports: a die of 2 with the +1
// gambling expertise allows reads row 3 off the page, and a log that printed
// the 2 alone would send an auditor to the wrong row.
func (l *log) dieWithModifier(step string, face, modifier int) int {
	seq := l.next()

	l.events = append(l.events, traveller.ThrowEvent{
		Seq: seq, Step: step, Dice: []int{face}, DM: modifier, Succeeded: true,
	})

	return seq
}

// dice records a roll with no target to meet, keeping every die. The record
// logs the dice and not only their total, because that is what an auditor
// walks against the page.
func (l *log) dice(step string, faces ...int) int {
	seq := l.next()

	l.events = append(l.events, traveller.ThrowEvent{
		Seq: seq, Step: step, Dice: faces, Succeeded: true,
	})

	return seq
}

// choice records a choice point: which one, who answered, what was offered,
// and what was chosen.
func (l *log) choice(point traveller.ChoicePoint, by traveller.DecidedBy, from []string, chosen string) {
	l.events = append(l.events, traveller.ChoiceEvent{
		Seq: l.next(), Point: point, By: by, Alternatives: from, Chosen: chosen,
	})
}

// outcome records a consequence, the event that caused it, and any recorded
// readings that governed it.
func (l *log) outcomef(because int, errata []traveller.Erratum, format string, args ...any) {
	for _, e := range errata {
		l.errata[e] = true
	}

	l.events = append(l.events, traveller.OutcomeEvent{
		Seq:         l.next(),
		Because:     because,
		Description: fmt.Sprintf(format, args...),
		Errata:      errata,
	})
}

// stamped is every erratum that governed this generation, in id order.
func (l *log) stamped() []traveller.Erratum {
	var stamped []traveller.Erratum

	for _, e := range traveller.Errata {
		if l.errata[e] {
			stamped = append(stamped, e)
		}
	}

	return stamped
}
