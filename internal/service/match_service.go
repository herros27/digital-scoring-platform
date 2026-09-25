package service

import (
	"errors"

	"github.com/herros27/digital-scoring/internal/hub"
	"github.com/herros27/digital-scoring/internal/orchestrator"
	"github.com/herros27/digital-scoring/internal/statemachine"
	"github.com/herros27/digital-scoring/internal/store"
	"github.com/herros27/digital-scoring/internal/rules"
)

type MatchRepository interface {
	Create(m *store.Match) error
	GetByID(id string) (*store.Match, error)
	UpdateStatus(id, status string) error
	SetWinner(matchID, winnerID string) error
}

type MatchService struct {
	matchRepo MatchRepository
	compRepo  CompetitionRepository
	hubManager *hub.Manager
	eventStore *store.MatchEventStore
	scoreStore *store.MatchScoreStore
	matchStore *store.MatchStore

	// activeOrchestrators menyimpan match yang sedang live di arena: map[arenaID]*Orchestrator
	activeOrchestrators map[string]*orchestrator.Orchestrator
}

func NewMatchService(
	matchRepo MatchRepository,
	compRepo CompetitionRepository,
	hubManager *hub.Manager,
	eventStore *store.MatchEventStore,
	scoreStore *store.MatchScoreStore,
	matchStore *store.MatchStore,
) *MatchService {
	return &MatchService{
		matchRepo:           matchRepo,
		compRepo:            compRepo,
		hubManager:          hubManager,
		eventStore:          eventStore,
		scoreStore:          scoreStore,
		matchStore:          matchStore,
		activeOrchestrators: make(map[string]*orchestrator.Orchestrator),
	}
}

func (s *MatchService) GetMatchByID(id string) (*store.Match, error) {
	if id == "" {
		return nil, errors.New("match id is required")
	}
	return s.matchRepo.GetByID(id)
}

func (s *MatchService) CreateMatch(m *store.Match) error {
	if m.CompetitionID == "" {
		return errors.New("competition id is required")
	}
	if m.AthleteRedID == "" || m.AthleteBlueID == "" {
		return errors.New("both athletes (red and blue) must be specified")
	}
	if m.AthleteRedID == m.AthleteBlueID {
		return errors.New("athlete red and blue cannot be the same person")
	}
	m.Status = "idle"
	m.CurrentRound = 1
	return s.matchRepo.Create(m)
}

// LoadMatchToArena menginisialisasi Orchestrator untuk pertandingan yang akan bertanding di gelanggang/arena
func (s *MatchService) LoadMatchToArena(matchID, arenaID, rulesetPath string) error {
	match, err := s.matchRepo.GetByID(matchID)
	if err != nil || match == nil {
		return errors.New("match not found")
	}
	if rulesetPath == "" {
		rulesetPath = "rulesets/ipsi_2024.json" // default
	}
	// 1. Load Ruleset
	ruleset, err := rules.LoadRuleset(rulesetPath)
	if err != nil {
		return errors.New("failed to load ruleset: " + err.Error())
	}
	// 2. Buat StateMachine baru
	sm := statemachine.New(matchID)
	// 3. Dapatkan Arena Hub
	arenaHub := s.hubManager.GetOrCreate(arenaID)
	// 4. Inisialisasi Orchestrator
	orch := orchestrator.New(
		matchID,
		arenaID,
		sm,
		ruleset,
		arenaHub,
		s.matchStore,
		s.eventStore,
		s.scoreStore,
	)
	s.activeOrchestrators[arenaID] = orch
	// 5. Update status match di DB menjadi 'ready'
	return s.matchRepo.UpdateStatus(matchID, "ready")
}