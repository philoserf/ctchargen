package chargen

import (
	"errors"
	"testing"
)

type agingBandCase struct {
	name    string
	rounds  []AgingRound
	wantErr bool
}

func agingBandCases() []agingBandCase {
	return []agingBandCase{
		{
			name: "bounded bands then an open-ended last",
			rounds: []AgingRound{
				{FromAge: 34, ThroughAge: 45},
				{FromAge: 46, ThroughAge: 65},
				{FromAge: 66},
			},
		},
		{
			name:   "a single open-ended band is the last band",
			rounds: []AgingRound{{FromAge: 34}},
		},
		{
			name: "open-ended band before the end swallows every later band",
			rounds: []AgingRound{
				{FromAge: 34},
				{FromAge: 46, ThroughAge: 65},
			},
			wantErr: true,
		},
		{
			name:    "bounded band ending before it begins",
			rounds:  []AgingRound{{FromAge: 46, ThroughAge: 34}},
			wantErr: true,
		},
		{
			name: "bands out of order, so roundFor would match the wrong one",
			rounds: []AgingRound{
				{FromAge: 46, ThroughAge: 65},
				{FromAge: 34, ThroughAge: 45},
			},
			wantErr: true,
		},
		{
			name: "gaps between bands are allowed; the printed table has them",
			rounds: []AgingRound{
				{FromAge: 34, ThroughAge: 46},
				{FromAge: 50, ThroughAge: 62},
				{FromAge: 66},
			},
		},
		{
			name:   "no rounds at all is the caller's check, not this one",
			rounds: nil,
		},
	}
}

// validateAgingBands guards a sentinel that cannot be exercised from
// outside the package: the embedded aging table is valid, so only an
// internal test can feed it a malformed one. Everything else in the
// package is tested externally.
func TestValidateAgingBands(t *testing.T) {
	for _, tt := range agingBandCases() {
		t.Run(tt.name, func(t *testing.T) {
			err := validateAgingBands(tt.rounds)

			if tt.wantErr {
				if err == nil {
					t.Fatal("want an error, got nil")
				}

				if !errors.Is(err, ErrInvalidChart) {
					t.Errorf("want ErrInvalidChart, got %v", err)
				}

				return
			}

			if err != nil {
				t.Errorf("want no error, got %v", err)
			}
		})
	}
}

// A misspelled through_age is the failure this guards against: the field
// zeroes, roundFor reads the zero as open-ended, and a bounded band
// silently matches every age above its start.
func TestDecodeStrictRejectsUnknownFields(t *testing.T) {
	table := &agingTable{}

	err := decodeStrict([]byte(`{"page": 9, "rounds": [], "roundz": []}`), table)
	if err == nil {
		t.Fatal("want an error for an unknown field, got nil")
	}
}
