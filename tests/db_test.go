package tests

import (
	"database/sql"
	"errors"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/joho/godotenv"
)

const defaultTestDSN = "postgres://scoring_user@localhost:5432/digital_scoring?sslmode=disable"

// testDSN alamat database untuk tes, berurutan: TEST_DB_URL (shell/CI), DB_URL dari .env di akar repo, lalu bawaan
// lokal tanpa password. DB_URL dibaca langsung dari berkas .env (bukan dari shell, supaya DB_URL milik terminal lain
// tidak terpakai diam-diam) dan hanya dipakai bila menunjuk ke mesin ini: tes membuat dan menghapus data.
func testDSN() string {
	if dsn := os.Getenv("TEST_DB_URL"); dsn != "" {
		return dsn
	}
	if env, err := godotenv.Read(filepath.Join("..", ".env")); err == nil {
		if dsn := env["DB_URL"]; dsn != "" && isLocalDSN(dsn) {
			return dsn
		}
	}
	return defaultTestDSN
}

func isLocalDSN(dsn string) bool {
	u, err := url.Parse(dsn)
	if err != nil {
		return false
	}
	switch u.Hostname() {
	case "localhost", "127.0.0.1", "::1":
		return true
	}
	return false
}

// openTestDB membuka database tes. Tes dilewati hanya bila server database memang tidak terjangkau. Bila server ada
// tetapi menolak sambungan karena password, tes GAGAL dengan petunjuk: dilewati diam-diam akan terlihat seperti lulus.
func openTestDB(t *testing.T, purpose string) *sql.DB {
	t.Helper()
	dsn := testDSN()
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Skipf("database tidak bisa dibuka (%v); %s dilewati", err, purpose)
	}
	if err := db.Ping(); err != nil {
		db.Close()
		var pgErr *pgconn.PgError
		if (errors.As(err, &pgErr) && strings.HasPrefix(pgErr.Code, "28")) || strings.Contains(err.Error(), "password") {
			t.Fatalf("database menolak sambungan tes (%v): isi password di DB_URL pada .env, atau set TEST_DB_URL", err)
		}
		t.Skipf("database tidak tersedia (%v); %s dilewati", err, purpose)
	}
	return db
}
