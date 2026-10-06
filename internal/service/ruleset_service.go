package service

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/herros27/digital-scoring/internal/domain"
	"github.com/herros27/digital-scoring/internal/rules"
	"github.com/herros27/digital-scoring/internal/store"
)

type RulesetRepository interface {
	Create(r *store.Ruleset) error
	Update(r *store.Ruleset) error
	Delete(id string) error
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

type CreateRulesetInput struct {
	Name    string            `json:"name" validate:"required,min=2,max=100"`
	Version string            `json:"version" validate:"required"`
	Scoring rules.ScoringRule `json:"scoring"`
	Status  string            `json:"status" validate:"omitempty,oneof=draft active archived"`
}

// CreateRuleset membuat ruleset baru secara dinamis
func (s *RulesetService) CreateRuleset(input CreateRulesetInput) (*domain.Ruleset, error) {
	if input.Name == "" || input.Version == "" {
		return nil, domain.ErrInvalidInput
	}
	if err := input.Scoring.Validate(); err != nil {
		return nil, fmt.Errorf("%w: %s", domain.ErrInvalidInput, err)
	}
	status := input.Status
	if status == "" {
		status = "active"
	}

	rs := &domain.Ruleset{
		Name:    input.Name,
		Version: input.Version,
		Scoring: input.Scoring,
		Status:  status,
	}

	if s.repo != nil {
		if err := s.repo.Create(rs); err != nil {
			return nil, err
		}
		return rs, nil
	}

	rs.ID = "ruleset-" + input.Name
	rs.CreatedAt = time.Now()
	return rs, nil
}

// UpdateRuleset memperbarui ruleset dinamis
func (s *RulesetService) UpdateRuleset(id string, input CreateRulesetInput) (*domain.Ruleset, error) {
	if id == "" || input.Name == "" || input.Version == "" {
		return nil, domain.ErrInvalidInput
	}
	if err := input.Scoring.Validate(); err != nil {
		return nil, fmt.Errorf("%w: %s", domain.ErrInvalidInput, err)
	}
	status := input.Status
	if status == "" {
		status = "active"
	}

	rs := &domain.Ruleset{
		ID:      id,
		Name:    input.Name,
		Version: input.Version,
		Scoring: input.Scoring,
		Status:  status,
	}

	if s.repo != nil {
		if err := s.repo.Update(rs); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return nil, domain.ErrNotFound
			}
			return nil, err
		}
		return rs, nil
	}

	return rs, nil
}

// DeleteRuleset menghapus ruleset
func (s *RulesetService) DeleteRuleset(id string) error {
	if id == "" {
		return domain.ErrInvalidInput
	}
	if s.repo != nil {
		if err := s.repo.Delete(id); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return domain.ErrNotFound
			}
			return err
		}
	}
	return nil
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

// ResolveRuleset memuat engine Ruleset dari DB (ID/Nama) atau fallback ke file JSON/hardcoded
func (s *RulesetService) ResolveRuleset(identifier string) (*rules.Ruleset, error) {
	if identifier != "" && s.repo != nil {
		if r, err := s.repo.GetByID(identifier); err == nil && r != nil {
			return toRulesStruct(r)
		}
		if list, err := s.repo.List(); err == nil {
			for _, item := range list {
				if item.Name == identifier || item.ID == identifier {
					return toRulesStruct(&item)
				}
			}
		}
	}

	return rules.LoadRuleset(identifier)
}

func toRulesStruct(r *domain.Ruleset) (*rules.Ruleset, error) {
	bytes, err := json.Marshal(r.Scoring)
	if err != nil {
		return nil, err
	}
	var sc rules.ScoringRule
	if err := json.Unmarshal(bytes, &sc); err != nil {
		return nil, err
	}
	return &rules.Ruleset{
		Name:    r.Name,
		Version: r.Version,
		Scoring: sc,
	}, nil
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
