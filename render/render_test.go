package render_test

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/philoserf/ctchargen/chargen"
	"github.com/philoserf/ctchargen/render"
)

// The fixtures are the renderer's own output over the engine's fixture
// characters (same seeds as chargen's goldens), compared byte for byte.
// Regenerate deliberately with `task goldens`.
var update = flag.Bool("update", false, "rewrite the golden fixtures")

type declineDecider struct{}

func (declineDecider) Decide(c chargen.Choice) (chargen.Decision, error) {
	if c.Label == chargen.ChoiceSubmitToDraft {
		return chargen.Decision{Pick: chargen.No, By: chargen.ByPlayer}, nil
	}

	d, err := chargen.AutoPolicy{}.Decide(c)
	if err != nil {
		return d, fmt.Errorf("delegating to the auto policy: %w", err)
	}

	d.By = chargen.ByPlayer

	return d, nil
}

func TestGoldenRenders(t *testing.T) {
	fixtures := []struct {
		name    string
		seed    uint64
		auto    bool
		decider chargen.Decider
	}{
		{"other-careerist", 3, true, chargen.AutoPolicy{}},
		{"death-in-service", 5, true, chargen.AutoPolicy{}},
		{"civilian-declined-draft", 175, false, declineDecider{}},
	}

	for _, f := range fixtures {
		char, err := chargen.Generate(chargen.Config{Seed: f.seed, Auto: f.auto}, f.decider)
		if err != nil {
			t.Fatalf("Generate(%s): %v", f.name, err)
		}

		outputs := []struct {
			suffix string
			got    string
		}{
			{"sheet", render.Sheet(char)},
			{"history", render.History(char)},
		}

		for _, out := range outputs {
			t.Run(f.name+"/"+out.suffix, func(t *testing.T) {
				path := filepath.Join("testdata", f.name+"."+out.suffix+".md")
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
						out.suffix, f.name)
				}
			})
		}
	}
}
