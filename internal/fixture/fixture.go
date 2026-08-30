// Package fixture is the one definition of the golden-fixture set: the
// characters the chargen and render packages both compare their output
// against, byte for byte.
//
// It exists because the set was previously transcribed by hand in each
// of those two test packages. Nothing detected drift between the copies:
// change a seed in one and both suites still pass — each matching its own
// testdata — while the two trees describe different people under the same
// fixture name, and `task goldens` writes both. The set is data two
// packages must agree on, so it gets one home, the way service.Gambling
// and the ship-class constants do.
//
// Test-only, but a regular package rather than a _test.go file: an
// external test package cannot import identifiers from another package's
// tests. Nothing in the command imports it, so it is not linked into the
// binary.
//
// Only the external test packages (chargen_test, render_test) may import
// it. This package imports chargen, so chargen's own package-internal
// tests — charts_internal_test.go, muster_internal_test.go, which are
// package chargen — would close a cycle. That fails at compile time
// rather than silently, but the shape is worth knowing before reaching
// for a fixture in one of them.
package fixture

import (
	"fmt"

	"github.com/philoserf/ctchargen/chargen"
)

// Fixture is one golden character: the inputs that generate it, and the
// Decider that answers its choice points.
type Fixture struct {
	Name    string
	Seed    uint64
	Service string
	Auto    bool

	// The auto policy's selectable rows (docs/POLICY.md); zero values are
	// the defaults.
	Skills      string
	Muster      string
	CareerTerms int

	// Decider answers the fixture's choice points. All fills it from the
	// fixture's own Config unless the roster set one, which only the
	// civilian does — so a fixture's strategy fields and the policy that
	// actually decides cannot drift apart.
	Decider chargen.Decider
}

// Config is the fixture's inputs as the engine takes them. One
// construction, used by every golden suite, so a fixture cannot come to
// mean different things in the chargen and render trees.
func (f Fixture) Config() chargen.Config {
	return chargen.Config{
		Seed: f.Seed, Service: f.Service, Auto: f.Auto,
		Skills: f.Skills, Muster: f.Muster, CareerTerms: f.CareerTerms,
	}
}

// All is the golden set: one careerist per service (forced with the
// --service flag's own mechanism), a draftee into a commissioned service
// so the first-term commission bar is exercised (p. 5), a death, a
// civilian who declined the draft, and the milestone 3 paths — a
// hereditary title assumed, a medical-crisis death (E006/E007), a scout
// ship in constructive possession, and a twice-received Free Trader.
//
// The crisis appears twice on purpose, because its two branches produce
// different records and the death alone left the other unwritten: the
// survivor recovers to 1 and pays 1D months of age (pp. 7-8), so it is the
// only fixture carrying a non-zero age_months — the one thing that puts
// the schema's 0-11 bound on that field in front of a real record.
//
// A fresh slice each call: callers are tests, and one filtering or
// reordering its copy must not reach the others.
func All() []Fixture {
	set := []Fixture{
		{Name: "navy-careerist", Seed: 3, Service: "navy", Auto: true},
		{Name: "marines-careerist", Seed: 8, Service: "marines", Auto: true},
		{Name: "army-careerist", Seed: 2, Service: "army", Auto: true},
		{Name: "scouts-careerist", Seed: 34, Service: "scouts", Auto: true},
		{Name: "merchants-careerist", Seed: 2, Service: "merchants", Auto: true},
		{Name: "other-careerist", Seed: 3, Service: "other", Auto: true},
		{Name: "draftee", Seed: 7, Auto: true},
		{Name: "death-in-service", Seed: 2, Auto: true},
		{Name: "civilian-declined-draft", Seed: 1, Auto: false, Decider: DeclineDecider{}},
		{Name: "duke", Seed: 4, Auto: true},
		{Name: "medical-crisis-death", Seed: 8, Service: "scouts", Auto: true},
		// Strength to zero at 46, saved, recovered, and 6 months older for
		// it — the top of the 1D, so the record sits at the busy end of the
		// month field rather than at 1.
		{Name: "medical-crisis-survivor", Seed: 231, Service: "navy", Auto: true},
		{Name: "scout-ship", Seed: 46, Service: "scouts", Auto: true},
		{Name: "free-trader", Seed: 145, Service: "merchants", Auto: true},
		// The one fixture built by a non-default policy: a term improving
		// himself, a term learning the trade, a term specialising. It is what
		// puts the selectable rows, and the inputs block that records them,
		// in front of a real record.
		{Name: "rounded-navy", Seed: 3, Service: "navy", Auto: true, Skills: chargen.SkillsRounded},
	}

	for i := range set {
		if set[i].Decider == nil {
			set[i].Decider = chargen.NewAutoPolicy(set[i].Config())
		}
	}

	return set
}

// DeclineDecider plays the one path the auto policy never takes: refusing
// the draft (E001), which produces the civilian fixture. Every other
// choice point is delegated, and the whole run is recorded as the
// player's rather than the policy's, because that is what it is.
type DeclineDecider struct{}

// Decide refuses the draft and otherwise follows the auto policy.
func (DeclineDecider) Decide(c chargen.Choice) (chargen.Decision, error) {
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
