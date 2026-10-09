package orchestrator

import (
	"testing"

	"github.com/herros27/digital-scoring/internal/hub"
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
