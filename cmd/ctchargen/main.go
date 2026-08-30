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
	"slices"
	"strconv"
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

  policy flags, with --auto, select how it decides (docs/POLICY.md):
                [--skills service|personal|advanced|rounded]
                [--muster cash|benefits] [--career max|N]
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

// newFlagSet builds a subcommand's flag set with its own output
// discarded. The flag package writes a help request and a parse error to
// the same place, and those are not the same kind of thing here; silencing
// it lets reportParse pick the stream per outcome.
func newFlagSet(name string) *flag.FlagSet {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	return fs
}

// reportParse answers a flag.Parse failure and chooses where the answer
// goes. flag.ErrHelp is the sentinel for -h/--help: a handled request, so
// the flag list goes to stdout and the exit is clean — the same treatment
// the top-level --help gets, so that redirecting either one captures it.
// Anything else is a usage error, and goes to stderr with the flag list
// after it.
func reportParse(fs *flag.FlagSet, err error, stdout, stderr io.Writer) int {
	if errors.Is(err, flag.ErrHelp) {
		fmt.Fprintf(stdout, "usage: ctchargen %s [flags]\n", fs.Name())
		fs.SetOutput(stdout)
		fs.PrintDefaults()

		return exitOK
	}

	fmt.Fprintf(stderr, "ctchargen %s: %v\n", fs.Name(), err)
	fs.SetOutput(stderr)
	fs.PrintDefaults()

	return exitUsage
}

// policyFlags registers the auto policy's three selectable rows
// (docs/POLICY.md) on a subcommand, shared by `new` and `batch` so the two
// cannot drift apart. Zero values are the defaults, so a caller who names
// none of them gets the policy this tool has always applied.
type policyFlags struct {
	skills *string
	muster *string
	career *string
}

func registerPolicyFlags(fs *flag.FlagSet) policyFlags {
	strategies := chargen.PolicyStrategies()

	return policyFlags{
		skills: fs.String("skills", "",
			"auto: which skills table each eligibility goes to — "+strings.Join(strategies["skills"], "|")),
		muster: fs.String("muster", "",
			"auto: which mustering-out table to prefer — "+strings.Join(strategies["muster"], "|")+
				"; benefits never rolls for cash, so the character musters out with none"),
		career: fs.String("career", "",
			"auto: max, or the term to leave after — intent only; the throw still decides (pp. 6-7)"),
	}
}

var (
	errBadStrategy       = errors.New("unknown policy strategy")
	errPolicyWithoutAuto = errors.New("the policy flags select how --auto decides, so they need --auto")
)

// apply validates the selections and writes them into the config. Naming a
// policy flag without --auto is refused rather than ignored: in interactive
// mode the player decides, and quietly discarding a flag the user typed is
// worse than saying no to it.
func (p policyFlags) apply(cfg *chargen.Config, auto bool) error {
	if (*p.skills != "" || *p.muster != "" || *p.career != "") && !auto {
		return errPolicyWithoutAuto
	}

	if err := named("skills", *p.skills, &cfg.Skills); err != nil {
		return err
	}

	if err := named("muster", *p.muster, &cfg.Muster); err != nil {
		return err
	}

	return career(*p.career, &cfg.CareerTerms)
}

// named checks one flag against the strategies the policy publishes for it.
func named(flag, value string, field *string) error {
	if value == "" {
		return nil
	}

	allowed := chargen.PolicyStrategies()[flag]
	if !slices.Contains(allowed, value) {
		return fmt.Errorf("%w: --%s %q, want one of %s",
			errBadStrategy, flag, value, strings.Join(allowed, ", "))
	}

	*field = value

	return nil
}

// career takes a term number rather than a strategy name, so it is checked
// against the rules' own cap rather than against a published list.
func career(value string, field *int) error {
	if value == "" || value == "max" {
		return nil
	}

	terms, err := strconv.Atoi(value)
	if err != nil || terms < 1 || terms > 7 {
		return fmt.Errorf("%w: --career %q, want max or a term 1-7 (voluntary service caps at 7, p. 7)",
			errBadStrategy, value)
	}

	*field = terms

	return nil
}

func runNew(args []string, seedSource func() (uint64, error), stdin io.Reader, stdout, stderr io.Writer) int {
	fs := newFlagSet("new")
	seed := fs.Uint64("seed", 0, "RNG seed (default: drawn from the OS)")
	auto := fs.Bool("auto", false, "apply the policy (docs/POLICY.md) to every choice, instead of asking")
	svc := fs.String("service", "", "force the enlistment attempt only; a failed throw still goes to the draft (p. 5)")
	name := fs.String("name", "", "character name (blank by default; the book's naming section is advice, not a table)")
	outPath := fs.String("o", "", "write the JSON record to this file instead of stdout")
	force := fs.Bool("force", false, "overwrite an existing output file")
	policy := registerPolicyFlags(fs)

	if err := fs.Parse(args); err != nil {
		return reportParse(fs, err, stdout, stderr)
	}

	if fs.NArg() != 0 {
		fmt.Fprintf(stderr, "ctchargen new: unexpected argument %q (flags precede any filename)\n", fs.Arg(0))

		return exitUsage
	}

	// A mistyped strategy is a usage error, so it is answered before
	// anything is drawn, reserved, or asked of the player.
	cfg := chargen.Config{Name: *name, Service: *svc, Auto: *auto}
	if err := policy.apply(&cfg, *auto); err != nil {
		fmt.Fprintf(stderr, "ctchargen new: %v\n", err)

		return exitUsage
	}

	if err := resolveSeed(fs, seed, seedSource); err != nil {
		fmt.Fprintf(stderr, "ctchargen new: %v\n", err)

		return exitError
	}

	cfg.Seed = *seed

	// Before the first prompt, not after the last one.
	if err := reserveOutput(*outPath, *force); err != nil {
		fmt.Fprintf(stderr, "ctchargen new: %v\n", err)

		return exitError
	}

	var decider chargen.Decider = chargen.NewAutoPolicy(cfg)
	if !*auto {
		decider = newPrompter(stdin, stderr)
	}

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
	fs := newFlagSet("batch")
	count := fs.Int("count", 0, "number of characters to generate")
	seed := fs.Uint64("seed", 0, "base RNG seed (default: drawn from the OS); member i uses seed+i")
	auto := fs.Bool("auto", false, "required: batch applies the policy (docs/POLICY.md)")
	svc := fs.String("service", "", "force each member's enlistment attempt only (p. 5)")
	outPath := fs.String("o", "", "JSONL file, or an existing directory for one file per character")
	force := fs.Bool("force", false, "overwrite existing output files")
	policy := registerPolicyFlags(fs)

	if err := fs.Parse(args); err != nil {
		return reportParse(fs, err, stdout, stderr)
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

	member := chargen.Config{Service: *svc, Auto: true}
	if err := policy.apply(&member, true); err != nil {
		fmt.Fprintf(stderr, "ctchargen batch: %v\n", err)

		return exitUsage
	}

	for i := range *count {
		cfg := member
		cfg.Seed = *seed + uint64(i) // #nosec G115 -- count is small; uint64 wraparound would be harmless and recorded

		char, err := chargen.Generate(cfg, chargen.NewAutoPolicy(cfg))
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
	// A path that does not exist yet names a new JSONL file; a path that
	// cannot be statted at all is reported rather than quietly treated as
	// one, which would surface later as a confusing write error against a
	// shape the user did not ask for.
	if outPath != "" {
		switch info, err := os.Stat(outPath); {
		case err == nil && info.IsDir():
			return writeBatchDir(chars, outPath, force)
		case err != nil && !errors.Is(err, os.ErrNotExist):
			return fmt.Errorf("checking %s: %w", outPath, err)
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
	fs := newFlagSet("replay")
	ignore := fs.Bool("ignore-provenance", false, "waive the version match — and nothing else")

	if err := fs.Parse(args); err != nil {
		return reportParse(fs, err, stdout, stderr)
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

// refuseExisting reports an occupied path, or a path that cannot be
// statted at all — which is not the same as a free one and must not be
// treated as one.
func refuseExisting(path string) error {
	if _, err := os.Stat(path); err == nil {
		return fmt.Errorf("%s exists; %w", path, errExists)
	} else if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("checking %s: %w", path, err)
	}

	return nil
}

// reserveOutput refuses an occupied destination before the caller spends
// anything reaching it. An interactive generation is the player's
// evening, and finding the collision afterwards throws the whole
// playthrough away — so `new` asks first, as writeBatchDir does for the
// batch directory. This races a concurrent writer, as any such check
// does; writeFile's own check is still the guard, and this one exists to
// fail at the right moment.
func reserveOutput(path string, force bool) error {
	if path == "" || force {
		return nil
	}

	return refuseExisting(path)
}

func writeFile(path string, data []byte, force bool) error {
	if !force {
		if err := refuseExisting(path); err != nil {
			return err
		}
	}

	// 0644: a character record is the user's shareable output, not a secret.
	if err := os.WriteFile(path, data, 0o644); err != nil { // #nosec G306
		return fmt.Errorf("writing %s: %w", path, err)
	}

	return nil
}

func runRender(args []string, stdout, stderr io.Writer) int {
	fs := newFlagSet("render")
	history := fs.Bool("history", false, "render the generation record transcript instead of the sheet")

	if err := fs.Parse(args); err != nil {
		return reportParse(fs, err, stdout, stderr)
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
