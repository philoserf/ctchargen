package chargen

import (
	"fmt"
	"slices"
	"sync"

	"github.com/philoserf/ctchargen/dice"
	"github.com/philoserf/ctchargen/service"
)

// Errata identifiers stamped into every record this engine writes: the
// docs/ERRATA.md readings that are wired into the procedure itself and so
// applied to every generation.
var appliedErrata = []string{
	"E001", // the draft is optional: "may submit" governs over "must submit" (p. 5)
	"E002", // per-term order is the exposition's, not the Jamison example's (pp. 5-6, 24)
}

// stampErratum records a data-dependent docs/ERRATA.md reading (one whose
// ambiguity this particular generation actually crossed).
func (g *generator) stampErratum(id string) {
	if !slices.Contains(g.char.Errata, id) {
		g.char.Errata = append(g.char.Errata, id)
	}
}

// The rule data is embedded, immutable, and read-only once loaded, so
// the parse and the whole load-time validation are done once per process
// rather than once per character — batch and the test suites generate
// many. The cached error is returned too: a build defect must keep
// failing every generation, not just the first.
var (
	registryOnce = sync.OnceValues(service.Load)
	agingOnce    = sync.OnceValues(loadAgingTable)
	nobilityOnce = sync.OnceValues(loadNobility)
)

// Config is the caller's inputs to one generation.
type Config struct {
	Seed uint64
	Name string
	// Service forces the enlistment attempt only: the throw is still
	// made, and a failed throw still goes to the draft (p. 5).
	Service string
	// Auto records which mode ran; the record's choices carry who decided.
	Auto bool
	// Skills, Muster, and CareerTerms select the auto policy's three
	// configurable rows (docs/POLICY.md). Zero values are the defaults, so
	// a Config that names none of them asks for the policy this tool has
	// always applied. They are recorded in the character's inputs, which is
	// where everything the caller supplied belongs.
	Skills      string
	Muster      string
	CareerTerms int
}

// Generate runs the whole prior-service procedure (Book 1 pp. 4-25) and
// returns the completed record. Death and a declined draft are completed
// generations, not errors.
func Generate(cfg Config, decider Decider) (*Character, error) {
	reg, err := registryOnce()
	if err != nil {
		return nil, fmt.Errorf("loading rule data: %w", err)
	}

	if cfg.Service != "" {
		if _, err := reg.Service(cfg.Service); err != nil {
			return nil, fmt.Errorf("--service: %w", err)
		}
	}

	// The policy selections are checked here for the same reason Service
	// is: they are recorded verbatim in the character's inputs, and a
	// value the policy does not recognise would be stamped there as the
	// policy that generated him while the default was quietly applied.
	if err := validatePolicyInputs(cfg); err != nil {
		return nil, err
	}

	aging, err := agingOnce()
	if err != nil {
		return nil, err
	}

	titles, err := nobilityOnce()
	if err != nil {
		return nil, err
	}

	g := &generator{
		reg:        reg,
		stream:     dice.New(cfg.Seed),
		decider:    decider,
		agingTable: aging,
		nobility:   titles,
		nextAging:  34, // aging begins at age 34, the end of term 4 (p. 7)
		char: &Character{
			SchemaVersion: SchemaVersion,
			Ruleset:       Ruleset,
			EngineVersion: EngineVersion,
			PolicyVersion: PolicyVersion,
			RNG:           RNG{Algorithm: dice.Algorithm, Seed: cfg.Seed},
			Inputs: Inputs{
				Auto: cfg.Auto, Name: cfg.Name, Service: cfg.Service,
				Skills: cfg.Skills, Muster: cfg.Muster, CareerTerms: cfg.CareerTerms,
			},
			Errata: slices.Clone(appliedErrata),
			Name:   cfg.Name,
			Age:    18, // all characters begin at age 18 (p. 4)
			Skills: []Skill{},
			Events: []Event{},
			// Empty, not nil: the schema declares benefits.weapons an
			// array and requires it, and a nil slice marshals as null.
			Benefits: Benefits{Weapons: []string{}},
		},
	}

	if err := g.run(); err != nil {
		return nil, err
	}

	return g.char, nil
}

type generator struct {
	reg        *service.Registry
	stream     *dice.Stream
	decider    Decider
	agingTable *agingTable
	nobility   *nobility
	nextAging  int
	char       *Character
	seq        int
}

func (g *generator) run() error {
	g.characteristics()

	svc, civilian, err := g.enlistment()
	if err != nil {
		return err
	}

	// svc is nil exactly when civilian is set. Both are tested because the
	// pairing is a convention between two functions rather than something
	// the type system carries, and the alternative encodings are barred:
	// a nil service with no bool trips nilnil, and a sentinel error would
	// put a declined draft in the error channel, where it does not belong.
	if civilian || svc == nil {
		// Declined the draft: an 18-year-old civilian, a valid record —
		// who may still hold a hereditary title (p. 5).
		return g.title("enlistment")
	}

	g.char.Service = svc.Name

	// Service-wide entries accrue on entering (p. 23; E004).
	if err := g.grantAutoSkills(svc, "enlistment", 0); err != nil {
		return err
	}

	for term := 1; ; term++ {
		left, err := g.term(svc, term)
		if err != nil {
			return err
		}

		if g.char.Death != nil {
			return nil
		}

		if left {
			break
		}
	}

	if err := g.musterOut(svc); err != nil {
		return err
	}

	// Muster-out characteristic alterations (p. 23) change the UPP too.
	g.char.UPP = g.char.Characteristics.UPP()

	g.retirement(svc, "muster-out")

	return g.title("muster-out")
}

// characteristics rolls 2D for each of the six, in order (p. 4), and
// derives the UPP (p. 8).
func (g *generator) characteristics() {
	step := "characteristics"
	g.step(step, "roll 2D for each characteristic, in order (p. 4)")

	for _, name := range service.CharacteristicNames() {
		total, _ := g.plainThrow(step, name)
		g.char.Characteristics.set(name, total)
	}

	g.char.UPP = g.char.Characteristics.UPP()
	g.outcome(step, fmt.Sprintf("UPP %s (p. 8)", g.char.UPP), 0)
}

// enlistment is the one enlistment attempt and, on rejection, the draft
// (p. 5). The bool reports a civilian who declined the draft — a
// completed generation with no service.
func (g *generator) enlistment() (*service.Service, bool, error) {
	step := "enlistment"
	g.step(step, "one enlistment attempt; rejection offers the draft (p. 5)")

	name := g.char.Inputs.Service
	if name == "" {
		picked, err := g.choose(Choice{Step: step, Label: ChoiceService, Options: g.reg.Names()})
		if err != nil {
			return nil, false, err
		}

		name = picked
	} else {
		g.outcome(step, "enlistment attempt forced to "+name+" by input, not choice", 0)
	}

	svc, err := g.reg.Service(name)
	if err != nil {
		return nil, false, fmt.Errorf("enlistment: %w", err)
	}

	_, ok, seq, err := g.targetThrow(step, "enlistment "+svc.Name, svc.Enlistment)
	if err != nil {
		return nil, false, err
	}

	if ok {
		g.outcome(step, "enlisted in the "+svc.Name, seq)

		return svc, false, nil
	}

	g.outcome(step, "rejected by the "+svc.Name, seq)

	submit, err := g.choose(Choice{Step: step, Label: ChoiceSubmitToDraft, Options: []string{Yes, No}})
	if err != nil {
		return nil, false, err
	}

	if submit == No {
		g.outcome(step, "declined the draft; enters play an 18-year-old civilian (E001)", 0)

		return nil, true, nil
	}

	roll, seq := g.plainRoll(step, "draft")

	drafted, err := g.reg.ByDraftNumber(roll)
	if err != nil {
		return nil, false, fmt.Errorf("draft rolled %d: %w", roll, err)
	}

	g.char.Drafted = true
	g.outcome(step, fmt.Sprintf("drafted into the %s (draft number %d)", drafted.Name, roll), seq)

	return drafted, false, nil
}

// term is one 4-year term (p. 5), in the exposition's order (E002):
// survival, commission, promotion, skills, reenlistment. It reports
// whether the character left the service (death included).
func (g *generator) term(svc *service.Service, term int) (bool, error) {
	step := fmt.Sprintf("term-%d", term)
	g.step(step, fmt.Sprintf("term %d begins, age %d", term, g.char.Age))

	_, ok, seq, err := g.targetThrow(step, "survival", svc.Survival)
	if err != nil {
		return false, err
	}

	if !ok {
		g.char.Death = &Death{Term: term, Cause: "failed the survival throw"}
		g.char.Terms = term - 1
		g.stampErratum("E003") // age of the dead: start of the fatal term
		g.outcome(step, fmt.Sprintf("died in service, term %d (survival failure is death, p. 5)", term), seq)

		return true, nil
	}

	bonus, err := g.commissionAndPromotion(svc, step, term)
	if err != nil {
		return false, err
	}

	if err := g.skills(&svc.Skills, step, term, bonus); err != nil {
		return false, err
	}

	stays, err := g.reenlistment(svc, step, term)
	if err != nil {
		return false, err
	}

	g.char.Age += 4 // each term is 4 years (p. 5)
	g.char.Terms = term

	// Aging closes the term (E005), after the reenlistment throw; it can
	// end the generation in a medical crisis (pp. 7-8).
	if err := g.aging(step, term); err != nil {
		return false, err
	}

	g.char.UPP = g.char.Characteristics.UPP()

	if g.char.Death != nil {
		return true, nil
	}

	if !stays {
		g.outcome(step, fmt.Sprintf("leaves the service after %d term(s), age %d", term, g.char.Age), 0)
	}

	return !stays, nil
}

// commissionAndPromotion is the term's one commission attempt (until
// achieved; not draftees' first term; p. 5) and, once commissioned —
// including in the very term of the commission — its one promotion
// attempt (p. 6). It reports the extra skill eligibility earned: 1 on
// commission, 1 on promotion (p. 6).
func (g *generator) commissionAndPromotion(svc *service.Service, step string, term int) (int, error) {
	if svc.Commission == nil {
		return 0, nil // non-existent in the Scout and Other services (p. 6, p. 10)
	}

	bonus := 0

	if g.char.Rank == 0 {
		earned, err := g.commission(svc, step, term)
		if err != nil {
			return 0, err
		}

		bonus += earned
	}

	if g.char.Rank >= 1 && g.char.Rank < len(svc.Ranks) {
		earned, err := g.promotion(svc, step)
		if err != nil {
			return 0, err
		}

		bonus += earned
	}

	return bonus, nil
}

func (g *generator) commission(svc *service.Service, step string, term int) (int, error) {
	if g.char.Drafted && term == 1 {
		g.outcome(step, "draftees are not eligible for commission during their first term (p. 5)", 0)

		return 0, nil
	}

	attempt, err := g.choose(Choice{Step: step, Label: ChoiceCommission, Options: []string{Yes, No}})
	if err != nil {
		return 0, err
	}

	if attempt == No {
		return 0, nil
	}

	_, ok, seq, err := g.targetThrow(step, "commission", *svc.Commission)
	if err != nil {
		return 0, err
	}

	if !ok {
		g.outcome(step, "commission denied this term (one attempt per term until successful, p. 5)", seq)

		return 0, nil
	}

	g.char.Rank = 1
	g.char.RankTitle = svc.Ranks[0]
	g.outcome(step, fmt.Sprintf("commissioned as %s (rank 1); +1 skill eligibility (pp. 5-6)", g.char.RankTitle), seq)

	if err := g.grantAutoSkills(svc, step, 1); err != nil {
		return 0, err
	}

	return 1, nil
}

func (g *generator) promotion(svc *service.Service, step string) (int, error) {
	attempt, err := g.choose(Choice{Step: step, Label: ChoicePromotion, Options: []string{Yes, No}})
	if err != nil {
		return 0, err
	}

	if attempt == No {
		return 0, nil
	}

	_, ok, seq, err := g.targetThrow(step, "promotion", *svc.Promotion)
	if err != nil {
		return 0, err
	}

	if !ok {
		g.outcome(step, "promotion denied this term (one attempt per term, p. 6)", seq)

		return 0, nil
	}

	g.char.Rank++
	g.char.RankTitle = svc.Ranks[g.char.Rank-1]
	text := fmt.Sprintf("promoted to %s (rank %d); +1 skill eligibility (p. 6)", g.char.RankTitle, g.char.Rank)
	g.outcome(step, text, seq)

	if err := g.grantAutoSkills(svc, step, g.char.Rank); err != nil {
		return 0, err
	}

	return 1, nil
}

// grantAutoSkills accrues the Rank and Service Skills box's entries for
// the given rank (0 = on entering the service), automatically and outside
// eligibility (p. 23; timing reading E004).
func (g *generator) grantAutoSkills(svc *service.Service, step string, rank int) error {
	for _, auto := range svc.AutoSkills {
		if auto.Rank != rank {
			continue
		}

		g.stampErratum("E004")

		if auto.Characteristic != "" {
			before, after, ok := g.char.Characteristics.Apply(auto.Characteristic, auto.Delta)
			if !ok {
				return fmt.Errorf("%w: unknown characteristic %q", ErrBadDecision, auto.Characteristic)
			}

			text := fmt.Sprintf("%+d %s (%d → %d), rank and service skills (p. 23; E004)",
				auto.Delta, auto.Characteristic, before, after)
			g.outcome(step, text, 0)

			continue
		}

		level := g.char.AddSkill(auto.Skill, auto.Category)
		g.outcome(step, fmt.Sprintf("%s-%d, rank and service skills (p. 23; E004)", auto.Skill, level), 0)
	}

	return nil
}

// skills spends the term's eligibility: 2 for the initial term, 1 per
// subsequent term, plus 1 for a commission and 1 for a promotion received
// this term (p. 6), each one die on a declared table (p. 11).
func (g *generator) skills(tables *service.SkillTables, step string, term, bonus int) error {
	eligibility := 1 + bonus
	if term == 1 {
		eligibility = 2 + bonus
	}

	for range eligibility {
		// TableNames hands back a fresh slice, so this is already a copy
		// no Decider can write through to the package's own.
		options := service.TableNames()
		if g.char.Characteristics.Education < 8 {
			options = options[:3] // the fourth table needs Education 8+ (p. 11)
		}

		table, err := g.choose(Choice{Step: step, Label: ChoiceSkillTable, Options: options})
		if err != nil {
			return err
		}

		rows, okTable := tables.Table(table)
		if !okTable {
			return fmt.Errorf("%w: skill table %q not in service data", ErrBadDecision, table)
		}

		roll, seq := g.plainRoll(step, "skill table "+table)

		if err := g.applySkillResult(step, rows[roll-1], seq); err != nil {
			return err
		}
	}

	return nil
}

func (g *generator) applySkillResult(step string, row service.SkillResult, ref int) error {
	switch {
	case row.Characteristic != "":
		before, after, ok := g.char.Characteristics.Apply(row.Characteristic, row.Delta)
		if !ok {
			return fmt.Errorf("%w: unknown characteristic %q", ErrBadDecision, row.Characteristic)
		}

		text := fmt.Sprintf("%+d %s (%d → %d), applied immediately (p. 12)", row.Delta, row.Characteristic, before, after)
		g.outcome(step, text, ref)
	case row.Weapon != "":
		options, err := g.reg.Weapons(row.Weapon)
		if err != nil {
			return fmt.Errorf("weapon expertise: %w", err)
		}

		weapon, err := g.choose(Choice{Step: step, Label: ChoiceWeapon, Category: row.Weapon, Options: options})
		if err != nil {
			return err
		}

		level := g.char.AddSkill(weapon, row.Weapon)
		text := fmt.Sprintf("%s expertise: %s-%d (weapon chosen immediately, pp. 11-13)", row.Weapon, weapon, level)
		g.outcome(step, text, ref)
	default:
		level := g.char.AddSkill(row.Skill, "")
		g.outcome(step, fmt.Sprintf("%s-%d", row.Skill, level), ref)
	}

	return nil
}

// reenlistment is made every term whether or not the character wants to
// stay (p. 6): failure forces him out, a 12 exactly forces him to stay
// (pp. 6-7). Voluntary service caps at 7 terms (p. 7). It reports whether
// the character stays for another term.
func (g *generator) reenlistment(svc *service.Service, step string, term int) (bool, error) {
	intent := No

	if term < 7 {
		picked, err := g.choose(Choice{Step: step, Label: ChoiceReenlist, Options: []string{Yes, No}})
		if err != nil {
			return false, err
		}

		intent = picked
	} else {
		g.outcome(step, "voluntary service caps at 7 terms; must attempt to leave (p. 7)", 0)
	}

	total, ok, seq, err := g.targetThrow(step, "reenlistment", svc.Reenlist)
	if err != nil {
		return false, err
	}

	switch {
	case !ok:
		g.outcome(step, "reenlistment denied; must leave the service (p. 6)", seq)

		return false, nil
	case total == 12:
		if term >= 7 {
			// Past the voluntary cap the 12 is the only thing still keeping
			// him in, and p. 7 grants "an additional term" in the singular
			// while p. 6 requires the throw again in that term: reading E009.
			g.stampErratum("E009")
		}

		g.outcome(step, "threw 12 exactly; must serve another term regardless of desires (pp. 6-7)", seq)

		return true, nil
	case intent == No:
		g.outcome(step, "chooses to leave the service", seq)

		return false, nil
	default:
		g.outcome(step, "reenlists for another term", seq)

		return true, nil
	}
}

// --- event helpers ---

func (g *generator) next() int {
	g.seq++

	return g.seq
}

func (g *generator) step(step, text string) {
	g.char.Events = append(g.char.Events, Event{Seq: g.next(), Kind: "step", Step: step, Text: text})
}

func (g *generator) outcome(step, text string, ref int) {
	g.char.Events = append(g.char.Events, Event{Seq: g.next(), Kind: "outcome", Step: step, Text: text, Ref: ref})
}

// plainThrow is a 2D throw with no target (the characteristic rolls). It
// reports the total and the event's sequence number.
func (g *generator) plainThrow(step, label string) (int, int) {
	d1, d2 := g.stream.Two()
	seq := g.next()
	g.char.Events = append(g.char.Events, Event{
		Seq: seq, Kind: "throw", Step: step, Label: label,
		Dice: []int{d1, d2}, Total: d1 + d2,
	})

	return d1 + d2, seq
}

// plainRoll is a single-die roll (draft, skills tables; FR9). It reports
// the value and the event's sequence number.
func (g *generator) plainRoll(step, label string) (int, int) {
	v := g.stream.One()
	seq := g.next()
	g.char.Events = append(g.char.Events, Event{
		Seq: seq, Kind: "throw", Step: step, Label: label,
		Dice: []int{v}, Total: v,
	})

	return v, seq
}

// targetThrow is a 2D throw against a Prior Service Table cell, with its
// cumulative characteristic DMs (pp. 5, 10). It reports the DM-modified
// total, whether the target was met, and the event's sequence number.
func (g *generator) targetThrow(step, label string, spec service.ThrowSpec) (int, bool, int, error) {
	target, err := dice.ParseTarget(spec.Target)
	if err != nil {
		return 0, false, 0, fmt.Errorf("%s %s: %w", step, label, err)
	}

	d1, d2 := g.stream.Two()
	total := d1 + d2

	var dms []EventDM

	for _, dm := range spec.DMs {
		if g.char.Characteristics.Get(dm.Characteristic) >= dm.Min {
			dms = append(dms, EventDM{Source: fmt.Sprintf("%s %d+", dm.Characteristic, dm.Min), Value: dm.DM})
			total += dm.DM
		}
	}

	success := target.Met(total)
	seq := g.next()
	g.char.Events = append(g.char.Events, Event{
		Seq: seq, Kind: "throw", Step: step, Label: label,
		Dice: []int{d1, d2}, DMs: dms, Target: spec.Target, Total: total, Success: &success,
	})

	return total, success, seq, nil
}

func (g *generator) choose(ch Choice) (string, error) {
	// Guarded here rather than in any one Decider: every choice point in
	// the procedure funnels through this call, and a Decider handed no
	// options has nothing to pick — the prompter would loop forever
	// refusing every answer.
	if len(ch.Options) == 0 {
		return "", fmt.Errorf("%w: choice %s at %s offers no options", ErrBadDecision, ch.Label, ch.Step)
	}

	decision, err := g.decider.Decide(ch)
	if err != nil {
		return "", fmt.Errorf("choice %s at %s: %w", ch.Label, ch.Step, err)
	}

	if !slices.Contains(ch.Options, decision.Pick) {
		return "", fmt.Errorf("%w: choice %s at %s: pick %q not among %v",
			ErrBadDecision, ch.Label, ch.Step, decision.Pick, ch.Options)
	}

	// Clone: the event log is a record of what was offered, not a window
	// onto whatever that slice holds later — the Decider was handed the
	// same slice and nothing here stops it writing to the elements. The
	// callers' own clones guard the other direction, package data against
	// a Decider that appends.
	g.char.Events = append(g.char.Events, Event{
		Seq: g.next(), Kind: "choice", Step: ch.Step, Label: ch.Label,
		By: decision.By, Options: slices.Clone(ch.Options), Picked: decision.Pick,
	})

	return decision.Pick, nil
}
