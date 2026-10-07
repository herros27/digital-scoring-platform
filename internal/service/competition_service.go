package service

import (
	"errors"
	"fmt"
	"regexp"
	"time"

	"github.com/herros27/digital-scoring/internal/domain"
)

// Repository interface yang dibutuhkan oleh CompetitionService
type CompetitionRepository interface {
	Create(c *domain.Competition) error
	GetByID(id string) (*domain.Competition, error)
	List() ([]domain.Competition, error)
	UpdateStatus(id, status string) error
	Update(c *domain.Competition) error
}

type CompetitionService struct {
	repo CompetitionRepository
}

func NewCompetitionService(repo CompetitionRepository) *CompetitionService {
	return &CompetitionService{repo: repo}
}

type CompetitionPagedRepository interface {
	ListPaged(status string, limit, offset int) ([]domain.Competition, int, error)
}

func (s *CompetitionService) GetAllCompetitions() ([]domain.Competition, error) {
	return s.repo.List()
}

func (s *CompetitionService) GetCompetitionsPaged(status string, page, limit int) ([]domain.Competition, int, error) {
	if page <= 0 {
		page = 1
	}
	if limit <= 0 {
		limit = 20
	}
	offset := (page - 1) * limit

	if pagedRepo, ok := s.repo.(CompetitionPagedRepository); ok {
		return pagedRepo.ListPaged(status, limit, offset)
	}

	all, err := s.repo.List()
	if err != nil {
		return nil, 0, err
	}

	var filtered []domain.Competition
	for _, c := range all {
		if status == "" || c.Status == status {
			filtered = append(filtered, c)
		}
	}
	total := len(filtered)
	if offset >= total {
		return []domain.Competition{}, total, nil
	}
	end := offset + limit
	if end > total {
		end = total
	}
	return filtered[offset:end], total, nil
}

func (s *CompetitionService) GetCompetitionByID(id string) (*domain.Competition, error) {
	if id == "" {
		return nil, domain.ErrInvalidInput
	}
	comp, err := s.repo.GetByID(id)
	if err != nil {
		return nil, err
	}
	if comp == nil {
		return nil, domain.ErrCompetitionNotFound
	}
	return comp, nil
}

var uuidRE = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

// validRulesetID: nil/kosong berarti memakai ruleset berdasarkan nama; selain itu harus berupa UUID.
func validRulesetID(id *string) error {
	if id == nil || *id == "" || uuidRE.MatchString(*id) {
		return nil
	}
	return fmt.Errorf("%w: ruleset_id harus berupa UUID ruleset", domain.ErrInvalidInput)
}

func (s *CompetitionService) CreateCompetition(c *domain.Competition) error {
	if c.Name == "" || c.StartDate == "" || c.EndDate == "" {
		return errors.New("competition name, start date, and end date are required")
	}

	if err := validRulesetID(c.RulesetID); err != nil {
		return err
	}

	// Default status selalu draft saat pertama dibuat
	c.Status = "draft"
	c.CreatedAt = time.Now()

	return s.repo.Create(c)
}

func (s *CompetitionService) ActivateCompetition(id string) error {
	comp, err := s.repo.GetByID(id)
	if err != nil {
		return err
	}
	if comp == nil {
		return domain.ErrCompetitionNotFound
	}
	if comp.Status == "active" {
		return domain.ErrAlreadyActive
	}
	return s.repo.UpdateStatus(id, "active")
}

// MatchActivityChecker kemampuan opsional repo kejuaraan untuk mengetahui apakah sudah ada partai berjalan.
type MatchActivityChecker interface {
	HasStartedMatches(competitionID string) (bool, error)
}

func (s *CompetitionService) UpdateCompetition(id string, update *domain.Competition) (*domain.Competition, error) {
	if id == "" {
		return nil, domain.ErrInvalidInput
	}
	comp, err := s.repo.GetByID(id)
	if err != nil {
		return nil, err
	}
	if comp == nil {
		return nil, domain.ErrCompetitionNotFound
	}

	if update.Name != "" {
		comp.Name = update.Name
	}
	if update.Location != "" {
		comp.Location = update.Location
	}
	if update.StartDate != "" {
		comp.StartDate = update.StartDate
	}
	if update.EndDate != "" {
		comp.EndDate = update.EndDate
	}
	if update.RulesetID != nil {
		if err := validRulesetID(update.RulesetID); err != nil {
			return nil, err
		}
	}

	// Aturan tidak boleh berganti di tengah turnamen: tolak bila sudah ada partai yang berjalan/selesai.
	rulesetChanged := (update.RulesetName != "" && update.RulesetName != comp.RulesetName) ||
		(update.RulesetID != nil && !sameRulesetID(update.RulesetID, comp.RulesetID))
	if rulesetChanged {
		if chk, ok := s.repo.(MatchActivityChecker); ok {
			started, err := chk.HasStartedMatches(comp.ID)
			if err != nil {
				return nil, err
			}
			if started {
				return nil, fmt.Errorf("%w: ruleset tidak dapat diganti karena sudah ada pertandingan yang berjalan atau selesai", domain.ErrConflict)
			}
		}
	}

	if update.RulesetName != "" {
		comp.RulesetName = update.RulesetName
	}
	if update.RulesetID != nil {
		if *update.RulesetID == "" {
			comp.RulesetID = nil
		} else {
			comp.RulesetID = update.RulesetID
		}
	}

	if err := s.repo.Update(comp); err != nil {
		return nil, err
	}
	return comp, nil
}

func (s *CompetitionService) PauseCompetition(id string) error {
	comp, err := s.repo.GetByID(id)
	if err != nil {
		return err
	}
	if comp == nil {
		return domain.ErrCompetitionNotFound
	}
	return s.repo.UpdateStatus(id, "paused")
}

func (s *CompetitionService) FinishCompetition(id string) error {
	comp, err := s.repo.GetByID(id)
	if err != nil {
		return err
	}
	if comp == nil {
		return domain.ErrCompetitionNotFound
	}
	return s.repo.UpdateStatus(id, "completed")
}

func (s *CompetitionService) CancelCompetition(id string) error {
	comp, err := s.repo.GetByID(id)
	if err != nil {
		return err
	}
	if comp == nil {
		return domain.ErrCompetitionNotFound
	}
	return s.repo.UpdateStatus(id, "cancelled")
}

func sameRulesetID(a, b *string) bool {
	av, bv := "", ""
	if a != nil {
		av = *a
	}
	if b != nil {
		bv = *b
	}
	return av == bv
}
