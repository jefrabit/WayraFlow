// Package httpapi exposes the WayraFlow HTTP/JSON API.
package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"wayraflow/backend/internal/domain"
	"wayraflow/backend/internal/optim"
	"wayraflow/backend/internal/sim"
	"wayraflow/backend/internal/store"
)

// Server wires the store and the engine into HTTP handlers.
type Server struct {
	store store.Store
	now   func() time.Time
}

// NewRouter returns the API handler with all routes registered.
func NewRouter(st store.Store) http.Handler {
	s := &Server{store: st, now: time.Now}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", s.health)
	mux.HandleFunc("GET /api/v1/network", s.network)
	mux.HandleFunc("GET /api/v1/scenarios", s.scenarios)
	mux.HandleFunc("GET /api/v1/plans", s.listPlans)
	mux.HandleFunc("POST /api/v1/plans", s.createPlan)
	mux.HandleFunc("GET /api/v1/plans/{id}", s.getPlan)
	mux.HandleFunc("GET /api/v1/runs/{id}", s.getRun)
	return withCORS(withLogging(mux))
}

func (s *Server) health(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()
	if err := s.store.Ping(ctx); err != nil {
		writeProblem(w, http.StatusServiceUnavailable, "Service Unavailable", "the database is not reachable")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) scenarios(w http.ResponseWriter, r *http.Request) {
	list, err := s.store.Scenarios(r.Context())
	if err != nil {
		writeProblem(w, http.StatusInternalServerError, "Internal Server Error", "could not read scenarios")
		return
	}
	writeJSON(w, http.StatusOK, list)
}

func (s *Server) network(w http.ResponseWriter, r *http.Request) {
	net, err := s.store.Network(r.Context())
	if err != nil {
		writeProblem(w, http.StatusInternalServerError, "Internal Server Error", "could not read the network")
		return
	}
	features := make([]map[string]any, 0, len(net.Nodes)+len(net.Corridors))
	for _, n := range net.Nodes {
		features = append(features, map[string]any{
			"type":     "Feature",
			"geometry": map[string]any{"type": "Point", "coordinates": []float64{n.Lon, n.Lat}},
			"properties": map[string]any{
				"id": n.ID, "name": n.Name, "kind": "node",
				"signalized": n.Signalized, "osmNode": n.OSMNodeID,
			},
		})
	}
	for _, c := range net.Corridors {
		coords := make([][]float64, 0, len(c.Nodes))
		for _, cn := range c.Nodes {
			if n, ok := net.Node(cn.NodeID); ok {
				coords = append(coords, []float64{n.Lon, n.Lat})
			}
		}
		features = append(features, map[string]any{
			"type":     "Feature",
			"geometry": map[string]any{"type": "LineString", "coordinates": coords},
			"properties": map[string]any{
				"id": c.ID, "name": c.Name, "kind": "corridor",
			},
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"type":        "FeatureCollection",
		"features":    features,
		"attribution": domain.SourceODbL,
	})
}

type createPlanRequest struct {
	ScenarioID string `json:"scenarioId"`
	Strategy   string `json:"strategy"`
	Seed       *int64 `json:"seed"`
	Config     *struct {
		Population  int `json:"populationSize"`
		Generations int `json:"maxGenerations"`
		BudgetMS    int `json:"budgetMs"`
	} `json:"config,omitempty"`
}

type createPlanResponse struct {
	PlanID     int64            `json:"planId"`
	RunID      int64            `json:"runId"`
	DurationMS int              `json:"durationMs"`
	Plan       domain.Plan      `json:"plan"`
	KPIs       domain.KPIReport `json:"kpis"`
	Nodes      []domain.NodeKPI `json:"nodes"`
}

func (s *Server) createPlan(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	var req createPlanRequest
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&req); err != nil {
		writeProblem(w, http.StatusBadRequest, "Bad Request", "invalid JSON body: "+err.Error())
		return
	}
	if req.ScenarioID == "" {
		writeProblem(w, http.StatusBadRequest, "Bad Request", "scenarioId is required")
		return
	}
	strategy := domain.Strategy(req.Strategy)
	if !strategy.Valid() {
		writeProblem(w, http.StatusBadRequest, "Bad Request", "strategy must be fixed, webster or ga")
		return
	}
	seed := int64(42)
	if req.Seed != nil {
		seed = *req.Seed
	}
	cfg := optim.DefaultConfig(seed)
	if req.Config != nil {
		if req.Config.Population != 0 {
			if req.Config.Population < 4 || req.Config.Population > 500 {
				writeProblem(w, http.StatusBadRequest, "Bad Request", "config.populationSize must be within [4, 500]")
				return
			}
			cfg.Population = req.Config.Population
		}
		if req.Config.Generations != 0 {
			if req.Config.Generations < 1 || req.Config.Generations > 1000 {
				writeProblem(w, http.StatusBadRequest, "Bad Request", "config.maxGenerations must be within [1, 1000]")
				return
			}
			cfg.Generations = req.Config.Generations
		}
		if req.Config.BudgetMS != 0 {
			if req.Config.BudgetMS < 100 || req.Config.BudgetMS > 10000 {
				writeProblem(w, http.StatusBadRequest, "Bad Request", "config.budgetMs must be within [100, 10000]")
				return
			}
			cfg.Budget = time.Duration(req.Config.BudgetMS) * time.Millisecond
		}
	}

	scenarios, err := s.store.Scenarios(r.Context())
	if err != nil {
		writeProblem(w, http.StatusInternalServerError, "Internal Server Error", "could not read scenarios")
		return
	}
	var sc domain.Scenario
	found := false
	for _, candidate := range scenarios {
		if candidate.ID == req.ScenarioID {
			sc, found = candidate, true
			break
		}
	}
	if !found {
		writeProblem(w, http.StatusBadRequest, "Bad Request", fmt.Sprintf("unknown scenario %q", req.ScenarioID))
		return
	}
	net, err := s.store.Network(r.Context())
	if err != nil {
		writeProblem(w, http.StatusInternalServerError, "Internal Server Error", "could not read the network")
		return
	}

	fixed, webster, ga, err := optim.BuildAllWithConfig(net, sc, seed, s.now(), cfg)
	if err != nil {
		writeProblem(w, http.StatusInternalServerError, "Internal Server Error", "the optimizer failed: "+err.Error())
		return
	}
	plans := map[domain.Strategy]domain.Plan{
		domain.StrategyFixed:   fixed,
		domain.StrategyWebster: webster,
		domain.StrategyGA:      ga,
	}
	results := map[domain.Strategy]sim.Result{}
	for strategy, p := range plans {
		res, err := sim.Run(net, sc, p, sim.DefaultHorizonS)
		if err != nil {
			writeProblem(w, http.StatusInternalServerError, "Internal Server Error", "the simulator failed: "+err.Error())
			return
		}
		results[strategy] = res
	}
	report := domain.KPIReport{
		Fixed:   results[domain.StrategyFixed].City,
		Webster: results[domain.StrategyWebster].City,
		GA:      results[domain.StrategyGA].City,
		ImprovementPct: map[string]float64{
			"webster": improvement(results[domain.StrategyFixed].City, results[domain.StrategyWebster].City),
			"ga":      improvement(results[domain.StrategyFixed].City, results[domain.StrategyGA].City),
		},
	}
	chosen := plans[strategy]
	chosen.KPIs = &report
	durationMS := int(time.Since(start).Milliseconds())

	planID, err := s.store.SavePlan(r.Context(), sc.ID, strategy, chosen)
	if err != nil {
		writeProblem(w, http.StatusInternalServerError, "Internal Server Error", "could not store the plan")
		return
	}
	runID, err := s.store.SaveRun(r.Context(), planID, report, durationMS)
	if err != nil {
		writeProblem(w, http.StatusInternalServerError, "Internal Server Error", "could not store the run")
		return
	}
	writeJSON(w, http.StatusCreated, createPlanResponse{
		PlanID:     planID,
		RunID:      runID,
		DurationMS: durationMS,
		Plan:       chosen,
		KPIs:       report,
		Nodes:      results[strategy].Nodes,
	})
}

func improvement(base, candidate domain.KPI) float64 {
	if base.AvgDelayS <= 0 {
		return 0
	}
	return (base.AvgDelayS - candidate.AvgDelayS) / base.AvgDelayS * 100
}

func (s *Server) listPlans(w http.ResponseWriter, r *http.Request) {
	limit := 20
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 100 {
			limit = n
		}
	}
	plans, err := s.store.ListPlans(r.Context(), limit)
	if err != nil {
		writeProblem(w, http.StatusInternalServerError, "Internal Server Error", "could not list plans")
		return
	}
	writeJSON(w, http.StatusOK, plans)
}

func (s *Server) getPlan(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		writeProblem(w, http.StatusBadRequest, "Bad Request", "plan id must be a positive integer")
		return
	}
	p, err := s.store.GetPlan(r.Context(), id)
	if errors.Is(err, store.ErrNotFound) {
		writeProblem(w, http.StatusNotFound, "Not Found", "plan not found")
		return
	}
	if err != nil {
		writeProblem(w, http.StatusInternalServerError, "Internal Server Error", "could not read the plan")
		return
	}
	if r.URL.Query().Get("download") == "1" {
		w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="wayraflow-plan-%d.json"`, id))
	}
	writeJSON(w, http.StatusOK, p)
}

func (s *Server) getRun(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		writeProblem(w, http.StatusBadRequest, "Bad Request", "run id must be a positive integer")
		return
	}
	kpis, err := s.store.GetRun(r.Context(), id)
	if errors.Is(err, store.ErrNotFound) {
		writeProblem(w, http.StatusNotFound, "Not Found", "run not found")
		return
	}
	if err != nil {
		writeProblem(w, http.StatusInternalServerError, "Internal Server Error", "could not read the run")
		return
	}
	writeJSON(w, http.StatusOK, kpis)
}

func pathID(r *http.Request) (int64, error) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id <= 0 {
		return 0, fmt.Errorf("invalid id")
	}
	return id, nil
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeProblem(w http.ResponseWriter, status int, title, detail string) {
	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"type":   "about:blank",
		"title":  title,
		"status": status,
		"detail": detail,
	})
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(status int) {
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}

func withLogging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(rec, r)
		slog.Info("request",
			"method", r.Method,
			"path", r.URL.Path,
			"status", rec.status,
			"duration_ms", time.Since(start).Milliseconds(),
		)
	})
}

func withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}
