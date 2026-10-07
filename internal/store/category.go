package store

import (
	"database/sql"

	"github.com/herros27/digital-scoring/internal/domain"
)

type CategoryStore struct {
	DB *sql.DB
}

const categoryColumns = `id, competition_id, name, gender, weight_class, ruleset_id, age_group, preset_standard, min_weight, max_weight, bracket_mode, created_at`

type rowScanner interface {
	Scan(dest ...any) error
}

func scanCategory(row rowScanner, c *Category) error {
	var minW, maxW sql.NullFloat64
	var weightClass sql.NullString
	if err := row.Scan(&c.ID, &c.CompetitionID, &c.Name, &c.Gender, &weightClass, &c.RulesetID,
		&c.AgeGroup, &c.PresetStandard, &minW, &maxW, &c.BracketMode, &c.CreatedAt); err != nil {
		return err
	}
	c.WeightClass = weightClass.String
	c.MinWeight, c.MaxWeight = nil, nil
	if minW.Valid {
		v := minW.Float64
		c.MinWeight = &v
	}
	if maxW.Valid {
		v := maxW.Float64
		c.MaxWeight = &v
	}
	return nil
}

func (s *CategoryStore) Create(c *Category) error {
	if c.AgeGroup == "" {
		c.AgeGroup = domain.AgeDewasa
	}
	if c.PresetStandard == "" {
		c.PresetStandard = domain.PresetIPSI
	}
	if c.BracketMode == "" {
		c.BracketMode = domain.BracketSingleElim
	}
	query := `INSERT INTO categories (competition_id, name, gender, weight_class, ruleset_id, age_group, preset_standard, min_weight, max_weight, bracket_mode)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10) RETURNING id, created_at`
	return s.DB.QueryRow(query, c.CompetitionID, c.Name, c.Gender, c.WeightClass, c.RulesetID,
		c.AgeGroup, c.PresetStandard, c.MinWeight, c.MaxWeight, c.BracketMode).Scan(&c.ID, &c.CreatedAt)
}

func (s *CategoryStore) GetByID(id string) (*Category, error) {
	c := &Category{}
	if err := scanCategory(s.DB.QueryRow(`SELECT `+categoryColumns+` FROM categories WHERE id = $1`, id), c); err != nil {
		return nil, err
	}
	return c, nil
}

func (s *CategoryStore) ListByCompetition(competitionID string) ([]Category, error) {
	rows, err := s.DB.Query(`SELECT `+categoryColumns+` FROM categories WHERE competition_id = $1 ORDER BY created_at`, competitionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []Category
	for rows.Next() {
		var c Category
		if err := scanCategory(rows, &c); err != nil {
			return nil, err
		}
		list = append(list, c)
	}
	return list, rows.Err()
}
