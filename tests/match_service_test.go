package tests

import (
	"errors"
	"testing"

	"github.com/herros27/digital-scoring/internal/domain"
	"github.com/herros27/digital-scoring/internal/hub"
	"github.com/herros27/digital-scoring/internal/service"
	"github.com/herros27/digital-scoring/internal/store"
)

type mockMatchRepo struct {
	matches map[string]*store.Match
}

func newMockMatchRepo() *mockMatchRepo {
	return &mockMatchRepo{matches: make(map[string]*store.Match)}
}

func (m *mockMatchRepo) Create(match *store.Match) error {
	m.matches[match.ID] = match
	return nil
}

func (m *mockMatchRepo) GetByID(id string) (*store.Match, error) {
	if match, ok := m.matches[id]; ok {
		return match, nil
	}
	return nil, domain.ErrNotFound
}

func (m *mockMatchRepo) UpdateStatus(id, status string) error {
	if match, ok := m.matches[id]; ok {
		match.Status = status
		return nil
	}
	return domain.ErrNotFound
}

func (m *mockMatchRepo) SetWinner(matchID, winnerID string) error {
	if match, ok := m.matches[matchID]; ok {
		match.WinnerID = winnerID
		return nil
	}
	return domain.ErrNotFound
}

type mockCompRepo struct{}

func (c *mockCompRepo) Create(comp *domain.Competition) error            { return nil }
func (c *mockCompRepo) GetByID(id string) (*domain.Competition, error)   { return &domain.Competition{ID: id}, nil }
func (c *mockCompRepo) List() ([]domain.Competition, error)              { return nil, nil }
func (c *mockCompRepo) UpdateStatus(id, status string) error             { return nil }

func TestMatchService_LifecycleAndArenaBusy(t *testing.T) {
	matchRepo := newMockMatchRepo()
	compRepo := &mockCompRepo{}
	hubManager := hub.NewManager()

	svc := service.NewMatchService(matchRepo, compRepo, hubManager, nil, nil, nil, nil)
	rulesetPath := "ipsi"

	// 1. Create Match
	m1 := &store.Match{
		ID:            "match-1",
		CompetitionID: "comp-1",
		AthleteRedID:  "athlete-red",
		AthleteBlueID: "athlete-blue",
	}
	if err := svc.CreateMatch(m1); err != nil {
		t.Fatalf("CreateMatch failed: %v", err)
	}

	// 2. Load match-1 to arena-1
	if err := svc.LoadMatchToArena("match-1", "arena-1", rulesetPath); err != nil {
		t.Fatalf("LoadMatchToArena failed: %v", err)
	}

	// 3. Test ARENA_BUSY validation: Loading match-2 to arena-1 while match-1 is still active
	m2 := &store.Match{
		ID:            "match-2",
		CompetitionID: "comp-1",
		AthleteRedID:  "athlete-red-2",
		AthleteBlueID: "athlete-blue-2",
	}
	_ = svc.CreateMatch(m2)

	errBusy := svc.LoadMatchToArena("match-2", "arena-1", rulesetPath)
	if errBusy == nil {
		t.Fatalf("expected ARENA_BUSY error when loading to occupied arena, got nil")
	}
	if !errors.Is(errBusy, domain.ErrArenaBusy) {
		t.Fatalf("expected domain.ErrArenaBusy, got: %v", errBusy)
	}

	// 4. Start match-1
	if err := svc.StartMatch("match-1"); err != nil {
		t.Fatalf("StartMatch failed: %v", err)
	}

	// 5. Pause match-1
	if err := svc.PauseMatch("match-1"); err != nil {
		t.Fatalf("PauseMatch failed: %v", err)
	}

	// 6. Resume match-1
	if err := svc.ResumeMatch("match-1"); err != nil {
		t.Fatalf("ResumeMatch failed: %v", err)
	}

	// 7. Finish match-1
	if err := svc.FinishMatch("match-1"); err != nil {
		t.Fatalf("FinishMatch failed: %v", err)
	}

	// 8. Now that match-1 is finished, arena-1 should be available for match-2
	if err := svc.LoadMatchToArena("match-2", "arena-1", rulesetPath); err != nil {
		t.Fatalf("expected arena-1 to be free after match-1 finished, but got error: %v", err)
	}

	// 9. Test CancelMatch on match-2
	if err := svc.CancelMatch("match-2"); err != nil {
		t.Fatalf("CancelMatch failed: %v", err)
	}

	// 10. Test GetMatchScores on match-1
	scoresDetail, err := svc.GetMatchScores("match-1")
	if err != nil {
		t.Fatalf("GetMatchScores failed: %v", err)
	}
	if scoresDetail.MatchID != "match-1" {
		t.Fatalf("expected MatchID 'match-1', got %s", scoresDetail.MatchID)
	}
}
