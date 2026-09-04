package render_test

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/philoserf/ctchargen/render"
)

// A record written earlier renders to the same sheet and the same transcript
// the generating run produced.
//
// This is what the render subcommand promises, and it is worth asserting
// rather than assuming: the two go through one renderer precisely so they
// cannot disagree, and this is the test that would notice if a second one
// appeared.
func TestARecordRendersToWhatGeneratedItRendered(t *testing.T) {
	t.Parallel()

	records, err := filepath.Glob(filepath.Join(goldenDir, "*.json"))
	if err != nil {
		t.Fatalf("looking for goldens: %v", err)
	}

	if len(records) == 0 {
		t.Fatal("no goldens found")
	}

	for _, path := range records {
		name := strings.TrimSuffix(filepath.Base(path), ".json")

		text, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("reading %s: %v", path, err)
		}

		sheet, err := render.SheetFrom(text)
		if err != nil {
			t.Errorf("%s: %v", name, err)

			continue
		}

		compare(t, name+".sheet.md", sheet)

		transcript, err := render.TranscriptFrom(text)
		if err != nil {
			t.Errorf("%s: %v", name, err)

			continue
		}

		compare(t, name+".transcript.md", transcript)
	}
}

func compare(t *testing.T, name, got string) {
	t.Helper()

	want, err := os.ReadFile(filepath.Join(goldenDir, name))
	if err != nil {
		t.Fatalf("reading %s: %v", name, err)
	}

	if got != string(want) {
		t.Errorf("%s rendered from the record differs from what generating it rendered", name)
	}
}

// A file that is not a record is refused, rather than rendered as a blank
// character.
func TestSomethingElseIsNotARecord(t *testing.T) {
	t.Parallel()

	for name, text := range map[string]string{
		"not JSON":      "# a character sheet\n",
		"another shape": `{"hello":"world"}`,
		"empty object":  `{}`,
	} {
		_, err := render.SheetFrom([]byte(text))
		if err == nil {
			t.Errorf("%s was accepted as a record", name)
		}
	}
}

// minimalRecord is the least decode accepts, so a test can pin one detail
// without carrying a whole character.
func minimalRecord(body string) []byte {
	return []byte(`{"upp":"777777","ruleset":"test",` + body + `}`)
}

// A ship's terms are read off its kind, not inferred from its numbers. A
// Free Trader received and never paid down carries years 0 and paymentYears
// 0, and is owned - not held in constructive possession, which is the scout
// ship's arrangement (p. 23).
func TestAShipsTermsFollowItsKind(t *testing.T) {
	t.Parallel()

	for name, tc := range map[string]struct{ kind, want, avoid string }{
		"a new Free Trader": {
			kind:  "Free Trader, Type A",
			want:  "owned free and clear",
			avoid: "constructive possession",
		},
		"a scout ship": {
			kind:  "Scout ship, Type S",
			want:  "held in constructive possession",
			avoid: "owned free and clear",
		},
	} {
		sheet, err := render.SheetFrom(minimalRecord(
			`"benefits":{"cash":0,"ships":[{"kind":"` + tc.kind +
				`","tons":200,"years":0,"paymentYears":0}]}`,
		))
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}

		if !strings.Contains(sheet, tc.want) {
			t.Errorf("%s does not read %q:\n%s", name, tc.want, sheet)
		}

		if strings.Contains(sheet, tc.avoid) {
			t.Errorf("%s reads %q, which is the other ship's terms:\n%s", name, tc.avoid, sheet)
		}
	}
}

// An event kind this build does not know is said to be unknown, rather than
// rendered as an outcome - which would print a blank numbered line and claim
// the transcript was complete. Reading another build's record is the whole
// reason the render subcommand exists.
func TestAnUnknownEventKindIsSaidToBeUnknown(t *testing.T) {
	t.Parallel()

	transcript, err := render.TranscriptFrom(minimalRecord(
		`"events":[{"seq":9,"kind":"portent"}]`,
	))
	if err != nil {
		t.Fatalf("reading a record with an unknown event kind: %v", err)
	}

	if !strings.Contains(transcript, `unknown event kind "portent"`) {
		t.Errorf("an unknown event kind rendered as %q", transcript)
	}
}

// Each of decode's two refusals names what is missing.
func TestARecordMustCarryAUPPAndARuleset(t *testing.T) {
	t.Parallel()

	_, err := render.SheetFrom([]byte(`{"ruleset":"test"}`))
	if err == nil || !strings.Contains(err.Error(), "no UPP") {
		t.Errorf("a record with no UPP was refused as %v", err)
	}

	_, err = render.SheetFrom([]byte(`{"upp":"777777"}`))
	if err == nil || !strings.Contains(err.Error(), "no ruleset") {
		t.Errorf("a record with no ruleset was refused as %v", err)
	}
}

// A record carrying an event that will not read is refused by both
// renderings, rather than rendered without it.
//
// The sheet is the one that changed: it now reads the events too, to find out
// whether the player answered any choice. A sheet that swallowed the failure
// would print the regenerate line - a promise that the seed brings this
// character back - about a record it could not finish reading.
func TestAnEventThatWillNotReadIsRefused(t *testing.T) {
	t.Parallel()

	// Both, because the sheet reads the events twice for two questions and
	// reaches them by different routes. A civilian returns from the headline
	// before it asks whose service this is, so only a record carrying one
	// exercises that path - and it is the path that would otherwise print a
	// headline about a record it could not finish reading.
	for name, unreadable := range map[string][]byte{
		"a civilian": minimalRecord(`"events":[3]`),
		"a serving character": minimalRecord(
			`"service":"Navy","terms":2,"enlistment":{"how":"enlisted"},"events":[3]`),
	} {
		_, err := render.SheetFrom(unreadable)
		if err == nil || !strings.Contains(err.Error(), "an event will not read") {
			t.Errorf("%s: the sheet rendered a record it could not read: %v", name, err)
		}

		_, err = render.TranscriptFrom(unreadable)
		if err == nil || !strings.Contains(err.Error(), "an event will not read") {
			t.Errorf("%s: the transcript rendered a record it could not read: %v", name, err)
		}
	}
}

// headlineCase is one record and what its headline must and must not read.
type headlineCase struct{ body, wants, does string }

// served is a record of a character in a service, with whatever the case
// under test wants to say about how he got there.
func served(service, how, provenance string) string {
	return fmt.Sprintf(`"service":%q,"terms":2,"enlistment":{"how":%q},%s`,
		service, how, provenance)
}

// chose is a Service choice event answered by whoever the case names.
func chose(by string) string {
	return fmt.Sprintf(`"inputs":{"seed":1},"events":[{"seq":1,"kind":"choice",`+
		`"point":"Service","by":%q,"alternatives":["Navy","Other"],"chosen":"Navy"}]`, by)
}

// unchanged is the headline of a character with nothing to explain.
const unchanged = "Navy, 2 terms"

// The headline says whose service it is, where that is not obvious.
//
// The service record four sections down has always been honest. The headline
// is the line a referee reads, and a character who was typed as a Marine and
// came back a Navy man had nothing there to explain it.
//
// Everything turns on the service ATTEMPTED, not on who named it. The draft
// can override either decider, and crediting one with an outcome it did not
// pick is worse than saying nothing.
func TestTheHeadlineSaysWhoseServiceItIs(t *testing.T) {
	t.Parallel()

	for name, tc := range map[string]headlineCase{
		"asked for one service and got another": {
			body:  served("Navy", "drafted", `"inputs":{"seed":1,"service":"Marines"}`),
			wants: "Navy (drafted after the Marines refused him)",
		},
		"asked for a service and got it": {
			body:  served("Navy", "enlisted", `"inputs":{"seed":1,"service":"Navy"}`),
			wants: unchanged, does: "refused",
		},
		"asked for nothing and the policy picked": {
			body:  served("Navy", "enlisted", chose("policy")),
			wants: "Navy (service chosen by the policy)",
		},
		// A player who named it himself is told nothing he does not know.
		"asked for nothing and the player picked": {
			body:  served("Navy", "enlisted", chose("player")),
			wants: unchanged, does: "chosen by",
		},
		// The draft overrides the attempt. Saying "chosen by the policy"
		// over a draft's outcome credits the policy with a choice it never
		// made, in the case this exists to expose.
		"the policy picked one and the draft gave another": {
			body:  served("Marines", "drafted", chose("policy")),
			wants: "Marines (drafted after the Navy refused him)", does: "chosen by the policy",
		},
		// And after a draft the player does not know either: he chose Navy.
		"the player picked one and the draft gave another": {
			body:  served("Marines", "drafted", chose("player")),
			wants: "Marines (drafted after the Navy refused him)",
		},
		// A record saying nothing about how he got there. The tool writes
		// none, but the renderer is handed records rather than generating
		// them, and inventing a decider is worse than saying nothing.
		"nothing recorded about the service at all": {
			body:  served("Navy", "enlisted", `"inputs":{"seed":1}`),
			wants: unchanged, does: "(",
		},
	} {
		checkHeadline(t, name, tc)
	}
}

// checkHeadline reads the headline out of the sheet and asserts on that
// alone. Searching the whole sheet is too loose - "(" would find the
// "(unnamed)" title - and the headline is what every case here is about.
func checkHeadline(t *testing.T, name string, tc headlineCase) {
	t.Helper()

	sheet, err := render.SheetFrom(minimalRecord(tc.body))
	if err != nil {
		t.Errorf("%s: %v", name, err)

		return
	}

	lead := ""

	for line := range strings.Lines(sheet) {
		if strings.HasPrefix(line, "UPP ") {
			lead = line

			break
		}
	}

	if lead == "" {
		t.Errorf("%s: no headline in\n%s", name, sheet)

		return
	}

	if !strings.Contains(lead, tc.wants) {
		t.Errorf("%s: the headline does not read %q: %q", name, tc.wants, lead)
	}

	if tc.does != "" && strings.Contains(lead, tc.does) {
		t.Errorf("%s: the headline reads %q and should not: %q", name, tc.does, lead)
	}
}

// One term is one term.
//
// "1 term" is a substring of "1 terms", so the singular case is pinned by what
// must NOT appear. Asserting only the positive would pass on the bug.
func TestOneTermIsSingular(t *testing.T) {
	t.Parallel()

	for name, tc := range map[string]struct {
		terms       int
		wants, does string
	}{
		"one":  {1, "1 term", "1 terms"},
		"two":  {2, "2 terms", ""},
		"none": {0, "0 terms", ""},
	} {
		sheet, err := render.SheetFrom(minimalRecord(fmt.Sprintf(
			`"service":"Navy","terms":%d,"enlistment":{"how":"enlisted"},"inputs":{"seed":1,"service":"Navy"}`,
			tc.terms)))
		if err != nil {
			t.Errorf("%s: %v", name, err)

			continue
		}

		if !strings.Contains(sheet, tc.wants) {
			t.Errorf("%s: the headline does not read %q:\n%s", name, tc.wants, sheet)
		}

		if tc.does != "" && strings.Contains(sheet, tc.does) {
			t.Errorf("%s: the headline reads %q:\n%s", name, tc.does, sheet)
		}
	}
}

// The errata are on the transcript and not on the sheet.
//
// On a sheet handed to a player they are codes he cannot expand; beside the
// outcome they governed they are how a reading is checked against the page,
// which is what the transcript is for.
func TestTheErrataAreOnTheTranscriptAndNotTheSheet(t *testing.T) {
	t.Parallel()

	record := minimalRecord(`"errata":["E001"],` +
		`"events":[{"seq":1,"kind":"outcome","description":"drafted","errata":["E001"]}]`)

	sheet, err := render.SheetFrom(record)
	if err != nil {
		t.Fatalf("the sheet: %v", err)
	}

	if strings.Contains(sheet, "E001") || strings.Contains(sheet, "Readings applied") {
		t.Errorf("the sheet carries an erratum id:\n%s", sheet)
	}

	transcript, err := render.TranscriptFrom(record)
	if err != nil {
		t.Fatalf("the transcript: %v", err)
	}

	if !strings.Contains(transcript, "E001") {
		t.Errorf("the transcript lost the erratum that governed an outcome:\n%s", transcript)
	}
}

// A backtick in the record does not break the code span holding the command.
//
// Markdown closes a code span on the first run of backticks matching the run
// that opened it, so a command carrying one inside a single-backtick span
// ends early and the reader is shown half a command line - which is worse
// than none, because the half looks complete.
//
// Shell quoting is what makes such a value harmless; this is the separate
// question of whether the reader sees all of what he is offered.
func TestABacktickDoesNotBreakTheCodeSpan(t *testing.T) {
	t.Parallel()

	sheet, err := render.SheetFrom(minimalRecord(
		"\"inputs\":{\"seed\":1,\"career\":\"serve`id`\",\"skills\":\"advanced\",\"muster\":\"cash\"}"))
	if err != nil {
		t.Fatalf("rendering: %v", err)
	}

	_, line, found := strings.Cut(sheet, "Regenerate with ")
	if !found {
		t.Fatalf("no regenerate line in\n%s", sheet)
	}

	fence := len(line) - len(strings.TrimLeft(line, "`"))
	if fence == 0 {
		t.Fatalf("the command is not in a code span: %q", line)
	}

	// The command sits between the fences; a run as long as the fence
	// anywhere inside would close the span early.
	body, _, _ := strings.Cut(line[fence:], strings.Repeat("`", fence))
	if strings.Contains(body, strings.Repeat("`", fence)) {
		t.Errorf("a %d-backtick fence does not hold a command containing one: %q", fence, body)
	}

	if !strings.Contains(body, "--sheet") {
		t.Errorf("the span closed before the end of the command: %q", body)
	}
}
