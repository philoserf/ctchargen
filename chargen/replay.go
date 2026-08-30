package chargen

import (
	"fmt"
	"reflect"
	"slices"

	"github.com/philoserf/ctchargen/dice"
)

// Replay re-runs the engine from a record's seed, inputs, and choice
// events, recomputing every throw; the stored event log is verification
// data, not input (docs/PRD.md, Replay and provenance contract). It
// returns an error naming the first diverging event's sequence number.
//
// ignoreProvenance waives the version match and nothing else;
// policy_version is never verified, because recorded choices are
// reapplied and the policy is never consulted.
func Replay(rec *Character, ignoreProvenance bool) error {
	if !ignoreProvenance {
		if err := checkProvenance(rec); err != nil {
			return err
		}
	}

	// Every input the record carries, not just the ones that steer the
	// engine: the regenerated record's inputs block is compared byte for
	// byte, so an input left behind here reads as a divergence.
	regen, err := Generate(Config{
		Seed:        rec.RNG.Seed,
		Name:        rec.Inputs.Name,
		Service:     rec.Inputs.Service,
		Auto:        rec.Inputs.Auto,
		Skills:      rec.Inputs.Skills,
		Muster:      rec.Inputs.Muster,
		CareerTerms: rec.Inputs.CareerTerms,
	}, &replayDecider{choices: recordedChoices(rec)})
	if err != nil {
		return fmt.Errorf("replay: %w", err)
	}

	// The stamps are provenance, not computation; carry the record's own
	// so the comparison checks only what the engine recomputed. All five
	// that checkProvenance compares are carried, the rng algorithm
	// included: waiving a stamp at the front door and reimposing it at the
	// back would fail an otherwise perfect replay — and fail it through
	// compare's byte check, whose "record altered outside the engine?" is
	// the one reading that cannot be right when every event matched.
	regen.SchemaVersion = rec.SchemaVersion
	regen.Ruleset = rec.Ruleset
	regen.EngineVersion = rec.EngineVersion
	regen.PolicyVersion = rec.PolicyVersion
	regen.RNG.Algorithm = rec.RNG.Algorithm

	return compare(rec, regen)
}

func checkProvenance(rec *Character) error {
	checks := []struct{ field, got, want string }{
		{"schema_version", rec.SchemaVersion, SchemaVersion},
		{"engine_version", rec.EngineVersion, EngineVersion},
		{"ruleset", rec.Ruleset, Ruleset},
		{"rng algorithm", rec.RNG.Algorithm, dice.Algorithm},
	}
	for _, c := range checks {
		if c.got != c.want {
			return fmt.Errorf("%w: record %s %q, this build %q (--ignore-provenance waives the match)",
				ErrProvenance, c.field, c.got, c.want)
		}
	}

	return nil
}

// replayDecider reapplies the recorded choices, in order, verifying each
// arrives at the same choice point it was recorded at.
type replayDecider struct {
	choices []Event
	next    int
}

func recordedChoices(rec *Character) []Event {
	var choices []Event

	for _, ev := range rec.Events {
		if ev.Kind == "choice" {
			choices = append(choices, ev)
		}
	}

	return choices
}

func (d *replayDecider) Decide(c Choice) (Decision, error) {
	if d.next >= len(d.choices) {
		return Decision{}, fmt.Errorf("%w: record has no choice left for %s at %s", ErrDiverged, c.Label, c.Step)
	}

	recorded := d.choices[d.next]
	d.next++

	if recorded.Label != c.Label || recorded.Step != c.Step {
		return Decision{}, fmt.Errorf("%w: event %d recorded choice %s at %s, engine asked %s at %s",
			ErrDiverged, recorded.Seq, recorded.Label, recorded.Step, c.Label, c.Step)
	}

	return Decision{Pick: recorded.Picked, By: recorded.By}, nil
}

func compare(rec, regen *Character) error {
	for i := range min(len(rec.Events), len(regen.Events)) {
		if !reflect.DeepEqual(rec.Events[i], regen.Events[i]) {
			return fmt.Errorf("%w at event %d: recorded %s, recomputed %s",
				ErrDiverged, rec.Events[i].Seq, describe(rec.Events[i]), describe(regen.Events[i]))
		}
	}

	if len(rec.Events) != len(regen.Events) {
		return fmt.Errorf("%w after event %d: recorded %d events, recomputed %d",
			ErrDiverged, min(len(rec.Events), len(regen.Events)), len(rec.Events), len(regen.Events))
	}

	recBytes, err := rec.MarshalRecord()
	if err != nil {
		return err
	}

	regenBytes, err := regen.MarshalRecord()
	if err != nil {
		return err
	}

	if !slices.Equal(recBytes, regenBytes) {
		return fmt.Errorf("%w: events match but the derived record differs (record altered outside the engine?)", ErrDiverged)
	}

	return nil
}

func describe(ev Event) string {
	switch ev.Kind {
	case "throw":
		return fmt.Sprintf("throw %s %v total %d", ev.Label, ev.Dice, ev.Total)
	case "choice":
		return fmt.Sprintf("choice %s = %s", ev.Label, ev.Picked)
	default:
		return fmt.Sprintf("%s %q", ev.Kind, ev.Text)
	}
}
