package service

import (
	"errors"
	"time"

	"github.com/herros27/digital-scoring/internal/domain"
)

// Repository interface yang dibutuhkan oleh CompetitionService
type CompetitionRepository interface {
	Create(c *domain.Competition) error
	GetByID(id string) (*domain.Competition, error)
	List() ([]domain.Competition, error)
	UpdateStatus(id, status string) error
}

type CompetitionService struct {
	repo CompetitionRepository
}

func NewCompetitionService(repo CompetitionRepository) *CompetitionService {
	return &CompetitionService{repo: repo}
}

func (s *CompetitionService) GetAllCompetitions() ([]domain.Competition, error) {
	return s.repo.List()
}

func (s *CompetitionService) GetCompetitionByID(id string) (*domain.Competition, error) {
	if id == "" {
		return nil, domain.ErrInvalidInput
	}
	comp, err := s.repo.GetByID(id)
	if err != nil {
		return nil, err
	}
	if comp == nil {
		return nil, domain.ErrCompetitionNotFound
	}
	return comp, nil
}

func (s *CompetitionService) CreateCompetition(c *domain.Competition) error {
	if c.Name == "" || c.StartDate == "" || c.EndDate == "" {
		return errors.New("competition name, start date, and end date are required")
	}

	// Default status selalu draft saat pertama dibuat
	c.Status = "draft"
	c.CreatedAt = time.Now()

	return s.repo.Create(c)
}

func (s *CompetitionService) ActivateCompetition(id string) error {
	comp, err := s.repo.GetByID(id)
	if err != nil {
		return err
	}
	if comp == nil {
		return domain.ErrCompetitionNotFound
	}
	if comp.Status == "active" {
		return domain.ErrAlreadyActive
	}
	return s.repo.UpdateStatus(id, "active")
}