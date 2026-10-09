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
	// started_at diisi sekali saat partai mulai berjalan (tidak ditimpa bila dilanjutkan/dipulihkan).
	_, err := s.DB.Exec(`UPDATE matches
		SET status = $1::text,
		    started_at = CASE WHEN $1::text = 'running' THEN COALESCE(started_at, NOW()) ELSE started_at END
		WHERE id = $2`, status, id)
	return err
}

func (s *MatchStore) SetWinner(matchID, winnerID string) error {
	_, err := s.DB.Exec(`UPDATE matches SET winner_id = $1, status = 'finished', finished_at = NOW() WHERE id = $2`, winnerID, matchID)
	return err
}

const matchColumns = `id, competition_id, arena_id, bracket_slot_id, athlete_red_id, athlete_blue_id, current_round, status, winner_id, created_at, started_at, finished_at`

func scanMatches(rows *sql.Rows) ([]Match, error) {
	list := []Match{}
	for rows.Next() {
		var m Match
		var arenaID, bracketSlotID, winnerID sql.NullString
		if err := rows.Scan(
			&m.ID, &m.CompetitionID, &arenaID, &bracketSlotID,
			&m.AthleteRedID, &m.AthleteBlueID, &m.CurrentRound,
			&m.Status, &winnerID, &m.CreatedAt, &m.StartedAt, &m.FinishedAt,
		); err != nil {
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
		list = append(list, m)
	}
	return list, rows.Err()
}

func (s *MatchStore) ListByCompetitionPaged(competitionID, status string, limit, offset int) ([]Match, int, error) {
	var total int
	countQuery := `SELECT COUNT(*) FROM matches WHERE competition_id = $1 AND ($2 = '' OR status = $2)`
	if err := s.DB.QueryRow(countQuery, competitionID, status).Scan(&total); err != nil {
		return nil, 0, err
	}

	rows, err := s.DB.Query(`
		SELECT `+matchColumns+`
		FROM matches
		WHERE competition_id = $1 AND ($2 = '' OR status = $2)
		ORDER BY created_at DESC
		LIMIT $3 OFFSET $4`, competitionID, status, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	list, err := scanMatches(rows)
	return list, total, err
}

// ListByArenaPaged daftar partai pada satu gelanggang dalam urutan antrean (paling awal dibuat lebih dulu).
func (s *MatchStore) ListByArenaPaged(competitionID, arenaID, status string, limit, offset int) ([]Match, int, error) {
	var total int
	countQuery := `SELECT COUNT(*) FROM matches WHERE competition_id = $1 AND arena_id = $2 AND ($3 = '' OR status = $3)`
	if err := s.DB.QueryRow(countQuery, competitionID, arenaID, status).Scan(&total); err != nil {
		return nil, 0, err
	}

	rows, err := s.DB.Query(`
		SELECT `+matchColumns+`
		FROM matches
		WHERE competition_id = $1 AND arena_id = $2 AND ($3 = '' OR status = $3)
		ORDER BY created_at ASC, id ASC
		LIMIT $4 OFFSET $5`, competitionID, arenaID, status, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	list, err := scanMatches(rows)
	return list, total, err
}

func (s *MatchStore) ListByCompetition(competitionID string) ([]Match, error) {
	list, _, err := s.ListByCompetitionPaged(competitionID, "", 10000, 0)
	return list, err
}

// MatchInfo data tampilan sebuah partai untuk scoreboard publik.
type MatchInfo struct {
	RedName, BlueName   string
	RedTeam, BlueTeam   string
	WeightClass, Gender string
	CompetitionName     string
	// Number nomor urut partai pada antrean gelanggangnya (paling awal dibuat = 1)
	Number int
}

// GetMatchInfo mengambil nama atlet, kontingen, kelas, nama kejuaraan, dan nomor partai.
func (s *MatchStore) GetMatchInfo(matchID string) (MatchInfo, error) {
	var info MatchInfo
	var rn, bn, rt, bt, wc, g, cn sql.NullString
	err := s.DB.QueryRow(`
		SELECT ra.name, ba.name, ra.team, ba.team,
		       COALESCE(rp.weight_class, bp.weight_class), COALESCE(rp.gender, bp.gender),
		       c.name,
		       (SELECT COUNT(*) FROM matches m2
		         WHERE m2.competition_id = m.competition_id
		           AND m2.arena_id IS NOT DISTINCT FROM m.arena_id
		           AND (m2.created_at, m2.id) <= (m.created_at, m.id))
		FROM matches m
		JOIN competitions c ON c.id = m.competition_id
		LEFT JOIN participants rp ON rp.id = m.athlete_red_id
		LEFT JOIN athletes ra ON ra.id = rp.athlete_id
		LEFT JOIN participants bp ON bp.id = m.athlete_blue_id
		LEFT JOIN athletes ba ON ba.id = bp.athlete_id
		WHERE m.id = $1`, matchID).Scan(&rn, &bn, &rt, &bt, &wc, &g, &cn, &info.Number)
	info.RedName, info.BlueName = rn.String, bn.String
	info.RedTeam, info.BlueTeam = rt.String, bt.String
	info.WeightClass, info.Gender = wc.String, g.String
	info.CompetitionName = cn.String
	return info, err
}

// GetCornerNames mengambil nama atlet sudut merah dan biru untuk sebuah pertandingan.
func (s *MatchStore) GetCornerNames(matchID string) (red, blue string, err error) {
	var r, b sql.NullString
	err = s.DB.QueryRow(`
		SELECT ra.name, ba.name
		FROM matches m
		LEFT JOIN participants rp ON rp.id = m.athlete_red_id
		LEFT JOIN athletes ra ON ra.id = rp.athlete_id
		LEFT JOIN participants bp ON bp.id = m.athlete_blue_id
		LEFT JOIN athletes ba ON ba.id = bp.athlete_id
		WHERE m.id = $1`, matchID).Scan(&r, &b)
	return r.String, b.String, err
}

// Reschedule mengembalikan partai berstatus cancelled ke idle dan menghapus skor per ronde,
// sehingga partai dapat dimuat ke gelanggang dan dimulai lagi dari 0-0. Mengembalikan false
// bila partai tidak berstatus cancelled (tidak ada yang diubah).
func (s *MatchStore) Reschedule(id string) (bool, error) {
	tx, err := s.DB.Begin()
	if err != nil {
		return false, err
	}
	defer tx.Rollback()

	res, err := tx.Exec(`UPDATE matches
		SET status = 'idle', current_round = 1, winner_id = NULL, started_at = NULL, finished_at = NULL
		WHERE id = $1 AND status = 'cancelled'`, id)
	if err != nil {
		return false, err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return false, nil
	}
	if _, err := tx.Exec(`DELETE FROM match_scores WHERE match_id = $1`, id); err != nil {
		return false, err
	}
	return true, tx.Commit()
}

// MedalPlacement satu medali yang diperoleh sebuah tim dari hasil bagan.
type MedalPlacement struct {
	Team  string
	Medal string // gold | silver | bronze
}

// MedalPlacements menurunkan medali dari struktur bagan: pemenang final emas, yang kalah di final perak, dan yang
// kalah di semifinal (partai yang pemenangnya melaju ke final) perunggu. Hanya partai berbagan yang selesai dan
// berpemenang yang dihitung; babak sebelum semifinal tidak memberi medali. Tim diambil dari atlet milik peserta
// (match menyimpan ID peserta, bukan ID atlet).
func (s *MatchStore) MedalPlacements(competitionID string) ([]MedalPlacement, error) {
	rows, err := s.DB.Query(`
		WITH decided AS (
			SELECT m.winner_id AS winner,
			       CASE WHEN m.winner_id = m.athlete_red_id THEN m.athlete_blue_id ELSE m.athlete_red_id END AS loser,
			       CASE WHEN sl.next_slot_id IS NULL THEN 'final'
			            WHEN nx.next_slot_id IS NULL THEN 'semi' END AS stage
			FROM matches m
			JOIN bracket_slots sl ON sl.id = m.bracket_slot_id
			LEFT JOIN bracket_slots nx ON nx.id = sl.next_slot_id
			WHERE m.competition_id = $1 AND m.status = 'finished' AND m.winner_id IS NOT NULL
		), placed AS (
			SELECT winner AS participant, 'gold' AS medal FROM decided WHERE stage = 'final'
			UNION ALL SELECT loser, 'silver' FROM decided WHERE stage = 'final'
			UNION ALL SELECT loser, 'bronze' FROM decided WHERE stage = 'semi'
		)
		SELECT COALESCE(NULLIF(a.team, ''), 'Independen'), pl.medal
		FROM placed pl
		JOIN participants p ON p.id = pl.participant
		JOIN athletes a ON a.id = p.athlete_id`, competitionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []MedalPlacement
	for rows.Next() {
		var mp MedalPlacement
		if err := rows.Scan(&mp.Team, &mp.Medal); err != nil {
			return nil, err
		}
		out = append(out, mp)
	}
	return out, rows.Err()
}
