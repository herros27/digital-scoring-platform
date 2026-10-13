package service

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
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

// ContingentPasswordManager kemampuan opsional repo kontingen untuk mengganti password.
type ContingentPasswordManager interface {
	UpdatePassword(id, passwordHash string) error
}

type ContingentService struct {
	repo       ContingentRepository
	loginGuard *auth.AttemptLimiter
}

func NewContingentService(repo ContingentRepository) *ContingentService {
	return &ContingentService{
		repo:       repo,
		loginGuard: auth.NewAttemptLimiter(5, 15*time.Minute, 15*time.Minute),
	}
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

	key := strings.ToLower(email)
	if s.loginGuard != nil {
		if locked, _ := s.loginGuard.Check(key); locked {
			return "", nil, errors.New("terlalu banyak percobaan login gagal, akun dikunci sementara")
		}
	}

	c, hash, err := s.repo.GetByEmail(email)
	if err != nil || c == nil {
		_ = bcrypt.CompareHashAndPassword([]byte(dummyBcryptHash), []byte(password))
		if s.loginGuard != nil {
			s.loginGuard.Fail(key)
		}
		return "", nil, errors.New("invalid email or password")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)); err != nil {
		if s.loginGuard != nil {
			s.loginGuard.Fail(key)
		}
		return "", nil, errors.New("invalid email or password")
	}

	if s.loginGuard != nil {
		s.loginGuard.Reset(key)
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

func (s *ContingentService) passwordRepo() (ContingentPasswordManager, error) {
	pm, ok := s.repo.(ContingentPasswordManager)
	if !ok {
		return nil, errors.New("contingent repository does not support password change")
	}
	return pm, nil
}

func (s *ContingentService) load(id string) (*domain.Contingent, error) {
	if id == "" {
		return nil, domain.ErrInvalidInput
	}
	c, err := s.repo.GetByID(id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if c == nil {
		return nil, domain.ErrNotFound
	}
	return c, nil
}

// ResetPassword mengganti password manajer kontingen tanpa password lama (lupa password); dilakukan admin/dewan.
// Sesi (token) yang sedang berjalan tetap berlaku sampai habis masa berlakunya.
func (s *ContingentService) ResetPassword(id, newPassword string) error {
	if err := checkPassword(newPassword); err != nil {
		return err
	}
	pm, err := s.passwordRepo()
	if err != nil {
		return err
	}
	if _, err := s.load(id); err != nil {
		return err
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	return pm.UpdatePassword(id, string(hash))
}

// ChangeOwnPassword mengganti password sendiri; password lama wajib benar (dijawab 400, bukan 401, supaya klien
// tidak mengira sesi berakhir).
func (s *ContingentService) ChangeOwnPassword(id, current, newPassword string) error {
	if err := checkPassword(newPassword); err != nil {
		return err
	}
	pm, err := s.passwordRepo()
	if err != nil {
		return err
	}
	c, err := s.load(id)
	if err != nil {
		return err
	}
	_, hash, err := s.repo.GetByEmail(c.Email)
	if err != nil || bcrypt.CompareHashAndPassword([]byte(hash), []byte(current)) != nil {
		return fmt.Errorf("%w: password lama salah", domain.ErrInvalidInput)
	}
	if current == newPassword {
		return fmt.Errorf("%w: password baru harus berbeda dari yang lama", domain.ErrInvalidInput)
	}
	newHash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	return pm.UpdatePassword(id, string(newHash))
}
