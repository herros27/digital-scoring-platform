package tests

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"golang.org/x/crypto/bcrypt"

	"github.com/herros27/digital-scoring/internal/api/handler"
	"github.com/herros27/digital-scoring/internal/auth"
	"github.com/herros27/digital-scoring/internal/store"
)

// pinTestEnv menyiapkan 1 kompetisi dengan 2 arena di database nyata. Dilewati bila
// database tidak tersedia (set TEST_DB_URL untuk memakai database lain).
func pinTestEnv(t *testing.T) (pins *auth.PinStore, comp, arenaA, arenaB string) {
	t.Helper()
	auth.SetBcryptCost(bcrypt.MinCost) // cost produksi (12) terlalu lambat untuk tes
	dsn := os.Getenv("TEST_DB_URL")
	if dsn == "" {
		dsn = "postgres://scoring_user@localhost:5432/digital_scoring?sslmode=disable"
	}
	db, err := sql.Open("pgx", dsn)
	if err != nil || db.Ping() != nil {
		t.Skip("database tidak tersedia; tes PIN dilewati")
	}
	t.Cleanup(func() { db.Close() })
	if err := store.RunMigrations(db); err != nil {
		t.Fatalf("migrasi: %v", err)
	}

	name := fmt.Sprintf("PIN test %d", time.Now().UnixNano())
	if err := db.QueryRow(`INSERT INTO competitions (name, location, start_date, end_date, status)
		VALUES ($1, 'x', '2026-01-01', '2026-01-02', 'draft') RETURNING id`, name).Scan(&comp); err != nil {
		t.Fatalf("insert competition: %v", err)
	}
	t.Cleanup(func() { _, _ = db.Exec(`DELETE FROM competitions WHERE id = $1`, comp) }) // cascade ke arena & PIN

	for i, dst := range []*string{&arenaA, &arenaB} {
		if err := db.QueryRow(`INSERT INTO arenas (competition_id, name) VALUES ($1, $2) RETURNING id`,
			comp, fmt.Sprintf("Arena %d", i+1)).Scan(dst); err != nil {
			t.Fatalf("insert arena: %v", err)
		}
	}
	return &auth.PinStore{DB: db}, comp, arenaA, arenaB
}

// uniquePIN membuat PIN acak yang sangat kecil kemungkinan bentrok dengan data lain di DB uji.
func uniquePIN(t *testing.T, role string) string {
	t.Helper()
	pin, err := auth.GeneratePIN(auth.PINLength(role))
	if err != nil {
		t.Fatal(err)
	}
	return pin
}

func TestPinLogin_ResolvesEverythingFromPIN(t *testing.T) {
	pins, comp, arenaA, arenaB := pinTestEnv(t)

	pinA := uniquePIN(t, "judge")
	pinB := uniquePIN(t, "judge")
	if _, err := pins.CreatePINWithSeat(comp, arenaA, "juri", 2, pinA, time.Hour); err != nil {
		t.Fatal(err)
	}
	if _, err := pins.CreatePINWithSeat(comp, arenaB, "juri", 1, pinB, time.Hour); err != nil {
		t.Fatal(err)
	}

	token, rec, err := pins.LoginWithPIN(pinA)
	if err != nil {
		t.Fatalf("login: %v", err)
	}
	if rec.CompetitionID != comp || rec.ArenaID != arenaA || rec.Role != "judge" || rec.SeatNumber != 2 {
		t.Fatalf("record salah: %+v", rec)
	}
	claims, err := auth.ParseToken(token)
	if err != nil || claims.ArenaID != arenaA || claims.CompetitionID != comp || claims.SeatNumber != 2 {
		t.Fatalf("token tidak memuat konteks dari PIN: %+v err=%v", claims, err)
	}

	// PIN arena lain menghasilkan arena lain
	_, recB, err := pins.LoginWithPIN(pinB)
	if err != nil || recB.ArenaID != arenaB || recB.SeatNumber != 1 {
		t.Fatalf("login arena B: %+v err=%v", recB, err)
	}
}

func TestPinLogin_RejectsUnknownAndExpired(t *testing.T) {
	pins, comp, arenaA, _ := pinTestEnv(t)

	if _, _, err := pins.LoginWithPIN(uniquePIN(t, "judge")); !errors.Is(err, auth.ErrInvalidPIN) {
		t.Fatalf("PIN tak dikenal harus ErrInvalidPIN, got %v", err)
	}

	pin := uniquePIN(t, "judge")
	if _, err := pins.CreatePINWithSeat(comp, arenaA, "juri", 1, pin, -time.Minute); err != nil {
		t.Fatal(err)
	}
	if _, _, err := pins.LoginWithPIN(pin); !errors.Is(err, auth.ErrInvalidPIN) {
		t.Fatalf("PIN kedaluwarsa harus ditolak, got %v", err)
	}
}

func TestPin_MustBeGloballyUnique(t *testing.T) {
	pins, comp, arenaA, arenaB := pinTestEnv(t)

	pin := uniquePIN(t, "judge")
	if _, err := pins.CreatePINWithSeat(comp, arenaA, "juri", 1, pin, time.Hour); err != nil {
		t.Fatal(err)
	}
	// kursi/arena lain tidak boleh memakai PIN yang sama
	if _, err := pins.CreatePINWithSeat(comp, arenaB, "juri", 1, pin, time.Hour); !errors.Is(err, auth.ErrPinTaken) {
		t.Fatalf("PIN bentrok harus ErrPinTaken, got %v", err)
	}
	// kursi yang sama boleh menimpa PIN-nya sendiri (rotasi), dan PIN lama tidak berlaku lagi
	newPin := uniquePIN(t, "judge")
	if _, err := pins.CreatePINWithSeat(comp, arenaA, "juri", 1, newPin, time.Hour); err != nil {
		t.Fatalf("rotasi PIN: %v", err)
	}
	if _, _, err := pins.LoginWithPIN(pin); !errors.Is(err, auth.ErrInvalidPIN) {
		t.Fatalf("PIN lama harus tidak berlaku setelah rotasi, got %v", err)
	}
	if _, _, err := pins.LoginWithPIN(newPin); err != nil {
		t.Fatalf("PIN baru harus berlaku: %v", err)
	}
}

func TestPin_ExpiredPinDoesNotBlockReuse(t *testing.T) {
	pins, comp, arenaA, arenaB := pinTestEnv(t)

	pin := uniquePIN(t, "judge")
	if _, err := pins.CreatePINWithSeat(comp, arenaA, "juri", 1, pin, -time.Minute); err != nil {
		t.Fatal(err)
	}
	if _, err := pins.CreatePINWithSeat(comp, arenaB, "juri", 1, pin, time.Hour); err != nil {
		t.Fatalf("PIN kedaluwarsa tidak boleh menghalangi pemakaian ulang: %v", err)
	}
}

func TestPin_FormatPerRole(t *testing.T) {
	pins, comp, arenaA, _ := pinTestEnv(t)

	if _, err := pins.CreatePINWithSeat(comp, arenaA, "wasit", 0, "123456", time.Hour); err == nil {
		t.Error("PIN wasit 6 digit harus ditolak")
	}
	if _, err := pins.CreatePINWithSeat(comp, arenaA, "juri", 1, "12345678", time.Hour); err == nil {
		t.Error("PIN juri 8 digit harus ditolak")
	}
	if _, err := pins.CreatePINWithSeat(comp, arenaA, "juri", 1, "12ab56", time.Hour); err == nil {
		t.Error("PIN non-angka harus ditolak")
	}

	rec, err := pins.CreateWithRandomPIN(comp, arenaA, "wasit", 0, time.Hour)
	if err != nil || len(rec.RawPIN) != 8 || strings.Trim(rec.RawPIN, "0123456789") != "" {
		t.Fatalf("PIN wasit acak harus 8 digit angka: %+v err=%v", rec, err)
	}
	rec, err = pins.CreateWithRandomPIN(comp, arenaA, "juri", 3, time.Hour)
	if err != nil || len(rec.RawPIN) != 6 {
		t.Fatalf("PIN juri acak harus 6 digit: %+v err=%v", rec, err)
	}
}

func postJudgeLogin(h *handler.AuthHandler, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest("POST", "/api/v1/auth/judge-login", strings.NewReader(body))
	req.RemoteAddr = "10.0.0.7:5555"
	rec := httptest.NewRecorder()
	h.JudgeLogin(rec, req)
	return rec
}

func newLoginHandler(pins *auth.PinStore) *handler.AuthHandler {
	return &handler.AuthHandler{
		Pins: pins,
		Guard: auth.NewPinGuard(auth.PinGuardConfig{
			Window: time.Minute, DeviceMax: 3, DeviceLockout: time.Minute,
			IPMax: 100, IPLockout: time.Minute, GlobalMax: 1000, GlobalLockout: time.Minute,
		}),
	}
}

func TestJudgeLoginHandler_PinOnly(t *testing.T) {
	pins, comp, arenaA, _ := pinTestEnv(t)
	h := newLoginHandler(pins)

	pin := uniquePIN(t, "judge")
	if _, err := pins.CreatePINWithSeat(comp, arenaA, "juri", 2, pin, time.Hour); err != nil {
		t.Fatal(err)
	}

	rec := postJudgeLogin(h, `{"pin":"`+pin+`"}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body)
	}
	var res struct {
		Data struct {
			Token         string `json:"token"`
			Role          string `json:"role"`
			ArenaID       string `json:"arena_id"`
			CompetitionID string `json:"competition_id"`
			SeatNumber    int    `json:"seat_number"`
		} `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &res); err != nil {
		t.Fatal(err)
	}
	d := res.Data
	if d.Token == "" || d.Role != "judge" || d.ArenaID != arenaA || d.CompetitionID != comp || d.SeatNumber != 2 {
		t.Fatalf("respons tidak lengkap: %+v", d)
	}
}

func TestJudgeLoginHandler_RejectsMalformedPIN(t *testing.T) {
	pins, _, _, _ := pinTestEnv(t)
	h := newLoginHandler(pins)

	for _, body := range []string{`{}`, `{"pin":"12345"}`, `{"pin":"123456789"}`, `{"pin":"12ab56"}`} {
		if rec := postJudgeLogin(h, body); rec.Code != http.StatusBadRequest {
			t.Errorf("%s -> %d, want 400", body, rec.Code)
		}
	}
	// format salah tidak dihitung sebagai percobaan menebak
	if locked, _, _ := h.Guard.Check("10.0.0.7", ""); locked {
		t.Error("input tidak valid tidak boleh mengunci klien")
	}
}

func TestJudgeLoginHandler_LocksAfterRepeatedWrongPINs(t *testing.T) {
	pins, comp, arenaA, _ := pinTestEnv(t)
	h := newLoginHandler(pins)

	good := uniquePIN(t, "judge")
	if _, err := pins.CreatePINWithSeat(comp, arenaA, "juri", 1, good, time.Hour); err != nil {
		t.Fatal(err)
	}
	wrong := `{"pin":"000000"}`
	if good == "000000" {
		wrong = `{"pin":"000001"}`
	}

	// login berhasil di tengah mereset hitungan
	postJudgeLogin(h, wrong)
	postJudgeLogin(h, wrong)
	if rec := postJudgeLogin(h, `{"pin":"`+good+`"}`); rec.Code != http.StatusOK {
		t.Fatalf("login benar harus 200, got %d", rec.Code)
	}

	for i := 0; i < 3; i++ {
		if rec := postJudgeLogin(h, wrong); rec.Code != http.StatusUnauthorized {
			t.Fatalf("percobaan %d: status = %d, want 401", i+1, rec.Code)
		}
	}
	// terkunci: bahkan PIN yang benar ditolak sementara
	rec := postJudgeLogin(h, `{"pin":"`+good+`"}`)
	if rec.Code != http.StatusTooManyRequests || rec.Header().Get("Retry-After") == "" {
		t.Fatalf("harus 429 dengan Retry-After, got %d %v", rec.Code, rec.Header())
	}
}

func TestCreatePINHandler_ConflictAndFormat(t *testing.T) {
	pins, comp, arenaA, arenaB := pinTestEnv(t)
	h := &handler.AuthHandler{Pins: pins}

	post := func(body string) *httptest.ResponseRecorder {
		req := httptest.NewRequest("POST", "/api/v1/auth/create-pin", strings.NewReader(body))
		rec := httptest.NewRecorder()
		h.CreatePIN(rec, req)
		return rec
	}
	pin := uniquePIN(t, "judge")
	mk := func(arena, role string, seat int, p string) string {
		return fmt.Sprintf(`{"competition_id":%q,"arena_id":%q,"role":%q,"seat_number":%d,"pin":%q}`, comp, arena, role, seat, p)
	}

	if rec := post(mk(arenaA, "juri", 1, pin)); rec.Code != http.StatusOK {
		t.Fatalf("buat PIN: %d %s", rec.Code, rec.Body)
	}
	if rec := post(mk(arenaB, "juri", 1, pin)); rec.Code != http.StatusConflict {
		t.Fatalf("PIN bentrok harus 409, got %d %s", rec.Code, rec.Body)
	}
	if rec := post(mk(arenaA, "wasit", 0, "123456")); rec.Code != http.StatusBadRequest {
		t.Fatalf("PIN wasit 6 digit harus 400, got %d", rec.Code)
	}
	// tanpa PIN -> server membuat PIN acak sesuai role
	rec := post(fmt.Sprintf(`{"competition_id":%q,"arena_id":%q,"role":"wasit"}`, comp, arenaA))
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"pin":"`) {
		t.Fatalf("PIN acak wasit: %d %s", rec.Code, rec.Body)
	}
}

func postJudgeLoginAs(h *handler.AuthHandler, device, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest("POST", "/api/v1/auth/judge-login", strings.NewReader(body))
	req.RemoteAddr = "10.0.0.7:5555" // semua tablet berbagi satu IP (mis. Docker Desktop)
	if device != "" {
		req.Header.Set("X-Device-Id", device)
	}
	rec := httptest.NewRecorder()
	h.JudgeLogin(rec, req)
	return rec
}

// Tablet yang salah PIN berulang hanya mengunci dirinya sendiri; tablet lain di IP yang sama tetap bisa login.
func TestJudgeLoginHandler_WrongPINsOnOneTabletDoNotLockOthersOnSameIP(t *testing.T) {
	pins, comp, arenaA, _ := pinTestEnv(t)
	h := newLoginHandler(pins) // ambang perangkat 3

	good := uniquePIN(t, "judge")
	if _, err := pins.CreatePINWithSeat(comp, arenaA, "juri", 3, good, time.Hour); err != nil {
		t.Fatal(err)
	}
	wrong := `{"pin":"000001"}`
	for i := 0; i < 3; i++ {
		if rec := postJudgeLoginAs(h, "tablet-A-0001", wrong); rec.Code != http.StatusUnauthorized {
			t.Fatalf("PIN salah ke-%d -> %d, want 401", i+1, rec.Code)
		}
	}
	rec := postJudgeLoginAs(h, "tablet-A-0001", `{"pin":"`+good+`"}`)
	if rec.Code != http.StatusTooManyRequests || rec.Header().Get("Retry-After") == "" {
		t.Fatalf("tablet A harus terkunci walau PIN benar: %d %s", rec.Code, rec.Body)
	}
	if !strings.Contains(rec.Body.String(), `"scope":"device"`) {
		t.Errorf("respons harus menyebut lapisan penolak (device): %s", rec.Body)
	}

	// tablet B, IP yang sama, PIN benar: harus berhasil
	if rec := postJudgeLoginAs(h, "tablet-B-0002", `{"pin":"`+good+`"}`); rec.Code != http.StatusOK {
		t.Fatalf("tablet B ikut terkunci: %d %s", rec.Code, rec.Body)
	}
}

// Mengganti PIN sebuah kursi dengan PIN yang bentrok dengan kursi lain ditolak, dan PIN lama kursi itu tetap berlaku
// (sebelumnya PIN lama sudah terhapus sebelum penolakan sehingga kursi kehilangan PIN).
func TestCreatePIN_RejectedReplacementKeepsOldPIN(t *testing.T) {
	pins, comp, arenaA, _ := pinTestEnv(t)

	pin1, pin2 := uniquePIN(t, "judge"), uniquePIN(t, "judge")
	if _, err := pins.CreatePINWithSeat(comp, arenaA, "juri", 1, pin1, time.Hour); err != nil {
		t.Fatal(err)
	}
	if _, err := pins.CreatePINWithSeat(comp, arenaA, "juri", 2, pin2, time.Hour); err != nil {
		t.Fatal(err)
	}

	if _, err := pins.CreatePINWithSeat(comp, arenaA, "juri", 1, pin2, time.Hour); !errors.Is(err, auth.ErrPinTaken) {
		t.Fatalf("PIN bentrok harus ErrPinTaken, got %v", err)
	}
	if _, rec, err := pins.LoginWithPIN(pin1); err != nil || rec.SeatNumber != 1 {
		t.Fatalf("PIN lama kursi 1 harus tetap berlaku: rec=%+v err=%v", rec, err)
	}

	// penggantian yang sah tetap menukar PIN lama dengan yang baru
	pin3 := uniquePIN(t, "judge")
	if _, err := pins.CreatePINWithSeat(comp, arenaA, "juri", 1, pin3, time.Hour); err != nil {
		t.Fatal(err)
	}
	if _, _, err := pins.LoginWithPIN(pin1); !errors.Is(err, auth.ErrInvalidPIN) {
		t.Errorf("PIN lama setelah diganti harus ditolak, got %v", err)
	}
	if _, _, err := pins.LoginWithPIN(pin3); err != nil {
		t.Errorf("PIN baru harus berlaku: %v", err)
	}
}
