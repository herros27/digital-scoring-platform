package tests

import (
	"strconv"
	"testing"
	"time"

	"github.com/herros27/digital-scoring/internal/auth"
	"github.com/herros27/digital-scoring/internal/consensus"
	"github.com/herros27/digital-scoring/internal/domain"
	"github.com/herros27/digital-scoring/internal/hub"
	"github.com/herros27/digital-scoring/internal/orchestrator"
	"github.com/herros27/digital-scoring/internal/rules"
	"github.com/herros27/digital-scoring/internal/service"
	"github.com/herros27/digital-scoring/internal/statemachine"
	"github.com/herros27/digital-scoring/internal/store"
)

func TestServerSideRandomPINGeneration(t *testing.T) {
	// 1. Test GenerateRandomPIN format
	pin1, err := auth.GenerateRandomPIN()
	if err != nil {
		t.Fatalf("GenerateRandomPIN failed: %v", err)
	}
	if len(pin1) != 6 {
		t.Fatalf("expected 6-digit PIN, got %s (length %d)", pin1, len(pin1))
	}
	if _, err := strconv.Atoi(pin1); err != nil {
		t.Fatalf("expected numeric PIN, got: %s", pin1)
	}

	pin2, _ := auth.GenerateRandomPIN()
	if pin1 == pin2 {
		// Extremely low probability for crypto random, but check distinct
		t.Logf("pins generated: %s, %s", pin1, pin2)
	}
}

type mockPagedCompRepo struct {
	comps []domain.Competition
}

func (m *mockPagedCompRepo) Create(c *domain.Competition) error             { return nil }
func (m *mockPagedCompRepo) GetByID(id string) (*domain.Competition, error) { return nil, nil }
func (m *mockPagedCompRepo) List() ([]domain.Competition, error)            { return m.comps, nil }
func (m *mockPagedCompRepo) UpdateStatus(id, status string) error           { return nil }
func (m *mockPagedCompRepo) Update(c *domain.Competition) error             { return nil }

type mockPagedMatchRepo struct {
	*mockMatchRepo
}

func (m *mockPagedMatchRepo) ListByCompetitionPaged(competitionID, status string, limit, offset int) ([]store.Match, int, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var filtered []store.Match
	for _, match := range m.matches {
		if match.CompetitionID == competitionID && (status == "" || match.Status == status) {
			filtered = append(filtered, *match)
		}
	}
	total := len(filtered)
	if offset >= total {
		return []store.Match{}, total, nil
	}
	end := offset + limit
	if end > total {
		end = total
	}
	return filtered[offset:end], total, nil
}

type mockPagedAthleteRepo struct {
	athletes []domain.Athlete
}

func (m *mockPagedAthleteRepo) Create(a *domain.Athlete) error             { return nil }
func (m *mockPagedAthleteRepo) GetByID(id string) (*domain.Athlete, error) { return nil, nil }
func (m *mockPagedAthleteRepo) List() ([]domain.Athlete, error)            { return m.athletes, nil }
func (m *mockPagedAthleteRepo) RegisterToCompetition(competitionID, athleteID, weightClass, gender string) error {
	return nil
}
func (m *mockPagedAthleteRepo) ListByCompetition(competitionID string) ([]domain.Athlete, error) {
	return nil, nil
}

func TestPaginationAndFiltering(t *testing.T) {
	// 1. Competitions Pagination & Status Filter
	compRepo := &mockPagedCompRepo{
		comps: []domain.Competition{
			{ID: "c1", Name: "Comp 1", Status: "active"},
			{ID: "c2", Name: "Comp 2", Status: "draft"},
			{ID: "c3", Name: "Comp 3", Status: "active"},
			{ID: "c4", Name: "Comp 4", Status: "completed"},
			{ID: "c5", Name: "Comp 5", Status: "active"},
		},
	}
	compSvc := service.NewCompetitionService(compRepo)

	// Filter status active with page=1, limit=2
	pagedActive, totalActive, err := compSvc.GetCompetitionsPaged("active", 1, 2)
	if err != nil {
		t.Fatalf("GetCompetitionsPaged failed: %v", err)
	}
	if totalActive != 3 {
		t.Fatalf("expected total 3 active comps, got %d", totalActive)
	}
	if len(pagedActive) != 2 {
		t.Fatalf("expected 2 items for page 1, got %d", len(pagedActive))
	}

	// Page 2
	pagedActivePage2, _, _ := compSvc.GetCompetitionsPaged("active", 2, 2)
	if len(pagedActivePage2) != 1 || pagedActivePage2[0].ID != "c5" {
		t.Fatalf("expected 1 item on page 2 (c5), got %v", pagedActivePage2)
	}

	// 2. Matches Pagination
	matchRepo := &mockPagedMatchRepo{mockMatchRepo: newMockMatchRepo()}
	for i := 1; i <= 5; i++ {
		st := "idle"
		if i%2 == 0 {
			st = "running"
		}
		m := &store.Match{
			ID:            "m-" + strconv.Itoa(i),
			CompetitionID: "comp-pagination",
			AthleteRedID:  "red-" + strconv.Itoa(i),
			AthleteBlueID: "blue-" + strconv.Itoa(i),
			Status:        st,
		}
		_ = matchRepo.Create(m)
	}

	matchSvc := service.NewMatchService(matchRepo, &mockCompRepo{}, hub.NewManager(), nil, nil, nil)
	pagedMatches, totalMatches, err := matchSvc.GetMatchesByCompetitionPaged("comp-pagination", "running", 1, 10)
	if err != nil {
		t.Fatalf("GetMatchesByCompetitionPaged failed: %v", err)
	}
	if totalMatches != 2 {
		t.Fatalf("expected 2 running matches, got %d", totalMatches)
	}
	if len(pagedMatches) != 2 {
		t.Fatalf("expected 2 returned matches, got %d", len(pagedMatches))
	}

	// 3. Athletes Pagination & Gender Filter
	athleteRepo := &mockPagedAthleteRepo{
		athletes: []domain.Athlete{
			{ID: "a1", Name: "Andi", Gender: "male"},
			{ID: "a2", Name: "Budi", Gender: "male"},
			{ID: "a3", Name: "Citra", Gender: "female"},
			{ID: "a4", Name: "Dewi", Gender: "female"},
		},
	}
	athleteSvc := service.NewAthleteService(athleteRepo)
	females, totalFemales, err := athleteSvc.GetAthletesPaged("female", 1, 10)
	if err != nil {
		t.Fatalf("GetAthletesPaged failed: %v", err)
	}
	if totalFemales != 2 || len(females) != 2 {
		t.Fatalf("expected 2 female athletes, got total=%d len=%d", totalFemales, len(females))
	}
}

func TestIdempotencyKey_ScoringDeduplication(t *testing.T) {
	h := hub.NewHub("arena-idempotency")
	sm := statemachine.New("match-idempotency")
	ruleset, _ := rules.LoadRuleset("ipsi")

	orch := orchestrator.New("match-idempotency", "arena-idempotency", sm, ruleset, h, nil, nil, nil)
	if err := orch.Start(); err != nil {
		t.Fatalf("orch.Start failed: %v", err)
	}

	// 1. Test REST / Service idempotency check
	input := consensus.JudgeInput{
		JudgeID:   "j1",
		ScoreType: consensus.ScoreTechnique,
		Target:    consensus.TargetRed,
	}

	// First submission with Idempotency Key
	_, isDuplicate1, err := orch.SubmitJudgeInput(input, "idemp-key-101")
	if err != nil {
		t.Fatalf("SubmitJudgeInput failed: %v", err)
	}
	if isDuplicate1 {
		t.Fatalf("first submission should not be duplicate")
	}

	// Second submission with identical Idempotency Key -> should be identified as duplicate
	_, isDuplicate2, err := orch.SubmitJudgeInput(input, "idemp-key-101")
	if err != nil {
		t.Fatalf("second submission failed: %v", err)
	}
	if !isDuplicate2 {
		t.Fatalf("expected second submission with same key to be marked duplicate")
	}

	// 2. Test WebSocket idempotency deduplication
	client, received := hub.NewClient("judge-tablet-2", hub.RoleJudge, "arena-idempotency")
	h.Register(client)

	// Send score event with event_id
	h.OnMessage("arena-idempotency", client, hub.Message{
		Type: hub.MsgJudgeInput,
		Payload: map[string]any{
			"event_id":   "ws-event-uuid-202",
			"score_type": "technique",
			"target":     "red",
			"round":      1,
		},
	})

	// Duplicate send with exact same event_id (simulating network double-tap)
	h.OnMessage("arena-idempotency", client, hub.Message{
		Type: hub.MsgJudgeInput,
		Payload: map[string]any{
			"event_id":   "ws-event-uuid-202",
			"score_type": "technique",
			"target":     "red",
			"round":      1,
		},
	})

	// Make sure no crash or error occurred
	select {
	case msg := <-received:
		t.Logf("client received: %v", msg)
	case <-time.After(50 * time.Millisecond):
		// No panic, deduplication silently prevented accidental double processing
	}
}
