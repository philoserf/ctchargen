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

// The three selectable rows of docs/POLICY.md. Each strategy is a pure
// function of the Choice, so the cases hand it the choice the engine
// would: Step carries the term, Options carry what the rules allow —
// including the fourth skills table, which appears only once Education
// has reached 8 (p. 11).
type strategyCase struct {
	name   string
	policy chargen.AutoPolicy
	choice chargen.Choice
	want   string
}

func strategyCases() []strategyCase {
	return append(skillTableCases(), musterAndCareerCases()...)
}

// The skill-table row, the one whose default suppresses two whole tables.
func skillTableCases() []strategyCase {
	threeTables := []string{"personal_development", "service_skills", "advanced_education"}
	fourTables := append(slices.Clone(threeTables), "advanced_education_8")

	return []strategyCase{
		{
			name:   "the default is unchanged by an empty policy",
			choice: chargen.Choice{Step: "term-1", Label: chargen.ChoiceSkillTable, Options: threeTables},
			want:   "service_skills",
		},
		{
			name:   "personal takes the first table",
			policy: chargen.AutoPolicy{Skills: chargen.SkillsPersonal},
			choice: chargen.Choice{Step: "term-2", Label: chargen.ChoiceSkillTable, Options: threeTables},
			want:   "personal_development",
		},
		{
			name:   "advanced takes the fourth table once Education has opened it",
			policy: chargen.AutoPolicy{Skills: chargen.SkillsAdvanced},
			choice: chargen.Choice{Step: "term-2", Label: chargen.ChoiceSkillTable, Options: fourTables},
			want:   "advanced_education_8",
		},
		{
			name:   "advanced settles for the third when it has not",
			policy: chargen.AutoPolicy{Skills: chargen.SkillsAdvanced},
			choice: chargen.Choice{Step: "term-2", Label: chargen.ChoiceSkillTable, Options: threeTables},
			want:   "advanced_education",
		},
		{
			name:   "rounded gives term 1 to personal development",
			policy: chargen.AutoPolicy{Skills: chargen.SkillsRounded},
			choice: chargen.Choice{Step: "term-1", Label: chargen.ChoiceSkillTable, Options: threeTables},
			want:   "personal_development",
		},
		{
			name:   "rounded gives term 3 to advanced education",
			policy: chargen.AutoPolicy{Skills: chargen.SkillsRounded},
			choice: chargen.Choice{Step: "term-3", Label: chargen.ChoiceSkillTable, Options: threeTables},
			want:   "advanced_education",
		},
		{
			name:   "rounded comes back around at term 4",
			policy: chargen.AutoPolicy{Skills: chargen.SkillsRounded},
			choice: chargen.Choice{Step: "term-4", Label: chargen.ChoiceSkillTable, Options: threeTables},
			want:   "personal_development",
		},
	}
}

func musterAndCareerCases() []strategyCase {
	return []strategyCase{
		{
			name:   "benefits never takes the cash table, even while it is offered",
			policy: chargen.AutoPolicy{Muster: chargen.MusterBenefits},
			choice: chargen.Choice{Step: "muster-out", Label: chargen.ChoiceMusterTable, Options: []string{"benefits", "cash"}},
			want:   "benefits",
		},
		{
			name:   "the default takes cash while the cap allows",
			choice: chargen.Choice{Step: "muster-out", Label: chargen.ChoiceMusterTable, Options: []string{"benefits", "cash"}},
			want:   "cash",
		},
		{
			name:   "a career length leaves once its term is reached",
			policy: chargen.AutoPolicy{CareerTerms: 3},
			choice: chargen.Choice{Step: "term-3", Label: chargen.ChoiceReenlist, Options: []string{chargen.Yes, chargen.No}},
			want:   chargen.No,
		},
		{
			name:   "and stays before it",
			policy: chargen.AutoPolicy{CareerTerms: 3},
			choice: chargen.Choice{Step: "term-2", Label: chargen.ChoiceReenlist, Options: []string{chargen.Yes, chargen.No}},
			want:   chargen.Yes,
		},
	}
}

func TestAutoPolicyStrategies(t *testing.T) {
	for _, tt := range strategyCases() {
		t.Run(tt.name, func(t *testing.T) {
			decision, err := tt.policy.Decide(tt.choice)
			if err != nil {
				t.Fatalf("Decide: %v", err)
			}

			if decision.Pick != tt.want {
				t.Errorf("picked %q, want %q", decision.Pick, tt.want)
			}

			if !slices.Contains(tt.choice.Options, decision.Pick) {
				t.Errorf("picked %q, which is not among %v", decision.Pick, tt.choice.Options)
			}
		})
	}
}

// Whatever the strategies, the policy must still answer every choice point
// the engine can present — that is what docs/POLICY.md means by total.
func TestEveryStrategyStaysTotal(t *testing.T) {
	policies := []chargen.AutoPolicy{
		{Skills: chargen.SkillsPersonal},
		{Skills: chargen.SkillsAdvanced},
		{Skills: chargen.SkillsRounded},
		{Muster: chargen.MusterBenefits},
		{CareerTerms: 1},
	}
	for _, policy := range policies {
		for _, label := range chargen.ChoiceLabels() {
			options := offered(label)

			decision, err := policy.Decide(chargen.Choice{Step: "term-2", Label: label, Options: options})
			if err != nil {
				t.Errorf("%+v cannot decide %s: %v", policy, label, err)

				continue
			}

			if !slices.Contains(options, decision.Pick) {
				t.Errorf("%+v picked %q for %s, not among %v", policy, decision.Pick, label, options)
			}
		}
	}
}
