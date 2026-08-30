package chargen

import (
	"errors"
	"fmt"
	"slices"
	"strings"
)

// Errors the procedure reports.
var (
	// ErrCannotDecide is a Decider presented with a choice point it has no
	// rule for.
	ErrCannotDecide = errors.New("cannot decide")
	// ErrBadDecision is a Decider answer outside the choice's options, or
	// engine state that should be impossible with validated data.
	ErrBadDecision = errors.New("invalid decision")
	// ErrProvenance is a record whose stamps do not match this build
	// (replay --ignore-provenance waives the match).
	ErrProvenance = errors.New("provenance mismatch")
	// ErrDiverged is a replay that did not reproduce the record.
	ErrDiverged = errors.New("replay diverged")
)

// Choice labels: the procedure's choice points (FR10 records who decided
// and what the alternatives were).
const (
	ChoiceService       = "service"
	ChoiceSubmitToDraft = "submit-to-draft"
	ChoiceCommission    = "commission-attempt"
	ChoicePromotion     = "promotion-attempt"
	ChoiceSkillTable    = "skill-table"
	ChoiceWeapon        = "weapon"
	ChoiceReenlist      = "reenlist-intent"
	ChoiceMusterTable   = "muster-table"
	ChoiceBenefitDM     = "benefit-dm"
	ChoiceCashDM        = "cash-dm"
	ChoiceMusterWeapon  = "muster-weapon"
	ChoiceTitle         = "assume-title"
)

// ExpertisePrefix marks a muster-weapon option that takes +1 expertise in
// an already-received benefit weapon in lieu of another weapon (p. 22).
const ExpertisePrefix = "expertise: "

// Yes and No are the picks for the boolean choice points.
const (
	Yes = "yes"
	No  = "no"
)

// Who made a decision, recorded in the event's "by" field.
const (
	ByPolicy = "policy"
	ByPlayer = "player"
)

// Choice is one choice point put to a Decider. Options always lists every
// valid pick; Category is set for weapon picks ("blade" or "gun").
type Choice struct {
	Step     string
	Label    string
	Category string
	Options  []string
}

// Decision is a Decider's answer: the pick, and who made it.
type Decision struct {
	Pick string
	By   string
}

// Decider answers every choice point of the procedure. The auto policy,
// the interactive prompter (milestone 4), and the replay reapplier all
// implement it, which is what lets one engine serve all three
// (docs/PRD.md, Architecture notes).
type Decider interface {
	Decide(c Choice) (Decision, error)
}

// AutoPolicy is the fixed default policy, docs/POLICY.md version 1: total (it
// can decide every valid choice point) and deterministic, tie-breaking by
// first-listed order in Book 1.
type AutoPolicy struct{}

// Decide applies docs/POLICY.md v1.
func (AutoPolicy) Decide(c Choice) (Decision, error) {
	var pick string

	switch c.Label {
	case ChoiceService, ChoiceWeapon:
		// First-listed in Book 1: services in the order of p. 5, weapons
		// in the order of the pp. 12-13 lists.
		pick = c.Options[0]
	case ChoiceSubmitToDraft, ChoiceCommission, ChoicePromotion, ChoiceReenlist,
		ChoiceBenefitDM, ChoiceCashDM, ChoiceTitle:
		// A rejected character submits to the draft; a serving character
		// attempts every commission and promotion open to him and
		// reenlists while the rules allow it; the optional muster DMs are
		// always taken; an eligible title is assumed.
		pick = Yes
	case ChoiceSkillTable:
		// Every eligibility goes to the service's Service Skills table.
		pick = "service_skills"
	case ChoiceMusterTable:
		// Cash while the three-roll cap allows, then material benefits.
		pick = "benefits"
		if slices.Contains(c.Options, "cash") {
			pick = "cash"
		}
	case ChoiceMusterWeapon:
		// Take +1 expertise in the first-listed already-received weapon
		// when the option exists; otherwise the first-listed weapon.
		pick = c.Options[0]

		for _, option := range c.Options {
			if strings.HasPrefix(option, ExpertisePrefix) {
				pick = option

				break
			}
		}
	default:
		return Decision{}, fmt.Errorf("auto policy v%s %w %q", PolicyVersion, ErrCannotDecide, c.Label)
	}

	return Decision{Pick: pick, By: ByPolicy}, nil
}
