package store

import (
	"database/sql"
)

type BracketStore struct {
	DB *sql.DB
}

// CreateBatch menyimpan banyak slot sekaligus dalam 1 transaksi
func (s *BracketStore) CreateBatch(slots []BracketSlot) error {
	tx, err := s.DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	query := `
		INSERT INTO bracket_slots (
			id, competition_id, weight_class, gender, round, slot_index, 
			participant_id, is_bye, winner_id, next_slot_id, category_id
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`
	stmt, err := tx.Prepare(query)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, slot := range slots {
		_, err := stmt.Exec(
			slot.ID, slot.CompetitionID, slot.WeightClass, slot.Gender,
			slot.Round, slot.SlotIndex, slot.ParticipantID, slot.IsBye,
			slot.WinnerID, slot.NextSlotID, slot.CategoryID,
		)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

// GetByCompetitionCategory mengambil semua slot bagan untuk kategori tertentu
func (s *BracketStore) GetByCompetitionCategory(competitionID, weightClass, gender string) ([]BracketSlot, error) {
	query := `
		SELECT id, competition_id, weight_class, gender, round, slot_index, participant_id, is_bye, winner_id, next_slot_id, category_id
		FROM bracket_slots
		WHERE competition_id = $1 AND weight_class = $2 AND gender = $3 AND category_id IS NULL
		ORDER BY round ASC, slot_index ASC
	`
	rows, err := s.DB.Query(query, competitionID, weightClass, gender)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []BracketSlot
	for rows.Next() {
		var b BracketSlot
		err := rows.Scan(
			&b.ID, &b.CompetitionID, &b.WeightClass, &b.Gender,
			&b.Round, &b.SlotIndex, &b.ParticipantID, &b.IsBye,
			&b.WinnerID, &b.NextSlotID, &b.CategoryID,
		)
		if err != nil {
			return nil, err
		}
		list = append(list, b)
	}
	return list, rows.Err()
}

// DeleteByCompetitionCategory menghapus seluruh slot bagan untuk kategori tertentu saat acak ulang bagan
func (s *BracketStore) DeleteByCompetitionCategory(competitionID, weightClass, gender string) error {
	// 1. Hapus partai idle lama pada kategori ini agar jadwal pertandingan bersih dan selaras dengan bagan baru
	deleteIdleQuery := `
		DELETE FROM matches 
		WHERE competition_id = $1 
		  AND status = 'idle' 
		  AND (
		    bracket_slot_id IN (
		      SELECT id FROM bracket_slots WHERE competition_id = $1 AND weight_class = $2 AND gender = $3 AND category_id IS NULL
		    )
		    OR athlete_red_id IN (
		      SELECT id FROM participants WHERE competition_id = $1 AND weight_class = $2 AND gender = $3 AND category_id IS NULL
		    )
		  )
	`
	if _, err := s.DB.Exec(deleteIdleQuery, competitionID, weightClass, gender); err != nil {
		return err
	}

	// 2. Unlink bracket_slot_id pada matches yang tersisa (misal match running/finished) agar tidak melanggar foreign key constraint
	unlinkQuery := `
		UPDATE matches SET bracket_slot_id = NULL 
		WHERE competition_id = $1 AND bracket_slot_id IN (
			SELECT id FROM bracket_slots WHERE competition_id = $1 AND weight_class = $2 AND gender = $3 AND category_id IS NULL
		)
	`
	if _, err := s.DB.Exec(unlinkQuery, competitionID, weightClass, gender); err != nil {
		return err
	}

	// 3. Hapus seluruh slot pada kategori tersebut
	query := `DELETE FROM bracket_slots WHERE competition_id = $1 AND weight_class = $2 AND gender = $3 AND category_id IS NULL`
	_, err := s.DB.Exec(query, competitionID, weightClass, gender)
	return err
}

// UpdateWinner memperbarui winner_id pada slot tertentu
func (s *BracketStore) UpdateWinner(slotID, winnerID string) error {
	query := `UPDATE bracket_slots SET winner_id = $1 WHERE id = $2`
	_, err := s.DB.Exec(query, winnerID, slotID)
	return err
}

// SetParticipant mengisi peserta di slot tertentu (misal saat menang dan naik ronde)
func (s *BracketStore) SetParticipant(slotID, participantID string) error {
	query := `UPDATE bracket_slots SET participant_id = $1 WHERE id = $2`
	_, err := s.DB.Exec(query, participantID, slotID)
	return err
}

// GetByID mengambil 1 slot berdasarkan ID
func (s *BracketStore) GetByID(id string) (*BracketSlot, error) {
	b := &BracketSlot{}
	query := `
		SELECT id, competition_id, weight_class, gender, round, slot_index, participant_id, is_bye, winner_id, next_slot_id, category_id
		FROM bracket_slots
		WHERE id = $1
	`
	err := s.DB.QueryRow(query, id).Scan(
		&b.ID, &b.CompetitionID, &b.WeightClass, &b.Gender,
		&b.Round, &b.SlotIndex, &b.ParticipantID, &b.IsBye,
		&b.WinnerID, &b.NextSlotID, &b.CategoryID,
	)
	return b, err
}

// AdvanceWinner memajukan pemenang pertandingan ke slot ronde berikutnya
func (s *BracketStore) AdvanceWinner(currentSlotID, winnerID string) error {
	currentSlot, err := s.GetByID(currentSlotID)
	if err != nil {
		return err
	}
	if currentSlot == nil {
		return sql.ErrNoRows
	}

	if err := s.UpdateWinner(currentSlotID, winnerID); err != nil {
		return err
	}

	if currentSlot.NextSlotID != nil && *currentSlot.NextSlotID != "" {
		if err := s.SetParticipant(*currentSlot.NextSlotID, winnerID); err != nil {
			return err
		}

		// Otomatis buat match untuk babak selanjutnya jika kedua sudut (merah & biru) sudah terisi pemenang
		parentSlot, err := s.GetByID(*currentSlot.NextSlotID)
		if err == nil && parentSlot != nil {
			siblingIndex := parentSlot.SlotIndex ^ 1
			var siblingSlot BracketSlot
			var errSibling error
			if parentSlot.CategoryID != nil && *parentSlot.CategoryID != "" {
				// Bagan terisolasi per kategori: kelas bobot yang sama di golongan umur berbeda tidak boleh bertabrakan.
				errSibling = s.DB.QueryRow(`
					SELECT id, competition_id, weight_class, gender, round, slot_index, participant_id, is_bye, winner_id, next_slot_id, category_id
					FROM bracket_slots
					WHERE competition_id = $1 AND category_id = $2 AND round = $3 AND slot_index = $4
					LIMIT 1`, parentSlot.CompetitionID, *parentSlot.CategoryID, parentSlot.Round, siblingIndex).Scan(
					&siblingSlot.ID, &siblingSlot.CompetitionID, &siblingSlot.WeightClass, &siblingSlot.Gender,
					&siblingSlot.Round, &siblingSlot.SlotIndex, &siblingSlot.ParticipantID, &siblingSlot.IsBye,
					&siblingSlot.WinnerID, &siblingSlot.NextSlotID, &siblingSlot.CategoryID,
				)
			} else {
				errSibling = s.DB.QueryRow(`
					SELECT id, competition_id, weight_class, gender, round, slot_index, participant_id, is_bye, winner_id, next_slot_id, category_id
					FROM bracket_slots
					WHERE competition_id = $1 AND weight_class = $2 AND gender = $3 AND category_id IS NULL AND round = $4 AND slot_index = $5
					LIMIT 1`, parentSlot.CompetitionID, parentSlot.WeightClass, parentSlot.Gender, parentSlot.Round, siblingIndex).Scan(
					&siblingSlot.ID, &siblingSlot.CompetitionID, &siblingSlot.WeightClass, &siblingSlot.Gender,
					&siblingSlot.Round, &siblingSlot.SlotIndex, &siblingSlot.ParticipantID, &siblingSlot.IsBye,
					&siblingSlot.WinnerID, &siblingSlot.NextSlotID, &siblingSlot.CategoryID,
				)
			}

			if errSibling == nil && siblingSlot.ParticipantID != nil && *siblingSlot.ParticipantID != "" {
				var redSlot, blueSlot *BracketSlot
				if parentSlot.SlotIndex%2 == 0 {
					redSlot = parentSlot
					blueSlot = &siblingSlot
				} else {
					redSlot = &siblingSlot
					blueSlot = parentSlot
				}

				var existingMatchID string
				checkMatchQuery := `SELECT id FROM matches WHERE competition_id = $1 AND bracket_slot_id = $2 LIMIT 1`
				errCheck := s.DB.QueryRow(checkMatchQuery, parentSlot.CompetitionID, redSlot.ID).Scan(&existingMatchID)
				if errCheck == sql.ErrNoRows {
					var arenaID sql.NullString
					_ = s.DB.QueryRow(`SELECT id FROM arenas WHERE competition_id = $1 ORDER BY name ASC LIMIT 1`, parentSlot.CompetitionID).Scan(&arenaID)

					insertMatchQuery := `
						INSERT INTO matches (competition_id, arena_id, bracket_slot_id, athlete_red_id, athlete_blue_id, current_round, status)
						VALUES ($1, $2, $3, $4, $5, $6, 'idle')
					`
					var aID any = nil
					if arenaID.Valid {
						aID = arenaID.String
					}
					_, _ = s.DB.Exec(insertMatchQuery, parentSlot.CompetitionID, aID, redSlot.ID, *redSlot.ParticipantID, *blueSlot.ParticipantID, redSlot.Round)
				}
			}
		}
	}

	return nil
}

// GetByCategoryID mengambil seluruh slot bagan milik satu kategori.
func (s *BracketStore) GetByCategoryID(competitionID, categoryID string) ([]BracketSlot, error) {
	rows, err := s.DB.Query(`
		SELECT id, competition_id, weight_class, gender, round, slot_index, participant_id, is_bye, winner_id, next_slot_id, category_id
		FROM bracket_slots
		WHERE competition_id = $1 AND category_id = $2
		ORDER BY round ASC, slot_index ASC`, competitionID, categoryID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []BracketSlot
	for rows.Next() {
		var b BracketSlot
		if err := rows.Scan(&b.ID, &b.CompetitionID, &b.WeightClass, &b.Gender,
			&b.Round, &b.SlotIndex, &b.ParticipantID, &b.IsBye,
			&b.WinnerID, &b.NextSlotID, &b.CategoryID); err != nil {
			return nil, err
		}
		list = append(list, b)
	}
	return list, rows.Err()
}

// DeleteByCategoryID menghapus bagan satu kategori (acak ulang) beserta partai idle-nya.
func (s *BracketStore) DeleteByCategoryID(competitionID, categoryID string) error {
	if _, err := s.DB.Exec(`
		DELETE FROM matches
		WHERE competition_id = $1 AND status = 'idle' AND (
			bracket_slot_id IN (SELECT id FROM bracket_slots WHERE competition_id = $1 AND category_id = $2)
			OR athlete_red_id IN (SELECT id FROM participants WHERE competition_id = $1 AND category_id = $2)
		)`, competitionID, categoryID); err != nil {
		return err
	}
	if _, err := s.DB.Exec(`
		UPDATE matches SET bracket_slot_id = NULL
		WHERE competition_id = $1 AND bracket_slot_id IN (
			SELECT id FROM bracket_slots WHERE competition_id = $1 AND category_id = $2
		)`, competitionID, categoryID); err != nil {
		return err
	}
	_, err := s.DB.Exec(`DELETE FROM bracket_slots WHERE competition_id = $1 AND category_id = $2`, competitionID, categoryID)
	return err
}

// CategoryBracketSummary ringkasan satu bagan kategori untuk daftar bagan publik.
type CategoryBracketSummary struct {
	CategoryID   string
	Participants int  // peserta nyata di ronde 1 (tanpa bye)
	Rounds       int  // jumlah ronde
	Completed    bool // babak final sudah punya pemenang
}

// ListCategorySummaries meringkas semua bagan berbasis kategori pada satu kejuaraan dalam satu query.
// Slot final adalah slot tanpa next_slot_id.
func (s *BracketStore) ListCategorySummaries(competitionID string) ([]CategoryBracketSummary, error) {
	rows, err := s.DB.Query(`
		SELECT category_id::text,
		       COUNT(*) FILTER (WHERE round = 1 AND participant_id IS NOT NULL),
		       MAX(round),
		       COALESCE(BOOL_OR(winner_id IS NOT NULL) FILTER (WHERE next_slot_id IS NULL), false)
		FROM bracket_slots
		WHERE competition_id = $1 AND category_id IS NOT NULL
		GROUP BY category_id`, competitionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	list := []CategoryBracketSummary{}
	for rows.Next() {
		var c CategoryBracketSummary
		if err := rows.Scan(&c.CategoryID, &c.Participants, &c.Rounds, &c.Completed); err != nil {
			return nil, err
		}
		list = append(list, c)
	}
	return list, rows.Err()
}

// HasStartedMatches true bila ada partai pada bagan kategori ini yang tidak lagi idle (berjalan, dijeda, selesai,
// atau dibatalkan). Mengacak ulang bagan seperti itu akan memutus partai dari slotnya dan merusak hasil.
func (s *BracketStore) HasStartedMatches(competitionID, categoryID string) (bool, error) {
	var n int
	err := s.DB.QueryRow(`
		SELECT COUNT(*) FROM matches m
		JOIN bracket_slots b ON b.id = m.bracket_slot_id
		WHERE m.competition_id = $1 AND b.category_id = $2 AND m.status <> 'idle'`,
		competitionID, categoryID).Scan(&n)
	return n > 0, err
}
