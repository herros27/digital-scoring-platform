package store

import "database/sql"

type CompetitionStore struct {
	DB *sql.DB
}

func (s *CompetitionStore) Create(c *Competition) error {
	query := `
		INSERT INTO competitions (name, location, start_date, end_date, ruleset_name, status)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at`
	return s.DB.QueryRow(query, c.Name, c.Location, c.StartDate, c.EndDate, c.RulesetName, c.Status).
		Scan(&c.ID, &c.CreatedAt)
}

func (s *CompetitionStore) GetByID(id string) (*Competition, error) {
	c := &Competition{}
	query := `SELECT id, name, location, start_date, end_date, ruleset_name, status, created_at FROM competitions WHERE id = $1`
	err := s.DB.QueryRow(query, id).Scan(
		&c.ID, &c.Name, &c.Location, &c.StartDate, &c.EndDate, &c.RulesetName, &c.Status, &c.CreatedAt,
	)
	return c, err
}

func (s *CompetitionStore) List() ([]Competition, error) {
	rows, err := s.DB.Query(`SELECT id, name, location, start_date, end_date, ruleset_name, status, created_at FROM competitions ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []Competition
	for rows.Next() {
		var c Competition
		rows.Scan(&c.ID, &c.Name, &c.Location, &c.StartDate, &c.EndDate, &c.RulesetName, &c.Status, &c.CreatedAt)
		list = append(list, c)
	}
	return list, rows.Err()
}

func (s *CompetitionStore) ListPaged(status string, limit, offset int) ([]Competition, int, error) {
	var total int
	countQuery := `SELECT COUNT(*) FROM competitions WHERE ($1 = '' OR status = $1)`
	if err := s.DB.QueryRow(countQuery, status).Scan(&total); err != nil {
		return nil, 0, err
	}

	query := `
		SELECT id, name, location, start_date, end_date, ruleset_name, status, created_at 
		FROM competitions 
		WHERE ($1 = '' OR status = $1) 
		ORDER BY created_at DESC 
		LIMIT $2 OFFSET $3`
	rows, err := s.DB.Query(query, status, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var list []Competition
	for rows.Next() {
		var c Competition
		if err := rows.Scan(&c.ID, &c.Name, &c.Location, &c.StartDate, &c.EndDate, &c.RulesetName, &c.Status, &c.CreatedAt); err != nil {
			return nil, 0, err
		}
		list = append(list, c)
	}
	if list == nil {
		list = []Competition{}
	}
	return list, total, rows.Err()
}

func (s *CompetitionStore) UpdateStatus(id, status string) error {
	_, err := s.DB.Exec(`UPDATE competitions SET status = $1 WHERE id = $2`, status, id)
	return err
}

func (s *CompetitionStore) Update(c *Competition) error {
	query := `
		UPDATE competitions 
		SET name = $1, location = $2, start_date = $3, end_date = $4, ruleset_name = $5 
		WHERE id = $6`
	_, err := s.DB.Exec(query, c.Name, c.Location, c.StartDate, c.EndDate, c.RulesetName, c.ID)
	return err
}