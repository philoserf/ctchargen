package render

import (
	"fmt"
	"strings"

	"github.com/philoserf/ctchargen/chargen"
	"github.com/philoserf/ctchargen/traveller"
)

// Sheet renders a character the way Book 1 summarises one: the UPP line,
// then skills and possessions, as p. 25 does for Jamison.
func Sheet(character *chargen.Character) string {
	var out strings.Builder

	fmt.Fprintf(&out, "# %s\n\n", nameOrBlank(character.Name))
	fmt.Fprintf(&out, "%s\n\n", headline(character))

	writeSection(&out, "Skills", skillLines(character))
	writeSection(&out, "Possessions", possessionLines(character))
	writeSection(&out, "Service record", serviceLines(character))

	if len(character.Errata) > 0 {
		lines := make([]string, 0, len(character.Errata))
		for _, erratum := range character.Errata {
			lines = append(lines, erratum.String())
		}

		writeSection(&out, "Readings applied", []string{strings.Join(lines, ", ")})
	}

	return out.String()
}

func nameOrBlank(name string) string {
	if name == "" {
		return "(unnamed)"
	}

	return name
}

// headline is the line the book itself leads with: the UPP, the age, and
// what the character is.
func headline(character *chargen.Character) string {
	parts := []string{"UPP " + character.Profile.UPP(), fmt.Sprintf("age %v", character.Age)}

	if !character.Served {
		return strings.Join(append(parts, "civilian, no prior service"), ", ")
	}

	service := fmt.Sprintf("%v, %d terms", character.Service, character.Terms)
	if character.RankTitle != "" {
		service = fmt.Sprintf("%v %s, %d terms", character.Service, character.RankTitle, character.Terms)
	}

	parts = append(parts, service)

	if character.Title.Assumed {
		parts = append(parts, character.Title.Rank.String())
	}

	return strings.Join(append(parts, foldDeparture(character.Departure).How), ", ")
}

func writeSection(out *strings.Builder, heading string, lines []string) {
	if len(lines) == 0 {
		return
	}

	fmt.Fprintf(out, "## %s\n\n", heading)

	for _, line := range lines {
		fmt.Fprintf(out, "- %s\n", line)
	}

	out.WriteString("\n")
}

func skillLines(character *chargen.Character) []string {
	lines := make([]string, 0, len(character.Skills))
	for _, skill := range character.Skills {
		lines = append(lines, skill.String())
	}

	return lines
}

func possessionLines(character *chargen.Character) []string {
	var lines []string

	if character.Benefits.Cash > 0 {
		lines = append(lines, character.Benefits.Cash.String())
	}

	for _, passage := range character.Benefits.Passages {
		lines = append(lines, passage.String())
	}

	if character.Benefits.TravellersAid {
		lines = append(lines, "Travellers' Aid Society membership")
	}

	for _, weapon := range character.Benefits.Weapons {
		lines = append(lines, string(weapon))
	}

	for _, ship := range character.Benefits.Ships {
		lines = append(lines, shipLine(ship))
	}

	if character.Pension > 0 {
		lines = append(lines, fmt.Sprintf("%v a year in retirement pay", character.Pension))
	}

	return lines
}

func shipLine(ship chargen.Ship) string {
	if ship.Kind == traveller.ScoutShip {
		return ship.Kind.String() + ", held in constructive possession"
	}

	if ship.PaymentYears == 0 {
		return fmt.Sprintf("%v, %d years old, owned free and clear", ship.Kind, ship.Years)
	}

	return fmt.Sprintf("%v, %d years old, with %d years of payments left",
		ship.Kind, ship.Years, ship.PaymentYears)
}

func serviceLines(character *chargen.Character) []string {
	if !character.Served {
		return []string{foldEnlistment(character.Enlistment).How}
	}

	enlistment := foldEnlistment(character.Enlistment)
	lines := []string{fmt.Sprintf("%s, %s", enlistment.Service, enlistment.How)}

	departure := foldDeparture(character.Departure)

	switch {
	case departure.Characteristic != "":
		return append(lines, fmt.Sprintf("%s in term %d (%s reached zero)",
			departure.How, character.Terms, departure.Characteristic))
	case departure.Fatal:
		return append(lines, fmt.Sprintf("%s in term %d", departure.How, character.Terms))
	default:
		return append(lines, fmt.Sprintf("%s after term %d", departure.How, character.Terms))
	}
}
