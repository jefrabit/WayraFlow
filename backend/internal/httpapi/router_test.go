package httpapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"wayraflow/backend/internal/domain"
	"wayraflow/backend/internal/store"
)

func newTestRouter() http.Handler {
	return NewRouter(store.NewMemory())
}

func TestHealthReturnsOK(t *testing.T) {
	rec := httptest.NewRecorder()
	newTestRouter().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/health", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	var body map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("invalid JSON body %q: %v", rec.Body.String(), err)
	}
	if body["status"] != "ok" {
		t.Fatalf("status field = %q, want ok", body["status"])
	}
}

func TestHealthRejectsNonGet(t *testing.T) {
	rec := httptest.NewRecorder()
	newTestRouter().ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/health", nil))
	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusMethodNotAllowed)
	}
}

func TestScenariosReturnsTheCatalog(t *testing.T) {
	rec := httptest.NewRecorder()
	newTestRouter().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/scenarios", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	var list []domain.Scenario
	if err := json.Unmarshal(rec.Body.Bytes(), &list); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if len(list) != 3 {
		t.Fatalf("scenarios = %d, want 3", len(list))
	}
}

func TestNetworkReturnsGeoJSON(t *testing.T) {
	rec := httptest.NewRecorder()
	newTestRouter().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/network", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	var body struct {
		Type     string            `json:"type"`
		Features []json.RawMessage `json:"features"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if body.Type != "FeatureCollection" {
		t.Fatalf("type = %q", body.Type)
	}
	if len(body.Features) != 11 {
		t.Fatalf("features = %d, want 11 (8 nodes + 3 corridors)", len(body.Features))
	}
}

func createPlan(t *testing.T, router http.Handler, body string) createPlanResponse {
	t.Helper()
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/plans", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	var out createPlanResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	return out
}

func TestCreatePlanReturnsAPersistedRun(t *testing.T) {
	router := newTestRouter()
	out := createPlan(t, router, `{"scenarioId":"base","strategy":"ga","seed":42}`)
	if out.PlanID <= 0 || out.RunID <= 0 {
		t.Fatalf("ids = %d/%d, want positive", out.PlanID, out.RunID)
	}
	if out.Plan.Schema != domain.PlanSchema {
		t.Fatalf("schema = %q", out.Plan.Schema)
	}
	if len(out.Plan.Nodes) != 8 {
		t.Fatalf("nodes = %d, want 8", len(out.Plan.Nodes))
	}
	if len(out.Plan.Corridors) != 3 {
		t.Fatalf("corridors = %d, want 3", len(out.Plan.Corridors))
	}
	if out.KPIs.GA.AvgDelayS <= 0 {
		t.Fatalf("GA delay = %v, want > 0", out.KPIs.GA.AvgDelayS)
	}
	if len(out.Nodes) != 8 {
		t.Fatalf("node KPIs = %d, want 8", len(out.Nodes))
	}
}

func TestCreatePlanRejectsUnknownScenarioAndStrategy(t *testing.T) {
	router := newTestRouter()
	for _, body := range []string{
		`{"scenarioId":"nope","strategy":"ga"}`,
		`{"scenarioId":"base","strategy":"magic"}`,
		`{"scenarioId":"base"}`,
		`{"strategy":"ga"}`,
		`{"scenarioId":"base","strategy":"ga","config":{"populationSize":2}}`,
	} {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/api/v1/plans", strings.NewReader(body))
		router.ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("body %s: status = %d, want 400", body, rec.Code)
		}
		if ct := rec.Header().Get("Content-Type"); ct != "application/problem+json" {
			t.Fatalf("body %s: Content-Type = %q", body, ct)
		}
	}
}

func TestPlanDownloadAndMissingPlan(t *testing.T) {
	router := newTestRouter()
	out := createPlan(t, router, `{"scenarioId":"base","strategy":"webster"}`)

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/plans/1?download=1", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	if cd := rec.Header().Get("Content-Disposition"); !strings.Contains(cd, "attachment") {
		t.Fatalf("Content-Disposition = %q", cd)
	}
	var p domain.Plan
	if err := json.Unmarshal(rec.Body.Bytes(), &p); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if p.Schema != domain.PlanSchema {
		t.Fatalf("schema = %q", p.Schema)
	}

	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/plans/999", nil))
	if rec.Code != http.StatusNotFound {
		t.Fatalf("missing plan status = %d, want 404", rec.Code)
	}
	if out.PlanID != 1 {
		t.Fatalf("plan id = %d", out.PlanID)
	}
}

func TestRunAndPlanList(t *testing.T) {
	router := newTestRouter()
	createPlan(t, router, `{"scenarioId":"dia-de-feria","strategy":"fixed"}`)

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/runs/1", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("run status = %d", rec.Code)
	}
	var kpis domain.KPIReport
	if err := json.Unmarshal(rec.Body.Bytes(), &kpis); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if kpis.Fixed.Vehicles == 0 {
		t.Fatal("run has no vehicles")
	}

	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/plans", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("list status = %d", rec.Code)
	}
	var list []store.PlanSummary
	if err := json.Unmarshal(rec.Body.Bytes(), &list); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if len(list) != 1 || list[0].ScenarioID != "dia-de-feria" {
		t.Fatalf("list = %+v", list)
	}
}

func TestCORSHandlesPreflight(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodOptions, "/api/v1/plans", nil)
	newTestRouter().ServeHTTP(rec, req)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204", rec.Code)
	}
	if rec.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Fatal("missing CORS header")
	}
}
