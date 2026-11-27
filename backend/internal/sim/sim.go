// Package sim is the mesoscopic city model: one queue per approach, one-second
// steps, platoon propagation along corridors and per-node KPIs.
package sim

import (
	"fmt"
	"math"

	"wayraflow/backend/internal/domain"
)

const (
	// DefaultHorizonS is the analysis hour.
	DefaultHorizonS = 3600
	// MainServicePerS is the discharge of a two-lane approach at saturation.
	MainServicePerS = 1.0
	// CrossServicePerS is the discharge of a single-lane side street.
	CrossServicePerS = 0.5
	// DefaultStorageM is the link storage when the distance is unknown.
	DefaultStorageM = 300.0
)

// Result is the measurement of one plan.
type Result struct {
	City  domain.KPI
	Nodes []domain.NodeKPI
}

// Run simulates the plan over the horizon. It is deterministic: there is no
// randomness, so the same inputs always produce the same KPIs.
func Run(net domain.Network, sc domain.Scenario, p domain.Plan, horizon int) (Result, error) {
	if horizon <= 0 {
		horizon = DefaultHorizonS
	}
	if err := sc.Demand.Mix.Validate(); err != nil {
		return Result{}, fmt.Errorf("scenario %s: %w", sc.ID, err)
	}
	byID := make(map[string]domain.NodePlan, len(p.Nodes))
	for _, n := range p.Nodes {
		byID[n.ID] = n
	}
	offsets := map[string]float64{}
	for _, c := range p.Corridors {
		for id, off := range c.OffsetsS {
			offsets[id] = off
		}
	}

	mix := sc.Demand.Mix
	lengthM := mix.AverageLengthM()
	nodes := make(map[string]*nodeSim, len(net.Nodes))
	for _, n := range net.Nodes {
		np, ok := byID[n.ID]
		if !ok {
			return Result{}, fmt.Errorf("plan has no timing for node %s", n.ID)
		}
		if len(np.Phases) < 2 {
			return Result{}, fmt.Errorf("node %s needs two phases", n.ID)
		}
		q := nodeFlow(net, sc, n.ID)
		storage := storageFor(net, n.ID)
		nodes[n.ID] = &nodeSim{
			id:         n.ID,
			qMain:      q,
			qCross:     q * 0.6,
			cycle:      np.CycleS,
			g1:         np.Phases[0].GreenS,
			g2:         np.Phases[1].GreenS,
			offset:     offsets[n.ID],
			departures: make([]float64, horizon),
			storageM:   storage,
			lengthM:    lengthM,
			mix:        mix,
		}
	}

	// A shared node is stepped once, in its primary corridor, but it still acts
	// as the upstream of the second corridor.
	primary := map[string]string{}
	for _, c := range net.Corridors {
		for _, cn := range c.Nodes {
			if _, ok := primary[cn.NodeID]; !ok {
				primary[cn.NodeID] = c.ID
			}
		}
	}
	type link struct {
		down   *nodeSim
		travel int
	}
	type corridor struct {
		id    string
		order []*nodeSim
		links []link
	}
	var corridors []corridor
	for _, c := range net.Corridors {
		cs := corridor{id: c.ID}
		for i, cn := range c.Nodes {
			cs.order = append(cs.order, nodes[cn.NodeID])
			if i > 0 {
				travel := int(math.Round(cn.DistanceM / 11.11))
				if travel < 1 {
					travel = 1
				}
				cs.links = append(cs.links, link{down: nodes[cn.NodeID], travel: travel})
			}
		}
		corridors = append(corridors, cs)
	}
	var isolated []*nodeSim
	for _, n := range net.Nodes {
		if primary[n.ID] == "" {
			isolated = append(isolated, nodes[n.ID])
		}
	}

	for t := 0; t < horizon; t++ {
		for _, c := range corridors {
			for i, ns := range c.order {
				if primary[ns.id] != c.id {
					continue
				}
				arrMain := ns.qMain / 3600
				if i > 0 {
					l := c.links[i-1]
					if t-l.travel >= 0 {
						arrMain = c.order[i-1].departures[t-l.travel]
					}
				}
				ns.step(t, arrMain, ns.qCross/3600)
			}
		}
		for _, ns := range isolated {
			ns.step(t, ns.qMain/3600, ns.qCross/3600)
		}
	}

	res := Result{Nodes: make([]domain.NodeKPI, 0, len(net.Nodes))}
	var totalDelay, totalServed, totalArrivals, totalStops, totalCO2 float64
	var maxQueue, maxQueueM float64
	var spill int
	for _, n := range net.Nodes {
		ns := nodes[n.ID]
		k := ns.kpi(horizon)
		res.Nodes = append(res.Nodes, k)
		totalDelay += ns.delay
		totalServed += ns.served
		totalArrivals += ns.arrivals
		totalStops += ns.stops
		totalCO2 += k.CO2KgH
		maxQueue = math.Max(maxQueue, k.MaxQueueVeh)
		maxQueueM = math.Max(maxQueueM, k.MaxQueueM)
		spill += ns.spillS
	}
	res.City = domain.KPI{
		AvgDelayS:   ratio(totalDelay, totalServed),
		MaxQueueVeh: maxQueue,
		MaxQueueM:   maxQueueM,
		StopsPerVeh: ratio(totalStops, totalArrivals),
		CO2KgH:      totalCO2,
		Vehicles:    int(math.Round(totalArrivals)),
		SpillbackS:  spill,
	}
	return res, nil
}

type nodeSim struct {
	id       string
	qMain    float64
	qCross   float64
	cycle    float64
	g1, g2   float64
	offset   float64
	storageM float64
	lengthM  float64
	mix      domain.Mix

	queue0, queue1 float64
	departures     []float64

	delay    float64
	served   float64
	arrivals float64
	stops    float64
	maxQueue float64
	spillS   int
}

func (n *nodeSim) step(t int, arrMain, arrCross float64) {
	local := math.Mod(float64(t)-n.offset, n.cycle)
	if local < 0 {
		local += n.cycle
	}
	green0 := local < n.g1
	green1 := local >= n.g1+5 && local < n.g1+5+n.g2

	hadQueue0 := n.queue0 > 1e-9
	hadQueue1 := n.queue1 > 1e-9
	n.queue0 += arrMain
	n.queue1 += arrCross
	n.arrivals += arrMain + arrCross
	if !green0 || hadQueue0 {
		n.stops += arrMain
	}
	if !green1 || hadQueue1 {
		n.stops += arrCross
	}

	var served0 float64
	if green0 {
		served0 = math.Min(n.queue0, MainServicePerS)
		n.queue0 -= served0
	}
	if green1 {
		n.queue1 -= math.Min(n.queue1, CrossServicePerS)
	}
	n.departures[t] = served0
	n.served += served0

	n.delay += n.queue0 + n.queue1
	total := n.queue0 + n.queue1
	if total > n.maxQueue {
		n.maxQueue = total
	}
	if total*n.lengthM > n.storageM {
		n.spillS++
	}
}

func (n *nodeSim) kpi(horizon int) domain.NodeKPI {
	scale := 3600 / float64(horizon)
	idle := n.delay / 3600 * n.mix.IdleLitersPerHour()
	starts := n.stops * n.mix.StartLiters()
	co2 := (idle + starts) * n.mix.FuelDensityKgPerL() * scale
	return domain.NodeKPI{
		NodeID:      n.id,
		AvgDelayS:   ratio(n.delay, n.served),
		MaxQueueVeh: n.maxQueue,
		MaxQueueM:   n.maxQueue * n.lengthM,
		StopsPerVeh: ratio(n.stops, n.arrivals),
		CO2KgH:      co2,
		Vehicles:    int(math.Round(n.arrivals)),
	}
}

func ratio(a, b float64) float64 {
	if b <= 0 {
		return 0
	}
	return a / b
}

// nodeFlow is the demand a node serves: the largest corridor crossing it, or
// the isolated demand of N8.
func nodeFlow(net domain.Network, sc domain.Scenario, nodeID string) float64 {
	var flow float64
	for _, c := range net.Corridors {
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

func storageFor(net domain.Network, nodeID string) float64 {
	for _, c := range net.Corridors {
		for _, cn := range c.Nodes {
			if cn.NodeID == nodeID && cn.DistanceM > 0 {
				return math.Min(math.Max(cn.DistanceM, 120), 800)
			}
		}
	}
	return DefaultStorageM
}
