package service

import (
	"errors"

	"github.com/herros27/digital-scoring/internal/domain"
	"github.com/herros27/digital-scoring/internal/store"
)

type ArenaRepository interface {
	Create(a *store.Arena) error
	GetByID(id string) (*store.Arena, error)
	ListByCompetition(competitionID string) ([]store.Arena, error)
	Update(a *store.Arena) error
	Delete(id string) error
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

func (s *ArenaService) GetArenaByID(id string) (*store.Arena, error) {
	if id == "" {
		return nil, domain.ErrInvalidInput
	}
	arena, err := s.arenaRepo.GetByID(id)
	if err != nil {
		return nil, err
	}
	if arena == nil {
		return nil, domain.ErrNotFound
	}
	return arena, nil
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

func (s *ArenaService) UpdateArena(id string, name string) (*store.Arena, error) {
	if id == "" {
		return nil, domain.ErrInvalidInput
	}
	if name == "" {
		return nil, errors.New("arena name cannot be empty")
	}
	arena, err := s.arenaRepo.GetByID(id)
	if err != nil {
		return nil, err
	}
	if arena == nil {
		return nil, domain.ErrNotFound
	}
	arena.Name = name
	if err := s.arenaRepo.Update(arena); err != nil {
		return nil, err
	}
	return arena, nil
}

func (s *ArenaService) DeleteArena(id string) error {
	if id == "" {
		return domain.ErrInvalidInput
	}
	arena, err := s.arenaRepo.GetByID(id)
	if err != nil {
		return err
	}
	if arena == nil {
		return domain.ErrNotFound
	}
	return s.arenaRepo.Delete(id)
}
