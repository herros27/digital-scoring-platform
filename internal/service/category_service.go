package service

import (
	"errors"
	"fmt"
	"regexp"
	"time"

	"github.com/herros27/digital-scoring/internal/domain"
)

type CategoryRepository interface {
	Create(c *domain.Category) error
	GetByID(id string) (*domain.Category, error)
	ListByCompetition(competitionID string) ([]domain.Category, error)
}

// BracketLocker kemampuan opsional repo kategori untuk menyimpan kunci draw bagan.
type BracketLocker interface {
	SetBracketLock(competitionID, categoryID string, locked bool) (*domain.Category, error)
}

type CategoryService struct {
	repo CategoryRepository
	// rulesetExists opsional: memastikan ruleset khusus kategori benar-benar ada.
	rulesetExists func(id string) bool
}

// WithRulesetLookup mengaktifkan validasi keberadaan ruleset khusus kategori.
func (s *CategoryService) WithRulesetLookup(f func(id string) bool) *CategoryService {
	s.rulesetExists = f
	return s
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
	if err := normalizeCategory(c); err != nil {
		return err
	}
	if c.RulesetID != nil && *c.RulesetID == "" {
		c.RulesetID = nil
	}
	if err := s.checkRuleset(c.RulesetID); err != nil {
		return err
	}
	// Satu kejuaraan hanya boleh memakai satu standar (PB IPSI ATAU PERSILAT 2026).
	existing, err := s.repo.ListByCompetition(c.CompetitionID)
	if err != nil {
		return err
	}
	for _, e := range existing {
		if e.PresetStandard != "" && e.PresetStandard != c.PresetStandard {
			return fmt.Errorf("%w: kejuaraan ini sudah memakai standar %s; reset kategori terlebih dahulu untuk beralih ke %s",
				domain.ErrConflict, e.PresetStandard, c.PresetStandard)
		}
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

// normalizeCategory mengisi default (IPSI, dewasa, single elimination) dan memvalidasi metadata kategori.
func normalizeCategory(c *domain.Category) error {
	if c.PresetStandard == "" {
		c.PresetStandard = domain.PresetIPSI
	}
	if c.PresetStandard != domain.PresetIPSI && c.PresetStandard != domain.PresetPersilat26 {
		return errors.New("preset_standard must be 'ipsi' or 'persilat_2026'")
	}
	if c.AgeGroup == "" {
		c.AgeGroup = domain.AgeDewasa
	}
	valid := false
	for _, b := range domain.AgeBands(c.PresetStandard) {
		if b.Code == c.AgeGroup {
			valid = true
			break
		}
	}
	if !valid {
		return errors.New("age_group '" + c.AgeGroup + "' is not valid for preset " + c.PresetStandard)
	}
	if c.BracketMode == "" {
		c.BracketMode = domain.BracketSingleElim
	}
	if c.BracketMode != domain.BracketSingleElim && c.BracketMode != domain.BracketPemasalan4 {
		return errors.New("bracket_mode must be 'single_elimination' or 'pemasalan_4'")
	}
	if c.MinWeight != nil && *c.MinWeight < 0 {
		return errors.New("min_weight cannot be negative")
	}
	if c.MaxWeight != nil && *c.MaxWeight <= 0 {
		return errors.New("max_weight must be positive")
	}
	if c.MinWeight != nil && c.MaxWeight != nil && *c.MinWeight > *c.MaxWeight {
		return errors.New("min_weight cannot exceed max_weight")
	}
	return nil
}

// CategoryResetter kemampuan opsional repo untuk menghapus seluruh kategori satu kejuaraan.
type CategoryResetter interface {
	DeleteAllByCompetition(competitionID string) (int64, error)
}

// ResetCategories menghapus semua kategori kejuaraan agar standar dapat diganti.
// Mengembalikan ErrConflict bila masih ada peserta yang terdaftar pada kategori.
func (s *CategoryService) ResetCategories(competitionID string) (int64, error) {
	if competitionID == "" {
		return 0, domain.ErrInvalidInput
	}
	r, ok := s.repo.(CategoryResetter)
	if !ok {
		return 0, errors.New("category repository does not support reset")
	}
	return r.DeleteAllByCompetition(competitionID)
}

var uuidRe = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

func (s *CategoryService) checkRuleset(id *string) error {
	if id == nil || *id == "" {
		return nil
	}
	if !uuidRe.MatchString(*id) {
		return fmt.Errorf("%w: ruleset_id harus berupa UUID ruleset", domain.ErrInvalidInput)
	}
	if s.rulesetExists != nil && !s.rulesetExists(*id) {
		return fmt.Errorf("%w: ruleset tidak ditemukan", domain.ErrInvalidInput)
	}
	return nil
}

// CategoryRulesetAssigner kemampuan opsional repo untuk menetapkan ruleset khusus kategori.
type CategoryRulesetAssigner interface {
	AssignRulesets(competitionID string, items []domain.CategoryRulesetAssignment) (int64, error)
}

// AssignRulesets menetapkan ruleset khusus (mis. durasi Usia Dini 2×1,5 menit) per kategori atau
// per golongan umur. RulesetID kosong mengembalikan kategori ke ruleset kejuaraan.
func (s *CategoryService) AssignRulesets(competitionID string, items []domain.CategoryRulesetAssignment) (int64, error) {
	if competitionID == "" || len(items) == 0 {
		return 0, domain.ErrInvalidInput
	}
	for i := range items {
		it := &items[i]
		if (it.AgeGroup == "") == (it.CategoryID == "") {
			return 0, fmt.Errorf("%w: isi salah satu dari age_group atau category_id", domain.ErrInvalidInput)
		}
		if it.AgeGroup != "" && !domain.ValidAgeGroup(it.AgeGroup) {
			return 0, fmt.Errorf("%w: age_group '%s' tidak dikenal", domain.ErrInvalidInput, it.AgeGroup)
		}
		if it.RulesetID != nil && *it.RulesetID == "" {
			it.RulesetID = nil
		}
		if err := s.checkRuleset(it.RulesetID); err != nil {
			return 0, err
		}
	}
	a, ok := s.repo.(CategoryRulesetAssigner)
	if !ok {
		return 0, errors.New("category repository does not support ruleset assignment")
	}
	return a.AssignRulesets(competitionID, items)
}

// SetBracketLock mengunci (locked=true) atau membuka draw bagan sebuah kategori. Bagan yang terkunci tidak bisa
// diacak ulang sampai dibuka; operasinya idempoten.
func (s *CategoryService) SetBracketLock(competitionID, categoryID string, locked bool) (*domain.Category, error) {
	if competitionID == "" || categoryID == "" {
		return nil, domain.ErrInvalidInput
	}
	bl, ok := s.repo.(BracketLocker)
	if !ok {
		return nil, domain.ErrInvalidInput
	}
	return bl.SetBracketLock(competitionID, categoryID, locked)
}
