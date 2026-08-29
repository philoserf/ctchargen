// Package render turns the canonical JSON character record into Markdown:
// the character sheet in the style the book itself summarizes a character
// (the UPP line plus skills, as in the Jamison example, pp. 23-25), and
// the generation record as a readable service-record transcript (FR10).
package render

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/philoserf/ctchargen/chargen"
)

// Sheet renders the character sheet.
func Sheet(c *chargen.Character) string {
	var b strings.Builder

	name := c.Name
	if name == "" {
		name = "Unnamed"
	}

	fmt.Fprintf(&b, "# %s\n\n", name)
	fmt.Fprintf(&b, "%s\n", statusLine(c))

	b.WriteString("\n## Skills\n\n")

	if len(c.Skills) == 0 {
		b.WriteString("None.\n")
	} else {
		parts := make([]string, 0, len(c.Skills))
		for _, s := range c.Skills {
			parts = append(parts, fmt.Sprintf("%s-%d", s.Name, s.Level))
		}

		fmt.Fprintf(&b, "%s\n", strings.Join(parts, ", "))
	}

	return b.String()
}

func statusLine(c *chargen.Character) string {
	if c.Service == "" {
		return fmt.Sprintf("Civilian (declined the draft), age %d. UPP %s.", c.Age, c.UPP)
	}

	who := c.Service
	if c.RankTitle != "" {
		who += " " + c.RankTitle
	}

	if c.Drafted {
		who += ", drafted"
	}

	if c.Death != nil {
		return fmt.Sprintf("%s — died in service, term %d (%s), age %d. UPP %s.",
			who, c.Death.Term, c.Death.Cause, c.Age, c.UPP)
	}

	return fmt.Sprintf("%s, %d %s, age %d. UPP %s.",
		who, c.Terms, plural(c.Terms, "term"), c.Age, c.UPP)
}

// History renders the generation record: every throw, choice, and outcome
// in chronological order.
func History(c *chargen.Character) string {
	var b strings.Builder

	name := c.Name
	if name == "" {
		name = "Unnamed"
	}

	fmt.Fprintf(&b, "# Generation record: %s\n\n", name)
	fmt.Fprintf(&b, "Seed %d (%s), engine %s, policy %s.\n",
		c.RNG.Seed, c.RNG.Algorithm, c.EngineVersion, c.PolicyVersion)

	step := ""

	for _, ev := range c.Events {
		if ev.Step != step {
			step = ev.Step

			fmt.Fprintf(&b, "\n## %s\n\n", step)
		}

		fmt.Fprintf(&b, "- (%d) %s\n", ev.Seq, describeEvent(ev))
	}

	return b.String()
}

func describeEvent(ev chargen.Event) string {
	switch ev.Kind {
	case "step":
		return ev.Text
	case "throw":
		return describeThrow(ev)
	case "choice":
		return fmt.Sprintf("%s: chose %s (by %s; options: %s)",
			ev.Label, ev.Picked, ev.By, strings.Join(ev.Options, ", "))
	default:
		text := ev.Text
		if ev.Ref != 0 {
			text = fmt.Sprintf("%s [from %d]", text, ev.Ref)
		}

		return "→ " + text
	}
}

func describeThrow(ev chargen.Event) string {
	dice := make([]string, 0, len(ev.Dice))
	for _, d := range ev.Dice {
		dice = append(dice, strconv.Itoa(d))
	}

	var s strings.Builder
	fmt.Fprintf(&s, "%s: threw %s", ev.Label, strings.Join(dice, "+"))

	for _, dm := range ev.DMs {
		fmt.Fprintf(&s, ", DM %+d (%s)", dm.Value, dm.Source)
	}

	fmt.Fprintf(&s, " = %d", ev.Total)

	if ev.Target != "" {
		verdict := "failure"
		if ev.Success != nil && *ev.Success {
			verdict = "success"
		}

		fmt.Fprintf(&s, " against %s: %s", ev.Target, verdict)
	}

	return s.String()
}

func plural(n int, word string) string {
	if n == 1 {
		return word
	}

	return word + "s"
}
