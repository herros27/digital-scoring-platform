package handler

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/herros27/digital-scoring/internal/domain"
	"github.com/herros27/digital-scoring/internal/service"
)

// fakeLockRepo repo kategori dalam memori yang juga mendukung kunci draw.
type fakeLockRepo struct{ cats map[string]*domain.Category }

func (f *fakeLockRepo) Create(c *domain.Category) error { f.cats[c.ID] = c; return nil }
func (f *fakeLockRepo) GetByID(id string) (*domain.Category, error) {
	if c, ok := f.cats[id]; ok {
		return c, nil
	}
	return nil, nil
}
func (f *fakeLockRepo) ListByCompetition(string) ([]domain.Category, error) { return nil, nil }
func (f *fakeLockRepo) SetBracketLock(comp, id string, locked bool) (*domain.Category, error) {
	c, ok := f.cats[id]
	if !ok || c.CompetitionID != comp {
		return nil, domain.ErrNotFound
	}
	c.BracketLocked = locked
	c.BracketLockedAt = nil
	if locked {
		now := time.Now()
		c.BracketLockedAt = &now
	}
	return c, nil
}

func lockFixture() (*fakeLockRepo, *service.CategoryService) {
	repo := &fakeLockRepo{cats: map[string]*domain.Category{
		"cat-1": {ID: "cat-1", CompetitionID: "comp-1", Name: "Dewasa Putra A", Gender: "male", WeightClass: "A"},
	}}
	return repo, service.NewCategoryService(repo)
}

func putLock(h *CategoryHandler, comp, id, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPut, "/api/v1/competitions/"+comp+"/categories/"+id+"/bracket-lock", strings.NewReader(body))
	req.SetPathValue("comp_id", comp)
	req.SetPathValue("id", id)
	w := httptest.NewRecorder()
	h.SetBracketLock(w, req)
	return w
}

func TestSetBracketLock_LocksAndUnlocks(t *testing.T) {
	repo, svc := lockFixture()
	h := &CategoryHandler{Service: svc}

	if w := putLock(h, "comp-1", "cat-1", `{"locked":true}`); w.Code != http.StatusOK || !strings.Contains(w.Body.String(), `"bracket_locked":true`) {
		t.Fatalf("kunci: %d %s", w.Code, w.Body)
	}
	if !repo.cats["cat-1"].BracketLocked {
		t.Fatal("status kunci harus tersimpan")
	}
	// idempoten
	if w := putLock(h, "comp-1", "cat-1", `{"locked":true}`); w.Code != http.StatusOK {
		t.Fatalf("kunci ulang harus tetap 200, got %d", w.Code)
	}
	if w := putLock(h, "comp-1", "cat-1", `{"locked":false}`); w.Code != http.StatusOK || !strings.Contains(w.Body.String(), `"bracket_locked":false`) {
		t.Fatalf("buka kunci: %d %s", w.Code, w.Body)
	}
	if repo.cats["cat-1"].BracketLocked {
		t.Fatal("kunci harus terbuka")
	}
}

func TestSetBracketLock_RejectsBadInputAndForeignCategory(t *testing.T) {
	_, svc := lockFixture()
	h := &CategoryHandler{Service: svc}
	if w := putLock(h, "comp-1", "cat-1", `{}`); w.Code != http.StatusBadRequest {
		t.Fatalf("tanpa 'locked' -> %d, want 400", w.Code)
	}
	if w := putLock(h, "comp-LAIN", "cat-1", `{"locked":true}`); w.Code != http.StatusNotFound {
		t.Fatalf("kategori milik kejuaraan lain -> %d, want 404", w.Code)
	}
	if w := putLock(h, "comp-1", "tidak-ada", `{"locked":true}`); w.Code != http.StatusNotFound {
		t.Fatalf("kategori tidak ada -> %d, want 404", w.Code)
	}
}

// Penegakan di server: bagan kategori yang terkunci ditolak SEBELUM menyentuh pembentukan bagan.
func TestGenerateBracket_RefusedWhileCategoryIsLocked(t *testing.T) {
	repo, svc := lockFixture()
	repo.cats["cat-1"].BracketLocked = true
	h := &BracketHandler{CategoryService: svc} // Service nil: tidak boleh sampai ke sana

	req := httptest.NewRequest(http.MethodPost, "/api/v1/competitions/comp-1/brackets/generate", strings.NewReader(`{"category_id":"cat-1"}`))
	req.SetPathValue("comp_id", "comp-1")
	w := httptest.NewRecorder()
	h.Generate(w, req)

	if w.Code != http.StatusConflict || !strings.Contains(w.Body.String(), "BRACKET_LOCKED") {
		t.Fatalf("status = %d body = %s, want 409 BRACKET_LOCKED", w.Code, w.Body)
	}
}
