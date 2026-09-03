package chargen

import (
	"slices"

	"github.com/philoserf/ctchargen/traveller"
)

// Character is the record a generation produces, complete whether it ended
// in mustering out or in death (FR8).
//
// A deliberate delta from sibling tools: Books 1-3 have no calendar and no
// birthdate rules, so the record carries an age and a count of terms, not a
// date. That is the books, not an omission.
type Character struct {
	Name    string
	Profile traveller.Profile
	Age     traveller.Age
	Terms   int

	// Enlistment is how the character entered a service, or that he
	// declined the draft and never did (p. 5).
	Enlistment traveller.Enlistment
	Service    traveller.ServiceName
	Served     bool
	Rank       traveller.Rank
	RankTitle  string

	Skills   []traveller.Skill
	Benefits Benefits
	Pension  traveller.Credits

	// Departure is how the service ended. It is nil for a civilian, who
	// never joined one.
	Departure traveller.Departure

	// Title is the nobility a final Social Standing of 11 or greater
	// confers, assessed once at the end of generation (E011).
	Title Title

	Inputs Inputs
	Events []traveller.Event
	Errata []traveller.Erratum
	Build  string
}

// Title is what the record carries about nobility: whether the character was
// eligible, which rank the value confers, and whether he assumed it. A
// character who died is assessed but not asked (E011).
type Title struct {
	Eligible bool
	Rank     traveller.Title
	Assumed  bool
}

// Ship is a starship received as a mustering out benefit (pp. 22-23).
type Ship struct {
	Kind traveller.ShipKind

	// Years is the ship's age. A Free Trader arrives new and ages ten years
	// with each further receipt; a Scout ship's age the page does not give.
	Years int

	// PaymentYears is how many years of payments remain on a Free Trader.
	// A Scout ship is held without title and carries none.
	PaymentYears int
}

// Benefits is what mustering out delivered (p. 9, pp. 21-23).
type Benefits struct {
	Cash          traveller.Credits
	Passages      []traveller.PassageClass
	TravellersAid bool
	Weapons       []traveller.WeaponName
	Ships         []Ship
}

// Inputs is everything a generation was given, so that a record says what
// made it.
type Inputs struct {
	Seed uint64
	Name string

	// Service is the service enlistment was attempted in, when one was
	// forced. It forces the attempt only: the throw is still made, and a
	// failed throw still goes to the draft (p. 5).
	Service traveller.ServiceName
	Forced  bool

	Career string
	Skills string
	Muster string
}

// addSkill records a skill, raising its level if it is already held. P. 12:
// "Each time an already acquired skill is again acquired, the level of the
// skill is increased by 1."
func (c *Character) addSkill(name traveller.SkillName) int {
	for i := range c.Skills {
		if c.Skills[i].Name == name {
			c.Skills[i].Level++

			return c.Skills[i].Level
		}
	}

	c.Skills = append(c.Skills, traveller.Skill{Name: name, Level: 1})

	return 1
}

// has reports whether the character holds a skill at any level. Gambling
// expertise is what allows the +1 on Mustering Out Table 2 (p. 9).
func (c *Character) has(name traveller.SkillName) bool {
	return slices.ContainsFunc(c.Skills, func(s traveller.Skill) bool { return s.Name == name })
}

// sortSkills puts the skill list in a stable order, so a record reads the
// same way twice.
func (c *Character) sortSkills() {
	slices.SortFunc(c.Skills, func(a, b traveller.Skill) int {
		switch {
		case a.Name < b.Name:
			return -1
		case a.Name > b.Name:
			return 1
		default:
			return 0
		}
	})
}
