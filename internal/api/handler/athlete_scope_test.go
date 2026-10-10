package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/herros27/digital-scoring/internal/auth"
	"github.com/herros27/digital-scoring/internal/domain"
	"github.com/herros27/digital-scoring/internal/service"
)

// scopeRepo repo atlet in-memory untuk menguji pembatasan data per kontingen.
type scopeRepo struct {
	athletes map[string]*domain.Athlete
	parts    []domain.Participant
}

func (f *scopeRepo) Create(a *domain.Athlete) error {
	a.ID = "ath-" + a.Name
	f.athletes[a.ID] = a
	return nil
}
func (f *scopeRepo) GetByID(id string) (*domain.Athlete, error) {
	a, ok := f.athletes[id]
	if !ok {
		return nil, domain.ErrNotFound
	}
	return a, nil
}
func (f *scopeRepo) List() ([]domain.Athlete, error)               { return nil, nil }
func (f *scopeRepo) RegisterToCompetition(_, _, _, _ string) error { return nil }
func (f *scopeRepo) ListByCompetition(string) ([]domain.Athlete, error) {
	out := []domain.Athlete{}
	for _, id := range []string{"a-own", "a-other", "a-admin"} { // urutan tetap agar hasil deterministik
		if a, ok := f.athletes[id]; ok {
			out = append(out, *a)
		}
	}
	return out, nil
}
func (f *scopeRepo) RegisterParticipant(compID, athID, contID, wc, gender, status string) error {
	f.parts = append(f.parts, domain.Participant{ID: "p-" + athID, CompetitionID: compID, AthleteID: athID, ContingentID: contID, RegistrationStatus: status})
	return nil
}
func (f *scopeRepo) UpdateParticipantStatus(string, string) error { return nil }
func (f *scopeRepo) ListParticipants(_, status string) ([]domain.Participant, error) {
	var out []domain.Participant
	for _, p := range f.parts {
		if status == "" || p.RegistrationStatus == status {
			out = append(out, p)
		}
	}
	return out, nil
}

func newScopeHandler() (*AthleteHandler, *scopeRepo) {
	repo := &scopeRepo{athletes: map[string]*domain.Athlete{
		"a-own":   {ID: "a-own", Name: "Milik Saya", ContingentID: "k1"},
		"a-other": {ID: "a-other", Name: "Milik Lain", ContingentID: "k2"},
		"a-admin": {ID: "a-admin", Name: "Dari Admin"}, // tanpa kontingen
	}}
	repo.parts = []domain.Participant{
		{ID: "p1", ContingentID: "k1", RegistrationStatus: "pending"},
		{ID: "p2", ContingentID: "k1", RegistrationStatus: "verified"},
		{ID: "p3", ContingentID: "k2", RegistrationStatus: "verified"},
		{ID: "p4", ContingentID: "", RegistrationStatus: "registered"}, // didaftarkan admin
	}
	return &AthleteHandler{Service: service.NewAthleteService(repo)}, repo
}

func asRole(r *http.Request, role, id string) *http.Request {
	return r.WithContext(auth.ContextWithClaims(context.Background(), &auth.Claims{Role: role, JudgePinID: id}))
}

func participantIDs(t *testing.T, rec *httptest.ResponseRecorder) []string {
	t.Helper()
	var body struct{ Data []domain.Participant }
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	ids := []string{}
	for _, p := range body.Data {
		ids = append(ids, p.ID)
	}
	return ids
}

func TestListParticipantsScopedToContingent(t *testing.T) {
	h, _ := newScopeHandler()
	cases := []struct {
		name, role, id, query string
		want                  []string
	}{
		{"kontingen hanya melihat miliknya", "contingent", "k1", "", []string{"p1", "p2"}},
		{"kontingen lain tidak melihat milik k1", "contingent", "k2", "", []string{"p3"}},
		{"filter status tetap dibatasi", "contingent", "k1", "?status=verified", []string{"p2"}},
		{"token kontingen tanpa id tidak melihat apa pun", "contingent", "", "", []string{}},
		{"admin melihat semua", "admin", "", "", []string{"p1", "p2", "p3", "p4"}},
		{"dewan melihat semua", "dewan", "", "", []string{"p1", "p2", "p3", "p4"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/v1/competitions/c1/participants"+tc.query, nil)
			req.SetPathValue("comp_id", "c1")
			rec := httptest.NewRecorder()
			h.ListParticipants(rec, asRole(req, tc.role, tc.id))
			if rec.Code != http.StatusOK {
				t.Fatalf("status = %d", rec.Code)
			}
			got := participantIDs(t, rec)
			if len(got) != len(tc.want) {
				t.Fatalf("got %v, want %v", got, tc.want)
			}
			for i := range got {
				if got[i] != tc.want[i] {
					t.Fatalf("got %v, want %v", got, tc.want)
				}
			}
		})
	}
}

func TestGetAthleteHidesOtherContingents(t *testing.T) {
	h, _ := newScopeHandler()
	get := func(role, id, athleteID string) int {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/athletes/"+athleteID, nil)
		req.SetPathValue("id", athleteID)
		rec := httptest.NewRecorder()
		h.Get(rec, asRole(req, role, id))
		return rec.Code
	}
	if c := get("contingent", "k1", "a-own"); c != http.StatusOK {
		t.Fatalf("atlet sendiri = %d, want 200", c)
	}
	if c := get("contingent", "k1", "a-other"); c != http.StatusNotFound {
		t.Fatalf("atlet kontingen lain = %d, want 404", c)
	}
	if c := get("admin", "", "a-other"); c != http.StatusOK {
		t.Fatalf("admin = %d, want 200", c)
	}
}

func TestCreateAthleteForcesContingentOwnership(t *testing.T) {
	h, repo := newScopeHandler()
	body, _ := json.Marshal(map[string]any{"name": "Baru", "gender": "male", "team": "T", "contingent_id": "k2"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/athletes", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.Create(rec, asRole(req, "contingent", "k1"))
	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, body=%s", rec.Code, rec.Body.String())
	}
	if got := repo.athletes["ath-Baru"].ContingentID; got != "k1" {
		t.Fatalf("contingent_id = %q, want k1 (tidak boleh memakai nilai dari body)", got)
	}
}

func TestRegisterRejectsOtherContingentsAthlete(t *testing.T) {
	h, repo := newScopeHandler()
	before := len(repo.parts)
	body, _ := json.Marshal(map[string]any{"athlete_id": "a-other", "weight_class": "A"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/competitions/c1/athletes", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("comp_id", "c1")
	rec := httptest.NewRecorder()
	h.RegisterToCompetition(rec, asRole(req, "contingent", "k1"))
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
	if len(repo.parts) != before {
		t.Fatal("peserta tidak boleh bertambah")
	}
}

func TestListAthletesByCompetitionScopedToContingent(t *testing.T) {
	h, _ := newScopeHandler()
	cases := []struct {
		name, role, id string
		want           []string
	}{
		{"kontingen hanya melihat atletnya", "contingent", "k1", []string{"a-own"}},
		{"kontingen lain hanya melihat atletnya", "contingent", "k2", []string{"a-other"}},
		{"token kontingen tanpa id tidak melihat apa pun", "contingent", "", []string{}},
		{"admin melihat semua", "admin", "", []string{"a-own", "a-other", "a-admin"}},
		{"operator melihat semua", "operator", "", []string{"a-own", "a-other", "a-admin"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/v1/competitions/c1/athletes", nil)
			req.SetPathValue("comp_id", "c1")
			rec := httptest.NewRecorder()
			h.ListByCompetition(rec, asRole(req, tc.role, tc.id))
			if rec.Code != http.StatusOK {
				t.Fatalf("status = %d", rec.Code)
			}
			var body struct{ Data []domain.Athlete }
			if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
				t.Fatal(err)
			}
			got := []string{}
			for _, a := range body.Data {
				got = append(got, a.ID)
			}
			if len(got) != len(tc.want) {
				t.Fatalf("got %v, want %v", got, tc.want)
			}
			for i := range got {
				if got[i] != tc.want[i] {
					t.Fatalf("got %v, want %v", got, tc.want)
				}
			}
		})
	}
}

func (f *scopeRepo) UpdateProfile(id string, birthDate, nik *string) error {
	a, ok := f.athletes[id]
	if !ok {
		return domain.ErrNotFound
	}
	if birthDate != nil {
		a.BirthDate = *birthDate
	}
	if nik != nil {
		a.NIK = *nik
	}
	return nil
}

func TestUpdateProfileOwnershipAndValidation(t *testing.T) {
	h, repo := newScopeHandler()
	put := func(role, id, athleteID, body string) int {
		req := httptest.NewRequest(http.MethodPut, "/api/v1/athletes/"+athleteID, bytes.NewReader([]byte(body)))
		req.Header.Set("Content-Type", "application/json")
		req.SetPathValue("id", athleteID)
		rec := httptest.NewRecorder()
		h.UpdateProfile(rec, asRole(req, role, id))
		return rec.Code
	}
	if c := put("contingent", "k1", "a-own", `{"birth_date":"2000-02-03"}`); c != http.StatusOK {
		t.Fatalf("atlet sendiri = %d, want 200", c)
	}
	if got := repo.athletes["a-own"].BirthDate; got != "2000-02-03" {
		t.Fatalf("birth_date tersimpan = %q", got)
	}
	if c := put("contingent", "k1", "a-other", `{"birth_date":"2000-02-03"}`); c != http.StatusNotFound {
		t.Fatalf("atlet kontingen lain = %d, want 404", c)
	}
	if repo.athletes["a-other"].BirthDate != "" {
		t.Fatal("atlet kontingen lain tidak boleh berubah")
	}
	if c := put("admin", "", "a-other", `{"birth_date":"2000-02-03","nik":"3201"}`); c != http.StatusOK {
		t.Fatalf("admin = %d, want 200", c)
	}
	if c := put("admin", "", "a-own", `{}`); c != http.StatusBadRequest {
		t.Fatalf("body kosong = %d, want 400", c)
	}
	if c := put("admin", "", "a-own", `{"birth_date":"31-12-2000"}`); c != http.StatusBadRequest {
		t.Fatalf("format tanggal salah = %d, want 400", c)
	}
	if c := put("admin", "", "a-own", `{"birth_date":"2099-01-01"}`); c != http.StatusBadRequest {
		t.Fatalf("tanggal masa depan = %d, want 400", c)
	}
	if c := put("admin", "", "tidak-ada", `{"birth_date":"2000-02-03"}`); c != http.StatusNotFound {
		t.Fatalf("atlet tidak ada = %d, want 404", c)
	}
}

func TestListParticipantsFlagsAgeCheck(t *testing.T) {
	h, repo := newScopeHandler()
	repo.parts = []domain.Participant{
		{ID: "ok", ContingentID: "k1", AgeGroup: domain.AgeDewasa, PresetStandard: domain.PresetIPSI, BirthDate: "2000-01-01"},
		{ID: "kosong", ContingentID: "k1", AgeGroup: domain.AgeDewasa, PresetStandard: domain.PresetIPSI},
		{ID: "salah", ContingentID: "k1", AgeGroup: domain.AgeDewasa, PresetStandard: domain.PresetIPSI, BirthDate: "2015-01-01"},
		{ID: "tanpa-kategori", ContingentID: "k1"},
	}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/competitions/c1/participants", nil)
	req.SetPathValue("comp_id", "c1")
	rec := httptest.NewRecorder()
	h.ListParticipants(rec, asRole(req, "admin", ""))

	var body struct{ Data []domain.Participant }
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	got := map[string]string{}
	for _, p := range body.Data {
		got[p.ID] = p.AgeCheck
	}
	want := map[string]string{"ok": "", "kosong": domain.AgeCheckMissing, "salah": domain.AgeCheckMismatch, "tanpa-kategori": ""}
	for id, w := range want {
		if got[id] != w {
			t.Fatalf("age_check[%s] = %q, want %q", id, got[id], w)
		}
	}
	// field internal tidak ikut keluar
	if bytes.Contains(rec.Body.Bytes(), []byte("preset_standard")) {
		t.Fatal("preset_standard tidak boleh muncul di respons")
	}
}

// Juri/wasit (token PIN) tidak boleh membaca tanggal lahir & NIK atlet; panitia dan kontingen pemilik boleh.
func TestPersonalDataHiddenFromPinRoles(t *testing.T) {
	h, repo := newScopeHandler()
	repo.athletes["a-own"].NIK, repo.athletes["a-own"].BirthDate = "3171000000000001", "2015-05-01"
	repo.parts[0].NIK, repo.parts[0].BirthDate = "3171000000000001", "2015-05-01"

	cases := []struct {
		role, id string
		visible  bool
	}{
		{"judge", "pin-1", false},
		{"wasit", "pin-2", false},
		{"admin", "", true},
		{"dewan", "", true},
		{"contingent", "k1", true},
	}
	for _, tc := range cases {
		t.Run(tc.role, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/v1/athletes/a-own", nil)
			req.SetPathValue("id", "a-own")
			rec := httptest.NewRecorder()
			h.Get(rec, asRole(req, tc.role, tc.id))
			var one struct{ Data domain.Athlete }
			if err := json.Unmarshal(rec.Body.Bytes(), &one); err != nil {
				t.Fatal(err)
			}
			if got := one.Data.NIK != "" || one.Data.BirthDate != ""; got != tc.visible {
				t.Errorf("athlete: data pribadi terlihat = %v, want %v (%s)", got, tc.visible, rec.Body)
			}

			req = httptest.NewRequest(http.MethodGet, "/api/v1/competitions/c1/participants", nil)
			req.SetPathValue("comp_id", "c1")
			rec = httptest.NewRecorder()
			h.ListParticipants(rec, asRole(req, tc.role, tc.id))
			var list struct{ Data []domain.Participant }
			if err := json.Unmarshal(rec.Body.Bytes(), &list); err != nil {
				t.Fatal(err)
			}
			for _, p := range list.Data {
				if p.ID != "p1" {
					continue
				}
				if got := p.NIK != "" || p.BirthDate != ""; got != tc.visible {
					t.Errorf("participant: data pribadi terlihat = %v, want %v", got, tc.visible)
				}
			}
		})
	}
	if repo.athletes["a-own"].NIK == "" {
		t.Error("data di repo ikut terhapus; penyamaran harus bekerja pada salinan")
	}
}
