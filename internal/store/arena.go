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
	rows, err := s.DB.Query(`SELECT id, competition_id, name, created_at FROM arenas WHERE competition_id = $1 ORDER BY name ASC, id ASC`, competitionID)
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

func (s *ArenaStore) GetByID(id string) (*Arena, error) {
	a := &Arena{}
	query := `SELECT id, competition_id, name, created_at FROM arenas WHERE id = $1`
	err := s.DB.QueryRow(query, id).Scan(&a.ID, &a.CompetitionID, &a.Name, &a.CreatedAt)
	if err != nil {
		return nil, err
	}
	return a, nil
}

func (s *ArenaStore) Update(a *Arena) error {
	query := `UPDATE arenas SET name = $1 WHERE id = $2`
	_, err := s.DB.Exec(query, a.Name, a.ID)
	return err
}

func (s *ArenaStore) Delete(id string) error {
	// Clean up foreign keys before removing arena
	_, _ = s.DB.Exec(`DELETE FROM judge_pins WHERE arena_id = $1`, id)
	_, _ = s.DB.Exec(`UPDATE matches SET arena_id = NULL WHERE arena_id = $1`, id)

	query := `DELETE FROM arenas WHERE id = $1`
	_, err := s.DB.Exec(query, id)
	return err
}

// InUse true bila gelanggang masih menjadi gelanggang asal sebuah kategori atau masih dirujuk partai.
func (s *ArenaStore) InUse(id string) (bool, error) {
	var used bool
	err := s.DB.QueryRow(`SELECT EXISTS (SELECT 1 FROM categories WHERE home_arena_id = $1)
		OR EXISTS (SELECT 1 FROM matches WHERE arena_id = $1)`, id).Scan(&used)
	return used, err
}
