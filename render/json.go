// Package render projects a generated character into the shapes a reader
// wants: JSON, a character sheet in the book's own style, and a transcript
// of the generation record.
//
// The domain types are the interface and JSON is a projection. Where the two
// disagree - a sum that must marshal flat, an array-backed profile that must
// marshal as six named keys - the codec here absorbs the difference and the
// domain type keeps its shape.
package render

import (
	"encoding/json"
	"fmt"

	"github.com/philoserf/ctchargen/chargen"
	"github.com/philoserf/ctchargen/traveller"
)

// JSON marshals a character, indented, with a trailing newline.
func JSON(character *chargen.Character) ([]byte, error) {
	projected, err := project(character)
	if err != nil {
		return nil, err
	}

	text, err := json.MarshalIndent(projected, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshalling the character: %w", err)
	}

	return append(text, '\n'), nil
}

type record struct {
	Name            string            `json:"name"`
	UPP             string            `json:"upp"`
	Characteristics map[string]int    `json:"characteristics"`
	Age             ageRecord         `json:"age"`
	Terms           int               `json:"terms"`
	Enlistment      enlistmentRecord  `json:"enlistment"`
	Service         string            `json:"service,omitempty"`
	Rank            int               `json:"rank,omitempty"`
	RankTitle       string            `json:"rankTitle,omitempty"`
	Skills          []skillRecord     `json:"skills"`
	Benefits        benefitsRecord    `json:"benefits"`
	Pension         int64             `json:"annualRetirementPay,omitempty"`
	Departure       *departureRecord  `json:"departure,omitempty"`
	Title           *titleRecord      `json:"title,omitempty"`
	Inputs          inputsRecord      `json:"inputs"`
	Errata          []string          `json:"errata"`
	Ruleset         string            `json:"ruleset"`
	Events          []json.RawMessage `json:"events"`
	Build           string            `json:"build,omitempty"`
}

type ageRecord struct {
	Years  int `json:"years"`
	Months int `json:"months,omitempty"`
}

type enlistmentRecord struct {
	How     string `json:"how"`
	Service string `json:"service,omitempty"`
}

type skillRecord struct {
	Name  string `json:"name"`
	Level int    `json:"level"`
}

type shipRecord struct {
	Kind         string `json:"kind"`
	Tons         int    `json:"tons"`
	Years        int    `json:"years"`
	PaymentYears int    `json:"paymentYears"`
}

type benefitsRecord struct {
	Cash          int64        `json:"cash"`
	Passages      []string     `json:"passages,omitempty"`
	TravellersAid bool         `json:"travellersAid,omitempty"`
	LostShips     int          `json:"scoutShipsLost,omitempty"`
	Weapons       []string     `json:"weapons,omitempty"`
	Ships         []shipRecord `json:"ships,omitempty"`
}

type departureRecord struct {
	How            string `json:"how"`
	Fatal          bool   `json:"fatal,omitempty"`
	Characteristic string `json:"characteristic,omitempty"`
}

type titleRecord struct {
	Eligible bool   `json:"eligible"`
	Rank     string `json:"rank"`
	Assumed  bool   `json:"assumed"`
}

type inputsRecord struct {
	Seed    uint64 `json:"seed"`
	Name    string `json:"name,omitempty"`
	Service string `json:"service,omitempty"`
	Career  string `json:"career"`
	Skills  string `json:"skills"`
	Muster  string `json:"muster"`
}

func project(character *chargen.Character) (record, error) {
	out := record{
		Name:            character.Name,
		UPP:             character.Profile.UPP(),
		Characteristics: map[string]int{},
		Age:             ageRecord{Years: character.Age.Years(), Months: character.Age.Months()},
		Terms:           character.Terms,
		Enlistment:      foldEnlistment(character.Enlistment),
		Rank:            int(character.Rank),
		RankTitle:       character.RankTitle,
		Skills:          make([]skillRecord, 0, len(character.Skills)),
		Benefits:        projectBenefits(character.Benefits, character.DuplicateShips),
		Pension:         int64(character.Pension),
		Inputs:          projectInputs(character.Inputs),
		Errata:          make([]string, 0, len(character.Errata)),
		Ruleset:         character.Ruleset,
		Events:          nil,
		Build:           character.Build,
	}

	events, err := projectEvents(character.Events)
	if err != nil {
		return record{}, fmt.Errorf("projecting the generation record: %w", err)
	}

	out.Events = events

	for _, characteristic := range traveller.Characteristics {
		out.Characteristics[characteristic.String()] = character.Profile[characteristic]
	}

	if character.Served {
		out.Service = character.Service.String()
	}

	for _, skill := range character.Skills {
		out.Skills = append(out.Skills, skillRecord{Name: string(skill.Name), Level: skill.Level})
	}

	for _, erratum := range character.Errata {
		out.Errata = append(out.Errata, erratum.String())
	}

	if character.Departure != nil {
		departure := foldDeparture(character.Departure)

		out.Departure = &departure
	}

	if character.Title.Eligible {
		out.Title = &titleRecord{
			Eligible: true, Rank: character.Title.Rank.String(), Assumed: character.Title.Assumed,
		}
	}

	return out, nil
}

func projectInputs(in chargen.Inputs) inputsRecord {
	out := inputsRecord{
		Seed: in.Seed, Name: in.Name,
		Career: in.Career, Skills: in.Skills, Muster: in.Muster,
	}
	if in.Forced {
		out.Service = in.Service.String()
	}

	return out
}

func projectBenefits(b chargen.Benefits, lost int) benefitsRecord {
	out := benefitsRecord{Cash: int64(b.Cash), TravellersAid: b.TravellersAid, LostShips: lost}

	for _, passage := range b.Passages {
		out.Passages = append(out.Passages, passage.String())
	}

	for _, weapon := range b.Weapons {
		out.Weapons = append(out.Weapons, string(weapon))
	}

	for _, ship := range b.Ships {
		out.Ships = append(out.Ships, shipRecord{
			Kind: ship.Kind.String(), Tons: ship.Tons,
			Years: ship.Years, PaymentYears: ship.PaymentYears,
		})
	}

	return out
}
