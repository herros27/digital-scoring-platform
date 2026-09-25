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
			participant_id, is_bye, winner_id, next_slot_id
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
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
			slot.WinnerID, slot.NextSlotID,
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
		SELECT id, competition_id, weight_class, gender, round, slot_index, participant_id, is_bye, winner_id, next_slot_id
		FROM bracket_slots
		WHERE competition_id = $1 AND weight_class = $2 AND gender = $3
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
			&b.WinnerID, &b.NextSlotID,
		)
		if err != nil {
			return nil, err
		}
		list = append(list, b)
	}
	return list, rows.Err()
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
		SELECT id, competition_id, weight_class, gender, round, slot_index, participant_id, is_bye, winner_id, next_slot_id
		FROM bracket_slots
		WHERE id = $1
	`
	err := s.DB.QueryRow(query, id).Scan(
		&b.ID, &b.CompetitionID, &b.WeightClass, &b.Gender,
		&b.Round, &b.SlotIndex, &b.ParticipantID, &b.IsBye,
		&b.WinnerID, &b.NextSlotID,
	)
	return b, err
}