package render

import (
	"fmt"
	"strings"

	"github.com/philoserf/ctchargen/traveller"
)

// EventLine renders one event as the transcript renders it, for a reader
// watching a generation happen rather than reading it afterwards.
//
// It goes through the same writeEvent the transcript does: the wire shape
// and the domain shape both converge on eventJSON before anything is
// written, so a live line and a transcript line cannot come to differ.
//
// A choice returns the empty string. Interactive mode is the only caller,
// and it has just asked the question and read the answer; repeating it back
// pushes the useful lines off the screen.
//
// It returns no error, as foldDeparture does not: every case of liveCodec
// fills a struct field, none of them can fail, and traveller.Event is
// sealed, so no fifth case can arrive from outside to fail either. An error
// return here would be a branch no test could reach.
func EventLine(event traveller.Event) string {
	var lined liveCodec

	_ = event.Fold(&lined)

	// Two kinds are written here rather than by the transcript's renderer,
	// because they are read by a person watching his own generation rather
	// than by one auditing someone else's.
	//
	// Both carry a sequence number the transcript does not print, and the
	// gap that leaves is what made the numbers appear to skip - 17, 18, then
	// 20. Nothing was ever missing; the headings and the questions were
	// simply unnumbered.
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
