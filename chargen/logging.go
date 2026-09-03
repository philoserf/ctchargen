package chargen

import (
	"fmt"
	"slices"

	"github.com/philoserf/ctchargen/traveller"
)

// logging wraps a Decider, writes each answer into the generation record,
// and delegates.
//
// It must implement every method of Decider to compile, which is the whole
// point of the shape: a choice point cannot reach the engine without
// reaching the log. Adding a method to Decider breaks this type until the
// log learns to record it.
type logging struct {
	to  Decider
	by  traveller.DecidedBy
	log *log
}

func names[T any](from []T, name func(T) string) []string {
	out := make([]string, 0, len(from))
	for _, v := range from {
		out = append(out, name(v))
	}

	return out
}

// expertiseIn names the p. 22 alternative, in one place, so that the record
// of what was offered and the record of what was chosen cannot word it
// differently.
func expertiseIn(weapon traveller.WeaponName) string {
	return "expertise in " + string(weapon)
}

func yesNo(b bool) string {
	if b {
		return "yes"
	}

	return "no"
}

// offered is what a yes-or-no choice point puts in front of the decider.
//
//nolint:gochecknoglobals // an immutable pair, and Go has no const slice.
var offered = []string{"yes", "no"}

func (l logging) Service(from []traveller.EnlistmentOffer) (traveller.ServiceName, error) {
	chosen, err := l.to.Service(from)
	if err != nil {
		return chosen, err
	}

	return chosen, l.record(traveller.ChoiceService,
		names(from, func(o traveller.EnlistmentOffer) string { return o.Service.String() }),
		chosen.String())
}

func (l logging) SubmitToDraft() (bool, error) {
	chosen, err := l.to.SubmitToDraft()
	if err != nil {
		return chosen, err
	}

	return chosen, l.record(traveller.ChoiceSubmitToDraft, offered, yesNo(chosen))
}

func (l logging) AttemptCommission() (bool, error) {
	chosen, err := l.to.AttemptCommission()
	if err != nil {
		return chosen, err
	}

	return chosen, l.record(traveller.ChoiceAttemptCommission, offered, yesNo(chosen))
}

func (l logging) AttemptPromotion() (bool, error) {
	chosen, err := l.to.AttemptPromotion()
	if err != nil {
		return chosen, err
	}

	return chosen, l.record(traveller.ChoiceAttemptPromotion, offered, yesNo(chosen))
}

func (l logging) SkillTable(from []traveller.SkillTable) (traveller.SkillTable, error) {
	chosen, err := l.to.SkillTable(from)
	if err != nil {
		return chosen, err
	}

	return chosen, l.record(traveller.ChoiceSkillTable,
		names(from, traveller.SkillTable.String), chosen.String())
}

func (l logging) Weapon(
	category traveller.WeaponCategory, from []traveller.WeaponName,
) (traveller.WeaponName, error) {
	chosen, err := l.to.Weapon(category, from)
	if err != nil {
		return chosen, err
	}

	return chosen, l.record(traveller.ChoiceWeapon,
		names(from, func(w traveller.WeaponName) string { return string(w) }), string(chosen))
}

func (l logging) ReenlistIntent(from []traveller.Intent) (traveller.Intent, error) {
	chosen, err := l.to.ReenlistIntent(from)
	if err != nil {
		return chosen, err
	}

	return chosen, l.record(traveller.ChoiceReenlistIntent, names(from, traveller.Intent.String), chosen.String())
}

func (l logging) MusterTable(from []traveller.MusterTable) (traveller.MusterTable, error) {
	chosen, err := l.to.MusterTable(from)
	if err != nil {
		return chosen, err
	}

	return chosen, l.record(traveller.ChoiceMusterTable, names(from, traveller.MusterTable.String), chosen.String())
}

func (l logging) MusterTable1DM() (bool, error) {
	chosen, err := l.to.MusterTable1DM()
	if err != nil {
		return chosen, err
	}

	return chosen, l.record(traveller.ChoiceMusterTable1DM, offered, yesNo(chosen))
}

func (l logging) MusterTable2DM() (bool, error) {
	chosen, err := l.to.MusterTable2DM()
	if err != nil {
		return chosen, err
	}

	return chosen, l.record(traveller.ChoiceMusterTable2DM, offered, yesNo(chosen))
}

func (l logging) MusterWeapon(
	category traveller.WeaponCategory, from, received []traveller.WeaponName,
) (traveller.WeaponBenefit, error) {
	chosen, err := l.to.MusterWeapon(category, from, received)
	if err != nil {
		return chosen, err
	}

	var described describeWeaponBenefit

	err = chosen.Fold(&described)
	if err != nil {
		return chosen, fmt.Errorf("describing the weapon benefit: %w", err)
	}

	// P. 22 offers the expertise beside the weapons, so the record of what
	// was offered says so - and the gate has the whole offer to check
	// against. Both halves are worded by the same functions that word the
	// answer, so the comparison is like with like.
	alternatives := names(from, func(w traveller.WeaponName) string { return string(w) })
	for _, held := range received {
		alternatives = append(alternatives, expertiseIn(held))
	}

	return chosen, l.record(traveller.ChoiceMusterWeapon, alternatives, described.text)
}

func (l logging) AssumeTitle(title traveller.Title) (bool, error) {
	chosen, err := l.to.AssumeTitle(title)
	if err != nil {
		return chosen, err
	}

	return chosen, l.record(traveller.ChoiceAssumeTitle, offered, yesNo(chosen))
}

// describeWeaponBenefit renders what was taken for a weapon row, by folding,
// so a third way of taking one cannot be added without this seeing it.
type describeWeaponBenefit struct{ text string }

func (d *describeWeaponBenefit) TakeWeapon(weapon traveller.WeaponName) error {
	d.text = string(weapon)

	return nil
}

func (d *describeWeaponBenefit) TakeExpertise(weapon traveller.WeaponName) error {
	d.text = expertiseIn(weapon)

	return nil
}

// record writes the answer into the generation record, and refuses one that
// was not offered.
//
// It is the single gate: the decorator is the only way a decider reaches the
// engine (generate.go wraps every one of them), and it already holds both
// the offered set and the answer, because it records both. Checking here
// rather than at each application site leaves one definition of answering
// outside the offer, which two cannot disagree with.
func (l logging) record(point traveller.ChoicePoint, from []string, chosen string) error {
	if !slices.Contains(from, chosen) {
		return fmt.Errorf("%w: %v offered %v, and was answered %q",
			errNotOffered, point, from, chosen)
	}

	l.log.choice(point, l.by, from, chosen)

	return nil
}
