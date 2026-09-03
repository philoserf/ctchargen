package traveller

import "fmt"

// Event is one entry in the generation record: the ordered log of the whole
// generation, which serves audit — verify any character against Book 1 by
// walking the log — and narrative.
//
// It is a sum for the same reason the others are: each kind carries only its
// own fields, and a renderer that grows a fifth kind cannot forget to handle
// it.
type Event interface {
	Fold(EventCases) error
	Sequence() int
	sealedEvent()
}

// EventCases handles each kind of log entry.
type EventCases interface {
	Step(StepEvent) error
	Throw(ThrowEvent) error
	Choice(ChoiceEvent) error
	Outcome(OutcomeEvent) error
}

// StepEvent records a procedure step being entered, with the pages that
// govern it.
type StepEvent struct {
	Seq   int
	Step  string
	Pages string
}

// ThrowEvent records one throw: the dice as rolled, the modifier applied,
// the target, and whether it was met.
//
// The dice are kept, not only their sum, because the log is what an auditor
// walks against the page — and because a 12 exactly on the reenlistment
// throw is a different event from a 12 reached any other way (pp. 6-7).
type ThrowEvent struct {
	Seq       int
	Step      string
	Dice      []int
	DM        int
	Target    Target
	Succeeded bool
}

// Total is the throw's result: the dice plus the modifier.
func (t ThrowEvent) Total() int {
	total := t.DM
	for _, die := range t.Dice {
		total += die
	}

	return total
}

// DecidedBy names who answered a choice — the player, or the auto policy.
//
// It is not the chargen.Decider interface and must not be confused with it:
// that is the thing asked, this is which kind of thing answered.
type DecidedBy int

// The two who can answer (FR9).
const (
	ByPlayer DecidedBy = iota
	ByPolicy
)

// DecidedBys is both answerers, for iteration.
var DecidedBys = [...]DecidedBy{ByPlayer, ByPolicy}

func (d DecidedBy) String() string {
	switch d {
	case ByPlayer:
		return "player"
	case ByPolicy:
		return "policy"
	}

	return fmt.Sprintf("DecidedBy(%d)", int(d))
}

// ChoiceEvent records a choice point: which one, who answered, what the
// alternatives were, and what was chosen.
type ChoiceEvent struct {
	Seq          int
	Point        ChoicePoint
	By           DecidedBy
	Alternatives []string
	Chosen       string
}

// OutcomeEvent records a consequence — a skill gained, a characteristic
// changed, years elapsed, a death — and the throw or choice that caused it.
//
// Because is the sequence number of the causing event, or zero where the
// consequence follows from the procedure itself rather than from a roll.
type OutcomeEvent struct {
	Seq         int
	Because     int
	Description string
	Errata      []Erratum
}

func (e StepEvent) Fold(c EventCases) error    { return c.Step(e) }
func (t ThrowEvent) Fold(c EventCases) error   { return c.Throw(t) }
func (e ChoiceEvent) Fold(c EventCases) error  { return c.Choice(e) }
func (e OutcomeEvent) Fold(c EventCases) error { return c.Outcome(e) }
func (e StepEvent) Sequence() int              { return e.Seq }
func (t ThrowEvent) Sequence() int             { return t.Seq }
func (e ChoiceEvent) Sequence() int            { return e.Seq }
func (e OutcomeEvent) Sequence() int           { return e.Seq }
func (StepEvent) sealedEvent()                 {}
func (ThrowEvent) sealedEvent()                {}
func (ChoiceEvent) sealedEvent()               {}
func (OutcomeEvent) sealedEvent()              {}
