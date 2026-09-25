package auth

import (
    "database/sql"
    "errors"
    "time"

    "golang.org/x/crypto/bcrypt"
)

const bcryptCost = 12

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
    PinHash       string
    ExpiresAt     time.Time
}

// GetByCompAndArena ambil semua PIN aktif untuk competition + arena tertentu
// (dipakai saat judge mau login, untuk cari record yang cocok)
func (s *PinStore) GetByCompAndArena(competitionID, arenaID string) ([]PinRecord, error) {
    rows, err := s.DB.Query(`
        SELECT id, competition_id, arena_id, role, pin_hash, expires_at
        FROM judge_pins
        WHERE competition_id = $1 AND arena_id = $2 AND expires_at > NOW()`,
        competitionID, arenaID,
    )
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    var list []PinRecord
    for rows.Next() {
        var r PinRecord
        rows.Scan(&r.ID, &r.CompetitionID, &r.ArenaID, &r.Role, &r.PinHash, &r.ExpiresAt)
        list = append(list, r)
    }
    return list, nil
}

// CreatePIN simpan PIN baru ke DB
func (s *PinStore) CreatePIN(competitionID, arenaID, role, pin string, ttl time.Duration) (*PinRecord, error) {
    hash, err := HashPIN(pin)
    if err != nil {
        return nil, err
    }
    expiresAt := time.Now().Add(ttl)
    r := &PinRecord{}
    err = s.DB.QueryRow(`
        INSERT INTO judge_pins (competition_id, arena_id, role, pin_hash, expires_at)
        VALUES ($1, $2, $3, $4, $5)
        RETURNING id, competition_id, arena_id, role, pin_hash, expires_at`,
        competitionID, arenaID, role, hash, expiresAt,
    ).Scan(&r.ID, &r.CompetitionID, &r.ArenaID, &r.Role, &r.PinHash, &r.ExpiresAt)
    return r, err
}

// LoginWithPIN cari PIN yang match, return token JWT kalau berhasil
// TTL token = 12 jam (selama match hari itu)
func (s *PinStore) LoginWithPIN(competitionID, arenaID, pin string) (string, *PinRecord, error) {
    records, err := s.GetByCompAndArena(competitionID, arenaID)
    if err != nil {
        return "", nil, err
    }

    for _, r := range records {
        if VerifyPIN(pin, r.PinHash) {
            token, err := GenerateToken(r.ID, r.CompetitionID, r.ArenaID, r.Role, 12*time.Hour)
            if err != nil {
                return "", nil, err
            }
            return token, &r, nil
        }
    }
    return "", nil, errors.New("invalid pin")
}