package chargen_test

import (
	"slices"
	"testing"

	"github.com/philoserf/ctchargen/chargen"
)

// offered is a plausible options list for each choice point — what the
// engine would hand the Decider there. Three labels need their real
// values because the policy picks by name rather than by position:
// skill-table names a table, muster-table names a table, and
// muster-weapon looks for the expertise prefix.
func offered(label string) []string {
	switch label {
	case chargen.ChoiceService:
		return []string{"Navy", "Marines", "Army", "Scouts", "Merchants", "Other"}
	case chargen.ChoiceSkillTable:
		return []string{"personal_development", "service_skills", "advanced_education"}
	case chargen.ChoiceMusterTable:
		return []string{"benefits", "cash"}
	case chargen.ChoiceWeapon:
		return []string{"Dagger", "Blade", "Foil"}
	case chargen.ChoiceMusterWeapon:
		return []string{"Dagger", "Blade", chargen.ExpertisePrefix + "Dagger"}
	default:
		return []string{chargen.Yes, chargen.No}
	}
}

// docs/POLICY.md calls the auto policy total — "it can decide every valid
// choice point the engine can present" — and nothing checked it. The
// Decide switch's default branch is loud, but only on a generation that
// actually reaches the new choice point, so a gap could sit unnoticed
// until some seed found it. ChoiceLabels is the registry both this and
// the prompter's wording are held against.
func TestAutoPolicyIsTotal(t *testing.T) {
	for _, label := range chargen.ChoiceLabels() {
		t.Run(label, func(t *testing.T) {
			options := offered(label)

			decision, err := chargen.AutoPolicy{}.Decide(chargen.Choice{
				Step: "test", Label: label, Options: options,
			})
			if err != nil {
				t.Fatalf("auto policy cannot decide %s: %v", label, err)
			}

			// The engine rejects a pick outside the options (chargen.choose),
			// so a policy that answers with something not on offer fails the
			// generation just as surely as one that cannot answer at all.
			if !slices.Contains(options, decision.Pick) {
				t.Errorf("picked %q, which is not among %v", decision.Pick, options)
			}

			if decision.By != chargen.ByPolicy {
				t.Errorf("decided by %q, want %q", decision.By, chargen.ByPolicy)
			}
		})
	}
}

// The list is handed to callers that may append to or write through it —
// service.CharacteristicNames, service.TableNames, Registry.Names, and
// Registry.Weapons all answer the same hazard the same way — so each call
// must yield a fresh one.
func TestChoiceLabelsAreNotShared(t *testing.T) {
	first := chargen.ChoiceLabels()
	first[0] = "rewritten"

	if second := chargen.ChoiceLabels(); second[0] == "rewritten" {
		t.Error("ChoiceLabels handed out the same backing array twice")
	}
}
