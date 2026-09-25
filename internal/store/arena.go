package store

import "database/sql"

type ArenaStore struct {
	DB *sql.DB
}

func (s *ArenaStore) Create(a *Arena) error {
	query := `INSERT INTO arenas (competition_id, name) VALUES ($1, $2) RETURNING id, created_at`
	return s.DB.QueryRow(query, a.CompetitionID, a.Name).Scan(&a.ID, &a.CreatedAt)
}

func (s *ArenaStore) ListByCompetition(competitionID string) ([]Arena, error) {
	rows, err := s.DB.Query(`SELECT id, competition_id, name, created_at FROM arenas WHERE competition_id = $1`, competitionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []Arena
	for rows.Next() {
		var a Arena
		rows.Scan(&a.ID, &a.CompetitionID, &a.Name, &a.CreatedAt)
		list = append(list, a)
	}
	return list, rows.Err()
}