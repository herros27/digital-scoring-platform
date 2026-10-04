package store

import "database/sql"

type MatchEventStore struct {
	DB *sql.DB
}

func (s *MatchEventStore) Append(e *MatchEvent) error {
	query := `
		INSERT INTO match_events (match_id, event_type, payload, judge_id)
		VALUES ($1, $2, $3, $4)
		RETURNING id, created_at`
	return s.DB.QueryRow(query, e.MatchID, e.EventType, e.Payload, e.JudgeID).
		Scan(&e.ID, &e.CreatedAt)
}

func (s *MatchEventStore) GetByMatch(matchID string) ([]MatchEvent, error) {
	rows, err := s.DB.Query(`SELECT id, match_id, event_type, payload, judge_id, created_at FROM match_events WHERE match_id = $1 ORDER BY created_at ASC`, matchID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []MatchEvent
	for rows.Next() {
		var e MatchEvent
		rows.Scan(&e.ID, &e.MatchID, &e.EventType, &e.Payload, &e.JudgeID, &e.CreatedAt)
		list = append(list, e)
	}
	return list, rows.Err()
}
