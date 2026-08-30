package chargen_test

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/philoserf/ctchargen/internal/fixture"
)

// TestSchemaMatchesStructs pins property names to struct fields. It does
// not read the schema's constraints, so the engine could — and did —
// write a null where the schema declares an array, or a zero where it
// declares a minimum of one, with every test green. This validates
// records against what the schema actually says.
//
// It understands only the constructs docs/character.schema.json uses:
// type, const, enum, pattern, required, additionalProperties, minimum,
// maximum, items, properties, and a local $ref. Anything else fails
// loudly rather than passing silently, so the schema cannot outgrow the
// checker unnoticed.
func TestRecordsConformToSchema(t *testing.T) {
	schema := loadSchema(t)

	// The golden roster, not a seed list of its own. A bare list was
	// transcribed here once and diverged silently: it carried the
	// fixtures' seeds without their Service inputs, so seed 46 generated a
	// Navy term-1 death rather than scout-ship, and between them the five
	// records held no ship, no weapon benefit, no Travellers' Aid, and no
	// title — leaving the schema's title object checked against no record
	// anywhere. Iterating internal/fixture is the same move golden_test.go
	// and render's goldens make, and for the same reason.
	//
	// One shape still reaches the schema through nothing here: a non-zero
	// age_months, which only a medical-crisis survivor accrues (1D months,
	// pp. 7-8) and the one crisis fixture is a death. The engine side of
	// that branch is covered by TestMedicalCrisis in the package's internal
	// tests; the schema's `maximum: 11` on the field is not, and would need
	// a survivor golden.
	t.Run("generated records", func(t *testing.T) {
		for _, f := range fixture.All() {
			t.Run(f.Name, func(t *testing.T) {
				char := generate(t, f)

				record, err := char.MarshalRecord()
				if err != nil {
					t.Fatal(err)
				}

				for _, problem := range conforms(t, schema, record) {
					t.Error(problem)
				}
			})
		}
	})

	// The examples beside the schema are hand-maintained, so nothing
	// otherwise stops them drifting from what the engine writes.
	t.Run("documented examples", func(t *testing.T) {
		for _, path := range []string{"../docs/character.minimal.json", "../docs/character.complete.json"} {
			raw, err := os.ReadFile(filepath.Clean(path))
			if err != nil {
				t.Fatal(err)
			}

			for _, problem := range conforms(t, schema, raw) {
				t.Errorf("%s: %s", path, problem)
			}
		}
	})
}

func loadSchema(t *testing.T) map[string]any {
	t.Helper()

	raw, err := os.ReadFile("../docs/character.schema.json")
	if err != nil {
		t.Fatal(err)
	}

	var schema map[string]any
	if err := json.Unmarshal(raw, &schema); err != nil {
		t.Fatal(err)
	}

	return schema
}

func conforms(t *testing.T, schema map[string]any, record []byte) []string {
	t.Helper()

	var value any
	if err := json.Unmarshal(record, &value); err != nil {
		t.Fatal(err)
	}

	v := &validator{t: t, schema: schema}
	v.check(schema, value, "")

	return v.problems
}

type validator struct {
	t        *testing.T
	schema   map[string]any
	problems []string
}

// known is every keyword this checker implements. A schema node using
// anything else is a test failure, not a silent pass.
var known = map[string]bool{
	"$schema": true, "$id": true, "$defs": true, "$ref": true,
	"title": true, "description": true,
	"type": true, "const": true, "enum": true, "pattern": true,
	"required": true, "additionalProperties": true,
	"minimum": true, "maximum": true, "items": true, "properties": true,
}

func (v *validator) check(node map[string]any, value any, path string) {
	node = v.resolve(node, path)

	for keyword := range node {
		if !known[keyword] {
			v.t.Fatalf("%s: schema uses %q, which this checker does not implement", path, keyword)
		}
	}

	v.checkType(node, value, path)
	v.checkConst(node, value, path)
	v.checkEnum(node, value, path)
	v.checkPattern(node, value, path)
	v.checkBounds(node, value, path)

	switch typed := value.(type) {
	case map[string]any:
		v.checkObject(node, typed, path)
	case []any:
		v.checkArray(node, typed, path)
	}
}

func (v *validator) resolve(node map[string]any, path string) map[string]any {
	ref, ok := node["$ref"].(string)
	if !ok {
		return node
	}

	name, found := strings.CutPrefix(ref, "#/$defs/")
	if !found {
		v.t.Fatalf("%s: only local $defs references are supported, got %q", path, ref)
	}

	defs, ok := v.schema["$defs"].(map[string]any)
	if !ok {
		v.t.Fatalf("%s: $ref %q but the schema has no $defs", path, ref)
	}

	target, ok := defs[name].(map[string]any)
	if !ok {
		v.t.Fatalf("%s: $ref %q does not resolve", path, ref)
	}

	return target
}

func (v *validator) checkType(node map[string]any, value any, path string) {
	declared, ok := node["type"].(string)
	if !ok {
		return
	}

	if !matchesType(declared, value) {
		v.reportf("%s: schema says %s, record has %s", path, declared, describeJSON(value))
	}
}

func matchesType(declared string, value any) bool {
	switch declared {
	case "object":
		_, ok := value.(map[string]any)

		return ok
	case "array":
		_, ok := value.([]any)

		return ok
	case "string":
		_, ok := value.(string)

		return ok
	case "boolean":
		_, ok := value.(bool)

		return ok
	case "integer", "number":
		number, ok := value.(float64)

		return ok && (declared == "number" || number == float64(int64(number)))
	case "null":
		return value == nil
	}

	return false
}

func (v *validator) checkConst(node map[string]any, value any, path string) {
	if want, ok := node["const"]; ok && value != want {
		v.reportf("%s: schema requires %v, record has %v", path, want, value)
	}
}

func (v *validator) checkEnum(node map[string]any, value any, path string) {
	allowed, ok := node["enum"].([]any)
	if !ok {
		return
	}

	if slices.Contains(allowed, value) {
		return
	}

	v.reportf("%s: schema allows %v, record has %v", path, allowed, value)
}

// Go's regexp is RE2 rather than ECMA-262, which JSON Schema specifies.
// The patterns in this schema are plain anchored character classes, so
// the two agree; a pattern needing ECMA-only syntax would fail to compile
// here and say so.
func (v *validator) checkPattern(node map[string]any, value any, path string) {
	pattern, ok := node["pattern"].(string)
	if !ok {
		return
	}

	text, ok := value.(string)
	if !ok {
		return
	}

	matched, err := regexp.MatchString(pattern, text)
	if err != nil {
		v.t.Fatalf("%s: schema pattern %q does not compile: %v", path, pattern, err)
	}

	if !matched {
		v.reportf("%s: schema pattern %q, record has %q", path, pattern, text)
	}
}

func (v *validator) checkBounds(node map[string]any, value any, path string) {
	number, ok := value.(float64)
	if !ok {
		return
	}

	if low, ok := node["minimum"].(float64); ok && number < low {
		v.reportf("%s: schema minimum %g, record has %g", path, low, number)
	}

	if high, ok := node["maximum"].(float64); ok && number > high {
		v.reportf("%s: schema maximum %g, record has %g", path, high, number)
	}
}

func (v *validator) checkObject(node, value map[string]any, path string) {
	props, _ := node["properties"].(map[string]any)

	for _, name := range stringsOf(node["required"]) {
		if _, present := value[name]; !present {
			v.reportf("%s.%s: required by the schema, absent from the record", path, name)
		}
	}

	if allowed, ok := node["additionalProperties"].(bool); ok && !allowed {
		for name := range value {
			if _, declared := props[name]; !declared {
				v.reportf("%s.%s: not declared, and the schema forbids extras", path, name)
			}
		}
	}

	for name, child := range value {
		sub, ok := props[name].(map[string]any)
		if !ok {
			continue
		}

		v.check(sub, child, path+"."+name)
	}
}

func (v *validator) checkArray(node map[string]any, value []any, path string) {
	sub, ok := node["items"].(map[string]any)
	if !ok {
		return
	}

	for i, element := range value {
		v.check(sub, element, fmt.Sprintf("%s[%d]", path, i))
	}
}

func (v *validator) reportf(format string, args ...any) {
	v.problems = append(v.problems, fmt.Sprintf(format, args...))
}

func stringsOf(value any) []string {
	raw, ok := value.([]any)
	if !ok {
		return nil
	}

	out := make([]string, 0, len(raw))

	for _, item := range raw {
		if s, ok := item.(string); ok {
			out = append(out, s)
		}
	}

	return out
}

func describeJSON(value any) string {
	switch value.(type) {
	case nil:
		return "null"
	case map[string]any:
		return "object"
	case []any:
		return "array"
	case string:
		return "string"
	case bool:
		return "boolean"
	case float64:
		return "number"
	}

	return "unknown"
}
