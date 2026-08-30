// Command ctchargen generates Classic Traveller characters (Books 1-3,
// © 1977 text). See docs/PRD.md for the v1 contract.
//
// Implemented subcommands: new, batch, render, replay, version.
package main

import (
	"crypto/rand"
	"encoding/binary"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime/debug"
	"strings"

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
  ctchargen new [--seed N] [--auto] [--service navy] [--name X] [-o file] [--force]
                (without --auto the player answers each choice; --auto applies docs/POLICY.md)
  ctchargen batch --count 20 --auto [--seed N] [--service navy] [-o dir|file.jsonl] [--force]
  ctchargen render [--history] character.json
  ctchargen replay [--ignore-provenance] character.json
  ctchargen version
`

func main() {
	os.Exit(run(os.Args[1:], randomSeed, os.Stdin, os.Stdout, os.Stderr))
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

func run(args []string, seedSource func() (uint64, error), stdin io.Reader, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprint(stderr, usage)

		return exitUsage
	}

	switch args[0] {
	case "-h", "--help":
		// An answered request, not a usage error: usage is this run's
		// output, so it goes to stdout and exits clean. Not a `help`
		// subcommand — docs/PRD.md's CLI sketch is the v1 contract and
		// lists the commands; these two are the flag package's convention.
		fmt.Fprint(stdout, usage)

		return exitOK
	case "new":
		return runNew(args[1:], seedSource, stdin, stdout, stderr)
	case "batch":
		return runBatch(args[1:], seedSource, stdout, stderr)
	case "render":
		return runRender(args[1:], stdout, stderr)
	case "replay":
		return runReplay(args[1:], stdout, stderr)
	case "version":
		return runVersion(stdout)
	default:
		fmt.Fprintf(stderr, "ctchargen: unknown command %q\n", args[0])
		fmt.Fprint(stderr, usage)

		return exitUsage
	}
}

// parseExit maps a flag.Parse failure onto an exit code. flag.ErrHelp is
// the sentinel for a -h/--help the flag package has already answered by
// printing the flag list: a handled request, not a usage error, and 0 is
// what flag.ExitOnError would have exited with.
func parseExit(err error) int {
	if errors.Is(err, flag.ErrHelp) {
		return exitOK
	}

	return exitUsage
}

func runNew(args []string, seedSource func() (uint64, error), stdin io.Reader, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("new", flag.ContinueOnError)
	fs.SetOutput(stderr)
	seed := fs.Uint64("seed", 0, "RNG seed (default: drawn from the OS)")
	auto := fs.Bool("auto", false, "apply the fixed default policy (docs/POLICY.md) to every choice")
	svc := fs.String("service", "", "force the enlistment attempt only; a failed throw still goes to the draft (p. 5)")
	name := fs.String("name", "", "character name (blank by default; the book's naming section is advice, not a table)")
	outPath := fs.String("o", "", "write the JSON record to this file instead of stdout")
	force := fs.Bool("force", false, "overwrite an existing output file")

	if err := fs.Parse(args); err != nil {
		return parseExit(err)
	}

	if fs.NArg() != 0 {
		fmt.Fprintf(stderr, "ctchargen new: unexpected argument %q (flags precede any filename)\n", fs.Arg(0))

		return exitUsage
	}

	if err := resolveSeed(fs, seed, seedSource); err != nil {
		fmt.Fprintf(stderr, "ctchargen new: %v\n", err)

		return exitError
	}

	var decider chargen.Decider = chargen.AutoPolicy{}
	if !*auto {
		decider = newPrompter(stdin, stderr)
	}

	cfg := chargen.Config{Seed: *seed, Name: *name, Service: *svc, Auto: *auto}

	char, err := chargen.Generate(cfg, decider)
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

// runBatch generates count characters, each member's seed derived from
// the base seed plus its index and recorded in its record (docs/PRD.md,
// CLI sketch). Output is JSONL to stdout or a file, or one JSON file per
// character when -o names a directory.
func runBatch(args []string, seedSource func() (uint64, error), stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("batch", flag.ContinueOnError)
	fs.SetOutput(stderr)
	count := fs.Int("count", 0, "number of characters to generate")
	seed := fs.Uint64("seed", 0, "base RNG seed (default: drawn from the OS); member i uses seed+i")
	auto := fs.Bool("auto", false, "required: batch applies the fixed default policy (docs/POLICY.md)")
	svc := fs.String("service", "", "force each member's enlistment attempt only (p. 5)")
	outPath := fs.String("o", "", "JSONL file, or an existing directory for one file per character")
	force := fs.Bool("force", false, "overwrite existing output files")

	if err := fs.Parse(args); err != nil {
		return parseExit(err)
	}

	if fs.NArg() != 0 || *count < 1 || !*auto {
		fmt.Fprintln(stderr, "ctchargen batch: requires --count N (≥1) and --auto, with flags before any filename")

		return exitUsage
	}

	if err := resolveSeed(fs, seed, seedSource); err != nil {
		fmt.Fprintf(stderr, "ctchargen batch: %v\n", err)

		return exitError
	}

	chars := make([]*chargen.Character, 0, *count)

	for i := range *count {
		memberSeed := *seed + uint64(i) // #nosec G115 -- count is small; uint64 wraparound would be harmless and recorded
		cfg := chargen.Config{Seed: memberSeed, Service: *svc, Auto: true}

		char, err := chargen.Generate(cfg, chargen.AutoPolicy{})
		if err != nil {
			fmt.Fprintf(stderr, "ctchargen batch: member %d: %v\n", i, err)

			return exitError
		}

		chars = append(chars, char)
	}

	if err := emitBatch(chars, *outPath, *force, stdout); err != nil {
		fmt.Fprintf(stderr, "ctchargen batch: %v\n", err)

		return exitError
	}

	return exitOK
}

func emitBatch(chars []*chargen.Character, outPath string, force bool, stdout io.Writer) error {
	if outPath != "" {
		if info, err := os.Stat(outPath); err == nil && info.IsDir() {
			return writeBatchDir(chars, outPath, force)
		}
	}

	lines, err := batchJSONL(chars)
	if err != nil {
		return err
	}

	if outPath == "" {
		if _, err := stdout.Write(lines); err != nil {
			return fmt.Errorf("writing batch: %w", err)
		}

		return nil
	}

	return writeFile(outPath, lines, force)
}

// batchJSONL renders one compact JSON record per line.
func batchJSONL(chars []*chargen.Character) ([]byte, error) {
	var out []byte

	for _, char := range chars {
		line, err := json.Marshal(char)
		if err != nil {
			return nil, fmt.Errorf("marshaling batch record: %w", err)
		}

		out = append(out, line...)
		out = append(out, '\n')
	}

	return out, nil
}

// writeBatchDir writes one file per character, all or nothing: every
// record is marshaled and every path checked before the first file is
// created, so a collision midway through does not leave a directory
// holding half of one run and half of another. The check races against a
// concurrent writer, as any such check does; it is there to make the
// ordinary rerun-without---force case clean, not to be atomic.
func writeBatchDir(chars []*chargen.Character, dir string, force bool) error {
	paths := make([]string, len(chars))
	records := make([][]byte, len(chars))

	for i, char := range chars {
		record, err := char.MarshalRecord()
		if err != nil {
			return fmt.Errorf("member %d: %w", i, err)
		}

		paths[i] = filepath.Join(dir, fmt.Sprintf("character-%04d.json", i))
		records[i] = record
	}

	if !force {
		if err := checkNoneExist(paths); err != nil {
			return err
		}
	}

	for i, path := range paths {
		if err := writeFile(path, records[i], force); err != nil {
			return err
		}
	}

	return nil
}

// checkNoneExist names every colliding file at once, rather than stopping
// at the first and leaving the rest to be discovered one rerun at a time.
func checkNoneExist(paths []string) error {
	var collisions []string

	for _, path := range paths {
		switch _, err := os.Stat(path); {
		case err == nil:
			collisions = append(collisions, path)
		case !errors.Is(err, os.ErrNotExist):
			return fmt.Errorf("checking %s: %w", path, err)
		}
	}

	if len(collisions) > 0 {
		return fmt.Errorf("%s exist; %w", strings.Join(collisions, ", "), errExists)
	}

	return nil
}

// runReplay re-runs the engine from the record's seed, inputs, and
// recorded choices, exiting non-zero at the first mismatch (docs/PRD.md,
// Replay and provenance contract).
func runReplay(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("replay", flag.ContinueOnError)
	fs.SetOutput(stderr)
	ignore := fs.Bool("ignore-provenance", false, "waive the version match — and nothing else")

	if err := fs.Parse(args); err != nil {
		return parseExit(err)
	}

	if fs.NArg() != 1 {
		fmt.Fprintln(stderr, "ctchargen replay: want exactly one character.json (flags precede the filename)")

		return exitUsage
	}

	data, err := os.ReadFile(fs.Arg(0))
	if err != nil {
		fmt.Fprintf(stderr, "ctchargen replay: %v\n", err)

		return exitError
	}

	char, err := chargen.UnmarshalRecord(data)
	if err != nil {
		fmt.Fprintf(stderr, "ctchargen replay: %v\n", err)

		return exitError
	}

	if err := chargen.Replay(char, *ignore); err != nil {
		fmt.Fprintf(stderr, "ctchargen replay: %v\n", err)

		return exitError
	}

	fmt.Fprintf(stdout, "replay verified: %d events reproduced from seed %d\n", len(char.Events), char.RNG.Seed)

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
		return parseExit(err)
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
