package store

import (
	"database/sql"
)

// ---------- Athlete ----------

type AthleteStore struct {
	DB *sql.DB
}

func (s *AthleteStore) Create(a *Athlete) error {
	query := `INSERT INTO athletes (name, gender, team) VALUES ($1, $2, $3) RETURNING id, created_at`
	return s.DB.QueryRow(query, a.Name, a.Gender, a.Team).Scan(&a.ID, &a.CreatedAt)
}

func (s *AthleteStore) GetByID(id string) (*Athlete, error) {
	a := &Athlete{}
	err := s.DB.QueryRow(`SELECT id, name, gender, team, created_at FROM athletes WHERE id = $1`, id).Scan(
		&a.ID, &a.Name, &a.Gender, &a.Team, &a.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return a, nil
}

func (s *AthleteStore) ListByCompetition(competitionID string) ([]Athlete, error) {
	rows, err := s.DB.Query(`
		SELECT a.id, a.name, a.gender, a.team, a.created_at
		FROM athletes a
		JOIN participants p ON p.athlete_id = a.id
		WHERE p.competition_id = $1`, competitionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []Athlete
	for rows.Next() {
		var a Athlete
		rows.Scan(&a.ID, &a.Name, &a.Gender, &a.Team, &a.CreatedAt)
		list = append(list, a)
	}
	return list, rows.Err()
}


