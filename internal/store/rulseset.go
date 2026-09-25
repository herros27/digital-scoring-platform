package store


import (
	"database/sql"
	// "fmt"
	// "time"
)


// ---------- Ruleset ----------

type RulesetStore struct {
	DB *sql.DB
}

func (s *RulesetStore) Create(r *Ruleset) error {
	query := `INSERT INTO rulesets (name, version, scoring, status) VALUES ($1, $2, $3, $4) RETURNING id, created_at`
	return s.DB.QueryRow(query, r.Name, r.Version, r.Scoring, r.Status).Scan(&r.ID, &r.CreatedAt)
}

func (s *RulesetStore) GetByID(id string) (*Ruleset, error) {
	r := &Ruleset{}
	err := s.DB.QueryRow(`SELECT id, name, version, scoring, status, created_at FROM rulesets WHERE id = $1`, id).Scan(
		&r.ID, &r.Name, &r.Version, &r.Scoring, &r.Status, &r.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return r, nil
}

func (s *RulesetStore) List() ([]Ruleset, error) {
	rows, err := s.DB.Query(`SELECT id, name, version, scoring, status, created_at FROM rulesets ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []Ruleset
	for rows.Next() {
		var r Ruleset
		rows.Scan(&r.ID, &r.Name, &r.Version, &r.Scoring, &r.Status, &r.CreatedAt)
		list = append(list, r)
	}
	return list, rows.Err()
}

func (s *RulesetStore) GetActive() (*Ruleset, error) {
	r := &Ruleset{}
	err := s.DB.QueryRow(`SELECT id, name, version, scoring, status, created_at FROM rulesets WHERE status = 'active' ORDER BY created_at DESC LIMIT 1`).Scan(
		&r.ID, &r.Name, &r.Version, &r.Scoring, &r.Status, &r.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return r, nil
}

// ---------- Match with state ----------

func (s *MatchStore) UpdateState(id, status string, round int) error {
	_, err := s.DB.Exec(`UPDATE matches SET status = $1, current_round = $2, updated_at = NOW() WHERE id = $3`, status, round, id)
	return err
}