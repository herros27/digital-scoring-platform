package auth

import (
	"crypto/rand"
	"database/sql"
	"errors"
	"fmt"
	"math/big"
	"strconv"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
)

const bcryptCost = 12

// NormalizeRoleAndSeat mengekstrak role standar ("judge") dan nomor kursi jika diberikan "judge_1" atau role & seat terpisah
func NormalizeRoleAndSeat(roleStr string, seat int) (string, int) {
	if roleStr == "juri" {
		roleStr = "judge"
	}
	if strings.HasPrefix(roleStr, "judge_") {
		numStr := strings.TrimPrefix(roleStr, "judge_")
		if n, err := strconv.Atoi(numStr); err == nil && seat == 0 {
			return "judge", n
		}
		return "judge", seat
	}
	return roleStr, seat
}

// GenerateRandomPIN menghasilkan PIN acak 6 digit secara cryptographically secure
func GenerateRandomPIN() (string, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(1000000))
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%06d", n.Int64()), nil
}

// HashPIN hash PIN 6 digit sebelum disimpan ke DB
func HashPIN(pin string) (string, error) {
	b, err := bcrypt.GenerateFromPassword([]byte(pin), bcryptCost)
	return string(b), err
}

// VerifyPIN cek PIN plain vs hash dari DB
func VerifyPIN(plain, hash string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(plain)) == nil
}

// PinStore akses tabel judge_pins
type PinStore struct {
	DB *sql.DB
}

type PinRecord struct {
	ID            string
	CompetitionID string
	ArenaID       string
	Role          string
	SeatNumber    int
	RawPIN        string
	PinHash       string
	ExpiresAt     time.Time
}

// GetByCompAndArena ambil semua PIN aktif untuk competition + arena tertentu
func (s *PinStore) GetByCompAndArena(competitionID, arenaID string) ([]PinRecord, error) {
	queryWithSeat := `
        SELECT id, competition_id, arena_id, role, COALESCE(seat_number, 0), COALESCE(raw_pin, ''), pin_hash, expires_at
        FROM judge_pins
        WHERE competition_id = $1 AND arena_id = $2 AND expires_at > NOW()
        ORDER BY role, seat_number ASC`
	rows, err := s.DB.Query(queryWithSeat, competitionID, arenaID)
	if err != nil {
		queryFallback := `
            SELECT id, competition_id, arena_id, role, COALESCE(seat_number, 0), pin_hash, expires_at
            FROM judge_pins
            WHERE competition_id = $1 AND arena_id = $2 AND expires_at > NOW()
            ORDER BY role, seat_number ASC`
		rows, err = s.DB.Query(queryFallback, competitionID, arenaID)
		if err != nil {
			return nil, err
		}
		defer rows.Close()

		var list []PinRecord
		for rows.Next() {
			var r PinRecord
			if err := rows.Scan(&r.ID, &r.CompetitionID, &r.ArenaID, &r.Role, &r.SeatNumber, &r.PinHash, &r.ExpiresAt); err != nil {
				return nil, err
			}
			r.Role, r.SeatNumber = NormalizeRoleAndSeat(r.Role, r.SeatNumber)
			list = append(list, r)
		}
		return list, rows.Err()
	}
	defer rows.Close()

	var list []PinRecord
	for rows.Next() {
		var r PinRecord
		if err := rows.Scan(&r.ID, &r.CompetitionID, &r.ArenaID, &r.Role, &r.SeatNumber, &r.RawPIN, &r.PinHash, &r.ExpiresAt); err != nil {
			return nil, err
		}
		r.Role, r.SeatNumber = NormalizeRoleAndSeat(r.Role, r.SeatNumber)
		list = append(list, r)
	}
	return list, rows.Err()
}

var ErrArenaNotFound = errors.New("gelanggang tidak ditemukan atau tidak terdaftar pada kompetisi ini")

// CreatePIN simpan PIN baru ke DB (backward-compatible)
func (s *PinStore) CreatePIN(competitionID, arenaID, role, pin string, ttl time.Duration) (*PinRecord, error) {
	return s.CreatePINWithSeat(competitionID, arenaID, role, 0, pin, ttl)
}

// CreatePINWithSeat simpan PIN baru ke DB dengan nomor kursi dan raw_pin
func (s *PinStore) CreatePINWithSeat(competitionID, arenaID, role string, seatNumber int, pin string, ttl time.Duration) (*PinRecord, error) {
	if s.DB != nil {
		var exists bool
		err := s.DB.QueryRow(`SELECT EXISTS(SELECT 1 FROM arenas WHERE id = $1 AND competition_id = $2)`, arenaID, competitionID).Scan(&exists)
		if err != nil || !exists {
			return nil, ErrArenaNotFound
		}
	}

	role, seatNumber = NormalizeRoleAndSeat(role, seatNumber)
	hash, err := HashPIN(pin)
	if err != nil {
		return nil, err
	}
	expiresAt := time.Now().Add(ttl)

	// Hapus PIN lama pada kursi/role ini di arena yang sama agar tidak menumpuk duplikat
	_, _ = s.DB.Exec(`
        DELETE FROM judge_pins
        WHERE competition_id = $1 AND arena_id = $2 AND role = $3 AND COALESCE(seat_number, 0) = $4`,
		competitionID, arenaID, role, seatNumber)

	r := &PinRecord{
		CompetitionID: competitionID,
		ArenaID:       arenaID,
		Role:          role,
		SeatNumber:    seatNumber,
		RawPIN:        pin,
		PinHash:       hash,
		ExpiresAt:     expiresAt,
	}

	queryWithRaw := `
        INSERT INTO judge_pins (competition_id, arena_id, role, seat_number, raw_pin, pin_hash, expires_at)
        VALUES ($1, $2, $3, $4, $5, $6, $7)
        RETURNING id, competition_id, arena_id, role, seat_number, COALESCE(raw_pin, ''), pin_hash, expires_at`
	err = s.DB.QueryRow(queryWithRaw, competitionID, arenaID, role, seatNumber, pin, hash, expiresAt).
		Scan(&r.ID, &r.CompetitionID, &r.ArenaID, &r.Role, &r.SeatNumber, &r.RawPIN, &r.PinHash, &r.ExpiresAt)
	if err != nil {
		// Fallback query jika kolom raw_pin belum ada
		queryFallback := `
            INSERT INTO judge_pins (competition_id, arena_id, role, seat_number, pin_hash, expires_at)
            VALUES ($1, $2, $3, $4, $5, $6)
            RETURNING id, competition_id, arena_id, role, seat_number, pin_hash, expires_at`
		err = s.DB.QueryRow(queryFallback, competitionID, arenaID, role, seatNumber, hash, expiresAt).
			Scan(&r.ID, &r.CompetitionID, &r.ArenaID, &r.Role, &r.SeatNumber, &r.PinHash, &r.ExpiresAt)
		if err != nil {
			return nil, err
		}
		r.RawPIN = pin
	}
	return r, nil
}

// LoginWithPIN cari PIN yang match, return token JWT kalau berhasil
func (s *PinStore) LoginWithPIN(competitionID, arenaID, pin string) (string, *PinRecord, error) {
	records, err := s.GetByCompAndArena(competitionID, arenaID)
	if err != nil {
		return "", nil, err
	}

	for _, r := range records {
		if VerifyPIN(pin, r.PinHash) {
			token, err := GenerateTokenWithSeat(r.ID, r.CompetitionID, r.ArenaID, r.Role, r.SeatNumber, 12*time.Hour)
			if err != nil {
				return "", nil, err
			}
			return token, &r, nil
		}
	}
	return "", nil, errors.New("invalid pin")
}
