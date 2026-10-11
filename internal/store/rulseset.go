package store

import (
	"database/sql"
	"encoding/json"
)

// ---------- Ruleset ----------

type RulesetStore struct {
	DB *sql.DB
}

func (s *RulesetStore) Create(r *Ruleset) error {
	scoringBytes, err := json.Marshal(r.Scoring)
	if err != nil {
		return err
	}
	query := `INSERT INTO rulesets (name, version, scoring, status) VALUES ($1, $2, $3, $4) RETURNING id, created_at`
	return s.DB.QueryRow(query, r.Name, r.Version, scoringBytes, r.Status).Scan(&r.ID, &r.CreatedAt)
}

func (s *RulesetStore) GetByID(id string) (*Ruleset, error) {
	r := &Ruleset{}
	var rawScoring []byte
	err := s.DB.QueryRow(`SELECT id, name, version, scoring, status, created_at FROM rulesets WHERE id = $1`, id).Scan(
		&r.ID, &r.Name, &r.Version, &rawScoring, &r.Status, &r.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	var scoringObj any
	if err := json.Unmarshal(rawScoring, &scoringObj); err == nil {
		r.Scoring = scoringObj
	} else {
		r.Scoring = string(rawScoring)
	}
	return r, nil
}

func (s *RulesetStore) Update(r *Ruleset) error {
	scoringBytes, err := json.Marshal(r.Scoring)
	if err != nil {
		return err
	}
	query := `UPDATE rulesets SET name = $1, version = $2, scoring = $3, status = $4 WHERE id = $5`
	res, err := s.DB.Exec(query, r.Name, r.Version, scoringBytes, r.Status, r.ID)
	if err != nil {
		return err
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (s *RulesetStore) Delete(id string) error {
	res, err := s.DB.Exec(`DELETE FROM rulesets WHERE id = $1`, id)
	if err != nil {
		return err
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (s *RulesetStore) List() ([]Ruleset, error) {
	rows, err := s.DB.Query(`SELECT id, name, version, scoring, status, created_at FROM rulesets ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []Ruleset
	for rows.Next() {
		var r Ruleset
		var rawScoring []byte
		if err := rows.Scan(&r.ID, &r.Name, &r.Version, &rawScoring, &r.Status, &r.CreatedAt); err != nil {
			return nil, err
		}
		var scoringObj any
		if err := json.Unmarshal(rawScoring, &scoringObj); err == nil {
			r.Scoring = scoringObj
		} else {
			r.Scoring = string(rawScoring)
		}
		list = append(list, r)
	}
	return list, rows.Err()
}

func (s *RulesetStore) GetActive() (*Ruleset, error) {
	r := &Ruleset{}
	var rawScoring []byte
	err := s.DB.QueryRow(`SELECT id, name, version, scoring, status, created_at FROM rulesets WHERE status = 'active' ORDER BY created_at DESC LIMIT 1`).Scan(
		&r.ID, &r.Name, &r.Version, &rawScoring, &r.Status, &r.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	var scoringObj any
	if err := json.Unmarshal(rawScoring, &scoringObj); err == nil {
		r.Scoring = scoringObj
	} else {
		r.Scoring = string(rawScoring)
	}
	return r, nil
}

// ---------- Match with state ----------

func (s *MatchStore) UpdateState(id, status string, round int) error {
	_, err := s.DB.Exec(`UPDATE matches SET status = $1, current_round = $2, updated_at = NOW() WHERE id = $3`, status, round, id)
	return err
}

// CompetitionsUsing label pemakai ruleset ini: nama kejuaraan yang memakainya (ruleset_id) dan, bila ada kategori yang
// terikat ke ruleset ini, "N kategori di <kejuaraan>" supaya jelas bagian mana yang masih merujuk.
func (s *RulesetStore) CompetitionsUsing(id string) ([]string, error) {
	rows, err := s.DB.Query(`
		SELECT name FROM competitions WHERE ruleset_id = $1
		UNION ALL
		SELECT COUNT(*)::text || ' kategori di ' || comp.name
		FROM categories k JOIN competitions comp ON comp.id = k.competition_id
		WHERE k.ruleset_id = $1 GROUP BY comp.name
		ORDER BY 1`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var names []string
	for rows.Next() {
		var n string
		if err := rows.Scan(&n); err != nil {
			return nil, err
		}
		names = append(names, n)
	}
	return names, rows.Err()
}
