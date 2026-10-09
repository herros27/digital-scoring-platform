package store

import (
	"database/sql"
	"errors"
	"fmt"

	"github.com/herros27/digital-scoring/internal/domain"
)

type CategoryStore struct {
	DB *sql.DB
}

const categoryColumns = `id, competition_id, name, gender, weight_class, ruleset_id, age_group, preset_standard, min_weight, max_weight, bracket_mode, bracket_locked_at, home_arena_id, created_at`

type rowScanner interface {
	Scan(dest ...any) error
}

func scanCategory(row rowScanner, c *Category) error {
	var minW, maxW sql.NullFloat64
	var weightClass sql.NullString
	var lockedAt sql.NullTime
	var homeArena sql.NullString
	if err := row.Scan(&c.ID, &c.CompetitionID, &c.Name, &c.Gender, &weightClass, &c.RulesetID,
		&c.AgeGroup, &c.PresetStandard, &minW, &maxW, &c.BracketMode, &lockedAt, &homeArena, &c.CreatedAt); err != nil {
		return err
	}
	c.HomeArenaID = nil
	if homeArena.Valid {
		v := homeArena.String
		c.HomeArenaID = &v
	}
	c.BracketLockedAt, c.BracketLocked = nil, lockedAt.Valid
	if lockedAt.Valid {
		t := lockedAt.Time
		c.BracketLockedAt = &t
	}
	c.WeightClass = weightClass.String
	c.MinWeight, c.MaxWeight = nil, nil
	if minW.Valid {
		v := minW.Float64
		c.MinWeight = &v
	}
	if maxW.Valid {
		v := maxW.Float64
		c.MaxWeight = &v
	}
	return nil
}

func (s *CategoryStore) Create(c *Category) error {
	if c.AgeGroup == "" {
		c.AgeGroup = domain.AgeDewasa
	}
	if c.PresetStandard == "" {
		c.PresetStandard = domain.PresetIPSI
	}
	if c.BracketMode == "" {
		c.BracketMode = domain.BracketSingleElim
	}
	query := `INSERT INTO categories (competition_id, name, gender, weight_class, ruleset_id, age_group, preset_standard, min_weight, max_weight, bracket_mode)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10) RETURNING id, created_at`
	return s.DB.QueryRow(query, c.CompetitionID, c.Name, c.Gender, c.WeightClass, c.RulesetID,
		c.AgeGroup, c.PresetStandard, c.MinWeight, c.MaxWeight, c.BracketMode).Scan(&c.ID, &c.CreatedAt)
}

func (s *CategoryStore) GetByID(id string) (*Category, error) {
	c := &Category{}
	if err := scanCategory(s.DB.QueryRow(`SELECT `+categoryColumns+` FROM categories WHERE id = $1`, id), c); err != nil {
		return nil, mapLookupErr(err)
	}
	return c, nil
}

func (s *CategoryStore) ListByCompetition(competitionID string) ([]Category, error) {
	rows, err := s.DB.Query(`SELECT `+categoryColumns+` FROM categories WHERE competition_id = $1 ORDER BY created_at`, competitionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []Category
	for rows.Next() {
		var c Category
		if err := scanCategory(rows, &c); err != nil {
			return nil, err
		}
		list = append(list, c)
	}
	return list, rows.Err()
}

// DeleteAllByCompetition menghapus seluruh kategori satu kejuaraan (reset standar).
// Ditolak dengan ErrConflict bila masih ada peserta yang terdaftar pada kategori tsb,
// agar peserta tidak kehilangan kelasnya secara diam-diam.
func (s *CategoryStore) DeleteAllByCompetition(competitionID string) (int64, error) {
	tx, err := s.DB.Begin()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	var used int
	if err := tx.QueryRow(`
		SELECT COUNT(*) FROM participants p
		JOIN categories c ON c.id = p.category_id
		WHERE c.competition_id = $1`, competitionID).Scan(&used); err != nil {
		return 0, err
	}
	if used > 0 {
		return 0, domain.ErrConflict
	}
	// Slot bagan kategori ikut dihapus; partai idle yang terhubung dibersihkan lebih dulu.
	if _, err := tx.Exec(`
		DELETE FROM matches WHERE competition_id = $1 AND status = 'idle' AND bracket_slot_id IN (
			SELECT id FROM bracket_slots WHERE competition_id = $1 AND category_id IS NOT NULL)`, competitionID); err != nil {
		return 0, err
	}
	if _, err := tx.Exec(`
		UPDATE matches SET bracket_slot_id = NULL WHERE competition_id = $1 AND bracket_slot_id IN (
			SELECT id FROM bracket_slots WHERE competition_id = $1 AND category_id IS NOT NULL)`, competitionID); err != nil {
		return 0, err
	}
	if _, err := tx.Exec(`DELETE FROM bracket_slots WHERE competition_id = $1 AND category_id IS NOT NULL`, competitionID); err != nil {
		return 0, err
	}
	res, err := tx.Exec(`DELETE FROM categories WHERE competition_id = $1`, competitionID)
	if err != nil {
		return 0, err
	}
	n, _ := res.RowsAffected()
	return n, tx.Commit()
}

// RulesetIDForSlot ruleset khusus kategori yang memiliki slot bagan ini ("" = ikut kejuaraan).
func (s *CategoryStore) RulesetIDForSlot(slotID string) (string, error) {
	var id sql.NullString
	err := s.DB.QueryRow(`
		SELECT c.ruleset_id::text FROM bracket_slots b
		JOIN categories c ON c.id = b.category_id
		WHERE b.id = $1`, slotID).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil
	}
	return id.String, err
}

// AssignRulesets menetapkan ruleset khusus per kategori / per golongan umur dalam satu transaksi.
// Ditolak (ErrConflict) bila kategori yang terdampak sudah punya partai berjalan/selesai.
func (s *CategoryStore) AssignRulesets(competitionID string, items []domain.CategoryRulesetAssignment) (int64, error) {
	tx, err := s.DB.Begin()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	var affected int64
	for _, it := range items {
		col, val := "age_group", it.AgeGroup
		if it.CategoryID != "" {
			col, val = "id", it.CategoryID
		}
		var started int
		if err := tx.QueryRow(`
			SELECT COUNT(*) FROM matches m
			JOIN bracket_slots b ON b.id = m.bracket_slot_id
			JOIN categories c ON c.id = b.category_id
			WHERE c.competition_id = $1 AND m.status <> 'idle' AND c.`+col+` = $2`, competitionID, val).Scan(&started); err != nil {
			return 0, err
		}
		if started > 0 {
			return 0, domain.ErrConflict
		}
		var rid any
		if it.RulesetID != nil && *it.RulesetID != "" {
			rid = *it.RulesetID
		}
		res, err := tx.Exec(`UPDATE categories SET ruleset_id = $1 WHERE competition_id = $2 AND `+col+` = $3`, rid, competitionID, val)
		if err != nil {
			return 0, err
		}
		n, _ := res.RowsAffected()
		affected += n
	}
	return affected, tx.Commit()
}

// SetBracketLock mengunci/membuka draw bagan sebuah kategori milik kejuaraan itu. Mengembalikan kategori terbaru,
// atau domain.ErrNotFound bila kategori bukan milik kejuaraan.
func (s *CategoryStore) SetBracketLock(competitionID, categoryID string, locked bool) (*Category, error) {
	query := `UPDATE categories SET bracket_locked_at = CASE WHEN $3 THEN COALESCE(bracket_locked_at, NOW()) ELSE NULL END
		WHERE id = $1 AND competition_id = $2 RETURNING ` + categoryColumns
	c := &Category{}
	if err := scanCategory(s.DB.QueryRow(query, categoryID, competitionID, locked), c); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return c, nil
}

// ParticipantCount jumlah peserta yang terdaftar pada kategori.
func (s *CategoryStore) ParticipantCount(categoryID string) (int, error) {
	var n int
	err := s.DB.QueryRow(`SELECT COUNT(*) FROM participants WHERE category_id = $1`, categoryID).Scan(&n)
	return n, err
}

// Update menyimpan perubahan atribut kategori (bukan ruleset dan kunci bagan; keduanya punya endpoint sendiri).
// domain.ErrNotFound bila kategori bukan milik kejuaraan itu.
func (s *CategoryStore) Update(c *Category) error {
	res, err := s.DB.Exec(`
		UPDATE categories
		SET name = $3, gender = $4, weight_class = $5, age_group = $6, min_weight = $7, max_weight = $8, bracket_mode = $9
		WHERE id = $1 AND competition_id = $2`,
		c.ID, c.CompetitionID, c.Name, c.Gender, c.WeightClass, c.AgeGroup, c.MinWeight, c.MaxWeight, c.BracketMode)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return domain.ErrNotFound
	}
	return nil
}

// Delete menghapus satu kategori beserta slot bagannya (partai idle yang terhubung ikut dihapus).
// Ditolak (ErrConflict) bila masih ada peserta pada kategori atau partainya sudah berjalan/selesai.
func (s *CategoryStore) Delete(competitionID, id string) error {
	tx, err := s.DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var exists int
	if err := tx.QueryRow(`SELECT COUNT(*) FROM categories WHERE id = $1 AND competition_id = $2`, id, competitionID).Scan(&exists); err != nil {
		return err
	}
	if exists == 0 {
		return domain.ErrNotFound
	}
	var participants int
	if err := tx.QueryRow(`SELECT COUNT(*) FROM participants WHERE category_id = $1`, id).Scan(&participants); err != nil {
		return err
	}
	if participants > 0 {
		return fmt.Errorf("%w: kategori masih memiliki %d peserta; batalkan pendaftaran mereka terlebih dahulu", domain.ErrConflict, participants)
	}
	var started int
	if err := tx.QueryRow(`
		SELECT COUNT(*) FROM matches m
		JOIN bracket_slots b ON b.id = m.bracket_slot_id
		WHERE b.category_id = $1 AND m.status <> 'idle'`, id).Scan(&started); err != nil {
		return err
	}
	if started > 0 {
		return fmt.Errorf("%w: kategori memiliki partai yang sudah berjalan atau selesai", domain.ErrConflict)
	}
	if _, err := tx.Exec(`
		DELETE FROM matches WHERE status = 'idle' AND bracket_slot_id IN (SELECT id FROM bracket_slots WHERE category_id = $1)`, id); err != nil {
		return err
	}
	if _, err := tx.Exec(`
		UPDATE matches SET bracket_slot_id = NULL WHERE bracket_slot_id IN (SELECT id FROM bracket_slots WHERE category_id = $1)`, id); err != nil {
		return err
	}
	if _, err := tx.Exec(`DELETE FROM bracket_slots WHERE category_id = $1`, id); err != nil {
		return err
	}
	if _, err := tx.Exec(`DELETE FROM categories WHERE id = $1 AND competition_id = $2`, id, competitionID); err != nil {
		return err
	}
	return tx.Commit()
}

// SetHomeArena menetapkan gelanggang asal kategori dan memindahkan partai kategori yang belum berjalan (idle) ke sana.
// Ditolak (ErrConflict) bila ada partai kategori yang sudah dimuat, berjalan, atau selesai: gelanggang asal harus tetap.
// ErrNotFound bila kategori atau gelanggang bukan milik kejuaraan itu.
func (s *CategoryStore) SetHomeArena(competitionID, categoryID, arenaID string) error {
	tx, err := s.DB.Begin()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	var ok int
	if err := tx.QueryRow(`SELECT COUNT(*) FROM categories c JOIN arenas a ON a.competition_id = c.competition_id
		WHERE c.id = $1 AND c.competition_id = $2 AND a.id = $3`, categoryID, competitionID, arenaID).Scan(&ok); err != nil {
		return err
	}
	if ok == 0 {
		return domain.ErrNotFound
	}
	var started int
	if err := tx.QueryRow(`SELECT COUNT(*) FROM matches m JOIN bracket_slots b ON b.id = m.bracket_slot_id
		WHERE b.category_id = $1 AND m.status <> 'idle' AND m.arena_id IS DISTINCT FROM $2::uuid`, categoryID, arenaID).Scan(&started); err != nil {
		return err
	}
	if started > 0 {
		return fmt.Errorf("%w: partai kategori ini sudah dimuat atau berjalan di gelanggang lain; gelanggang asal tidak dapat diubah", domain.ErrConflict)
	}
	if _, err := tx.Exec(`UPDATE categories SET home_arena_id = $2 WHERE id = $1`, categoryID, arenaID); err != nil {
		return err
	}
	if _, err := tx.Exec(`UPDATE matches m SET arena_id = $2 FROM bracket_slots b
		WHERE m.bracket_slot_id = b.id AND b.category_id = $1 AND m.status = 'idle'`, categoryID, arenaID); err != nil {
		return err
	}
	return tx.Commit()
}

// ArenaLoads perkiraan sisa partai per gelanggang dari kategori yang sudah punya gelanggang asal (peserta sah dikurangi
// satu, dikurangi partai yang sudah selesai). exceptCategoryID dikecualikan (kategori yang sedang dibuat ulang).
func (s *CategoryStore) ArenaLoads(competitionID, exceptCategoryID string) (map[string]int, error) {
	rows, err := s.DB.Query(`
		SELECT c.home_arena_id,
		       COALESCE(SUM(GREATEST(
		         (SELECT COUNT(*) FROM participants p
		           WHERE p.category_id = c.id AND p.registration_status NOT IN ('pending', 'rejected', 'overweight', 'underweight')) - 1
		         - (SELECT COUNT(*) FROM matches m JOIN bracket_slots b ON b.id = m.bracket_slot_id
		             WHERE b.category_id = c.id AND m.status = 'finished'), 0)), 0)
		FROM categories c
		WHERE c.competition_id = $1 AND c.home_arena_id IS NOT NULL AND c.id::text <> $2
		GROUP BY c.home_arena_id`, competitionID, exceptCategoryID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]int{}
	for rows.Next() {
		var id string
		var n int
		if err := rows.Scan(&id, &n); err != nil {
			return nil, err
		}
		out[id] = n
	}
	return out, rows.Err()
}
