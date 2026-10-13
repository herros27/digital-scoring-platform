package store

import (
	"database/sql"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgconn"

	"github.com/herros27/digital-scoring/internal/domain"
)

// Kode SQLSTATE PostgreSQL yang dipetakan ke error domain.
const (
	pgUniqueViolation = "23505"
	pgInvalidText     = "22P02" // mis. UUID dengan format salah
)

// mapLookupErr menerjemahkan error pencarian satu baris ke error domain supaya handler memberi
// 404/400 yang benar dan pesan SQL mentah tidak sampai ke klien.
func mapLookupErr(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, sql.ErrNoRows) {
		return domain.ErrNotFound
	}
	var pg *pgconn.PgError
	if errors.As(err, &pg) && pg.Code == pgInvalidText {
		return domain.ErrInvalidInput
	}
	return err
}

// mapParticipantInsertErr mengubah pelanggaran unique (competition_id, athlete_id) menjadi ErrConflict.
func mapParticipantInsertErr(err error) error {
	if err == nil {
		return nil
	}
	var pg *pgconn.PgError
	if errors.As(err, &pg) && pg.Code == pgUniqueViolation {
		return fmt.Errorf("%w: atlet sudah terdaftar pada kejuaraan ini", domain.ErrConflict)
	}
	return mapLookupErr(err)
}
