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
	"fmt"
	"io"
	"os"
	"path"
	"sort"
	"strconv"
	"strings"
)

func main() {
	if err := run(os.Args[1:], os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, "ratchet:", err)
		os.Exit(1)
	}
}

func run(args []string, out io.Writer) error {
	if len(args) != 3 {
		return fmt.Errorf("usage: ratchet check|update <coverage profile> <ratchet file>")
	}
	mode, profilePath, ratchetPath := args[0], args[1], args[2]

	profileText, err := os.ReadFile(profilePath)
	if err != nil {
		return err
	}

	measured, err := parseProfile(bytes.NewReader(profileText))
	if err != nil {
		return fmt.Errorf("reading %s: %w", profilePath, err)
	}

	switch mode {
	case "update":
		return write(ratchetPath, measured)
	case "check":
		recorded, err := readRatchet(ratchetPath)
		if err != nil {
			return err
		}
		return check(measured, recorded, out)
	default:
		return fmt.Errorf("unknown mode %q: want check or update", mode)
	}
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

		if _, seen := uncovered[pkg]; !seen {
			uncovered[pkg] = 0
		}
		if count == 0 {
			uncovered[pkg] += statements
		}
	}

	return uncovered, scanner.Err()
}

// parseLine reads one profile line, which has the shape
// "import/path/file.go:12.34,14.5 2 1" — location, statement count,
// execution count.
func parseLine(text string) (pkg string, statements, count int, err error) {
	rest, countField, ok := cutLast(text, " ")
	if !ok {
		return "", 0, 0, fmt.Errorf("malformed profile line %q", text)
	}
	location, statementField, ok := cutLast(rest, " ")
	if !ok {
		return "", 0, 0, fmt.Errorf("malformed profile line %q", text)
	}
	file, _, ok := cutLast(location, ":")
	if !ok {
		return "", 0, 0, fmt.Errorf("malformed profile location %q", location)
	}

	if statements, err = strconv.Atoi(statementField); err != nil {
		return "", 0, 0, fmt.Errorf("statement count in %q: %w", text, err)
	}
	if count, err = strconv.Atoi(countField); err != nil {
		return "", 0, 0, fmt.Errorf("execution count in %q: %w", text, err)
	}

	return path.Dir(file), statements, count, nil
}

func cutLast(s, sep string) (before, after string, found bool) {
	i := strings.LastIndex(s, sep)
	if i < 0 {
		return s, "", false
	}
	return s[:i], s[i+len(sep):], true
}

func readRatchet(name string) (map[string]int, error) {
	text, err := os.ReadFile(name)
	if err != nil {
		return nil, err
	}

	recorded := make(map[string]int)
	scanner := bufio.NewScanner(bytes.NewReader(text))
	for line := 1; scanner.Scan(); line++ {
		text := strings.TrimSpace(scanner.Text())
		if text == "" || strings.HasPrefix(text, "#") {
			continue
		}
		pkg, field, ok := cutLast(text, " ")
		if !ok {
			return nil, fmt.Errorf("%s line %d: malformed entry %q", name, line, text)
		}
		n, err := strconv.Atoi(field)
		if err != nil {
			return nil, fmt.Errorf("%s line %d: %w", name, line, err)
		}
		recorded[pkg] = n
	}

	return recorded, scanner.Err()
}

func write(name string, measured map[string]int) error {
	var b strings.Builder
	b.WriteString("# Uncovered statements per package. A number may fall; it may never rise.\n")
	b.WriteString("# Regenerate with: task ratchet:update\n")
	for _, pkg := range sorted(measured) {
		fmt.Fprintf(&b, "%s %d\n", pkg, measured[pkg])
	}

	return os.WriteFile(name, []byte(b.String()), 0o644)
}

// check compares what was measured against what was recorded. Every
// disagreement is reported, not only the first: one run should say
// everything that is wrong.
func check(measured, recorded map[string]int, out io.Writer) error {
	var risen, missing, stale, fallen []string

	for _, pkg := range sorted(measured) {
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
	for _, pkg := range sorted(recorded) {
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
		problems = append(problems, report("coverage improved — lock it in, a stale number is a ratchet that has stopped holding:", fallen))
	}

	if len(problems) > 0 {
		return fmt.Errorf("%s\n\nrun: task ratchet:update", strings.Join(problems, "\n\n"))
	}

	_, err := fmt.Fprintf(out, "ratchet holds: %d packages\n", len(measured))

	return err
}

func sorted(m map[string]int) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	return keys
}
