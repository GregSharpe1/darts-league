package league

import (
	"context"
	"testing"

	"github.com/greg/darts-league/backend/internal/autodarts"
)

func TestPlayerStatisticsPlotsOnlySupportedManualCoordinates(t *testing.T) {
	units, origin, axes := "board-radius", "bull", "x-right-y-up"
	for _, scenario := range []struct {
		name  string
		entry string
		units *string
		x     float64
		want  bool
	}{
		{"observed manual geometry", "manual", &units, 0, true},
		{"unknown geometry", "manual", nil, 0, false},
		{"unverified automatic geometry", "automatic", &units, 0, false},
		{"projection overflow", "manual", &units, 1e308, false},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			store, service := statisticsSetup(t)
			payload := statisticsPayload(false)
			dart := &payload.Detail.Legs[0].Visits[0].Throws[0]
			dart.EntryType = scenario.entry
			dart.Position = &autodarts.Position{X: scenario.x, Y: 0.6, Units: scenario.units, Origin: &origin, AxisOrientation: &axes, Provenance: scenario.entry}
			statisticsApprove(t, store, payload)
			got, err := service.PublicPlayerStatistics(context.Background(), 1, 1)
			if err != nil {
				t.Fatal(err)
			}
			count := 0
			if scenario.want {
				count = 1
			}
			if got.Coverage.PlottablePositions != count || len(got.Throws) == 0 || got.Throws[0].Plottable != scenario.want {
				t.Fatalf("plottable coverage: %+v, throws: %+v", got.Coverage, got.Throws)
			}
		})
	}
}
