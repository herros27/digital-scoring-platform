package service

import (
	"errors"

	"github.com/herros27/digital-scoring/internal/hub"
	"github.com/herros27/digital-scoring/internal/orchestrator"
	"github.com/herros27/digital-scoring/internal/rules"
	"github.com/herros27/digital-scoring/internal/statemachine"
	"github.com/herros27/digital-scoring/internal/store"
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

	// activeOrchestrators menyimpan match yang sedang live di arena: map[matchID]*Orchestrator
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

// LoadMatchToArena menginisialisasi Orchestrator untuk pertandingan yang akan bertanding di arena
func (s *MatchService) LoadMatchToArena(matchID, arenaID, rulesetPath string) error {
	if _, ok := s.activeOrchestrators[matchID]; ok {
		return errors.New("match already loaded to arena")
	}
	// Buat state machine dan ruleset baru
	sm := statemachine.New(matchID)
	r, err := rules.LoadRuleset(rulesetPath)
	if err != nil {
		return err
	}
	orch := orchestrator.New(matchID, arenaID, sm, r, s.hubManager.GetOrCreate(arenaID), s.matchStore, s.eventStore, s.scoreStore)
	s.activeOrchestrators[matchID] = orch
	return nil
}

// GetOrchestrator mengambil Orchestrator berdasarkan matchID
func (s *MatchService) GetOrchestrator(matchID string) (*orchestrator.Orchestrator, error) {
	o, ok := s.activeOrchestrators[matchID]
	if !ok {
		return nil, errors.New("match not loaded to arena")
	}
	return o, nil
}

func (s *MatchService) StartMatch(matchID string) error {
	o, err := s.GetOrchestrator(matchID)
	if err != nil {
		return err
	}
	o.Start()
	return nil
}

func (s *MatchService) PauseMatch(matchID string) error {
	o, err := s.GetOrchestrator(matchID)
	if err != nil {
		return err
	}
	o.Pause()
	return nil
}

func (s *MatchService) ResumeMatch(matchID string) error {
	o, err := s.GetOrchestrator(matchID)
	if err != nil {
		return err
	}
	o.Resume()
	return nil
}

func (s *MatchService) FinishMatch(matchID string) error {
	o, err := s.GetOrchestrator(matchID)
	if err != nil {
		return err
	}
	o.Finish()
	return nil
}