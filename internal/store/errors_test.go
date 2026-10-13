package store

import (
	"database/sql"
	"errors"
	"fmt"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"

	"github.com/herros27/digital-scoring/internal/domain"
)

func TestMapLookupErr(t *testing.T) {
	if mapLookupErr(nil) != nil {
		t.Fatal("nil harus tetap nil")
	}
	if err := mapLookupErr(sql.ErrNoRows); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("sql.ErrNoRows harus menjadi ErrNotFound, got %v", err)
	}
	// terbungkus pun tetap dikenali
	if err := mapLookupErr(fmt.Errorf("scan: %w", sql.ErrNoRows)); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("sql.ErrNoRows terbungkus harus menjadi ErrNotFound, got %v", err)
	}
	// UUID berformat salah: jangan bocorkan pesan SQL
	err := mapLookupErr(&pgconn.PgError{Code: "22P02", Message: `invalid input syntax for type uuid: "xyz"`})
	if !errors.Is(err, domain.ErrInvalidInput) {
		t.Fatalf("22P02 harus menjadi ErrInvalidInput, got %v", err)
	}
	if got := err.Error(); got != domain.ErrInvalidInput.Error() {
		t.Fatalf("pesan tidak boleh memuat detail SQL, got %q", got)
	}
	other := errors.New("boom")
	if err := mapLookupErr(other); err != other {
		t.Fatalf("error lain harus diteruskan apa adanya, got %v", err)
	}
}

func TestMapParticipantInsertErr(t *testing.T) {
	if mapParticipantInsertErr(nil) != nil {
		t.Fatal("nil harus tetap nil")
	}
	err := mapParticipantInsertErr(&pgconn.PgError{Code: "23505", ConstraintName: "ux_participants_competition_athlete"})
	if !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("23505 harus menjadi ErrConflict, got %v", err)
	}
	other := errors.New("boom")
	if err := mapParticipantInsertErr(other); err != other {
		t.Fatalf("error lain harus diteruskan apa adanya, got %v", err)
	}
}
