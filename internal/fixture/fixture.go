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
	Decider chargen.Decider
}

// All is the golden set: one careerist per service (forced with the
// --service flag's own mechanism), a draftee into a commissioned service
// so the first-term commission bar is exercised (p. 5), a death, a
// civilian who declined the draft, and the milestone 3 paths — a
// hereditary title assumed, a medical-crisis death (E006/E007), a scout
// ship in constructive possession, and a twice-received Free Trader.
//
// A fresh slice each call: callers are tests, and one filtering or
// reordering its copy must not reach the others.
func All() []Fixture {
	return []Fixture{
		{Name: "navy-careerist", Seed: 3, Service: "navy", Auto: true, Decider: chargen.AutoPolicy{}},
		{Name: "marines-careerist", Seed: 8, Service: "marines", Auto: true, Decider: chargen.AutoPolicy{}},
		{Name: "army-careerist", Seed: 2, Service: "army", Auto: true, Decider: chargen.AutoPolicy{}},
		{Name: "scouts-careerist", Seed: 34, Service: "scouts", Auto: true, Decider: chargen.AutoPolicy{}},
		{Name: "merchants-careerist", Seed: 2, Service: "merchants", Auto: true, Decider: chargen.AutoPolicy{}},
		{Name: "other-careerist", Seed: 3, Service: "other", Auto: true, Decider: chargen.AutoPolicy{}},
		{Name: "draftee", Seed: 7, Auto: true, Decider: chargen.AutoPolicy{}},
		{Name: "death-in-service", Seed: 2, Auto: true, Decider: chargen.AutoPolicy{}},
		{Name: "civilian-declined-draft", Seed: 1, Auto: false, Decider: DeclineDecider{}},
		{Name: "duke", Seed: 4, Auto: true, Decider: chargen.AutoPolicy{}},
		{Name: "medical-crisis-death", Seed: 8, Service: "scouts", Auto: true, Decider: chargen.AutoPolicy{}},
		{Name: "scout-ship", Seed: 46, Service: "scouts", Auto: true, Decider: chargen.AutoPolicy{}},
		{Name: "free-trader", Seed: 145, Service: "merchants", Auto: true, Decider: chargen.AutoPolicy{}},
	}
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
