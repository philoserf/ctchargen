package render_test

import (
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/philoserf/ctchargen/chargen"
	"github.com/philoserf/ctchargen/internal/fixture"
	"github.com/philoserf/ctchargen/render"
)

// The fixtures are the renderer's own output over the engine's fixture
// characters — the same set chargen's goldens use, from
// internal/fixture, so the two trees cannot come to describe different
// characters under one name. Compared byte for byte; regenerate
// deliberately with `task goldens`.
var update = flag.Bool("update", false, "rewrite the golden fixtures")

// The medical-crisis-survivor golden now carries months to the sheet, so
// this is no longer the only cover for them. It stays as the focused
// check: it pins both spellings against each other — months present and
// months absent — on one record, which no pair of goldens states as
// plainly (1D months, pp. 7-8).
func TestSheetCarriesRecoveryMonths(t *testing.T) {
	char := &chargen.Character{
		Service: "Scouts", Terms: 5, Age: 38, AgeMonths: 5, UPP: "77A643",
		Skills: []chargen.Skill{}, Benefits: chargen.Benefits{Weapons: []string{}},
	}

	if got := render.Sheet(char); !strings.Contains(got, "age 38 years 5 months") {
		t.Errorf("sheet does not carry the recovery months:\n%s", got)
	}

	char.AgeMonths = 0

	if got := render.Sheet(char); !strings.Contains(got, "age 38.") {
		t.Errorf("a whole-year age should read plainly:\n%s", got)
	}
}

// rendered is the subset of the golden roster this package pins, and it is
// deliberately smaller than chargen's. The engine's goldens carry one
// character per service because each service's rule tables differ and the
// bytes are the only thing pinning them; the renderer never branches on
// service, so a seventh careerist would be a second copy of the same
// evidence. What it does branch on is listed here, one fixture each:
//
//	civilian-declined-draft   no service at all
//	death-in-service          the died-in-service status line
//	duke                      an assumed hereditary title
//	free-trader               a mortgaged ship, passages, retirement pay
//	scout-ship                a ship held in constructive possession
//	medical-crisis-survivor   an age carrying months
//
// Measured: these six give the same 99.3% statement coverage as all
// fourteen. Four give 97.8%, three give 96.3% — six is where it saturates.
// Add a fixture here only when the renderer gains a branch, not when the
// engine gains a character.
var rendered = map[string]bool{
	"civilian-declined-draft": true,
	"death-in-service":        true,
	"duke":                    true,
	"free-trader":             true,
	"scout-ship":              true,
	"medical-crisis-survivor": true,
}

func TestGoldenRenders(t *testing.T) {
	for _, f := range fixture.All() {
		if !rendered[f.Name] {
			continue
		}

		char, err := chargen.Generate(chargen.Config{Seed: f.Seed, Service: f.Service, Auto: f.Auto}, f.Decider)
		if err != nil {
			t.Fatalf("Generate(%s): %v", f.Name, err)
		}

		outputs := []struct {
			suffix string
			got    string
		}{
			{"sheet", render.Sheet(char)},
			{"history", render.History(char)},
		}

		for _, out := range outputs {
			t.Run(f.Name+"/"+out.suffix, func(t *testing.T) {
				path := filepath.Join("testdata", f.Name+"."+out.suffix+".md")
				if *update {
					if err := os.WriteFile(path, []byte(out.got), 0o600); err != nil {
						t.Fatal(err)
					}

					return
				}

				want, err := os.ReadFile(filepath.Clean(path))
				if err != nil {
					t.Fatalf("%v (run `task goldens` to create fixtures)", err)
				}

				if out.got != string(want) {
					t.Errorf("%s render for %s diverges from fixture; if intended, run `task goldens` and read the diff",
						out.suffix, f.Name)
				}
			})
		}
	}
}
