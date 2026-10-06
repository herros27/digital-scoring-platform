package orchestrator

import (
	"testing"

	"github.com/herros27/digital-scoring/internal/hub"
)

func TestConsensusProgress_SentOnlyToJudgesAndCountsAgreement(t *testing.T) {
	r := newRig(t) // 2 dari 3 juri dalam 3 detik
	judge1, ch1 := hub.NewClient("judge-1", hub.RoleJudge, "arena-1")
	judge1.SeatNumber = 1
	r.h.Register(judge1)
	r.start(t)
	drain(r.judgeCh)
	drain(ch1)
	drain(r.refCh)
	drain(r.tvCh)

	// juri 2 menekan Pukulan merah: baru 1 dari 2
	r.send(r.judge, hub.MsgJudgeInput, map[string]any{"score_type": "technique", "target": "red", "event_id": "p1"})
	m := find(drain(ch1), hub.MsgConsensusProgress)
	if m == nil {
		t.Fatal("juri lain harus menerima kemajuan konsensus")
	}
	p := m.Payload.(map[string]any)
	if p["count"] != 1 || p["needed"] != 2 || p["target"] == nil || p["window_ms"] != int64(3000) {
		t.Fatalf("payload = %+v", p)
	}
	if find(drain(r.judgeCh), hub.MsgConsensusProgress) == nil {
		t.Error("juri yang menekan juga menerima kemajuan (konfirmasi tekanannya tercatat)")
	}
	if find(drain(r.refCh), hub.MsgConsensusProgress) != nil || find(drain(r.tvCh), hub.MsgConsensusProgress) != nil {
		t.Error("wasit dan scoreboard tidak boleh menerima kemajuan konsensus")
	}

	// juri 1 sepakat: poin sah, tidak ada lagi kemajuan "menunggu"
	r.send(judge1, hub.MsgJudgeInput, map[string]any{"score_type": "technique", "target": "red", "event_id": "p2"})
	msgs := drain(ch1)
	if find(msgs, hub.MsgScoreConfirmed) == nil {
		t.Fatal("kesepakatan harus menghasilkan score_confirmed")
	}
	if find(msgs, hub.MsgConsensusProgress) != nil {
		t.Error("setelah terkonfirmasi tidak ada kemajuan tambahan")
	}
}

func TestConsensusProgress_DifferentTechniqueDoesNotAddUp(t *testing.T) {
	r := newRig(t)
	judge1, ch1 := hub.NewClient("judge-1", hub.RoleJudge, "arena-1")
	judge1.SeatNumber = 1
	r.h.Register(judge1)
	r.start(t)
	drain(ch1)

	r.send(r.judge, hub.MsgJudgeInput, map[string]any{"score_type": "technique", "target": "red", "event_id": "q1"})
	r.send(judge1, hub.MsgJudgeInput, map[string]any{"score_type": "knockdown", "target": "red", "event_id": "q2"})
	var counts []any
	for _, m := range drain(ch1) {
		if m.Type == hub.MsgConsensusProgress {
			counts = append(counts, m.Payload.(map[string]any)["count"])
		}
	}
	if len(counts) != 2 || counts[0] != 1 || counts[1] != 1 {
		t.Fatalf("teknik berbeda tidak boleh dijumlahkan: %v", counts)
	}
	if got := r.o.Snapshot()["red_total"]; got != 0 {
		t.Fatalf("red_total = %v, want 0", got)
	}
}
