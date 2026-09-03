package chargen

import (
	"fmt"
	"slices"

	"github.com/philoserf/ctchargen/traveller"
)

// musterOut spends the benefit rolls the service earned (pp. 7, 9).
func (r *run) musterOut() error {
	rolls := r.tables.Muster.Rolls(r.char.Terms, r.char.Rank)
	if rolls == 0 {
		return nil
	}

	r.log.step(fmt.Sprintf("mustering out, %d rolls", rolls), "pp. 7, 9, 21-23")

	onTableTwo := 0

	for range rolls {
		table, err := r.chooseMusterTable(onTableTwo)
		if err != nil {
			return err
		}

		if table == traveller.TableTwo {
			onTableTwo++
		}

		err = r.benefit(table)
		if err != nil {
			return err
		}
	}

	return nil
}

// chooseMusterTable designates the table before the die. P. 9: "A maximum of
// three rolls on table 2 are allowed per character; all remaining rolls must
// be on table 1."
func (r *run) chooseMusterTable(onTableTwo int) (traveller.MusterTable, error) {
	if onTableTwo >= r.tables.Muster.MaxOnTable2 {
		return traveller.TableOne, nil
	}

	table, err := r.decide.MusterTable(traveller.MusterTables[:])
	if err != nil {
		return table, fmt.Errorf("designating a mustering out table: %w", err)
	}

	return table, nil
}

// benefit takes one roll on the designated table.
func (r *run) benefit(table traveller.MusterTable) error {
	modifier, err := r.musterModifier(table)
	if err != nil {
		return err
	}

	face := r.roll.Die()

	row := face + modifier
	seq := r.log.dieWithModifier(table.String(), face, modifier)

	benefit, cash, err := r.service.Row(row)
	if err != nil {
		return fmt.Errorf("consulting %v: %w", table, err)
	}

	if table == traveller.TableTwo {
		r.char.Benefits.Cash += cash
		r.log.outcomef(seq, nil, "%v, bringing the total to %v", cash, r.char.Benefits.Cash)

		return nil
	}

	applied := &applyBenefit{run: r, because: seq}

	err = benefit.Fold(applied)
	if err != nil {
		return fmt.Errorf("taking a benefit: %w", err)
	}

	return nil
}

// musterModifier offers the +1 each table allows, where the character has
// earned it (p. 9).
func (r *run) musterModifier(table traveller.MusterTable) (int, error) {
	if table == traveller.TableOne {
		// The rank is p. 9's number - "Characters with rank 5 or 6 may add
		// +1 to their rolls on this table" - so it is data with a cite, not
		// a constant written here.
		if int(r.char.Rank) < r.tables.Muster.MinRankForTable1Modifier {
			return 0, nil
		}

		take, err := r.decide.MusterTable1DM()
		if err != nil {
			return 0, fmt.Errorf("the table 1 modifier: %w", err)
		}

		if take {
			return r.tables.Muster.Table1DMFromRank5or6, nil
		}

		return 0, nil
	}

	if !r.char.has("Gambling") {
		return 0, nil
	}

	take, err := r.decide.MusterTable2DM()
	if err != nil {
		return 0, fmt.Errorf("the table 2 modifier: %w", err)
	}

	if take {
		return r.tables.Muster.Table2DMFromGambling, nil
	}

	return 0, nil
}

// assessTitle assesses nobility once, at the end of generation, against the
// final Social Standing (E011). A character who died is assessed but not
// asked: eligibility is a condition of the value, while assuming a title is
// an act he does not perform.
func (r *run) assessTitle() error {
	social := r.char.Profile[traveller.SocialStanding]

	rank, eligible := r.tables.TitleFor(social)
	if !eligible {
		return nil
	}

	r.log.step("titles", "p. 5; Book 3 p. 22")

	r.char.Title = Title{Eligible: true, Rank: rank}

	if r.dead {
		r.log.outcomef(0, []traveller.Erratum{traveller.E011},
			"held a social standing of %d at death, which is %v", social, rank)

		return nil
	}

	assume, err := r.decide.AssumeTitle(rank)
	if err != nil {
		return fmt.Errorf("assuming a title: %w", err)
	}

	r.char.Title.Assumed = assume

	r.log.outcomef(0, []traveller.Erratum{traveller.E011},
		"social standing %d confers %v, %s assumed", social, rank,
		map[bool]string{true: "and it is", false: "and it is not"}[assume])

	return nil
}

// applyBenefit is the fold that takes one row of Mustering Out Table 1.
// Adding an eighth kind to the sum stops this compiling until it is handled.
type applyBenefit struct {
	run     *run
	because int
}

func (a *applyBenefit) Cash(amount traveller.Credits) error {
	a.run.char.Benefits.Cash += amount
	a.run.log.outcomef(a.because, nil, "%v", amount)

	return nil
}

func (a *applyBenefit) Passage(class traveller.PassageClass) error {
	a.run.char.Benefits.Passages = append(a.run.char.Benefits.Passages, class)
	a.run.log.outcomef(a.because, nil, "a %v", class)

	return nil
}

func (a *applyBenefit) Alteration(characteristic traveller.Characteristic, delta int) error {
	return a.run.apply(traveller.AlterationResult{Characteristic: characteristic, Delta: delta},
		a.because, nil)
}

func (a *applyBenefit) WeaponPick(category traveller.WeaponCategory) error {
	list := a.run.tables.Weapons(category)
	received := a.run.receivedIn(category)

	taken, err := a.run.decide.MusterWeapon(category, list, received)
	if err != nil {
		return fmt.Errorf("taking a weapon benefit: %w", err)
	}

	took := &takeWeapon{run: a.run, because: a.because, category: category}

	err = taken.Fold(took)
	if err != nil {
		return fmt.Errorf("taking a weapon benefit: %w", err)
	}

	return nil
}

func (a *applyBenefit) TravellersAid() error {
	// P. 22: membership "may be achieved only once per character. If a die
	// roll indicates membership after it has already been achieved, the die
	// roll is wasted, and the character receives nothing for it."
	if a.run.char.Benefits.TravellersAid {
		a.run.log.outcomef(a.because, nil, "Travellers' Aid again, which is wasted")

		return nil
	}

	a.run.char.Benefits.TravellersAid = true
	a.run.log.outcomef(a.because, nil, "Travellers' Aid Society membership")

	return nil
}

func (a *applyBenefit) Ship(kind traveller.ShipKind) error {
	return a.run.receiveShip(a.because, kind)
}

func (a *applyBenefit) Nothing() error {
	a.run.log.outcomef(a.because, nil, "nothing: the table prints a dash for this service")

	return nil
}

// receivedIn is the weapons of one category already taken as benefits. P. 22
// bounds the expertise option twice: only "in a weapon received as a
// benefit", and only in lieu of one "of exactly the same type".
func (r *run) receivedIn(category traveller.WeaponCategory) []traveller.WeaponName {
	list := r.tables.Weapons(category)

	var received []traveller.WeaponName

	for _, held := range r.char.Benefits.Weapons {
		if slices.Contains(list, held) && !slices.Contains(received, held) {
			received = append(received, held)
		}
	}

	return received
}

// takeWeapon is the fold that records what a repeat weapon row was taken as.
type takeWeapon struct {
	run      *run
	because  int
	category traveller.WeaponCategory
}

func (t *takeWeapon) TakeWeapon(weapon traveller.WeaponName) error {
	if !weaponInList(t.run.tables.Weapons(t.category), weapon) {
		return fmt.Errorf("%w: %q is not on the %v list", errNotAWeapon, weapon, t.category)
	}

	t.run.char.Benefits.Weapons = append(t.run.char.Benefits.Weapons, weapon)
	t.run.log.outcomef(t.because, nil, "a %s", weapon)

	return nil
}

func (t *takeWeapon) TakeExpertise(weapon traveller.WeaponName) error {
	if !slices.Contains(t.run.receivedIn(t.category), weapon) {
		return fmt.Errorf("%w: expertise may only be taken in a weapon received as a benefit, and %q was not",
			errNotAWeapon, weapon)
	}

	level := t.run.char.addSkill(traveller.SkillName(weapon))
	t.run.log.outcomef(t.because, nil, "expertise instead: %s-%d", weapon, level)

	return nil
}

// The Free Trader's terms, from p. 22: the ship is received "liable for the
// monthly payments (which amount to about 150,000 credits) for the next
// forty years", and each further receipt "is considered to represent actual
// possession of the ship for a ten-year period. The ship is 10 years older,
// and the total payment term is reduced by ten years."
const (
	freeTraderPaymentYears = 40
	freeTraderYearsPerRoll = 10
)

// receiveShip takes a starship benefit (pp. 22-23).
func (r *run) receiveShip(because int, kind traveller.ShipKind) error {
	for i := range r.char.Benefits.Ships {
		if r.char.Benefits.Ships[i].Kind == kind {
			return r.receiveShipAgain(because, i)
		}
	}

	ship := Ship{Kind: kind, Tons: r.tables.Hull(kind)}
	if kind == traveller.FreeTrader {
		ship.PaymentYears = freeTraderPaymentYears
	}

	r.char.Benefits.Ships = append(r.char.Benefits.Ships, ship)
	r.log.outcomef(because, nil, "a %v", kind)

	return nil
}

// receiveShipAgain applies a duplicate. A Free Trader's repeats pay the ship
// off ten years at a time and age it by as much; a Scout ship's are lost -
// p. 23: "Only one scout ship may be acquired by a character, and throws
// resulting in additional ships are lost."
func (r *run) receiveShipAgain(because, at int) error {
	ship := &r.char.Benefits.Ships[at]

	if ship.Kind == traveller.ScoutShip {
		r.char.DuplicateShips++
		r.log.outcomef(because, nil, "a second scout ship, which is lost")

		return nil
	}

	ship.Years += freeTraderYearsPerRoll

	ship.PaymentYears = max(0, ship.PaymentYears-freeTraderYearsPerRoll)

	r.log.outcomef(because, nil,
		"the Free Trader again: %d years old, with %d years of payments left",
		ship.Years, ship.PaymentYears)

	return nil
}
