package chargen_test

import (
	"testing"

	"github.com/philoserf/ctchargen/chargen"
	"github.com/philoserf/ctchargen/service"
)

func TestUPP(t *testing.T) {
	// The book's own example: 779C99 (p. 25).
	c := chargen.Characteristics{
		Strength: 7, Dexterity: 7, Endurance: 9,
		Intelligence: 12, Education: 9, SocialStanding: 9,
	}
	if got := c.UPP(); got != "779C99" {
		t.Errorf("UPP() = %q, want 779C99 (p. 25)", got)
	}

	high := chargen.Characteristics{
		Strength: 10, Dexterity: 11, Endurance: 12,
		Intelligence: 13, Education: 14, SocialStanding: 15,
	}
	if got := high.UPP(); got != "ABCDEF" {
		t.Errorf("UPP() = %q, want ABCDEF (10-15 are A-F, p. 8)", got)
	}
}

// Values never exceed 15 and do not go below 1 through play of the
// procedure (p. 4).
func TestApplyClamps(t *testing.T) {
	c := chargen.Characteristics{Strength: 15, SocialStanding: 1}

	if _, after, _ := c.Apply("strength", 1); after != 15 {
		t.Errorf("Apply(+1) at 15 = %d, want clamp at 15 (p. 4)", after)
	}

	if _, after, _ := c.Apply("social_standing", -1); after != 1 {
		t.Errorf("Apply(-1) at 1 = %d, want clamp at 1 (p. 4)", after)
	}
}

// An unrecognised name must alter nothing and say so: the callers write
// the returned before/after into the event log, so a silent miss would
// put an alteration in the record that never happened.
func TestApplyRejectsUnknownCharacteristic(t *testing.T) {
	c := chargen.Characteristics{Strength: 7}

	before, after, ok := c.Apply("stength", 1) // deliberate typo
	if ok {
		t.Error("Apply reported a typo'd name as applied")
	}

	if before != 0 || after != 0 {
		t.Errorf("Apply on an unknown name = (%d, %d), want (0, 0)", before, after)
	}

	if c.Strength != 7 {
		t.Errorf("Apply on an unknown name altered strength to %d, want 7", c.Strength)
	}
}

// Every name the service data can carry must round-trip, so the record's
// six fields cannot drift out of step with service.CharacteristicNames.
func TestApplyHandlesEveryCharacteristicName(t *testing.T) {
	for i, name := range service.CharacteristicNames {
		var c chargen.Characteristics

		want := i + 1
		if _, after, ok := c.Apply(name, want); !ok || after != want {
			t.Errorf("Apply(%q, %d) = (%d, %t), want (%d, true)", name, want, after, ok, want)
		}
	}
}

func TestAddSkill(t *testing.T) {
	c := &chargen.Character{}

	if level := c.AddSkill("Brawling", ""); level != 1 {
		t.Errorf("first acquisition level %d, want 1 (p. 13)", level)
	}

	if level := c.AddSkill("Brawling", ""); level != 2 {
		t.Errorf("second acquisition level %d, want 2 (p. 13)", level)
	}

	if len(c.Skills) != 1 {
		t.Errorf("%d skill entries, want 1 accumulated entry", len(c.Skills))
	}
}

func TestUnmarshalRejectsUnknownFields(t *testing.T) {
	if _, err := chargen.UnmarshalRecord([]byte(`{"schema_version":"1","not_a_field":true}`)); err == nil {
		t.Error("UnmarshalRecord accepted an unknown field")
	}
}
