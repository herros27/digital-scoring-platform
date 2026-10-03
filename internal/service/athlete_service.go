package service

import (
	"errors"
	"time"

	"github.com/herros27/digital-scoring/internal/domain"
)

type AthleteRepository interface {
	Create(a *domain.Athlete) error
	GetByID(id string) (*domain.Athlete, error)
	List() ([]domain.Athlete, error)
	RegisterToCompetition(competitionID, athleteID, weightClass, gender string) error
	ListByCompetition(competitionID string) ([]domain.Athlete, error)
}

type AthleteService struct {
	repo AthleteRepository
}

func NewAthleteService(repo AthleteRepository) *AthleteService {
	return &AthleteService{repo: repo}
}

func (s *AthleteService) CreateAthlete(a *domain.Athlete) error {
	if a.Name == "" {
		return errors.New("athlete name is required")
	}
	if a.Gender == "" {
		return errors.New("gender is required")
	}
	a.CreatedAt = time.Now()
	return s.repo.Create(a)
}

func (s *AthleteService) GetAthleteByID(id string) (*domain.Athlete, error) {
	if id == "" {
		return nil, domain.ErrInvalidInput
	}
	athlete, err := s.repo.GetByID(id)
	if err != nil {
		return nil, err
	}
	if athlete == nil {
		return nil, domain.ErrNotFound
	}
	return athlete, nil
}

func (s *AthleteService) ListAthletes() ([]domain.Athlete, error) {
	return s.repo.List()
}

func (s *AthleteService) RegisterAthleteToCompetition(competitionID, athleteID, weightClass, gender string) error {
	if competitionID == "" || athleteID == "" {
		return domain.ErrInvalidInput
	}
	return s.repo.RegisterToCompetition(competitionID, athleteID, weightClass, gender)
}

func (s *AthleteService) ListAthletesByCompetition(competitionID string) ([]domain.Athlete, error) {
	if competitionID == "" {
		return nil, domain.ErrInvalidInput
	}
	return s.repo.ListByCompetition(competitionID)
}
