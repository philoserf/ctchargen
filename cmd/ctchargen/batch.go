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
			// Not a usage error: the flags were well formed, and what
			// failed was the search. The bound's own comment says
			// reaching it means something is wrong rather than unlucky,
			// and that is a run-time condition and not a misuse.
			return done, fmt.Errorf(
				"%w: --survivors drew %d seeds for %d living characters and found %d",
				errSearch, i, count, done.written,
			)
		}

		character, err := member(base, policy, i)
		if err != nil {
			return done, err
		}

		if traveller.Fatal(character.Departure) {
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
// One record is held at a time. #98 buffered the whole run to check the paths
// it would actually write, because --survivors breaks the arithmetic the check
// used to rest on - and buffered unconditionally, so a directory batch held
// about 650 MB at --count 50000 (#99). That is the property #44 had just
// removed from standard output.
//
// The collision check is what needed the paths, not the writing, so only the
// check pays. Without --survivors the paths are still base through
// base+count-1 and no character need exist to know them: plannedPaths takes no
// policy and no roller, so it cannot generate, and that is the whole of why
// the common case is free again. With the flag, survivingPaths walks once
// keeping seeds - eight bytes each, not thirteen kilobytes - and the write
// walks again, which generation being deterministic makes the same characters.
//
// A batch that fails partway leaves what it had written. That was true before
// #98 too: the guarantee here is that a run never silently replaces a file, not
// that a directory is written atomically.
func intoDirectory(base chargen.Inputs, policy chargen.Policy,
	count int, dir string, force, survivors bool,
) (tally, error) {
	err := os.MkdirAll(dir, recordDirMode)
	if err != nil {
		return tally{}, fmt.Errorf("creating %s: %w", dir, err)
	}

	if !force {
		err = noMemberExists(plannedOrSurviving(base, policy, count, survivors, dir))
		if err != nil {
			return tally{}, err
		}
	}

	return eachMember(base, policy, count, survivors, survivorAttempts,
		func(character *chargen.Character) error {
			encoded, err := render.JSON(character)
			if err != nil {
				return fmt.Errorf("seed %d: %w", character.Inputs.Seed, err)
			}

			where, err := openDestination(nil, memberPath(dir, character.Inputs.Seed), force)
			if err != nil {
				return err
			}

			return where.write(string(encoded))
		})
}

// plannedOrSurviving is the paths the run will write, by whichever route the
// flags allow.
func plannedOrSurviving(base chargen.Inputs, policy chargen.Policy,
	count int, survivors bool, dir string,
) []string {
	if survivors {
		return survivingPaths(base, policy, count, dir)
	}

	return plannedPaths(base.Seed, count, dir)
}

// plannedPaths is where members 0 through count-1 will be written.
//
// It takes a seed, a count and a directory, and no policy and no roller, so
// it cannot generate a character even by accident. That signature is the
// guarantee: without --survivors, checking for collisions costs nothing but
// arithmetic (#99).
func plannedPaths(base uint64, count int, dir string) []string {
	paths := make([]string, 0, count)
	for i := range count {
		paths = append(paths, memberPath(dir, memberSeed(base, i)))
	}

	return paths
}

// survivingPaths is where the members that live will be written.
//
// Under --survivors which seeds are written depends on which characters
// survive, so this walks once to find out. It keeps the paths and lets each
// character go, because holding the run is what #99 is about; the write walks
// again, and generation being deterministic it makes the same characters.
func survivingPaths(base chargen.Inputs, policy chargen.Policy, count int, dir string) []string {
	var paths []string

	// An error here is not reported: the walk that writes makes the same
	// calls in the same order and will meet the same one, where it can say
	// what was written before it.
	_, _ = eachMember(base, policy, count, true, survivorAttempts,
		func(character *chargen.Character) error {
			paths = append(paths, memberPath(dir, character.Inputs.Seed))

			return nil
		})

	return paths
}

// noMemberExists refuses the batch if any member's file is already there.
//
// It takes the paths the run will write, and how those are arrived at differs
// by flag: without --survivors they are arithmetic, and with it they are the
// seeds a walk found living. A check derived from the arithmetic alone would
// look at files the run will never write and miss the ones it will.
func noMemberExists(paths []string) error {
	for _, path := range paths {
		_, err := os.Stat(path)
		if err == nil {
			return fmt.Errorf("%w: %s %w; pass --force to replace it",
				errUsage, path, errWouldOverwrite)
		}
	}

	return nil
}
