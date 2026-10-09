package service

import (
	"errors"
	"fmt"
	"log"
	"sort"
	"strings"
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
	mu               sync.RWMutex
	matchRepo        MatchRepository
	compRepo         CompetitionRepository
	hubManager       *hub.Manager
	eventStore       *store.MatchEventStore
	scoreStore       *store.MatchScoreStore
	matchStore       *store.MatchStore
	bracketStore     orchestrator.BracketAdvancer
	rulesetResolver  RulesetResolver
	categoryRulesets CategoryRulesetLookup
	arenaStore       *store.ArenaStore
	runtimeStore     *store.MatchRuntimeStore
	// participantLookup opsional: memvalidasi peserta saat partai dibuat manual (ada, sekejuaraan, terverifikasi).
	participantLookup func(id string) (*domain.Participant, error)

	// activeOrchestrators menyimpan match yang sedang live di arena: map[matchID]*Orchestrator
	activeOrchestrators map[string]*orchestrator.Orchestrator
}

// classLabel label kelas pertandingan, mis. "Kelas A Putri" (sama dengan konsol wasit).
func classLabel(weightClass, gender string) string {
	g := gender
	switch strings.ToLower(gender) {
	case "male", "putra":
		g = "Putra"
	case "female", "putri":
		g = "Putri"
	}
	return strings.TrimSpace(fmt.Sprintf("Kelas %s %s", weightClass, g))
}

// ArenaContext nama gelanggang dan nama kejuaraannya (untuk layar scoreboard saat belum ada partai).
func (s *MatchService) ArenaContext(arenaID string) (arenaName, competitionName string) {
	if s.arenaStore == nil {
		return "", ""
	}
	a, err := s.arenaStore.GetByID(arenaID)
	if err != nil || a == nil {
		return "", ""
	}
	if s.compRepo != nil {
		if c, err := s.compRepo.GetByID(a.CompetitionID); err == nil && c != nil {
			competitionName = c.Name
		}
	}
	return a.Name, competitionName
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

// SetParticipantLookup mengaktifkan validasi peserta pada pembuatan partai manual.
func (s *MatchService) SetParticipantLookup(f func(id string) (*domain.Participant, error)) {
	s.participantLookup = f
}

// CategoryRulesetLookup mencari ruleset khusus kategori dari slot bagan sebuah partai.
type CategoryRulesetLookup interface {
	RulesetIDForSlot(slotID string) (string, error)
}

// SetCategoryRulesetLookup mengaktifkan ruleset khusus per kategori (menimpa ruleset kejuaraan).
func (s *MatchService) SetCategoryRulesetLookup(l CategoryRulesetLookup) {
	s.categoryRulesets = l
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
		return fmt.Errorf("%w: peserta merah dan biru tidak boleh sama", domain.ErrInvalidInput)
	}
	if s.participantLookup != nil {
		for corner, id := range map[string]string{"merah": m.AthleteRedID, "biru": m.AthleteBlueID} {
			p, err := s.participantLookup(id)
			if err != nil || p == nil || p.CompetitionID != m.CompetitionID {
				return fmt.Errorf("%w: peserta sudut %s tidak ditemukan pada kejuaraan ini", domain.ErrInvalidInput, corner)
			}
			if p.RegistrationStatus != "verified" {
				return fmt.Errorf("%w: peserta sudut %s (%s) belum terverifikasi", domain.ErrInvalidInput, corner, p.Name)
			}
		}
	}
	if s.arenaStore != nil && m.ArenaID != "" {
		a, err := s.arenaStore.GetByID(m.ArenaID)
		if err != nil || a == nil || a.CompetitionID != m.CompetitionID {
			return fmt.Errorf("%w: gelanggang tidak ditemukan pada kejuaraan ini", domain.ErrInvalidInput)
		}
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
			// Prioritas: ruleset khusus kategori (mis. durasi Usia Dini), lalu ruleset kejuaraan.
			if m.BracketSlotID != "" && s.categoryRulesets != nil {
				if rid, errCat := s.categoryRulesets.RulesetIDForSlot(m.BracketSlotID); errCat == nil && rid != "" {
					rulesetPath = rid
				}
			}
			if rulesetPath == "" {
				if comp, errComp := s.compRepo.GetByID(m.CompetitionID); errComp == nil && comp != nil {
					if comp.RulesetID != nil && *comp.RulesetID != "" {
						rulesetPath = *comp.RulesetID
					} else if comp.RulesetName != "" {
						rulesetPath = comp.RulesetName
					}
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
	if s.runtimeStore != nil {
		orch.SetRuntimeStore(s.runtimeStore)
	}
	s.activeOrchestrators[matchID] = orch
	if s.matchStore != nil {
		if err := s.matchStore.SetArena(matchID, arenaID); err != nil {
			log.Printf("[match] gagal mencatat gelanggang partai %s: %v", matchID, err)
		}
	}
	// beri tahu layar yang sudah tersambung (scoreboard, tablet, konsol) bahwa partai baru dimuat
	orch.BroadcastSnapshot()
	return nil
}

// SetRuntimeStore memasang penyimpanan keadaan partai aktif untuk pemulihan setelah server restart.
func (s *MatchService) SetRuntimeStore(rs *store.MatchRuntimeStore) { s.runtimeStore = rs }

// RestoreActiveMatches memulihkan partai yang masih berlangsung saat server terakhir berhenti.
// Mengembalikan jumlah partai yang dipulihkan. Baris untuk partai yang sudah selesai/dibatalkan atau
// rusak dibersihkan. Ronde yang berjalan dipulihkan dalam keadaan dijeda (lihat orchestrator.Restore).
func (s *MatchService) RestoreActiveMatches() (int, error) {
	if s.runtimeStore == nil {
		return 0, nil
	}
	records, err := s.runtimeStore.List()
	if err != nil {
		return 0, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	restored := 0
	for _, rec := range records {
		if _, exists := s.activeOrchestrators[rec.MatchID]; exists {
			continue
		}
		if s.matchStore != nil {
			if m, err := s.matchStore.GetByID(rec.MatchID); err != nil || m == nil || m.Status == "finished" || m.Status == "cancelled" {
				_ = s.runtimeStore.Delete(rec.MatchID)
				continue
			}
		}
		var mStore orchestrator.MatchStore
		if s.matchStore != nil {
			mStore = s.matchStore
		} else if s.matchRepo != nil {
			mStore = s.matchRepo
		}
		orch, err := orchestrator.Restore(rec, s.hubManager.GetOrCreate(rec.ArenaID), mStore, s.eventStore, s.scoreStore, s.bracketStore, s.runtimeStore)
		if err != nil {
			log.Printf("[match] gagal memulihkan partai %s: %v", rec.MatchID, err)
			_ = s.runtimeStore.Delete(rec.MatchID)
			continue
		}
		s.activeOrchestrators[rec.MatchID] = orch
		restored++
	}
	return restored, nil
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

// ArenaForMatch gelanggang tempat partai berada: gelanggang orchestrator bila sedang dimuat, selain itu
// gelanggang pada data partai. Kosong bila partai tidak diketahui. Dipakai untuk membatasi token PIN
// (terikat satu gelanggang) agar tidak mengontrol partai di gelanggang lain.
func (s *MatchService) ArenaForMatch(matchID string) string {
	s.mu.RLock()
	o, ok := s.activeOrchestrators[matchID]
	s.mu.RUnlock()
	if ok {
		return o.GetArenaID()
	}
	if s.matchRepo != nil {
		if m, err := s.matchRepo.GetByID(matchID); err == nil && m != nil {
			return m.ArenaID
		}
	}
	return ""
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

// DeclareOutcome menyatakan hasil khusus (ko, tko, walkover, withdrawal) dengan winner = "red"/"blue".
func (s *MatchService) DeclareOutcome(matchID, outcome, winner, note string) error {
	o, err := s.GetOrchestrator(matchID)
	if err != nil {
		return err
	}
	return o.DeclareOutcome(outcome, winner, note)
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
		if info, err := s.matchStore.GetMatchInfo(live.GetMatchID()); err == nil {
			snapshot["red_athlete"] = info.RedName
			snapshot["blue_athlete"] = info.BlueName
			snapshot["red_team"] = info.RedTeam
			snapshot["blue_team"] = info.BlueTeam
			snapshot["class_label"] = classLabel(info.WeightClass, info.Gender)
			snapshot["competition_name"] = info.CompetitionName
			snapshot["match_number"] = info.Number
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
	if s.runtimeStore != nil {
		_ = s.runtimeStore.Delete(matchID)
	}
	if s.eventStore != nil {
		_ = s.eventStore.Append(&store.MatchEvent{MatchID: matchID, EventType: "match_rescheduled", Payload: "{}"})
	}
	return nil
}

// UpcomingPerArena jumlah partai menunggu yang ditampilkan per gelanggang pada jadwal publik.
const UpcomingPerArena = 3

// ListSchedule menyusun jadwal publik: kejuaraan aktif, gelanggangnya, dan beberapa partai
// berikutnya per gelanggang. Partai yang sudah dimuat ke gelanggang (tampil di daftar live)
// tidak diulang di jadwal.
func (s *MatchService) ListSchedule() ([]map[string]any, error) {
	if s.matchStore == nil {
		return []map[string]any{}, nil
	}
	rows, err := s.matchStore.ListUpcoming()
	if err != nil {
		return nil, err
	}
	s.mu.RLock()
	loaded := make(map[string]bool, len(s.activeOrchestrators))
	for id, orch := range s.activeOrchestrators {
		if orch.IsActive() {
			loaded[id] = true
		}
	}
	s.mu.RUnlock()

	result := []map[string]any{}
	var comp, arena map[string]any
	var compID, arenaID string
	for _, r := range rows {
		if comp == nil || r.CompetitionID != compID {
			compID, arenaID = r.CompetitionID, ""
			comp = map[string]any{
				"competition_id": r.CompetitionID, "name": r.CompetitionName, "location": r.Location,
				"start_date": r.StartDate, "end_date": r.EndDate, "arenas": []map[string]any{},
			}
			result = append(result, comp)
		}
		if arena == nil || r.ArenaID != arenaID {
			arenaID = r.ArenaID
			arena = map[string]any{"arena_id": r.ArenaID, "arena_name": r.ArenaName, "upcoming": []map[string]any{}}
			comp["arenas"] = append(comp["arenas"].([]map[string]any), arena)
		}
		if r.MatchID == "" || loaded[r.MatchID] {
			continue
		}
		up := arena["upcoming"].([]map[string]any)
		if len(up) >= UpcomingPerArena {
			continue
		}
		arena["upcoming"] = append(up, map[string]any{
			"match_id": r.MatchID, "match_number": r.Number,
			"red_athlete": r.RedName, "blue_athlete": r.BlueName,
			"red_team": r.RedTeam, "blue_team": r.BlueTeam,
			"class_label": classLabel(r.WeightClass, r.Gender),
		})
	}
	return result, nil
}
