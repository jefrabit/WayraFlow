package plan

import (
	"strings"
	"testing"
	"time"

	"wayraflow/backend/internal/catalog"
	"wayraflow/backend/internal/domain"
)

func validPlan() domain.Plan {
	return domain.Plan{
		Schema:      domain.PlanSchema,
		City:        "Juliaca",
		Scenario:    "base",
		GeneratedAt: time.Date(2026, 11, 27, 16, 0, 0, 0, time.UTC),
		Source:      domain.SourceODbL,
		Corridors: []domain.CorridorPlan{
			{ID: "C1", CycleS: 90, OffsetsS: map[string]float64{"N2": 0, "N1": 84.7, "N3": 80.6}},
		},
		Nodes: []domain.NodePlan{
			{ID: "N2", OSMNode: 1282294742, Corridor: "C1", CycleS: 90, Phases: []domain.Phase{
				{ID: 1, GreenS: 40, AmberS: 3, AllRedS: 2},
				{ID: 2, GreenS: 40, AmberS: 3, AllRedS: 2},
			}},
			{ID: "N1", OSMNode: 257687118, Corridor: "C1", CycleS: 90, Phases: []domain.Phase{
				{ID: 1, GreenS: 40, AmberS: 3, AllRedS: 2},
				{ID: 2, GreenS: 40, AmberS: 3, AllRedS: 2},
			}},
			{ID: "N3", OSMNode: 1281999136, Corridor: "C1", CycleS: 90, Phases: []domain.Phase{
				{ID: 1, GreenS: 40, AmberS: 3, AllRedS: 2},
				{ID: 2, GreenS: 40, AmberS: 3, AllRedS: 2},
			}},
		},
	}
}

func TestValidateAcceptsAWellFormedPlan(t *testing.T) {
	if err := Validate(validPlan()); err != nil {
		t.Fatalf("Validate() = %v, want nil", err)
	}
}

func TestValidateRejectsBadPhaseSum(t *testing.T) {
	p := validPlan()
	p.Nodes[0].Phases[0].GreenS = 39
	err := Validate(p)
	if err == nil {
		t.Fatal("Validate() = nil, want an error")
	}
	if !strings.Contains(err.Error(), "sum of green+amber+allRed") {
		t.Fatalf("error = %q, want a phase sum error", err)
	}
}

func TestValidateRejectsOffsetOutOfRange(t *testing.T) {
	p := validPlan()
	p.Corridors[0].OffsetsS["N1"] = 90
	if err := Validate(p); err == nil {
		t.Fatal("Validate() = nil, want an error for offset == cycle")
	}
	p.Corridors[0].OffsetsS["N1"] = -1
	if err := Validate(p); err == nil {
		t.Fatal("Validate() = nil, want an error for a negative offset")
	}
}

func TestValidateRejectsUnknownNodeInOffsets(t *testing.T) {
	p := validPlan()
	p.Corridors[0].OffsetsS["N99"] = 10
	if err := Validate(p); err == nil {
		t.Fatal("Validate() = nil, want an error for an unknown node")
	}
}

func TestValidateRejectsMismatchedCorridorCycle(t *testing.T) {
	p := validPlan()
	p.Corridors[0].CycleS = 100
	if err := Validate(p); err == nil {
		t.Fatal("Validate() = nil, want an error for a cycle mismatch")
	}
}

func TestValidateRejectsUnknownCorridor(t *testing.T) {
	p := validPlan()
	p.Nodes[0].Corridor = "C9"
	if err := Validate(p); err == nil {
		t.Fatal("Validate() = nil, want an error for an unknown corridor")
	}
}

func TestValidateRejectsMinimumGreen(t *testing.T) {
	p := validPlan()
	p.Nodes[0].Phases[0].GreenS = 5
	p.Nodes[0].Phases[1].GreenS = 75
	if err := Validate(p); err == nil {
		t.Fatal("Validate() = nil, want an error for green < 7 s")
	}
}

func TestValidateRejectsDuplicateNode(t *testing.T) {
	p := validPlan()
	p.Nodes = append(p.Nodes, p.Nodes[0])
	if err := Validate(p); err == nil {
		t.Fatal("Validate() = nil, want an error for a duplicate node")
	}
}

func TestCatalogNetworkIsValidForThePlanContract(t *testing.T) {
	net := catalog.Network()
	if len(net.Nodes) != 8 {
		t.Fatalf("nodes = %d, want 8", len(net.Nodes))
	}
	if len(net.Corridors) != 3 {
		t.Fatalf("corridors = %d, want 3", len(net.Corridors))
	}
	for _, c := range net.Corridors {
		for _, cn := range c.Nodes {
			if _, ok := net.Node(cn.NodeID); !ok {
				t.Fatalf("corridor %s references unknown node %s", c.ID, cn.NodeID)
			}
		}
	}
}
