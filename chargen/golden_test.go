package chargen_test

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/philoserf/ctchargen/chargen"
)

// The fixtures are the engine's own output, compared byte for byte.
// Regenerate deliberately with `task goldens`, and read the diff before
// committing it.
var update = flag.Bool("update", false, "rewrite the golden fixtures")

// declineDecider plays the one path the auto policy never takes: refusing
// the draft (E001), producing the civilian fixture.
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

// Fixtures shared with the render package's goldens: name, seed, and how
// the choices are made.
type fixture struct {
	Name    string
	Seed    uint64
	Auto    bool
	Decider chargen.Decider
}

func fixtures() []fixture {
	return []fixture{
		{Name: "other-careerist", Seed: 3, Auto: true, Decider: chargen.AutoPolicy{}},
		{Name: "death-in-service", Seed: 5, Auto: true, Decider: chargen.AutoPolicy{}},
		{Name: "civilian-declined-draft", Seed: 175, Auto: false, Decider: declineDecider{}},
	}
}

func generate(t *testing.T, f fixture) *chargen.Character {
	t.Helper()

	char, err := chargen.Generate(chargen.Config{Seed: f.Seed, Auto: f.Auto}, f.Decider)
	if err != nil {
		t.Fatalf("Generate(%s): %v", f.Name, err)
	}

	return char
}

func TestGoldenFixtures(t *testing.T) {
	for _, f := range fixtures() {
		t.Run(f.Name, func(t *testing.T) {
			char := generate(t, f)

			got, err := char.MarshalRecord()
			if err != nil {
				t.Fatal(err)
			}

			path := filepath.Join("testdata", f.Name+".json")
			if *update {
				if err := os.WriteFile(path, got, 0o600); err != nil {
					t.Fatal(err)
				}

				return
			}

			want, err := os.ReadFile(filepath.Clean(path))
			if err != nil {
				t.Fatalf("%v (run `task goldens` to create fixtures)", err)
			}

			if string(got) != string(want) {
				t.Errorf("record for %s diverges from fixture; if intended, run `task goldens` and read the diff",
					f.Name)
			}
		})
	}
}

// Every fixture must replay: regenerating from seed and recorded choices
// reproduces the identical record (docs/PRD.md goal 3).
func TestReplayRoundTrip(t *testing.T) {
	for _, f := range fixtures() {
		t.Run(f.Name, func(t *testing.T) {
			char := generate(t, f)

			bytes, err := char.MarshalRecord()
			if err != nil {
				t.Fatal(err)
			}

			parsed, err := chargen.UnmarshalRecord(bytes)
			if err != nil {
				t.Fatal(err)
			}

			if err := chargen.Replay(parsed, false); err != nil {
				t.Errorf("Replay(%s): %v", f.Name, err)
			}
		})
	}
}

func TestReplayDetectsTampering(t *testing.T) {
	char := generate(t, fixtures()[0])

	// Find a recorded throw and change a die.
	for i := range char.Events {
		if char.Events[i].Kind == "throw" {
			char.Events[i].Dice[0] = char.Events[i].Dice[0]%6 + 1

			err := chargen.Replay(char, false)
			if err == nil {
				t.Fatal("Replay accepted a record with an altered die")
			}

			if !strings.Contains(err.Error(), "event") {
				t.Errorf("divergence error %q does not name the event", err)
			}

			return
		}
	}

	t.Fatal("fixture has no throw events")
}

func TestReplayChecksProvenance(t *testing.T) {
	char := generate(t, fixtures()[0])
	char.EngineVersion = "0.0.0-elsewhere"

	if err := chargen.Replay(char, false); err == nil {
		t.Error("Replay accepted a foreign engine_version without --ignore-provenance")
	}

	// The waiver waives the version match and nothing else; the replay
	// itself must still verify.
	if err := chargen.Replay(char, true); err != nil {
		t.Errorf("Replay with ignoreProvenance: %v", err)
	}
}
