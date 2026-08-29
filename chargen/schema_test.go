package chargen_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/philoserf/ctchargen/chargen"
)

// The schema is documentation of what the engine writes; this pins it to
// the structs so neither drifts. A field added to a struct without a
// schema property (or vice versa) fails here.
func TestSchemaMatchesStructs(t *testing.T) {
	raw, err := os.ReadFile("../docs/character.schema.json")
	if err != nil {
		t.Fatal(err)
	}

	var schema map[string]any
	if err := json.Unmarshal(raw, &schema); err != nil {
		t.Fatal(err)
	}

	root := properties(t, schema)

	checkKeys(t, "root", root, reflect.TypeFor[chargen.Character]())
	checkKeys(t, "rng", properties(t, child(t, root, "rng")), reflect.TypeFor[chargen.RNG]())
	checkKeys(t, "inputs", properties(t, child(t, root, "inputs")), reflect.TypeFor[chargen.Inputs]())
	characteristics := properties(t, child(t, root, "characteristics"))
	checkKeys(t, "characteristics", characteristics, reflect.TypeFor[chargen.Characteristics]())
	checkKeys(t, "skills item", properties(t, items(t, child(t, root, "skills"))), reflect.TypeFor[chargen.Skill]())
	checkKeys(t, "death", properties(t, child(t, root, "death")), reflect.TypeFor[chargen.Death]())

	benefits := properties(t, child(t, root, "benefits"))
	checkKeys(t, "benefits", benefits, reflect.TypeFor[chargen.Benefits]())
	checkKeys(t, "passages", properties(t, child(t, benefits, "passages")), reflect.TypeFor[chargen.Passages]())
	checkKeys(t, "ship", properties(t, child(t, benefits, "ship")), reflect.TypeFor[chargen.Ship]())
	checkKeys(t, "title", properties(t, child(t, root, "title")), reflect.TypeFor[chargen.Title]())

	event := properties(t, items(t, child(t, root, "events")))
	checkKeys(t, "events item", event, reflect.TypeFor[chargen.Event]())
	checkKeys(t, "dms item", properties(t, items(t, child(t, event, "dms"))), reflect.TypeFor[chargen.EventDM]())
}

// The examples beside the schema are engine output; they must parse
// strictly.
func TestSchemaExamplesParse(t *testing.T) {
	for _, path := range []string{"../docs/character.minimal.json", "../docs/character.complete.json"} {
		raw, err := os.ReadFile(filepath.Clean(path))
		if err != nil {
			t.Fatal(err)
		}

		if _, err := chargen.UnmarshalRecord(raw); err != nil {
			t.Errorf("%s: %v", path, err)
		}
	}
}

func properties(t *testing.T, node map[string]any) map[string]any {
	t.Helper()

	props, ok := node["properties"].(map[string]any)
	if !ok {
		t.Fatalf("node has no properties object: %v", node)
	}

	return props
}

func child(t *testing.T, props map[string]any, name string) map[string]any {
	t.Helper()

	node, ok := props[name].(map[string]any)
	if !ok {
		t.Fatalf("schema has no object property %q", name)
	}

	return node
}

func items(t *testing.T, node map[string]any) map[string]any {
	t.Helper()

	item, ok := node["items"].(map[string]any)
	if !ok {
		t.Fatalf("array schema has no items object: %v", node)
	}

	return item
}

func checkKeys(t *testing.T, label string, props map[string]any, typ reflect.Type) {
	t.Helper()

	tags := jsonTags(typ)
	for _, tag := range tags {
		if _, ok := props[tag]; !ok {
			t.Errorf("%s: struct field %q missing from schema", label, tag)
		}
	}

	for name := range props {
		if !slices.Contains(tags, name) {
			t.Errorf("%s: schema property %q has no struct field", label, name)
		}
	}
}

func jsonTags(typ reflect.Type) []string {
	var tags []string

	for field := range typ.Fields() {
		tag := field.Tag.Get("json")
		if tag == "" || tag == "-" {
			continue
		}

		tags = append(tags, strings.Split(tag, ",")[0])
	}

	return tags
}
