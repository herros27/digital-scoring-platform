package store

import "database/sql"

// UpcomingRow satu baris jadwal publik: gelanggang pada kejuaraan aktif beserta satu partai
// yang masih menunggu (MatchID kosong bila gelanggang belum punya partai menunggu).
type UpcomingRow struct {
	CompetitionID, CompetitionName, Location, StartDate, EndDate string
	ArenaID, ArenaName                                           string
	MatchID                                                      string
	Number                                                       int
	RedName, BlueName, RedTeam, BlueTeam                         string
	WeightClass, Gender                                          string
}

// ListUpcoming mengembalikan gelanggang semua kejuaraan berstatus active berikut partai berstatus
// idle pada antreannya (urut nomor partai). Nomor partai sama dengan GetMatchInfo: urutan dibuat
// dalam gelanggang, termasuk partai yang sudah selesai.
func (s *MatchStore) ListUpcoming() ([]UpcomingRow, error) {
	rows, err := s.DB.Query(`
		WITH numbered AS (
			SELECT m.id, m.arena_id, m.status, m.athlete_red_id, m.athlete_blue_id,
			       ROW_NUMBER() OVER (PARTITION BY m.competition_id, m.arena_id ORDER BY m.created_at, m.id) AS number
			FROM matches m
			JOIN competitions c ON c.id = m.competition_id
			WHERE c.status = 'active'
		)
		SELECT c.id, c.name, COALESCE(c.location, ''), COALESCE(c.start_date::text, ''), COALESCE(c.end_date::text, ''),
		       a.id, a.name,
		       COALESCE(n.id::text, ''), COALESCE(n.number, 0),
		       COALESCE(ra.name, ''), COALESCE(ba.name, ''), COALESCE(ra.team, ''), COALESCE(ba.team, ''),
		       COALESCE(rp.weight_class, bp.weight_class, ''), COALESCE(rp.gender, bp.gender, '')
		FROM competitions c
		JOIN arenas a ON a.competition_id = c.id
		LEFT JOIN numbered n ON n.arena_id = a.id AND n.status = 'idle'
		LEFT JOIN participants rp ON rp.id = n.athlete_red_id
		LEFT JOIN athletes ra ON ra.id = rp.athlete_id
		LEFT JOIN participants bp ON bp.id = n.athlete_blue_id
		LEFT JOIN athletes ba ON ba.id = bp.athlete_id
		WHERE c.status = 'active'
		ORDER BY c.created_at DESC, c.id, a.name, a.id, n.number`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanUpcoming(rows)
}

func scanUpcoming(rows *sql.Rows) ([]UpcomingRow, error) {
	var list []UpcomingRow
	for rows.Next() {
		var r UpcomingRow
		if err := rows.Scan(&r.CompetitionID, &r.CompetitionName, &r.Location, &r.StartDate, &r.EndDate,
			&r.ArenaID, &r.ArenaName, &r.MatchID, &r.Number,
			&r.RedName, &r.BlueName, &r.RedTeam, &r.BlueTeam, &r.WeightClass, &r.Gender); err != nil {
			return nil, err
		}
		list = append(list, r)
	}
	return list, rows.Err()
}
