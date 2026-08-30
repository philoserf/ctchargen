package chargen_test

import (
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/philoserf/ctchargen/chargen"
	"github.com/philoserf/ctchargen/internal/fixture"
)

// The fixtures are the engine's own output, compared byte for byte.
// Regenerate deliberately with `task goldens`, and read the diff before
// committing it.
var update = flag.Bool("update", false, "rewrite the golden fixtures")

func generate(t *testing.T, f fixture.Fixture) *chargen.Character {
	t.Helper()

	char, err := chargen.Generate(f.Config(), f.Decider)
	if err != nil {
		t.Fatalf("Generate(%s): %v", f.Name, err)
	}

	return char
}

func TestGoldenFixtures(t *testing.T) {
	for _, f := range fixture.All() {
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
	for _, f := range fixture.All() {
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
	char := generate(t, fixture.All()[0])

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
	char := generate(t, fixture.All()[0])
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

// The RNG algorithm is one of the four stamps checkProvenance compares, so
// the waiver must reach it too. It is reachable only through the final
// byte comparison, whose message ("record altered outside the engine?")
// is the one message that cannot be right here: every event matched,
// which is the evidence that nothing was altered.
func TestReplayWaivesTheAlgorithmStamp(t *testing.T) {
	char := generate(t, fixture.All()[0])
	char.RNG.Algorithm = "pcg64-elsewhere"

	if err := chargen.Replay(char, false); err == nil {
		t.Error("Replay accepted a foreign rng algorithm without --ignore-provenance")
	}

	if err := chargen.Replay(char, true); err != nil {
		t.Errorf("Replay with ignoreProvenance: %v", err)
	}
}
