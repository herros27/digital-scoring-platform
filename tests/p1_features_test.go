package tests

import (
	"errors"
	"testing"
	"time"

	"github.com/herros27/digital-scoring/internal/domain"
	"github.com/herros27/digital-scoring/internal/hub"
	"github.com/herros27/digital-scoring/internal/orchestrator"
	"github.com/herros27/digital-scoring/internal/rules"
	"github.com/herros27/digital-scoring/internal/service"
	"github.com/herros27/digital-scoring/internal/statemachine"
	"github.com/herros27/digital-scoring/internal/store"
)

type mockBracketAdvancer struct {
	advanced map[string]string // slotID -> winnerID
}

func newMockBracketAdvancer() *mockBracketAdvancer {
	return &mockBracketAdvancer{advanced: make(map[string]string)}
}

func (b *mockBracketAdvancer) AdvanceWinner(slotID, winnerID string) error {
	b.advanced[slotID] = winnerID
	return nil
}

type mockRulesetRepo struct {
	rulesets []store.Ruleset
}

func (r *mockRulesetRepo) List() ([]store.Ruleset, error) {
	return r.rulesets, nil
}

func (r *mockRulesetRepo) GetByID(id string) (*store.Ruleset, error) {
	for _, rs := range r.rulesets {
		if rs.ID == id {
			return &rs, nil
		}
	}
	return nil, domain.ErrNotFound
}

func (r *mockRulesetRepo) Create(rs *store.Ruleset) error {
	r.rulesets = append(r.rulesets, *rs)
	return nil
}

func (r *mockRulesetRepo) Update(rs *store.Ruleset) error {
	for i, existing := range r.rulesets {
		if existing.ID == rs.ID {
			r.rulesets[i] = *rs
			return nil
		}
	}
	return domain.ErrNotFound
}

func (r *mockRulesetRepo) Delete(id string) error {
	for i, existing := range r.rulesets {
		if existing.ID == id {
			r.rulesets = append(r.rulesets[:i], r.rulesets[i+1:]...)
			return nil
		}
	}
	return domain.ErrNotFound
}

func (r *mockRulesetRepo) GetActive() (*store.Ruleset, error) {
	if len(r.rulesets) > 0 {
		return &r.rulesets[0], nil
	}
	return nil, domain.ErrNotFound
}

func TestAutomaticBracketAdvancement(t *testing.T) {
	matchRepo := newMockMatchRepo()
	compRepo := &mockCompRepo{}
	hubMgr := hub.NewManager()
	bracketAdvancer := newMockBracketAdvancer()

	matchRepo.onFinish = func(slotID, winnerID string) { _ = bracketAdvancer.AdvanceWinner(slotID, winnerID) }
	svc := service.NewMatchService(matchRepo, compRepo, hubMgr, nil, nil, nil)

	// Create match linked to a bracket slot
	match := &store.Match{
		ID:            "match-bracket-1",
		CompetitionID: "comp-1",
		ArenaID:       "arena-bracket",
		BracketSlotID: "slot-round1-A",
		AthleteRedID:  "athlete-101",
		AthleteBlueID: "athlete-102",
	}
	if err := svc.CreateMatch(match); err != nil {
		t.Fatalf("CreateMatch failed: %v", err)
	}

	if err := svc.LoadMatchToArena("match-bracket-1", "arena-bracket", "ipsi"); err != nil {
		t.Fatalf("LoadMatchToArena failed: %v", err)
	}

	if err := svc.StartMatch("match-bracket-1"); err != nil {
		t.Fatalf("StartMatch failed: %v", err)
	}

	// Skor seri: pertandingan tidak boleh berakhir dan pemenang tidak boleh maju ke bagan
	// sebelum wasit memutuskan
	if err := svc.FinishMatch("match-bracket-1"); err == nil {
		t.Fatal("FinishMatch pada skor seri harus ditolak")
	}
	if _, advanced := bracketAdvancer.advanced["slot-round1-A"]; advanced {
		t.Fatal("bagan tidak boleh maju sebelum ada pemenang")
	}

	// Keputusan wasit -> finalize, pemenang ditetapkan dan dimajukan di bagan
	if err := svc.FinishMatchWithWinner("match-bracket-1", "red"); err != nil {
		t.Fatalf("FinishMatchWithWinner failed: %v", err)
	}

	// Check if match status and winner were updated
	finishedMatch, err := matchRepo.GetByID("match-bracket-1")
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}
	if finishedMatch.WinnerID != "athlete-101" {
		t.Fatalf("expected winner 'athlete-101' (keputusan wasit: red), got %s", finishedMatch.WinnerID)
	}

	// Verify bracket advancer received the winner for slot-round1-A
	winnerInSlot, ok := bracketAdvancer.advanced["slot-round1-A"]
	if !ok {
		t.Fatalf("expected slot-round1-A to be advanced, but was not called")
	}
	if winnerInSlot != "athlete-101" {
		t.Fatalf("expected advanced winner 'athlete-101', got %s", winnerInSlot)
	}
}

func TestRulesetService_ListingAndRetrieval(t *testing.T) {
	// Case 1: Custom DB ruleset
	customRepo := &mockRulesetRepo{
		rulesets: []store.Ruleset{
			{
				ID:        "custom-ruleset-1",
				Name:      "IPSI Custom 2026",
				Version:   "2026.1",
				Status:    "active",
				CreatedAt: time.Now(),
			},
		},
	}
	customSvc := service.NewRulesetService(customRepo)
	list, err := customSvc.ListRulesets()
	if err != nil {
		t.Fatalf("ListRulesets failed: %v", err)
	}
	if len(list) != 1 || list[0].ID != "custom-ruleset-1" {
		t.Fatalf("expected 1 ruleset with ID 'custom-ruleset-1', got: %v", list)
	}

	rs, err := customSvc.GetRulesetByID("custom-ruleset-1")
	if err != nil {
		t.Fatalf("GetRulesetByID failed: %v", err)
	}
	if rs.Name != "IPSI Custom 2026" {
		t.Fatalf("expected 'IPSI Custom 2026', got: %s", rs.Name)
	}

	// Case 2: Fallback to built-ins when DB is empty
	emptySvc := service.NewRulesetService(&mockRulesetRepo{})
	builtins, err := emptySvc.ListRulesets()
	if err != nil {
		t.Fatalf("ListRulesets fallback failed: %v", err)
	}
	if len(builtins) < 2 {
		t.Fatalf("expected at least 2 built-in rulesets, got %d", len(builtins))
	}

	ipsi, err := emptySvc.GetRulesetByID("ipsi-2024")
	if err != nil {
		t.Fatalf("GetRulesetByID fallback failed: %v", err)
	}
	if ipsi.Name != "IPSI" {
		t.Fatalf("expected name IPSI, got: %s", ipsi.Name)
	}

	// Case 3: Not found
	_, errNotFound := emptySvc.GetRulesetByID("non-existent")
	if !errors.Is(errNotFound, domain.ErrNotFound) {
		t.Fatalf("expected domain.ErrNotFound, got: %v", errNotFound)
	}
}

func TestMatchEvents_QueryService(t *testing.T) {
	svc := service.NewMatchService(newMockMatchRepo(), &mockCompRepo{}, hub.NewManager(), nil, nil, nil)

	// Valid ID with nil store returns empty list gracefully
	events, err := svc.GetMatchEvents("match-1")
	if err != nil {
		t.Fatalf("GetMatchEvents failed: %v", err)
	}
	if len(events) != 0 {
		t.Fatalf("expected 0 events, got %d", len(events))
	}

	// Empty matchID returns invalid input error
	_, errInvalid := svc.GetMatchEvents("")
	if !errors.Is(errInvalid, domain.ErrInvalidInput) {
		t.Fatalf("expected domain.ErrInvalidInput, got: %v", errInvalid)
	}
}

func TestWebSocket_SyncRequest(t *testing.T) {
	h := hub.NewHub("arena-sync-test")
	sm := statemachine.New("match-sync")
	ruleset, _ := rules.LoadRuleset("ipsi")

	_ = orchestrator.New("match-sync", "arena-sync-test", sm, ruleset, h, nil, nil, nil)

	// Setup mock client using NewClient and register into hub
	client, received := hub.NewClient("judge-tablet-1", hub.RoleJudge, "arena-sync-test")
	h.Register(client)

	// Trigger sync_request via hub.OnMessage callback (registered by orchestrator)
	h.OnMessage("arena-sync-test", client, hub.Message{
		Type: hub.MsgSyncRequest,
	})

	select {
	case msg := <-received:
		if msg.Type != hub.MsgSyncResponse {
			t.Fatalf("expected MsgSyncResponse, got %s", msg.Type)
		}
		payload, ok := msg.Payload.(map[string]any)
		if !ok {
			t.Fatalf("expected map[string]any payload, got %T", msg.Payload)
		}
		if payload["match_id"] != "match-sync" {
			t.Fatalf("expected match_id 'match-sync', got %v", payload["match_id"])
		}
		if payload["arena_id"] != "arena-sync-test" {
			t.Fatalf("expected arena_id 'arena-sync-test', got %v", payload["arena_id"])
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("timed out waiting for sync_response")
	}
}
