package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/philoserf/ctchargen/chargen"
	"github.com/philoserf/ctchargen/render"
	"github.com/philoserf/ctchargen/traveller"
)

// errNoAnswer reports that the input ended while the procedure was still
// asking. It is an error rather than a default, because a character finished
// by a decision nobody made is not the character anybody asked for.
var errNoAnswer = errors.New("the input ended while the procedure was still asking")

// player is the Decider a person drives: the third implementation, after the
// auto policy and the scripted deciders the tests use, and the only one that
// cannot be replayed.
//
// It never invents an answer. Every question it asks is one the engine
// asked, with exactly the alternatives the engine offered, and an answer it
// cannot read is asked again rather than guessed at.
type player struct {
	in  *bufio.Scanner
	out io.Writer

	// seed is carried only so that the line offered when the input ends can
	// name it. A drawn seed is otherwise nowhere the player can reach, and
	// it is the half of a resumption he cannot retype from memory.
	seed uint64

	// replay is what --answers supplied, taken before the input is read.
	// When it runs out the procedure asks the next question as normal -
	// that is the whole point, and a resumption that stopped at the end of
	// the list would be a replay rather than a way back in.
	replay []int

	// given is every answer accepted so far, in order, which is what a
	// resumption needs and what nothing else records: the log holds what was
	// chosen, not which number was typed to choose it.
	given []int

	// wrote is the first failure writing to the player, kept so that a
	// broken pipe stops the generation at the next question rather than
	// being swallowed a dozen times over.
	wrote error
}

func newPlayer(in io.Reader, out io.Writer, seed uint64, replay []int) *player {
	if in == nil {
		// Nothing to read from is not a reason to invent answers; the first
		// question asked will report that the input ended.
		in = strings.NewReader("")
	}

	return &player{in: bufio.NewScanner(in), out: out, seed: seed, replay: replay}
}

// pick asks which of a list the player wants and hands back the thing
// itself, so that every choice over a list is one function rather than the
// same "ask, then index" pair written out once per choice point - and its
// failure is one branch rather than one per choice point.
func pick[T any](p *player, question string, from []T, name func(T) string) (T, error) {
	options := make([]string, 0, len(from))
	for _, v := range from {
		options = append(options, name(v))
	}

	chosen, err := p.choose(question, options)
	if err != nil {
		var none T

		return none, err
	}

	return from[chosen], nil
}

func (p *player) Service(from []traveller.EnlistmentOffer) (traveller.ServiceName, error) {
	offer, err := pick(p, "Which service will you try to enlist in? (p. 10)", from,
		func(o traveller.EnlistmentOffer) string {
			return fmt.Sprintf("%v (enlistment %v, and you earn %+d)", o.Service, o.Target, o.DM)
		})

	return offer.Service, err
}

func (p *player) SubmitToDraft() (bool, error) {
	return p.confirm("Rejected. Submit to the draft? (p. 5)")
}

func (p *player) AttemptCommission() (bool, error) {
	return p.confirm("Attempt a commission this term? (p. 6)")
}

func (p *player) AttemptPromotion() (bool, error) {
	return p.confirm("Attempt a promotion this term? (p. 6)")
}

func (p *player) SkillTable(from []traveller.SkillTable) (traveller.SkillTable, error) {
	return pick(p, "Which skills table? It is designated before the die (p. 11)",
		from, skillTableMenu)
}

// skillTableMenu names a skills table for a person reading a list at speed.
//
// Two of them are identical up to a parenthesis - "Advanced Education Table"
// and "Advanced Education Table (education 8+)" - which is honest and easy to
// pick wrong. What separates them is moved to the front, because the eye scans
// down the left of a numbered list and never reaches a parenthesis.
//
// The wording here is the menu's, not the record's, for the reason
// MusterWeapon's is: they address two readers and nothing compares them.
func skillTableMenu(table traveller.SkillTable) string {
	if table == traveller.AdvancedEducationEight {
		return "Education 8+ table (the second Advanced Education table)"
	}

	return table.String()
}

func (p *player) Weapon(
	category traveller.WeaponCategory, from []traveller.WeaponName, _ chargen.Vary,
) (traveller.WeaponName, error) {
	question := fmt.Sprintf("%v: name the weapon, immediately (pp. 11-13)", category)

	return pick(p, question, from, func(w traveller.WeaponName) string { return string(w) })
}

func (p *player) ReenlistIntent(from []traveller.Intent) (traveller.Intent, error) {
	return pick(p, "The term is over. What now? (pp. 6-7, 21)", from, traveller.Intent.String)
}

func (p *player) MusterTable(from []traveller.MusterTable) (traveller.MusterTable, error) {
	return pick(p, "Which mustering out table? It is designated before the die (p. 9)",
		from, traveller.MusterTable.String)
}

func (p *player) MusterTable1DM() (bool, error) {
	return p.confirm("Your rank allows +1 on this table. Take it? (p. 9)")
}

func (p *player) MusterTable2DM() (bool, error) {
	return p.confirm("Your gambling expertise allows +1 on this table. Take it? (p. 9)")
}

// weaponOffer pairs one entry of the p. 22 weapon row with what taking it
// means, so that the wording and the benefit are built in one place and
// nothing has to work out afterwards which list an index fell into.
type weaponOffer struct {
	text  string
	means traveller.WeaponBenefit
}

func (p *player) MusterWeapon(
	category traveller.WeaponCategory, from, received []traveller.WeaponName,
) (traveller.WeaponBenefit, error) {
	offers := make([]weaponOffer, 0, len(from)+len(received))
	for _, weapon := range from {
		offers = append(offers, weaponOffer{
			text: string(weapon), means: traveller.TakeWeapon{Weapon: weapon},
		})
	}

	// P. 22 offers the expertise only in lieu of a weapon "of exactly the
	// same type", so it appears only for one already received.
	//
	// The wording here is the menu's, not the record's: a person choosing
	// from a list reads "+1 expertise in the Dagger" better than the
	// record's "expertise in Dagger". They are deliberately two, because
	// they address two readers, and nothing compares them - the engine's
	// gate folds the benefit and compares that.
	for _, held := range received {
		offers = append(offers, weaponOffer{
			text: "+1 expertise in the " + string(held), means: traveller.TakeExpertise{Weapon: held},
		})
	}

	question := fmt.Sprintf("%v: take a weapon, or expertise in one already received (p. 22)", category)

	chosen, err := pick(p, question, offers, func(o weaponOffer) string { return o.text })

	return chosen.means, err
}

func (p *player) AssumeTitle(title traveller.Title) (bool, error) {
	return p.confirm(fmt.Sprintf(
		"Your social standing makes you %v. Assume the title? (p. 5; Book 3 p. 22)", title,
	))
}

// sayf writes to the player, remembering the first failure.
func (p *player) sayf(format string, args ...any) {
	if p.wrote != nil {
		return
	}

	_, p.wrote = fmt.Fprintf(p.out, format, args...)
}

// watch shows the player the procedure between his questions.
//
// Every event now renders to something: the choices used to come back empty
// and were skipped here, which is what left holes in the numbering. Nothing
// returns nothing, so there is no longer anything to skip - writeEvent's own
// default writes a line even for a kind this build does not know.
func (p *player) watch(event traveller.Event) {
	p.sayf("%s", render.EventLine(event))
}

// choose asks for one of a numbered list and returns its index.
//
// The list is printed once. A bad answer used to re-print the whole thing,
// which buried the complaint under six lines of menu and read as though the
// tool had said nothing at all - the reported behaviour. Now the complaint
// names what was typed and the prompt comes back on its own.
func (p *player) choose(question string, options []string) (int, error) {
	p.sayf("\n%s\n", question)

	for i, option := range options {
		p.sayf("  %d) %s\n", i+1, option)
	}

	for {
		replayed, ok, err := p.next(len(options))
		if err != nil {
			return 0, err
		}

		if ok {
			p.sayf("> %d\n\n", replayed)

			return replayed - 1, p.wroteErr()
		}

		p.sayf("> ")

		if p.wrote != nil {
			return 0, p.wroteErr()
		}

		if !p.in.Scan() {
			return 0, p.stopped()
		}

		typed := strings.TrimSpace(p.in.Text())

		chosen, err := strconv.Atoi(typed)
		if err == nil && chosen >= 1 && chosen <= len(options) {
			p.sayf("\n")

			p.given = append(p.given, chosen)

			return chosen - 1, nil
		}

		p.sayf("  %q is not one of them; answer with a number from 1 to %d\n",
			typed, len(options))
	}
}

// next takes the head of a replayed list, if there is one left.
//
// A replayed answer is echoed at the prompt as though it had been typed, so
// that a resumed session reads the same as the one it continues rather than
// jumping to a question with no visible answers above it.
func (p *player) next(options int) (int, bool, error) {
	if len(p.replay) == 0 {
		return 0, false, nil
	}

	chosen := p.replay[0]

	p.replay = p.replay[1:]

	// Out of range for this question, which means the list belongs to some
	// other run. Saying so beats answering the question with it, and beats
	// falling through to the prompt, which would spend the next answer on
	// the retry.
	if chosen < 1 || chosen > options {
		return 0, false, fmt.Errorf(
			"%w: answer %d of --%s is %d, and this question offers 1 to %d",
			errUsage, len(p.given)+1, answersFlag, chosen, options,
		)
	}

	p.given = append(p.given, chosen)

	return chosen, true, nil
}

// leftover refuses a list longer than the run had questions for.
//
// The same signal as an answer out of range, and the same answer to it: the
// questions a seed asks are fixed, so a resumption that replays its own
// answers consumes every one of them. Anything left means the list came from
// another run, and a character built from the first half of it is wrong in a
// way nothing on the sheet shows.
func (p *player) leftover() error {
	if len(p.replay) == 0 {
		return nil
	}

	return fmt.Errorf("%w: the procedure ran out of questions with %d of --%s unspent, "+
		"so the list belongs to another run", errUsage, len(p.replay), answersFlag)
}

func (p *player) wroteErr() error {
	if p.wrote == nil {
		return nil
	}

	return fmt.Errorf("asking the player: %w", p.wrote)
}

// stopped reports why the answers ran out, and offers the way back in.
//
// A scanner returns false for two different reasons and only one of them is
// the input ending: a line past bufio's 64KB bound, or a read failure on the
// reader beneath, both come back the same way. Blaming the input for ending
// sends the reader looking in the wrong place.
//
// Either way the half-built character is lost - it is not a record, and does
// not match the schema - but the seed and the answers are enough to walk back
// to the same question, and those are what a long session cannot retype. So
// the offer is made before the two are told apart: a read that failed is the
// stop the operator did not choose, and the one that most needs the way back.
func (p *player) stopped() error {
	p.sayf("\n%s\n", p.resumeLine())

	err := p.in.Err()
	if err != nil {
		return fmt.Errorf("reading the answer: %w", err)
	}

	return errNoAnswer
}

// resumeLine is the flags that walk back to the question the input ended on.
//
// It names flags to add rather than a whole command line, because the
// operator's own is in his shell history and this one cannot know what else
// he typed - and it says "the same command" for that reason. An answer is an
// index into the list a question offered, so a re-run that drops --service
// or --name asks a different sequence and spends the answers on it. Both
// values are digits, so neither needs quoting and neither can carry
// anything.
func (p *player) resumeLine() string {
	if len(p.given) == 0 {
		return fmt.Sprintf(
			"Nothing was answered. The same command with --seed %d walks the same character again.",
			p.seed,
		)
	}

	answered := make([]string, 0, len(p.given))
	for _, chosen := range p.given {
		answered = append(answered, strconv.Itoa(chosen))
	}

	return fmt.Sprintf(
		"Re-run the same command with --seed %d --answers %s to pick up at this question.",
		p.seed, strings.Join(answered, ","),
	)
}

// confirm asks a question the procedure puts as yes or no.
func (p *player) confirm(question string) (bool, error) {
	chosen, err := p.choose(question, []string{"yes", "no"})

	return chosen == 0, err
}
