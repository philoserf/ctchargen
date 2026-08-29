package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"slices"
	"strconv"
	"strings"

	"github.com/philoserf/ctchargen/chargen"
)

// errInputClosed is standard input ending mid-generation: without an
// answer the procedure cannot continue, and a partial record is not a
// record.
var errInputClosed = errors.New("standard input closed before generation finished")

// prompter is the interactive Decider: it walks the procedure step by
// step, putting every choice point to the player on the terminal.
// Prompts go to stderr so the JSON record on stdout stays clean.
type prompter struct {
	in  *bufio.Scanner
	out io.Writer
}

func newPrompter(in io.Reader, out io.Writer) *prompter {
	return &prompter{in: bufio.NewScanner(in), out: out}
}

// Decide presents the options and reads a pick: a number from the list,
// or the option text itself.
func (p *prompter) Decide(c chargen.Choice) (chargen.Decision, error) {
	fmt.Fprintf(p.out, "\n[%s] %s\n", c.Step, describeChoice(c))

	for i, option := range c.Options {
		fmt.Fprintf(p.out, "  %d) %s\n", i+1, option)
	}

	for {
		fmt.Fprint(p.out, "> ")

		if !p.in.Scan() {
			if err := p.in.Err(); err != nil {
				return chargen.Decision{}, fmt.Errorf("reading input: %w", err)
			}

			return chargen.Decision{}, errInputClosed
		}

		text := strings.TrimSpace(p.in.Text())

		if n, err := strconv.Atoi(text); err == nil && n >= 1 && n <= len(c.Options) {
			return chargen.Decision{Pick: c.Options[n-1], By: chargen.ByPlayer}, nil
		}

		if slices.Contains(c.Options, text) {
			return chargen.Decision{Pick: text, By: chargen.ByPlayer}, nil
		}

		fmt.Fprintf(p.out, "pick 1-%d (or type the option)\n", len(c.Options))
	}
}

// choicePrompts words each choice point for the player; the two weapon
// prompts are built in describeChoice because they name the category.
var choicePrompts = map[string]string{
	chargen.ChoiceService:       "which service does the character attempt to enlist in?",
	chargen.ChoiceSubmitToDraft: "rejected — submit to the draft? (declining ends generation as a civilian)",
	chargen.ChoiceCommission:    "attempt a commission this term?",
	chargen.ChoicePromotion:     "attempt a promotion this term?",
	chargen.ChoiceSkillTable:    "which skills table for this eligibility? (declared before the die is rolled)",
	chargen.ChoiceReenlist:      "intend to reenlist for another term?",
	chargen.ChoiceMusterTable:   "which mustering-out table for this roll?",
	chargen.ChoiceBenefitDM:     "apply the rank 5-6 +1 to this benefits roll?",
	chargen.ChoiceCashDM:        "apply the gambling +1 to this cash roll?",
	chargen.ChoiceTitle:         "assume the hereditary title?",
}

func describeChoice(c chargen.Choice) string {
	switch c.Label {
	case chargen.ChoiceWeapon:
		return "which " + c.Category + " weapon for the new expertise?"
	case chargen.ChoiceMusterWeapon:
		return "which " + c.Category + " weapon (or expertise in one already received)?"
	}

	if prompt, ok := choicePrompts[c.Label]; ok {
		return prompt
	}

	return c.Label
}
