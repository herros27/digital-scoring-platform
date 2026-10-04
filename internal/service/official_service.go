package service

import (
	"errors"
	"time"

	"github.com/herros27/digital-scoring/internal/auth"
	"github.com/herros27/digital-scoring/internal/domain"
	"golang.org/x/crypto/bcrypt"
)

type OfficialRepository interface {
	Create(o *domain.Official, passwordHash string) error
	GetByID(id string) (*domain.Official, error)
	GetByEmail(email string) (*domain.Official, string, error)
	List() ([]domain.Official, error)
	Update(o *domain.Official) error
	Delete(id string) error
}

type OfficialService struct {
	repo OfficialRepository
}

func NewOfficialService(repo OfficialRepository) *OfficialService {
	return &OfficialService{repo: repo}
}

func (s *OfficialService) Login(email, password string) (string, *domain.Official, error) {
	if email == "" || password == "" {
		return "", nil, errors.New("email and password are required")
	}

	official, hash, err := s.repo.GetByEmail(email)
	if err != nil || official == nil {
		return "", nil, errors.New("invalid email or password")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)); err != nil {
		return "", nil, errors.New("invalid email or password")
	}

	// 24 hour token for admin/operator
	token, err := auth.GenerateToken(official.ID, "", "", official.Role, 24*time.Hour)
	if err != nil {
		return "", nil, err
	}

	return token, official, nil
}

func (s *OfficialService) CreateOfficial(name, email, password, role string) (*domain.Official, error) {
	if name == "" || email == "" || password == "" {
		return nil, errors.New("name, email, and password are required")
	}
	if role == "" {
		role = "operator"
	}

	existing, _, _ := s.repo.GetByEmail(email)
	if existing != nil {
		return nil, domain.ErrConflict
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	official := &domain.Official{
		Name:  name,
		Email: email,
		Role:  role,
	}

	if err := s.repo.Create(official, string(hash)); err != nil {
		return nil, err
	}
	return official, nil
}

func (s *OfficialService) GetOfficialByID(id string) (*domain.Official, error) {
	if id == "" {
		return nil, domain.ErrInvalidInput
	}
	o, err := s.repo.GetByID(id)
	if err != nil {
		return nil, err
	}
	if o == nil {
		return nil, domain.ErrNotFound
	}
	return o, nil
}

func (s *OfficialService) ListOfficials() ([]domain.Official, error) {
	return s.repo.List()
}

func (s *OfficialService) UpdateOfficial(id, name, role string) (*domain.Official, error) {
	if id == "" {
		return nil, domain.ErrInvalidInput
	}
	o, err := s.repo.GetByID(id)
	if err != nil {
		return nil, err
	}
	if o == nil {
		return nil, domain.ErrNotFound
	}
	if name != "" {
		o.Name = name
	}
	if role != "" {
		o.Role = role
	}
	if err := s.repo.Update(o); err != nil {
		return nil, err
	}
	return o, nil
}

func (s *OfficialService) DeleteOfficial(id string) error {
	if id == "" {
		return domain.ErrInvalidInput
	}
	o, err := s.repo.GetByID(id)
	if err != nil {
		return err
	}
	if o == nil {
		return domain.ErrNotFound
	}
	return s.repo.Delete(id)
}
