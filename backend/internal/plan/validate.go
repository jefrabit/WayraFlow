// Package plan validates the exported traffic plan against the
// wayraflow.plan/v1 contract before it is stored or downloaded.
package plan

import (
	"fmt"
	"math"
	"sort"

	"wayraflow/backend/internal/domain"
)

// Tolerance absorbs floating point rounding in the phase sums.
const Tolerance = 1e-6

// ValidationError points at the field that broke the contract.
type ValidationError struct {
	Field  string
	Reason string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("%s: %s", e.Field, e.Reason)
}

func invalid(field, format string, args ...any) error {
	return &ValidationError{Field: field, Reason: fmt.Sprintf(format, args...)}
}

// Validate checks the whole document: phase sums, cycle ranges, offsets and
// references. A plan that does not pass is never persisted.
func Validate(p domain.Plan) error {
	if p.Schema != domain.PlanSchema {
		return invalid("schema", "must be %q, got %q", domain.PlanSchema, p.Schema)
	}
	if p.City == "" {
		return invalid("city", "is required")
	}
	if p.Scenario == "" {
		return invalid("scenario", "is required")
	}
	if len(p.Nodes) == 0 {
		return invalid("nodes", "at least one node is required")
	}

	nodes := make(map[string]domain.NodePlan, len(p.Nodes))
	for i, n := range p.Nodes {
		field := fmt.Sprintf("nodes[%d]", i)
		if n.ID == "" {
			return invalid(field+".id", "is required")
		}
		if _, dup := nodes[n.ID]; dup {
			return invalid(field+".id", "duplicate node %q", n.ID)
		}
		if n.CycleS < 40 || n.CycleS > 120 {
			return invalid(field+".cycle_s", "must be within [40, 120], got %v", n.CycleS)
		}
		if len(n.Phases) == 0 {
			return invalid(field+".phases", "at least one phase is required")
		}
		var sum float64
		for j, ph := range n.Phases {
			pfield := fmt.Sprintf("%s.phases[%d]", field, j)
			if ph.GreenS < 7 {
				return invalid(pfield+".green_s", "minimum green is 7 s, got %v", ph.GreenS)
			}
			if ph.AmberS <= 0 {
				return invalid(pfield+".amber_s", "must be > 0")
			}
			if ph.AllRedS < 0 {
				return invalid(pfield+".allRed_s", "must be >= 0")
			}
			sum += ph.Duration()
		}
		if math.Abs(sum-n.CycleS) > Tolerance {
			return invalid(field+".phases", "sum of green+amber+allRed is %v, cycle is %v", sum, n.CycleS)
		}
		nodes[n.ID] = n
	}

	corridors := make(map[string]domain.CorridorPlan, len(p.Corridors))
	for i, c := range p.Corridors {
		field := fmt.Sprintf("corridors[%d]", i)
		if c.ID == "" {
			return invalid(field+".id", "is required")
		}
		if _, dup := corridors[c.ID]; dup {
			return invalid(field+".id", "duplicate corridor %q", c.ID)
		}
		if c.CycleS < 40 || c.CycleS > 120 {
			return invalid(field+".cycle_s", "must be within [40, 120], got %v", c.CycleS)
		}
		if len(c.OffsetsS) == 0 {
			return invalid(field+".offsets_s", "at least one offset is required")
		}
		for nodeID, offset := range c.OffsetsS {
			n, ok := nodes[nodeID]
			if !ok {
				return invalid(field+".offsets_s", "unknown node %q", nodeID)
			}
			if math.Abs(n.CycleS-c.CycleS) > Tolerance {
				return invalid(field+".cycle_s", "corridor cycle %v differs from node %s cycle %v", c.CycleS, nodeID, n.CycleS)
			}
			if offset < 0 || offset >= c.CycleS {
				return invalid(field+".offsets_s."+nodeID, "must be within [0, cycle), got %v", offset)
			}
		}
		corridors[c.ID] = c
	}

	for _, n := range p.Nodes {
		if n.Corridor == "" {
			continue
		}
		if _, ok := corridors[n.Corridor]; !ok {
			return invalid("nodes."+n.ID+".corridor", "unknown corridor %q", n.Corridor)
		}
	}

	return nil
}

// SortedNodeIDs is a small helper for deterministic error messages and tests.
func SortedNodeIDs(p domain.Plan) []string {
	ids := make([]string, 0, len(p.Nodes))
	for _, n := range p.Nodes {
		ids = append(ids, n.ID)
	}
	sort.Strings(ids)
	return ids
}
