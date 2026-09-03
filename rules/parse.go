package rules

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/philoserf/ctchargen/traveller"
)

// The lift's parsers. Every one of them checks a printed string against the
// domain's own closed alphabet rather than against a second list kept here:
// a name that does not name a value of the type is a defect in the data
// file, and the lift is where it must be caught.

// named indexes an alphabet by the name its values give themselves, so that
// a data file's spelling is checked against the type and nothing else.
func named[T fmt.Stringer](values []T) map[string]T {
	index := make(map[string]T, len(values))
	for _, v := range values {
		index[v.String()] = v
	}

	return index
}

var (
	characteristics  = named(traveller.Characteristics[:])
	serviceNames     = named(traveller.ServiceNames[:])
	skillTables      = named(traveller.SkillTables[:])
	weaponCategories = named(traveller.WeaponCategories[:])
	passageClasses   = named(traveller.PassageClasses[:])
	shipKinds        = named(traveller.ShipKinds[:])
	titles           = named(traveller.Titles[:])
)

func lookup[T any](index map[string]T, name, kind string) (T, error) {
	value, ok := index[name]
	if !ok {
		var zero T

		return zero, fmt.Errorf("%q is not a %s", name, kind)
	}

	return value, nil
}

func parseCharacteristic(name string) (traveller.Characteristic, error) {
	return lookup(characteristics, name, "characteristic")
}

func parseService(name string) (traveller.ServiceName, error) {
	return lookup(serviceNames, name, "service")
}

func parseSkillTable(name string) (traveller.SkillTable, error) {
	return lookup(skillTables, name, "skills table")
}

func parseTitle(name string) (traveller.Title, error) {
	return lookup(titles, name, "title")
}

// alteration matches a characteristic alteration as the tables print one:
// "+1 Strength", "-1 Social Standing", "+2 Education".
var alteration = regexp.MustCompile(`^([+-])(\d+) (.+)$`)

// parseAlteration reads a characteristic alteration, reporting whether the
// cell is one at all.
func parseAlteration(cell string) (traveller.Characteristic, int, bool, error) {
	match := alteration.FindStringSubmatch(cell)
	if match == nil {
		return 0, 0, false, nil
	}

	size, err := strconv.Atoi(match[2])
	if err != nil {
		return 0, 0, false, fmt.Errorf("alteration %q: %w", cell, err)
	}
	if match[1] == "-" {
		size = -size
	}

	characteristic, err := parseCharacteristic(match[3])
	if err != nil {
		return 0, 0, false, fmt.Errorf("alteration %q: %w", cell, err)
	}

	return characteristic, size, true, nil
}

// parseDM reads one die modifier as the Prior Service Table prints it: an
// amount, and the characteristic and threshold that earn it. The data file
// spells the condition the way p. 10 does, "Intelligence 8+".
func parseDM(amount int, condition string) (DM, error) {
	name, threshold, ok := strings.CutLast(condition, " ")
	if !ok {
		return DM{}, fmt.Errorf("die modifier %q: want a characteristic and a threshold", condition)
	}

	characteristic, err := parseCharacteristic(name)
	if err != nil {
		return DM{}, fmt.Errorf("die modifier %q: %w", condition, err)
	}

	target, err := traveller.ParseTarget(threshold)
	if err != nil {
		return DM{}, fmt.Errorf("die modifier %q: %w", condition, err)
	}

	return DM{Amount: amount, Characteristic: characteristic, Threshold: target}, nil
}

// parseTableResult reads one cell of the Acquired Skills Table (p. 11).
// P. 12 names exactly three kinds a cell can be, and this is where a cell
// that is none of them is caught.
func parseTableResult(cell string, normalize map[string]string) (traveller.TableResult, error) {
	characteristic, size, isAlteration, err := parseAlteration(cell)
	if err != nil {
		return nil, err
	}
	if isAlteration {
		return traveller.AlterationResult{Characteristic: characteristic, Delta: size}, nil
	}

	if category, ok := weaponCategories[expand(cell, normalize)]; ok {
		return traveller.WeaponPickResult{Category: category}, nil
	}

	return traveller.SkillResult{Name: traveller.SkillName(expand(cell, normalize))}, nil
}

// parseBenefitRow reads one cell of Mustering Out Table 1 (p. 9).
func parseBenefitRow(cell string, normalize map[string]string) (traveller.BenefitRow, error) {
	if cell == emDash {
		return traveller.NoBenefit{}, nil
	}

	characteristic, size, isAlteration, err := parseAlteration(cell)
	if err != nil {
		return nil, err
	}
	if isAlteration {
		return traveller.AlterationBenefit{Characteristic: characteristic, Delta: size}, nil
	}

	spelled := expand(cell, normalize)

	if class, ok := passageClasses[spelled]; ok {
		return traveller.PassageBenefit{Class: class}, nil
	}
	if kind, ok := shipKinds[spelled]; ok {
		return traveller.ShipBenefit{Kind: kind}, nil
	}
	if spelled == travellersAid {
		return traveller.TravellersAidBenefit{}, nil
	}

	// A weapon row prints the bare category: "Blade", "Gun". P. 9: "Weapon
	// benefits must be declared as to type immediately."
	if category, ok := weaponCategories[spelled+" Combat"]; ok {
		return traveller.WeaponCategoryBenefit{Category: category}, nil
	}

	return nil, fmt.Errorf("%q is not a benefit any row of Table 1 prints", cell)
}

// emDash is what Table 1 prints where a service has no benefit at that row.
// It is a hyphen in the data file and an em dash on the page; text
// extraction turns it into the digit 4, which is why nothing here is ever
// read from extracted text.
const emDash = "-"

const travellersAid = "Travellers' Aid"

// expand applies E012: a printed abbreviation becomes the name its own
// description carries. A string with no entry is already spelled out.
func expand(printed string, normalize map[string]string) string {
	if spelled, ok := normalize[printed]; ok {
		return spelled
	}

	return printed
}
