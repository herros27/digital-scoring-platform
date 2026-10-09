package hub

import "testing"

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
