package autodarts

import (
	"encoding/json"
	"testing"
)

func TestDetailConflictsAreRetainedNotRepaired(t *testing.T) {
	for _, tc := range []struct {
		name, reason string
		change       func(*Payload)
	}{
		{"transition", "score_transition", func(p *Payload) { p.Detail.Legs[0].Visits[0].EndRemaining = 2 }},
		{"unknown player", "unknown_player", func(p *Payload) { p.Detail.Legs[0].Visits[0].PlayerID = "other" }},
		{"winner", "leg_winner_conflict", func(p *Payload) { p.Detail.Legs[0].WinnerID = &p.Players[1].ID }},
		{"complete claim", "turn_continuity", func(p *Payload) { p.Detail.Coverage = "complete" }},
		{"single out", "score_transition", func(p *Payload) {
			v := &p.Detail.Legs[0].Visits[0]
			v.StartRemaining = 20
			v.Throws[0].Segment.Bed = "single"
		}},
		{"after checkout", "throws_after_terminal_dart", func(p *Payload) {
			v := &p.Detail.Legs[0].Visits[0]
			dart := v.Throws[0]
			dart.Number = 2
			v.Throws = append(v.Throws, dart)
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var p Payload
			if err := json.Unmarshal(example(t, "producer"), &p); err != nil {
				t.Fatal(err)
			}
			tc.change(&p)
			body, err := json.Marshal(p)
			if err != nil {
				t.Fatal(err)
			}
			got, err := Parse(body)
			if err != nil {
				t.Fatal(err)
			}
			if got.ReviewReason != tc.reason {
				t.Fatalf("got %q want %q", got.ReviewReason, tc.reason)
			}
			if len(got.Payload) == 0 {
				t.Fatal("discarded conflicting original")
			}
		})
	}
}

func TestBustRollback(t *testing.T) {
	var p Payload
	if err := json.Unmarshal(example(t, "producer"), &p); err != nil {
		t.Fatal(err)
	}
	l := &p.Detail.Legs[0]
	l.Completed = false
	l.WinnerID = nil
	v := &l.Visits[0]
	v.StartRemaining = 32
	v.EndRemaining = 32
	v.Bust = true
	body, err := json.Marshal(p)
	if err != nil {
		t.Fatal(err)
	}
	got, err := Parse(body)
	if err != nil {
		t.Fatal(err)
	}
	if got.ReviewReason != "" {
		t.Fatal(got.ReviewReason)
	}
}

func TestDetailArrayLimits(t *testing.T) {
	for _, change := range []func(*Payload){
		func(p *Payload) { p.Players = append(p.Players, p.Players[0]) },
		func(p *Payload) {
			for len(p.Detail.Legs) < 6 {
				p.Detail.Legs = append(p.Detail.Legs, p.Detail.Legs[0])
			}
		},
		func(p *Payload) { p.Detail.Legs[0].Visits = make([]Visit, 201) },
		func(p *Payload) { p.Detail.Legs[0].Visits[0].Throws = make([]Throw, 4) },
	} {
		var p Payload
		if err := json.Unmarshal(example(t, "producer"), &p); err != nil {
			t.Fatal(err)
		}
		change(&p)
		body, err := json.Marshal(p)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Parse(body); err == nil {
			t.Fatal("accepted excessive array")
		}
	}
}
