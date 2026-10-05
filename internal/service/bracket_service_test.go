package service_test

import (
	"testing"

	"github.com/herros27/digital-scoring/internal/service"
	"github.com/herros27/digital-scoring/internal/store"
)

type mockBracketRepo struct {
	slots map[string]store.BracketSlot
}

func (m *mockBracketRepo) CreateBatch(slots []store.BracketSlot) error {
	for _, s := range slots {
		m.slots[s.ID] = s
	}
	return nil
}

func (m *mockBracketRepo) GetByCompetitionCategory(competitionID, weightClass, gender string) ([]store.BracketSlot, error) {
	var res []store.BracketSlot
	for _, s := range m.slots {
		if s.CompetitionID == competitionID && s.WeightClass == weightClass && s.Gender == gender {
			res = append(res, s)
		}
	}
	return res, nil
}

func (m *mockBracketRepo) DeleteByCompetitionCategory(competitionID, weightClass, gender string) error {
	for id, s := range m.slots {
		if s.CompetitionID == competitionID && s.WeightClass == weightClass && s.Gender == gender {
			delete(m.slots, id)
		}
	}
	return nil
}

func (m *mockBracketRepo) GetByID(id string) (*store.BracketSlot, error) {
	if s, ok := m.slots[id]; ok {
		return &s, nil
	}
	return nil, nil
}

func (m *mockBracketRepo) UpdateWinner(slotID, winnerID string) error {
	if s, ok := m.slots[slotID]; ok {
		s.WinnerID = &winnerID
		m.slots[slotID] = s
	}
	return nil
}

func (m *mockBracketRepo) SetParticipant(slotID, participantID string) error {
	if s, ok := m.slots[slotID]; ok {
		s.ParticipantID = &participantID
		m.slots[slotID] = s
	}
	return nil
}

type mockMatchCreator struct {
	created []*store.Match
}

func (m *mockMatchCreator) Create(match *store.Match) error {
	m.created = append(m.created, match)
	return nil
}

type mockArenaLister struct {
	arenas []store.Arena
}

func (m *mockArenaLister) ListByCompetition(compID string) ([]store.Arena, error) {
	return m.arenas, nil
}

func TestGenerateBracket_AutoSyncMatches(t *testing.T) {
	bRepo := &mockBracketRepo{slots: make(map[string]store.BracketSlot)}
	mRepo := &mockMatchCreator{}
	aRepo := &mockArenaLister{
		arenas: []store.Arena{
			{ID: "arena-a", Name: "Gelanggang A"},
			{ID: "arena-b", Name: "Gelanggang B"},
		},
	}

	svc := service.NewBracketService(bRepo).WithMatchStore(mRepo).WithArenaStore(aRepo)

	participantIDs := []string{"p1", "p2", "p3", "p4"}
	slots, err := svc.GenerateBracket("comp-1", "A", "male", participantIDs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(slots) == 0 {
		t.Fatalf("expected generated slots, got 0")
	}

	// 4 participants = 2 matches in round 1
	if len(mRepo.created) != 2 {
		t.Fatalf("expected 2 matches created in round 1, got %d", len(mRepo.created))
	}

	m1 := mRepo.created[0]
	if m1.AthleteRedID != "p1" || m1.AthleteBlueID != "p2" {
		t.Errorf("match 1 athletes mismatch: red=%s blue=%s", m1.AthleteRedID, m1.AthleteBlueID)
	}
	if m1.CurrentRound != 1 || m1.Status != "idle" {
		t.Errorf("match 1 round/status mismatch: round=%d status=%s", m1.CurrentRound, m1.Status)
	}
	if m1.ArenaID != "arena-a" {
		t.Errorf("expected arena-a for match 1, got %s", m1.ArenaID)
	}

	m2 := mRepo.created[1]
	if m2.AthleteRedID != "p3" || m2.AthleteBlueID != "p4" {
		t.Errorf("match 2 athletes mismatch: red=%s blue=%s", m2.AthleteRedID, m2.AthleteBlueID)
	}
	if m2.ArenaID != "arena-b" {
		t.Errorf("expected arena-b for match 2, got %s", m2.ArenaID)
	}
}
