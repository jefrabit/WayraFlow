package store

import (
	"context"
	"sort"
	"sync"
	"time"

	"wayraflow/backend/internal/catalog"
	"wayraflow/backend/internal/domain"
)

// Memory is an in-memory Store for tests and for running the API without a
// database. It keeps the same behavior as the Postgres implementation.
type Memory struct {
	mu        sync.RWMutex
	plans     map[int64]domain.Plan
	runs      map[int64]domain.KPIReport
	summaries map[int64]PlanSummary
	nextPlan  int64
	nextRun   int64
	now       func() time.Time
}

// NewMemory returns an empty in-memory store seeded with the catalog.
func NewMemory() *Memory {
	return &Memory{
		plans:     map[int64]domain.Plan{},
		runs:      map[int64]domain.KPIReport{},
		summaries: map[int64]PlanSummary{},
		now:       time.Now,
	}
}

func (m *Memory) Ping(context.Context) error { return nil }

func (m *Memory) Scenarios(context.Context) ([]domain.Scenario, error) {
	return catalog.Scenarios(), nil
}

func (m *Memory) Network(context.Context) (domain.Network, error) {
	return catalog.Network(), nil
}

func (m *Memory) SavePlan(_ context.Context, scenarioID string, strategy domain.Strategy, p domain.Plan) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.nextPlan++
	id := m.nextPlan
	m.plans[id] = p
	m.summaries[id] = PlanSummary{ID: id, ScenarioID: scenarioID, Strategy: strategy, CreatedAt: m.now().UTC()}
	return id, nil
}

func (m *Memory) SaveRun(_ context.Context, planID int64, kpis domain.KPIReport, _ int) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.nextRun++
	id := m.nextRun
	m.runs[id] = kpis
	return id, nil
}

func (m *Memory) GetPlan(_ context.Context, id int64) (domain.Plan, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	p, ok := m.plans[id]
	if !ok {
		return domain.Plan{}, ErrNotFound
	}
	return p, nil
}

func (m *Memory) GetRun(_ context.Context, id int64) (domain.KPIReport, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	k, ok := m.runs[id]
	if !ok {
		return domain.KPIReport{}, ErrNotFound
	}
	return k, nil
}

func (m *Memory) ListPlans(_ context.Context, limit int) ([]PlanSummary, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]PlanSummary, 0, len(m.summaries))
	for _, s := range m.summaries {
		out = append(out, s)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID > out[j].ID })
	if limit > 0 && len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

func (m *Memory) Close() {}
