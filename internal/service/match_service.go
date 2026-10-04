package service

import (
	"errors"
	"sync"

	"github.com/herros27/digital-scoring/internal/consensus"
	"github.com/herros27/digital-scoring/internal/domain"
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

type RulesetResolver interface {
	ResolveRuleset(identifier string) (*rules.Ruleset, error)
}

type MatchService struct {
	mu              sync.RWMutex
	matchRepo       MatchRepository
	compRepo        CompetitionRepository
	hubManager      *hub.Manager
	eventStore      *store.MatchEventStore
	scoreStore      *store.MatchScoreStore
	matchStore      *store.MatchStore
	bracketStore    orchestrator.BracketAdvancer
	rulesetResolver RulesetResolver

	// activeOrchestrators menyimpan match yang sedang live di arena: map[matchID]*Orchestrator
	activeOrchestrators map[string]*orchestrator.Orchestrator
}

func (s *MatchService) SetRulesetResolver(resolver RulesetResolver) {
	s.rulesetResolver = resolver
}

func NewMatchService(
	matchRepo MatchRepository,
	compRepo CompetitionRepository,
	hubManager *hub.Manager,
	eventStore *store.MatchEventStore,
	scoreStore *store.MatchScoreStore,
	matchStore *store.MatchStore,
	bracketStore orchestrator.BracketAdvancer,
) *MatchService {
	return &MatchService{
		matchRepo:           matchRepo,
		compRepo:            compRepo,
		hubManager:          hubManager,
		eventStore:          eventStore,
		scoreStore:          scoreStore,
		matchStore:          matchStore,
		bracketStore:        bracketStore,
		activeOrchestrators: make(map[string]*orchestrator.Orchestrator),
	}
}

func (s *MatchService) GetMatchByID(id string) (*store.Match, error) {
	if id == "" {
		return nil, domain.ErrInvalidInput
	}
	return s.matchRepo.GetByID(id)
}

type MatchPagedRepository interface {
	ListByCompetitionPaged(competitionID, status string, limit, offset int) ([]store.Match, int, error)
}

func (s *MatchService) GetMatchesByCompetitionPaged(competitionID, status string, page, limit int) ([]store.Match, int, error) {
	if competitionID == "" {
		return nil, 0, domain.ErrInvalidInput
	}
	if page <= 0 {
		page = 1
	}
	if limit <= 0 {
		limit = 20
	}
	offset := (page - 1) * limit

	if pagedRepo, ok := s.matchRepo.(MatchPagedRepository); ok {
		return pagedRepo.ListByCompetitionPaged(competitionID, status, limit, offset)
	}
	if s.matchStore != nil {
		return s.matchStore.ListByCompetitionPaged(competitionID, status, limit, offset)
	}

	return []store.Match{}, 0, nil
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
	s.mu.Lock()
	defer s.mu.Unlock()

	// 1. Cek apakah match sudah aktif di arena
	if existing, ok := s.activeOrchestrators[matchID]; ok && existing.IsActive() {
		return errors.New("match already loaded to arena")
	}

	// 2. Cek apakah arena sedang digunakan oleh match aktif lain (ARENA_BUSY)
	for _, orch := range s.activeOrchestrators {
		if orch.GetArenaID() == arenaID && orch.IsActive() {
			return domain.ErrArenaBusy
		}
	}

	// 3. Buat state machine dan ruleset baru
	sm := statemachine.New(matchID)

	// Jika rulesetPath kosong, coba ambil ruleset_id dari competition
	if rulesetPath == "" && s.matchRepo != nil && s.compRepo != nil {
		if m, errMatch := s.matchRepo.GetByID(matchID); errMatch == nil && m != nil {
			if comp, errComp := s.compRepo.GetByID(m.CompetitionID); errComp == nil && comp != nil {
				if comp.RulesetID != nil && *comp.RulesetID != "" {
					rulesetPath = *comp.RulesetID
				} else if comp.RulesetName != "" {
					rulesetPath = comp.RulesetName
				}
			}
		}
	}

	var r *rules.Ruleset
	var err error
	if s.rulesetResolver != nil {
		r, err = s.rulesetResolver.ResolveRuleset(rulesetPath)
	} else {
		r, err = rules.LoadRuleset(rulesetPath)
	}
	if err != nil {
		return err
	}

	var mStore orchestrator.MatchStore
	if s.matchStore != nil {
		mStore = s.matchStore
	} else if s.matchRepo != nil {
		mStore = s.matchRepo
	}

	orch := orchestrator.New(matchID, arenaID, sm, r, s.hubManager.GetOrCreate(arenaID), mStore, s.eventStore, s.scoreStore, s.bracketStore)
	s.activeOrchestrators[matchID] = orch
	return nil
}

// GetOrchestrator mengambil Orchestrator berdasarkan matchID
func (s *MatchService) GetOrchestrator(matchID string) (*orchestrator.Orchestrator, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

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
	return o.Start()
}

func (s *MatchService) PauseMatch(matchID string) error {
	o, err := s.GetOrchestrator(matchID)
	if err != nil {
		return err
	}
	return o.Pause()
}

func (s *MatchService) ResumeMatch(matchID string) error {
	o, err := s.GetOrchestrator(matchID)
	if err != nil {
		return err
	}
	return o.Resume()
}

func (s *MatchService) FinishMatch(matchID string) error {
	o, err := s.GetOrchestrator(matchID)
	if err != nil {
		return err
	}
	return o.Finish()
}

func (s *MatchService) CancelMatch(matchID string) error {
	o, err := s.GetOrchestrator(matchID)
	if err != nil {
		return err
	}
	return o.Cancel()
}

func (s *MatchService) GetMatchScores(matchID string) (*domain.MatchScoreDetail, error) {
	if matchID == "" {
		return nil, domain.ErrInvalidInput
	}

	var scores []domain.MatchScore
	if s.scoreStore != nil {
		var err error
		scores, err = s.scoreStore.GetByMatch(matchID)
		if err != nil {
			return nil, err
		}
	}

	totalRed := 0
	totalBlue := 0
	for _, sc := range scores {
		totalRed += sc.AthleteRedScore - sc.RedPenalties
		totalBlue += sc.AthleteBlueScore - sc.BluePenalties
	}

	if scores == nil {
		scores = []domain.MatchScore{}
	}

	return &domain.MatchScoreDetail{
		MatchID:   matchID,
		TotalRed:  totalRed,
		TotalBlue: totalBlue,
		Rounds:    scores,
		Consensus: &domain.ConsensusSummary{
			RedScore:  totalRed,
			BlueScore: totalBlue,
		},
		Judges: []domain.JudgeScoreSummary{},
	}, nil
}

func (s *MatchService) GetMatchEvents(matchID string) ([]domain.MatchEvent, error) {
	if matchID == "" {
		return nil, domain.ErrInvalidInput
	}
	if s.eventStore == nil {
		return []domain.MatchEvent{}, nil
	}
	events, err := s.eventStore.GetByMatch(matchID)
	if err != nil {
		return nil, err
	}
	if events == nil {
		events = []domain.MatchEvent{}
	}
	return events, nil
}

func (s *MatchService) SubmitScoreInput(matchID string, input consensus.JudgeInput, idempotencyKey string) (*consensus.ConsensusResult, bool, error) {
	o, err := s.GetOrchestrator(matchID)
	if err != nil {
		return nil, false, err
	}
	return o.SubmitJudgeInput(input, idempotencyKey)
}
