package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sort"
	"strings"
	"testing"

	"github.com/herros27/digital-scoring/internal/domain"
	"github.com/herros27/digital-scoring/internal/store"
)

type fakePubComps struct {
	comps map[string]*domain.Competition
}

func (f fakePubComps) GetCompetitionsPaged(status string, _, _ int) ([]domain.Competition, int, error) {
	var out []domain.Competition
	ids := []string{}
	for id := range f.comps {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		if c := f.comps[id]; status == "" || c.Status == status {
			out = append(out, *c)
		}
	}
	return out, len(out), nil
}
func (f fakePubComps) GetCompetitionByID(id string) (*domain.Competition, error) {
	if c, ok := f.comps[id]; ok {
		return c, nil
	}
	return nil, domain.ErrCompetitionNotFound
}

type fakePubCats struct{ cats []domain.Category }

func (f fakePubCats) ListCategories(compID string) ([]domain.Category, error) {
	var out []domain.Category
	for _, c := range f.cats {
		if c.CompetitionID == compID {
			out = append(out, c)
		}
	}
	return out, nil
}
func (f fakePubCats) GetCategoryByID(string) (*domain.Category, error) { return nil, nil }

type fakePubBrackets struct {
	slots map[string][]store.BracketSlot // per category_id
	sums  []store.CategoryBracketSummary
}

func (f fakePubBrackets) GetBracketTreeByCategory(_, categoryID string) ([]store.BracketSlot, error) {
	return f.slots[categoryID], nil
}
func (f fakePubBrackets) ListCategoryBracketSummaries(string) ([]store.CategoryBracketSummary, error) {
	return f.sums, nil
}

type fakePubParts struct{ parts []domain.Participant }

func (f fakePubParts) ListParticipants(string, string) ([]domain.Participant, error) {
	return f.parts, nil
}

func sp(s string) *string { return &s }

// newPublicBracketHandler menyiapkan: kejuaraan aktif c-act (bagan Remaja + kategori tanpa bagan),
// kejuaraan draft c-draft, dan peserta dengan data sensitif lengkap.
func newPublicBracketHandler() *PublicBracketHandler {
	w := 41.5
	return &PublicBracketHandler{
		Competitions: fakePubComps{comps: map[string]*domain.Competition{
			"c-act":   {ID: "c-act", Name: "Kejuaraan Aktif", Location: "GOR", StartDate: "2026-06-01", EndDate: "2026-06-03", Status: "active"},
			"c-draft": {ID: "c-draft", Name: "Masih Draft", Status: "draft"},
			"c-done":  {ID: "c-done", Name: "Selesai", Status: "finished"},
		}},
		Categories: fakePubCats{cats: []domain.Category{
			{ID: "cat-remaja", CompetitionID: "c-act", Name: "Remaja Putra A", Gender: "male", AgeGroup: "remaja", WeightClass: "A"},
			{ID: "cat-kosong", CompetitionID: "c-act", Name: "Dewasa Putri B", Gender: "female", AgeGroup: "dewasa", WeightClass: "B"},
			{ID: "cat-draft", CompetitionID: "c-draft", Name: "Draft Kat", Gender: "male", AgeGroup: "dewasa", WeightClass: "A"},
		}},
		Brackets: fakePubBrackets{
			sums: []store.CategoryBracketSummary{{CategoryID: "cat-remaja", Participants: 2, Rounds: 1}, {CategoryID: "cat-draft", Participants: 2, Rounds: 1}},
			slots: map[string][]store.BracketSlot{
				"cat-remaja": {
					{ID: "s1", Round: 1, SlotIndex: 0, ParticipantID: sp("p1"), WinnerID: sp("p1")},
					{ID: "s2", Round: 1, SlotIndex: 1, ParticipantID: sp("p2"), WinnerID: sp("p1")},
					{ID: "s3", Round: 1, SlotIndex: 2, IsBye: true},
				},
			},
		},
		Participants: fakePubParts{parts: []domain.Participant{
			{ID: "p1", AthleteID: "ath-1", ContingentID: "k-1", Name: "Budi Santoso", Club: "Tapak Suci", NIK: "3201010101010001", BirthDate: "2010-03-04", ActualWeight: &w, DocsVerified: map[string]bool{"health": true}, RegistrationStatus: "verified"},
			{ID: "p2", AthleteID: "ath-2", ContingentID: "k-2", Name: "Candra", Club: "PSHT", NIK: "3201010101010002", BirthDate: "2009-01-02", RegistrationStatus: "verified"},
		}},
	}
}

func publicGet(t *testing.T, fn http.HandlerFunc, path string, pv map[string]string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, path, nil)
	for k, v := range pv {
		req.SetPathValue(k, v)
	}
	rec := httptest.NewRecorder()
	fn(rec, req)
	return rec
}

// keysOf mengumpulkan semua nama field JSON (rekursif) dari sebuah dokumen.
func keysOf(v any, into map[string]bool) {
	switch x := v.(type) {
	case map[string]any:
		for k, val := range x {
			into[k] = true
			keysOf(val, into)
		}
	case []any:
		for _, val := range x {
			keysOf(val, into)
		}
	}
}

func TestPublicBracketOnlyListsActiveCompetitions(t *testing.T) {
	h := newPublicBracketHandler()
	rec := publicGet(t, h.ListCompetitions, "/api/v1/public/competitions", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	var body struct{ Data []publicCompetitionDTO }
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if len(body.Data) != 1 || body.Data[0].ID != "c-act" {
		t.Fatalf("hanya kejuaraan aktif yang boleh tampil, got %+v", body.Data)
	}
}

func TestPublicBracketHiddenUnlessCompetitionActive(t *testing.T) {
	h := newPublicBracketHandler()
	for _, id := range []string{"c-draft", "c-done", "tidak-ada"} {
		if rec := publicGet(t, h.ListBrackets, "/x", map[string]string{"comp_id": id}); rec.Code != http.StatusNotFound {
			t.Fatalf("daftar bagan %s = %d, want 404", id, rec.Code)
		}
		rec := publicGet(t, h.GetBracket, "/x", map[string]string{"comp_id": id, "category_id": "cat-remaja"})
		if rec.Code != http.StatusNotFound {
			t.Fatalf("detail bagan %s = %d, want 404", id, rec.Code)
		}
	}
}

func TestPublicBracketListSkipsCategoriesWithoutBracket(t *testing.T) {
	h := newPublicBracketHandler()
	rec := publicGet(t, h.ListBrackets, "/x", map[string]string{"comp_id": "c-act"})
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	var body struct{ Data []publicBracketListItem }
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if len(body.Data) != 1 || body.Data[0].CategoryID != "cat-remaja" || body.Data[0].Participants != 2 {
		t.Fatalf("got %+v, want hanya cat-remaja dengan 2 peserta", body.Data)
	}
}

func TestPublicBracketDetailShowsNamesAndWinner(t *testing.T) {
	h := newPublicBracketHandler()
	rec := publicGet(t, h.GetBracket, "/x", map[string]string{"comp_id": "c-act", "category_id": "cat-remaja"})
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body=%s", rec.Code, rec.Body.String())
	}
	var body struct{ Data publicBracketDetail }
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	s := body.Data.Slots
	if len(s) != 3 {
		t.Fatalf("slots = %d, want 3", len(s))
	}
	if s[0].Participant == nil || s[0].Participant.Name != "Budi Santoso" || s[0].Participant.Team != "Tapak Suci" {
		t.Fatalf("slot 1 = %+v", s[0].Participant)
	}
	if !s[0].IsWinner || s[1].IsWinner {
		t.Fatalf("is_winner salah: %v %v", s[0].IsWinner, s[1].IsWinner)
	}
	if !s[2].IsBye || s[2].Participant != nil {
		t.Fatalf("bye harus tanpa peserta: %+v", s[2])
	}
}

func TestPublicBracketDetailRejectsOtherCompetitionOrMissingBracket(t *testing.T) {
	h := newPublicBracketHandler()
	for _, cat := range []string{"cat-draft", "cat-kosong", "tidak-ada"} {
		rec := publicGet(t, h.GetBracket, "/x", map[string]string{"comp_id": "c-act", "category_id": cat})
		if rec.Code != http.StatusNotFound {
			t.Fatalf("kategori %s = %d, want 404", cat, rec.Code)
		}
	}
}

// Whitelist field + pencarian nilai sensitif: jika ada yang menambah field ke DTO publik, tes ini gagal.
func TestPublicBracketResponsesLeakNoPersonalData(t *testing.T) {
	h := newPublicBracketHandler()
	allowed := map[string]bool{
		// kejuaraan
		"success": true, "data": true, "id": true, "name": true, "location": true, "start_date": true, "end_date": true,
		// daftar/detail bagan
		"competition": true, "category": true, "category_id": true, "gender": true, "age_group": true,
		"weight_class": true, "participants": true, "rounds": true, "completed": true, "slots": true,
		// slot
		"round": true, "slot_index": true, "is_bye": true, "next_slot_id": true, "is_winner": true,
		"participant": true, "team": true,
	}
	secrets := []string{"3201010101010001", "3201010101010002", "2010-03-04", "2009-01-02", "41.5", "ath-1", "ath-2", "k-1", "k-2", "p1", "p2", "docs_verified", "nik", "birth_date"}

	responses := map[string]*httptest.ResponseRecorder{
		"competitions": publicGet(t, h.ListCompetitions, "/x", nil),
		"list":         publicGet(t, h.ListBrackets, "/x", map[string]string{"comp_id": "c-act"}),
		"detail":       publicGet(t, h.GetBracket, "/x", map[string]string{"comp_id": "c-act", "category_id": "cat-remaja"}),
	}
	for name, rec := range responses {
		var doc any
		if err := json.Unmarshal(rec.Body.Bytes(), &doc); err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		found := map[string]bool{}
		keysOf(doc, found)
		for k := range found {
			if !allowed[k] {
				t.Errorf("%s: field %q tidak ada di whitelist publik", name, k)
			}
		}
		raw := rec.Body.String()
		for _, secret := range secrets {
			if strings.Contains(raw, `"`+secret+`"`) || strings.Contains(raw, secret+",") || strings.Contains(raw, secret+"}") {
				t.Errorf("%s: respons memuat data sensitif %q", name, secret)
			}
		}
	}
}
