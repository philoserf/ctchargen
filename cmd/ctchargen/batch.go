package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/philoserf/ctchargen/chargen"
	"github.com/philoserf/ctchargen/render"
)

// batch generates many characters from one base seed.
//
// It emits what it rolled. Death is an outcome and not an error, so a
// character killed by a survival throw is written like anyone else;
// filtering the dead is the caller's business, and no flag rerolls them
// silently.
func batch(args []string, out io.Writer) error {
	flags := flag.NewFlagSet("batch", flag.ContinueOnError)
	flags.SetOutput(io.Discard)

	var (
		count   = flags.Int("count", 0, "how many characters to generate")
		seed    = flags.Uint64(seedFlag, 0, "the base seed, drawn if absent; member i is generated from it plus i")
		auto    = flags.Bool("auto", false, "required: a batch has nobody to ask")
		service = flags.String("service", "",
			"attempt every enlistment in this service: "+serviceChoices())
		name   = flags.String("name", "", "the name to give every character")
		career = flags.String(careerFlag, chargen.CareerServe,
			"the career strategy: "+strategyChoices(careerFlag))
		skills = flags.String(skillsFlag, chargen.SkillsAdvanced,
			"the skills strategy: "+strategyChoices(skillsFlag))
		muster = flags.String(musterFlag, chargen.MusterCash,
			"the mustering out strategy: "+strategyChoices(musterFlag))
		output = flags.String("o", "",
			"a .jsonl file, or a directory to write one file per character; absent, JSONL to standard output")
		force = flags.Bool("force", false, "replace output files that already exist")
	)

	err := flags.Parse(args)

	switch {
	case errors.Is(err, flag.ErrHelp):
		return writeHelp(out, batchUsage, flags)
	case err != nil:
		return fmt.Errorf("%w: %w; run `ctchargen batch --help`", errUsage, err)
	}

	// batch takes no positional argument either, for the reason `new` does
	// not: a word the parser stops at leaves the flags after it unparsed.
	if flags.NArg() > 0 {
		return fmt.Errorf("%w: %s; it takes no arguments, and was given %q",
			errUsage, batchUsage, flags.Arg(0))
	}

	if !*auto {
		return fmt.Errorf("%w: batch requires --auto; there is nobody to ask", errUsage)
	}

	if *count < 1 {
		return fmt.Errorf("%w: --count must be at least 1", errUsage)
	}

	base, err := inputsFrom(*seed, *name, *service, *career, *skills, *muster, isSet(flags, seedFlag))
	if err != nil {
		return err
	}

	policy := chargen.Policy{Career: *career, Skills: *skills, Muster: *muster}

	invalid := policy.Validate()
	if invalid != nil {
		return fmt.Errorf("%w: %w", errUsage, invalid)
	}

	if namesDirectory(*output) {
		return intoDirectory(base, policy, *count, *output, *force)
	}

	return intoStream(out, base, policy, *count, *output, *force)
}

// memberSeed is the seed of batch member i.
//
// Members number from zero, so `batch --count 1 --seed N` produces exactly
// what `new --seed N` produces, and every seed a referee can type is
// reachable as a member. The addition wraps, because a base near the top of
// the range is a seed like any other and must not be an error.
//
// The 2^53 bound is not applied here. It covers only a seed the tool draws
// for itself; an explicit seed above it, and a derived seed that passes it
// because the base landed within --count of the bound, are written to the
// record as given rather than silently re-bounded.
func memberSeed(base uint64, i int) uint64 { return base + uint64(i) } //nolint:gosec // i is a count

// member generates one member of the batch, carrying its own derived seed -
// not the base - so that its record regenerates it.
func member(base chargen.Inputs, policy chargen.Policy, i int) (*chargen.Character, error) {
	inputs := base

	inputs.Seed = memberSeed(base.Seed, i)

	character, err := chargen.Generate(inputs, policy)
	if err != nil {
		return nil, fmt.Errorf("member %d: %w", i, err)
	}

	stamp(character)

	return character, nil
}

// intoStream writes the batch as JSONL, one record to the line.
func intoStream(out io.Writer, base chargen.Inputs, policy chargen.Policy,
	count int, path string, force bool,
) error {
	var lines strings.Builder

	for i := range count {
		character, err := member(base, policy, i)
		if err != nil {
			return err
		}

		encoded, err := render.JSONLine(character)
		if err != nil {
			return fmt.Errorf("member %d: %w", i, err)
		}

		lines.Write(encoded)
	}

	where, err := openDestination(out, path, force)
	if err != nil {
		return err
	}

	return where.write(lines.String())
}

// intoDirectory writes one file per character, named for its own seed.
//
// Every member's path is known before a character is generated, so the whole
// batch is checked for collisions before any of it is written. A batch that
// stopped halfway would leave a directory holding some of the run and no
// record of which files were new, and the obvious retry - --force - would
// then replace the very file the refusal was protecting.
func intoDirectory(base chargen.Inputs, policy chargen.Policy,
	count int, dir string, force bool,
) error {
	err := os.MkdirAll(dir, recordDirMode)
	if err != nil {
		return fmt.Errorf("creating %s: %w", dir, err)
	}

	if !force {
		err = noMemberExists(base.Seed, count, dir)
		if err != nil {
			return err
		}
	}

	for i := range count {
		character, err := member(base, policy, i)
		if err != nil {
			return err
		}

		encoded, err := render.JSON(character)
		if err != nil {
			return fmt.Errorf("member %d: %w", i, err)
		}

		where, err := openDestination(nil, memberPath(dir, character.Inputs.Seed), force)
		if err != nil {
			return err
		}

		err = where.write(string(encoded))
		if err != nil {
			return err
		}
	}

	return nil
}

// noMemberExists refuses the batch if any member's file is already there.
func noMemberExists(base uint64, count int, dir string) error {
	for i := range count {
		path := memberPath(dir, memberSeed(base, i))

		_, err := os.Stat(path)
		if err == nil {
			return fmt.Errorf("%w: %s %w; pass --force to replace it",
				errUsage, path, errWouldOverwrite)
		}
	}

	return nil
}
