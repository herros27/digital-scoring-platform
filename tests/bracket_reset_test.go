package tests

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/herros27/digital-scoring/internal/api/handler"
	"github.com/herros27/digital-scoring/internal/service"
	"github.com/herros27/digital-scoring/internal/store"
)

type bracketRig struct {
	*adminDB
	h *handler.BracketHandler
}

func newBracketRig(t *testing.T) *bracketRig {
	t.Helper()
	a := newAdminDB(t)
	arenas := &store.ArenaStore{DB: a.db}
	matches := &store.MatchStore{DB: a.db}
	bsvc := service.NewBracketService(&store.BracketStore{DB: a.db}).WithMatchStore(matches).WithArenaStore(arenas).WithCategoryArenas(a.categories)
	if err := arenas.Create(&store.Arena{CompetitionID: a.comp, Name: "Gelanggang Uji"}); err != nil {
		t.Fatal(err)
	}
	return &bracketRig{a, &handler.BracketHandler{Service: bsvc, AthleteService: a.athSvc, CategoryService: a.catSvc}}
}

func (r *bracketRig) slots(t *testing.T, catID string) (n int) {
	t.Helper()
	if err := r.db.QueryRow(`SELECT COUNT(DISTINCT participant_id) FROM bracket_slots WHERE category_id = $1`, catID).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return
}

func (r *bracketRig) reset(catID string) int {
	req := httptest.NewRequest(http.MethodDelete, "/x", nil)
	req.SetPathValue("comp_id", r.comp)
	req.SetPathValue("id", catID)
	w := httptest.NewRecorder()
	r.h.Reset(w, req)
	return w.Code
}

// Reset bagan menghapus slot dan partai idle kategori tetapi peserta tetap terdaftar; ditolak bila draw terkunci
// atau sudah ada partai yang dimulai.
func TestBracketReset(t *testing.T) {
	r := newBracketRig(t)
	cat := r.category(t, "Kat Reset Bagan")
	var ids []string
	for i := 0; i < 4; i++ {
		ids = append(ids, r.participant(t, cat, "", "verified"))
	}
	gen := func() {
		t.Helper()
		if _, err := r.h.Service.GenerateCategoryBracket(r.comp, cat.ID, cat.WeightClass, cat.Gender, "", ids); err != nil {
			t.Fatal(err)
		}
	}
	gen()
	if r.slots(t, cat.ID) != 4 {
		t.Fatalf("bagan awal harus berisi 4 peserta, dapat %d", r.slots(t, cat.ID))
	}

	if _, err := r.db.Exec(`UPDATE categories SET bracket_locked_at = NOW() WHERE id = $1`, cat.ID); err != nil {
		t.Fatal(err)
	}
	if code := r.reset(cat.ID); code != http.StatusConflict {
		t.Errorf("reset bagan terkunci = %d, want 409", code)
	}
	_, _ = r.db.Exec(`UPDATE categories SET bracket_locked_at = NULL WHERE id = $1`, cat.ID)

	_, _ = r.db.Exec(`UPDATE matches SET status = 'running' WHERE id = (
		SELECT m.id FROM matches m JOIN bracket_slots b ON b.id = m.bracket_slot_id WHERE b.category_id = $1 LIMIT 1)`, cat.ID)
	if code := r.reset(cat.ID); code != http.StatusConflict {
		t.Errorf("reset bagan yang sudah berjalan = %d, want 409", code)
	}
	_, _ = r.db.Exec(`UPDATE matches SET status = 'idle' WHERE competition_id = $1`, r.comp)

	if code := r.reset(cat.ID); code != http.StatusOK {
		t.Fatalf("reset bagan = %d, want 200", code)
	}
	var slots, matches, parts int
	_ = r.db.QueryRow(`SELECT COUNT(*) FROM bracket_slots WHERE category_id = $1`, cat.ID).Scan(&slots)
	_ = r.db.QueryRow(`SELECT COUNT(*) FROM matches WHERE competition_id = $1`, r.comp).Scan(&matches)
	_ = r.db.QueryRow(`SELECT COUNT(*) FROM participants WHERE competition_id = $1`, r.comp).Scan(&parts)
	if slots != 0 || matches != 0 || parts != 4 {
		t.Errorf("setelah reset: slot=%d partai=%d peserta=%d, want 0/0/4", slots, matches, parts)
	}
	gen() // setelah reset, bagan dapat dibuat ulang
}

// Peserta yang lolos verifikasi setelah bagan ada masuk otomatis (bagan dibentuk ulang); bila draw terkunci atau
// kategori belum punya bagan, bagan tidak disentuh.
func TestBracketAutoJoin(t *testing.T) {
	r := newBracketRig(t)
	cat := r.category(t, "Kat Auto Bagan")
	ids := []string{r.participant(t, cat, "", "verified"), r.participant(t, cat, "", "verified")}

	if note := r.h.AutoJoin(r.comp, cat.ID); note != "" || r.slots(t, cat.ID) != 0 {
		t.Fatalf("kategori tanpa bagan tidak boleh dibuat otomatis: note=%q slot=%d", note, r.slots(t, cat.ID))
	}
	if _, err := r.h.Service.GenerateCategoryBracket(r.comp, cat.ID, cat.WeightClass, cat.Gender, "", ids); err != nil {
		t.Fatal(err)
	}

	r.participant(t, cat, "", "verified")
	if note := r.h.AutoJoin(r.comp, cat.ID); note == "" {
		t.Error("bagan yang dibentuk ulang harus melaporkan catatan")
	}
	if n := r.slots(t, cat.ID); n != 3 {
		t.Fatalf("peserta baru harus ikut masuk bagan: %d slot terisi, want 3", n)
	}

	_, _ = r.db.Exec(`UPDATE categories SET bracket_locked_at = NOW() WHERE id = $1`, cat.ID)
	r.participant(t, cat, "", "verified")
	if note := r.h.AutoJoin(r.comp, cat.ID); note == "" || r.slots(t, cat.ID) != 3 {
		t.Errorf("draw terkunci tidak boleh diubah: note=%q slot=%d", note, r.slots(t, cat.ID))
	}
}
