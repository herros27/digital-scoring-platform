package tests

import (
	"testing"

	"github.com/herros27/digital-scoring/internal/domain"
	"github.com/herros27/digital-scoring/internal/service"
	"github.com/herros27/digital-scoring/internal/store"
)

type mockArenaRepo struct {
	arenas map[string]*store.Arena
}

func newMockArenaRepo() *mockArenaRepo {
	return &mockArenaRepo{arenas: make(map[string]*store.Arena)}
}

func (m *mockArenaRepo) Create(a *store.Arena) error {
	m.arenas[a.ID] = a
	return nil
}

func (m *mockArenaRepo) GetByID(id string) (*store.Arena, error) {
	if a, ok := m.arenas[id]; ok {
		return a, nil
	}
	return nil, nil
}

func (m *mockArenaRepo) ListByCompetition(compID string) ([]store.Arena, error) {
	var list []store.Arena
	for _, a := range m.arenas {
		if a.CompetitionID == compID {
			list = append(list, *a)
		}
	}
	return list, nil
}

func (m *mockArenaRepo) Update(a *store.Arena) error {
	m.arenas[a.ID] = a
	return nil
}

func (m *mockArenaRepo) Delete(id string) error {
	delete(m.arenas, id)
	return nil
}

func TestArenaService_CRUD(t *testing.T) {
	arenaRepo := newMockArenaRepo()
	compRepo := &mockCompRepo{}
	svc := service.NewArenaService(arenaRepo, compRepo)

	// 1. Create Arena
	created, err := svc.CreateArena("comp-1", "Gelanggang A")
	if err != nil {
		t.Fatalf("CreateArena failed: %v", err)
	}
	created.ID = "arena-1"
	_ = arenaRepo.Create(created)

	// 2. Get Arena By ID
	fetched, err := svc.GetArenaByID("arena-1")
	if err != nil {
		t.Fatalf("GetArenaByID failed: %v", err)
	}
	if fetched.Name != "Gelanggang A" {
		t.Errorf("expected Gelanggang A, got %s", fetched.Name)
	}

	// 3. Update Arena Name
	updated, err := svc.UpdateArena("arena-1", "Gelanggang Utama")
	if err != nil {
		t.Fatalf("UpdateArena failed: %v", err)
	}
	if updated.Name != "Gelanggang Utama" {
		t.Errorf("expected Gelanggang Utama, got %s", updated.Name)
	}

	// 4. Delete Arena
	if err := svc.DeleteArena("arena-1"); err != nil {
		t.Fatalf("DeleteArena failed: %v", err)
	}

	// 5. Verify deleted returns ErrNotFound
	_, errNotFound := svc.GetArenaByID("arena-1")
	if errNotFound != domain.ErrNotFound {
		t.Errorf("expected domain.ErrNotFound after delete, got %v", errNotFound)
	}
}

func TestCompetitionLifecycle_AndUpdates(t *testing.T) {
	compMap := make(map[string]*domain.Competition)
	compRepo := &mockLifecycleCompRepo{comps: compMap}
	svc := service.NewCompetitionService(compRepo)

	c1 := &domain.Competition{
		ID:          "comp-10",
		Name:        "Piala Presiden 2026",
		Location:    "Jakarta",
		StartDate:   "2026-10-01",
		EndDate:     "2026-10-05",
		RulesetName: "ipsi",
		Status:      "draft",
	}
	_ = compRepo.Create(c1)

	// 1. Update Details
	updateReq := &domain.Competition{
		Name:     "Piala Presiden Edisi Khusus 2026",
		Location: "Bandung",
	}
	updated, err := svc.UpdateCompetition("comp-10", updateReq)
	if err != nil {
		t.Fatalf("UpdateCompetition failed: %v", err)
	}
	if updated.Name != "Piala Presiden Edisi Khusus 2026" || updated.Location != "Bandung" {
		t.Errorf("unexpected updated fields: %+v", updated)
	}

	// 2. Activate
	if err := svc.ActivateCompetition("comp-10"); err != nil {
		t.Fatalf("ActivateCompetition failed: %v", err)
	}
	if compMap["comp-10"].Status != "active" {
		t.Errorf("expected status active, got %s", compMap["comp-10"].Status)
	}

	// 3. Pause
	if err := svc.PauseCompetition("comp-10"); err != nil {
		t.Fatalf("PauseCompetition failed: %v", err)
	}
	if compMap["comp-10"].Status != "paused" {
		t.Errorf("expected status paused, got %s", compMap["comp-10"].Status)
	}

	// 4. Finish
	if err := svc.FinishCompetition("comp-10"); err != nil {
		t.Fatalf("FinishCompetition failed: %v", err)
	}
	if compMap["comp-10"].Status != "completed" {
		t.Errorf("expected status completed, got %s", compMap["comp-10"].Status)
	}

	// 5. Cancel
	if err := svc.CancelCompetition("comp-10"); err != nil {
		t.Fatalf("CancelCompetition failed: %v", err)
	}
	if compMap["comp-10"].Status != "cancelled" {
		t.Errorf("expected status cancelled, got %s", compMap["comp-10"].Status)
	}
}

type mockLifecycleCompRepo struct {
	comps map[string]*domain.Competition
}

func (m *mockLifecycleCompRepo) Create(c *domain.Competition) error {
	m.comps[c.ID] = c
	return nil
}

func (m *mockLifecycleCompRepo) GetByID(id string) (*domain.Competition, error) {
	if c, ok := m.comps[id]; ok {
		return c, nil
	}
	return nil, nil
}

func (m *mockLifecycleCompRepo) List() ([]domain.Competition, error) {
	var list []domain.Competition
	for _, c := range m.comps {
		list = append(list, *c)
	}
	return list, nil
}

func (m *mockLifecycleCompRepo) UpdateStatus(id, status string) error {
	if c, ok := m.comps[id]; ok {
		c.Status = status
		return nil
	}
	return domain.ErrNotFound
}

func (m *mockLifecycleCompRepo) Update(c *domain.Competition) error {
	m.comps[c.ID] = c
	return nil
}
