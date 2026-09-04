package render

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/philoserf/ctchargen/chargen"
	"github.com/philoserf/ctchargen/traveller"
)

// Sheet renders a character the way Book 1 summarises one: the UPP line,
// then skills and possessions, as p. 25 does for Jamison.
func Sheet(character *chargen.Character) (string, error) {
	projected, err := project(character)
	if err != nil {
		return "", err
	}

	return sheetOf(projected)
}

// SheetFrom renders a record that was written earlier, which is what the
// render subcommand reads.
//
// It goes through the same renderer as Sheet rather than a second one over
// the same values: a sheet describes the record, so the record is what it
// should be written from, and one renderer cannot disagree with itself.
func SheetFrom(text []byte) (string, error) {
	projected, err := decode(text)
	if err != nil {
		return "", err
	}

	return sheetOf(projected)
}

// sheetOf writes the sheet, and ends it with how to get the character back.
//
// The footer is last because it is not the character: a referee reads the
// headline and the possessions, and reaches for the seed only once he has
// decided he wants this one.
func sheetOf(r record) (string, error) {
	var out strings.Builder

	fmt.Fprintf(&out, "# %s\n\n", nameOrBlank(r.Name))
	fmt.Fprintf(&out, "%s\n\n", headline(r))

	writeSection(&out, "Skills", skillLines(r))
	writeSection(&out, "Possessions", possessionLines(r))
	writeSection(&out, "Service record", serviceLines(r))

	if len(r.Errata) > 0 {
		writeSection(&out, "Readings applied", []string{strings.Join(r.Errata, ", ")})
	}

	told, err := provenance(r, sheetRendering)
	if err != nil {
		return "", err
	}

	fmt.Fprintf(&out, "---\n\n%s\n", told)

	return out.String(), nil
}

func nameOrBlank(name string) string {
	if name == "" {
		return "(unnamed)"
	}

	return name
}

// age reads as whole years, or years and months where a medical crisis
// recovery added some (pp. 7-8).
func age(a ageRecord) string {
	if a.Months == 0 {
		return strconv.Itoa(a.Years)
	}

	return fmt.Sprintf("%d years %d months", a.Years, a.Months)
}

// headline is the line the book itself leads with: the UPP, the age, and
// what the character is.
func headline(r record) string {
	parts := []string{"UPP " + r.UPP, "age " + age(r.Age)}

	if r.Service == "" {
		return strings.Join(append(parts, "civilian, no prior service"), ", ")
	}

	service := fmt.Sprintf("%s, %d terms", r.Service, r.Terms)
	if r.RankTitle != "" {
		service = fmt.Sprintf("%s %s, %d terms", r.Service, r.RankTitle, r.Terms)
	}

	parts = append(parts, service)

	if r.Title != nil && r.Title.Assumed {
		parts = append(parts, r.Title.Rank)
	}

	if r.Departure != nil {
		parts = append(parts, r.Departure.How)
	}

	return strings.Join(parts, ", ")
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

func skillLines(r record) []string {
	lines := make([]string, 0, len(r.Skills))
	for _, skill := range r.Skills {
		lines = append(lines, fmt.Sprintf("%s-%d", skill.Name, skill.Level))
	}

	return lines
}

func possessionLines(r record) []string {
	var lines []string

	if r.Benefits.Cash > 0 {
		lines = append(lines, credits(r.Benefits.Cash))
	}

	lines = append(lines, r.Benefits.Passages...)

	if r.Benefits.TravellersAid {
		lines = append(lines, "Travellers' Aid Society membership")
	}

	lines = append(lines, r.Benefits.Weapons...)

	for _, ship := range r.Benefits.Ships {
		lines = append(lines, shipLine(ship))
	}

	if r.Pension > 0 {
		lines = append(lines, credits(r.Pension)+" a year in retirement pay")
	}

	return lines
}

func credits(amount int64) string { return fmt.Sprintf("CR %d", amount) }

// shipLine reads a ship's terms off its kind, which is what the terms turn
// on: a scout ship is held without title (p. 23) whatever its age, and a
// Free Trader is owned whatever is left to pay. Inferring the one from the
// other's numbers would call a new, unencumbered Free Trader a scout ship.
func shipLine(ship shipRecord) string {
	if ship.Kind == traveller.ScoutShip.String() {
		return fmt.Sprintf("%s, %d tons, held in constructive possession", ship.Kind, ship.Tons)
	}

	if ship.PaymentYears == 0 {
		return fmt.Sprintf("%s, %d tons, %d years old, owned free and clear",
			ship.Kind, ship.Tons, ship.Years)
	}

	return fmt.Sprintf("%s, %d tons, %d years old, with %d years of payments left",
		ship.Kind, ship.Tons, ship.Years, ship.PaymentYears)
}

func serviceLines(r record) []string {
	if r.Service == "" {
		return []string{r.Enlistment.How}
	}

	lines := []string{fmt.Sprintf("%s, %s", r.Enlistment.Service, r.Enlistment.How)}

	if r.Departure == nil {
		return lines
	}

	switch {
	case r.Departure.Characteristic != "":
		return append(lines, fmt.Sprintf("%s in term %d (%s reached zero)",
			r.Departure.How, r.Terms, r.Departure.Characteristic))
	case r.Departure.Fatal:
		return append(lines, fmt.Sprintf("%s in term %d", r.Departure.How, r.Terms))
	default:
		return append(lines, fmt.Sprintf("%s after term %d", r.Departure.How, r.Terms))
	}
}
