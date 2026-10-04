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

type AthletePagedRepository interface {
	ListPaged(gender string, limit, offset int) ([]domain.Athlete, int, error)
}

func (s *AthleteService) ListAthletes() ([]domain.Athlete, error) {
	return s.repo.List()
}

func (s *AthleteService) GetAthletesPaged(gender string, page, limit int) ([]domain.Athlete, int, error) {
	if page <= 0 {
		page = 1
	}
	if limit <= 0 {
		limit = 20
	}
	offset := (page - 1) * limit

	if pagedRepo, ok := s.repo.(AthletePagedRepository); ok {
		return pagedRepo.ListPaged(gender, limit, offset)
	}

	all, err := s.repo.List()
	if err != nil {
		return nil, 0, err
	}

	var filtered []domain.Athlete
	for _, a := range all {
		if gender == "" || a.Gender == gender {
			filtered = append(filtered, a)
		}
	}
	total := len(filtered)
	if offset >= total {
		return []domain.Athlete{}, total, nil
	}
	end := offset + limit
	if end > total {
		end = total
	}
	return filtered[offset:end], total, nil
}

func (s *AthleteService) RegisterAthleteToCompetition(competitionID, athleteID, weightClass, gender string) error {
	if competitionID == "" || athleteID == "" {
		return domain.ErrInvalidInput
	}
	return s.repo.RegisterToCompetition(competitionID, athleteID, weightClass, gender)
}

type ParticipantManager interface {
	RegisterParticipant(competitionID, athleteID, contingentID, weightClass, gender, status string) error
	UpdateParticipantStatus(participantID, status string) error
	ListParticipants(competitionID, status string) ([]domain.Participant, error)
}

func (s *AthleteService) RegisterParticipant(competitionID, athleteID, contingentID, weightClass, gender, status string) error {
	if competitionID == "" || athleteID == "" {
		return domain.ErrInvalidInput
	}
	if pm, ok := s.repo.(ParticipantManager); ok {
		return pm.RegisterParticipant(competitionID, athleteID, contingentID, weightClass, gender, status)
	}
	return s.repo.RegisterToCompetition(competitionID, athleteID, weightClass, gender)
}

func (s *AthleteService) VerifyParticipant(participantID, status string) error {
	if participantID == "" || (status != "verified" && status != "rejected") {
		return domain.ErrInvalidInput
	}
	if pm, ok := s.repo.(ParticipantManager); ok {
		return pm.UpdateParticipantStatus(participantID, status)
	}
	return nil
}

func (s *AthleteService) ListParticipants(competitionID, status string) ([]domain.Participant, error) {
	if competitionID == "" {
		return nil, domain.ErrInvalidInput
	}
	if pm, ok := s.repo.(ParticipantManager); ok {
		return pm.ListParticipants(competitionID, status)
	}
	return []domain.Participant{}, nil
}

func (s *AthleteService) ListAthletesByCompetition(competitionID string) ([]domain.Athlete, error) {
	if competitionID == "" {
		return nil, domain.ErrInvalidInput
	}
	return s.repo.ListByCompetition(competitionID)
}
