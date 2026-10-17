package docsgate_test

import (
	"encoding/json"
	"slices"
	"strings"
	"testing"

	"github.com/philoserf/ctchargen/chargen"
)

// The schema's three strategy enums name exactly the strategies the code
// takes, in both directions.
//
// The point enum is held through the goldens, because every choice point
// must be reached by one. A strategy has no such chain: one added without a
// golden that uses it would leave the schema refusing records the tool
// writes, with every test green (#177).
func TestTheSchemaNamesEveryStrategy(t *testing.T) {
	t.Parallel()

	var schema struct {
		Properties struct {
			Inputs struct {
				Properties map[string]struct {
					Enum []string `json:"enum"`
				} `json:"properties"`
			} `json:"inputs"`
		} `json:"properties"`
	}

	err := json.Unmarshal([]byte(read(t, "character.schema.json")), &schema)
	if err != nil {
		t.Fatalf("the schema is not JSON: %v", err)
	}

	for field, choices := range map[string]string{
		"career": chargen.CareerChoices(),
		"skills": chargen.SkillsChoices(),
		"muster": chargen.MusterChoices(),
	} {
		inSchema := slices.Sorted(slices.Values(schema.Properties.Inputs.Properties[field].Enum))
		if len(inSchema) == 0 {
			t.Errorf("the schema's inputs.%s has no enum, so this checks nothing", field)
		}

		inCode := slices.Sorted(slices.Values(strings.Split(choices, ", ")))

		compare(t, inCode, inSchema, "--"+field, "the schema's inputs."+field)
	}
}
