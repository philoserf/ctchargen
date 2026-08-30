package main

import (
	"testing"

	"github.com/philoserf/ctchargen/chargen"
)

// describeChoice falls back to returning the bare label, so a choice
// point added to the engine without a line in choicePrompts shows the
// player a raw identifier — "muster-table" instead of a question —
// while every test stays green. chargen.ChoiceLabels is the registry
// that makes the omission fail here instead.
func TestEveryChoicePointIsWorded(t *testing.T) {
	for _, label := range chargen.ChoiceLabels() {
		t.Run(label, func(t *testing.T) {
			// Category is set for the two weapon prompts, which name it;
			// the rest ignore it.
			got := describeChoice(chargen.Choice{Step: "test", Label: label, Category: "blade"})

			if got == label {
				t.Errorf("choice %s has no wording: the prompt is the bare label", label)
			}

			if got == "" {
				t.Errorf("choice %s has an empty prompt", label)
			}
		})
	}
}
