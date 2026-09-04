package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"math/rand/v2"
	"strconv"
	"strings"

	"github.com/philoserf/ctchargen/chargen"
	"github.com/philoserf/ctchargen/render"
	"github.com/philoserf/ctchargen/traveller"
)

// maxDrawnSeed bounds a seed the tool draws for itself at 2^53 - 1, above
// which consecutive integers are no longer all exactly representable in an
// IEEE-754 double. A reader that parses JSON numbers as doubles would
// otherwise round it silently, and a rounded seed is a different character.
//
// The bound covers only a drawn seed. An explicit --seed is the operator's
// own number and is written to the record as given.
const maxDrawnSeed = 1<<53 - 1

func newCharacter(args []string, in io.Reader, out, asking io.Writer) error {
	flags := flag.NewFlagSet("new", flag.ContinueOnError)
	flags.SetOutput(io.Discard)

	var (
		seed    = flags.Uint64(seedFlag, 0, "the seed to generate from; one is drawn if absent")
		auto    = flags.Bool("auto", false, "decide by the policy at every choice, rather than asking")
		service = flags.String("service", "", "attempt enlistment in this service: "+serviceChoices())
		name    = flags.String("name", "", "the character's name")
		career  = flags.String(careerFlag, chargen.CareerServe,
			"the --auto career strategy: "+strategyChoices(careerFlag))
		skills = flags.String(skillsFlag, chargen.SkillsAdvanced,
			"the --auto skills strategy: "+strategyChoices(skillsFlag))
		muster = flags.String(musterFlag, chargen.MusterCash,
			"the --auto mustering out strategy: "+strategyChoices(musterFlag))
		sheet   = flags.Bool("sheet", false, "write the character sheet rather than JSON")
		history = flags.Bool("history", false, "write the generation record rather than JSON")
		output  = flags.String("o", "", "write to this file rather than to standard output")
		force   = flags.Bool("force", false, "replace the output file if it already exists")
		replay  = flags.String(answersFlag, "",
			"answers to replay before asking, as 1,2,1; what a stopped session prints")
	)

	err := flags.Parse(args)

	switch {
	case errors.Is(err, flag.ErrHelp):
		return writeHelp(out, newUsage, flags)
	case err != nil:
		return fmt.Errorf("%w: %w; run `ctchargen new --help`", errUsage, err)
	}

	// `new` takes no positional argument, and the one it used to ignore was
	// a typo that became an interactive session on a seed nobody chose. The
	// flag package stops at the first non-flag word, so `new foo --auto`
	// left --auto unparsed as well: refusing the word catches both.
	if flags.NArg() > 0 {
		return fmt.Errorf("%w: %s; it takes no arguments, and was given %q",
			errUsage, newUsage, flags.Arg(0))
	}

	inputs, err := inputsFrom(*seed, *name, *service, *career, *skills, *muster,
		isSet(flags, seedFlag))
	if err != nil {
		return err
	}

	answers, err := answersFrom(*replay)
	if err != nil {
		return err
	}

	return writeCharacter(inputs, answers, newRendering{
		auto: *auto, sheet: *sheet, history: *history, output: *output, force: *force,
	}, in, out, asking)
}

// newRendering is what the flags say to do with the character once it exists,
// gathered so that newCharacter reads as the four steps it takes rather than
// as eleven flags threaded through them.
type newRendering struct {
	auto, sheet, history, force bool
	output                      string
}

func writeCharacter(inputs chargen.Inputs, answers []int, how newRendering,
	in io.Reader, out, asking io.Writer,
) error {
	character, err := generate(inputs, how.auto, answers, in, asking)
	if err != nil {
		return err
	}

	stamp(character)

	text, err := rendered(character, how.sheet, how.history)
	if err != nil {
		return err
	}

	where, err := openDestination(out, how.output, how.force)
	if err != nil {
		return err
	}

	return where.write(text)
}

func isSet(flags *flag.FlagSet, name string) bool {
	found := false

	flags.Visit(func(f *flag.Flag) {
		if f.Name == name {
			found = true
		}
	})

	return found
}

func inputsFrom(seed uint64, name, service, career, skills, muster string, seedGiven bool) (
	chargen.Inputs, error,
) {
	in := chargen.Inputs{
		Seed: seed, Name: name, Career: career, Skills: skills, Muster: muster,
	}

	if !seedGiven {
		in.Seed = rand.Uint64N(maxDrawnSeed)
	}

	if service == "" {
		return in, nil
	}

	for _, known := range traveller.ServiceNames {
		if strings.EqualFold(known.String(), service) {
			in.Service, in.Forced = known, true

			return in, nil
		}
	}

	return in, fmt.Errorf("%w: no service is called %q", errUsage, service)
}

// generate runs the procedure under the auto policy, or asks the player.
//
// The strategies come off the inputs rather than off the flags a second
// time, so that what is checked and what is written into the record cannot
// name different strategies. They are validated in both modes: an
// interactive run records them too, and a record naming a strategy that is
// not a POLICY.md row is one its own schema refuses.
//
// Interactive mode watches the record as it is written, so the throws and
// their consequences reach the player between his questions; --auto passes
// no observer, because nobody is reading.
// answersFrom reads --answers, which is a list of the numbers a player typed.
//
// Empty is not an error and not an empty run: it is the ordinary case, a
// session nobody is resuming.
func answersFrom(list string) ([]int, error) {
	if list == "" {
		return nil, nil
	}

	fields := strings.Split(list, ",")

	answers := make([]int, 0, len(fields))

	for _, field := range fields {
		chosen, err := strconv.Atoi(strings.TrimSpace(field))
		if err != nil {
			return nil, fmt.Errorf("%w: --%s takes numbers separated by commas, and %q is not one",
				errUsage, answersFlag, field)
		}

		answers = append(answers, chosen)
	}

	return answers, nil
}

func generate(inputs chargen.Inputs, auto bool, answers []int, in io.Reader, asking io.Writer) (
	*chargen.Character, error,
) {
	policy := chargen.Policy{
		Career: inputs.Career, Skills: inputs.Skills, Muster: inputs.Muster,
	}

	invalid := policy.Validate()
	if invalid != nil {
		return nil, fmt.Errorf("%w: %w", errUsage, invalid)
	}

	if auto {
		character, err := chargen.Generate(inputs, policy)
		if err != nil {
			return nil, fmt.Errorf("generating: %w", err)
		}

		return character, nil
	}

	asked := newPlayer(in, asking, inputs.Seed, answers)

	character, err := chargen.Generate(inputs, asked,
		chargen.WithObserver(asked.watch), chargen.WithAnswerer(traveller.ByPlayer))
	if err != nil {
		return nil, fmt.Errorf("generating: %w", err)
	}

	return character, nil
}

// rendered is the character in whichever of the three shapes was asked for.
func rendered(character *chargen.Character, sheet, history bool) (string, error) {
	switch {
	case sheet:
		text, err := render.Sheet(character)

		return text, wrapRender(err)
	case history:
		text, err := render.Transcript(character)

		return text, wrapRender(err)
	default:
		encoded, err := render.JSON(character)
		if err != nil {
			return "", fmt.Errorf("rendering: %w", err)
		}

		return string(encoded), nil
	}
}

// wrapRender gives a rendering failure the context of the command that asked
// for it.
func wrapRender(err error) error {
	if err == nil {
		return nil
	}

	return fmt.Errorf("rendering: %w", err)
}
