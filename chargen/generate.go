package chargen

import (
	"fmt"

	"github.com/philoserf/ctchargen/dice"
	"github.com/philoserf/ctchargen/rules"
	"github.com/philoserf/ctchargen/traveller"
)

// startingAge is where every character begins. P. 4: "All characters begin
// the game the same way, untrained, inexperienced, about 18 years of age."
const startingAge = 18

// Ruleset names the text this tool implements. All page cites in a record's
// generation log are to these three books and nothing else.
const Ruleset = "Classic Traveller, Books 1-3 (FFE reprint of the 1977 text)"

// Generate walks Book 1's character generation procedure, pp. 4-25, rolling
// from the seed the inputs carry.
//
// The order within a term is the exposition's, not the worked example's
// (E002): survival, commission, promotion, skills, reenlistment, and then
// the aging round at the end of the term (E006). That order is what a seed
// means, so it is a fixed choice rather than an incidental one.
func Generate(in Inputs, decider Decider) (*Character, error) {
	return GenerateWith(in, dice.New(in.Seed), decider)
}

// GenerateWith walks the same procedure against a roller of the caller's
// own, which is how a test replays a particular path - a career past the
// Aging Table's last printed term, say, that no seed would reach.
func GenerateWith(in Inputs, roller Roller, decider Decider) (*Character, error) {
	tables, err := rules.Load()
	if err != nil {
		return nil, fmt.Errorf("loading the rules: %w", err)
	}

	record := newLog()

	run := &run{
		tables: tables,
		roll:   roller,
		decide: logging{to: decider, by: traveller.ByPolicy, log: record},
		log:    record,
		char: &Character{
			Name: in.Name, Age: traveller.NewAge(startingAge), Inputs: in, Ruleset: Ruleset,
		},
		maxTerm: 0,
	}

	err = run.generate()
	if err != nil {
		return nil, err
	}

	run.char.sortSkills()

	run.char.Events = record.events
	run.char.Errata = record.stamped()

	return run.char, nil
}

// run is one generation in progress.
type run struct {
	tables *rules.Rules
	roll   Roller
	decide Decider
	log    *log
	char   *Character

	service  rules.Service
	drafted  bool
	dead     bool
	maxTerm  int
	eligible int
}

func (r *run) generate() error {
	r.rollProfile()

	err := r.enlist()
	if err != nil {
		return err
	}

	if !r.char.Served {
		return r.assessTitle()
	}

	err = r.serve()
	if err != nil {
		return err
	}

	if !r.dead {
		err := r.musterOut()
		if err != nil {
			return err
		}

		r.pension()
	}

	return r.assessTitle()
}

// rollProfile rolls the six characteristics in the order p. 4 rolls them.
func (r *run) rollProfile() {
	r.log.step("characteristics", "p. 4")

	for _, c := range traveller.Characteristics {
		first, second := r.roll.TwoDice()

		r.char.Profile[c] = first + second

		seq := r.log.dice(c.String(), first, second)
		r.log.outcomef(seq, nil, "%v %d", c, first+second)
	}

	r.log.outcomef(0, nil, "UPP %s at age %v", r.char.Profile.UPP(), r.char.Age)
}
