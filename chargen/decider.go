package chargen

import (
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"
)

// Errors this package reports: the procedure's own, and the record
// decoder's.
var (
	// ErrCannotDecide is a Decider presented with a choice point it has no
	// rule for.
	ErrCannotDecide = errors.New("cannot decide")
	// ErrBadDecision is a Decider answer outside the choice's options, or
	// engine state that should be impossible with validated data.
	ErrBadDecision = errors.New("invalid decision")
	// ErrInvalidChart is a broken embedded chart: a build defect in the
	// aging or nobility data, surfaced at load rather than at some later
	// roll (service.ErrInvalidData is its counterpart for the service
	// tables).
	ErrInvalidChart = errors.New("invalid chart data")
	// ErrProvenance is a record whose stamps do not match this build
	// (replay --ignore-provenance waives the match).
	ErrProvenance = errors.New("provenance mismatch")
	// ErrDiverged is a replay that did not reproduce the record.
	ErrDiverged = errors.New("replay diverged")
	// ErrTrailingData is a record file holding more than the one record
	// (UnmarshalRecord).
	ErrTrailingData = errors.New("trailing data after the record")
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

// ChoiceLabels is every choice point the procedure can present, in the
// order the procedure reaches them. Two things dispatch on the label and
// would otherwise degrade quietly when a thirteenth is added: the auto
// policy's table, which docs/POLICY.md calls total, and the prompter's
// wording, which falls back to showing the player the bare label. Both
// are tested against this list, so the list is the one place a new
// choice point has to be registered.
//
// A fresh slice each call, like Registry.Names and Registry.Weapons: a
// caller that appends to what it is handed must not reach the package's
// own copy.
func ChoiceLabels() []string {
	return []string{
		ChoiceService,
		ChoiceSubmitToDraft,
		ChoiceCommission,
		ChoicePromotion,
		ChoiceSkillTable,
		ChoiceWeapon,
		ChoiceReenlist,
		ChoiceMusterTable,
		ChoiceBenefitDM,
		ChoiceCashDM,
		ChoiceMusterWeapon,
		ChoiceTitle,
	}
}

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

// Strategy names for the three configurable rows of docs/POLICY.md. The
// empty string is the default in every case, so a zero AutoPolicy is the
// policy this tool has always applied.
const (
	SkillsService  = "service"  // every eligibility to the Service Skills table
	SkillsPersonal = "personal" // every eligibility to Personal Development
	SkillsAdvanced = "advanced" // the most advanced table on offer
	SkillsRounded  = "rounded"  // one term on each of the first three, in book order

	MusterCash     = "cash"     // cash while the three-roll cap allows
	MusterBenefits = "benefits" // material benefits only; the character musters out with nothing
)

// PolicyStrategies is every flag the auto policy accepts and the values it
// accepts for it, in the order docs/POLICY.md lists them. It is the one
// registry: the CLI validates against it and TestPolicyStrategiesAreDocumented
// holds the document to it, the way ChoiceLabels serves the decision table
// and the prompter's wording.
//
// A fresh map each call, for the reason Registry.Names and ChoiceLabels
// hand out fresh slices.
func PolicyStrategies() map[string][]string {
	return map[string][]string{
		"skills": {SkillsService, SkillsPersonal, SkillsAdvanced, SkillsRounded},
		"muster": {MusterCash, MusterBenefits},
	}
}

// AutoPolicy is the default policy of docs/POLICY.md: total (it can decide
// every valid choice point) and deterministic, tie-breaking by first-listed
// order in Book 1. PolicyVersion is the one place the version lives; a copy
// here would only go stale.
//
// Three rows are selectable. Every strategy is a pure function of the
// Choice it is handed — Step carries the term ("term-4"), and Options
// carries what the rules currently allow, so the fourth skills table
// appears only once Education has opened it and "cash" disappears once the
// three-roll cap is spent. That keeps the policy free of per-character
// state, which is what lets it stay a value type: a zero AutoPolicy is
// still the default policy, and every caller that writes AutoPolicy{} is
// unaffected.
type AutoPolicy struct {
	// Skills selects the skill-table row; "" is SkillsService.
	Skills string
	// Muster selects the muster-table row; "" is MusterCash.
	Muster string
	// CareerTerms is the term the character intends to leave after; 0
	// serves while the rules allow. Intent only — the reenlistment throw
	// is still required every term (p. 6) and a 12 still overrides it
	// (pp. 6-7).
	CareerTerms int
}

// NewAutoPolicy builds the policy a Config asks for. The Config is the one
// source, so a record cannot come to name a policy the decider did not
// actually apply.
func NewAutoPolicy(cfg Config) AutoPolicy {
	return AutoPolicy{Skills: cfg.Skills, Muster: cfg.Muster, CareerTerms: cfg.CareerTerms}
}

// Decide applies the docs/POLICY.md decision table.
func (p AutoPolicy) Decide(c Choice) (Decision, error) {
	// The engine guards this too (chargen.choose), but AutoPolicy is
	// exported and documented as total, so a caller reaching it directly
	// gets an error rather than the index-out-of-range the picks below
	// would otherwise raise.
	if len(c.Options) == 0 {
		return Decision{}, fmt.Errorf("auto policy v%s %w %q: no options offered",
			PolicyVersion, ErrCannotDecide, c.Label)
	}

	var pick string

	switch c.Label {
	case ChoiceService, ChoiceWeapon:
		// First-listed in Book 1: services in the order of p. 5, weapons
		// in the order of the pp. 12-13 lists.
		pick = c.Options[0]
	case ChoiceSubmitToDraft, ChoiceCommission, ChoicePromotion,
		ChoiceBenefitDM, ChoiceCashDM, ChoiceTitle:
		// A rejected character submits to the draft; a serving character
		// attempts every commission and promotion open to him; the optional
		// muster DMs are always taken; an eligible title is assumed.
		pick = Yes
	case ChoiceReenlist:
		pick = p.reenlistPick(c)
	case ChoiceSkillTable:
		pick = p.skillTablePick(c)
	case ChoiceMusterTable:
		pick = p.musterTablePick(c)
	case ChoiceMusterWeapon:
		pick = musterWeaponPick(c.Options)
	default:
		return Decision{}, fmt.Errorf("auto policy v%s %w %q", PolicyVersion, ErrCannotDecide, c.Label)
	}

	return Decision{Pick: pick, By: ByPolicy}, nil
}

// termOf reads the term number out of a choice's step ("term-4"). It
// reports 0 for the steps that are not a term — enlistment, muster-out —
// where no strategy keyed on the term applies.
func termOf(step string) int {
	digits, found := strings.CutPrefix(step, "term-")
	if !found {
		return 0
	}

	term, err := strconv.Atoi(digits)
	if err != nil {
		return 0
	}

	return term
}

// reenlistPick answers the reenlistment intent. CareerTerms is the term the
// character means to leave after; the throw still decides whether he may,
// and a 12 still overrides him (pp. 6-7), so this is intent and nothing
// more.
func (p AutoPolicy) reenlistPick(c Choice) string {
	term := termOf(c.Step)
	if p.CareerTerms > 0 && term >= p.CareerTerms {
		return No
	}

	return Yes
}

// skillTablePick allocates one eligibility. Options holds the tables the
// character may use: three, or four once Education has reached 8 (p. 11),
// which is what lets "advanced" reach for the fourth without being told
// the character's Education.
func (p AutoPolicy) skillTablePick(c Choice) string {
	switch p.Skills {
	case SkillsPersonal:
		return c.Options[0]
	case SkillsAdvanced:
		// The last on offer is the most advanced the character may use.
		return c.Options[len(c.Options)-1]
	case SkillsRounded:
		// One term on each of the first three, in the book's order: a term
		// improving himself, a term learning the trade, a term specialising.
		// Every eligibility of a term goes to that term's table.
		term := termOf(c.Step)
		if term < 1 {
			return c.Options[0]
		}

		return c.Options[(term-1)%3]
	default:
		return "service_skills"
	}
}

// musterTablePick splits the muster rolls between the two tables. Cash
// leaves the options once the three-roll cap is spent (pp. 7, 9).
func (p AutoPolicy) musterTablePick(c Choice) string {
	if p.Muster == MusterBenefits {
		return "benefits"
	}

	if slices.Contains(c.Options, "cash") {
		return "cash"
	}

	return "benefits"
}

// musterWeaponPick takes +1 expertise in the first-listed already-received
// benefit weapon when that option is on offer, and otherwise the
// first-listed weapon of the category (docs/POLICY.md, muster-weapon).
// Options is non-empty: Decide checks before dispatching.
func musterWeaponPick(options []string) string {
	for _, option := range options {
		if strings.HasPrefix(option, ExpertisePrefix) {
			return option
		}
	}

	return options[0]
}
