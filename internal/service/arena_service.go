package service

import (
	"errors"

	"github.com/herros27/digital-scoring/internal/store"
)

type ArenaRepository interface {
	Create(a *store.Arena) error
	ListByCompetition(competitionID string) ([]store.Arena, error)
}

type ArenaService struct {
	arenaRepo ArenaRepository
	compRepo  CompetitionRepository
}

func NewArenaService(arenaRepo ArenaRepository, compRepo CompetitionRepository) *ArenaService {
	return &ArenaService{
		arenaRepo: arenaRepo,
		compRepo:  compRepo,
	}
}

func (s *ArenaService) GetArenasByCompetition(competitionID string) ([]store.Arena, error) {
	if competitionID == "" {
		return nil, errors.New("competition id is required")
	}
	return s.arenaRepo.ListByCompetition(competitionID)
}

func (s *ArenaService) CreateArena(competitionID string, name string) (*store.Arena, error) {
	if competitionID == "" {
		return nil, errors.New("competition id is required")
	}
	if name == "" {
		return nil, errors.New("arena name cannot be empty")
	}

	// Validasi apakah competition ada
	comp, err := s.compRepo.GetByID(competitionID)
	if err != nil || comp == nil {
		return nil, errors.New("competition not found")
	}

	arena := &store.Arena{
		CompetitionID: competitionID,
		Name:          name,
	}

	if err := s.arenaRepo.Create(arena); err != nil {
		return nil, err
	}
	return arena, nil
}