package store

import "database/sql"

type MatchStore struct {
	DB *sql.DB
}

func (s *MatchStore) Create(m *Match) error {
	query := `
		INSERT INTO matches (competition_id, arena_id, bracket_slot_id, athlete_red_id, athlete_blue_id, current_round, status)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, created_at`

	// Jika string kosong, kirim nil agar tersimpan sebagai NULL di PostgreSQL
	var bracketSlotID any = m.BracketSlotID
	if m.BracketSlotID == "" {
		bracketSlotID = nil
	}
	var arenaID any = m.ArenaID
	if m.ArenaID == "" {
		arenaID = nil
	}
	return s.DB.QueryRow(query,
		m.CompetitionID, 
		arenaID, 
		bracketSlotID,
		m.AthleteRedID, 
		m.AthleteBlueID, 
		m.CurrentRound, 
		m.Status,
	).Scan(&m.ID, &m.CreatedAt)
}

func (s *MatchStore) GetByID(id string) (*Match, error) {
	m := &Match{}
	query := `SELECT id, competition_id, arena_id, bracket_slot_id, athlete_red_id, athlete_blue_id, current_round, status, winner_id, created_at, started_at, finished_at FROM matches WHERE id = $1`

	var arenaID, bracketSlotID, winnerID sql.NullString

	err := s.DB.QueryRow(query, id).Scan(
		&m.ID, 
		&m.CompetitionID, 
		&arenaID, 
		&bracketSlotID,
		&m.AthleteRedID, 
		&m.AthleteBlueID, 
		&m.CurrentRound, 
		&m.Status,
		&winnerID, 
		&m.CreatedAt, 
		&m.StartedAt, 
		&m.FinishedAt,
	)
	if err != nil {
		return nil, err
	}

	if arenaID.Valid {
		m.ArenaID = arenaID.String
	}
	if bracketSlotID.Valid {
		m.BracketSlotID = bracketSlotID.String
	}
	if winnerID.Valid {
		m.WinnerID = winnerID.String
	}

	return m, nil
}

func (s *MatchStore) UpdateStatus(id, status string) error {
	_, err := s.DB.Exec(`UPDATE matches SET status = $1 WHERE id = $2`, status, id)
	return err
}

func (s *MatchStore) SetWinner(matchID, winnerID string) error {
	_, err := s.DB.Exec(`UPDATE matches SET winner_id = $1, status = 'finished', finished_at = NOW() WHERE id = $2`, winnerID, matchID)
	return err
}

func (s *MatchStore) ListByCompetitionPaged(competitionID, status string, limit, offset int) ([]Match, int, error) {
	var total int
	countQuery := `SELECT COUNT(*) FROM matches WHERE competition_id = $1 AND ($2 = '' OR status = $2)`
	if err := s.DB.QueryRow(countQuery, competitionID, status).Scan(&total); err != nil {
		return nil, 0, err
	}

	query := `
		SELECT id, competition_id, arena_id, bracket_slot_id, athlete_red_id, athlete_blue_id, current_round, status, winner_id, created_at, started_at, finished_at 
		FROM matches 
		WHERE competition_id = $1 AND ($2 = '' OR status = $2)
		ORDER BY created_at DESC 
		LIMIT $3 OFFSET $4`
	rows, err := s.DB.Query(query, competitionID, status, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var list []Match
	for rows.Next() {
		var m Match
		var arenaID, bracketSlotID, winnerID sql.NullString
		if err := rows.Scan(
			&m.ID, &m.CompetitionID, &arenaID, &bracketSlotID,
			&m.AthleteRedID, &m.AthleteBlueID, &m.CurrentRound,
			&m.Status, &winnerID, &m.CreatedAt, &m.StartedAt, &m.FinishedAt,
		); err != nil {
			return nil, 0, err
		}
		if arenaID.Valid {
			m.ArenaID = arenaID.String
		}
		if bracketSlotID.Valid {
			m.BracketSlotID = bracketSlotID.String
		}
		if winnerID.Valid {
			m.WinnerID = winnerID.String
		}
		list = append(list, m)
	}
	if list == nil {
		list = []Match{}
	}
	return list, total, rows.Err()
}