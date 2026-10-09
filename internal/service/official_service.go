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

type OfficialRepository interface {
	Create(o *domain.Official, passwordHash string) error
	GetByID(id string) (*domain.Official, error)
	GetByEmail(email string) (*domain.Official, string, error)
	List() ([]domain.Official, error)
	Update(o *domain.Official) error
	Delete(id string) error
	UpdatePassword(id, passwordHash string) error
	CountByRole(role string) (int, error)
}

// MinPasswordLen panjang minimal password baru (reset, ganti sendiri, dan admin awal).
const MinPasswordLen = 8

// Actor petugas yang sedang melakukan perubahan (dari klaim token).
type Actor struct {
	ID   string
	Role string
}

// dewanManaged peran yang boleh dikelola dewan; admin dan dewan hanya dikelola admin.
var dewanManaged = map[string]bool{"wasit": true, "operator": true, "juri": true}

// CanManageRole apakah actorRole boleh membuat, mengubah, menghapus, atau mereset password akun berperan targetRole.
func CanManageRole(actorRole, targetRole string) bool {
	switch strings.ToLower(actorRole) {
	case "admin":
		return true
	case "dewan":
		return dewanManaged[strings.ToLower(targetRole)]
	}
	return false
}

func forbidden(format string, a ...any) error {
	return fmt.Errorf("%w: %s", domain.ErrForbidden, fmt.Sprintf(format, a...))
}

func conflict(format string, a ...any) error {
	return fmt.Errorf("%w: %s", domain.ErrConflict, fmt.Sprintf(format, a...))
}

func checkPassword(p string) error {
	if len(p) < MinPasswordLen {
		return fmt.Errorf("%w: password minimal %d karakter", domain.ErrInvalidInput, MinPasswordLen)
	}
	return nil
}

// load mengambil akun; tidak ditemukan (baris tidak ada atau repo mengembalikan nil) selalu menjadi ErrNotFound.
func (s *OfficialService) load(id string) (*domain.Official, error) {
	if id == "" {
		return nil, domain.ErrInvalidInput
	}
	o, err := s.repo.GetByID(id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if o == nil {
		return nil, domain.ErrNotFound
	}
	return o, nil
}

// lastAdmin true bila akun admin ini satu-satunya admin.
func (s *OfficialService) lastAdmin(o *domain.Official) (bool, error) {
	if o.Role != "admin" {
		return false, nil
	}
	n, err := s.repo.CountByRole("admin")
	if err != nil {
		return false, err
	}
	return n <= 1, nil
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
	return s.load(id)
}

func (s *OfficialService) ListOfficials() ([]domain.Official, error) {
	return s.repo.List()
}

func (s *OfficialService) UpdateOfficial(id, name, role string) (*domain.Official, error) {
	o, err := s.load(id)
	if err != nil {
		return nil, err
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
	if _, err := s.load(id); err != nil {
		return err
	}
	return s.repo.Delete(id)
}

// CreateOfficialBy membuat akun atas nama actor: dewan hanya boleh membuat wasit, operator, dan juri.
func (s *OfficialService) CreateOfficialBy(actor Actor, name, email, password, role string) (*domain.Official, error) {
	if role == "" {
		role = "operator"
	}
	if !CanManageRole(actor.Role, role) {
		return nil, forbidden("peran %s tidak boleh membuat akun berperan %s", actor.Role, role)
	}
	if err := checkPassword(password); err != nil {
		return nil, err
	}
	return s.CreateOfficial(name, email, password, role)
}

// UpdateOfficialBy mengubah nama dan peran. Peran admin terakhir tidak bisa diturunkan.
func (s *OfficialService) UpdateOfficialBy(actor Actor, id, name, role string) (*domain.Official, error) {
	o, err := s.load(id)
	if err != nil {
		return nil, err
	}
	if !CanManageRole(actor.Role, o.Role) || (role != "" && !CanManageRole(actor.Role, role)) {
		return nil, forbidden("peran %s tidak boleh mengubah akun ini ke peran %s", actor.Role, role)
	}
	if role != "" && role != o.Role {
		if last, err := s.lastAdmin(o); err != nil {
			return nil, err
		} else if last {
			return nil, conflict("admin terakhir tidak dapat diturunkan perannya")
		}
	}
	return s.UpdateOfficial(id, name, role)
}

// DeleteOfficialBy menghapus akun. Akun sendiri dan admin terakhir tidak dapat dihapus.
func (s *OfficialService) DeleteOfficialBy(actor Actor, id string) error {
	o, err := s.load(id)
	if err != nil {
		return err
	}
	if !CanManageRole(actor.Role, o.Role) {
		return forbidden("peran %s tidak boleh menghapus akun berperan %s", actor.Role, o.Role)
	}
	if o.ID == actor.ID {
		return conflict("tidak dapat menghapus akun sendiri")
	}
	if last, err := s.lastAdmin(o); err != nil {
		return err
	} else if last {
		return conflict("admin terakhir tidak dapat dihapus")
	}
	return s.repo.Delete(id)
}

// ResetPassword mengganti password akun lain tanpa password lama (lupa password). Token lama tetap berlaku sampai habis.
func (s *OfficialService) ResetPassword(actor Actor, id, newPassword string) error {
	if err := checkPassword(newPassword); err != nil {
		return err
	}
	o, err := s.load(id)
	if err != nil {
		return err
	}
	if !CanManageRole(actor.Role, o.Role) {
		return forbidden("peran %s tidak boleh mereset password akun berperan %s", actor.Role, o.Role)
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	return s.repo.UpdatePassword(id, string(hash))
}

// ChangeOwnPassword mengganti password sendiri; password lama wajib benar.
func (s *OfficialService) ChangeOwnPassword(id, current, newPassword string) error {
	if err := checkPassword(newPassword); err != nil {
		return err
	}
	o, err := s.load(id)
	if err != nil {
		return err
	}
	_, hash, err := s.repo.GetByEmail(o.Email)
	if err != nil || bcrypt.CompareHashAndPassword([]byte(hash), []byte(current)) != nil {
		return fmt.Errorf("%w: password lama salah", domain.ErrUnauthorized)
	}
	if current == newPassword {
		return fmt.Errorf("%w: password baru harus berbeda dari yang lama", domain.ErrInvalidInput)
	}
	newHash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	return s.repo.UpdatePassword(id, string(newHash))
}

// EnsureAdmin membuat admin pertama dari konfigurasi (ADMIN_EMAIL/ADMIN_PASSWORD) bila belum ada admin sama sekali.
// Aman dipanggil di setiap start: bila sudah ada admin, tidak melakukan apa pun dan tidak menimpa password.
// Email atau password kosong berarti fitur tidak dipakai (created=false, tanpa galat).
func (s *OfficialService) EnsureAdmin(name, email, password string) (created bool, err error) {
	if email == "" && password == "" {
		return false, nil
	}
	if email == "" || password == "" {
		return false, fmt.Errorf("%w: ADMIN_EMAIL dan ADMIN_PASSWORD harus diisi bersamaan", domain.ErrInvalidInput)
	}
	if !strings.Contains(email, "@") {
		return false, fmt.Errorf("%w: ADMIN_EMAIL bukan alamat email", domain.ErrInvalidInput)
	}
	if err := checkPassword(password); err != nil {
		return false, err
	}
	n, err := s.repo.CountByRole("admin")
	if err != nil {
		return false, err
	}
	if n > 0 {
		return false, nil
	}
	if name == "" {
		name = "Administrator"
	}
	if _, err := s.CreateOfficial(name, email, password, "admin"); err != nil {
		return false, err
	}
	return true, nil
}
