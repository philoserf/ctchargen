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

// An Option is an optional input to a generation: dice of the caller's own,
// or somebody watching the record as it is written.
//
// Options rather than a function per combination, so that the default -
// seeded dice and nobody watching - stays the shortest call, and the engine
// does not grow an entry point every time it gains an optional input.
type Option func(*run)

// WithRoller walks the procedure against a roller of the caller's own, which
// is how a test replays a particular path: a career past the Aging Table's
// last printed term, say, that no seed would reach.
func WithRoller(roller Roller) Option {
	return func(r *run) { r.roll = roller }
}

// WithObserver calls watch with each event as the record is written, which
// is how interactive mode shows the procedure between its questions.
//
// It returns nothing, so a watcher cannot change what it watches.
func WithObserver(watch func(traveller.Event)) Option {
	return func(r *run) { r.log.watch = watch }
}

// WithAnswerer names who the decider stands for, which every choice event
// then records (FR9). A run says the policy answered unless it is told that
// a person did, so a record read back names the kind of thing that made
// each choice rather than assuming the automatic one.
func WithAnswerer(who traveller.DecidedBy) Option {
	return func(r *run) { r.by = who }
}

// Generate walks Book 1's character generation procedure, pp. 4-25, rolling
// from the seed the inputs carry unless a roller is given.
//
// The order within a term is the exposition's, not the worked example's
// (E002): survival, commission, promotion, skills, reenlistment, and then
// the aging round at the end of the term (E006). That order is what a seed
// means, so it is a fixed choice rather than an incidental one.
func Generate(in Inputs, decider Decider, options ...Option) (*Character, error) {
	tables, err := rules.Load()
	if err != nil {
		return nil, fmt.Errorf("loading the rules: %w", err)
	}

	record := newLog()

	run := &run{
		tables: tables,
		roll:   dice.New(in.Seed),
		log:    record,
		by:     traveller.ByPolicy,
		char: &Character{
			Name: in.Name, Age: traveller.NewAge(startingAge), Inputs: in, Ruleset: Ruleset,
		},
	}

	for _, option := range options {
		option(run)
	}

	invalid := validate(decider)
	if invalid != nil {
		return nil, invalid
	}

	run.decide = logging{to: decider, by: run.by, log: record}

	err = run.generate()
	if err != nil {
		return nil, err
	}

	run.char.sortSkills()

	run.char.Events = record.events
	run.char.Errata = record.stamped()

	return run.char, nil
}

// selfChecking is a decider that can be handed a configuration it does not
// recognize, and knows it.
type selfChecking interface{ Validate() error }

// validate asks a decider whether it was built out of things that exist.
//
// The engine already refuses an answer from outside the offered set (FR9);
// this refuses a decider that could not have produced a legal answer in the
// first place. Before it existed, `Policy{Muster: "gold"}` generated a whole
// character - the strategy matched no branch, so it silently behaved as
// another one - and wrote "gold" into the record's inputs, where no row of
// POLICY.md answers to it. The command validated its flags and the engine
// trusted whatever it was handed, which is the same shape of gap as trusting
// an answer the procedure never offered.
func validate(decider Decider) error {
	checkable, ok := decider.(selfChecking)
	if !ok {
		return nil
	}

	invalid := checkable.Validate()
	if invalid != nil {
		return fmt.Errorf("the decider: %w", invalid)
	}

	return nil
}

// run is one generation in progress.
type run struct {
	tables *rules.Rules
	roll   Roller
	decide Decider
	by     traveller.DecidedBy
	log    *log
	char   *Character

	service  rules.Service
	drafted  bool
	dead     bool
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
