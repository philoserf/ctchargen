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
	"github.com/philoserf/ctchargen/traveller"
)

// batch generates many characters from one base seed.
//
// It emits what it rolled. Death is an outcome and not an error, so a
// character killed by a survival throw is written like anyone else and no
// flag rerolls him.
// --survivors does not reroll either: it passes over a dead character and
// goes on to the next seed, so every character written is still the character
// that seed makes. What changes is which seeds appear (#33).
//
// It always says what it did. A run that wrote seventy-four corpses and
// printed nothing was the trap the report named.
func batch(args []string, out, asking io.Writer) error {
	flags := flag.NewFlagSet("batch", flag.ContinueOnError)
	flags.SetOutput(io.Discard)

	var (
		count   = flags.Int("count", 0, "how many characters to generate")
		seed    = flags.Uint64(seedFlag, 0, "the base seed, drawn if absent; member i is generated from it plus i")
		auto    = flags.Bool("auto", false, "required: a batch has nobody to ask")
		service = flags.String("service", "",
			"attempt every enlistment in this service: "+serviceChoices())
		name   = flags.String("name", "", "the name to give every character")
		career = flags.String(careerFlag, chargen.CareerServe.String(),
			"the career `strategy`: "+chargen.CareerChoices())
		skills = flags.String(skillsFlag, chargen.SkillsAdvanced.String(),
			"the skills `strategy`: "+chargen.SkillsChoices())
		muster = flags.String(musterFlag, chargen.MusterCash.String(),
			"the mustering out `strategy`: "+chargen.MusterChoices())
		output = flags.String("o", "",
			"a .jsonl file, or a directory to write one file per character; absent, JSONL to standard output")
		force     = flags.Bool("force", false, "replace output files that already exist")
		survivors = flags.Bool("survivors", false,
			"pass over characters who died, and go on until --count of them are living")
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

	policy := chargen.Policy{Career: base.Career, Skills: base.Skills, Muster: base.Muster}

	return write(out, asking, base, policy, *count, *output, *force, *survivors)
}

// write sends the batch where it was asked for, and says what it did either
// way. A directory takes one file per character; anything else is JSONL.
func write(out, asking io.Writer, base chargen.Inputs, policy chargen.Policy,
	count int, path string, force, survivors bool,
) error {
	var (
		done tally
		err  error
	)

	if namesDirectory(path) {
		done, err = intoDirectory(base, policy, count, path, force, survivors)
	} else {
		done, err = intoStream(out, base, policy, count, path, force, survivors)
	}

	if err != nil {
		return err
	}

	return done.report(asking)
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
//
// A failure names the seed and not the position i, for the reason writingTo
// does: under --survivors the two differ, i is an index into seeds drawn
// rather than into records written, and the seed is the one a referee can
// hand back to `new`.
func member(base chargen.Inputs, policy chargen.Policy, i int) (*chargen.Character, error) {
	inputs := base

	inputs.Seed = memberSeed(base.Seed, i)

	character, err := chargen.Generate(inputs, policy)
	if err != nil {
		return nil, fmt.Errorf("seed %d: %w", inputs.Seed, err)
	}

	stamp(character)

	return character, nil
}

// tally is what a batch did, for the line it prints when it is done.
type tally struct {
	written int
	died    int
	passed  int
}

// report writes the closing line.
//
// It goes to the channel the tool talks to the operator on and never to the
// data channel: a summary on standard output would be a line of JSONL that is
// not JSON. After #44 it also arrives after records the reader has already
// seen, which is why it counts rather than warns.
func (t tally) report(asking io.Writer) error {
	line := fmt.Sprintf("%d written", t.written)

	switch {
	case t.passed > 0:
		line += fmt.Sprintf(", %d passed over for dying", t.passed)
	case t.died > 0:
		line += fmt.Sprintf(", %d died", t.died)
	default:
		line += ", none died"
	}

	_, err := fmt.Fprintln(asking, line)
	if err != nil {
		return fmt.Errorf("reporting the batch: %w", err)
	}

	return nil
}

// survivorAttempts bounds how many seeds --survivors draws per character
// asked for. It is a parameter of eachMember rather than read from there, so
// that a test can reach the bound without needing a hundred deaths per
// character to do it.
//
// It bounds a loop that would otherwise have none, which is the shape of
// defect #54 names elsewhere. The referee who reported #33 measured 74 dead
// in 100 under the default strategy - about four seeds a survivor - so a
// hundred is a margin of twenty-five over the worst case anyone has seen, and
// reaching it means something is wrong rather than unlucky.
const survivorAttempts = 100

// fatalDeparture folds a departure down to whether it killed him.
//
// It folds rather than type-switching because that is what the sum is sealed
// for: a third way to die adds a method to DepartureCases and this file stops
// compiling, where a type switch with a default would quietly go on calling
// the new death survival and --survivors would write a corpse.
type fatalDeparture struct{ fatal bool }

func (*fatalDeparture) Discharged() error { return nil }
func (*fatalDeparture) ForcedOut() error  { return nil }
func (*fatalDeparture) Retired() error    { return nil }

func (f *fatalDeparture) KilledBySurvivalThrow() error {
	f.fatal = true

	return nil
}

func (f *fatalDeparture) KilledByMedicalCrisis(traveller.Characteristic) error {
	f.fatal = true

	return nil
}

// died reports a character killed during generation. A civilian has no
// departure at all, and did not die of it.
func died(character *chargen.Character) bool {
	if character.Departure == nil {
		return false
	}

	var fatal fatalDeparture

	// The cases above return nothing but nil, so there is no error to carry.
	_ = character.Departure.Fold(&fatal)

	return fatal.fatal
}

// eachMember walks the batch, handing each member that is to be written to
// take, and reports what it did.
//
// Without --survivors every member is written, dead or alive. With it a dead
// character is passed over and the next seed tried - so a written member is
// still exactly the character his own seed makes, and `new --seed <that>`
// brings him back. What changes is which seeds appear, not what any of them
// means (#33).
func eachMember(base chargen.Inputs, policy chargen.Policy, count int, survivors bool,
	attempts int, take func(*chargen.Character) error,
) (tally, error) {
	var done tally

	for i := 0; done.written < count; i++ {
		if survivors && i >= count*attempts {
			return done, fmt.Errorf(
				"%w: --survivors drew %d seeds for %d living characters and found %d",
				errUsage, i, count, done.written,
			)
		}

		character, err := member(base, policy, i)
		if err != nil {
			return done, err
		}

		if died(character) {
			if survivors {
				done.passed++

				continue
			}

			done.died++
		}

		err = take(character)
		if err != nil {
			return done, err
		}

		done.written++
	}

	return done, nil
}

// intoStream writes the batch as JSONL, one record to the line.
//
// To standard output it streams, writing each record as it is generated.
// JSONL exists to be piped, and `batch --count 50000 --auto | jq` used to
// produce nothing until the run finished and to hold the whole run in memory
// while it did (#44). The atomicity that justifies buffering is about not
// leaving a file half-written and not clobbering one on a retry, and there is
// no file to clobber on standard output (#64).
func intoStream(out io.Writer, base chargen.Inputs, policy chargen.Policy,
	count int, path string, force, survivors bool,
) (tally, error) {
	if path == "" {
		return eachMember(base, policy, count, survivors, survivorAttempts, writingTo(out))
	}

	return buffered(base, policy, count, path, force, survivors)
}

// writingTo is the take a streamed run uses: encode the member and write it.
//
// It names the member by its own seed rather than by its position, because
// under --survivors those differ and the seed is the one that brings him back.
func writingTo(out io.Writer) func(*chargen.Character) error {
	return func(character *chargen.Character) error {
		encoded, err := render.JSONLine(character)
		if err != nil {
			return fmt.Errorf("seed %d: %w", character.Inputs.Seed, err)
		}

		_, err = out.Write(encoded)
		if err != nil {
			return fmt.Errorf("seed %d: %w", character.Inputs.Seed, err)
		}

		return nil
	}
}

// buffered generates the whole batch before opening the file, so a run that
// fails partway opens nothing at all.
//
// It is the same walk: buffering is streaming to somewhere that cannot be
// half-read. What differs is only where it writes and when the file is
// opened, which is the whole of the distinction #64 drew.
func buffered(base chargen.Inputs, policy chargen.Policy,
	count int, path string, force, survivors bool,
) (tally, error) {
	var lines strings.Builder

	done, err := eachMember(base, policy, count, survivors, survivorAttempts, writingTo(&lines))
	if err != nil {
		return done, err
	}

	// nil, because path is never empty here: intoStream sends an empty one to
	// the stream above, so openDestination always opens a file.
	where, err := openDestination(nil, path, force)
	if err != nil {
		return done, err
	}

	return done, where.write(lines.String())
}

// intoDirectory writes one file per character, named for its own seed.
//
// The whole batch is generated and rendered before any of it is written, so
// the collision check covers the paths the run actually produces and an
// encoding that fails does so before the first file is opened. A batch that stopped
// halfway would leave a directory holding some of the run and no record of
// which files were new, and the obvious retry - --force - would then replace
// the very file the refusal was protecting.
//
// It used to check the paths before generating, because members were seeds
// base through base+count-1 and every path was known in advance. Under
// --survivors they are not: which seeds are written depends on which
// characters live (#33). Checking what was actually made keeps the guarantee
// and stops it resting on an arithmetic that no longer holds.
func intoDirectory(base chargen.Inputs, policy chargen.Policy,
	count int, dir string, force, survivors bool,
) (tally, error) {
	err := os.MkdirAll(dir, recordDirMode)
	if err != nil {
		return tally{}, fmt.Errorf("creating %s: %w", dir, err)
	}

	var roster []memberFile

	done, err := eachMember(base, policy, count, survivors, survivorAttempts,
		func(character *chargen.Character) error {
			encoded, renderErr := render.JSON(character)
			if renderErr != nil {
				return fmt.Errorf("seed %d: %w", character.Inputs.Seed, renderErr)
			}

			roster = append(roster, memberFile{
				path:    memberPath(dir, character.Inputs.Seed),
				encoded: string(encoded),
			})

			return nil
		})
	if err != nil {
		return done, err
	}

	if !force {
		err = noMemberExists(roster)
		if err != nil {
			return done, err
		}
	}

	for _, file := range roster {
		where, err := openDestination(nil, file.path, force)
		if err != nil {
			return done, err
		}

		err = where.write(file.encoded)
		if err != nil {
			return done, err
		}
	}

	return done, nil
}

// memberFile is one member's file, named and rendered before anything is
// opened.
//
// The roster is held as text rather than as characters, so a run releases
// each character as it goes rather than keeping the whole batch alive, and an
// encoding that fails does so before the first file is opened rather than
// halfway through the directory.
type memberFile struct {
	path    string
	encoded string
}

// noMemberExists refuses the batch if any member's file is already there.
//
// It takes the roster rather than a base and a count: under --survivors the
// members written are not the seeds base through base+count-1, and a check
// derived from that arithmetic would both look at files the run will never
// write and miss the ones it will.
func noMemberExists(roster []memberFile) error {
	for _, file := range roster {
		_, err := os.Stat(file.path)
		if err == nil {
			return fmt.Errorf("%w: %s %w; pass --force to replace it",
				errUsage, file.path, errWouldOverwrite)
		}
	}

	return nil
}
