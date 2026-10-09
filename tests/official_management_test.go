package tests

import (
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/herros27/digital-scoring/internal/api/handler"
	"github.com/herros27/digital-scoring/internal/auth"
	"github.com/herros27/digital-scoring/internal/domain"
	"github.com/herros27/digital-scoring/internal/service"
	"github.com/herros27/digital-scoring/internal/store"
)

func newOfficialSvc(t *testing.T) (*service.OfficialService, *mockOfficialRepo) {
	t.Helper()
	repo := newMockOfficialRepo()
	return service.NewOfficialService(repo), repo
}

// seedOfficial membuat akun dengan ID unik (mock memberi ID tetap bila kosong).
func seedOfficial(t *testing.T, repo *mockOfficialRepo, id, email, role, password string) *domain.Official {
	t.Helper()
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.MinCost)
	if err != nil {
		t.Fatal(err)
	}
	o := &domain.Official{ID: id, Name: "Nama " + id, Email: email, Role: role}
	if err := repo.Create(o, string(hash)); err != nil {
		t.Fatal(err)
	}
	return o
}

func TestCanManageRole_Matrix(t *testing.T) {
	cases := []struct {
		actor, target string
		want          bool
	}{
		{"admin", "admin", true}, {"admin", "dewan", true}, {"admin", "wasit", true}, {"admin", "operator", true},
		{"dewan", "admin", false}, {"dewan", "dewan", false},
		{"dewan", "wasit", true}, {"dewan", "operator", true}, {"dewan", "juri", true},
		{"wasit", "operator", false}, {"operator", "operator", false}, {"contingent", "operator", false}, {"", "operator", false},
	}
	for _, c := range cases {
		if got := service.CanManageRole(c.actor, c.target); got != c.want {
			t.Errorf("CanManageRole(%q,%q) = %v, want %v", c.actor, c.target, got, c.want)
		}
	}
}

func TestOfficialService_DewanTidakBolehMembuatAdminAtauDewan(t *testing.T) {
	svc, _ := newOfficialSvc(t)
	dewan := service.Actor{ID: "d1", Role: "dewan"}
	for _, role := range []string{"admin", "dewan"} {
		if _, err := svc.CreateOfficialBy(dewan, "X Y", role+"@silat.id", "test_password_1", role); !errors.Is(err, domain.ErrForbidden) {
			t.Errorf("dewan membuat %s: err = %v, want ErrForbidden", role, err)
		}
	}
	if _, err := svc.CreateOfficialBy(dewan, "Wasit Baru", "wasit.baru@silat.id", "test_password_1", "wasit"); err != nil {
		t.Errorf("dewan membuat wasit harus boleh: %v", err)
	}
}

func TestOfficialService_PasswordTerlaluPendekDitolak(t *testing.T) {
	svc, _ := newOfficialSvc(t)
	if _, err := svc.CreateOfficialBy(service.Actor{ID: "a", Role: "admin"}, "Nama", "a@silat.id", "short", "operator"); !errors.Is(err, domain.ErrInvalidInput) {
		t.Errorf("password pendek: err = %v, want ErrInvalidInput", err)
	}
}

func TestOfficialService_AdminTerakhirDilindungi(t *testing.T) {
	svc, repo := newOfficialSvc(t)
	seedOfficial(t, repo, "adm1", "adm1@silat.id", "admin", "test_password_1")
	actor := service.Actor{ID: "adm1", Role: "admin"}

	if _, err := svc.UpdateOfficialBy(actor, "adm1", "Nama", "operator"); !errors.Is(err, domain.ErrConflict) {
		t.Errorf("menurunkan admin terakhir: err = %v, want ErrConflict", err)
	}
	// admin kedua: admin pertama boleh dihapus oleh admin kedua, tetapi tidak oleh dirinya sendiri
	seedOfficial(t, repo, "adm2", "adm2@silat.id", "admin", "test_password_1")
	if err := svc.DeleteOfficialBy(actor, "adm1"); !errors.Is(err, domain.ErrConflict) {
		t.Errorf("hapus akun sendiri: err = %v, want ErrConflict", err)
	}
	if err := svc.DeleteOfficialBy(service.Actor{ID: "adm2", Role: "admin"}, "adm1"); err != nil {
		t.Errorf("admin kedua menghapus admin pertama: %v", err)
	}
	// kini adm2 satu-satunya admin
	if err := svc.DeleteOfficialBy(service.Actor{ID: "x", Role: "admin"}, "adm2"); !errors.Is(err, domain.ErrConflict) {
		t.Errorf("hapus admin terakhir: err = %v, want ErrConflict", err)
	}
}

func TestOfficialService_DewanTidakBolehMengubahAtauMenghapusAdmin(t *testing.T) {
	svc, repo := newOfficialSvc(t)
	seedOfficial(t, repo, "adm", "adm@silat.id", "admin", "test_password_1")
	seedOfficial(t, repo, "op", "op@silat.id", "operator", "test_password_1")
	dewan := service.Actor{ID: "d", Role: "dewan"}

	if _, err := svc.UpdateOfficialBy(dewan, "adm", "Nama Baru", "admin"); !errors.Is(err, domain.ErrForbidden) {
		t.Errorf("ubah admin oleh dewan: %v", err)
	}
	if err := svc.DeleteOfficialBy(dewan, "adm"); !errors.Is(err, domain.ErrForbidden) {
		t.Errorf("hapus admin oleh dewan: %v", err)
	}
	if _, err := svc.UpdateOfficialBy(dewan, "op", "Operator", "admin"); !errors.Is(err, domain.ErrForbidden) {
		t.Errorf("menaikkan operator jadi admin oleh dewan: %v", err)
	}
	if _, err := svc.UpdateOfficialBy(dewan, "op", "Operator Baru", "wasit"); err != nil {
		t.Errorf("dewan mengubah operator jadi wasit: %v", err)
	}
	if err := svc.DeleteOfficialBy(dewan, "tidak-ada"); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("hapus akun tak ada: %v", err)
	}
}

func TestOfficialService_ResetPassword(t *testing.T) {
	svc, repo := newOfficialSvc(t)
	seedOfficial(t, repo, "adm", "adm@silat.id", "admin", "test_password_1")
	seedOfficial(t, repo, "ws", "ws@silat.id", "wasit", "mock_pass_old")

	if err := svc.ResetPassword(service.Actor{ID: "adm", Role: "admin"}, "ws", "mock_pass_reset"); err != nil {
		t.Fatalf("reset: %v", err)
	}
	if _, _, err := svc.Login("ws@silat.id", "mock_pass_old"); err == nil {
		t.Error("password lama masih bisa dipakai setelah reset")
	}
	if _, _, err := svc.Login("ws@silat.id", "mock_pass_reset"); err != nil {
		t.Errorf("password baru tidak bisa dipakai: %v", err)
	}
	if err := svc.ResetPassword(service.Actor{ID: "adm", Role: "admin"}, "ws", "pendek"); !errors.Is(err, domain.ErrInvalidInput) {
		t.Errorf("password pendek: %v", err)
	}
	if err := svc.ResetPassword(service.Actor{ID: "d", Role: "dewan"}, "adm", "mock_pass_reset"); !errors.Is(err, domain.ErrForbidden) {
		t.Errorf("dewan mereset admin: %v", err)
	}
	if err := svc.ResetPassword(service.Actor{ID: "adm", Role: "admin"}, "tidak-ada", "mock_pass_reset"); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("reset akun tak ada: %v", err)
	}
}

func TestOfficialService_ChangeOwnPassword(t *testing.T) {
	svc, repo := newOfficialSvc(t)
	seedOfficial(t, repo, "op", "op@silat.id", "operator", "mock_pass_old")

	if err := svc.ChangeOwnPassword("op", "salah-salah", "mock_pass_reset"); !errors.Is(err, domain.ErrInvalidInput) {
		t.Errorf("password lama salah: %v", err)
	}
	if err := svc.ChangeOwnPassword("op", "mock_pass_old", "pendek"); !errors.Is(err, domain.ErrInvalidInput) {
		t.Errorf("password baru pendek: %v", err)
	}
	if err := svc.ChangeOwnPassword("op", "mock_pass_old", "mock_pass_old"); !errors.Is(err, domain.ErrInvalidInput) {
		t.Errorf("password baru sama dengan lama: %v", err)
	}
	if err := svc.ChangeOwnPassword("op", "mock_pass_old", "mock_pass_reset"); err != nil {
		t.Fatalf("ganti password: %v", err)
	}
	if _, _, err := svc.Login("op@silat.id", "mock_pass_reset"); err != nil {
		t.Errorf("login dengan password baru: %v", err)
	}
	if _, _, err := svc.Login("op@silat.id", "mock_pass_old"); err == nil {
		t.Error("password lama masih berlaku")
	}
}

func TestOfficialService_EnsureAdmin(t *testing.T) {
	t.Run("tanpa konfigurasi tidak melakukan apa pun", func(t *testing.T) {
		svc, repo := newOfficialSvc(t)
		created, err := svc.EnsureAdmin("", "", "")
		if created || err != nil || len(repo.officials) != 0 {
			t.Errorf("created=%v err=%v officials=%d", created, err, len(repo.officials))
		}
	})
	t.Run("membuat admin pertama yang bisa login, lalu idempoten", func(t *testing.T) {
		svc, repo := newOfficialSvc(t)
		created, err := svc.EnsureAdmin("Ketua Panitia", "ketua@lomba.id", "test_pass_valid_1")
		if err != nil || !created {
			t.Fatalf("created=%v err=%v", created, err)
		}
		if _, o, err := svc.Login("ketua@lomba.id", "test_pass_valid_1"); err != nil || o.Role != "admin" || o.Name != "Ketua Panitia" {
			t.Fatalf("login admin awal: %+v %v", o, err)
		}
		// panggilan ulang (restart) tidak membuat akun lagi dan tidak menimpa password yang sudah diganti
		if err := svc.ChangeOwnPassword(repo.firstID(), "test_pass_valid_1", "mock_pass_admin_replaced"); err != nil {
			t.Fatal(err)
		}
		created, err = svc.EnsureAdmin("Ketua Panitia", "ketua@lomba.id", "test_pass_valid_1")
		if created || err != nil {
			t.Errorf("ulang: created=%v err=%v", created, err)
		}
		if _, _, err := svc.Login("ketua@lomba.id", "mock_pass_admin_replaced"); err != nil {
			t.Errorf("password hasil ganti tertimpa saat restart: %v", err)
		}
	})
	t.Run("tidak membuat admin baru bila sudah ada admin lain", func(t *testing.T) {
		svc, repo := newOfficialSvc(t)
		seedOfficial(t, repo, "adm", "lama@silat.id", "admin", "test_password_1")
		created, err := svc.EnsureAdmin("Baru", "baru@silat.id", "test_pass_valid_1")
		if created || err != nil || len(repo.officials) != 1 {
			t.Errorf("created=%v err=%v officials=%d", created, err, len(repo.officials))
		}
	})
	t.Run("konfigurasi salah ditolak", func(t *testing.T) {
		svc, _ := newOfficialSvc(t)
		for name, c := range map[string][2]string{
			"hanya email":     {"a@silat.id", ""},
			"hanya password":  {"", "test_pass_valid_1"},
			"email jelek":     {"bukan-email", "test_pass_valid_1"},
			"password pendek": {"a@silat.id", "pendek"},
		} {
			if _, err := svc.EnsureAdmin("", c[0], c[1]); !errors.Is(err, domain.ErrInvalidInput) {
				t.Errorf("%s: err = %v, want ErrInvalidInput", name, err)
			}
		}
	})
}

// --- HTTP ---------------------------------------------------------------------------------------------------------

func officialReq(method, path, body, actorID, role string) *http.Request {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	if role != "" {
		req = req.WithContext(auth.ContextWithClaims(req.Context(), &auth.Claims{JudgePinID: actorID, Role: role}))
	}
	return req
}

func TestOfficialHandler_StatusMapping(t *testing.T) {
	repo := newMockOfficialRepo()
	svc := service.NewOfficialService(repo)
	h := &handler.OfficialHandler{Service: svc}
	seedOfficial(t, repo, "adm", "adm@silat.id", "admin", "test_password_1")
	seedOfficial(t, repo, "ws", "ws@silat.id", "wasit", "test_password_1")

	do := func(fn http.HandlerFunc, req *http.Request, id string) int {
		if id != "" {
			req.SetPathValue("id", id)
		}
		w := httptest.NewRecorder()
		fn(w, req)
		return w.Code
	}

	if c := do(h.Create, officialReq("POST", "/officials", `{"name":"Dewan B","email":"b@silat.id","password":"test_password_1","role":"dewan"}`, "d", "dewan"), ""); c != http.StatusForbidden {
		t.Errorf("dewan membuat dewan: %d, want 403", c)
	}
	if c := do(h.Create, officialReq("POST", "/officials", `{"name":"Dup","email":"ws@silat.id","password":"test_password_1","role":"wasit"}`, "adm", "admin"), ""); c != http.StatusConflict {
		t.Errorf("email ganda: %d, want 409", c)
	}
	if c := do(h.Create, officialReq("POST", "/officials", `{"name":"Pendek","email":"p@silat.id","password":"abc","role":"wasit"}`, "adm", "admin"), ""); c != http.StatusBadRequest {
		t.Errorf("password pendek: %d, want 400", c)
	}
	if c := do(h.Create, officialReq("POST", "/officials", `{"name":"Tanpa Klaim","email":"t@silat.id","password":"test_password_1","role":"wasit"}`, "", ""), ""); c != http.StatusForbidden {
		t.Errorf("tanpa klaim: %d, want 403", c)
	}
	if c := do(h.Delete, officialReq("DELETE", "/officials/adm", "", "adm", "admin"), "adm"); c != http.StatusConflict {
		t.Errorf("hapus diri sendiri: %d, want 409", c)
	}
	if c := do(h.Delete, officialReq("DELETE", "/officials/x", "", "adm", "admin"), "tidak-ada"); c != http.StatusNotFound {
		t.Errorf("hapus tak ada: %d, want 404", c)
	}
	if c := do(h.ResetPassword, officialReq("PUT", "/officials/ws/password", `{"password":"mock_pass_reset"}`, "adm", "admin"), "ws"); c != http.StatusOK {
		t.Errorf("reset: %d, want 200", c)
	}
	if c := do(h.ResetPassword, officialReq("PUT", "/officials/ws/password", `{"password":"pendek"}`, "adm", "admin"), "ws"); c != http.StatusBadRequest {
		t.Errorf("reset pendek: %d, want 400", c)
	}
	if c := do(h.ResetPassword, officialReq("PUT", "/officials/adm/password", `{"password":"mock_pass_reset"}`, "d", "dewan"), "adm"); c != http.StatusForbidden {
		t.Errorf("dewan mereset admin: %d, want 403", c)
	}
	if c := do(h.ChangePassword, officialReq("POST", "/auth/change-password", `{"current_password":"salah","new_password":"mock_pass_reset"}`, "ws", "wasit"), ""); c != http.StatusBadRequest {
		t.Errorf("password lama salah: %d, want 400 (bukan 401: klien akan keluar dari sesi)", c)
	}
	if c := do(h.ChangePassword, officialReq("POST", "/auth/change-password", `{"current_password":"mock_pass_reset","new_password":"mock_pass_newer"}`, "ws", "wasit"), ""); c != http.StatusOK {
		t.Errorf("ganti password: %d, want 200", c)
	}
}

// --- Database nyata -------------------------------------------------------------------------------------------------

func TestOfficialStore_PasswordDanHitungPeran(t *testing.T) {
	dsn := os.Getenv("TEST_DB_URL")
	if dsn == "" {
		dsn = "postgres://scoring_user@localhost:5432/digital_scoring?sslmode=disable"
	}
	db, err := sql.Open("pgx", dsn)
	if err != nil || db.Ping() != nil {
		t.Skip("database tidak tersedia; tes store petugas dilewati")
	}
	t.Cleanup(func() { db.Close() }) // didaftarkan lebih dulu agar berjalan SETELAH cleanup penghapusan akun di bawah
	if err := store.RunMigrations(db); err != nil {
		t.Fatalf("migrasi: %v", err)
	}
	repo := &store.OfficialStore{DB: db}
	svc := service.NewOfficialService(repo)

	before, err := repo.CountByRole("operator")
	if err != nil {
		t.Fatal(err)
	}
	email := fmt.Sprintf("uji-%d@silat.test", time.Now().UnixNano())
	o, err := svc.CreateOfficialBy(service.Actor{ID: "a", Role: "admin"}, "Uji Petugas", email, "mock_pass_test1", "operator")
	if err != nil {
		t.Fatalf("buat: %v", err)
	}
	t.Cleanup(func() { _, _ = db.Exec(`DELETE FROM officials WHERE id = $1`, o.ID) })

	if n, _ := repo.CountByRole("operator"); n != before+1 {
		t.Errorf("CountByRole = %d, want %d", n, before+1)
	}
	if err := svc.ResetPassword(service.Actor{ID: "a", Role: "admin"}, o.ID, "mock_pass_test2"); err != nil {
		t.Fatalf("reset: %v", err)
	}
	if _, _, err := svc.Login(email, "mock_pass_test1"); err == nil {
		t.Error("password lama masih berlaku di database")
	}
	if _, _, err := svc.Login(email, "mock_pass_test2"); err != nil {
		t.Errorf("login password baru: %v", err)
	}
	if err := svc.ChangeOwnPassword(o.ID, "mock_pass_test2", "mock_pass_test3"); err != nil {
		t.Fatalf("ganti sendiri: %v", err)
	}
	if _, err := svc.GetOfficialByID("00000000-0000-0000-0000-000000000000"); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("GetOfficialByID tak ada: err = %v, want ErrNotFound", err)
	}
}
