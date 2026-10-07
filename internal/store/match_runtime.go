package store

import (
	"database/sql"
	"time"
)

// MatchRuntime keadaan sebuah partai aktif yang disimpan untuk pemulihan setelah server restart.
type MatchRuntime struct {
	MatchID          string
	ArenaID          string
	State            string
	TimerRemainingMS int64
	TimerRunning     bool
	Data             []byte // JSON: skor, sanksi, hitungan teknik, keputusan, ruleset
	UpdatedAt        time.Time
}

type MatchRuntimeStore struct {
	DB *sql.DB
}

// Save menyimpan (upsert) keadaan terbaru sebuah partai.
func (s *MatchRuntimeStore) Save(r *MatchRuntime) error {
	_, err := s.DB.Exec(`
		INSERT INTO match_runtime (match_id, arena_id, state, timer_remaining_ms, timer_running, data, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, NOW())
		ON CONFLICT (match_id) DO UPDATE SET
			arena_id = EXCLUDED.arena_id,
			state = EXCLUDED.state,
			timer_remaining_ms = EXCLUDED.timer_remaining_ms,
			timer_running = EXCLUDED.timer_running,
			data = EXCLUDED.data,
			updated_at = NOW()`,
		r.MatchID, r.ArenaID, r.State, r.TimerRemainingMS, r.TimerRunning, r.Data)
	return err
}

// Delete menghapus keadaan sebuah partai (selesai, dibatalkan, atau dijadwalkan ulang).
func (s *MatchRuntimeStore) Delete(matchID string) error {
	_, err := s.DB.Exec(`DELETE FROM match_runtime WHERE match_id = $1`, matchID)
	return err
}

// List mengembalikan semua partai aktif yang tersimpan (untuk dipulihkan saat server start).
func (s *MatchRuntimeStore) List() ([]MatchRuntime, error) {
	rows, err := s.DB.Query(`
		SELECT match_id, arena_id, state, timer_remaining_ms, timer_running, data, updated_at
		FROM match_runtime ORDER BY updated_at`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []MatchRuntime
	for rows.Next() {
		var r MatchRuntime
		if err := rows.Scan(&r.MatchID, &r.ArenaID, &r.State, &r.TimerRemainingMS, &r.TimerRunning, &r.Data, &r.UpdatedAt); err != nil {
			return nil, err
		}
		list = append(list, r)
	}
	return list, rows.Err()
}
