package service

import (
	"errors"
	"time"

	"github.com/herros27/digital-scoring/internal/store"
)

// Repository interface yang dibutuhkan oleh CompetitionService
type CompetitionRepository interface {
	Create(c *store.Competition) error
	GetByID(id string) (*store.Competition, error)
	List() ([]store.Competition, error)
	UpdateStatus(id, status string) error
}

type CompetitionService struct {
	repo CompetitionRepository
}

func NewCompetitionService(repo CompetitionRepository) *CompetitionService {
	return &CompetitionService{repo: repo}
}

func (s *CompetitionService) GetAllCompetitions() ([]store.Competition, error) {
	return s.repo.List()
}

func (s *CompetitionService) GetCompetitionByID(id string) (*store.Competition, error) {
	if id == "" {
		return nil, errors.New("competition id cannot be empty")
	}
	return s.repo.GetByID(id)
}

func (s *CompetitionService) CreateCompetition(c *store.Competition) error {
	if c.Name == "" {
		return errors.New("competition name is required")
	}
	if c.StartDate == "" || c.EndDate == "" {
		return errors.New("start date and end date are required")
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
		return errors.New("competition not found")
	}
	if comp.Status == "active" {
		return errors.New("competition is already active")
	}
	return s.repo.UpdateStatus(id, "active")
}