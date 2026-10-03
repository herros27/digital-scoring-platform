package service

import (
	"time"

	"github.com/herros27/digital-scoring/internal/domain"
	"github.com/herros27/digital-scoring/internal/rules"
	"github.com/herros27/digital-scoring/internal/store"
)

type RulesetRepository interface {
	List() ([]store.Ruleset, error)
	GetByID(id string) (*store.Ruleset, error)
	GetActive() (*store.Ruleset, error)
}

type RulesetService struct {
	repo RulesetRepository
}

func NewRulesetService(repo RulesetRepository) *RulesetService {
	return &RulesetService{repo: repo}
}

// ListRulesets mengambil daftar ruleset dari DB atau fallback ke built-in rulesets
func (s *RulesetService) ListRulesets() ([]domain.Ruleset, error) {
	if s.repo != nil {
		list, err := s.repo.List()
		if err == nil && len(list) > 0 {
			return list, nil
		}
	}

	return getBuiltinRulesets(), nil
}

// GetRulesetByID mengambil detail ruleset berdasarkan ID
func (s *RulesetService) GetRulesetByID(id string) (*domain.Ruleset, error) {
	if id == "" {
		return nil, domain.ErrInvalidInput
	}
	if s.repo != nil {
		r, err := s.repo.GetByID(id)
		if err == nil && r != nil {
			return r, nil
		}
	}

	for _, b := range getBuiltinRulesets() {
		if b.ID == id || b.Name == id {
			return &b, nil
		}
	}

	return nil, domain.ErrNotFound
}

func getBuiltinRulesets() []domain.Ruleset {
	now := time.Now()
	ipsi, _ := rules.LoadRuleset("ipsi")
	persilat, _ := rules.LoadRuleset("persilat")

	var list []domain.Ruleset
	if ipsi != nil {
		list = append(list, domain.Ruleset{
			ID:        "ipsi-2024",
			Name:      ipsi.Name,
			Version:   ipsi.Version,
			Scoring:   ipsi.Scoring,
			Status:    "active",
			CreatedAt: now,
		})
	}
	if persilat != nil {
		list = append(list, domain.Ruleset{
			ID:        "persilat-2024",
			Name:      persilat.Name,
			Version:   persilat.Version,
			Scoring:   persilat.Scoring,
			Status:    "active",
			CreatedAt: now,
		})
	}
	return list
}
