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
// All six must load: the draft is a one-die roll over them (p. 5), so a
// missing one would turn a legal roll into a runtime failure. See
// validateRegistry.
var bookOrder = []string{"Navy", "Marines", "Army", "Scouts", "Merchants", "Other"}

// Gambling is the skill Book 1 p. 9 reads to offer a +1 DM on the cash
// table. The engine finds it by name, so the spelling is pinned here and
// asserted against the loaded tables (registryGrants) rather than left to
// agree with the data by luck.
const Gambling = "Gambling"

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

// AutoSkill is one row of the Rank and Service Skills box (p. 23): a
// skill or characteristic alteration that accrues automatically, without
// eligibility, by virtue of a specific service (Rank 0) or a specific
// rank (Rank 1-6). Skill names a specific skill or weapon — never a
// category choice — with Category set when it is a weapon.
type AutoSkill struct {
	Rank           int    `json:"rank"`
	Skill          string `json:"skill,omitempty"`
	Category       string `json:"category,omitempty"`
	Characteristic string `json:"characteristic,omitempty"`
	Delta          int    `json:"delta,omitempty"`
}

// Benefit is one row of the service's Mustering Out Table 1 (p. 9):
// exactly one of a passage, a characteristic alteration, a weapon benefit
// (category, weapon chosen on receipt, p. 22), Travellers' Aid
// membership, or a ship — or nothing at all (the table's "—" rows).
type Benefit struct {
	Passage        string `json:"passage,omitempty"` // "low", "middle", "high"
	Characteristic string `json:"characteristic,omitempty"`
	Delta          int    `json:"delta,omitempty"`
	Weapon         string `json:"weapon,omitempty"` // "blade" or "gun"
	TravellersAid  bool   `json:"travellers_aid,omitempty"`
	Ship           string `json:"ship,omitempty"` // "scout" or "free_trader"
}

// Muster is the service's two mustering-out tables (p. 9): Table 1
// material benefits and Table 2 cash, each indexed by one die (a +1 DM
// can reach row 7).
type Muster struct {
	Page     int       `json:"page"`
	Benefits []Benefit `json:"benefits"`
	Cash     []int     `json:"cash"`
}

// Service is one column of the Prior Service Table plus the service's
// ranks, skills tables, and mustering-out tables.
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
	AutoSkills       []AutoSkill `json:"auto_skills"`
	RetirementPay    bool        `json:"retirement_pay"` // Navy, Marines, Army, Merchants only (p. 21)
	Muster           Muster      `json:"muster"`
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

	if err := validateRegistry(reg); err != nil {
		return nil, err
	}

	return reg, nil
}

// validateRegistry checks the loaded set as a whole, which the per-file
// validation cannot: every service present, every draft number distinct,
// and the one skill the engine looks up by name actually grantable.
func validateRegistry(r *Registry) error {
	if len(r.order) != len(bookOrder) {
		missing := make([]string, 0, len(bookOrder))

		for _, name := range bookOrder {
			if _, ok := r.services[strings.ToLower(name)]; !ok {
				missing = append(missing, name)
			}
		}

		return fmt.Errorf("%w: missing service data for %s", ErrInvalidData, strings.Join(missing, ", "))
	}

	if err := validateDraftNumbers(r); err != nil {
		return err
	}

	if !registryGrants(r, Gambling) {
		return fmt.Errorf("%w: no service's skills grant %q, so the p. 9 cash-table DM is unreachable",
			ErrInvalidData, Gambling)
	}

	return nil
}

// validateDraftNumbers rejects a repeated draft number. With all six
// services present and each number already range-checked to 1-6 by
// validateService, distinctness gives full 1-6 coverage by pigeonhole —
// so the one-die draft roll (p. 5) always resolves. Iteration is over
// order, not the map, so a collision is reported the same way every run.
func validateDraftNumbers(r *Registry) error {
	var taken [7]string

	for _, name := range r.order {
		svc := r.services[strings.ToLower(name)]
		if prior := taken[svc.DraftNumber]; prior != "" {
			return fmt.Errorf("%w: %s and %s both take draft number %d",
				ErrInvalidData, prior, svc.Name, svc.DraftNumber)
		}

		taken[svc.DraftNumber] = svc.Name
	}

	return nil
}

// registryGrants reports whether any loaded service can confer the named
// skill, by an Acquired Skills table or the Rank and Service Skills box.
// The check is registry-wide because most services grant Gambling
// nowhere; only the set as a whole has to be able to.
func registryGrants(r *Registry, skill string) bool {
	for _, svc := range r.services {
		for _, table := range TableNames {
			rows, _ := svc.Skills.Table(table)
			for _, row := range rows {
				if row.Skill == skill {
					return true
				}
			}
		}

		for _, auto := range svc.AutoSkills {
			if auto.Skill == skill {
				return true
			}
		}
	}

	return false
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

	if err := validateAutoSkills(svc); err != nil {
		return err
	}

	if err := validateMuster(&svc.Muster); err != nil {
		return err
	}

	return validateTables(&svc.Skills)
}

func validateMuster(m *Muster) error {
	// Seven rows each: one die plus the possible +1 DM (p. 9).
	if len(m.Benefits) != 7 || len(m.Cash) != 7 {
		return fmt.Errorf("%w: muster tables want 7 rows each, got %d benefits and %d cash",
			ErrInvalidData, len(m.Benefits), len(m.Cash))
	}

	for i, amount := range m.Cash {
		if amount <= 0 {
			return fmt.Errorf("%w: muster cash row %d is %d", ErrInvalidData, i+1, amount)
		}
	}

	for i, b := range m.Benefits {
		if err := validateBenefit(b); err != nil {
			return fmt.Errorf("muster benefit row %d: %w", i+1, err)
		}
	}

	return validateOneShipKind(m)
}

// validateOneShipKind keeps a single service from offering both ships. A
// record holds one ship, and the two kinds are not interchangeable — a
// Scout is constructive possession with no mortgage, a Free Trader owes
// 40 years of payments (pp. 22-23) — so a character holding one and
// receiving the other has no defined outcome here. No reading of p. 23
// for that case has been made; until one is, the data may not produce it.
func validateOneShipKind(m *Muster) error {
	var scout, trader bool

	for _, b := range m.Benefits {
		switch b.Ship {
		case "scout":
			scout = true
		case "free_trader":
			trader = true
		}
	}

	if scout && trader {
		return fmt.Errorf("%w: muster benefits offer both a scout ship and a free trader", ErrInvalidData)
	}

	return nil
}

func validateBenefit(b Benefit) error {
	if err := validateBenefitFields(b); err != nil {
		return err
	}

	set := 0
	kinds := []bool{b.Passage != "", b.Characteristic != "", b.Weapon != "", b.TravellersAid, b.Ship != ""}

	for _, present := range kinds {
		if present {
			set++
		}
	}

	if set > 1 {
		return fmt.Errorf("%w: benefit row sets %d kinds, want at most one", ErrInvalidData, set)
	}

	return nil
}

func validateBenefitFields(b Benefit) error {
	if err := validateEnum("passage", b.Passage, "low", "middle", "high"); err != nil {
		return err
	}

	if err := validateEnum("weapon category", b.Weapon, "blade", "gun"); err != nil {
		return err
	}

	if err := validateEnum("ship", b.Ship, "scout", "free_trader"); err != nil {
		return err
	}

	return validateBenefitAlteration(b)
}

// validateEnum accepts the empty string (field absent) or a listed value.
func validateEnum(label, v string, allowed ...string) error {
	if v == "" || slices.Contains(allowed, v) {
		return nil
	}

	return fmt.Errorf("%w: %s %q", ErrInvalidData, label, v)
}

func validateBenefitAlteration(b Benefit) error {
	if b.Characteristic == "" {
		if b.Delta != 0 {
			return fmt.Errorf("%w: delta without a characteristic", ErrInvalidData)
		}

		return nil
	}

	if !validCharacteristic(b.Characteristic) || b.Delta < 1 || b.Delta > 2 {
		return fmt.Errorf("%w: alteration %s %+d", ErrInvalidData, b.Characteristic, b.Delta)
	}

	return nil
}

func validateAutoSkills(svc *Service) error {
	for i, auto := range svc.AutoSkills {
		if auto.Rank < 0 || auto.Rank > len(svc.Ranks) {
			return fmt.Errorf("%w: auto skill %d at rank %d, service has %d ranks", ErrInvalidData, i, auto.Rank, len(svc.Ranks))
		}

		if err := validateAutoSkill(auto); err != nil {
			return fmt.Errorf("auto skill %d: %w", i, err)
		}
	}

	return nil
}

func validateAutoSkill(auto AutoSkill) error {
	hasSkill := auto.Skill != ""
	hasAlteration := auto.Characteristic != ""

	if hasSkill == hasAlteration {
		return fmt.Errorf("%w: want exactly one of skill or characteristic", ErrInvalidData)
	}

	if hasAlteration && (!validCharacteristic(auto.Characteristic) || auto.Delta != 1) {
		return fmt.Errorf("%w: alteration %s %+d (p. 23 grants only +1)",
			ErrInvalidData, auto.Characteristic, auto.Delta)
	}

	if auto.Category != "" && (auto.Category != "blade" && auto.Category != "gun" || !hasSkill) {
		return fmt.Errorf("%w: category %q", ErrInvalidData, auto.Category)
	}

	return nil
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
