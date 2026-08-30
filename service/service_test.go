package service_test

import (
	"slices"
	"testing"

	"github.com/philoserf/ctchargen/service"
)

func TestLoadValidates(t *testing.T) {
	reg, err := service.Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	// All six services, in the book's order (p. 5, p. 10).
	want := []string{"Navy", "Marines", "Army", "Scouts", "Merchants", "Other"}
	if got := reg.Names(); !slices.Equal(got, want) {
		t.Fatalf("Names() = %v, want %v", got, want)
	}

	// Every draft number resolves (p. 5: the draft can land anywhere), and
	// each resolves to the service the lookup by name returns, so the two
	// routes into the registry cannot disagree.
	for n := 1; n <= 6; n++ {
		drafted, err := reg.ByDraftNumber(n)
		if err != nil {
			t.Errorf("ByDraftNumber(%d): %v", n, err)

			continue
		}

		named, err := reg.Service(drafted.Name)
		if err != nil {
			t.Errorf("Service(%q): %v", drafted.Name, err)

			continue
		}

		if named != drafted {
			t.Errorf("draft number %d and name %q resolve to different services", n, drafted.Name)
		}
	}
}

// The exported name lists are handed to callers that may write through
// them — the UPP's digit order is read from one of them on every record —
// so each call must yield a slice of its own, as Registry.Names and
// chargen.ChoiceLabels do.
func TestNameListsAreNotShared(t *testing.T) {
	tests := map[string]func() []string{
		"CharacteristicNames": service.CharacteristicNames,
		"TableNames":          service.TableNames,
	}
	for name, list := range tests {
		t.Run(name, func(t *testing.T) {
			first := list()
			if len(first) == 0 {
				t.Fatal("empty list")
			}

			first[0] = "rewritten"

			if second := list(); second[0] == "rewritten" {
				t.Errorf("%s handed out the same backing array twice", name)
			}
		})
	}
}

func TestWeaponsListsInBookOrder(t *testing.T) {
	reg, err := service.Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	blades, err := reg.Weapons("blade")
	if err != nil {
		t.Fatalf("Weapons(blade): %v", err)
	}

	if len(blades) == 0 || blades[0] != "Dagger" {
		t.Errorf("Weapons(blade) = %v, want Dagger first (p. 12)", blades)
	}

	guns, err := reg.Weapons("gun")
	if err != nil {
		t.Fatalf("Weapons(gun): %v", err)
	}

	if len(guns) == 0 || guns[0] != "Body Pistol" {
		t.Errorf("Weapons(gun) = %v, want Body Pistol first (p. 13)", guns)
	}

	if _, err := reg.Weapons("club"); err == nil {
		t.Error("Weapons(club) succeeded, want error")
	}
}
