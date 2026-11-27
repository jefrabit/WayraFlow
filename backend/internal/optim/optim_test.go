package optim

import (
	"reflect"
	"testing"
	"time"

	"wayraflow/backend/internal/catalog"
	"wayraflow/backend/internal/domain"
	"wayraflow/backend/internal/plan"
)

var fixedNow = time.Date(2026, 11, 27, 16, 0, 0, 0, time.UTC)

func baseScenario(t *testing.T) domain.Scenario {
	t.Helper()
	sc, ok := catalog.Scenario("base")
	if !ok {
		t.Fatal("base scenario missing from catalog")
	}
	return sc
}

func TestFixedPlanUsesNinetySecondsAndEqualGreens(t *testing.T) {
	p, err := Build(catalog.Network(), baseScenario(t), domain.StrategyFixed, 42, fixedNow)
	if err != nil {
		t.Fatalf("Build() = %v", err)
	}
	for _, n := range p.Nodes {
		if n.CycleS != 90 {
			t.Errorf("node %s cycle = %v, want 90", n.ID, n.CycleS)
		}
		for _, ph := range n.Phases {
			if ph.GreenS != 40 {
				t.Errorf("node %s green = %v, want 40", n.ID, ph.GreenS)
			}
		}
	}
	if err := plan.Validate(p); err != nil {
		t.Fatalf("plan.Validate() = %v", err)
	}
}

func TestWebsterRespectsCycleBoundsAndGreens(t *testing.T) {
	for _, id := range []string{"base", "hora-punta-regional", "dia-de-feria"} {
		sc, _ := catalog.Scenario(id)
		p, err := Build(catalog.Network(), sc, domain.StrategyWebster, 42, fixedNow)
		if err != nil {
			t.Fatalf("%s: Build() = %v", id, err)
		}
		for _, n := range p.Nodes {
			if n.CycleS < MinCycle || n.CycleS > MaxCycle {
				t.Errorf("%s node %s cycle = %v, out of bounds", id, n.ID, n.CycleS)
			}
			for _, ph := range n.Phases {
				if ph.GreenS < MinGreen {
					t.Errorf("%s node %s green = %v, below minimum", id, n.ID, ph.GreenS)
				}
			}
		}
	}
}

func TestSharedNodeKeepsOneOffsetInBothCorridors(t *testing.T) {
	p, err := Build(catalog.Network(), baseScenario(t), domain.StrategyGA, 7, fixedNow)
	if err != nil {
		t.Fatalf("Build() = %v", err)
	}
	var c1, c2 *domain.CorridorPlan
	for i := range p.Corridors {
		switch p.Corridors[i].ID {
		case "C1":
			c1 = &p.Corridors[i]
		case "C2":
			c2 = &p.Corridors[i]
		}
	}
	if c1 == nil || c2 == nil {
		t.Fatal("C1 or C2 missing from the plan")
	}
	if c1.CycleS != c2.CycleS {
		t.Fatalf("C1 cycle %v != C2 cycle %v, they share N3", c1.CycleS, c2.CycleS)
	}
	if c1.OffsetsS["N3"] != c2.OffsetsS["N3"] {
		t.Fatalf("N3 offset differs between corridors: %v vs %v", c1.OffsetsS["N3"], c2.OffsetsS["N3"])
	}
}

func TestGASatisfiesTheContract(t *testing.T) {
	for _, id := range []string{"base", "hora-punta-regional", "dia-de-feria"} {
		sc, _ := catalog.Scenario(id)
		p, err := Build(catalog.Network(), sc, domain.StrategyGA, 42, fixedNow)
		if err != nil {
			t.Fatalf("%s: Build() = %v", id, err)
		}
		if err := plan.Validate(p); err != nil {
			t.Fatalf("%s: invalid plan: %v", id, err)
		}
		if len(p.Nodes) != 8 {
			t.Fatalf("%s: nodes = %d, want 8", id, len(p.Nodes))
		}
		if len(p.Corridors) != 3 {
			t.Fatalf("%s: corridors = %d, want 3", id, len(p.Corridors))
		}
	}
}

func TestGAIsDeterministicForTheSameSeed(t *testing.T) {
	sc := baseScenario(t)
	a, err := Build(catalog.Network(), sc, domain.StrategyGA, 1234, fixedNow)
	if err != nil {
		t.Fatalf("first Build() = %v", err)
	}
	b, err := Build(catalog.Network(), sc, domain.StrategyGA, 1234, fixedNow)
	if err != nil {
		t.Fatalf("second Build() = %v", err)
	}
	if !reflect.DeepEqual(a, b) {
		t.Fatal("same seed produced different plans")
	}
}

func TestSplitGreensKeepsTheSumExact(t *testing.T) {
	for _, cycle := range []float64{40, 47, 90, 120} {
		for _, split := range []float64{0, 0.1, 0.5, 0.9, 1} {
			g := splitGreens(cycle, split)
			sum := g[0] + g[1] + 2*(AmberS+AllRedS)
			if diff := sum - cycle; diff > 1e-9 || diff < -1e-9 {
				t.Fatalf("cycle %v split %v: sum = %v, want %v", cycle, split, sum, cycle)
			}
			if g[0] < MinGreen || g[1] < MinGreen {
				t.Fatalf("cycle %v split %v: greens %v below minimum", cycle, split, g)
			}
		}
	}
}

func TestBuildRejectsUnknownStrategy(t *testing.T) {
	if _, err := Build(catalog.Network(), baseScenario(t), domain.Strategy("magic"), 0, fixedNow); err == nil {
		t.Fatal("Build() = nil, want an error")
	}
}
