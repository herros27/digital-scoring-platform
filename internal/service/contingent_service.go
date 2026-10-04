package service

import (
	"errors"
	"time"

	"github.com/herros27/digital-scoring/internal/auth"
	"github.com/herros27/digital-scoring/internal/domain"
	"golang.org/x/crypto/bcrypt"
)

type ContingentRepository interface {
	Create(c *domain.Contingent, passwordHash string) error
	GetByID(id string) (*domain.Contingent, error)
	GetByEmail(email string) (*domain.Contingent, string, error)
	List() ([]domain.Contingent, error)
}

type ContingentService struct {
	repo ContingentRepository
}

func NewContingentService(repo ContingentRepository) *ContingentService {
	return &ContingentService{repo: repo}
}

func (s *ContingentService) Register(name, picName, email, phone, password string) (*domain.Contingent, error) {
	if name == "" || picName == "" || email == "" || phone == "" || password == "" {
		return nil, errors.New("name, pic_name, email, phone, and password are required")
	}

	existing, _, _ := s.repo.GetByEmail(email)
	if existing != nil {
		return nil, domain.ErrConflict
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	c := &domain.Contingent{
		Name:    name,
		PicName: picName,
		Email:   email,
		Phone:   phone,
	}

	if err := s.repo.Create(c, string(hash)); err != nil {
		return nil, err
	}
	return c, nil
}

func (s *ContingentService) Login(email, password string) (string, *domain.Contingent, error) {
	if email == "" || password == "" {
		return "", nil, errors.New("email and password are required")
	}

	c, hash, err := s.repo.GetByEmail(email)
	if err != nil || c == nil {
		return "", nil, errors.New("invalid email or password")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)); err != nil {
		return "", nil, errors.New("invalid email or password")
	}

	// 24 hour token with role "contingent"
	token, err := auth.GenerateToken(c.ID, "", "", "contingent", 24*time.Hour)
	if err != nil {
		return "", nil, err
	}

	return token, c, nil
}

func (s *ContingentService) GetByID(id string) (*domain.Contingent, error) {
	if id == "" {
		return nil, domain.ErrInvalidInput
	}
	c, err := s.repo.GetByID(id)
	if err != nil {
		return nil, err
	}
	if c == nil {
		return nil, domain.ErrNotFound
	}
	return c, nil
}

func (s *ContingentService) List() ([]domain.Contingent, error) {
	return s.repo.List()
}
