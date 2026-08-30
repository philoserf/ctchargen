package chargen_test

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/philoserf/ctchargen/chargen"
)

// The documents in docs/ make claims about the code that only a reader
// checks. These are the claims a test can check instead.

// POLICY.md states the policy_version at the top, and every record stamps
// the one the engine holds. A reader comparing a record against the
// document has no way to tell which is right when they disagree, so they
// must not be able to.
func TestPolicyDocumentStatesTheStampedVersion(t *testing.T) {
	raw := readDoc(t, "../docs/POLICY.md")

	stated := regexp.MustCompile("`policy_version`: \\*\\*([^*]+)\\*\\*").FindStringSubmatch(raw)
	if stated == nil {
		t.Fatal("docs/POLICY.md does not state a `policy_version`, or no longer states it in the expected form")
	}

	if stated[1] != chargen.PolicyVersion {
		t.Errorf("docs/POLICY.md states policy_version %q, the engine stamps %q",
			stated[1], chargen.PolicyVersion)
	}
}

// POLICY.md calls its decision table total: it must answer every choice
// point the procedure can present. ChoiceLabels is the registry the engine
// and the prompter are already tested against; this holds the document to
// the same list, so a thirteenth choice point cannot arrive with no
// documented policy for it.
func TestPolicyTableCoversEveryChoicePoint(t *testing.T) {
	raw := readDoc(t, "../docs/POLICY.md")

	// The first cell of each table row, which is the choice label.
	rows := regexp.MustCompile("(?m)^\\| `([a-z-]+)`").FindAllStringSubmatch(raw, -1)

	documented := make([]string, 0, len(rows))
	for _, row := range rows {
		documented = append(documented, row[1])
	}

	if len(documented) == 0 {
		t.Fatal("found no choice-point rows in docs/POLICY.md; the scan is broken, not the document")
	}

	for _, label := range chargen.ChoiceLabels() {
		if !slices.Contains(documented, label) {
			t.Errorf("choice point %q has no row in docs/POLICY.md's decision table", label)
		}
	}

	for _, label := range documented {
		if !slices.Contains(chargen.ChoiceLabels(), label) {
			t.Errorf("docs/POLICY.md has a row for %q, which is not a choice point the engine presents", label)
		}
	}
}

// COVERAGE.md's Test column is the claim that a rule is actually covered.
// A renamed or deleted test leaves the claim standing and the rule
// uncovered, which is the shape of rot this project has already hit twice.
// Only the direction that matters is checked: a named test must exist.
// Most tests are not COVERAGE rows, so the reverse would be noise.
func TestCoverageNamesRealTests(t *testing.T) {
	raw := readDoc(t, "../docs/COVERAGE.md")

	existing := testFunctionsInRepo(t)

	var checked int

	for _, match := range regexp.MustCompile("`([^`]+)`").FindAllStringSubmatch(raw, -1) {
		name := match[1]
		// Fixture names are backticked in the same column; only the Go
		// identifiers are claims about the test tree.
		if !strings.HasPrefix(name, "Test") || strings.ContainsAny(name, " /.-") {
			continue
		}

		checked++

		if !slices.Contains(existing, name) {
			t.Errorf("docs/COVERAGE.md names %s, which no test declares", name)
		}
	}

	if checked == 0 {
		t.Fatal("found no test names in docs/COVERAGE.md; the scan is broken, not the document")
	}
}

func readDoc(t *testing.T, path string) string {
	t.Helper()

	raw, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		t.Fatal(err)
	}

	return string(raw)
}

// testFunctionsInRepo collects every declared Test function across the
// module, since COVERAGE.md cites tests from every package.
func testFunctionsInRepo(t *testing.T) []string {
	t.Helper()

	declaration := regexp.MustCompile(`(?m)^func (Test\w+)\(`)

	// The walk only collects paths; the files are read afterwards. Reading
	// inside the callback is a symlink race (gosec G122), and the sources
	// are not going anywhere between the two loops.
	var files []string

	err := filepath.WalkDir("..", func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return fmt.Errorf("walking the module: %w", err)
		}

		if entry.IsDir() {
			if entry.Name() == ".git" {
				return fs.SkipDir
			}

			return nil
		}

		if strings.HasSuffix(path, "_test.go") {
			files = append(files, path)
		}

		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	var names []string

	for _, path := range files {
		raw, err := os.ReadFile(filepath.Clean(path))
		if err != nil {
			t.Fatal(err)
		}

		for _, match := range declaration.FindAllStringSubmatch(string(raw), -1) {
			names = append(names, match[1])
		}
	}

	if len(names) == 0 {
		t.Fatal("found no test declarations in the module; the scan is broken, not the tests")
	}

	return names
}
