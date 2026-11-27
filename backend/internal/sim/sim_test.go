package sim_test

import (
	"math"
	"reflect"
	"testing"
	"time"

	"wayraflow/backend/internal/catalog"
	"wayraflow/backend/internal/domain"
	"wayraflow/backend/internal/optim"
	"wayraflow/backend/internal/plan"
	"wayraflow/backend/internal/sim"
)

var simNow = time.Date(2026, 11, 27, 16, 0, 0, 0, time.UTC)

func scenario(t *testing.T, id string) domain.Scenario {
	t.Helper()
	sc, ok := catalog.Scenario(id)
	if !ok {
		t.Fatalf("scenario %s missing", id)
	}
	return sc
}

func fixedPlan(t *testing.T, sc domain.Scenario) domain.Plan {
	t.Helper()
	p, err := optim.Build(catalog.Network(), sc, domain.StrategyFixed, 42, simNow)
	if err != nil {
		t.Fatalf("optim.Build() = %v", err)
	}
	return p
}

func TestZeroDemandProducesZeroDelay(t *testing.T) {
	sc := scenario(t, "base")
	sc.Demand.Corridors = map[string]float64{}
	p := fixedPlan(t, sc)
	res, err := sim.Run(catalog.Network(), sc, p, 600)
	if err != nil {
		t.Fatalf("sim.Run() = %v", err)
	}
	if res.City.AvgDelayS != 0 || res.City.CO2KgH != 0 {
		t.Fatalf("zero demand produced delay=%v co2=%v", res.City.AvgDelayS, res.City.CO2KgH)
	}
}

func TestRunIsDeterministic(t *testing.T) {
	sc := scenario(t, "base")
	p := fixedPlan(t, sc)
	a, err := sim.Run(catalog.Network(), sc, p, 1200)
	if err != nil {
		t.Fatalf("sim.Run() = %v", err)
	}
	b, err := sim.Run(catalog.Network(), sc, p, 1200)
	if err != nil {
		t.Fatalf("sim.Run() = %v", err)
	}
	if !reflect.DeepEqual(a, b) {
		t.Fatal("two runs with the same inputs differ")
	}
}

func TestProgressionBeatsZeroOffsets(t *testing.T) {
	sc := scenario(t, "hora-punta-regional")
	net := catalog.Network()
	wave := fixedPlan(t, sc)
	flat := fixedPlan(t, sc)
	for i := range flat.Corridors {
		for id := range flat.Corridors[i].OffsetsS {
			flat.Corridors[i].OffsetsS[id] = 0
		}
	}
	if err := plan.Validate(flat); err != nil {
		t.Fatalf("flat plan invalid: %v", err)
	}
	a, err := sim.Run(net, sc, wave, 1800)
	if err != nil {
		t.Fatalf("sim.Run(wave) = %v", err)
	}
	b, err := sim.Run(net, sc, flat, 1800)
	if err != nil {
		t.Fatalf("sim.Run(flat) = %v", err)
	}
	if a.City.AvgDelayS >= b.City.AvgDelayS {
		t.Fatalf("green wave delay %v >= zero-offset delay %v", a.City.AvgDelayS, b.City.AvgDelayS)
	}
}

func TestHigherDemandIncreasesDelay(t *testing.T) {
	net := catalog.Network()
	base := scenario(t, "base")
	peak := scenario(t, "hora-punta-regional")
	p := fixedPlan(t, base)
	a, err := sim.Run(net, base, p, 1800)
	if err != nil {
		t.Fatalf("sim.Run(base) = %v", err)
	}
	b, err := sim.Run(net, peak, p, 1800)
	if err != nil {
		t.Fatalf("sim.Run(peak) = %v", err)
	}
	if b.City.AvgDelayS <= a.City.AvgDelayS {
		t.Fatalf("peak delay %v <= base delay %v", b.City.AvgDelayS, a.City.AvgDelayS)
	}
}

func TestKPIsAreFiniteAndNonNegative(t *testing.T) {
	for _, id := range []string{"base", "hora-punta-regional", "dia-de-feria"} {
		sc := scenario(t, id)
		p, err := optim.Build(catalog.Network(), sc, domain.StrategyGA, 42, simNow)
		if err != nil {
			t.Fatalf("%s: Build() = %v", id, err)
		}
		res, err := sim.Run(catalog.Network(), sc, p, 3600)
		if err != nil {
			t.Fatalf("%s: sim.Run() = %v", id, err)
		}
		values := []float64{res.City.AvgDelayS, res.City.MaxQueueVeh, res.City.MaxQueueM, res.City.StopsPerVeh, res.City.CO2KgH}
		for _, v := range values {
			if math.IsNaN(v) || math.IsInf(v, 0) || v < 0 {
				t.Fatalf("%s: invalid KPI %v", id, v)
			}
		}
		if len(res.Nodes) != 8 {
			t.Fatalf("%s: nodes = %d, want 8", id, len(res.Nodes))
		}
	}
}
