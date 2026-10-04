package service

import (
	"errors"

	"github.com/herros27/digital-scoring/internal/bracket"
	"github.com/herros27/digital-scoring/internal/store"
)

type BracketRepository interface {
	CreateBatch(slots []store.BracketSlot) error
	GetByCompetitionCategory(competitionID, weightClass, gender string) ([]store.BracketSlot, error)
	GetByID(id string) (*store.BracketSlot, error)
	UpdateWinner(slotID, winnerID string) error
	SetParticipant(slotID, participantID string) error
}

type BracketService struct {
	repo BracketRepository
}

func NewBracketService(repo BracketRepository) *BracketService {
	return &BracketService{repo: repo}
}

// GenerateBracket membuat seluruh slot turnamen untuk kategori tertentu
func (s *BracketService) GenerateBracket(competitionID, weightClass, gender string, participantIDs []string) ([]store.BracketSlot, error) {
	if len(participantIDs) < 2 {
		return nil, errors.New("minimum 2 participants required to generate a bracket")
	}

	// Cek apakah sudah pernah di-generate sebelumnya
	existing, err := s.repo.GetByCompetitionCategory(competitionID, weightClass, gender)
	if err != nil {
		return nil, err
	}
	if len(existing) > 0 {
		return nil, errors.New("bracket for this category already exists")
	}

	slots := bracket.GenerateSlots(competitionID, weightClass, gender, participantIDs)
	if err := s.repo.CreateBatch(slots); err != nil {
		return nil, err
	}

	return slots, nil
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
