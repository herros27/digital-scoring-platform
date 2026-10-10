package hub

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

func TestReleaseOnlyByOwner(t *testing.T) {
	h := NewHub("a1")
	first, second := new(int), new(int)
	h.Attach(first, func(string, *Client, Message) {}, func() {})
	h.Attach(second, func(string, *Client, Message) {}, func() {})

	h.Release(first) // bukan pemilik: tidak boleh melepas callback milik partai berikutnya
	if h.OnMessage == nil {
		t.Fatal("Release oleh non-pemilik melepas callback")
	}
	h.Release(second)
	if h.OnMessage != nil || h.onPresence != nil {
		t.Fatal("Release oleh pemilik harus melepas callback")
	}
}

// Dua koneksi dengan ID sama (tablet kembar / sambung ulang sebelum koneksi lama dibuang): balasan atas
// permintaan harus sampai ke koneksi yang meminta, bukan ke koneksi lain berID sama.
func TestSendToConn_ReachesOnlyRequestingConnection(t *testing.T) {
	h := NewHub("arena-1")
	oldConn, oldCh := NewClient("pin-1", RoleJudge, "arena-1")
	newConn, newCh := NewClient("pin-1", RoleJudge, "arena-1")
	h.Register(oldConn)
	h.Register(newConn)

	for i := 0; i < 50; i++ { // urutan iterasi map acak: ulangi agar tidak kebetulan lolos
		h.SendToConn(newConn, Message{Type: MsgSyncResponse})
	}
	if got := len(newCh); got != 50 {
		t.Fatalf("koneksi peminta menerima %d/50 balasan", got)
	}
	if got := len(oldCh); got != 0 {
		t.Fatalf("koneksi lain berID sama tidak boleh menerima balasan, got %d", got)
	}
}

func TestSendToConn_IgnoresReleasedConnection(t *testing.T) {
	h := NewHub("arena-1")
	gone, ch := NewClient("pin-1", RoleJudge, "arena-1")
	// tidak terdaftar (sudah dibuang): tidak boleh panik maupun mengirim
	h.SendToConn(gone, Message{Type: MsgError})
	if len(ch) != 0 {
		t.Fatal("koneksi yang sudah dilepas tidak boleh menerima pesan")
	}
}

func TestSendToClient_DeliversToEveryConnectionWithThatID(t *testing.T) {
	h := NewHub("arena-1")
	a, aCh := NewClient("pin-1", RoleJudge, "arena-1")
	b, bCh := NewClient("pin-1", RoleJudge, "arena-1")
	other, otherCh := NewClient("pin-2", RoleJudge, "arena-1")
	h.Register(a)
	h.Register(b)
	h.Register(other)

	h.SendToClient("pin-1", Message{Type: MsgScoreConfirmed})
	if len(aCh) != 1 || len(bCh) != 1 {
		t.Fatalf("kedua koneksi pin-1 harus menerima: a=%d b=%d", len(aCh), len(bCh))
	}
	if len(otherCh) != 0 {
		t.Fatal("ID lain tidak boleh menerima")
	}
}

// Klien yang tidak pernah membalas ping (jaringan putus tanpa menutup soket) harus dibuang server, sedangkan
// klien yang membaca (gorilla otomatis membalas pong) tetap terhubung.
func TestKeepalive_DropsSilentClientKeepsLiveOne(t *testing.T) {
	oldWait, oldPeriod := pongWait, pingPeriod
	pongWait, pingPeriod = 400*time.Millisecond, 100*time.Millisecond
	defer func() { pongWait, pingPeriod = oldWait, oldPeriod }()

	h := NewHub("arena-1")
	go h.Run()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h.HandleWSWithSeat(w, r, r.URL.Query().Get("id"), RoleJudge, 1)
	}))
	defer srv.Close()
	url := "ws" + strings.TrimPrefix(srv.URL, "http")

	live, _, err := websocket.DefaultDialer.Dial(url+"?id=live", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer live.Close()
	go func() { // membaca = memproses ping dan membalas pong otomatis
		for {
			if _, _, err := live.ReadMessage(); err != nil {
				return
			}
		}
	}()
	silent, _, err := websocket.DefaultDialer.Dial(url+"?id=silent", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer silent.Close() // tidak pernah membaca: ping tidak dibalas

	count := func() int {
		h.mu.RLock()
		defer h.mu.RUnlock()
		return len(h.clients)
	}
	deadline := time.Now().Add(3 * time.Second)
	for count() != 1 && time.Now().Before(deadline) {
		time.Sleep(50 * time.Millisecond)
	}
	if count() != 1 {
		t.Fatalf("klien terhubung = %d, want 1 (yang diam harus diputus)", count())
	}
	time.Sleep(600 * time.Millisecond) // lewat pongWait: klien aktif tetap bertahan
	if count() != 1 {
		t.Fatalf("klien aktif ikut terputus, terhubung = %d", count())
	}
	h.mu.RLock()
	for c := range h.clients {
		if c.ID != "live" {
			t.Errorf("klien yang bertahan = %q, want live", c.ID)
		}
	}
	h.mu.RUnlock()

	// tutup semua koneksi dan tunggu pump berhenti sebelum durasi bawaan dipulihkan (hindari data race)
	live.Close()
	silent.Close()
	for count() != 0 && time.Now().Before(deadline.Add(3*time.Second)) {
		time.Sleep(20 * time.Millisecond)
	}
}

func TestReadLimit_DropsOversizedMessage(t *testing.T) {
	h := NewHub("arena-1")
	go h.Run()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h.HandleWSWithSeat(w, r, "spectator", RoleScoreboard, 0)
	}))
	defer srv.Close()

	conn, _, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(srv.URL, "http"), nil)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	if err := conn.WriteMessage(websocket.TextMessage, []byte(strings.Repeat("a", maxMessageSize+1))); err != nil {
		t.Fatal(err)
	}
	_ = conn.SetReadDeadline(time.Now().Add(3 * time.Second))
	if _, _, err := conn.ReadMessage(); !websocket.IsCloseError(err, websocket.CloseMessageTooBig) {
		t.Fatalf("err = %v, want close 1009 (message too big)", err)
	}
}
