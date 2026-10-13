package tests

import (
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/herros27/digital-scoring/internal/api/handler"
	"github.com/herros27/digital-scoring/internal/auth"
	"github.com/herros27/digital-scoring/internal/domain"
	"github.com/herros27/digital-scoring/internal/hub"
	"github.com/herros27/digital-scoring/internal/service"
	"github.com/herros27/digital-scoring/internal/store"
)

// --- Kontingen: reset dan ganti password -------------------------------------------------------------------------

func seedContingent(t *testing.T, svc *service.ContingentService) *domain.Contingent {
	t.Helper()
	c, err := svc.Register("Tapak Suci", "Pak Budi", "ts@kontingen.id", "081234567", "password-lama-1")
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func TestContingentService_ResetPassword(t *testing.T) {
	repo := newMockContingentRepo()
	svc := service.NewContingentService(repo)
	c := seedContingent(t, svc)

	if err := svc.ResetPassword(c.ID, "pendek"); !errors.Is(err, domain.ErrInvalidInput) {
		t.Errorf("password pendek: %v", err)
	}
	if err := svc.ResetPassword("tidak-ada", "password-baru-9"); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("kontingen tak ada: %v", err)
	}
	if err := svc.ResetPassword(c.ID, "password-baru-9"); err != nil {
		t.Fatalf("reset: %v", err)
	}
	if _, _, err := svc.Login(c.Email, "password-lama-1"); err == nil {
		t.Error("password lama masih berlaku")
	}
	if _, _, err := svc.Login(c.Email, "password-baru-9"); err != nil {
		t.Errorf("password baru tidak berlaku: %v", err)
	}
}

func TestContingentService_ChangeOwnPassword(t *testing.T) {
	repo := newMockContingentRepo()
	svc := service.NewContingentService(repo)
	c := seedContingent(t, svc)

	if err := svc.ChangeOwnPassword(c.ID, "mock_pass_wrong", "password-baru-9"); !errors.Is(err, domain.ErrInvalidInput) {
		t.Errorf("password lama salah harus 400-ish (ErrInvalidInput): %v", err)
	}
	if err := svc.ChangeOwnPassword(c.ID, "password-lama-1", "password-lama-1"); !errors.Is(err, domain.ErrInvalidInput) {
		t.Errorf("password sama: %v", err)
	}
	if err := svc.ChangeOwnPassword(c.ID, "password-lama-1", "password-baru-9"); err != nil {
		t.Fatalf("ganti: %v", err)
	}
	if _, _, err := svc.Login(c.Email, "password-baru-9"); err != nil {
		t.Errorf("login password baru: %v", err)
	}
}

func TestContingentHandler_PasswordEndpoints(t *testing.T) {
	repo := newMockContingentRepo()
	svc := service.NewContingentService(repo)
	c := seedContingent(t, svc)
	h := &handler.ContingentHandler{Service: svc}

	call := func(fn http.HandlerFunc, body, role, claimID, pathID string) int {
		req := httptest.NewRequest("PUT", "/x", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		if role != "" {
			req = req.WithContext(auth.ContextWithClaims(req.Context(), &auth.Claims{JudgePinID: claimID, Role: role}))
		}
		if pathID != "" {
			req.SetPathValue("id", pathID)
		}
		w := httptest.NewRecorder()
		fn(w, req)
		return w.Code
	}
	if code := call(h.ResetPassword, `{"password":"password-baru-9"}`, "admin", "a", c.ID); code != http.StatusOK {
		t.Errorf("reset: %d", code)
	}
	if code := call(h.ResetPassword, `{"password":"pendek"}`, "admin", "a", c.ID); code != http.StatusBadRequest {
		t.Errorf("reset pendek: %d", code)
	}
	if code := call(h.ResetPassword, `{"password":"password-baru-9"}`, "admin", "a", "tidak-ada"); code != http.StatusNotFound {
		t.Errorf("reset tak ada: %d", code)
	}
	if code := call(h.ChangePassword, `{"current_password":"salah","new_password":"password-baru-8"}`, "contingent", c.ID, ""); code != http.StatusBadRequest {
		t.Errorf("password lama salah: %d, want 400", code)
	}
	if code := call(h.ChangePassword, `{"current_password":"password-baru-9","new_password":"password-baru-8"}`, "contingent", c.ID, ""); code != http.StatusOK {
		t.Errorf("ganti sendiri: %d", code)
	}
	if code := call(h.ChangePassword, `{"current_password":"x","new_password":"password-baru-8"}`, "admin", "a", ""); code != http.StatusForbidden {
		t.Errorf("bukan kontingen: %d, want 403", code)
	}
}

// --- Kategori: ubah dan hapus (aturan, tanpa database) ------------------------------------------------------------

type editableCategoryRepo struct {
	*mockCategoryRepo
	participants map[string]int
	deleted      []string
}

func newEditableCategoryRepo() *editableCategoryRepo {
	return &editableCategoryRepo{mockCategoryRepo: &mockCategoryRepo{categories: map[string]*domain.Category{}}, participants: map[string]int{}}
}

func (r *editableCategoryRepo) Update(c *domain.Category) error {
	cur, ok := r.categories[c.ID]
	if !ok || cur.CompetitionID != c.CompetitionID {
		return domain.ErrNotFound
	}
	*cur = *c
	return nil
}
func (r *editableCategoryRepo) Delete(comp, id string) error {
	if r.participants[id] > 0 {
		return domain.ErrConflict
	}
	r.deleted = append(r.deleted, id)
	delete(r.categories, id)
	return nil
}
func (r *editableCategoryRepo) ParticipantCount(id string) (int, error) {
	return r.participants[id], nil
}

func putCategory(r *editableCategoryRepo, id string) *domain.Category {
	min, max := 50.0, 55.0
	c := &domain.Category{ID: id, CompetitionID: "c1", Name: "Tanding Dewasa Putra B", Gender: "male", WeightClass: "B",
		AgeGroup: domain.AgeDewasa, PresetStandard: domain.PresetIPSI, BracketMode: domain.BracketSingleElim, MinWeight: &min, MaxWeight: &max}
	r.categories[id] = c
	return c
}

func TestCategoryService_Update_TanpaPeserta(t *testing.T) {
	repo := newEditableCategoryRepo()
	putCategory(repo, "k1")
	svc := service.NewCategoryService(repo)
	min, max := 55.0, 60.0

	got, err := svc.UpdateCategory("c1", "k1", &domain.Category{Name: "Tanding Dewasa Putra C", Gender: "putra", WeightClass: "C", MinWeight: &min, MaxWeight: &max})
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	if got.Name != "Tanding Dewasa Putra C" || got.WeightClass != "C" || *got.MinWeight != 55 || *got.MaxWeight != 60 {
		t.Errorf("hasil update tidak sesuai: %+v", got)
	}
	if got.PresetStandard != domain.PresetIPSI || got.AgeGroup != domain.AgeDewasa {
		t.Errorf("standar dan golongan umur harus dipertahankan: %+v", got)
	}
}

func TestCategoryService_Update_Aturan(t *testing.T) {
	repo := newEditableCategoryRepo()
	putCategory(repo, "k1")
	svc := service.NewCategoryService(repo)
	min, max := 50.0, 55.0
	base := func() *domain.Category {
		return &domain.Category{Name: "Nama Baru", Gender: "male", WeightClass: "B", MinWeight: &min, MaxWeight: &max}
	}

	if _, err := svc.UpdateCategory("c-lain", "k1", base()); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("kejuaraan lain: %v", err)
	}
	if _, err := svc.UpdateCategory("c1", "tidak-ada", base()); err == nil {
		t.Error("kategori tak ada harus galat")
	}
	in := base()
	in.PresetStandard = domain.PresetPersilat26
	if _, err := svc.UpdateCategory("c1", "k1", in); !errors.Is(err, domain.ErrConflict) {
		t.Errorf("ganti standar: %v, want ErrConflict", err)
	}
	in = base()
	bad := 70.0
	in.MinWeight = &bad // min 70 > max 55
	if _, err := svc.UpdateCategory("c1", "k1", in); !errors.Is(err, domain.ErrInvalidInput) {
		t.Errorf("min > max: %v, want ErrInvalidInput", err)
	}
	in = base()
	in.AgeGroup = "singa" // golongan PERSILAT pada preset IPSI
	if _, err := svc.UpdateCategory("c1", "k1", in); !errors.Is(err, domain.ErrInvalidInput) {
		t.Errorf("golongan umur salah standar: %v", err)
	}
}

func TestCategoryService_Update_SetelahAdaPeserta_HanyaNama(t *testing.T) {
	repo := newEditableCategoryRepo()
	putCategory(repo, "k1")
	repo.participants["k1"] = 3
	svc := service.NewCategoryService(repo)
	min, max := 50.0, 55.0

	if got, err := svc.UpdateCategory("c1", "k1", &domain.Category{Name: "Hanya Ganti Nama", Gender: "putra", WeightClass: "B", MinWeight: &min, MaxWeight: &max}); err != nil || got.Name != "Hanya Ganti Nama" {
		t.Errorf("ganti nama (gender putra setara male) harus lolos: %v %+v", err, got)
	}
	newMax := 60.0
	if _, err := svc.UpdateCategory("c1", "k1", &domain.Category{Name: "X Y", Gender: "male", WeightClass: "B", MinWeight: &min, MaxWeight: &newMax}); !errors.Is(err, domain.ErrConflict) {
		t.Errorf("ubah batas bobot saat ada peserta: %v, want ErrConflict", err)
	}
	if _, err := svc.UpdateCategory("c1", "k1", &domain.Category{Name: "X Y", Gender: "female", WeightClass: "B", MinWeight: &min, MaxWeight: &max}); !errors.Is(err, domain.ErrConflict) {
		t.Errorf("ubah gender saat ada peserta: %v, want ErrConflict", err)
	}
}

func TestCategoryService_Delete(t *testing.T) {
	repo := newEditableCategoryRepo()
	putCategory(repo, "k1")
	putCategory(repo, "k2")
	repo.participants["k2"] = 1
	svc := service.NewCategoryService(repo)

	if err := svc.DeleteCategory("c1", "k1"); err != nil {
		t.Errorf("hapus kategori kosong: %v", err)
	}
	if err := svc.DeleteCategory("c1", "k2"); !errors.Is(err, domain.ErrConflict) {
		t.Errorf("hapus kategori berpeserta: %v, want ErrConflict", err)
	}
	if err := svc.DeleteCategory("", "k2"); !errors.Is(err, domain.ErrInvalidInput) {
		t.Errorf("id kosong: %v", err)
	}
}

// --- Database nyata: ubah/hapus kategori dan batal daftar peserta --------------------------------------------------

type adminDB struct {
	db         *sql.DB
	comp       string
	categories *store.CategoryStore
	athletes   *store.AthleteStore
	catSvc     *service.CategoryService
	athSvc     *service.AthleteService
}

func newAdminDB(t *testing.T) *adminDB {
	t.Helper()
	db := openTestDB(t, "tes administrasi kategori/peserta")
	if err := store.RunMigrations(db); err != nil {
		t.Fatalf("migrasi: %v", err)
	}
	a := &adminDB{db: db, categories: &store.CategoryStore{DB: db}, athletes: &store.AthleteStore{DB: db}}
	a.catSvc = service.NewCategoryService(a.categories)
	a.athSvc = service.NewAthleteService(a.athletes).WithCategoryRepo(a.categories)
	if err := db.QueryRow(`INSERT INTO competitions (name, location, start_date, end_date, status)
		VALUES ($1, 'x', '2026-06-01', '2026-06-03', 'draft') RETURNING id`, fmt.Sprintf("Admin test %d", time.Now().UnixNano())).Scan(&a.comp); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _, _ = db.Exec(`DELETE FROM competitions WHERE id = $1`, a.comp); db.Close() })
	return a
}

func (a *adminDB) category(t *testing.T, name string) *domain.Category {
	t.Helper()
	min, max := 50.0, 55.0
	c := &domain.Category{CompetitionID: a.comp, Name: name, Gender: "male", WeightClass: "B", MinWeight: &min, MaxWeight: &max}
	if err := a.catSvc.CreateCategory(c); err != nil {
		t.Fatal(err)
	}
	return c
}

func (a *adminDB) contingent(t *testing.T, tag string) string {
	t.Helper()
	var id string
	if err := a.db.QueryRow(`INSERT INTO contingents (name, pic_name, email, phone, password_hash) VALUES ($1,'PIC',$2,'081','x') RETURNING id`,
		"Kontingen "+tag, fmt.Sprintf("%s-%d@kontingen.test", tag, time.Now().UnixNano())).Scan(&id); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _, _ = a.db.Exec(`DELETE FROM contingents WHERE id = $1`, id) })
	return id
}

func (a *adminDB) participant(t *testing.T, cat *domain.Category, contingentID, status string) string {
	t.Helper()
	ath := &domain.Athlete{Name: "Atlet " + status, Gender: "male", Team: "Tim", ContingentID: contingentID, BirthDate: "1999-01-01"}
	if err := a.athletes.Create(ath); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _, _ = a.db.Exec(`DELETE FROM athletes WHERE id = $1`, ath.ID) })
	if err := a.athletes.RegisterParticipantInCategory(a.comp, ath.ID, contingentID, cat.ID, cat.WeightClass, "male", status); err != nil {
		t.Fatal(err)
	}
	var pid string
	if err := a.db.QueryRow(`SELECT id FROM participants WHERE athlete_id = $1 AND competition_id = $2`, ath.ID, a.comp).Scan(&pid); err != nil {
		t.Fatal(err)
	}
	return pid
}

func TestCategoryStore_UpdateDanDelete(t *testing.T) {
	a := newAdminDB(t)
	cat := a.category(t, "Kategori Uji A")
	min, max := 55.0, 60.0

	got, err := a.catSvc.UpdateCategory(a.comp, cat.ID, &domain.Category{Name: "Kategori Uji A2", Gender: "male", WeightClass: "C", MinWeight: &min, MaxWeight: &max})
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	if got.Name != "Kategori Uji A2" || got.WeightClass != "C" {
		t.Errorf("update tidak tercermin: %+v", got)
	}
	var name, wc string
	var gotMax float64
	if err := a.db.QueryRow(`SELECT name, weight_class, max_weight FROM categories WHERE id = $1`, cat.ID).Scan(&name, &wc, &gotMax); err != nil || name != "Kategori Uji A2" || wc != "C" || gotMax != 60 {
		t.Errorf("baris DB: %v %s %s %v", err, name, wc, gotMax)
	}
	if _, err := a.catSvc.UpdateCategory("00000000-0000-0000-0000-000000000000", cat.ID, &domain.Category{Name: "Salah Kejuaraan", Gender: "male", WeightClass: "C"}); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("kejuaraan lain: %v", err)
	}

	// kategori berpeserta: tidak bisa dihapus, hanya nama yang bisa diubah
	cont := a.contingent(t, "uji")
	pid := a.participant(t, cat, cont, "pending")
	if err := a.catSvc.DeleteCategory(a.comp, cat.ID); !errors.Is(err, domain.ErrConflict) {
		t.Errorf("hapus kategori berpeserta: %v, want ErrConflict", err)
	}
	if _, err := a.catSvc.UpdateCategory(a.comp, cat.ID, &domain.Category{Name: "Kategori Uji A3", Gender: "male", WeightClass: "D", MinWeight: &min, MaxWeight: &max}); !errors.Is(err, domain.ErrConflict) {
		t.Errorf("ubah kelas saat ada peserta: %v, want ErrConflict", err)
	}
	if _, err := a.catSvc.UpdateCategory(a.comp, cat.ID, &domain.Category{Name: "Kategori Uji A3", Gender: "male", WeightClass: "C", MinWeight: &min, MaxWeight: &max}); err != nil {
		t.Errorf("ganti nama saat ada peserta: %v", err)
	}

	// setelah peserta dibatalkan, kategori dapat dihapus
	if err := a.athSvc.UnregisterParticipant(a.comp, pid, "", false); err != nil {
		t.Fatalf("batal daftar: %v", err)
	}
	if err := a.catSvc.DeleteCategory(a.comp, cat.ID); err != nil {
		t.Errorf("hapus kategori kosong: %v", err)
	}
	if err := a.catSvc.DeleteCategory(a.comp, cat.ID); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("hapus dua kali: %v, want ErrNotFound", err)
	}
}

func TestCategoryStore_Delete_MenghapusSlotBaganDanTolakPartaiBerjalan(t *testing.T) {
	a := newAdminDB(t)
	cat := a.category(t, "Kategori Uji Bagan")
	var slot string
	if err := a.db.QueryRow(`INSERT INTO bracket_slots (competition_id, weight_class, gender, round, slot_index, category_id)
		VALUES ($1,'B','male',1,0,$2) RETURNING id`, a.comp, cat.ID).Scan(&slot); err != nil {
		t.Fatal(err)
	}
	var match string
	if err := a.db.QueryRow(`INSERT INTO matches (competition_id, bracket_slot_id, status) VALUES ($1,$2,'running') RETURNING id`, a.comp, slot).Scan(&match); err != nil {
		t.Skipf("tidak dapat menyiapkan partai uji (skema berbeda): %v", err)
	}
	if err := a.catSvc.DeleteCategory(a.comp, cat.ID); !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("hapus kategori dengan partai berjalan: %v, want ErrConflict", err)
	}
	if _, err := a.db.Exec(`UPDATE matches SET status = 'idle' WHERE id = $1`, match); err != nil {
		t.Fatal(err)
	}
	if err := a.catSvc.DeleteCategory(a.comp, cat.ID); err != nil {
		t.Fatalf("hapus kategori dengan partai idle: %v", err)
	}
	var slots, matches int
	_ = a.db.QueryRow(`SELECT COUNT(*) FROM bracket_slots WHERE id = $1`, slot).Scan(&slots)
	_ = a.db.QueryRow(`SELECT COUNT(*) FROM matches WHERE id = $1`, match).Scan(&matches)
	if slots != 0 || matches != 0 {
		t.Errorf("slot (%d) dan partai idle (%d) harus ikut terhapus", slots, matches)
	}
}

func TestUnregisterParticipant_AturanKepemilikanDanBagan(t *testing.T) {
	a := newAdminDB(t)
	cat := a.category(t, "Kategori Uji Peserta")
	own, other := a.contingent(t, "milik"), a.contingent(t, "lain")

	pOther := a.participant(t, cat, other, "pending")
	if err := a.athSvc.UnregisterParticipant(a.comp, pOther, own, true); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("peserta kontingen lain: %v, want ErrNotFound (tidak membocorkan keberadaan)", err)
	}

	pVerified := a.participant(t, cat, own, "verified")
	if err := a.athSvc.UnregisterParticipant(a.comp, pVerified, own, true); !errors.Is(err, domain.ErrForbidden) {
		t.Errorf("peserta sudah diverifikasi: %v, want ErrForbidden", err)
	}
	if err := a.athSvc.UnregisterParticipant(a.comp, pVerified, "", false); err != nil {
		t.Errorf("admin membatalkan peserta terverifikasi (belum di bagan): %v", err)
	}

	pOwn := a.participant(t, cat, own, "pending")
	if err := a.athSvc.UnregisterParticipant("00000000-0000-0000-0000-000000000000", pOwn, own, true); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("kejuaraan salah: %v, want ErrNotFound", err)
	}
	if err := a.athSvc.UnregisterParticipant(a.comp, pOwn, own, true); err != nil {
		t.Errorf("kontingen membatalkan pesertanya sendiri yang belum diverifikasi: %v", err)
	}
	if err := a.athSvc.UnregisterParticipant(a.comp, pOwn, "", false); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("batal dua kali: %v, want ErrNotFound", err)
	}

	// peserta yang sudah masuk bagan tidak boleh dihapus (ganti status lalu buat ulang bagan)
	pBracket := a.participant(t, cat, other, "verified")
	if _, err := a.db.Exec(`INSERT INTO bracket_slots (competition_id, weight_class, gender, round, slot_index, participant_id, category_id)
		VALUES ($1,'B','male',1,0,$2,$3)`, a.comp, pBracket, cat.ID); err != nil {
		t.Fatal(err)
	}
	if err := a.athSvc.UnregisterParticipant(a.comp, pBracket, "", false); !errors.Is(err, domain.ErrConflict) {
		t.Errorf("peserta di bagan: %v, want ErrConflict", err)
	}
}

// --- Partai manual: validasi peserta ---------------------------------------------------------------------------

func TestMatchService_CreateMatch_ValidasiPeserta(t *testing.T) {
	svc := service.NewMatchService(newMockMatchRepo(), &mockCompRepo{}, hub.NewManager(), nil, nil, nil)
	parts := map[string]*domain.Participant{
		"merah":   {ID: "merah", CompetitionID: "c1", RegistrationStatus: "verified", Name: "Merah"},
		"biru":    {ID: "biru", CompetitionID: "c1", RegistrationStatus: "verified", Name: "Biru"},
		"pending": {ID: "pending", CompetitionID: "c1", RegistrationStatus: "pending", Name: "Belum"},
		"asing":   {ID: "asing", CompetitionID: "c2", RegistrationStatus: "verified", Name: "Asing"},
	}
	svc.SetParticipantLookup(func(id string) (*domain.Participant, error) {
		if p, ok := parts[id]; ok {
			return p, nil
		}
		return nil, sql.ErrNoRows
	})
	mk := func(red, blue string) *store.Match {
		return &store.Match{CompetitionID: "c1", AthleteRedID: red, AthleteBlueID: blue}
	}

	if err := svc.CreateMatch(mk("merah", "biru")); err != nil {
		t.Errorf("partai sah ditolak: %v", err)
	}
	for name, m := range map[string]*store.Match{
		"peserta sama":             mk("merah", "merah"),
		"peserta tak ada":          mk("merah", "hantu"),
		"peserta kejuaraan lain":   mk("merah", "asing"),
		"peserta belum verifikasi": mk("pending", "biru"),
	} {
		if err := svc.CreateMatch(m); !errors.Is(err, domain.ErrInvalidInput) {
			t.Errorf("%s: err = %v, want ErrInvalidInput", name, err)
		}
	}
}
