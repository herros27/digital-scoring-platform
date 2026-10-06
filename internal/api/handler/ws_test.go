package handler

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"

	"github.com/herros27/digital-scoring/internal/auth"
	"github.com/herros27/digital-scoring/internal/hub"
)

func newWSServer(t *testing.T) (*httptest.Server, *hub.Manager) {
	t.Helper()
	mgr := hub.NewManager()
	h := &MatchHandler{HubManager: mgr}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /ws/arena/{arena_id}", h.HandleWS)
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv, mgr
}

func dial(srv *httptest.Server, arenaID, token string) (*websocket.Conn, *http.Response, error) {
	url := "ws" + strings.TrimPrefix(srv.URL, "http") + "/ws/arena/" + arenaID
	if token != "" {
		url += "?token=" + token
	}
	return websocket.DefaultDialer.Dial(url, nil)
}

func mustToken(t *testing.T, role, arenaID string) string {
	t.Helper()
	tok, err := auth.GenerateToken("pin-1", "comp-1", arenaID, role, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	return tok
}

func TestWS_SpectatorWithoutHubGets404(t *testing.T) {
	srv, mgr := newWSServer(t)

	_, resp, err := dial(srv, "arena-x", "")
	if err == nil || resp == nil || resp.StatusCode != http.StatusNotFound {
		t.Fatalf("want 404, got err=%v resp=%v", err, resp)
	}
	if _, getErr := mgr.Get("arena-x"); getErr == nil {
		t.Fatal("penonton publik tidak boleh membuat hub baru")
	}
}

func TestWS_SpectatorConnectsToExistingHub(t *testing.T) {
	srv, mgr := newWSServer(t)
	mgr.GetOrCreate("arena-a")

	conn, _, err := dial(srv, "arena-a", "")
	if err != nil {
		t.Fatalf("penonton publik harus bisa terhubung: %v", err)
	}
	conn.Close()
}

func TestWS_TokenArenaMismatchForbiddenForWasit(t *testing.T) {
	srv, _ := newWSServer(t)

	_, resp, err := dial(srv, "arena-a", mustToken(t, "wasit", "arena-b"))
	if err == nil || resp == nil || resp.StatusCode != http.StatusForbidden {
		t.Fatalf("want 403, got err=%v resp=%v", err, resp)
	}
}

func TestWS_UnscopedEmailTokenOnlyForOfficials(t *testing.T) {
	srv, _ := newWSServer(t)

	for _, role := range []string{"admin", "dewan", "wasit"} {
		conn, _, err := dial(srv, "arena-a", mustToken(t, role, ""))
		if err != nil {
			t.Fatalf("%s harus bisa terhubung tanpa arena: %v", role, err)
		}
		conn.Close()
	}

	for _, role := range []string{"judge", "juri", "contingent"} {
		_, resp, err := dial(srv, "arena-a", mustToken(t, role, ""))
		if err == nil || resp == nil || resp.StatusCode != http.StatusForbidden {
			t.Fatalf("%s tanpa arena harus ditolak 403, got err=%v resp=%v", role, err, resp)
		}
	}
}

func TestWS_WasitScopedToArenaConnects(t *testing.T) {
	srv, _ := newWSServer(t)

	conn, _, err := dial(srv, "arena-a", mustToken(t, "wasit", "arena-a"))
	if err != nil {
		t.Fatalf("wasit dengan token arena yang benar harus terhubung: %v", err)
	}
	conn.Close()
}

func TestWS_InvalidTokenUnauthorized(t *testing.T) {
	srv, _ := newWSServer(t)

	_, resp, err := dial(srv, "arena-a", "bukan-jwt")
	if err == nil || resp == nil || resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("want 401, got err=%v resp=%v", err, resp)
	}
}
