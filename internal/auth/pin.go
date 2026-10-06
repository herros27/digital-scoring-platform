package auth

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"golang.org/x/crypto/bcrypt"
)

var bcryptCost = 12

// SetBcryptCost mengubah cost bcrypt. Hanya untuk tes (cost 12 sangat lambat di bawah -race).
func SetBcryptCost(cost int) { bcryptCost = cost }

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

const (
	// JudgePINLength: PIN juri (dan role non-wasit lain) 6 digit.
	JudgePINLength = 6
	// RefereePINLength: PIN wasit lebih panjang karena wasit dapat mengendalikan pertandingan.
	RefereePINLength = 8
)

var (
	// ErrInvalidPIN: PIN tidak ditemukan atau sudah kedaluwarsa.
	ErrInvalidPIN = errors.New("invalid pin")
	// ErrPinTaken: PIN sudah dipakai kursi lain yang masih aktif. Login memakai PIN saja,
	// sehingga setiap PIN harus unik di seluruh sistem.
	ErrPinTaken = errors.New("pin sudah dipakai, gunakan PIN lain")
)

// PINLength mengembalikan panjang PIN yang disyaratkan untuk sebuah role.
func PINLength(role string) int {
	if role == "wasit" || role == "referee" {
		return RefereePINLength
	}
	return JudgePINLength
}

// ValidatePIN memastikan PIN hanya berisi digit dan panjangnya sesuai role.
func ValidatePIN(role, pin string) error {
	want := PINLength(role)
	if len(pin) != want {
		return fmt.Errorf("PIN untuk role %s harus %d digit", role, want)
	}
	for _, c := range pin {
		if c < '0' || c > '9' {
			return errors.New("PIN hanya boleh berisi angka")
		}
	}
	return nil
}

// GeneratePIN menghasilkan PIN acak dengan jumlah digit tertentu (cryptographically secure).
func GeneratePIN(digits int) (string, error) {
	max := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(digits)), nil)
	n, err := rand.Int(rand.Reader, max)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%0*d", digits, n), nil
}

// GenerateRandomPIN menghasilkan PIN acak 6 digit (PIN juri).
func GenerateRandomPIN() (string, error) {
	return GeneratePIN(JudgePINLength)
}

// HashPIN hash PIN sebelum disimpan ke DB
func HashPIN(pin string) (string, error) {
	b, err := bcrypt.GenerateFromPassword([]byte(pin), bcryptCost)
	return string(b), err
}

// VerifyPIN cek PIN plain vs hash dari DB
func VerifyPIN(plain, hash string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(plain)) == nil
}

// PinLookup menghitung kunci pencarian PIN: HMAC-SHA256 (hex) dengan kunci turunan JWT secret.
// Deterministik sehingga bisa diberi indeks unik; PIN tidak bisa ditebak dari nilainya tanpa
// secret server. Mengganti JWT secret membuat semua PIN lama tidak berlaku.
func PinLookup(pin string) string {
	keyMAC := hmac.New(sha256.New, jwtSecret)
	keyMAC.Write([]byte("digital-scoring/pin-lookup/v1"))
	mac := hmac.New(sha256.New, keyMAC.Sum(nil))
	mac.Write([]byte(pin))
	return hex.EncodeToString(mac.Sum(nil))
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

const pinColumns = `id, competition_id, arena_id, role, COALESCE(seat_number, 0), COALESCE(raw_pin, ''), pin_hash, expires_at`

func scanPin(sc interface{ Scan(...any) error }) (PinRecord, error) {
	var r PinRecord
	err := sc.Scan(&r.ID, &r.CompetitionID, &r.ArenaID, &r.Role, &r.SeatNumber, &r.RawPIN, &r.PinHash, &r.ExpiresAt)
	r.Role, r.SeatNumber = NormalizeRoleAndSeat(r.Role, r.SeatNumber)
	return r, err
}

// GetByCompAndArena ambil semua PIN aktif untuk competition + arena tertentu
func (s *PinStore) GetByCompAndArena(competitionID, arenaID string) ([]PinRecord, error) {
	rows, err := s.DB.Query(`
        SELECT `+pinColumns+`
        FROM judge_pins
        WHERE competition_id = $1 AND arena_id = $2 AND expires_at > NOW()
        ORDER BY role, seat_number ASC`, competitionID, arenaID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []PinRecord
	for rows.Next() {
		r, err := scanPin(rows)
		if err != nil {
			return nil, err
		}
		list = append(list, r)
	}
	return list, rows.Err()
}

var ErrArenaNotFound = errors.New("gelanggang tidak ditemukan atau tidak terdaftar pada kompetisi ini")

// CreatePIN simpan PIN baru ke DB (backward-compatible)
func (s *PinStore) CreatePIN(competitionID, arenaID, role, pin string, ttl time.Duration) (*PinRecord, error) {
	return s.CreatePINWithSeat(competitionID, arenaID, role, 0, pin, ttl)
}

// CreatePINWithSeat simpan PIN baru ke DB dengan nomor kursi dan raw_pin.
// PIN harus valid untuk role-nya dan unik di antara PIN aktif (ErrPinTaken bila bentrok).
func (s *PinStore) CreatePINWithSeat(competitionID, arenaID, role string, seatNumber int, pin string, ttl time.Duration) (*PinRecord, error) {
	role, seatNumber = NormalizeRoleAndSeat(role, seatNumber)
	if err := ValidatePIN(role, pin); err != nil {
		return nil, err
	}

	var exists bool
	err := s.DB.QueryRow(`SELECT EXISTS(SELECT 1 FROM arenas WHERE id = $1 AND competition_id = $2)`, arenaID, competitionID).Scan(&exists)
	if err != nil || !exists {
		return nil, ErrArenaNotFound
	}

	hash, err := HashPIN(pin)
	if err != nil {
		return nil, err
	}
	lookup := PinLookup(pin)
	expiresAt := time.Now().Add(ttl)

	// Hapus PIN lama pada kursi/role ini di arena yang sama agar tidak menumpuk duplikat,
	// dan PIN kedaluwarsa dengan nilai yang sama agar tidak menghalangi indeks unik.
	_, _ = s.DB.Exec(`
        DELETE FROM judge_pins
        WHERE (competition_id = $1 AND arena_id = $2 AND role = $3 AND COALESCE(seat_number, 0) = $4)
           OR (pin_lookup = $5 AND expires_at <= NOW())`,
		competitionID, arenaID, role, seatNumber, lookup)

	r, err := scanPin(s.DB.QueryRow(`
        INSERT INTO judge_pins (competition_id, arena_id, role, seat_number, raw_pin, pin_hash, pin_lookup, expires_at)
        VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
        RETURNING `+pinColumns,
		competitionID, arenaID, role, seatNumber, pin, hash, lookup, expiresAt))
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return nil, ErrPinTaken
		}
		return nil, err
	}
	return &r, nil
}

// CreateWithRandomPIN membuat PIN acak yang unik (panjang sesuai role); mengulang bila bentrok.
func (s *PinStore) CreateWithRandomPIN(competitionID, arenaID, role string, seatNumber int, ttl time.Duration) (*PinRecord, error) {
	normRole, _ := NormalizeRoleAndSeat(role, seatNumber)
	for attempt := 0; attempt < 20; attempt++ {
		pin, err := GeneratePIN(PINLength(normRole))
		if err != nil {
			return nil, err
		}
		rec, err := s.CreatePINWithSeat(competitionID, arenaID, role, seatNumber, pin, ttl)
		if errors.Is(err, ErrPinTaken) {
			continue
		}
		return rec, err
	}
	return nil, ErrPinTaken
}

// LoginWithPIN mencari PIN aktif berdasarkan nilainya saja (turnamen, arena, role, dan kursi
// ditentukan server dari data PIN) dan mengembalikan token JWT bila berhasil.
func (s *PinStore) LoginWithPIN(pin string) (string, *PinRecord, error) {
	r, err := scanPin(s.DB.QueryRow(`
        SELECT `+pinColumns+`
        FROM judge_pins
        WHERE pin_lookup = $1 AND expires_at > NOW()`, PinLookup(pin)))
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil, ErrInvalidPIN
	}
	if err != nil {
		return "", nil, err
	}
	// Pertahanan berlapis: pastikan hash bcrypt juga cocok
	if !VerifyPIN(pin, r.PinHash) {
		return "", nil, ErrInvalidPIN
	}

	token, err := GenerateTokenWithSeat(r.ID, r.CompetitionID, r.ArenaID, r.Role, r.SeatNumber, 12*time.Hour)
	if err != nil {
		return "", nil, err
	}
	return token, &r, nil
}
