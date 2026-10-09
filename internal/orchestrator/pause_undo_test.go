package orchestrator

import (
	"testing"

	"github.com/herros27/digital-scoring/internal/hub"
	"github.com/herros27/digital-scoring/internal/statemachine"
	"github.com/herros27/digital-scoring/internal/store"
)

func progressCounts(msgs []hub.Message) []int {
	var out []int
	for _, m := range msgs {
		if m.Type != hub.MsgConsensusProgress {
			continue
		}
		if p, ok := m.Payload.(map[string]any); ok {
			if c, ok := p["count"].(int); ok {
				out = append(out, c)
			}
		}
	}
	return out
}

func TestJudgeScoreRejectedWhilePaused(t *testing.T) {
	r := newRigWith(t, customRuleset()) // satu juri cukup: skor langsung terkonfirmasi bila diterima
	r.start(t)
	r.send(r.referee, hub.MsgStateChange, map[string]any{"action": "pause"})
	drain(r.judgeCh)

	r.send(r.judge, hub.MsgJudgeInput, map[string]any{"score_type": "technique", "target": "red", "event_id": "p1"})
	if c := errCode(drain(r.judgeCh)); c != "SCORING_NOT_ALLOWED" {
		t.Fatalf("kode = %q, want SCORING_NOT_ALLOWED saat dijeda", c)
	}
	if got := r.o.Snapshot()["red_total"]; got != 0 {
		t.Fatalf("red_total = %v, skor tidak boleh masuk saat dijeda", got)
	}

	r.send(r.referee, hub.MsgStateChange, map[string]any{"action": "resume"})
	r.send(r.judge, hub.MsgJudgeInput, map[string]any{"score_type": "technique", "target": "red", "event_id": "p2"})
	if got := r.o.Snapshot()["red_total"]; got != 1 {
		t.Fatalf("red_total = %v, want 1 setelah dilanjutkan", got)
	}
}

func TestUndoRetractsPendingInput(t *testing.T) {
	r := newRig(t) // 3 juri, 2 harus sepakat
	r.start(t)
	other, otherCh := hub.NewClient("judge-3", hub.RoleJudge, "arena-1")
	other.SeatNumber = 3
	r.h.Register(other)

	r.send(r.judge, hub.MsgJudgeInput, map[string]any{"score_type": "technique", "target": "red", "event_id": "u1"})
	r.send(r.judge, hub.MsgJudgeInput, map[string]any{"score_type": "undo", "target": "red", "event_id": "u2"})
	if got := progressCounts(drain(otherCh)); len(got) != 2 || got[0] != 1 || got[1] != 0 {
		t.Fatalf("progres = %v, want [1 0] (tekan lalu batal)", got)
	}

	// juri lain menekan setelah pembatalan: baru 1 suara, belum jadi poin
	r.send(other, hub.MsgJudgeInput, map[string]any{"score_type": "technique", "target": "red", "event_id": "u3"})
	if got := r.o.Snapshot()["red_total"]; got != 0 {
		t.Fatalf("red_total = %v: input yang dibatalkan tidak boleh ikut dihitung", got)
	}
}

func TestUndoWithoutPendingInputIsReported(t *testing.T) {
	r := newRig(t)
	r.start(t)
	drain(r.judgeCh)
	r.send(r.judge, hub.MsgJudgeInput, map[string]any{"score_type": "undo", "target": "red", "event_id": "n1"})
	if c := errCode(drain(r.judgeCh)); c != "UNDO_NOT_POSSIBLE" {
		t.Fatalf("kode = %q, want UNDO_NOT_POSSIBLE", c)
	}
	r.send(r.judge, hub.MsgJudgeInput, map[string]any{"score_type": "undo", "target": "hijau", "event_id": "n2"})
	if c := errCode(drain(r.judgeCh)); c != "BAD_PAYLOAD" {
		t.Fatalf("kode = %q, want BAD_PAYLOAD untuk target tidak valid", c)
	}
}

func TestUndoDoesNotRevertConfirmedPoints(t *testing.T) {
	r := newRigWith(t, customRuleset())
	r.start(t)
	r.send(r.judge, hub.MsgJudgeInput, map[string]any{"score_type": "technique", "target": "red", "event_id": "c1"})
	drain(r.judgeCh)
	r.send(r.judge, hub.MsgJudgeInput, map[string]any{"score_type": "undo", "target": "red", "event_id": "c2"})
	if c := errCode(drain(r.judgeCh)); c != "UNDO_NOT_POSSIBLE" {
		t.Fatalf("kode = %q, poin terkonfirmasi tidak boleh dibatalkan juri", c)
	}
	if got := r.o.Snapshot()["red_total"]; got != 1 {
		t.Fatalf("red_total = %v, want 1", got)
	}
}

// statusRecorder MatchStore palsu yang mencatat urutan status yang disimpan.
type statusRecorder struct{ statuses []string }

func (s *statusRecorder) UpdateStatus(_, status string) error {
	s.statuses = append(s.statuses, status)
	return nil
}
func (s *statusRecorder) SetWinner(string, string) error       { return nil }
func (s *statusRecorder) GetByID(string) (*store.Match, error) { return nil, nil }

// Daftar partai di UI panitia membaca status dari database: jeda harus tersimpan sebagai "paused" (agar tombol
// Lanjutkan muncul) dan kembali "running" saat dilanjutkan.
func TestPauseAndResumePersistMatchStatus(t *testing.T) {
	rec := &statusRecorder{}
	h := hub.NewHub("arena-1")
	o := New("match-1", "arena-1", statemachine.New("match-1"), customRuleset(), h, rec, nil, nil, nil)
	referee, _ := hub.NewClient("ref-1", hub.RoleReferee, "arena-1")
	h.Register(referee)

	for _, action := range []string{"start", "pause", "resume"} {
		o.handleMessage("arena-1", referee, hub.Message{Type: hub.MsgStateChange, Payload: map[string]any{"action": action}})
	}
	want := []string{"running", "paused", "running"}
	if len(rec.statuses) != len(want) {
		t.Fatalf("status tersimpan = %v, want %v", rec.statuses, want)
	}
	for i := range want {
		if rec.statuses[i] != want[i] {
			t.Fatalf("status tersimpan = %v, want %v", rec.statuses, want)
		}
	}
}
