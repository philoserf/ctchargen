package render

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/philoserf/ctchargen/chargen"
	"github.com/philoserf/ctchargen/traveller"
)

// Transcript renders the generation record as prose: every step entered,
// every throw made, every choice answered, every consequence.
//
// It is what makes a character auditable against Book 1 - walk the log with
// the page open - and it is the narrative service record besides.
func Transcript(character *chargen.Character) string {
	var out strings.Builder

	fmt.Fprintf(&out, "# Generation record: %s\n\n", nameOrBlank(character.Name))
	fmt.Fprintf(&out, "Seed %d, strategies %s/%s/%s.\n\n",
		character.Inputs.Seed, character.Inputs.Career, character.Inputs.Skills, character.Inputs.Muster)

	lines := &transcriber{out: &out}

	for _, event := range character.Events {
		err := event.Fold(lines)
		if err != nil {
			fmt.Fprintf(&out, "    (unreadable event %d)\n", event.Sequence())
		}
	}

	return out.String()
}

type transcriber struct{ out *strings.Builder }

func (t *transcriber) Step(from traveller.StepEvent) error {
	fmt.Fprintf(t.out, "\n## %s (%s)\n\n", from.Step, from.Pages)

	return nil
}

func (t *transcriber) Throw(from traveller.ThrowEvent) error {
	dice := make([]string, 0, len(from.Dice))
	for _, die := range from.Dice {
		dice = append(dice, strconv.Itoa(die))
	}

	line := fmt.Sprintf("%3d. %s: rolled %s", from.Seq, from.Step, strings.Join(dice, "+"))

	if from.DM != 0 {
		line += fmt.Sprintf(" %+d", from.DM)
	}

	if from.Target.Number() != 0 {
		line += fmt.Sprintf(" = %d against %s, %s",
			from.Total(), from.Target, met(from.Succeeded))
	}

	fmt.Fprintln(t.out, line)

	return nil
}

func met(succeeded bool) string {
	if succeeded {
		return "made"
	}

	return "missed"
}

func (t *transcriber) Choice(from traveller.ChoiceEvent) error {
	fmt.Fprintf(t.out, "%3d. %s: %s chose %s from %s\n",
		from.Seq, from.Point, from.By, from.Chosen, strings.Join(from.Alternatives, ", "))

	return nil
}

func (t *transcriber) Outcome(from traveller.OutcomeEvent) error {
	line := fmt.Sprintf("%3d. %s", from.Seq, from.Description)

	if from.Because != 0 {
		line += fmt.Sprintf(" (from %d)", from.Because)
	}

	if len(from.Errata) > 0 {
		ids := make([]string, 0, len(from.Errata))
		for _, erratum := range from.Errata {
			ids = append(ids, erratum.String())
		}

		line += " [" + strings.Join(ids, " ") + "]"
	}

	fmt.Fprintln(t.out, line)

	return nil
}
