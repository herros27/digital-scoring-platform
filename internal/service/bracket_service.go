package service

import (
	"errors"
	"sort"

	"github.com/herros27/digital-scoring/internal/bracket"
	"github.com/herros27/digital-scoring/internal/domain"
	"github.com/herros27/digital-scoring/internal/store"
)

type BracketRepository interface {
	CreateBatch(slots []store.BracketSlot) error
	GetByCompetitionCategory(competitionID, weightClass, gender string) ([]store.BracketSlot, error)
	DeleteByCompetitionCategory(competitionID, weightClass, gender string) error
	GetByID(id string) (*store.BracketSlot, error)
}

// CategoryBracketRepository adalah kemampuan opsional repo bagan untuk isolasi per category_id.
// BracketProgressChecker kemampuan opsional repo bagan untuk mengetahui apakah bagan kategori sudah dimainkan.
type BracketProgressChecker interface {
	HasStartedMatches(competitionID, categoryID string) (bool, error)
}

type CategoryBracketRepository interface {
	GetByCategoryID(competitionID, categoryID string) ([]store.BracketSlot, error)
	DeleteByCategoryID(competitionID, categoryID string) error
}

// CategoryBracketSummarizer adalah kemampuan opsional repo bagan untuk meringkas bagan per kategori.
type CategoryBracketSummarizer interface {
	ListCategorySummaries(competitionID string) ([]store.CategoryBracketSummary, error)
}

type MatchCreatorRepository interface {
	Create(m *store.Match) error
}

type ArenaListRepository interface {
	ListByCompetition(competitionID string) ([]store.Arena, error)
}

// CategoryArenaRepository kemampuan opsional repo kategori untuk menetapkan gelanggang asal kategori.
type CategoryArenaRepository interface {
	GetByID(id string) (*domain.Category, error)
	SetHomeArena(competitionID, categoryID, arenaID string) error
	ArenaLoads(competitionID, exceptCategoryID string) (map[string]int, error)
}

type BracketService struct {
	repo           BracketRepository
	matchRepo      MatchCreatorRepository
	arenaRepo      ArenaListRepository
	categoryArenas CategoryArenaRepository
}

// WithCategoryArenas mengaktifkan penetapan gelanggang asal per kategori saat bagan dibuat.
func (s *BracketService) WithCategoryArenas(r CategoryArenaRepository) *BracketService {
	s.categoryArenas = r
	return s
}

// pickHomeArena memilih gelanggang asal kategori: yang sudah ditetapkan bila masih ada, selain itu gelanggang dengan
// sisa partai paling sedikit (seri: nama paling awal; `arenas` sudah terurut). Hasilnya pasti untuk kondisi yang sama.
func (s *BracketService) pickHomeArena(competitionID, categoryID string, arenas []store.Arena) (string, error) {
	if categoryID == "" || s.categoryArenas == nil {
		return arenas[0].ID, nil
	}
	if cat, err := s.categoryArenas.GetByID(categoryID); err == nil && cat != nil && cat.HomeArenaID != nil {
		for _, a := range arenas {
			if a.ID == *cat.HomeArenaID {
				return a.ID, nil
			}
		}
	}
	loads, err := s.categoryArenas.ArenaLoads(competitionID, categoryID)
	if err != nil {
		return "", err
	}
	best := arenas[0].ID
	for _, a := range arenas[1:] {
		if loads[a.ID] < loads[best] {
			best = a.ID
		}
	}
	return best, nil
}

func NewBracketService(repo BracketRepository) *BracketService {
	return &BracketService{repo: repo}
}

func (s *BracketService) WithMatchStore(m MatchCreatorRepository) *BracketService {
	s.matchRepo = m
	return s
}

func (s *BracketService) WithArenaStore(a ArenaListRepository) *BracketService {
	s.arenaRepo = a
	return s
}

// GenerateBracket membuat seluruh slot turnamen untuk kategori tertentu (berdasarkan kelas bobot + gender).
func (s *BracketService) GenerateBracket(competitionID, weightClass, gender string, participantIDs []string) ([]store.BracketSlot, error) {
	return s.generate(competitionID, "", weightClass, gender, "", participantIDs)
}

// GenerateCategoryBracket membuat bagan yang terisolasi per category_id. bracketMode
// "pemasalan_4" menghasilkan mini-bracket (maks 4 peserta per grup) untuk kategori usia dini.
func (s *BracketService) GenerateCategoryBracket(competitionID, categoryID, weightClass, gender, bracketMode string, participantIDs []string) ([]store.BracketSlot, error) {
	if categoryID == "" {
		return nil, errors.New("category_id is required")
	}
	if _, ok := s.repo.(CategoryBracketRepository); !ok {
		return nil, errors.New("bracket repository does not support per-category brackets")
	}
	return s.generate(competitionID, categoryID, weightClass, gender, bracketMode, participantIDs)
}

func (s *BracketService) generate(competitionID, categoryID, weightClass, gender, bracketMode string, participantIDs []string) ([]store.BracketSlot, error) {
	if len(participantIDs) < 2 {
		return nil, errors.New("minimum 2 participants required to generate a bracket")
	}

	// Gelanggang harus ada sebelum bagan dibuat: partai tanpa gelanggang tidak muncul di antrean wasit mana pun.
	var arenas []store.Arena
	homeArenaID := ""
	if s.matchRepo != nil && s.arenaRepo != nil {
		var err error
		if arenas, err = s.arenaRepo.ListByCompetition(competitionID); err != nil {
			return nil, err
		}
		if len(arenas) == 0 {
			return nil, errors.New("belum ada gelanggang pada kejuaraan ini; buat gelanggang terlebih dahulu sebelum membuat bagan")
		}
		if homeArenaID, err = s.pickHomeArena(competitionID, categoryID, arenas); err != nil {
			return nil, err
		}
	}

	// Hapus slot bagan sebelumnya jika ada (mendukung regenerasi / acak ulang)
	if categoryID != "" {
		cr := s.repo.(CategoryBracketRepository)
		existing, err := cr.GetByCategoryID(competitionID, categoryID)
		if err != nil {
			return nil, err
		}
		if len(existing) > 0 {
			if err := cr.DeleteByCategoryID(competitionID, categoryID); err != nil {
				return nil, err
			}
		}
	} else {
		existing, err := s.repo.GetByCompetitionCategory(competitionID, weightClass, gender)
		if err != nil {
			return nil, err
		}
		if len(existing) > 0 {
			if err := s.repo.DeleteByCompetitionCategory(competitionID, weightClass, gender); err != nil {
				return nil, err
			}
		}
	}

	var slots []store.BracketSlot
	if bracketMode == domain.BracketPemasalan4 {
		slots = bracket.GenerateMiniSlots(competitionID, weightClass, gender, participantIDs)
	} else {
		slots = bracket.GenerateSlots(competitionID, weightClass, gender, participantIDs)
	}
	bracket.AssignCategory(slots, categoryID)
	if err := s.repo.CreateBatch(slots); err != nil {
		return nil, err
	}

	// Otomatis sinkronisasi partai Babak 1 ke tabel matches (Jadwal Pertandingan)
	if s.matchRepo != nil {
		// Semua partai kategori (termasuk babak lanjutan, lihat BracketStore.AdvanceWinner) memakai gelanggang asal.
		if categoryID != "" && s.categoryArenas != nil && homeArenaID != "" {
			if err := s.categoryArenas.SetHomeArena(competitionID, categoryID, homeArenaID); err != nil {
				return nil, err
			}
		}

		// Pasangkan slot yang bersebelahan (0 & 1, 2 & 3, ...) di tiap ronde. Ronde 1 selalu; ronde berikutnya hanya
		// bila kedua slot sudah terisi sejak awal, yaitu peserta yang lolos BYE dan bertemu sesama pemegang BYE.
		// Sisanya dibuat otomatis saat pemenang maju (BracketStore.AdvanceWinner).
		byRound := map[int][]store.BracketSlot{}
		for _, sl := range slots {
			byRound[sl.Round] = append(byRound[sl.Round], sl)
		}
		rounds := make([]int, 0, len(byRound))
		for r := range byRound {
			rounds = append(rounds, r)
		}
		sort.Ints(rounds)

		for _, r := range rounds {
			list := byRound[r]
			sort.Slice(list, func(i, j int) bool { return list[i].SlotIndex < list[j].SlotIndex })
			for i := 0; i+1 < len(list); i += 2 {
				slotRed, slotBlue := list[i], list[i+1]
				if slotRed.SlotIndex^1 != slotBlue.SlotIndex {
					continue
				}
				// Partai valid jika kedua sudut memiliki peserta sah (bukan BYE / kosong)
				if slotRed.ParticipantID == nil || slotRed.IsBye || slotBlue.ParticipantID == nil || slotBlue.IsBye {
					continue
				}
				m := &store.Match{
					CompetitionID: competitionID,
					ArenaID:       homeArenaID,
					BracketSlotID: slotRed.ID,
					AthleteRedID:  *slotRed.ParticipantID,
					AthleteBlueID: *slotBlue.ParticipantID,
					CurrentRound:  r,
					Status:        "idle",
				}
				if err := s.matchRepo.Create(m); err != nil {
					return nil, err
				}
			}
		}
	}

	return slots, nil
}

// ResetCategoryBracket menghapus bagan satu kategori beserta partai idle-nya. Pemanggil memastikan bagan boleh direset.
func (s *BracketService) ResetCategoryBracket(competitionID, categoryID string) error {
	cr, ok := s.repo.(CategoryBracketRepository)
	if !ok {
		return errors.New("bracket repository does not support per-category brackets")
	}
	return cr.DeleteByCategoryID(competitionID, categoryID)
}

// GetBracketTreeByCategory mengambil bagan terisolasi milik satu kategori.
func (s *BracketService) GetBracketTreeByCategory(competitionID, categoryID string) ([]store.BracketSlot, error) {
	cr, ok := s.repo.(CategoryBracketRepository)
	if !ok {
		return nil, errors.New("bracket repository does not support per-category brackets")
	}
	return cr.GetByCategoryID(competitionID, categoryID)
}

// GetBracketTree mengambil struktur pohon bagan pertandingan
func (s *BracketService) GetBracketTree(competitionID, weightClass, gender string) ([]store.BracketSlot, error) {
	return s.repo.GetByCompetitionCategory(competitionID, weightClass, gender)
}

// ListCategoryBracketSummaries meringkas semua bagan kategori pada satu kejuaraan (untuk daftar bagan publik).
func (s *BracketService) ListCategoryBracketSummaries(competitionID string) ([]store.CategoryBracketSummary, error) {
	sm, ok := s.repo.(CategoryBracketSummarizer)
	if !ok {
		return nil, errors.New("bracket repository does not support category summaries")
	}
	return sm.ListCategorySummaries(competitionID)
}

// CategoryBracketStarted true bila ada partai bagan kategori itu yang sudah dimulai/selesai/dibatalkan, sehingga
// bagan tidak boleh dibentuk ulang.
func (s *BracketService) CategoryBracketStarted(competitionID, categoryID string) (bool, error) {
	pc, ok := s.repo.(BracketProgressChecker)
	if !ok {
		return false, nil
	}
	return pc.HasStartedMatches(competitionID, categoryID)
}
