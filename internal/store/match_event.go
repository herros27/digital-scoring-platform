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

// GetByMatch mengembalikan log percobaan partai yang berlaku. Kejadian sebelum penjadwalan ulang
// terakhir (`match_rescheduled`) tetap tersimpan di tabel untuk audit tetapi tidak ikut ditampilkan.
func (s *MatchEventStore) GetByMatch(matchID string) ([]MatchEvent, error) {
	rows, err := s.DB.Query(`SELECT id, match_id, event_type, payload, judge_id, created_at FROM match_events WHERE match_id = $1
		AND created_at >= COALESCE((SELECT MAX(created_at) FROM match_events WHERE match_id = $1 AND event_type = 'match_rescheduled'), '-infinity'::timestamptz)
		ORDER BY created_at ASC`, matchID)
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
