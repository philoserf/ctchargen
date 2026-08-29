package chargen

import (
	"embed"
	"encoding/json"
	"fmt"
	"slices"
	"strconv"

	"github.com/philoserf/ctchargen/dice"
	"github.com/philoserf/ctchargen/service"
)

// The charts that are not per-service: the Aging Table (Book 1 p. 9) and
// the Nobility table (Book 3 p. 22, consulted because Book 1 p. 4 points
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
	if err := json.Unmarshal(raw, table); err != nil {
		return nil, fmt.Errorf("parsing aging table: %w", err)
	}

	if len(table.Rounds) == 0 {
		return nil, fmt.Errorf("%w: aging table has no rounds", ErrBadDecision)
	}

	for _, round := range table.Rounds {
		for _, throw := range round.Throws {
			if _, err := dice.ParseTarget(throw.Save); err != nil {
				return nil, fmt.Errorf("aging round from %d: %w", round.FromAge, err)
			}

			if throw.Loss < 1 || !validAgingCharacteristic(throw.Characteristic) {
				return nil, fmt.Errorf("%w: aging round from %d: %s loss %d",
					ErrBadDecision, round.FromAge, throw.Characteristic, throw.Loss)
			}
		}
	}

	return table, nil
}

func validAgingCharacteristic(name string) bool {
	return slices.Contains(service.CharacteristicNames, name)
}

// nobility maps Social Standing 11-15 to the hereditary title (Book 3
// p. 22; Book 1 p. 4 points there).
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
	if err := json.Unmarshal(raw, table); err != nil {
		return nil, fmt.Errorf("parsing nobility table: %w", err)
	}

	for social := 11; social <= 15; social++ {
		if table.titleFor(social) == "" {
			return nil, fmt.Errorf("%w: nobility table missing social standing %d", ErrBadDecision, social)
		}
	}

	return table, nil
}
