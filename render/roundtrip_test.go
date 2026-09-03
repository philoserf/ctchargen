package render_test

import (
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
