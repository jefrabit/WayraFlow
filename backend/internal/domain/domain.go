// Package domain holds the types and rules shared by the simulator, the
// optimizer and the HTTP layer. It imports nothing from the project.
package domain

import (
	"fmt"
	"math"
	"time"
)

// PlanSchema identifies the exported traffic plan document.
const PlanSchema = "wayraflow.plan/v1"

// SourceODbL is the attribution required by the OpenStreetMap license.
const SourceODbL = "OpenStreetMap contributors, ODbL 1.0"

// City is the only city modeled by the MVP.
const City = "Juliaca"

// Strategy is the optimization approach used to build a plan.
type Strategy string

const (
	StrategyFixed   Strategy = "fixed"
	StrategyWebster Strategy = "webster"
	StrategyGA      Strategy = "ga"
)

// Valid reports whether s is one of the strategies accepted by the API.
func (s Strategy) Valid() bool {
	switch s {
	case StrategyFixed, StrategyWebster, StrategyGA:
		return true
	}
	return false
}

// Mix is the vehicle composition of the demand. The four fractions must add up
// to 1.
type Mix struct {
	L5 float64 `json:"L5"`
	M1 float64 `json:"M1"`
	M2 float64 `json:"M2"`
	N  float64 `json:"N"`
}

// Validate checks that the mix is a probability distribution.
func (m Mix) Validate() error {
	sum := m.L5 + m.M1 + m.M2 + m.N
	for name, v := range map[string]float64{"L5": m.L5, "M1": m.M1, "M2": m.M2, "N": m.N} {
		if v < 0 {
			return fmt.Errorf("mix %s must be >= 0, got %v", name, v)
		}
	}
	if math.Abs(sum-1) > 1e-6 {
		return fmt.Errorf("mix must add up to 1, got %v", sum)
	}
	return nil
}

// EquivalentFactor converts a vehicle of each class to passenger-car units.
// Values are assumptions documented in README section 4.3.
func (m Mix) EquivalentFactor() float64 {
	return m.L5*0.5 + m.M1*1.0 + m.M2*2.0 + m.N*2.5
}

// AverageLengthM is the mean space occupied by a queued vehicle, including the
// safety gap.
func (m Mix) AverageLengthM() float64 {
	return m.L5*4.5 + m.M1*7.0 + m.M2*13.0 + m.N*15.0
}

// IdleLitersPerHour is the fuel burned while stopped, weighted by the mix.
func (m Mix) IdleLitersPerHour() float64 {
	return m.L5*0.3 + m.M1*0.6 + m.M2*3.8 + m.N*3.8
}

// StartLiters is the extra fuel of one stop, weighted by the mix.
func (m Mix) StartLiters() float64 {
	return m.L5*0.005 + m.M1*0.010 + m.M2*0.040 + m.N*0.040
}

// FuelDensityKgPerL is the CO2 factor per liter, weighted by the mix. Gasoline
// is used for L5 and M1, diesel for heavy vehicles (README section 5).
func (m Mix) FuelDensityKgPerL() float64 {
	return (m.L5+m.M1)*2.35 + (m.M2+m.N)*2.69
}

// Demand is the hourly flow per corridor, in vehicles per hour.
type Demand struct {
	Corridors map[string]float64 `json:"corridors"`
	Mix       Mix                `json:"mix"`
}

// Scenario is a reproducible traffic condition.
type Scenario struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Demand      Demand `json:"demand"`
}

// Flow returns the demand of a corridor, or zero when it is not defined.
func (s Scenario) Flow(corridor string) float64 {
	return s.Demand.Corridors[corridor]
}

// Node is a signalized intersection of the network.
type Node struct {
	ID         string  `json:"id"`
	Name       string  `json:"name"`
	OSMNodeID  int64   `json:"osmNode"`
	Signalized bool    `json:"signalized"`
	Lon        float64 `json:"lon"`
	Lat        float64 `json:"lat"`
}

// CorridorNode is one node inside a corridor, ordered from west to east or
// north to south depending on the corridor.
type CorridorNode struct {
	NodeID    string  `json:"nodeId"`
	Position  int     `json:"position"`
	DistanceM float64 `json:"distanceM"` // from the previous node
}

// Corridor is a group of nodes coordinated with a common cycle.
type Corridor struct {
	ID    string         `json:"id"`
	Name  string         `json:"name"`
	Nodes []CorridorNode `json:"nodes"`
}

// Network is the road network used by the engine.
type Network struct {
	City      string     `json:"city"`
	Nodes     []Node     `json:"nodes"`
	Corridors []Corridor `json:"corridors"`
}

// Node returns the node with the given id.
func (n Network) Node(id string) (Node, bool) {
	for _, node := range n.Nodes {
		if node.ID == id {
			return node, true
		}
	}
	return Node{}, false
}

// Corridor returns the corridor with the given id.
func (n Network) Corridor(id string) (Corridor, bool) {
	for _, c := range n.Corridors {
		if c.ID == id {
			return c, true
		}
	}
	return Corridor{}, false
}

// Position returns the corridor and position of a node, if it belongs to one.
func (n Network) Position(nodeID string) (Corridor, CorridorNode, bool) {
	for _, c := range n.Corridors {
		for _, cn := range c.Nodes {
			if cn.NodeID == nodeID {
				return c, cn, true
			}
		}
	}
	return Corridor{}, CorridorNode{}, false
}

// Phase is one stage of a signal cycle.
type Phase struct {
	ID        int      `json:"id"`
	Movements []string `json:"movements"`
	GreenS    float64  `json:"green_s"`
	AmberS    float64  `json:"amber_s"`
	AllRedS   float64  `json:"allRed_s"`
}

// Duration is the total time the phase reserves in the cycle.
func (p Phase) Duration() float64 {
	return p.GreenS + p.AmberS + p.AllRedS
}

// NodePlan is the timing of one intersection.
type NodePlan struct {
	ID       string  `json:"id"`
	OSMNode  int64   `json:"osmNode"`
	Corridor string  `json:"corridor"`
	CycleS   float64 `json:"cycle_s"`
	Phases   []Phase `json:"phases"`
}

// CorridorPlan coordinates the offsets of a corridor.
type CorridorPlan struct {
	ID       string             `json:"id"`
	CycleS   float64            `json:"cycle_s"`
	OffsetsS map[string]float64 `json:"offsets_s"`
}

// KPI is the measured performance of one strategy.
type KPI struct {
	AvgDelayS   float64 `json:"avgDelay_s"`
	MaxQueueVeh float64 `json:"maxQueue_veh"`
	MaxQueueM   float64 `json:"maxQueue_m"`
	StopsPerVeh float64 `json:"stopsPerVeh"`
	CO2KgH      float64 `json:"co2_kg_h"`
	Vehicles    int     `json:"vehicles"`
	SpillbackS  int     `json:"spillback_s,omitempty"`
}

// KPIReport compares the three strategies. Improvement is the percentage
// reduction of average delay against the fixed baseline.
type KPIReport struct {
	Fixed          KPI                `json:"fixed"`
	Webster        KPI                `json:"webster"`
	GA             KPI                `json:"ga"`
	ImprovementPct map[string]float64 `json:"improvement_pct"`
}

// NodeKPI is the per-node measurement for the chosen strategy.
type NodeKPI struct {
	NodeID      string  `json:"nodeId"`
	AvgDelayS   float64 `json:"avgDelay_s"`
	MaxQueueVeh float64 `json:"maxQueue_veh"`
	MaxQueueM   float64 `json:"maxQueue_m"`
	StopsPerVeh float64 `json:"stopsPerVeh"`
	CO2KgH      float64 `json:"co2_kg_h"`
	Vehicles    int     `json:"vehicles"`
}

// Plan is the exported document. It is the public interface of the system.
type Plan struct {
	Schema      string         `json:"schema"`
	City        string         `json:"city"`
	Scenario    string         `json:"scenario"`
	Strategy    Strategy       `json:"strategy,omitempty"`
	GeneratedAt time.Time      `json:"generatedAt"`
	Source      string         `json:"source"`
	Corridors   []CorridorPlan `json:"corridors"`
	Nodes       []NodePlan     `json:"nodes"`
	KPIs        *KPIReport     `json:"kpis,omitempty"`
}
