package hub

import (
	"testing"
	"time"
)

// Klien yang antreannya penuh diputus oleh Broadcast; kehadiran harus diperbarui supaya panel wasit tidak tetap
// menampilkan perangkat itu online.
func TestBroadcast_DroppingSlowClientNotifiesPresence(t *testing.T) {
	h := NewHub("arena-1")
	presence := make(chan struct{}, 4)
	h.Attach(h, nil, func() { presence <- struct{}{} })

	slow, _ := NewClient("tablet-lambat", RoleJudge, "arena-1") // tidak pernah dibaca: antrean 256 akan penuh
	h.Register(slow)

	for i := 0; i < cap(slow.send)+1; i++ {
		h.Broadcast(Message{Type: MsgTimerSync})
	}

	select {
	case <-presence:
	case <-time.After(2 * time.Second):
		t.Fatal("kehadiran harus diperbarui setelah klien lambat diputus")
	}
	if p := h.Presence(); len(p.JudgeSeats) != 0 {
		t.Errorf("klien yang diputus masih tercatat hadir: %+v", p)
	}
}

// Pengiriman normal tidak memicu pembaruan kehadiran.
func TestBroadcast_NoPresenceNotifyWhenNobodyDropped(t *testing.T) {
	h := NewHub("arena-1")
	called := make(chan struct{}, 1)
	h.Attach(h, nil, func() { called <- struct{}{} })
	c, _ := NewClient("juri", RoleJudge, "arena-1")
	h.Register(c)
	h.Broadcast(Message{Type: MsgTimerSync})
	select {
	case <-called:
		t.Fatal("tidak ada klien yang diputus, kehadiran tidak perlu diperbarui")
	case <-time.After(200 * time.Millisecond):
	}
}
