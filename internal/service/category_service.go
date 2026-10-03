package service

import (
	"errors"
	"time"

	"github.com/herros27/digital-scoring/internal/domain"
)

type CategoryRepository interface {
	Create(c *domain.Category) error
	GetByID(id string) (*domain.Category, error)
	ListByCompetition(competitionID string) ([]domain.Category, error)
}

type CategoryService struct {
	repo CategoryRepository
}

func NewCategoryService(repo CategoryRepository) *CategoryService {
	return &CategoryService{repo: repo}
}

func (s *CategoryService) CreateCategory(c *domain.Category) error {
	if c.CompetitionID == "" {
		return errors.New("competition_id is required")
	}
	if c.Name == "" {
		return errors.New("category name is required")
	}
	if c.Gender == "" {
		return errors.New("gender is required")
	}
	c.CreatedAt = time.Now()
	return s.repo.Create(c)
}

func (s *CategoryService) GetCategoryByID(id string) (*domain.Category, error) {
	if id == "" {
		return nil, domain.ErrInvalidInput
	}
	cat, err := s.repo.GetByID(id)
	if err != nil {
		return nil, err
	}
	if cat == nil {
		return nil, domain.ErrNotFound
	}
	return cat, nil
}

func (s *CategoryService) ListCategories(competitionID string) ([]domain.Category, error) {
	if competitionID == "" {
		return nil, domain.ErrInvalidInput
	}
	return s.repo.ListByCompetition(competitionID)
}
