package store

import (
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/herros27/digital-scoring/internal/domain"
)

// ---------- Athlete ----------

type AthleteStore struct {
	DB *sql.DB
}

// athleteColumns membaca contingent_id (” bila atlet didaftarkan admin) dan birth_date sebagai teks YYYY-MM-DD (” untuk data lama tanpa tanggal lahir).
const athleteColumns = `id, name, gender, team, COALESCE(to_char(birth_date, 'YYYY-MM-DD'), ''), COALESCE(nik, ''), COALESCE(contingent_id::text, ''), created_at`

func scanAthlete(row rowScanner, a *Athlete) error {
	return row.Scan(&a.ID, &a.Name, &a.Gender, &a.Team, &a.BirthDate, &a.NIK, &a.ContingentID, &a.CreatedAt)
}

func (s *AthleteStore) Create(a *Athlete) error {
	if a.ContingentID != "" {
		query := `INSERT INTO athletes (name, gender, team, contingent_id, birth_date, nik)
			VALUES ($1, $2, $3, $4, NULLIF($5, '')::date, NULLIF($6, '')) RETURNING id, created_at`
		return s.DB.QueryRow(query, a.Name, a.Gender, a.Team, a.ContingentID, a.BirthDate, a.NIK).Scan(&a.ID, &a.CreatedAt)
	}
	query := `INSERT INTO athletes (name, gender, team, birth_date, nik)
		VALUES ($1, $2, $3, NULLIF($4, '')::date, NULLIF($5, '')) RETURNING id, created_at`
	return s.DB.QueryRow(query, a.Name, a.Gender, a.Team, a.BirthDate, a.NIK).Scan(&a.ID, &a.CreatedAt)
}

func (s *AthleteStore) GetByID(id string) (*Athlete, error) {
	a := &Athlete{}
	err := scanAthlete(s.DB.QueryRow(`SELECT `+athleteColumns+` FROM athletes WHERE id = $1`, id), a)
	if err != nil {
		return nil, mapLookupErr(err)
	}
	return a, nil
}

func (s *AthleteStore) List() ([]Athlete, error) {
	rows, err := s.DB.Query(`SELECT ` + athleteColumns + ` FROM athletes ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []Athlete
	for rows.Next() {
		var a Athlete
		if err := scanAthlete(rows, &a); err != nil {
			return nil, err
		}
		list = append(list, a)
	}
	return list, rows.Err()
}

func (s *AthleteStore) ListPaged(gender string, limit, offset int) ([]Athlete, int, error) {
	var total int
	countQuery := `SELECT COUNT(*) FROM athletes WHERE ($1 = '' OR gender = $1)`
	if err := s.DB.QueryRow(countQuery, gender).Scan(&total); err != nil {
		return nil, 0, err
	}

	query := `
		SELECT ` + athleteColumns + `
		FROM athletes 
		WHERE ($1 = '' OR gender = $1) 
		ORDER BY name ASC 
		LIMIT $2 OFFSET $3`
	rows, err := s.DB.Query(query, gender, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var list []Athlete
	for rows.Next() {
		var a Athlete
		if err := scanAthlete(rows, &a); err != nil {
			return nil, 0, err
		}
		list = append(list, a)
	}
	if list == nil {
		list = []Athlete{}
	}
	return list, total, rows.Err()
}

func (s *AthleteStore) RegisterToCompetition(competitionID, athleteID, weightClass, gender string) error {
	return s.RegisterParticipant(competitionID, athleteID, "", weightClass, gender, "registered")
}

// RegisterParticipantInCategory mendaftarkan atlet ke kategori spesifik (category_id).
func (s *AthleteStore) RegisterParticipantInCategory(competitionID, athleteID, contingentID, categoryID, weightClass, gender, status string) error {
	if status == "" {
		status = "pending"
	}
	_, err := s.DB.Exec(`
		INSERT INTO participants (competition_id, athlete_id, contingent_id, category_id, weight_class, gender, registration_status)
		VALUES ($1, $2, NULLIF($3, '')::uuid, NULLIF($4, '')::uuid, $5, $6, $7)`,
		competitionID, athleteID, contingentID, categoryID, weightClass, gender, status)
	return mapParticipantInsertErr(err)
}

func (s *AthleteStore) RegisterParticipant(competitionID, athleteID, contingentID, weightClass, gender, status string) error {
	if status == "" {
		status = "pending"
	}
	if contingentID != "" {
		query := `INSERT INTO participants (competition_id, athlete_id, contingent_id, weight_class, gender, registration_status) VALUES ($1, $2, $3, $4, $5, $6)`
		_, err := s.DB.Exec(query, competitionID, athleteID, contingentID, weightClass, gender, status)
		return mapParticipantInsertErr(err)
	}
	query := `INSERT INTO participants (competition_id, athlete_id, weight_class, gender, registration_status) VALUES ($1, $2, $3, $4, $5)`
	_, err := s.DB.Exec(query, competitionID, athleteID, weightClass, gender, status)
	return mapParticipantInsertErr(err)
}

func (s *AthleteStore) UpdateParticipantStatus(participantID, status string) error {
	query := `UPDATE participants SET registration_status = $1 WHERE id = $2`
	res, err := s.DB.Exec(query, status, participantID)
	if err != nil {
		return mapLookupErr(err)
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return domain.ErrNotFound
	}
	return nil
}

const participantSelect = `
		SELECT p.id, p.competition_id, 
		       COALESCE(p.athlete_id::text, ''), 
		       COALESCE(p.contingent_id::text, a.contingent_id::text, ''), 
		       COALESCE(a.name, p.name, ''), 
		       COALESCE(NULLIF(a.team, ''), c.name, ''), 
		       COALESCE(p.weight_class, ''), 
		       p.gender, 
		       COALESCE(p.registration_status, 'registered'), 
		       COALESCE(p.category_id::text, ''),
		       COALESCE(cat.name, ''),
		       COALESCE(cat.age_group, ''),
		       COALESCE(cat.preset_standard, ''),
		       COALESCE(to_char(a.birth_date, 'YYYY-MM-DD'), ''),
		       COALESCE(a.nik, ''),
		       p.actual_weight,
		       COALESCE(p.docs_verified, '{}'::jsonb),
		       p.created_at
		FROM participants p
		LEFT JOIN athletes a ON p.athlete_id = a.id
		LEFT JOIN contingents c ON COALESCE(p.contingent_id, a.contingent_id) = c.id
		LEFT JOIN categories cat ON p.category_id = cat.id`

func scanParticipant(row rowScanner, part *Participant) error {
	var weight sql.NullFloat64
	var docs []byte
	if err := row.Scan(&part.ID, &part.CompetitionID, &part.AthleteID, &part.ContingentID,
		&part.Name, &part.Club, &part.WeightClass, &part.Gender, &part.RegistrationStatus,
		&part.CategoryID, &part.CategoryName, &part.AgeGroup, &part.PresetStandard, &part.BirthDate, &part.NIK,
		&weight, &docs, &part.CreatedAt); err != nil {
		return err
	}
	if weight.Valid {
		w := weight.Float64
		part.ActualWeight = &w
	}
	part.DocsVerified = map[string]bool{}
	if len(docs) > 0 {
		_ = json.Unmarshal(docs, &part.DocsVerified)
	}
	return nil
}

func (s *AthleteStore) ListParticipants(competitionID, status string) ([]Participant, error) {
	var rows *sql.Rows
	var err error
	if status != "" {
		rows, err = s.DB.Query(participantSelect+` WHERE p.competition_id = $1 AND p.registration_status = $2 ORDER BY p.created_at DESC`, competitionID, status)
	} else {
		rows, err = s.DB.Query(participantSelect+` WHERE p.competition_id = $1 ORDER BY p.created_at DESC`, competitionID)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []Participant
	for rows.Next() {
		var part Participant
		if err := scanParticipant(rows, &part); err != nil {
			return nil, err
		}
		list = append(list, part)
	}
	if list == nil {
		list = []Participant{}
	}
	return list, rows.Err()
}

// GetParticipantByID mengambil satu peserta beserta metadata kategori/atlet.
func (s *AthleteStore) GetParticipantByID(id string) (*Participant, error) {
	part := &Participant{}
	if err := scanParticipant(s.DB.QueryRow(participantSelect+` WHERE p.id = $1`, id), part); err != nil {
		return nil, mapLookupErr(err)
	}
	return part, nil
}

// SaveVerification menyimpan status, hasil timbang aktual, dan checklist berkas sekaligus.
func (s *AthleteStore) SaveVerification(participantID, status string, actualWeight *float64, docs map[string]bool) error {
	if docs == nil {
		docs = map[string]bool{}
	}
	docsJSON, err := json.Marshal(docs)
	if err != nil {
		return err
	}
	res, err := s.DB.Exec(`UPDATE participants
		SET registration_status = $1, actual_weight = COALESCE($2, actual_weight), docs_verified = $3::jsonb
		WHERE id = $4`, status, actualWeight, string(docsJSON), participantID)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (s *AthleteStore) ListByCompetition(competitionID string) ([]Athlete, error) {
	rows, err := s.DB.Query(`
		SELECT a.id, a.name, a.gender, a.team, COALESCE(to_char(a.birth_date, 'YYYY-MM-DD'), ''), COALESCE(a.nik, ''), COALESCE(a.contingent_id::text, ''), a.created_at
		FROM athletes a
		JOIN participants p ON p.athlete_id = a.id
		WHERE p.competition_id = $1 ORDER BY a.name`, competitionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []Athlete
	for rows.Next() {
		var a Athlete
		if err := scanAthlete(rows, &a); err != nil {
			return nil, err
		}
		list = append(list, a)
	}
	return list, rows.Err()
}

// UpdateProfile memperbarui tanggal lahir dan/atau NIK atlet; field nil tidak diubah. NIK "" menghapus NIK.
func (s *AthleteStore) UpdateProfile(id string, birthDate, nik *string) error {
	res, err := s.DB.Exec(`
		UPDATE athletes
		SET birth_date = CASE WHEN $2::text IS NULL THEN birth_date ELSE NULLIF($2::text, '')::date END,
		    nik        = CASE WHEN $3::text IS NULL THEN nik ELSE NULLIF($3::text, '') END
		WHERE id = $1`, id, birthDate, nik)
	if err != nil {
		return mapLookupErr(err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return domain.ErrNotFound
	}
	return nil
}

// UpdateIdentity memperbarui nama dan/atau perguruan atlet; field nil tidak diubah.
func (s *AthleteStore) UpdateIdentity(id string, name, team *string) error {
	res, err := s.DB.Exec(`UPDATE athletes SET name = COALESCE($2, name), team = COALESCE($3, team) WHERE id = $1`, id, name, team)
	if err != nil {
		return mapLookupErr(err)
	}
	if n, err := res.RowsAffected(); err != nil {
		return err
	} else if n == 0 {
		return domain.ErrNotFound
	}
	return nil
}

// DeleteAthleteIfUnused menghapus data atlet bila tidak lagi terdaftar di kejuaraan mana pun
// (atlet yang masih dipakai kejuaraan lain dibiarkan).
func (s *AthleteStore) DeleteAthleteIfUnused(id string) error {
	_, err := s.DB.Exec(`DELETE FROM athletes a WHERE a.id = $1 AND NOT EXISTS (SELECT 1 FROM participants p WHERE p.athlete_id = a.id)`, id)
	return err
}

// ChangeParticipantCategory memindahkan peserta ke kategori lain. Ditolak (ErrConflict) bila peserta sudah dirujuk bagan
// atau partai. Peserta yang sudah diverifikasi/ditolak kembali menunggu verifikasi karena batas bobot dan berkas bisa berbeda.
func (s *AthleteStore) ChangeParticipantCategory(competitionID, participantID, categoryID, weightClass string) error {
	tx, err := s.DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var refs int
	if err := tx.QueryRow(`
		SELECT (SELECT COUNT(*) FROM bracket_slots WHERE participant_id = $1 OR winner_id = $1)
		     + (SELECT COUNT(*) FROM matches WHERE athlete_red_id = $1 OR athlete_blue_id = $1 OR winner_id = $1)`, participantID).Scan(&refs); err != nil {
		return err
	}
	if refs > 0 {
		return fmt.Errorf("%w: peserta sudah masuk bagan atau partai; reset bagan kategorinya terlebih dahulu sebelum memindahkan kategori", domain.ErrConflict)
	}
	res, err := tx.Exec(`
		UPDATE participants
		SET category_id = $3::uuid, weight_class = $4,
		    registration_status = CASE WHEN registration_status IN ('registered', 'pending') THEN registration_status ELSE 'pending' END
		WHERE id = $1 AND competition_id = $2`, participantID, competitionID, categoryID, weightClass)
	if err != nil {
		return mapLookupErr(err)
	}
	if n, err := res.RowsAffected(); err != nil {
		return err
	} else if n == 0 {
		return domain.ErrNotFound
	}
	return tx.Commit()
}

// DeleteParticipant membatalkan pendaftaran satu peserta dari kejuaraan (atlet tetap tersimpan di basis data atlet).
// Ditolak (ErrConflict) bila peserta sudah dirujuk bagan atau partai; domain.ErrNotFound bila bukan peserta kejuaraan itu.
func (s *AthleteStore) DeleteParticipant(competitionID, participantID string) error {
	tx, err := s.DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var exists int
	if err := tx.QueryRow(`SELECT COUNT(*) FROM participants WHERE id = $1 AND competition_id = $2`, participantID, competitionID).Scan(&exists); err != nil {
		return err
	}
	if exists == 0 {
		return domain.ErrNotFound
	}
	var refs int
	if err := tx.QueryRow(`
		SELECT (SELECT COUNT(*) FROM bracket_slots WHERE participant_id = $1 OR winner_id = $1)
		     + (SELECT COUNT(*) FROM matches WHERE athlete_red_id = $1 OR athlete_blue_id = $1 OR winner_id = $1)`, participantID).Scan(&refs); err != nil {
		return err
	}
	if refs > 0 {
		return fmt.Errorf("%w: peserta sudah masuk bagan atau partai; reset bagan kategorinya terlebih dahulu sebelum membatalkan atau menghapus peserta", domain.ErrConflict)
	}
	if _, err := tx.Exec(`DELETE FROM participants WHERE id = $1 AND competition_id = $2`, participantID, competitionID); err != nil {
		return err
	}
	return tx.Commit()
}
