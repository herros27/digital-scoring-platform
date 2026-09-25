package store

import "database/sql"


type CategoryStore struct {
	DB *sql.DB
}

func (s *CategoryStore) Create(c *Category) error {
	query := `INSERT INTO categories (competition_id, name, gender, weight_class, ruleset_id) VALUES ($1, $2, $3, $4, $5) RETURNING id, created_at`
	return s.DB.QueryRow(query, c.CompetitionID, c.Name, c.Gender, c.WeightClass, c.RulesetID).Scan(&c.ID, &c.CreatedAt)
}

func (s *CategoryStore) GetByID(id string) (*Category, error) {
	c := &Category{}
	err := s.DB.QueryRow(`SELECT id, competition_id, name, gender, weight_class, ruleset_id, created_at FROM categories WHERE id = $1`, id).Scan(
		&c.ID, &c.CompetitionID, &c.Name, &c.Gender, &c.WeightClass, &c.RulesetID, &c.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return c, nil
}

func (s *CategoryStore) ListByCompetition(competitionID string) ([]Category, error) {
	rows, err := s.DB.Query(`SELECT id, competition_id, name, gender, weight_class, ruleset_id, created_at FROM categories WHERE competition_id = $1 ORDER BY created_at`, competitionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []Category
	for rows.Next() {
		var c Category
		rows.Scan(&c.ID, &c.CompetitionID, &c.Name, &c.Gender, &c.WeightClass, &c.RulesetID, &c.CreatedAt)
		list = append(list, c)
	}
	return list, rows.Err()
}

