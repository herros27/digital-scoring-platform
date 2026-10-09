package store

import (
	"database/sql"
	"strings"
)

// nullableID mengubah string kosong menjadi NULL untuk kolom UUID opsional.
func nullableID(id *string) any {
	if id == nil || *id == "" {
		return nil
	}
	return *id
}

// setRulesetID menerjemahkan hasil scan kolom ruleset_id (NULL -> nil).
func setRulesetID(c *Competition, v sql.NullString) {
	if v.Valid {
		id := v.String
		c.RulesetID = &id
	}
}

type CompetitionStore struct {
	DB *sql.DB
}

func (s *CompetitionStore) Create(c *Competition) error {
	query := `
		INSERT INTO competitions (name, location, start_date, end_date, ruleset_name, ruleset_id, status)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, created_at`
	return s.DB.QueryRow(query, c.Name, c.Location, c.StartDate, c.EndDate, c.RulesetName, nullableID(c.RulesetID), c.Status).
		Scan(&c.ID, &c.CreatedAt)
}

func (s *CompetitionStore) GetByID(id string) (*Competition, error) {
	c := &Competition{}
	query := `SELECT id, name, location, start_date, end_date, ruleset_name, ruleset_id, status, created_at FROM competitions WHERE id = $1`
	var rulesetID sql.NullString
	err := s.DB.QueryRow(query, id).Scan(
		&c.ID, &c.Name, &c.Location, &c.StartDate, &c.EndDate, &c.RulesetName, &rulesetID, &c.Status, &c.CreatedAt,
	)
	setRulesetID(c, rulesetID)
	return c, err
}

func (s *CompetitionStore) List() ([]Competition, error) {
	rows, err := s.DB.Query(`SELECT id, name, location, start_date, end_date, ruleset_name, ruleset_id, status, created_at FROM competitions ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []Competition
	for rows.Next() {
		var c Competition
		var rulesetID sql.NullString
		rows.Scan(&c.ID, &c.Name, &c.Location, &c.StartDate, &c.EndDate, &c.RulesetName, &rulesetID, &c.Status, &c.CreatedAt)
		setRulesetID(&c, rulesetID)
		list = append(list, c)
	}
	return list, rows.Err()
}

// ListPaged daftar kejuaraan berhalaman menurut status ("" = semua).
func (s *CompetitionStore) ListPaged(status string, limit, offset int) ([]Competition, int, error) {
	return s.ListPagedSearch(status, "", limit, offset)
}

// likePattern membungkus teks pencarian untuk ILIKE dan menetralkan karakter khusus (%, _, \).
func likePattern(q string) string {
	return "%" + strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(q) + "%"
}

// competitionFilter klausa WHERE bersama untuk hitungan dan daftar: status (opsional) dan teks pencarian pada nama,
// lokasi, nama ruleset, atau ID.
const competitionFilter = `
	WHERE ($1 = '' OR status = $1)
	  AND ($2 = '' OR name ILIKE $3 OR location ILIKE $3 OR ruleset_name ILIKE $3 OR id::text ILIKE $3)`

// ListPagedSearch daftar kejuaraan berhalaman, difilter status dan teks pencarian q. Urutan: yang sedang berlangsung
// lebih dulu (aktif, dijeda, draft, selesai, lalu dibatalkan), lalu yang terbaru, agar kejuaraan nyata tidak tertimbun
// kejuaraan uji yang dibatalkan. Total dihitung dengan filter yang sama.
func (s *CompetitionStore) ListPagedSearch(status, q string, limit, offset int) ([]Competition, int, error) {
	pattern := likePattern(q)
	var total int
	if err := s.DB.QueryRow(`SELECT COUNT(*) FROM competitions`+competitionFilter, status, q, pattern).Scan(&total); err != nil {
		return nil, 0, err
	}

	query := `
		SELECT id, name, location, start_date, end_date, ruleset_name, ruleset_id, status, created_at
		FROM competitions` + competitionFilter + `
		ORDER BY CASE status WHEN 'active' THEN 0 WHEN 'paused' THEN 1 WHEN 'draft' THEN 2 WHEN 'completed' THEN 3 ELSE 4 END,
		         created_at DESC
		LIMIT $4 OFFSET $5`
	rows, err := s.DB.Query(query, status, q, pattern, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var list []Competition
	for rows.Next() {
		var c Competition
		var rulesetID sql.NullString
		if err := rows.Scan(&c.ID, &c.Name, &c.Location, &c.StartDate, &c.EndDate, &c.RulesetName, &rulesetID, &c.Status, &c.CreatedAt); err != nil {
			return nil, 0, err
		}
		setRulesetID(&c, rulesetID)
		list = append(list, c)
	}
	if list == nil {
		list = []Competition{}
	}
	return list, total, rows.Err()
}

// CountByStatus jumlah kejuaraan per status di seluruh basis data (tidak bergantung filter atau halaman).
func (s *CompetitionStore) CountByStatus() (map[string]int, error) {
	rows, err := s.DB.Query(`SELECT status, COUNT(*) FROM competitions GROUP BY status`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]int{}
	for rows.Next() {
		var st string
		var n int
		if err := rows.Scan(&st, &n); err != nil {
			return nil, err
		}
		out[st] = n
	}
	return out, rows.Err()
}

func (s *CompetitionStore) UpdateStatus(id, status string) error {
	_, err := s.DB.Exec(`UPDATE competitions SET status = $1 WHERE id = $2`, status, id)
	return err
}

func (s *CompetitionStore) Update(c *Competition) error {
	query := `
		UPDATE competitions 
		SET name = $1, location = $2, start_date = $3, end_date = $4, ruleset_name = $5, ruleset_id = $6 
		WHERE id = $7`
	_, err := s.DB.Exec(query, c.Name, c.Location, c.StartDate, c.EndDate, c.RulesetName, nullableID(c.RulesetID), c.ID)
	return err
}

// HasStartedMatches true bila kejuaraan sudah punya partai yang berjalan atau selesai
// (status selain idle). Ruleset tidak boleh diganti sesudah itu.
func (s *CompetitionStore) HasStartedMatches(id string) (bool, error) {
	var started bool
	err := s.DB.QueryRow(`SELECT EXISTS (SELECT 1 FROM matches WHERE competition_id = $1 AND status <> 'idle')`, id).Scan(&started)
	return started, err
}
