package chargen_test

import (
	"errors"
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

// Decide guards an empty options list because AutoPolicy is exported and
// documented as total, so a caller reaching it directly must get an answer
// or an error rather than an index-out-of-range. A strategy that picked by
// position reopened that hole for every list shorter than the three tables
// — `rounded` on a one-element list panicked — so the guard is checked at
// the short end too, not only at zero.
func TestSkillStrategiesDoNotIndexPastTheOptions(t *testing.T) {
	for _, skills := range chargen.PolicyStrategies()["skills"] {
		t.Run(skills, func(t *testing.T) {
			for _, step := range []string{"term-1", "term-2", "term-3", "muster-out"} {
				if _, err := (chargen.AutoPolicy{Skills: skills}).Decide(chargen.Choice{
					Step: step, Label: chargen.ChoiceSkillTable, Options: []string{"personal_development"},
				}); err != nil {
					t.Errorf("%s: %v", step, err)
				}
			}
		})
	}
}

// The CLI answers a mistyped strategy, but Config is exported and its three
// policy fields are copied verbatim into the record's inputs. Unchecked, an
// unrecognised strategy would be stamped there as the policy that generated
// the character while the default was silently applied, and a career length
// outside 1-7 would write a record docs/character.schema.json rejects.
func TestGenerateRefusesUnrecognisedPolicyInputs(t *testing.T) {
	tests := []struct {
		name string
		cfg  chargen.Config
	}{
		{"an unknown skills strategy", chargen.Config{Seed: 3, Service: "navy", Auto: true, Skills: "bogus"}},
		{"an unknown muster strategy", chargen.Config{Seed: 3, Service: "navy", Auto: true, Muster: "bogus"}},
		{"a career past the 7-term cap", chargen.Config{Seed: 3, Service: "navy", Auto: true, CareerTerms: 99}},
		{"a negative career", chargen.Config{Seed: 3, Service: "navy", Auto: true, CareerTerms: -1}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			char, err := chargen.Generate(tt.cfg, chargen.NewAutoPolicy(tt.cfg))
			if !errors.Is(err, chargen.ErrBadConfig) {
				t.Fatalf("Generate = %v, want %v", err, chargen.ErrBadConfig)
			}

			if char != nil {
				t.Error("a refused config still produced a record")
			}
		})
	}
}

// The defaults must stay reachable through the same door: an empty
// strategy and a zero career are what every existing caller passes.
func TestGenerateAcceptsTheDefaultsAndEveryStrategy(t *testing.T) {
	cfgs := []chargen.Config{{Seed: 3, Service: "navy", Auto: true}}
	for _, skills := range chargen.PolicyStrategies()["skills"] {
		cfgs = append(cfgs, chargen.Config{Seed: 3, Service: "navy", Auto: true, Skills: skills})
	}

	for _, muster := range chargen.PolicyStrategies()["muster"] {
		cfgs = append(cfgs, chargen.Config{Seed: 3, Service: "navy", Auto: true, Muster: muster})
	}

	for term := 1; term <= 7; term++ {
		cfgs = append(cfgs, chargen.Config{Seed: 3, Service: "navy", Auto: true, CareerTerms: term})
	}

	for _, cfg := range cfgs {
		if _, err := chargen.Generate(cfg, chargen.NewAutoPolicy(cfg)); err != nil {
			t.Errorf("Generate(%+v): %v", cfg, err)
		}
	}
}
