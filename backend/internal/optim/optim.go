// Package optim builds traffic signal plans: the current fixed cycle, the
// Webster baseline and a genetic algorithm that coordinates greens and offsets.
package optim

import (
	"fmt"
	"math"
	"math/rand"
	"sort"
	"time"

	"wayraflow/backend/internal/domain"
	"wayraflow/backend/internal/plan"
	"wayraflow/backend/internal/sim"
)

const (
	// selectionHorizonS is the window used to pick the final GA candidate. It
	// matches the reported analysis hour, so the winner cannot look better on a
	// short window and worse on the full one.
	selectionHorizonS = sim.DefaultHorizonS
	// MinCycle and MaxCycle bound every corridor cycle (README 4.5).
	MinCycle = 40.0
	MaxCycle = 120.0
	// FixedCycle is the assumed current plan (README 4.6).
	FixedCycle = 90.0
	// AmberS and AllRedS are the clearance times of each phase.
	AmberS  = 3.0
	AllRedS = 2.0
	// MinGreen is the shortest green a pedestrian can cross with.
	MinGreen = 7.0
	// SatFlowVehH is the saturation flow of a two-lane approach.
	SatFlowVehH = 3600.0
	// CrossSatFlowVehH is the saturation flow of a single-lane side street.
	CrossSatFlowVehH = 1800.0
	// ProgressionSpeedMS is the design speed of the green wave (40 km/h).
	ProgressionSpeedMS = 11.11
	// CrossFlowRatio is the side-street demand as a fraction of the corridor.
	CrossFlowRatio = 0.6
)

// Config controls the genetic algorithm.
type Config struct {
	Population  int
	Generations int
	Budget      time.Duration
	Seed        int64
}

// DefaultConfig returns the parameters documented in README 4.5.
func DefaultConfig(seed int64) Config {
	return Config{Population: 60, Generations: 150, Budget: 3 * time.Second, Seed: seed}
}

// BuildAll returns the three plans for a scenario. They share the timestamp so
// the comparison is reproducible.
func BuildAll(net domain.Network, sc domain.Scenario, seed int64, now time.Time) (domain.Plan, domain.Plan, domain.Plan, error) {
	return BuildAllWithConfig(net, sc, seed, now, DefaultConfig(seed))
}

// BuildAllWithConfig is BuildAll with explicit GA parameters.
func BuildAllWithConfig(net domain.Network, sc domain.Scenario, seed int64, now time.Time, cfg Config) (domain.Plan, domain.Plan, domain.Plan, error) {
	fixed, err := build(net, sc, domain.StrategyFixed, seed, now, cfg)
	if err != nil {
		return domain.Plan{}, domain.Plan{}, domain.Plan{}, err
	}
	webster, err := build(net, sc, domain.StrategyWebster, seed, now, cfg)
	if err != nil {
		return domain.Plan{}, domain.Plan{}, domain.Plan{}, err
	}
	ga, err := build(net, sc, domain.StrategyGA, seed, now, cfg)
	if err != nil {
		return domain.Plan{}, domain.Plan{}, domain.Plan{}, err
	}
	return fixed, webster, ga, nil
}

// Build returns a validated plan for the requested strategy.
func Build(net domain.Network, sc domain.Scenario, strategy domain.Strategy, seed int64, now time.Time) (domain.Plan, error) {
	return build(net, sc, strategy, seed, now, DefaultConfig(seed))
}

func build(net domain.Network, sc domain.Scenario, strategy domain.Strategy, seed int64, now time.Time, cfg Config) (domain.Plan, error) {
	if !strategy.Valid() {
		return domain.Plan{}, fmt.Errorf("unknown strategy %q", strategy)
	}
	if err := sc.Demand.Mix.Validate(); err != nil {
		return domain.Plan{}, fmt.Errorf("scenario %s: %w", sc.ID, err)
	}
	topo := newTopology(net)
	var timing timing
	switch strategy {
	case domain.StrategyFixed:
		timing = fixedTiming(topo)
	case domain.StrategyWebster:
		timing = websterTiming(topo, sc)
	case domain.StrategyGA:
		timing = gaTiming(topo, sc, cfg)
	}
	p := assemble(topo, sc, timing, strategy, now)
	if err := plan.Validate(p); err != nil {
		return domain.Plan{}, fmt.Errorf("generated plan is invalid: %w", err)
	}
	return p, nil
}

// timing is the decoded decision of any strategy. Offsets are absolute per
// node: N3 belongs to C1 and C2, and one signal cannot have two clocks.
type timing struct {
	cycle  map[int]float64       // group index -> cycle seconds
	green  map[string][2]float64 // node id -> green of each phase
	offset map[string]float64    // node id -> absolute offset
}

func fixedTiming(topo topology) timing {
	t := timing{
		cycle:  map[int]float64{},
		green:  map[string][2]float64{},
		offset: map[string]float64{},
	}
	for i := range topo.groups {
		t.cycle[i] = FixedCycle
	}
	total := FixedCycle - 2*(AmberS+AllRedS)
	for _, n := range topo.net.Nodes {
		t.green[n.ID] = [2]float64{total / 2, total / 2}
	}
	t.offset = progressionOffsets(topo, t.cycle)
	return t
}

func websterTiming(topo topology, sc domain.Scenario) timing {
	t := timing{
		cycle:  map[int]float64{},
		green:  map[string][2]float64{},
		offset: map[string]float64{},
	}
	// The cycle is chosen per group: corridors that share a node must share it.
	for gi := range topo.groups {
		var cycle float64
		for _, n := range topo.net.Nodes {
			if topo.nodeGroup[n.ID] != gi {
				continue
			}
			if c := websterCycle(topo, sc, n.ID); c > cycle {
				cycle = c
			}
		}
		if cycle == 0 {
			cycle = FixedCycle
		}
		t.cycle[gi] = cycle
	}
	for _, n := range topo.net.Nodes {
		c := t.cycle[topo.nodeGroup[n.ID]]
		t.green[n.ID] = splitGreens(c, websterSplit(topo, sc, n.ID))
	}
	t.offset = progressionOffsets(topo, t.cycle)
	return t
}

// websterCycle applies C = (1.5L + 5) / (1 - Y) with L = 10 s of lost time.
func websterCycle(topo topology, sc domain.Scenario, nodeID string) float64 {
	y := flowRatios(topo, sc, nodeID)
	sum := y[0] + y[1]
	if sum >= 0.95 {
		return MaxCycle
	}
	c := (1.5*(2*(AmberS+AllRedS)) + 5) / (1 - sum)
	return math.Round(math.Min(math.Max(c, MinCycle), MaxCycle))
}

func websterSplit(topo topology, sc domain.Scenario, nodeID string) float64 {
	y := flowRatios(topo, sc, nodeID)
	if y[0]+y[1] == 0 {
		return 0.5
	}
	return y[0] / (y[0] + y[1])
}

// flowRatios returns y = q/s for the main and cross phases of a node.
func flowRatios(topo topology, sc domain.Scenario, nodeID string) [2]float64 {
	flow := topo.flow(sc, nodeID)
	eq := sc.Demand.Mix.EquivalentFactor()
	return [2]float64{flow * eq / SatFlowVehH, flow * CrossFlowRatio * eq / CrossSatFlowVehH}
}

// splitGreens distributes the available green proportionally to split, keeping
// the 7 s minimum and making the sum exact.
func splitGreens(cycle, split float64) [2]float64 {
	total := cycle - 2*(AmberS+AllRedS)
	if split < 0 {
		split = 0
	}
	if split > 1 {
		split = 1
	}
	g1 := MinGreen + (total-2*MinGreen)*split
	g1 = math.Round(g1*10) / 10
	g2 := math.Round((total-g1)*10) / 10
	if g1 < MinGreen {
		g1 = MinGreen
		g2 = total - g1
	}
	if g2 < MinGreen {
		g2 = MinGreen
		g1 = total - g2
	}
	return [2]float64{g1, g2}
}

// progressionOffsets aligns every node with the travel time from the first one
// of its corridor. A shared node keeps the offset of the first corridor that
// reaches it, so C2 inherits N3 from C1.
func progressionOffsets(topo topology, cycles map[int]float64) map[string]float64 {
	out := map[string]float64{}
	for _, c := range topo.net.Corridors {
		cycle := cycles[topo.corridorGroup[c.ID]]
		base, seen := out[c.Nodes[0].NodeID]
		if !seen {
			base = 0
			out[c.Nodes[0].NodeID] = 0
		}
		var distance float64
		for i, cn := range c.Nodes {
			if i == 0 {
				continue
			}
			distance += cn.DistanceM
			if _, ok := out[cn.NodeID]; ok {
				continue
			}
			out[cn.NodeID] = round1(math.Mod(base+distance/ProgressionSpeedMS, cycle))
		}
	}
	return out
}

// assemble turns a timing decision into a wayraflow.plan/v1 document.
func assemble(topo topology, sc domain.Scenario, t timing, strategy domain.Strategy, now time.Time) domain.Plan {
	p := domain.Plan{
		Schema:      domain.PlanSchema,
		City:        domain.City,
		Scenario:    sc.ID,
		Strategy:    strategy,
		GeneratedAt: now.UTC().Truncate(time.Second),
		Source:      domain.SourceODbL,
	}
	for _, n := range topo.net.Nodes {
		corridor := topo.nodeCorridor[n.ID]
		cycle := t.cycle[topo.nodeGroup[n.ID]]
		g := t.green[n.ID]
		movements := []string{"Calle transversal"}
		if corridor != "" {
			if c, ok := topo.net.Corridor(corridor); ok {
				movements = []string{c.Name}
			}
		}
		p.Nodes = append(p.Nodes, domain.NodePlan{
			ID:       n.ID,
			OSMNode:  n.OSMNodeID,
			Corridor: corridor,
			CycleS:   cycle,
			Phases: []domain.Phase{
				{ID: 1, Movements: movements, GreenS: g[0], AmberS: AmberS, AllRedS: AllRedS},
				{ID: 2, Movements: []string{"Acceso transversal"}, GreenS: g[1], AmberS: AmberS, AllRedS: AllRedS},
			},
		})
	}
	for _, c := range topo.net.Corridors {
		cycle := t.cycle[topo.corridorGroup[c.ID]]
		offsets := map[string]float64{}
		for _, cn := range c.Nodes {
			offsets[cn.NodeID] = round1(math.Mod(t.offset[cn.NodeID], cycle))
		}
		p.Corridors = append(p.Corridors, domain.CorridorPlan{ID: c.ID, CycleS: cycle, OffsetsS: offsets})
	}
	return p
}

func round1(v float64) float64 { return math.Round(v*10) / 10 }

// topology resolves which corridors share a cycle and which corridor is the
// primary one for each node.
type topology struct {
	net           domain.Network
	groups        [][]string // corridor ids per group
	corridorGroup map[string]int
	nodeCorridor  map[string]string
	nodeGroup     map[string]int
}

func newTopology(net domain.Network) topology {
	parent := map[string]string{}
	for _, c := range net.Corridors {
		parent[c.ID] = c.ID
	}
	var find func(string) string
	find = func(x string) string {
		if parent[x] != x {
			parent[x] = find(parent[x])
		}
		return parent[x]
	}
	byNode := map[string][]string{}
	for _, c := range net.Corridors {
		for _, cn := range c.Nodes {
			byNode[cn.NodeID] = append(byNode[cn.NodeID], c.ID)
		}
	}
	for _, ids := range byNode {
		for i := 1; i < len(ids); i++ {
			a, b := find(ids[0]), find(ids[i])
			if a != b {
				parent[b] = a
			}
		}
	}
	groupOfRoot := map[string]int{}
	var groups [][]string
	corridorGroup := map[string]int{}
	for _, c := range net.Corridors {
		root := find(c.ID)
		gi, ok := groupOfRoot[root]
		if !ok {
			gi = len(groups)
			groupOfRoot[root] = gi
			groups = append(groups, nil)
		}
		groups[gi] = append(groups[gi], c.ID)
		corridorGroup[c.ID] = gi
	}
	nodeCorridor := map[string]string{}
	nodeGroup := map[string]int{}
	for _, n := range net.Nodes {
		if ids := byNode[n.ID]; len(ids) > 0 {
			nodeCorridor[n.ID] = ids[0]
			nodeGroup[n.ID] = corridorGroup[ids[0]]
			continue
		}
		groups = append(groups, nil)
		nodeGroup[n.ID] = len(groups) - 1
	}
	for gi := range groups {
		sort.Strings(groups[gi])
	}
	return topology{net: net, groups: groups, corridorGroup: corridorGroup, nodeCorridor: nodeCorridor, nodeGroup: nodeGroup}
}

// flow returns the demand a node serves: the largest corridor that crosses it,
// or the isolated demand of N8.
func (t topology) flow(sc domain.Scenario, nodeID string) float64 {
	var flow float64
	for _, c := range t.net.Corridors {
		for _, cn := range c.Nodes {
			if cn.NodeID == nodeID {
				if q := sc.Flow(c.ID); q > flow {
					flow = q
				}
			}
		}
	}
	if flow == 0 {
		flow = sc.Flow(nodeID)
	}
	return flow
}

// ---- Genetic algorithm ----

// genome is a flat chromosome with a fixed layout:
// [cycle per group | green split per node | offset per coordinated node].
type genome struct {
	low, high []float64
	cycleBase int
	splitBase int
	offsetIdx map[string]int
}

func newGenome(topo topology) *genome {
	g := &genome{offsetIdx: map[string]int{}}
	for range topo.groups {
		g.low = append(g.low, MinCycle)
		g.high = append(g.high, MaxCycle)
	}
	g.cycleBase = 0
	g.splitBase = len(g.low)
	for range topo.net.Nodes {
		g.low = append(g.low, 0.1)
		g.high = append(g.high, 0.9)
	}
	for _, n := range topo.net.Nodes {
		if topo.nodeCorridor[n.ID] == "" {
			continue
		}
		g.offsetIdx[n.ID] = len(g.low)
		g.low = append(g.low, 0)
		g.high = append(g.high, 1)
	}
	return g
}

func (g *genome) random(r *rand.Rand) []float64 {
	out := make([]float64, len(g.low))
	for i := range out {
		out[i] = g.low[i] + r.Float64()*(g.high[i]-g.low[i])
	}
	return out
}

func (g *genome) clamp(v []float64) {
	for i := range v {
		if v[i] < g.low[i] {
			v[i] = g.low[i]
		}
		if v[i] > g.high[i] {
			v[i] = g.high[i]
		}
	}
}

func (g *genome) decode(topo topology, v []float64) timing {
	t := timing{
		cycle:  map[int]float64{},
		green:  map[string][2]float64{},
		offset: map[string]float64{},
	}
	for gi := range topo.groups {
		t.cycle[gi] = math.Round(v[g.cycleBase+gi])
	}
	for i, n := range topo.net.Nodes {
		t.green[n.ID] = splitGreens(t.cycle[topo.nodeGroup[n.ID]], v[g.splitBase+i])
	}
	for _, n := range topo.net.Nodes {
		idx, ok := g.offsetIdx[n.ID]
		if !ok {
			continue
		}
		t.offset[n.ID] = round1(v[idx] * t.cycle[topo.nodeGroup[n.ID]])
	}
	return t
}

func (g *genome) fitness(topo topology, sc domain.Scenario, v []float64) float64 {
	t := g.decode(topo, v)
	mix := sc.Demand.Mix
	var cost float64
	for _, n := range topo.net.Nodes {
		c := t.cycle[topo.nodeGroup[n.ID]]
		green := t.green[n.ID]
		q := topo.flow(sc, n.ID)
		qCross := q * CrossFlowRatio
		d1, x1 := uniformDelay(c, green[0], q*mix.EquivalentFactor(), SatFlowVehH)
		d2, x2 := uniformDelay(c, green[1], qCross*mix.EquivalentFactor(), CrossSatFlowVehH)
		if topo.position(n.ID) > 0 && green[0] > 0 {
			d1 *= progressionFactor(t, topo, n.ID, c, green[0])
		}
		cost += q*d1 + qCross*d2
		if x1 > 0.95 {
			cost += 1e5 * (x1 - 0.95)
		}
		if x2 > 0.95 {
			cost += 1e5 * (x2 - 0.95)
		}
		stops := q*(1-green[0]/c) + qCross*(1-green[1]/c)
		delayVehS := q*d1 + qCross*d2
		liters := delayVehS/3600*mix.IdleLitersPerHour() + stops*mix.StartLiters()
		cost += 0.05 * liters * mix.FuelDensityKgPerL()
	}
	return cost
}

func (t topology) position(nodeID string) int {
	for _, c := range t.net.Corridors {
		for _, cn := range c.Nodes {
			if cn.NodeID == nodeID {
				return cn.Position
			}
		}
	}
	return 0
}

// progressionFactor rewards an offset that puts the downstream green where the
// platoon arrives.
func progressionFactor(t timing, topo topology, nodeID string, cycle, green float64) float64 {
	for _, c := range topo.net.Corridors {
		for i, cn := range c.Nodes {
			if cn.NodeID != nodeID || i == 0 {
				continue
			}
			up := c.Nodes[i-1]
			ideal := t.offset[up.NodeID] + up.DistanceM/ProgressionSpeedMS
			mis := math.Abs(circular(t.offset[nodeID]-ideal, cycle))
			align := math.Max(0, 1-mis/green)
			return 1 - 0.35*align
		}
	}
	return 1
}

func circular(v, cycle float64) float64 {
	v = math.Mod(v, cycle)
	if v > cycle/2 {
		v -= cycle
	}
	return v
}

// uniformDelay is Webster's first term, with an explicit penalty when demand
// exceeds capacity.
func uniformDelay(cycle, green, qEq, satFlow float64) (float64, float64) {
	lambda := green / cycle
	if lambda <= 0 {
		return 1e6, 2
	}
	x := qEq / (lambda * satFlow)
	if x >= 0.98 {
		return 1e6, x
	}
	return cycle * math.Pow(1-lambda, 2) / (2 * (1 - lambda*x)), x
}

// candidate is one chromosome with its evaluated cost.
type candidate struct {
	genes []float64
	cost  float64
}

func gaTiming(topo topology, sc domain.Scenario, cfg Config) timing {
	if cfg.Population < 4 {
		cfg.Population = 4
	}
	if cfg.Generations < 1 {
		cfg.Generations = 1
	}
	if cfg.Budget <= 0 {
		cfg.Budget = 3 * time.Second
	}
	r := rand.New(rand.NewSource(cfg.Seed))
	g := newGenome(topo)
	start := time.Now()

	pop := make([]candidate, cfg.Population)
	for i := range pop {
		pop[i] = candidate{genes: g.random(r)}
		pop[i].cost = g.fitness(topo, sc, pop[i].genes)
	}
	best := pop[0]
	for _, ind := range pop {
		if ind.cost < best.cost {
			best = ind
		}
	}

	for gen := 0; gen < cfg.Generations; gen++ {
		if gen%10 == 0 && time.Since(start) > cfg.Budget {
			break
		}
		sort.SliceStable(pop, func(i, j int) bool { return pop[i].cost < pop[j].cost })
		next := make([]candidate, 0, cfg.Population)
		next = append(next, pop[0], pop[1]) // elitism
		for len(next) < cfg.Population {
			a := tournament(r, pop)
			b := tournament(r, pop)
			child := blx(r, g, a.genes, b.genes)
			mutate(r, g, child)
			next = append(next, candidate{genes: child, cost: g.fitness(topo, sc, child)})
		}
		pop = next
		for _, ind := range pop {
			if ind.cost < best.cost {
				best = ind
			}
		}
	}

	// The analytical model guides the search; the simulator decides. The whole
	// final population and both baselines compete, so the GA never returns a
	// plan worse than the fixed cycle it is supposed to beat.
	sort.SliceStable(pop, func(i, j int) bool { return pop[i].cost < pop[j].cost })
	candidates := []timing{fixedTiming(topo), websterTiming(topo, sc)}
	for i := range pop {
		candidates = append(candidates, g.decode(topo, pop[i].genes))
	}
	chosen := candidates[0]
	chosenCost := math.Inf(1)
	for _, t := range candidates {
		if cost := simulatedCost(topo, sc, t); cost < chosenCost {
			chosen, chosenCost = t, cost
		}
	}
	return chosen
}

// simulatedCost evaluates a timing with the same mesoscopic model that produces
// the reported KPIs.
func simulatedCost(topo topology, sc domain.Scenario, t timing) float64 {
	p := assemble(topo, sc, t, domain.StrategyGA, time.Time{})
	res, err := sim.Run(topo.net, sc, p, selectionHorizonS)
	if err != nil {
		return math.Inf(1)
	}
	return res.City.AvgDelayS*float64(res.City.Vehicles) +
		0.05*res.City.CO2KgH +
		50*float64(res.City.SpillbackS)
}

func tournament(r *rand.Rand, pop []candidate) candidate {
	best := pop[r.Intn(len(pop))]
	for i := 1; i < 3; i++ {
		cand := pop[r.Intn(len(pop))]
		if cand.cost < best.cost {
			best = cand
		}
	}
	return best
}

func blx(r *rand.Rand, g *genome, a, b []float64) []float64 {
	const alpha = 0.3
	child := make([]float64, len(a))
	for i := range child {
		lo, hi := math.Min(a[i], b[i]), math.Max(a[i], b[i])
		span := hi - lo
		child[i] = lo - alpha*span + r.Float64()*(span*(1+2*alpha))
	}
	g.clamp(child)
	return child
}

func mutate(r *rand.Rand, g *genome, v []float64) {
	const rate = 0.05
	for i := range v {
		if r.Float64() >= rate {
			continue
		}
		v[i] += r.NormFloat64() * 0.05 * (g.high[i] - g.low[i])
	}
	g.clamp(v)
}
