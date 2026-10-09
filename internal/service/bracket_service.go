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
	UpdateWinner(slotID, winnerID string) error
	SetParticipant(slotID, participantID string) error
}

// CategoryBracketRepository adalah kemampuan opsional repo bagan untuk isolasi per category_id.
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

type BracketService struct {
	repo      BracketRepository
	matchRepo MatchCreatorRepository
	arenaRepo ArenaListRepository
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
		var arenas []store.Arena
		if s.arenaRepo != nil {
			arenas, _ = s.arenaRepo.ListByCompetition(competitionID)
		}

		// Filter dan urutkan slot Ronde 1
		var round1Slots []store.BracketSlot
		for _, sl := range slots {
			if sl.Round == 1 {
				round1Slots = append(round1Slots, sl)
			}
		}
		sort.Slice(round1Slots, func(i, j int) bool {
			return round1Slots[i].SlotIndex < round1Slots[j].SlotIndex
		})

		// Pasangkan slot Ronde 1 (0 & 1, 2 & 3, ...)
		matchCounter := 0
		for i := 0; i+1 < len(round1Slots); i += 2 {
			slotRed := round1Slots[i]
			slotBlue := round1Slots[i+1]

			// Partai valid jika kedua sudut memiliki peserta sah (bukan double BYE / salah satu BYE)
			if slotRed.ParticipantID != nil && !slotRed.IsBye && slotBlue.ParticipantID != nil && !slotBlue.IsBye {
				var arenaID string
				if len(arenas) > 0 {
					arenaID = arenas[matchCounter%len(arenas)].ID
				}

				m := &store.Match{
					CompetitionID: competitionID,
					ArenaID:       arenaID,
					BracketSlotID: slotRed.ID,
					AthleteRedID:  *slotRed.ParticipantID,
					AthleteBlueID: *slotBlue.ParticipantID,
					CurrentRound:  1,
					Status:        "idle",
				}
				_ = s.matchRepo.Create(m)
				matchCounter++
			}
		}
	}

	return slots, nil
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

// AdvanceWinner memajukan pemenang pertandingan ke slot ronde berikutnya
func (s *BracketService) AdvanceWinner(currentSlotID, winnerID string) error {
	currentSlot, err := s.repo.GetByID(currentSlotID)
	if err != nil {
		return err
	}
	if currentSlot == nil {
		return errors.New("slot not found")
	}

	// Update pemenang pada slot saat ini
	if err := s.repo.UpdateWinner(currentSlotID, winnerID); err != nil {
		return err
	}

	// Jika ada babak selanjutnya, masukkan winner ke next_slot_id
	if currentSlot.NextSlotID != nil {
		return s.repo.SetParticipant(*currentSlot.NextSlotID, winnerID)
	}

	// Jika tidak ada next_slot_id, berarti ini adalah babak Final (Juara 1)
	return nil
}

// ListCategoryBracketSummaries meringkas semua bagan kategori pada satu kejuaraan (untuk daftar bagan publik).
func (s *BracketService) ListCategoryBracketSummaries(competitionID string) ([]store.CategoryBracketSummary, error) {
	sm, ok := s.repo.(CategoryBracketSummarizer)
	if !ok {
		return nil, errors.New("bracket repository does not support category summaries")
	}
	return sm.ListCategorySummaries(competitionID)
}
