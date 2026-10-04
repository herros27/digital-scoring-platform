package tests

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/herros27/digital-scoring/internal/api/handler"
	"github.com/herros27/digital-scoring/internal/domain"
	"github.com/herros27/digital-scoring/internal/hub"
	"github.com/herros27/digital-scoring/internal/rules"
	"github.com/herros27/digital-scoring/internal/service"
	"github.com/herros27/digital-scoring/internal/store"
)

type memoryRulesetRepo struct {
	data map[string]store.Ruleset
}

func newMemoryRulesetRepo() *memoryRulesetRepo {
	return &memoryRulesetRepo{data: make(map[string]store.Ruleset)}
}

func (m *memoryRulesetRepo) Create(r *store.Ruleset) error {
	if r.ID == "" {
		r.ID = "ruleset-uuid-" + r.Version
	}
	m.data[r.ID] = *r
	return nil
}

func (m *memoryRulesetRepo) Update(r *store.Ruleset) error {
	if _, ok := m.data[r.ID]; !ok {
		return domain.ErrNotFound
	}
	m.data[r.ID] = *r
	return nil
}

func (m *memoryRulesetRepo) Delete(id string) error {
	if _, ok := m.data[id]; !ok {
		return domain.ErrNotFound
	}
	delete(m.data, id)
	return nil
}

func (m *memoryRulesetRepo) List() ([]store.Ruleset, error) {
	var list []store.Ruleset
	for _, v := range m.data {
		list = append(list, v)
	}
	return list, nil
}

func (m *memoryRulesetRepo) GetByID(id string) (*store.Ruleset, error) {
	if r, ok := m.data[id]; ok {
		return &r, nil
	}
	return nil, domain.ErrNotFound
}

func (m *memoryRulesetRepo) GetActive() (*store.Ruleset, error) {
	for _, v := range m.data {
		if v.Status == "active" {
			return &v, nil
		}
	}
	return nil, domain.ErrNotFound
}

func TestRulesetService_CRUDAndResolve(t *testing.T) {
	repo := newMemoryRulesetRepo()
	svc := service.NewRulesetService(repo)

	// 1. Create Ruleset via Service
	input := service.CreateRulesetInput{
		Name:    "Kejurda Remaja 2026",
		Version: "1.0",
		Scoring: rules.ScoringRule{
			TechniquePoints:  1,
			KnockdownPoints:  3,
			FallPoints:       4,
			MaxPenalties:     3,
			PenaltyThreshold: 1,
			RoundDuration:    90,
			BreakDuration:    45,
			TotalRounds:      2,
		},
		Status: "active",
	}

	created, err := svc.CreateRuleset(input)
	if err != nil {
		t.Fatalf("CreateRuleset failed: %v", err)
	}
	if created.ID == "" || created.Name != "Kejurda Remaja 2026" {
		t.Fatalf("unexpected created ruleset: %+v", created)
	}

	// 2. Resolve Ruleset for Engine
	resolved, err := svc.ResolveRuleset(created.ID)
	if err != nil {
		t.Fatalf("ResolveRuleset by ID failed: %v", err)
	}
	if resolved.Scoring.RoundDuration != 90 || resolved.Scoring.FallPoints != 4 {
		t.Fatalf("unexpected resolved scoring: %+v", resolved.Scoring)
	}

	// 3. Update Ruleset
	input.Scoring.FallPoints = 5
	updated, err := svc.UpdateRuleset(created.ID, input)
	if err != nil {
		t.Fatalf("UpdateRuleset failed: %v", err)
	}
	resolvedUpdated, _ := svc.ResolveRuleset(created.ID)
	if resolvedUpdated.Scoring.FallPoints != 5 {
		t.Fatalf("expected updated FallPoints=5, got %d", resolvedUpdated.Scoring.FallPoints)
	}
	_ = updated

	// 4. Delete Ruleset
	if err := svc.DeleteRuleset(created.ID); err != nil {
		t.Fatalf("DeleteRuleset failed: %v", err)
	}
	if _, err := svc.GetRulesetByID(created.ID); err == nil {
		t.Fatalf("expected error after delete, got nil")
	}
}

func TestRulesetHandler_HTTP_CRUD(t *testing.T) {
	repo := newMemoryRulesetRepo()
	svc := service.NewRulesetService(repo)
	h := &handler.RulesetHandler{Service: svc}

	// 1. POST /api/v1/rulesets
	body, _ := json.Marshal(map[string]any{
		"name":    "O2SN Silat 2026",
		"version": "2026.1",
		"scoring": map[string]int{
			"technique_points":  1,
			"knockdown_points":  2,
			"fall_points":       3,
			"max_penalties":     3,
			"penalty_threshold": 1,
			"round_duration":    100,
			"break_duration":    50,
			"total_rounds":      3,
		},
		"status": "active",
	})
	req := httptest.NewRequest("POST", "/api/v1/rulesets", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	h.Create(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201 Created, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp struct {
		Data domain.Ruleset `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	rulesetID := resp.Data.ID

	// 2. GET /api/v1/rulesets/{id}
	getReq := httptest.NewRequest("GET", "/api/v1/rulesets/"+rulesetID, nil)
	getReq.SetPathValue("id", rulesetID)
	getRec := httptest.NewRecorder()
	h.Get(getRec, getReq)

	if getRec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d: %s", getRec.Code, getRec.Body.String())
	}

	// 3. PUT /api/v1/rulesets/{id}
	updateBody, _ := json.Marshal(map[string]any{
		"name":    "O2SN Silat 2026 Updated",
		"version": "2026.2",
		"scoring": map[string]int{
			"technique_points":  2,
			"knockdown_points":  3,
			"fall_points":       4,
			"max_penalties":     3,
			"penalty_threshold": 1,
			"round_duration":    120,
			"break_duration":    60,
			"total_rounds":      3,
		},
		"status": "active",
	})
	putReq := httptest.NewRequest("PUT", "/api/v1/rulesets/"+rulesetID, bytes.NewReader(updateBody))
	putReq.SetPathValue("id", rulesetID)
	putRec := httptest.NewRecorder()
	h.Update(putRec, putReq)

	if putRec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK on update, got %d: %s", putRec.Code, putRec.Body.String())
	}

	// 4. DELETE /api/v1/rulesets/{id}
	delReq := httptest.NewRequest("DELETE", "/api/v1/rulesets/"+rulesetID, nil)
	delReq.SetPathValue("id", rulesetID)
	delRec := httptest.NewRecorder()
	h.Delete(delRec, delReq)

	if delRec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK on delete, got %d: %s", delRec.Code, delRec.Body.String())
	}
}

type mockCompRepoWithRuleset struct {
	comps map[string]*domain.Competition
}

func (c *mockCompRepoWithRuleset) Create(comp *domain.Competition) error { return nil }
func (c *mockCompRepoWithRuleset) GetByID(id string) (*domain.Competition, error) {
	if comp, ok := c.comps[id]; ok {
		return comp, nil
	}
	return nil, domain.ErrNotFound
}
func (c *mockCompRepoWithRuleset) List() ([]domain.Competition, error)   { return nil, nil }
func (c *mockCompRepoWithRuleset) UpdateStatus(id, status string) error  { return nil }
func (c *mockCompRepoWithRuleset) Update(comp *domain.Competition) error { return nil }

func TestMatchService_DynamicRulesetLoading(t *testing.T) {
	repo := newMemoryRulesetRepo()
	rulesetSvc := service.NewRulesetService(repo)

	// Create custom ruleset
	customRuleset, err := rulesetSvc.CreateRuleset(service.CreateRulesetInput{
		Name:    "Turnamen Khusus Silat",
		Version: "1.0",
		Scoring: rules.ScoringRule{
			TechniquePoints: 2,
			KnockdownPoints: 5,
			RoundDuration:   60,
			BreakDuration:   30,
			TotalRounds:     1,
		},
	})
	if err != nil {
		t.Fatalf("CreateRuleset failed: %v", err)
	}

	matchRepo := newMockMatchRepo()
	compRepo := &mockCompRepoWithRuleset{
		comps: map[string]*domain.Competition{
			"comp-dyn": {
				ID:        "comp-dyn",
				Name:      "Kejuaraan Khusus",
				RulesetID: &customRuleset.ID,
			},
		},
	}
	matchRepo.matches["match-dyn"] = &store.Match{
		ID:            "match-dyn",
		CompetitionID: "comp-dyn",
		ArenaID:       "arena-dyn",
		Status:        "idle",
	}

	hubMgr := hub.NewManager()
	matchSvc := service.NewMatchService(matchRepo, compRepo, hubMgr, nil, nil, nil, nil)
	matchSvc.SetRulesetResolver(rulesetSvc)

	// Load match to arena dengan rulesetPath kosong -> otomatis mengambil dari comp.RulesetID!
	if err := matchSvc.LoadMatchToArena("match-dyn", "arena-dyn", ""); err != nil {
		t.Fatalf("LoadMatchToArena with dynamic ruleset failed: %v", err)
	}

	orch, err := matchSvc.GetOrchestrator("match-dyn")
	if err != nil {
		t.Fatalf("GetOrchestrator failed: %v", err)
	}
	if orch == nil {
		t.Fatalf("orchestrator is nil")
	}
}
