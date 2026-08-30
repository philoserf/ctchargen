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

	// Every draft number resolves (p. 5: the draft can land anywhere).
	for n := 1; n <= 6; n++ {
		if _, err := reg.ByDraftNumber(n); err != nil {
			t.Errorf("ByDraftNumber(%d): %v", n, err)
		}
	}
}

// Milestone 1 ships Other alone: no commissions, no ranks, enlistment 3+,
// draft number 6 (p. 10).
func TestOtherDefinition(t *testing.T) {
	reg, err := service.Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	other, err := reg.Service("other")
	if err != nil {
		t.Fatalf("Service(other) error: %v", err)
	}

	if other.Commission != nil || other.Promotion != nil || len(other.Ranks) != 0 {
		t.Error("Other must have no commissions, promotions, or ranks (p. 10)")
	}

	if other.Enlistment.Target != "3+" || other.DraftNumber != 6 {
		t.Errorf("Other enlistment %s / draft %d, want 3+ / 6 (p. 10)", other.Enlistment.Target, other.DraftNumber)
	}

	byDraft, err := reg.ByDraftNumber(6)
	if err != nil || byDraft.Name != "Other" {
		t.Errorf("ByDraftNumber(6) = %v, %v; want Other", byDraft, err)
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
