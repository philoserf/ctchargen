package render

import (
	"encoding/json"
	"fmt"

	"github.com/philoserf/ctchargen/traveller"
)

// The codecs for the sums. Each is a fold, so a case added to a sum stops
// this package compiling until the projection learns to write it.

type enlistmentCodec struct{ out enlistmentRecord }

func (e *enlistmentCodec) Enlisted(service traveller.ServiceName) error {
	e.out = enlistmentRecord{How: "enlisted", Service: service.String()}

	return nil
}

func (e *enlistmentCodec) Drafted(service traveller.ServiceName) error {
	e.out = enlistmentRecord{How: "drafted", Service: service.String()}

	return nil
}

func (e *enlistmentCodec) DeclinedTheDraft() error {
	e.out = enlistmentRecord{How: "declined the draft"}

	return nil
}

func foldEnlistment(from traveller.Enlistment) enlistmentRecord {
	if from == nil {
		return enlistmentRecord{}
	}

	var codec enlistmentCodec

	// A fold over a codec cannot fail: every case only assigns.
	_ = from.Fold(&codec)

	return codec.out
}

type departureCodec struct{ out departureRecord }

func (d *departureCodec) Discharged() error {
	d.out = departureRecord{How: "discharged"}

	return nil
}

func (d *departureCodec) ForcedOut() error {
	d.out = departureRecord{How: "forced out"}

	return nil
}

func (d *departureCodec) Retired() error {
	d.out = departureRecord{How: "retired"}

	return nil
}

func (d *departureCodec) KilledBySurvivalThrow() error {
	d.out = departureRecord{How: "killed by the survival throw"}

	return nil
}

func (d *departureCodec) KilledByMedicalCrisis(characteristic traveller.Characteristic) error {
	d.out = departureRecord{
		How: "killed by a medical crisis", Characteristic: characteristic.String(),
	}

	return nil
}

func foldDeparture(from traveller.Departure) departureRecord {
	var codec departureCodec

	_ = from.Fold(&codec)

	return codec.out
}

type stepJSON struct {
	Seq   int    `json:"seq"`
	Kind  string `json:"kind"`
	Step  string `json:"step"`
	Pages string `json:"pages,omitempty"`
}

type throwJSON struct {
	Seq       int    `json:"seq"`
	Kind      string `json:"kind"`
	Step      string `json:"step"`
	Dice      []int  `json:"dice"`
	DM        int    `json:"dm,omitempty"`
	Target    string `json:"target,omitempty"`
	Total     int    `json:"total"`
	Succeeded bool   `json:"succeeded"`
}

type choiceJSON struct {
	Seq          int      `json:"seq"`
	Kind         string   `json:"kind"`
	Point        string   `json:"point"`
	By           string   `json:"by"`
	Alternatives []string `json:"alternatives"`
	Chosen       string   `json:"chosen"`
}

type outcomeJSON struct {
	Seq         int      `json:"seq"`
	Kind        string   `json:"kind"`
	Because     int      `json:"because,omitempty"`
	Description string   `json:"description"`
	Errata      []string `json:"errata,omitempty"`
}

type eventCodec struct{ out json.RawMessage }

func (e *eventCodec) Step(from traveller.StepEvent) error {
	return e.write(stepJSON{Seq: from.Seq, Kind: "step", Step: from.Step, Pages: from.Pages})
}

func (e *eventCodec) Throw(from traveller.ThrowEvent) error {
	out := throwJSON{
		Seq: from.Seq, Kind: "throw", Step: from.Step, Dice: from.Dice,
		DM: from.DM, Total: from.Total(), Succeeded: from.Succeeded,
	}
	if from.Target.Number() != 0 {
		out.Target = from.Target.String()
	}

	return e.write(out)
}

func (e *eventCodec) Choice(from traveller.ChoiceEvent) error {
	return e.write(choiceJSON{
		Seq: from.Seq, Kind: "choice", Point: from.Point.String(), By: from.By.String(),
		Alternatives: from.Alternatives, Chosen: from.Chosen,
	})
}

func (e *eventCodec) Outcome(from traveller.OutcomeEvent) error {
	out := outcomeJSON{
		Seq: from.Seq, Kind: "outcome", Because: from.Because, Description: from.Description,
	}
	for _, erratum := range from.Errata {
		out.Errata = append(out.Errata, erratum.String())
	}

	return e.write(out)
}

// projectEvents writes every event, and refuses to write any of them if one
// will not marshal. The record claims to be the complete account of a
// generation, so an entry silently missing from it is worse than a failure
// to write the record at all.
func projectEvents(events []traveller.Event) ([]json.RawMessage, error) {
	out := make([]json.RawMessage, 0, len(events))

	for _, event := range events {
		var codec eventCodec

		err := event.Fold(&codec)
		if err != nil {
			return nil, fmt.Errorf("event %d: %w", event.Sequence(), err)
		}

		out = append(out, codec.out)
	}

	return out, nil
}

func (e *eventCodec) write(v any) error {
	text, err := json.Marshal(v)

	e.out = text

	//nolint:wrapcheck // the caller adds the context; this is one marshal.
	return err
}
