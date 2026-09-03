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
