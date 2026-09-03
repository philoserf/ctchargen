// Command ratchet holds each package's count of uncovered statements at or
// below a checked-in number.
//
// The count is an integer, not a percentage, and that is the whole point. A
// percentage holds still while a guarded branch adds one covered statement
// and one uncovered; an integer count trips on the first.
//
// Usage:
//
//	ratchet check  <coverage profile> <ratchet file>
//	ratchet update <coverage profile> <ratchet file>
//
// check fails when a package's uncovered count rises, when a package is
// missing from the ratchet file, when the file names a package the profile
// does not, and when a count has fallen — because a stale number is a
// ratchet that has stopped holding. update rewrites the file from the
// profile.
package main

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"maps"
	"os"
	"path"
	"slices"
	"strconv"
	"strings"
)

// ratchetFileMode is the mode the ratchet file is written with. It is
// rewritten wholesale by `task ratchet:update` and read by the gate, so it
// needs no group or world access of its own.
const ratchetFileMode = 0o600

// The four ways this tool refuses to run, each matchable.
var (
	errUsage   = errors.New("usage: ratchet check|update <coverage profile> <ratchet file>")
	errProfile = errors.New("malformed coverage profile")
	errRatchet = errors.New("malformed ratchet file")
	errHeld    = errors.New("the coverage ratchet does not hold")
)

func main() {
	err := run(os.Args[1:], os.Stdout)
	if err != nil {
		fmt.Fprintln(os.Stderr, "ratchet:", err)
		os.Exit(1)
	}
}

func run(args []string, out io.Writer) error {
	const wantArgs = 3

	if len(args) != wantArgs {
		return errUsage
	}

	mode, profilePath, ratchetPath := args[0], args[1], args[2]

	// The mode is checked before anything is read, so a mistyped verb is
	// named as one rather than reported as whatever the file access hits
	// first.
	if mode != "check" && mode != "update" {
		return fmt.Errorf("%w: unknown mode %q", errUsage, mode)
	}

	profileText, err := os.ReadFile(profilePath)
	if err != nil {
		return fmt.Errorf("%w: %w", errProfile, err)
	}

	measured, err := parseProfile(bytes.NewReader(profileText))
	if err != nil {
		return fmt.Errorf("reading %s: %w", profilePath, err)
	}

	if mode == "update" {
		return write(ratchetPath, measured)
	}

	recorded, err := readRatchet(ratchetPath)
	if err != nil {
		return err
	}

	return check(measured, recorded, out)
}

// parseProfile counts, per package, the statements a coverage profile marks
// as never executed. Every package the profile mentions appears in the
// result, including those with nothing uncovered.
func parseProfile(r io.Reader) (map[string]int, error) {
	uncovered := make(map[string]int)

	scanner := bufio.NewScanner(r)
	for line := 1; scanner.Scan(); line++ {
		text := strings.TrimSpace(scanner.Text())
		if text == "" || strings.HasPrefix(text, "mode:") {
			continue
		}

		pkg, statements, count, err := parseLine(text)
		if err != nil {
			return nil, fmt.Errorf("line %d: %w", line, err)
		}

		// The add is unconditional so that a package with nothing
		// uncovered still earns an entry: a later regression in it must
		// read as a rise, not as a package the ratchet has never seen.
		n := 0
		if count == 0 {
			n = statements
		}

		uncovered[pkg] += n
	}

	err := scanner.Err()
	if err != nil {
		return nil, fmt.Errorf("%w: %w", errProfile, err)
	}

	return uncovered, nil
}

// parseLine reads one profile line, which has the shape
// "import/path/file.go:12.34,14.5 2 1" — location, statement count,
// execution count.
// an error - say nothing about which is which; the names are the
// documentation.
//
//nolint:nonamedreturns // four unlabelled results - a path, two counts and
func parseLine(text string) (pkg string, statements, count int, err error) {
	rest, countField, ok := strings.CutLast(text, " ")
	if !ok {
		return "", 0, 0, fmt.Errorf("%w: line %q", errProfile, text)
	}

	location, statementField, ok := strings.CutLast(rest, " ")
	if !ok {
		return "", 0, 0, fmt.Errorf("%w: line %q", errProfile, text)
	}

	file, _, ok := strings.CutLast(location, ":")
	if !ok {
		return "", 0, 0, fmt.Errorf("%w: location %q", errProfile, location)
	}

	statements, err = strconv.Atoi(statementField)
	if err != nil {
		return "", 0, 0, fmt.Errorf("%w: statement count in %q: %w", errProfile, text, err)
	}

	count, err = strconv.Atoi(countField)
	if err != nil {
		return "", 0, 0, fmt.Errorf("%w: execution count in %q: %w", errProfile, text, err)
	}

	return path.Dir(file), statements, count, nil
}

func readRatchet(name string) (map[string]int, error) {
	data, err := os.ReadFile(name)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", errRatchet, err)
	}

	recorded := make(map[string]int)

	scanner := bufio.NewScanner(bytes.NewReader(data))
	for line := 1; scanner.Scan(); line++ {
		text := strings.TrimSpace(scanner.Text())
		if text == "" || strings.HasPrefix(text, "#") {
			continue
		}

		pkg, field, ok := strings.CutLast(text, " ")
		if !ok {
			return nil, fmt.Errorf("%w: %s line %d: entry %q", errRatchet, name, line, text)
		}

		n, atoiErr := strconv.Atoi(field)
		if atoiErr != nil {
			return nil, fmt.Errorf("%w: %s line %d: %w", errRatchet, name, line, atoiErr)
		}

		recorded[pkg] = n
	}

	err = scanner.Err()
	if err != nil {
		return nil, fmt.Errorf("%w: %w", errRatchet, err)
	}

	return recorded, nil
}

func write(name string, measured map[string]int) error {
	var b strings.Builder

	b.WriteString("# Uncovered statements per package. A number may fall; it may never rise.\n")
	b.WriteString("# Regenerate with: task ratchet:update\n")

	for _, pkg := range slices.Sorted(maps.Keys(measured)) {
		fmt.Fprintf(&b, "%s %d\n", pkg, measured[pkg])
	}

	err := os.WriteFile(name, []byte(b.String()), ratchetFileMode)
	if err != nil {
		return fmt.Errorf("writing %s: %w", name, err)
	}

	return nil
}

// check compares what was measured against what was recorded. Every
// disagreement is reported, not only the first: one run should say
// everything that is wrong.
func check(measured, recorded map[string]int, out io.Writer) error {
	var risen, missing, stale, fallen []string

	for _, pkg := range slices.Sorted(maps.Keys(measured)) {
		was, known := recorded[pkg]
		switch {
		case !known:
			missing = append(missing, fmt.Sprintf("  %s has %d uncovered and no entry", pkg, measured[pkg]))
		case measured[pkg] > was:
			risen = append(risen, fmt.Sprintf("  %s: %d uncovered, was %d", pkg, measured[pkg], was))
		case measured[pkg] < was:
			fallen = append(fallen, fmt.Sprintf("  %s: %d uncovered, was %d", pkg, measured[pkg], was))
		}
	}

	for _, pkg := range slices.Sorted(maps.Keys(recorded)) {
		if _, ok := measured[pkg]; !ok {
			stale = append(stale, fmt.Sprintf("  %s is recorded but the profile does not mention it", pkg))
		}
	}

	report := func(headline string, lines []string) string {
		return headline + "\n" + strings.Join(lines, "\n")
	}

	var problems []string

	if len(risen) > 0 {
		problems = append(problems, report("coverage fell — these packages gained uncovered statements:", risen))
	}

	if len(missing) > 0 {
		problems = append(problems, report("new packages are not in the ratchet:", missing))
	}

	if len(stale) > 0 {
		problems = append(problems, report("the ratchet names packages that no longer exist:", stale))
	}

	if len(fallen) > 0 {
		problems = append(problems, report(
			"coverage improved — lock it in, a stale number is a ratchet that has stopped holding:", fallen,
		))
	}

	if len(problems) > 0 {
		return fmt.Errorf("%w:\n\n%s\n\nrun: task ratchet:update", errHeld, strings.Join(problems, "\n\n"))
	}

	_, err := fmt.Fprintf(out, "ratchet holds: %d packages\n", len(measured))
	if err != nil {
		return fmt.Errorf("reporting the result: %w", err)
	}

	return nil
}
