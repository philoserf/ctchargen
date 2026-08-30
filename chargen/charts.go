package chargen

import (
	"bytes"
	"embed"
	"encoding/json"
	"fmt"
	"slices"
	"strconv"

	"github.com/philoserf/ctchargen/dice"
	"github.com/philoserf/ctchargen/service"
)

// The charts that are not per-service: the Aging Table (Book 1 p. 9) and
// the Nobility table (Book 3 p. 22, consulted because Book 1 p. 5 points
// there). Embedded data with load-time validation, like the service
// tables (docs/PRD.md, Architecture notes).
//
//go:embed data/*.json
var chartFS embed.FS

// AgingThrow is one saving throw of an aging round: on a failed save the
// named characteristic drops by Loss (p. 9).
type AgingThrow struct {
	Characteristic string `json:"characteristic"`
	Loss           int    `json:"loss"`
	Save           string `json:"save"`
}

// AgingRound is one age band of the Aging Table. ThroughAge 0 means the
// band is open-ended ("74+" continues every four years, p. 9).
type AgingRound struct {
	FromAge    int          `json:"from_age"`
	ThroughAge int          `json:"through_age"`
	Throws     []AgingThrow `json:"throws"`
}

type agingTable struct {
	Page   int          `json:"page"`
	Rounds []AgingRound `json:"rounds"`
}

// decodeStrict parses embedded chart JSON, rejecting unknown fields so a
// misspelled key is a load-time failure rather than a silently zeroed
// field — and a silently wrong rule. The service tables are read the same
// way (service.loadServices).
func decodeStrict(raw []byte, dst any) error {
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()

	if err := dec.Decode(dst); err != nil {
		return fmt.Errorf("decoding chart: %w", err)
	}

	return nil
}

// roundFor finds the band covering an age, or nil below the first band.
func (t *agingTable) roundFor(age int) *AgingRound {
	for i := range t.Rounds {
		r := &t.Rounds[i]
		if age >= r.FromAge && (r.ThroughAge == 0 || age <= r.ThroughAge) {
			return r
		}
	}

	return nil
}

func loadAgingTable() (*agingTable, error) {
	raw, err := chartFS.ReadFile("data/aging.json")
	if err != nil {
		return nil, fmt.Errorf("reading aging table: %w", err)
	}

	table := &agingTable{}
	if err := decodeStrict(raw, table); err != nil {
		return nil, fmt.Errorf("parsing aging table: %w", err)
	}

	if len(table.Rounds) == 0 {
		return nil, fmt.Errorf("%w: aging table has no rounds", ErrInvalidChart)
	}

	if err := validateAgingBands(table.Rounds); err != nil {
		return nil, err
	}

	for _, round := range table.Rounds {
		for _, throw := range round.Throws {
			if _, err := dice.ParseTarget(throw.Save); err != nil {
				// Both sentinels, as service.validateThrowSpec does: a broken
				// chart must answer to ErrInvalidChart however it is broken.
				return nil, fmt.Errorf("%w: aging round from %d: %w", ErrInvalidChart, round.FromAge, err)
			}

			if throw.Loss < 1 || !validCharacteristic(throw.Characteristic) {
				return nil, fmt.Errorf("%w: aging round from %d: %s loss %d",
					ErrInvalidChart, round.FromAge, throw.Characteristic, throw.Loss)
			}
		}
	}

	return table, nil
}

// validateAgingBands pins the ThroughAge == 0 sentinel down. roundFor
// treats a zero as an open-ended band, so a dropped through_age would
// silently turn a bounded band into one matching every age above its
// start — and roundFor returns the first match, so band order decides
// which rule applies. Only the last round may be open-ended, a bounded
// one must not end before it begins, and the bands must ascend.
//
// Gaps between bands are deliberately not checked: the printed table has
// them (p. 9), and aging simply does not apply in a gap.
func validateAgingBands(rounds []AgingRound) error {
	for i, round := range rounds {
		openEnded := round.ThroughAge == 0

		if openEnded && i != len(rounds)-1 {
			return fmt.Errorf("%w: aging round from %d is open-ended but is not the last",
				ErrInvalidChart, round.FromAge)
		}

		if !openEnded && round.ThroughAge < round.FromAge {
			return fmt.Errorf("%w: aging round from %d ends at %d",
				ErrInvalidChart, round.FromAge, round.ThroughAge)
		}

		if i > 0 && round.FromAge <= rounds[i-1].FromAge {
			return fmt.Errorf("%w: aging round from %d does not follow the round from %d",
				ErrInvalidChart, round.FromAge, rounds[i-1].FromAge)
		}
	}

	return nil
}

// validCharacteristic reports whether a name is one of the six the record
// carries. The service package validates its own data at load; this is
// the same check on the chargen side, for the charts and for the
// characteristic mutators that would otherwise alter nothing silently.
func validCharacteristic(name string) bool {
	return slices.Contains(service.CharacteristicNames(), name)
}

// nobility maps Social Standing 11-15 to the hereditary title (Book 3
// p. 22; Book 1 p. 5 points there).
type nobility struct {
	Book3Page int               `json:"book3_page"`
	Titles    map[string]string `json:"titles"`
}

func (n *nobility) titleFor(social int) string {
	return n.Titles[strconv.Itoa(social)]
}

func loadNobility() (*nobility, error) {
	raw, err := chartFS.ReadFile("data/nobility.json")
	if err != nil {
		return nil, fmt.Errorf("reading nobility table: %w", err)
	}

	table := &nobility{}
	if err := decodeStrict(raw, table); err != nil {
		return nil, fmt.Errorf("parsing nobility table: %w", err)
	}

	for social := 11; social <= 15; social++ {
		if table.titleFor(social) == "" {
			return nil, fmt.Errorf("%w: nobility table missing social standing %d", ErrInvalidChart, social)
		}
	}

	return table, nil
}
