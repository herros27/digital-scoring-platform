package service

import (
	"errors"
	"fmt"
	"strings"
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
	repo         AthleteRepository
	categoryRepo CategoryRepository
}

// WithCategoryRepo mengaktifkan validasi kategori (umur/gender/bobot) pada pendaftaran & verifikasi.
func (s *AthleteService) WithCategoryRepo(r CategoryRepository) *AthleteService {
	s.categoryRepo = r
	return s
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
	a.BirthDate = strings.TrimSpace(a.BirthDate)
	a.NIK = strings.TrimSpace(a.NIK)
	if a.BirthDate != "" {
		if _, err := domain.ParseBirthDate(a.BirthDate); err != nil {
			return err
		}
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

// ---------- Kategori, timbang badan, & verifikasi berkas ----------

// CategoryParticipantManager adalah kemampuan opsional repo atlet untuk fitur kategori IPSI.
type CategoryParticipantManager interface {
	RegisterParticipantInCategory(competitionID, athleteID, contingentID, categoryID, weightClass, gender, status string) error
	GetParticipantByID(id string) (*domain.Participant, error)
	SaveVerification(participantID, status string, actualWeight *float64, docs map[string]bool) error
}

// RegisterParticipantToCategory mendaftarkan atlet ke kategori resmi setelah memvalidasi gender &
// golongan umur pada tanggal acuan (hari pertama kejuaraan). Mengembalikan kategori yang dipakai.
func (s *AthleteService) RegisterParticipantToCategory(competitionID, athleteID, contingentID, categoryID, status string, ref time.Time) (*domain.Category, error) {
	if competitionID == "" || athleteID == "" || categoryID == "" {
		return nil, domain.ErrInvalidInput
	}
	if s.categoryRepo == nil {
		return nil, errors.New("category validation is not configured")
	}
	cm, ok := s.repo.(CategoryParticipantManager)
	if !ok {
		return nil, errors.New("athlete repository does not support category registration")
	}
	athlete, err := s.GetAthleteByID(athleteID)
	if err != nil {
		return nil, err
	}
	cat, err := s.CheckCategoryEligibility(competitionID, categoryID, athlete, ref)
	if err != nil {
		return nil, err
	}
	if err := cm.RegisterParticipantInCategory(competitionID, athleteID, contingentID, cat.ID, cat.WeightClass, domain.NormalizeGender(athlete.Gender), status); err != nil {
		return nil, err
	}
	return cat, nil
}

// CheckCategoryEligibility memastikan kategori milik kompetisi dan atlet (gender + golongan umur)
// sah untuk kategori itu, tanpa menyimpan apa pun. Dipanggil sebelum atlet baru dibuat agar tidak ada data yatim.
func (s *AthleteService) CheckCategoryEligibility(competitionID, categoryID string, a *domain.Athlete, ref time.Time) (*domain.Category, error) {
	if s.categoryRepo == nil {
		return nil, errors.New("category validation is not configured")
	}
	cat, err := s.categoryRepo.GetByID(categoryID)
	if err != nil {
		return nil, errors.New("category not found")
	}
	if cat == nil || cat.CompetitionID != competitionID {
		return nil, errors.New("category not found in this competition")
	}
	if err := domain.CheckEligibility(a, cat, ref); err != nil {
		return nil, err
	}
	return cat, nil
}

// VerifyInput data meja verifikasi: hasil timbang aktual + checklist berkas.
type VerifyInput struct {
	Status       string
	ActualWeight *float64
	Docs         map[string]bool
	// Ref tanggal acuan umur (hari pertama kejuaraan); nol = hari ini.
	Ref time.Time
}

// VerifyResult hasil akhir verifikasi (status bisa berbeda dari yang diminta bila bobot tidak sah).
type VerifyResult struct {
	Status string
	Reason string
}

// VerifyParticipantDetailed menerapkan aturan resmi: tanpa toleransi bobot (0 kg) dan
// berkas perlindungan atlet di bawah umur harus lengkap sebelum status VERIFIED.
func (s *AthleteService) VerifyParticipantDetailed(participantID string, in VerifyInput) (*VerifyResult, error) {
	if participantID == "" || (in.Status != "verified" && in.Status != "rejected") {
		return nil, domain.ErrInvalidInput
	}
	if in.ActualWeight != nil && (*in.ActualWeight <= 0 || *in.ActualWeight > 300) {
		return nil, errors.New("actual_weight must be between 0 and 300 kg")
	}
	cm, ok := s.repo.(CategoryParticipantManager)
	if !ok {
		// Repo lama tanpa dukungan kategori: perilaku verifikasi sederhana.
		return &VerifyResult{Status: in.Status}, s.VerifyParticipant(participantID, in.Status)
	}
	part, err := cm.GetParticipantByID(participantID)
	if err != nil {
		return nil, err
	}
	if part == nil {
		return nil, domain.ErrNotFound
	}

	docs := part.DocsVerified
	if in.Docs != nil {
		docs = in.Docs
	}
	weight := part.ActualWeight
	if in.ActualWeight != nil {
		weight = in.ActualWeight
	}

	result := &VerifyResult{Status: in.Status}
	if in.Status == "verified" && part.CategoryID != "" && s.categoryRepo != nil {
		cat, err := s.categoryRepo.GetByID(part.CategoryID)
		if err != nil {
			return nil, err
		}
		ref := in.Ref
		if ref.IsZero() {
			ref = time.Now()
		}
		switch domain.AgeCheck(part.BirthDate, cat.AgeGroup, cat.PresetStandard, ref) {
		case domain.AgeCheckMissing:
			return nil, errors.New("tanggal lahir atlet belum diisi; lengkapi dahulu untuk memastikan golongan umur")
		case domain.AgeCheckMismatch:
			return nil, errors.New("usia atlet tidak sesuai golongan umur kategori; periksa tanggal lahir atau pindahkan ke kategori yang tepat")
		}
		if missing := domain.MissingDocs(cat.AgeGroup, part.Gender, docs); len(missing) > 0 {
			return nil, errors.New("berkas belum lengkap: " + strings.Join(missing, ", "))
		}
		if cat.HasWeightBounds() {
			if weight == nil {
				return nil, errors.New("hasil timbang badan (actual_weight) wajib diisi untuk kategori bertingkat bobot")
			}
			switch cat.CheckWeight(*weight) {
			case domain.WeightOver:
				result.Status = domain.ParticipantOverweight
				result.Reason = "bobot melebihi batas kelas (tanpa toleransi)"
			case domain.WeightUnder:
				result.Status = domain.ParticipantUnderweight
				result.Reason = "bobot di bawah batas kelas (tanpa toleransi)"
			}
		}
	}

	if err := cm.SaveVerification(participantID, result.Status, in.ActualWeight, docs); err != nil {
		return nil, err
	}
	return result, nil
}

// AthleteProfileUpdater kemampuan opsional repo atlet untuk melengkapi tanggal lahir/NIK.
type AthleteProfileUpdater interface {
	UpdateProfile(id string, birthDate, nik *string) error
}

// UpdateAthleteProfile melengkapi/memperbaiki tanggal lahir dan NIK atlet (field nil tidak diubah).
// Dipakai untuk data lama yang belum punya tanggal lahir agar golongan umur bisa diperiksa.
func (s *AthleteService) UpdateAthleteProfile(id string, birthDate, nik *string) (*domain.Athlete, error) {
	if id == "" || (birthDate == nil && nik == nil) {
		return nil, domain.ErrInvalidInput
	}
	if birthDate != nil {
		v := strings.TrimSpace(*birthDate)
		if _, err := domain.ParseBirthDate(v); err != nil {
			return nil, err
		}
		birthDate = &v
	}
	if nik != nil {
		v := strings.TrimSpace(*nik)
		if len(v) > 50 {
			return nil, errors.New("nik must be at most 50 characters")
		}
		nik = &v
	}
	up, ok := s.repo.(AthleteProfileUpdater)
	if !ok {
		return nil, errors.New("athlete repository does not support profile updates")
	}
	if err := up.UpdateProfile(id, birthDate, nik); err != nil {
		return nil, err
	}
	return s.GetAthleteByID(id)
}

// ParticipantRemover kemampuan opsional repo atlet untuk membatalkan pendaftaran peserta.
type ParticipantRemover interface {
	DeleteParticipant(competitionID, participantID string) error
}

// UnregisterParticipant membatalkan pendaftaran peserta dari kejuaraan. Admin/dewan boleh kapan saja (selama peserta
// belum masuk bagan/partai). Manajer kontingen (restricted) hanya boleh membatalkan peserta kontingennya sendiri yang
// belum diverifikasi; peserta kontingen lain dijawab ErrNotFound agar keberadaannya tidak bocor.
func (s *AthleteService) UnregisterParticipant(competitionID, participantID, contingentScope string, restricted bool) error {
	if competitionID == "" || participantID == "" {
		return domain.ErrInvalidInput
	}
	rm, ok := s.repo.(ParticipantRemover)
	if !ok {
		return errors.New("athlete repository does not support unregistering participants")
	}
	if restricted {
		cm, ok := s.repo.(CategoryParticipantManager)
		if !ok {
			return errors.New("athlete repository does not support participant lookup")
		}
		part, err := cm.GetParticipantByID(participantID)
		if err != nil || part == nil || part.CompetitionID != competitionID || contingentScope == "" || part.ContingentID != contingentScope {
			return domain.ErrNotFound
		}
		if part.RegistrationStatus == "verified" {
			return fmt.Errorf("%w: peserta sudah diverifikasi; hubungi panitia untuk membatalkan pendaftaran", domain.ErrForbidden)
		}
	}
	return rm.DeleteParticipant(competitionID, participantID)
}
