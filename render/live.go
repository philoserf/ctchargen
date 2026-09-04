package render

import (
	"fmt"
	"strings"

	"github.com/philoserf/ctchargen/traveller"
)

// EventLine renders one event for a reader watching a generation happen
// rather than reading it afterwards.
//
// Throws and outcomes go through the same writeEvent the transcript does:
// the wire shape and the domain shape both converge on eventJSON before
// anything is written, so those lines cannot come to differ.
//
// Steps and choices are written here instead, and deliberately do differ.
// A step's heading is unnumbered in the transcript, and a choice used to
// render as nothing at all here; either leaves a gap a watcher reads as a
// lost event, so both are written with their number. The choice is echoed
// as that number and the answer alone rather than the transcript's full
// form, because the player has just been asked the question and can still
// see the point, the decider and the alternatives above him; repeating them
// pushes the useful lines off the screen.
//
// It returns no error, as foldDeparture does not: every case of liveCodec
// fills a struct field, none of them can fail, and traveller.Event is
// sealed, so no fifth case can arrive from outside to fail either. An error
// return here would be a branch no test could reach.
func EventLine(event traveller.Event) string {
	var lined liveCodec

	_ = event.Fold(&lined)

	// The gap these two close is what made the numbers appear to skip - 17,
	// 18, then 20. Nothing was ever missing: the headings printed without
	// their number, and the questions printed nothing at all.
	switch lined.event.Kind {
	case kindStep:
		return fmt.Sprintf("\n## %d. %s (%s)\n\n",
			lined.event.Seq, lined.event.Step, lined.event.Pages)
	case kindChoice:
		// Echoed after the answer, because a choice's number does not exist
		// until it is made - the prompt cannot carry it. The transcript's
		// own line names the point, the decider and the alternatives, all
		// of which the person who just typed the answer can see above.
		return fmt.Sprintf("%3d. you chose %s\n", lined.event.Seq, lined.event.Chosen)
	}

	var out strings.Builder

	writeEvent(&out, lined.event)

	return out.String()
}

// liveCodec folds a domain event into the wire shape the renderer reads.
// It is a fold, so a fifth kind of event stops this compiling.
type liveCodec struct{ event eventJSON }

func (l *liveCodec) Step(from traveller.StepEvent) error {
	l.event = eventJSON{Seq: from.Seq, Kind: kindStep, Step: from.Step, Pages: from.Pages}

	return nil
}

func (l *liveCodec) Throw(from traveller.ThrowEvent) error {
	l.event = eventJSON{
		Seq: from.Seq, Kind: kindThrow, Step: from.Step, Dice: from.Dice,
		DM: from.DM, Total: from.Total(), Succeeded: from.Succeeded,
	}
	if from.Target.Number() != 0 {
		l.event.Target = from.Target.String()
	}

	return nil
}

func (l *liveCodec) Choice(from traveller.ChoiceEvent) error {
	// Chosen is carried because the live view echoes it. It was dropped
	// while choices rendered as nothing at all, which is a field the fold
	// cannot catch: a case that stops filling one still compiles, and only
	// what reads it notices.
	l.event = eventJSON{Seq: from.Seq, Kind: kindChoice, Chosen: from.Chosen}

	return nil
}

func (l *liveCodec) Outcome(from traveller.OutcomeEvent) error {
	l.event = eventJSON{
		Seq: from.Seq, Kind: kindOutcome, Because: from.Because, Description: from.Description,
	}
	for _, erratum := range from.Errata {
		l.event.Errata = append(l.event.Errata, erratum.String())
	}

	return nil
}
