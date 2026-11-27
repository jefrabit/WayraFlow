package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"wayraflow/backend/internal/domain"
)

// Postgres is the production Store backed by PostgreSQL/PostGIS.
type Postgres struct {
	pool *pgxpool.Pool
}

// NewPostgres wraps an existing pool.
func NewPostgres(pool *pgxpool.Pool) *Postgres { return &Postgres{pool: pool} }

func (p *Postgres) Ping(ctx context.Context) error { return p.pool.Ping(ctx) }

func (p *Postgres) Scenarios(ctx context.Context) ([]domain.Scenario, error) {
	rows, err := p.pool.Query(ctx, `SELECT id, name, COALESCE(description, ''), demand FROM scenarios ORDER BY id`)
	if err != nil {
		return nil, fmt.Errorf("query scenarios: %w", err)
	}
	defer rows.Close()
	var out []domain.Scenario
	for rows.Next() {
		var s domain.Scenario
		var raw []byte
		if err := rows.Scan(&s.ID, &s.Name, &s.Description, &raw); err != nil {
			return nil, fmt.Errorf("scan scenario: %w", err)
		}
		if err := json.Unmarshal(raw, &s.Demand); err != nil {
			return nil, fmt.Errorf("scenario %s demand: %w", s.ID, err)
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

func (p *Postgres) Network(ctx context.Context) (domain.Network, error) {
	net := domain.Network{City: domain.City}
	rows, err := p.pool.Query(ctx, `
		SELECT id, name, osm_node_id, signalized, ST_X(geom), ST_Y(geom)
		FROM nodes ORDER BY id`)
	if err != nil {
		return net, fmt.Errorf("query nodes: %w", err)
	}
	for rows.Next() {
		var n domain.Node
		if err := rows.Scan(&n.ID, &n.Name, &n.OSMNodeID, &n.Signalized, &n.Lon, &n.Lat); err != nil {
			rows.Close()
			return net, fmt.Errorf("scan node: %w", err)
		}
		net.Nodes = append(net.Nodes, n)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return net, err
	}

	rows, err = p.pool.Query(ctx, `SELECT id, name FROM corridors ORDER BY id`)
	if err != nil {
		return net, fmt.Errorf("query corridors: %w", err)
	}
	index := map[string]int{}
	for rows.Next() {
		var c domain.Corridor
		if err := rows.Scan(&c.ID, &c.Name); err != nil {
			rows.Close()
			return net, fmt.Errorf("scan corridor: %w", err)
		}
		index[c.ID] = len(net.Corridors)
		net.Corridors = append(net.Corridors, c)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return net, err
	}

	rows, err = p.pool.Query(ctx, `
		SELECT corridor_id, node_id, position, COALESCE(distance_m, 0)
		FROM corridor_nodes ORDER BY corridor_id, position`)
	if err != nil {
		return net, fmt.Errorf("query corridor nodes: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var corridorID string
		var cn domain.CorridorNode
		if err := rows.Scan(&corridorID, &cn.NodeID, &cn.Position, &cn.DistanceM); err != nil {
			return net, fmt.Errorf("scan corridor node: %w", err)
		}
		if i, ok := index[corridorID]; ok {
			net.Corridors[i].Nodes = append(net.Corridors[i].Nodes, cn)
		}
	}
	return net, rows.Err()
}

func (p *Postgres) SavePlan(ctx context.Context, scenarioID string, strategy domain.Strategy, plan domain.Plan) (int64, error) {
	raw, err := json.Marshal(plan)
	if err != nil {
		return 0, fmt.Errorf("marshal plan: %w", err)
	}
	var id int64
	err = p.pool.QueryRow(ctx,
		`INSERT INTO plans (scenario_id, strategy, plan) VALUES ($1, $2, $3) RETURNING id`,
		scenarioID, string(strategy), raw).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("insert plan: %w", err)
	}
	return id, nil
}

func (p *Postgres) SaveRun(ctx context.Context, planID int64, kpis domain.KPIReport, durationMS int) (int64, error) {
	raw, err := json.Marshal(kpis)
	if err != nil {
		return 0, fmt.Errorf("marshal kpis: %w", err)
	}
	var id int64
	err = p.pool.QueryRow(ctx,
		`INSERT INTO runs (plan_id, kpis, duration_ms) VALUES ($1, $2, $3) RETURNING id`,
		planID, raw, durationMS).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("insert run: %w", err)
	}
	return id, nil
}

func (p *Postgres) GetPlan(ctx context.Context, id int64) (domain.Plan, error) {
	var raw []byte
	err := p.pool.QueryRow(ctx, `SELECT plan FROM plans WHERE id = $1`, id).Scan(&raw)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Plan{}, ErrNotFound
	}
	if err != nil {
		return domain.Plan{}, fmt.Errorf("select plan: %w", err)
	}
	var plan domain.Plan
	if err := json.Unmarshal(raw, &plan); err != nil {
		return domain.Plan{}, fmt.Errorf("decode plan %d: %w", id, err)
	}
	return plan, nil
}

func (p *Postgres) GetRun(ctx context.Context, id int64) (domain.KPIReport, error) {
	var raw []byte
	err := p.pool.QueryRow(ctx, `SELECT kpis FROM runs WHERE id = $1`, id).Scan(&raw)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.KPIReport{}, ErrNotFound
	}
	if err != nil {
		return domain.KPIReport{}, fmt.Errorf("select run: %w", err)
	}
	var kpis domain.KPIReport
	if err := json.Unmarshal(raw, &kpis); err != nil {
		return domain.KPIReport{}, fmt.Errorf("decode run %d: %w", id, err)
	}
	return kpis, nil
}

func (p *Postgres) ListPlans(ctx context.Context, limit int) ([]PlanSummary, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	rows, err := p.pool.Query(ctx, `
		SELECT id, scenario_id, strategy, created_at FROM plans ORDER BY id DESC LIMIT $1`, limit)
	if err != nil {
		return nil, fmt.Errorf("list plans: %w", err)
	}
	defer rows.Close()
	var out []PlanSummary
	for rows.Next() {
		var s PlanSummary
		if err := rows.Scan(&s.ID, &s.ScenarioID, &s.Strategy, &s.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan plan summary: %w", err)
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

func (p *Postgres) Close() { p.pool.Close() }
