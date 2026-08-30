// Package chargen is the prior-service procedure of Book 1 pp. 4-25: the
// character record, the generation event log, the engine that walks the
// procedure, and the deciders that answer its choice points. The JSON
// record is the source of truth; everything else renders or verifies it
// (docs/PRD.md).
package chargen

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/philoserf/ctchargen/service"
)

// Version stamps carried by every record (docs/PRD.md, Replay and
// provenance contract).
const (
	// SchemaVersion tracks the shape of the records the engine writes
	// (docs/character.schema.json).
	SchemaVersion = "2"
	// EngineVersion changes when generation behaviour changes: rules,
	// dice-stream consumption order, or the RNG construction.
	EngineVersion = "0.5.0"
	// PolicyVersion identifies the docs/POLICY.md decision table the auto mode
	// applies. Never verified on replay: recorded choices are reapplied,
	// the policy is not consulted.
	PolicyVersion = "3"
	// Ruleset pins the pages every rule was read from.
	Ruleset = "Classic Traveller Books 1-3, © 1977 text, FFE reprints"
)

// RNG names the algorithm and seed the dice stream was built from.
type RNG struct {
	Algorithm string `json:"algorithm"`
	Seed      uint64 `json:"seed"`
}

// Inputs is everything the caller supplied; with the seed, it is all a
// replay needs beyond the recorded choices.
type Inputs struct {
	Auto    bool   `json:"auto"`
	Name    string `json:"name"`
	Service string `json:"service"` // --service: forces the enlistment attempt only
}

// Characteristics are stored numeric (2-12 initially, 1-15 through play;
// p. 4), rolled in this order.
type Characteristics struct {
	Strength       int `json:"strength"`
	Dexterity      int `json:"dexterity"`
	Endurance      int `json:"endurance"`
	Intelligence   int `json:"intelligence"`
	Education      int `json:"education"`
	SocialStanding int `json:"social_standing"`
}

// Get reads a characteristic by its service-data name.
func (c *Characteristics) Get(name string) int {
	switch name {
	case service.Strength:
		return c.Strength
	case service.Dexterity:
		return c.Dexterity
	case service.Endurance:
		return c.Endurance
	case service.Intelligence:
		return c.Intelligence
	case service.Education:
		return c.Education
	case service.SocialStanding:
		return c.SocialStanding
	}

	return 0
}

// Apply alters a characteristic, clamped to 1-15: values never exceed 15
// and do not go below 1 outside calamitous injury or aging (p. 4).
// It reports the value before and then after the alteration, and whether
// the name was one of the six.
//
// The name is checked because set ignores an unrecognised one: without
// the check Apply would return (0, 1) for a name it never stored, and the
// caller would write an alteration into the event log that the
// characteristics block does not show. Replay compares record against
// record, so it cannot catch a record that is internally consistent and
// wrong.
func (c *Characteristics) Apply(name string, delta int) (int, int, bool) {
	if !validCharacteristic(name) {
		return 0, 0, false
	}

	before := c.Get(name)
	after := min(max(before+delta, 1), 15)
	c.set(name, after)

	return before, after, true
}

// UPP is the Universal Personality Profile: the six characteristics as
// hexadecimal digits in rolled order, 10-15 as A-F (p. 8).
func (c *Characteristics) UPP() string {
	const hexDigits = "0123456789ABCDEF"

	digits := make([]byte, 0, 6)
	for _, name := range service.CharacteristicNames() {
		digits = append(digits, hexDigits[c.Get(name)])
	}

	return string(digits)
}

// applyAging reduces a characteristic with a floor of 0, not 1: aging is
// one of the two ways a value may fall below 1 (p. 4), and a zero is a
// medical crisis (pp. 7-8). It checks the name for the same reason Apply
// does.
func (c *Characteristics) applyAging(name string, loss int) (int, int, bool) {
	if !validCharacteristic(name) {
		return 0, 0, false
	}

	before := c.Get(name)
	after := max(before-loss, 0)
	c.set(name, after)

	return before, after, true
}

func (c *Characteristics) set(name string, v int) {
	switch name {
	case service.Strength:
		c.Strength = v
	case service.Dexterity:
		c.Dexterity = v
	case service.Endurance:
		c.Endurance = v
	case service.Intelligence:
		c.Intelligence = v
	case service.Education:
		c.Education = v
	case service.SocialStanding:
		c.SocialStanding = v
	}
}

// Skill is a name and level ("Brawling-2"). Weapon expertise is recorded
// under the specific weapon's name, as the book does ("Dagger-1", p. 25),
// with Category preserving the table result ("blade" or "gun") that
// demanded the pick (pp. 11-13).
type Skill struct {
	Name     string `json:"name"`
	Level    int    `json:"level"`
	Category string `json:"category,omitempty"`
}

// Death records the term and cause for a character the procedure killed —
// a completed generation, not an error (docs/PRD.md, Decisions).
type Death struct {
	Term  int    `json:"term"`
	Cause string `json:"cause"`
}

// Passages are travel tickets received at mustering out, by class
// (values CR 10,000 / 8,000 / 1,000, sellable at 90%; pp. 21-22).
type Passages struct {
	High   int `json:"high"`
	Middle int `json:"middle"`
	Low    int `json:"low"`
}

// Ship is a starship benefit (pp. 22-23). A Free Trader (Type A, Book 2
// p. 19) starts owing 40 years of payments; each additional receipt pays
// off 10 years and ages the ship 10 years. A Scout ship (Type S, Book 2
// p. 18) is constructive possession: no title, no sale, no mortgage,
// duplicates lost.
type Ship struct {
	Class                  string `json:"class"`
	Book2Page              int    `json:"book2_page"`
	Receipts               int    `json:"receipts"`
	AgeYears               int    `json:"age_years"`
	PaymentYearsRemaining  int    `json:"payment_years_remaining"`
	ConstructivePossession bool   `json:"constructive_possession"`
}

// Benefits is everything mustering out conferred (FR6, FR8).
type Benefits struct {
	Cash          int      `json:"cash"` // integer credits
	Passages      Passages `json:"passages"`
	TravellersAid bool     `json:"travellers_aid"`
	Weapons       []string `json:"weapons"` // physical weapons received (p. 22)
	Ship          *Ship    `json:"ship,omitempty"`
}

// Title records hereditary-title eligibility (Social Standing 11+, p. 4)
// and the choice (FR7). The title is Book 3 p. 22's designation.
type Title struct {
	Title   string `json:"title"`
	Assumed bool   `json:"assumed"`
}

// EventDM is one applied die modification, with the rule that granted it.
type EventDM struct {
	Source string `json:"source"`
	Value  int    `json:"value"`
}

// Event is one entry of the generation record (FR10): a procedure step
// entered, a throw, a choice, or an outcome. Fields are populated per
// kind; Seq starts at 1 so a Ref of 0 means "none".
type Event struct {
	Seq   int    `json:"seq"`
	Kind  string `json:"kind"` // "step", "throw", "choice", "outcome"
	Step  string `json:"step"`
	Label string `json:"label,omitempty"`

	// throw
	Dice    []int     `json:"dice,omitempty"`
	DMs     []EventDM `json:"dms,omitempty"`
	Target  string    `json:"target,omitempty"`
	Total   int       `json:"total,omitempty"`
	Success *bool     `json:"success,omitempty"`

	// choice
	By      string   `json:"by,omitempty"` // "policy" or "player"
	Options []string `json:"options,omitempty"`
	Picked  string   `json:"picked,omitempty"`

	// outcome
	Text string `json:"text,omitempty"`
	Ref  int    `json:"ref,omitempty"` // seq of the causing throw or choice
}

// Character is the record (FR8). JSON is canonical; the Markdown sheet is
// a render of it.
type Character struct {
	SchemaVersion string   `json:"schema_version"`
	Ruleset       string   `json:"ruleset"`
	EngineVersion string   `json:"engine_version"`
	PolicyVersion string   `json:"policy_version"`
	RNG           RNG      `json:"rng"`
	Inputs        Inputs   `json:"inputs"`
	Errata        []string `json:"errata"`

	Name string `json:"name"`
	Age  int    `json:"age"`
	// AgeMonths is the fraction of a year beyond Age (0-11), accrued only
	// by medical-crisis recovery (1D months, pp. 7-8).
	AgeMonths       int             `json:"age_months"`
	Terms           int             `json:"terms"`
	Service         string          `json:"service"` // "" for a civilian who declined the draft
	Drafted         bool            `json:"drafted"`
	Rank            int             `json:"rank"`
	RankTitle       string          `json:"rank_title,omitempty"`
	Characteristics Characteristics `json:"characteristics"`
	UPP             string          `json:"upp"`
	Skills          []Skill         `json:"skills"`
	Benefits        Benefits        `json:"benefits"`
	RetirementPay   int             `json:"retirement_pay"` // CR per year (pp. 7, 21)
	Title           *Title          `json:"title,omitempty"`
	Death           *Death          `json:"death,omitempty"`

	Events []Event `json:"events"`
}

// AddAgeMonths ages the character by whole months, carrying into years.
func (c *Character) AddAgeMonths(months int) {
	c.AgeMonths += months
	c.Age += c.AgeMonths / 12
	c.AgeMonths %= 12
}

// AddSkill raises the named skill by one level, creating it at level 1 on
// first acquisition (p. 13: Skill-1, Skill-2, ... with no cap). It
// reports the new level.
func (c *Character) AddSkill(name, category string) int {
	for i := range c.Skills {
		if c.Skills[i].Name == name {
			c.Skills[i].Level++

			return c.Skills[i].Level
		}
	}

	c.Skills = append(c.Skills, Skill{Name: name, Level: 1, Category: category})

	return 1
}

// MarshalRecord renders the canonical JSON bytes: two-space indent with a
// trailing newline. Golden fixtures and CLI output both use exactly this.
func (c *Character) MarshalRecord() ([]byte, error) {
	out, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshaling character record: %w", err)
	}

	return append(out, '\n'), nil
}

// UnmarshalRecord parses a character record, rejecting unknown fields so
// a record from a newer schema fails loudly rather than silently dropping
// data, and rejecting anything after the record so a file holding two is
// not read as one. This decoder is the strict one — chargen.decodeStrict
// and service.decodeStrict read go:embed data this repository writes,
// where a second value cannot appear — because its input is a file the
// user names, and because replay's verdict is computed from the parsed
// value and a re-marshal of it, never from the bytes on disk: whatever
// this drops, nothing downstream ever sees.
func UnmarshalRecord(data []byte) (*Character, error) {
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()

	char := &Character{}
	if err := dec.Decode(char); err != nil {
		return nil, fmt.Errorf("parsing character record: %w", err)
	}

	// More skips whitespace, so MarshalRecord's trailing newline passes.
	if dec.More() {
		return nil, fmt.Errorf("parsing character record: %w", ErrTrailingData)
	}

	return char, nil
}
