// Command ctchargen generates Classic Traveller characters (Books 1-3,
// © 1977 text). See docs/PRD.md for the v1 contract.
//
// Implemented subcommands: new (auto mode), render, version.
package main

import (
	"crypto/rand"
	"encoding/binary"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"runtime/debug"

	"github.com/philoserf/ctchargen/chargen"
	"github.com/philoserf/ctchargen/dice"
	"github.com/philoserf/ctchargen/render"
)

// Exit codes: 0 success, 1 operational error, 2 usage error (the flag
// package's own convention).
const (
	exitOK    = 0
	exitError = 1
	exitUsage = 2
)

const usage = `usage:
  ctchargen new [--seed N] [--auto] [--name X] [-o file] [--force]
  ctchargen render [--history] character.json
  ctchargen replay [--ignore-provenance] character.json
  ctchargen batch --count 20 --auto [-o dir|file.jsonl]
  ctchargen version
`

func main() {
	os.Exit(run(os.Args[1:], randomSeed, os.Stdout, os.Stderr))
}

// randomSeed draws a seed from the OS entropy source: the one deliberate
// exception to the seeded-stream rule, which is engine-scoped. The chosen
// seed is recorded in the record's rng provenance, so replay stays exact.
func randomSeed() (uint64, error) {
	var buf [8]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return 0, fmt.Errorf("drawing random seed: %w", err)
	}

	return binary.LittleEndian.Uint64(buf[:]), nil
}

func run(args []string, seedSource func() (uint64, error), stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprint(stderr, usage)

		return exitUsage
	}

	switch args[0] {
	case "new":
		return runNew(args[1:], seedSource, stdout, stderr)
	case "render":
		return runRender(args[1:], stdout, stderr)
	case "version":
		return runVersion(stdout)
	case "batch", "replay":
		fmt.Fprintf(stderr, "ctchargen %s: not yet implemented (see docs/PRD.md)\n", args[0])

		return exitError
	default:
		fmt.Fprintf(stderr, "ctchargen: unknown command %q\n", args[0])
		fmt.Fprint(stderr, usage)

		return exitUsage
	}
}

func runNew(args []string, seedSource func() (uint64, error), stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("new", flag.ContinueOnError)
	fs.SetOutput(stderr)
	seed := fs.Uint64("seed", 0, "RNG seed (default: drawn from the OS)")
	auto := fs.Bool("auto", false, "apply the fixed default policy (POLICY.md) to every choice")
	name := fs.String("name", "", "character name (blank by default; the book's naming section is advice, not a table)")
	outPath := fs.String("o", "", "write the JSON record to this file instead of stdout")
	force := fs.Bool("force", false, "overwrite an existing output file")

	if err := fs.Parse(args); err != nil {
		return exitUsage
	}

	if fs.NArg() != 0 {
		fmt.Fprintf(stderr, "ctchargen new: unexpected argument %q (flags precede any filename)\n", fs.Arg(0))

		return exitUsage
	}

	if !*auto {
		fmt.Fprintln(stderr, "ctchargen new: interactive mode is not yet implemented; use --auto")

		return exitError
	}

	if err := resolveSeed(fs, seed, seedSource); err != nil {
		fmt.Fprintf(stderr, "ctchargen new: %v\n", err)

		return exitError
	}

	char, err := chargen.Generate(chargen.Config{Seed: *seed, Name: *name, Auto: true}, chargen.AutoPolicy{})
	if err != nil {
		fmt.Fprintf(stderr, "ctchargen new: %v\n", err)

		return exitError
	}

	if err := emitRecord(char, *outPath, *force, stdout); err != nil {
		fmt.Fprintf(stderr, "ctchargen new: %v\n", err)

		return exitError
	}

	return exitOK
}

// resolveSeed draws a random seed only when --seed was not given, so 0
// stays a usable explicit seed.
func resolveSeed(fs *flag.FlagSet, seed *uint64, seedSource func() (uint64, error)) error {
	given := false

	fs.Visit(func(f *flag.Flag) {
		if f.Name == "seed" {
			given = true
		}
	})

	if given {
		return nil
	}

	drawn, err := seedSource()
	if err != nil {
		return err
	}

	*seed = drawn

	return nil
}

func emitRecord(char *chargen.Character, outPath string, force bool, stdout io.Writer) error {
	out, err := char.MarshalRecord()
	if err != nil {
		return fmt.Errorf("rendering record: %w", err)
	}

	if outPath == "" {
		if _, err := stdout.Write(out); err != nil {
			return fmt.Errorf("writing record: %w", err)
		}

		return nil
	}

	return writeFile(outPath, out, force)
}

// errExists refuses to overwrite an existing file without --force
// (docs/PRD.md, CLI sketch).
var errExists = errors.New("use --force to overwrite")

func writeFile(path string, data []byte, force bool) error {
	if !force {
		if _, err := os.Stat(path); err == nil {
			return fmt.Errorf("%s exists; %w", path, errExists)
		} else if !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("checking %s: %w", path, err)
		}
	}

	// 0644: a character record is the user's shareable output, not a secret.
	if err := os.WriteFile(path, data, 0o644); err != nil { // #nosec G306
		return fmt.Errorf("writing %s: %w", path, err)
	}

	return nil
}

func runRender(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("render", flag.ContinueOnError)
	fs.SetOutput(stderr)
	history := fs.Bool("history", false, "render the generation record transcript instead of the sheet")

	if err := fs.Parse(args); err != nil {
		return exitUsage
	}

	if fs.NArg() != 1 {
		fmt.Fprintln(stderr, "ctchargen render: want exactly one character.json (flags precede the filename)")

		return exitUsage
	}

	data, err := os.ReadFile(fs.Arg(0))
	if err != nil {
		fmt.Fprintf(stderr, "ctchargen render: %v\n", err)

		return exitError
	}

	char, err := chargen.UnmarshalRecord(data)
	if err != nil {
		fmt.Fprintf(stderr, "ctchargen render: %v\n", err)

		return exitError
	}

	text := render.Sheet(char)
	if *history {
		text = render.History(char)
	}

	fmt.Fprint(stdout, text)

	return exitOK
}

func runVersion(stdout io.Writer) int {
	build := "unknown"
	if info, ok := debug.ReadBuildInfo(); ok {
		build = info.Main.Version
	}

	fmt.Fprintf(stdout, "ctchargen %s\n", build)
	fmt.Fprintf(stdout, "schema_version %s\n", chargen.SchemaVersion)
	fmt.Fprintf(stdout, "engine_version %s\n", chargen.EngineVersion)
	fmt.Fprintf(stdout, "policy_version %s\n", chargen.PolicyVersion)
	fmt.Fprintf(stdout, "ruleset %s\n", chargen.Ruleset)
	fmt.Fprintf(stdout, "rng %s\n", dice.Algorithm)

	return exitOK
}
