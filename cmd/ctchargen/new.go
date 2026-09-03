package main

import (
	"flag"
	"fmt"
	"io"
	"math/rand/v2"
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

func newCharacter(args []string, out io.Writer) error {
	flags := flag.NewFlagSet("new", flag.ContinueOnError)
	flags.SetOutput(io.Discard)

	var (
		seed    = flags.Uint64("seed", 0, "the seed to generate from; one is drawn if absent")
		auto    = flags.Bool("auto", false, "let the policy decide, rather than the player")
		service = flags.String("service", "", "force the enlistment attempt into this service")
		name    = flags.String("name", "", "the character's name")
		career  = flags.String("career", chargen.CareerServe, "the --auto career strategy")
		skills  = flags.String("skills", chargen.SkillsAdvanced, "the --auto skills strategy")
		muster  = flags.String("muster", chargen.MusterCash, "the --auto mustering out strategy")
		sheet   = flags.Bool("sheet", false, "write the character sheet rather than JSON")
		history = flags.Bool("history", false, "write the generation record rather than JSON")
		output  = flags.String("o", "", "write to this file rather than to standard output")
		force   = flags.Bool("force", false, "replace the output file if it already exists")
	)

	err := flags.Parse(args)
	if err != nil {
		return fmt.Errorf("%w: %w", errUsage, err)
	}

	if !*auto {
		return fmt.Errorf(
			"%w: interactive mode arrives at milestone 4; pass --auto to let the policy decide",
			errUsage,
		)
	}

	inputs, err := inputsFrom(*seed, *name, *service, *career, *skills, *muster,
		isSet(flags, "seed"))
	if err != nil {
		return err
	}

	policy := chargen.Policy{Career: *career, Skills: *skills, Muster: *muster}

	invalid := policy.Validate()
	if invalid != nil {
		return fmt.Errorf("%w: %w", errUsage, invalid)
	}

	// Defensive, and unreachable from here today: Generate fails when an
	// embedded table will not lift or when a Decider returns an error, and
	// the auto policy never returns one. It is not dead - an interactive
	// decider, arriving at milestone 4, can fail on any answer - so the
	// branch stays and cmd carries one uncovered statement for it.
	character, err := chargen.Generate(inputs, policy)
	if err != nil {
		return fmt.Errorf("generating: %w", err)
	}

	stamp(character)

	text, err := rendered(character, *sheet, *history)
	if err != nil {
		return err
	}

	where, err := openDestination(out, *output, *force)
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
