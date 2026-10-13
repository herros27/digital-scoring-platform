package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/herros27/digital-scoring/internal/hub"
	"github.com/herros27/digital-scoring/internal/service"
	"github.com/herros27/digital-scoring/internal/store"
)

type fakeArenaRepo struct{ arenas map[string]*store.Arena }

func (f *fakeArenaRepo) Create(a *store.Arena) error { f.arenas[a.ID] = a; return nil }
func (f *fakeArenaRepo) GetByID(id string) (*store.Arena, error) {
	if a, ok := f.arenas[id]; ok {
		return a, nil
	}
	return nil, nil
}
func (f *fakeArenaRepo) ListByCompetition(string) ([]store.Arena, error) { return nil, nil }
func (f *fakeArenaRepo) Update(*store.Arena) error                       { return nil }
func (f *fakeArenaRepo) Delete(string) error                             { return nil }

func presenceRequest(h *ArenaHandler, comp, arena string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, "/api/v1/competitions/"+comp+"/arenas/"+arena+"/presence", nil)
	req.SetPathValue("comp_id", comp)
	req.SetPathValue("id", arena)
	w := httptest.NewRecorder()
	h.Presence(w, req)
	return w
}

func decodePresence(t *testing.T, w *httptest.ResponseRecorder) arenaPresenceDTO {
	t.Helper()
	var env struct {
		Data arenaPresenceDTO `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &env); err != nil {
		t.Fatalf("respons bukan JSON: %v (%s)", err, w.Body.String())
	}
	return env.Data
}

func TestArenaPresence_ReportsOnlyConnectedDevices(t *testing.T) {
	repo := &fakeArenaRepo{arenas: map[string]*store.Arena{
		"arena-1": {ID: "arena-1", CompetitionID: "comp-1"},
		"arena-2": {ID: "arena-2", CompetitionID: "comp-1"},
	}}
	hubs := hub.NewManager()
	h := &ArenaHandler{Service: service.NewArenaService(repo, nil), Hubs: hubs}

	// belum ada hub: semua nol (bukan galat)
	got := decodePresence(t, presenceRequest(h, "comp-1", "arena-1"))
	if len(got.JudgeSeats) != 0 || got.Referees != 0 || got.Scoreboards != 0 {
		t.Fatalf("tanpa hub harus kosong, got %+v", got)
	}

	hb := hubs.GetOrCreate("arena-1")
	for _, c := range []struct {
		id   string
		role hub.Role
		seat int
	}{{"j3", hub.RoleJudge, 3}, {"j1", hub.RoleJudge, 1}, {"ref", hub.RoleReferee, 0}, {"tv", hub.RoleScoreboard, 0}} {
		cl, _ := hub.NewClient(c.id, c.role, "arena-1")
		cl.SeatNumber = c.seat
		hb.Register(cl)
	}

	got = decodePresence(t, presenceRequest(h, "comp-1", "arena-1"))
	if len(got.JudgeSeats) != 2 || got.JudgeSeats[0] != 1 || got.JudgeSeats[1] != 3 {
		t.Fatalf("kursi online = %v, want [1 3] terurut", got.JudgeSeats)
	}
	if got.Referees != 1 || got.Scoreboards != 1 || got.Admins != 0 {
		t.Fatalf("hitungan perangkat salah: %+v", got)
	}

	// gelanggang lain di kejuaraan yang sama tidak ikut terhitung
	other := decodePresence(t, presenceRequest(h, "comp-1", "arena-2"))
	if len(other.JudgeSeats) != 0 || other.Referees != 0 {
		t.Fatalf("gelanggang lain harus kosong, got %+v", other)
	}
}

func TestArenaPresence_ArenaOfAnotherCompetitionIsNotFound(t *testing.T) {
	repo := &fakeArenaRepo{arenas: map[string]*store.Arena{"arena-1": {ID: "arena-1", CompetitionID: "comp-1"}}}
	h := &ArenaHandler{Service: service.NewArenaService(repo, nil), Hubs: hub.NewManager()}
	if w := presenceRequest(h, "comp-OTHER", "arena-1"); w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404 untuk gelanggang milik kejuaraan lain", w.Code)
	}
	if w := presenceRequest(h, "comp-1", "tidak-ada"); w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404 untuk gelanggang yang tidak ada", w.Code)
	}
}
