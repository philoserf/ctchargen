// Package service holds the data-driven definitions of the six prior
// services: the Prior Service Table (Book 1 p. 10), the Table of Ranks
// (p. 10), the four Acquired Skills tables (p. 11), and the weapons lists
// (pp. 12-13). The tables are embedded JSON validated at load time;
// exceptional mechanics stay in the chargen package (docs/PRD.md,
// Architecture notes).
package service

import (
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/philoserf/ctchargen/dice"
)

//go:embed data/*.json
var dataFS embed.FS

// bookOrder is the services as Book 1 lists them (p. 5, p. 10). It is the
// tie-break order for the auto policy and the display order everywhere.
// Only the entries with a data file are available; through milestone 1
// that is Other alone.
var bookOrder = []string{"Navy", "Marines", "Army", "Scouts", "Merchants", "Other"}

// Characteristics as the data files name them, in the rolled order (p. 4).
const (
	Strength       = "strength"
	Dexterity      = "dexterity"
	Endurance      = "endurance"
	Intelligence   = "intelligence"
	Education      = "education"
	SocialStanding = "social_standing"
)

// CharacteristicNames is the rolled order (p. 4), used for validation and
// for iterating in a stable order.
var CharacteristicNames = []string{Strength, Dexterity, Endurance, Intelligence, Education, SocialStanding}

// DM is one row of a throw's cumulative die modifications: +DM when the
// named characteristic is at or above Min (p. 5: "If both stated
// characteristics are present in the required level, the die modification
// is cumulative").
type DM struct {
	Characteristic string `json:"characteristic"`
	Min            int    `json:"min"`
	DM             int    `json:"dm"`
}

// ThrowSpec is a target with its characteristic DMs, one cell of the
// Prior Service Table.
type ThrowSpec struct {
	Target string `json:"target"`
	DMs    []DM   `json:"dms"`
}

// SkillResult is one row of an Acquired Skills table (p. 11): exactly one
// of a characteristic alteration, a named skill, or a weapon-expertise
// category that demands an immediate weapon choice (pp. 11-13).
type SkillResult struct {
	Characteristic string `json:"characteristic,omitempty"`
	Delta          int    `json:"delta,omitempty"`
	Skill          string `json:"skill,omitempty"`
	Weapon         string `json:"weapon,omitempty"`
}

// SkillTables is the four Acquired Skills tables for one service (p. 11).
// The fourth is gated on Education 8+.
type SkillTables struct {
	PersonalDevelopment []SkillResult `json:"personal_development"`
	ServiceSkills       []SkillResult `json:"service_skills"`
	AdvancedEducation   []SkillResult `json:"advanced_education"`
	AdvancedEducation8  []SkillResult `json:"advanced_education_8"`
}

// TableNames is the four Acquired Skills tables in the book's order
// (p. 11); the fourth is available only at Education 8+.
var TableNames = []string{"personal_development", "service_skills", "advanced_education", "advanced_education_8"}

// Table looks a skills table up by its name from TableNames.
func (t *SkillTables) Table(name string) ([]SkillResult, bool) {
	switch name {
	case "personal_development":
		return t.PersonalDevelopment, true
	case "service_skills":
		return t.ServiceSkills, true
	case "advanced_education":
		return t.AdvancedEducation, true
	case "advanced_education_8":
		return t.AdvancedEducation8, true
	}

	return nil, false
}

// Service is one column of the Prior Service Table plus the service's
// ranks and skills tables.
type Service struct {
	Name             string      `json:"name"`
	PriorServicePage int         `json:"prior_service_page"`
	SkillsPage       int         `json:"skills_page"`
	Enlistment       ThrowSpec   `json:"enlistment"`
	DraftNumber      int         `json:"draft_number"`
	Survival         ThrowSpec   `json:"survival"`
	Commission       *ThrowSpec  `json:"commission"`
	Promotion        *ThrowSpec  `json:"promotion"`
	Reenlist         ThrowSpec   `json:"reenlist"`
	Ranks            []string    `json:"ranks"`
	Skills           SkillTables `json:"skills"`
}

// Registry is the loaded, validated rule data.
type Registry struct {
	services map[string]*Service // keyed by lower-cased name
	order    []string            // available services in book order
	weapons  map[string][]string // "blade" and "gun" lists, book order
}

// Errors the registry reports: broken embedded data (a build defect), and
// lookups of things the data does not hold.
var (
	ErrInvalidData = errors.New("invalid service data")
	ErrUnavailable = errors.New("not available")
)

// Load parses and validates the embedded data files. Invalid data is a
// build defect surfaced at first use, not at some later roll.
func Load() (*Registry, error) {
	reg := &Registry{services: map[string]*Service{}, weapons: map[string][]string{}}

	if err := reg.loadWeapons(); err != nil {
		return nil, err
	}

	if err := reg.loadServices(); err != nil {
		return nil, err
	}

	for _, name := range bookOrder {
		if _, ok := reg.services[strings.ToLower(name)]; ok {
			reg.order = append(reg.order, name)
		}
	}

	if len(reg.order) != len(reg.services) {
		return nil, fmt.Errorf("%w: a data file names a service outside the book's six", ErrInvalidData)
	}

	return reg, nil
}

// Service looks a service up by name, case-insensitively.
func (r *Registry) Service(name string) (*Service, error) {
	svc, ok := r.services[strings.ToLower(name)]
	if !ok {
		return nil, fmt.Errorf("service %q %w (have %s)", name, ErrUnavailable, strings.Join(r.order, ", "))
	}

	return svc, nil
}

// ByDraftNumber resolves a one-die draft roll (p. 5) to its service.
func (r *Registry) ByDraftNumber(n int) (*Service, error) {
	for _, svc := range r.services {
		if svc.DraftNumber == n {
			return svc, nil
		}
	}

	return nil, fmt.Errorf("draft number %d: service %w", n, ErrUnavailable)
}

// Names is the available services in book order.
func (r *Registry) Names() []string { return append([]string(nil), r.order...) }

// Weapons is the book-order list for a weapon category ("blade", pp. 12;
// "gun", p. 13).
func (r *Registry) Weapons(category string) ([]string, error) {
	list, ok := r.weapons[category]
	if !ok {
		return nil, fmt.Errorf("weapon category %q %w: want blade or gun", category, ErrUnavailable)
	}

	return append([]string(nil), list...), nil
}

func (r *Registry) loadServices() error {
	entries, err := dataFS.ReadDir("data")
	if err != nil {
		return fmt.Errorf("reading embedded data: %w", err)
	}

	for _, entry := range entries {
		if entry.Name() == "weapons.json" {
			continue
		}

		raw, err := dataFS.ReadFile("data/" + entry.Name())
		if err != nil {
			return fmt.Errorf("reading %s: %w", entry.Name(), err)
		}

		svc := &Service{}

		dec := json.NewDecoder(strings.NewReader(string(raw)))
		dec.DisallowUnknownFields()

		if err := dec.Decode(svc); err != nil {
			return fmt.Errorf("parsing %s: %w", entry.Name(), err)
		}

		if err := validateService(svc); err != nil {
			return fmt.Errorf("%s: %w", entry.Name(), err)
		}

		r.services[strings.ToLower(svc.Name)] = svc
	}

	return nil
}

func (r *Registry) loadWeapons() error {
	raw, err := dataFS.ReadFile("data/weapons.json")
	if err != nil {
		return fmt.Errorf("reading weapons.json: %w", err)
	}

	var parsed struct {
		BladesPage int      `json:"blades_page"`
		GunsPage   int      `json:"guns_page"`
		Blade      []string `json:"blade"`
		Gun        []string `json:"gun"`
	}

	dec := json.NewDecoder(strings.NewReader(string(raw)))
	dec.DisallowUnknownFields()

	if err := dec.Decode(&parsed); err != nil {
		return fmt.Errorf("parsing weapons.json: %w", err)
	}

	if len(parsed.Blade) == 0 || len(parsed.Gun) == 0 {
		return fmt.Errorf("%w: weapons.json has an empty weapon list", ErrInvalidData)
	}

	r.weapons["blade"] = parsed.Blade
	r.weapons["gun"] = parsed.Gun

	return nil
}

func validateService(svc *Service) error {
	if !slices.Contains(bookOrder, svc.Name) {
		return fmt.Errorf("%w: name %q is not one of the book's six services", ErrInvalidData, svc.Name)
	}

	if svc.DraftNumber < 1 || svc.DraftNumber > 6 {
		return fmt.Errorf("%w: draft number %d out of 1-6", ErrInvalidData, svc.DraftNumber)
	}

	if err := validateThrows(svc); err != nil {
		return err
	}

	if err := validateRankStructure(svc); err != nil {
		return err
	}

	return validateTables(&svc.Skills)
}

func validateThrows(svc *Service) error {
	throws := []struct {
		label string
		spec  *ThrowSpec
	}{
		{"enlistment", &svc.Enlistment},
		{"survival", &svc.Survival},
		{"reenlist", &svc.Reenlist},
		{"commission", svc.Commission},
		{"promotion", svc.Promotion},
	}
	for _, t := range throws {
		if t.spec == nil {
			continue
		}

		if err := validateThrowSpec(t.label, t.spec); err != nil {
			return err
		}
	}

	// Reenlistment allows no DMs (p. 6), and the 12-exactly rule (pp. 6-7)
	// reads the bare dice, so the data must not carry any.
	if len(svc.Reenlist.DMs) != 0 {
		return fmt.Errorf("%w: reenlist DMs present, but no DMs are allowed (p. 6)", ErrInvalidData)
	}

	return nil
}

func validateThrowSpec(label string, spec *ThrowSpec) error {
	if _, err := dice.ParseTarget(spec.Target); err != nil {
		return fmt.Errorf("%s: %w", label, err)
	}

	for _, dm := range spec.DMs {
		if !validCharacteristic(dm.Characteristic) {
			return fmt.Errorf("%w: %s DM names unknown characteristic %q", ErrInvalidData, label, dm.Characteristic)
		}

		if dm.DM == 0 || dm.Min < 1 || dm.Min > 15 {
			return fmt.Errorf("%w: %s DM on %s: dm %d at min %d out of range",
				ErrInvalidData, label, dm.Characteristic, dm.DM, dm.Min)
		}
	}

	return nil
}

// validateRankStructure holds the p. 10 note: commissions and promotions
// are non-existent in the Scout and Other services, and a promotion
// without a commission path would be unreachable (p. 6).
func validateRankStructure(svc *Service) error {
	if (svc.Commission == nil) != (svc.Promotion == nil) {
		return fmt.Errorf("%w: commission and promotion must be both present or both absent", ErrInvalidData)
	}

	if svc.Commission != nil && len(svc.Ranks) == 0 {
		return fmt.Errorf("%w: commission path without ranks", ErrInvalidData)
	}

	if svc.Commission == nil && len(svc.Ranks) != 0 {
		return fmt.Errorf("%w: ranks without a commission path", ErrInvalidData)
	}

	return nil
}

func validateTables(skills *SkillTables) error {
	for _, name := range TableNames {
		rows, _ := skills.Table(name)
		if len(rows) != 6 {
			return fmt.Errorf("%w: %s has %d rows, want 6 (one per die face)", ErrInvalidData, name, len(rows))
		}

		for i, row := range rows {
			if err := validateSkillResult(row); err != nil {
				return fmt.Errorf("%s row %d: %w", name, i+1, err)
			}
		}
	}

	return nil
}

func validateSkillResult(row SkillResult) error {
	set := 0

	if row.Characteristic != "" {
		set++

		if err := validateAlteration(row); err != nil {
			return err
		}
	} else if row.Delta != 0 {
		return fmt.Errorf("%w: delta without a characteristic", ErrInvalidData)
	}

	if row.Skill != "" {
		set++
	}

	if row.Weapon != "" {
		set++

		if row.Weapon != "blade" && row.Weapon != "gun" {
			return fmt.Errorf("%w: weapon category %q, want blade or gun", ErrInvalidData, row.Weapon)
		}
	}

	if set != 1 {
		return fmt.Errorf("%w: want exactly one of characteristic, skill, weapon; got %d", ErrInvalidData, set)
	}

	return nil
}

func validateAlteration(row SkillResult) error {
	if !validCharacteristic(row.Characteristic) {
		return fmt.Errorf("%w: unknown characteristic %q", ErrInvalidData, row.Characteristic)
	}

	if row.Delta != 1 && row.Delta != -1 {
		return fmt.Errorf("%w: characteristic alteration delta %d, want +1 or -1 (p. 11)", ErrInvalidData, row.Delta)
	}

	return nil
}

func validCharacteristic(name string) bool {
	return slices.Contains(CharacteristicNames, name)
}
