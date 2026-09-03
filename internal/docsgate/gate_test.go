// Package docsgate holds the documents to the code, in both directions.
//
// The PRD's testing section asks for three of these, and they exist for a
// failure mode nothing else catches: a reading applied in code with no entry
// in ERRATA.md, or an entry no code reaches, or a choice point the engine
// grew without POLICY.md noticing. Each gate is verified by breaking it; a
// gate never seen to fail has not been shown to hold.
//
// The package has no non-test code on purpose. It is the one place allowed
// to know about both traveller and chargen and about the documents beside
// them.
package docsgate_test

import (
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"slices"
	"testing"

	"github.com/philoserf/ctchargen/chargen"
	"github.com/philoserf/ctchargen/traveller"
)

// docs is where the governing documents live, relative to this package.
const docs = "../../docs"

// erratumHeading matches ERRATA.md's own heading form: "### E001 — ...".
var erratumHeading = regexp.MustCompile(`(?m)^### (E\d{3}) `)

// policyRow matches POLICY.md's own row form, a level-three heading whose
// text is the Go method it describes: "### `SubmitToDraft() (bool, error)`".
var policyRow = regexp.MustCompile("(?m)^### `([A-Z][A-Za-z0-9]*)\\(")

func read(t *testing.T, name string) string {
	t.Helper()

	text, err := os.ReadFile(filepath.Join(docs, name))
	if err != nil {
		t.Fatalf("the gate cannot read the document it guards: %v", err)
	}
	if len(text) == 0 {
		t.Fatalf("%s is empty", name)
	}

	return string(text)
}

// found returns the first capture of every match, sorted and deduplicated.
//
// A repeat is reported before it is compacted away. Comparing sets cannot
// notice one — two headings for the same id agree with the code as well as
// one does — and this is the only place positioned to: an erratum id is
// documented as never reused, and a second POLICY.md row for a method is a
// second answer to a question that has one.
func found(t *testing.T, re *regexp.Regexp, text, what string) []string {
	t.Helper()

	var names []string
	for _, m := range re.FindAllStringSubmatch(text, -1) {
		names = append(names, m[1])
	}
	if len(names) == 0 {
		t.Fatalf("no %s found; the gate's pattern no longer matches the document", what)
	}
	slices.Sort(names)

	for i := 1; i < len(names); i++ {
		if names[i] == names[i-1] {
			t.Errorf("%s: %s appears more than once; an id is never reused", what, names[i])
		}
	}

	return slices.Compact(names)
}

// compare reports every disagreement between two sets, not only the first.
func compare(t *testing.T, inCode, inDoc []string, code, doc string) {
	t.Helper()

	for _, name := range inCode {
		if !slices.Contains(inDoc, name) {
			t.Errorf("%s is in %s but has no entry in %s", name, code, doc)
		}
	}
	for _, name := range inDoc {
		if !slices.Contains(inCode, name) {
			t.Errorf("%s is in %s but does not exist in %s", name, doc, code)
		}
	}
}

// Every erratum the code can stamp resolves to a heading, and every heading
// is reachable from the code.
func TestErrataMatchTheDocument(t *testing.T) {
	t.Parallel()

	inCode := make([]string, 0, len(traveller.Errata))
	for _, e := range traveller.Errata {
		inCode = append(inCode, e.String())
	}
	slices.Sort(inCode)

	inDoc := found(t, erratumHeading, read(t, "ERRATA.md"), "erratum headings")

	compare(t, inCode, inDoc, "the Erratum enum", "ERRATA.md")
}

// deciderMethods is every method of the Decider interface, by name.
func deciderMethods() []string {
	iface := reflect.TypeFor[chargen.Decider]()

	names := make([]string, 0, iface.NumMethod())
	for m := range iface.Methods() {
		names = append(names, m.Name)
	}
	slices.Sort(names)

	return names
}

// Every POLICY.md row names a Decider method, and every method has a row.
// This is what makes the auto policy total: a method with no row is a
// question the policy was never told how to answer.
func TestPolicyRowsMatchTheDecider(t *testing.T) {
	t.Parallel()

	inDoc := found(t, policyRow, read(t, "POLICY.md"), "policy rows")

	compare(t, deciderMethods(), inDoc, "the Decider interface", "POLICY.md")
}

// ChoicePoint is meant to be a rendering of the Decider interface rather
// than a second list kept parallel to it. It is a second list — this is what
// stops it drifting into a different one.
func TestChoicePointsMatchTheDecider(t *testing.T) {
	t.Parallel()

	inCode := make([]string, 0, len(traveller.ChoicePoints))
	for _, c := range traveller.ChoicePoints {
		inCode = append(inCode, c.String())
	}
	slices.Sort(inCode)

	compare(t, inCode, deciderMethods(), "the ChoicePoint enum", "the Decider interface")
}

// notYetReachable names the errata no generated character can carry yet,
// because the service or the step that would stamp them is not implemented.
//
// The PRD asks that every erratum heading be reachable by some path. That
// gate cannot hold until milestone 3, so it ships with its exemptions
// written down and printed on every run. The list may only shrink, and must
// be empty when the six services and mustering out are complete.
//
// It is a list and not a map keyed by Erratum on purpose: a map keyed by the
// enum reads as a complete table of every reading, and this is the opposite
// — the few that are still owed.
type exemption struct {
	erratum traveller.Erratum
	because string
}

const (
	noEngineYet      = "the engine does not run this step yet"
	notThisMilestone = "needs a service the engine does not run yet (milestone 2)"
)

var notYetReachable = []exemption{
	{traveller.E001, noEngineYet},
	{traveller.E002, noEngineYet},
	{traveller.E003, noEngineYet},
	{traveller.E004, noEngineYet},
	{traveller.E005, notThisMilestone},
	{traveller.E006, noEngineYet},
	{traveller.E007, noEngineYet},
	{traveller.E008, noEngineYet},
	{traveller.E009, noEngineYet},
	{traveller.E010, noEngineYet},
	{traveller.E011, noEngineYet},
	{traveller.E013, notThisMilestone},
	{traveller.E014, noEngineYet},
}

// stampedOnNoRecord is E012 and only E012, which by design names no record:
// a spelling is a transcription, not a reading, and nothing about a
// character changes with the choice.
var stampedOnNoRecord = []traveller.Erratum{traveller.E012}

// Until the engine exists, this reports what the reachability gate still
// owes rather than asserting it. It fails only if the exemption list names
// an erratum that does not exist, which would mean the list has gone stale
// in the one direction that hides work.
func TestReachabilityGateOwes(t *testing.T) {
	t.Parallel()

	t.Logf("reachability gate owes %d of %d errata; it must owe none once the six services and mustering out are complete",
		len(notYetReachable), len(traveller.Errata))

	for _, e := range notYetReachable {
		if !slices.Contains(traveller.Errata[:], e.erratum) {
			t.Errorf("%s is exempted from the reachability gate but is not an erratum", e.erratum)
		}
		if slices.Contains(stampedOnNoRecord, e.erratum) {
			t.Errorf("%s is both exempted and stamped on no record; it needs one status, not two", e.erratum)
		}

		// The reason is the whole point of writing the exemption down, so
		// it is printed rather than merely stored: a list of ids says what
		// is owed, and only the reason says what would settle it.
		t.Logf("  %s: %s", e.erratum, e.because)
	}
}
