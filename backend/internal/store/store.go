// Package store persists plans and runs. The Postgres implementation is the
// production one; the memory implementation keeps tests and demos independent
// from a database.
package store

import (
	"context"
	"errors"
	"time"

	"wayraflow/backend/internal/domain"
)

// ErrNotFound is returned when a plan or run does not exist.
var ErrNotFound = errors.New("not found")

// PlanSummary is the list view of a stored plan.
type PlanSummary struct {
	ID         int64           `json:"id"`
	ScenarioID string          `json:"scenarioId"`
	Strategy   domain.Strategy `json:"strategy"`
	CreatedAt  time.Time       `json:"createdAt"`
}

// Store is the persistence contract used by the HTTP layer.
type Store interface {
	Ping(ctx context.Context) error
	Scenarios(ctx context.Context) ([]domain.Scenario, error)
	Network(ctx context.Context) (domain.Network, error)
	SavePlan(ctx context.Context, scenarioID string, strategy domain.Strategy, p domain.Plan) (int64, error)
	SaveRun(ctx context.Context, planID int64, kpis domain.KPIReport, durationMS int) (int64, error)
	GetPlan(ctx context.Context, id int64) (domain.Plan, error)
	GetRun(ctx context.Context, id int64) (domain.KPIReport, error)
	ListPlans(ctx context.Context, limit int) ([]PlanSummary, error)
	Close()
}
