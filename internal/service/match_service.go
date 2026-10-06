package service

import (
	"errors"
	"fmt"
	"sort"
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
	arenaStore      *store.ArenaStore

	// activeOrchestrators menyimpan match yang sedang live di arena: map[matchID]*Orchestrator
	activeOrchestrators map[string]*orchestrator.Orchestrator
}

// ArenaName mengembalikan nama gelanggang, atau "" bila tidak diketahui.
func (s *MatchService) ArenaName(arenaID string) string {
	if s.arenaStore == nil {
		return ""
	}
	if a, err := s.arenaStore.GetByID(arenaID); err == nil {
		return a.Name
	}
	return ""
}

// SetArenaStore memungkinkan daftar scoreboard publik menampilkan nama gelanggang.
func (s *MatchService) SetArenaStore(a *store.ArenaStore) { s.arenaStore = a }

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

// GetMatchesByArenaPaged daftar partai pada satu gelanggang (urutan antrean).
func (s *MatchService) GetMatchesByArenaPaged(competitionID, arenaID, status string, page, limit int) ([]store.Match, int, error) {
	if competitionID == "" || arenaID == "" {
		return nil, 0, domain.ErrInvalidInput
	}
	if page <= 0 {
		page = 1
	}
	if limit <= 0 {
		limit = 20
	}
	if s.matchStore != nil {
		return s.matchStore.ListByArenaPaged(competitionID, arenaID, status, limit, (page-1)*limit)
	}

	// Tanpa store (mis. repositori tiruan): saring dari daftar kompetisi
	all, _, err := s.GetMatchesByCompetitionPaged(competitionID, status, 1, 10000)
	if err != nil {
		return nil, 0, err
	}
	filtered := make([]store.Match, 0, len(all))
	for _, m := range all {
		if m.ArenaID == arenaID {
			filtered = append(filtered, m)
		}
	}
	total := len(filtered)
	start := (page - 1) * limit
	if start > total {
		start = total
	}
	end := start + limit
	if end > total {
		end = total
	}
	return filtered[start:end], total, nil
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
	return s.FinishMatchWithWinner(matchID, "")
}

// FinishMatchWithWinner mengakhiri pertandingan; winner ("red"/"blue") = keputusan wasit, wajib bila skor seri.
func (s *MatchService) FinishMatchWithWinner(matchID, winner string) error {
	return s.FinishMatchWithDecision(matchID, winner, "")
}

// FinishMatchWithDecision seperti FinishMatchWithWinner dengan dasar keputusan (method): salah satu
// metode manual seri pada ruleset (extra_round, lighter_weight, coin_toss); kosong = keputusan wasit.
func (s *MatchService) FinishMatchWithDecision(matchID, winner, method string) error {
	o, err := s.GetOrchestrator(matchID)
	if err != nil {
		return err
	}
	if winner == "" {
		return o.Finish()
	}
	return o.FinishWithDecision(winner, method)
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

// GetLiveSnapshot mengembalikan keadaan pertandingan yang sedang aktif di sebuah arena
// untuk layar scoreboard publik. ok=false bila tidak ada pertandingan aktif.
func (s *MatchService) GetLiveSnapshot(arenaID string) (snapshot map[string]any, ok bool) {
	s.mu.RLock()
	var live *orchestrator.Orchestrator
	for _, orch := range s.activeOrchestrators {
		if orch.GetArenaID() == arenaID && orch.IsActive() {
			live = orch
			break
		}
	}
	s.mu.RUnlock()
	if live == nil {
		return nil, false
	}

	snapshot = live.Snapshot()
	snapshot["status"] = snapshot["state"]
	snapshot["arena_name"] = s.ArenaName(arenaID)
	if s.matchStore != nil {
		if red, blue, err := s.matchStore.GetCornerNames(live.GetMatchID()); err == nil {
			snapshot["red_athlete"] = red
			snapshot["blue_athlete"] = blue
		}
	}
	return snapshot, true
}

// ListLiveArenas mengembalikan ringkasan semua gelanggang yang sedang punya pertandingan
// aktif (belum selesai/dibatalkan), diurutkan menurut nama gelanggang.
func (s *MatchService) ListLiveArenas() []map[string]any {
	s.mu.RLock()
	live := make([]*orchestrator.Orchestrator, 0, len(s.activeOrchestrators))
	for _, orch := range s.activeOrchestrators {
		if orch.IsActive() {
			live = append(live, orch)
		}
	}
	s.mu.RUnlock()

	list := make([]map[string]any, 0, len(live))
	for _, orch := range live {
		snap := orch.Snapshot()
		item := map[string]any{
			"arena_id":   orch.GetArenaID(),
			"arena_name": s.ArenaName(orch.GetArenaID()),
			"match_id":   orch.GetMatchID(),
			"state":      snap["state"],
			"round":      snap["round"],
			"red_total":  snap["red_total"],
			"blue_total": snap["blue_total"],
		}
		if s.matchStore != nil {
			if red, blue, err := s.matchStore.GetCornerNames(orch.GetMatchID()); err == nil {
				item["red_athlete"] = red
				item["blue_athlete"] = blue
			}
		}
		list = append(list, item)
	}
	sort.Slice(list, func(i, j int) bool {
		ni, _ := list[i]["arena_name"].(string)
		nj, _ := list[j]["arena_name"].(string)
		if ni != nj {
			return ni < nj
		}
		ai, _ := list[i]["arena_id"].(string)
		aj, _ := list[j]["arena_id"].(string)
		return ai < aj
	})
	return list
}

// RescheduleMatch mengulang partai yang dibatalkan dari 0-0. Hanya partai berstatus cancelled
// yang dapat dijadwalkan ulang; skor lama dihapus dan log lama disembunyikan (tetap tersimpan).
func (s *MatchService) RescheduleMatch(matchID string) error {
	if matchID == "" || s.matchStore == nil {
		return domain.ErrInvalidInput
	}
	s.mu.RLock()
	existing, ok := s.activeOrchestrators[matchID]
	s.mu.RUnlock()
	if ok && existing.IsActive() {
		return fmt.Errorf("%w: partai masih aktif di gelanggang", domain.ErrInvalidStateTransition)
	}
	if m, err := s.matchStore.GetByID(matchID); err != nil || m == nil {
		return domain.ErrMatchNotFound
	}
	changed, err := s.matchStore.Reschedule(matchID)
	if err != nil {
		return err
	}
	if !changed {
		return fmt.Errorf("%w: hanya partai yang dibatalkan yang dapat dijadwalkan ulang", domain.ErrInvalidStateTransition)
	}
	s.mu.Lock()
	if old, ok := s.activeOrchestrators[matchID]; ok {
		old.Release()
		delete(s.activeOrchestrators, matchID)
	}
	s.mu.Unlock()
	if s.eventStore != nil {
		_ = s.eventStore.Append(&store.MatchEvent{MatchID: matchID, EventType: "match_rescheduled", Payload: "{}"})
	}
	return nil
}
