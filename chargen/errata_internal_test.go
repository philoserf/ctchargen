package chargen

import (
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"
)

// The authority model rests on two halves staying in step: an
// interpretation is recorded in docs/ERRATA.md with its page cite, and the
// engine stamps its identifier on every record the reading governed. Only
// the code half is compiled, so the document half can rot silently — and
// did, in the other direction, when COVERAGE.md kept naming a test that no
// longer covered its row.
//
// What this catches: an id stamped with no entry to explain it, and an
// entry no code path can ever stamp. What it does NOT catch is the failure
// that produced E009 — a reading implemented with neither an entry nor a
// stamp, which leaves both sets equal and this test green. Nothing
// mechanical catches that; it took reading p. 7 and noticing that "an
// additional term" is singular. This gate is the floor, not the ceiling.

var (
	stampCall    = regexp.MustCompile(`stampErratum\("(E\d{3})"\)`)
	errataHeader = regexp.MustCompile(`(?m)^## (E\d{3})\b(.*)$`)
)

func TestErrataIDsMatchTheDocument(t *testing.T) {
	code := errataIDsInCode(t)
	documented, _ := errataHeadings(t)

	for _, id := range code {
		if !slices.Contains(documented, id) {
			t.Errorf("%s is stamped on records but docs/ERRATA.md has no entry for it: "+
				"a reading is being applied with nothing to explain or cite it", id)
		}
	}

	for _, id := range documented {
		if !slices.Contains(code, id) {
			t.Errorf("docs/ERRATA.md documents %s but no code path stamps it: "+
				"either the stamp was lost or the entry describes a reading the engine does not make", id)
		}
	}
}

// Every entry must carry a page cite in its heading. CLAUDE.md's rule is
// that no rule and no reading enters without one, and an entry written
// without a cite is the one kind of ERRATA defect that cannot be caught by
// comparing it to anything else.
func TestErrataEntriesCiteAPage(t *testing.T) {
	_, headings := errataHeadings(t)

	for id, heading := range headings {
		if !strings.Contains(heading, "p. ") && !strings.Contains(heading, "pp. ") {
			t.Errorf("%s heading carries no page cite: %q", id, strings.TrimSpace(heading))
		}
	}
}

// errataIDsInCode collects what the engine can stamp: the entries applied
// to every record, plus every literal handed to stampErratum anywhere in
// the package's own source.
func errataIDsInCode(t *testing.T) []string {
	t.Helper()

	ids := slices.Clone(appliedErrata)

	sources, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatal(err)
	}

	for _, name := range sources {
		if strings.HasSuffix(name, "_test.go") {
			continue
		}

		raw, err := os.ReadFile(filepath.Clean(name))
		if err != nil {
			t.Fatal(err)
		}

		for _, match := range stampCall.FindAllStringSubmatch(string(raw), -1) {
			if !slices.Contains(ids, match[1]) {
				ids = append(ids, match[1])
			}
		}
	}

	if len(ids) == 0 {
		t.Fatal("found no errata identifiers in the package source; the scan is broken, not the data")
	}

	return ids
}

// errataHeadings returns the documented ids and their heading lines,
// failing on a duplicate: two entries under one id would let the second
// silently describe a reading no reader would find.
func errataHeadings(t *testing.T) ([]string, map[string]string) {
	t.Helper()

	raw, err := os.ReadFile("../docs/ERRATA.md")
	if err != nil {
		t.Fatal(err)
	}

	var ids []string

	headings := map[string]string{}

	for _, match := range errataHeader.FindAllStringSubmatch(string(raw), -1) {
		id := match[1]
		if _, seen := headings[id]; seen {
			t.Errorf("docs/ERRATA.md has more than one %s entry", id)

			continue
		}

		ids = append(ids, id)
		headings[id] = match[2]
	}

	if len(ids) == 0 {
		t.Fatal("found no errata headings in docs/ERRATA.md; the scan is broken, not the document")
	}

	return ids, headings
}
