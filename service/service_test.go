package service_test

import (
	"testing"

	"github.com/philoserf/ctchargen/service"
)

func TestLoadValidates(t *testing.T) {
	reg, err := service.Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	if got := reg.Names(); len(got) == 0 {
		t.Fatal("Load() produced no services")
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
	if err != nil || blades[0] != "Dagger" {
		t.Errorf("Weapons(blade) starts %v (err %v), want Dagger first (p. 12)", blades[:1], err)
	}

	guns, err := reg.Weapons("gun")
	if err != nil || guns[0] != "Body Pistol" {
		t.Errorf("Weapons(gun) starts %v (err %v), want Body Pistol first (p. 13)", guns[:1], err)
	}

	if _, err := reg.Weapons("club"); err == nil {
		t.Error("Weapons(club) succeeded, want error")
	}
}
