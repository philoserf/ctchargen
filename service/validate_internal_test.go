package service

import (
	"errors"
	"strings"
	"testing"
)

// These guards cannot be reached through Load: the shipped data is valid
// by construction, so only an internal test can hand them a broken
// registry. The externally-visible behaviour stays in service_test.go.

func registry(services ...*Service) *Registry {
	r := &Registry{services: map[string]*Service{}, weapons: map[string][]string{}}

	for _, svc := range services {
		r.services[strings.ToLower(svc.Name)] = svc
		r.order = append(r.order, svc.Name)
	}

	return r
}

// gambler is a service whose skills grant Gambling, so registryGrants is
// satisfied and the draft-number checks are what a case exercises.
func gambler(name string, draft int) *Service {
	svc := &Service{Name: name, DraftNumber: draft}
	svc.Skills.ServiceSkills = []SkillResult{{Skill: Gambling}}

	return svc
}

func TestValidateDraftNumbers(t *testing.T) {
	tests := []struct {
		name    string
		reg     *Registry
		wantErr bool
	}{
		{
			name: "distinct numbers",
			reg:  registry(gambler("Navy", 1), gambler("Marines", 2)),
		},
		{
			name:    "a repeated number would make the draft roll ambiguous",
			reg:     registry(gambler("Navy", 1), gambler("Marines", 1)),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateDraftNumbers(tt.reg)
			if tt.wantErr != (err != nil) {
				t.Fatalf("wantErr %v, got %v", tt.wantErr, err)
			}

			if tt.wantErr && !errors.Is(err, ErrInvalidData) {
				t.Errorf("want ErrInvalidData, got %v", err)
			}
		})
	}
}

func TestValidateRegistryWantsAllSix(t *testing.T) {
	err := validateRegistry(registry(gambler("Navy", 1)))
	if err == nil {
		t.Fatal("want an error for a five-service registry, got nil")
	}

	if !errors.Is(err, ErrInvalidData) {
		t.Errorf("want ErrInvalidData, got %v", err)
	}
}

func TestRegistryGrants(t *testing.T) {
	viaTable := &Service{Name: "Army"}
	viaTable.Skills.AdvancedEducation = []SkillResult{{Skill: Gambling}}

	viaBox := &Service{Name: "Navy", AutoSkills: []AutoSkill{{Rank: 1, Skill: Gambling}}}

	none := &Service{Name: "Scouts"}
	none.Skills.ServiceSkills = []SkillResult{{Skill: "Pilot"}}

	tests := []struct {
		name string
		reg  *Registry
		want bool
	}{
		{"granted by a skills table", registry(viaTable), true},
		{"granted by the rank and service skills box", registry(viaBox), true},
		{"granted nowhere", registry(none), false},
		{"granted by one service among several", registry(none, viaTable), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := registryGrants(tt.reg, Gambling); got != tt.want {
				t.Errorf("registryGrants = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestValidateOneShipKind(t *testing.T) {
	both := &Muster{Benefits: []Benefit{{Ship: "scout"}, {Ship: "free_trader"}}}
	if err := validateOneShipKind(both); err == nil {
		t.Error("want an error when one service offers both ships, got nil")
	} else if !errors.Is(err, ErrInvalidData) {
		t.Errorf("want ErrInvalidData, got %v", err)
	}

	repeat := &Muster{Benefits: []Benefit{{Ship: "free_trader"}, {Ship: "free_trader"}}}
	if err := validateOneShipKind(repeat); err != nil {
		t.Errorf("a repeated ship of one kind is the p. 22-23 rule, not an error: %v", err)
	}
}
