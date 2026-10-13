package orchestrator

import (
	"testing"

	"github.com/herros27/digital-scoring/internal/consensus"
	"github.com/herros27/digital-scoring/internal/hub"
	"github.com/herros27/digital-scoring/internal/rules"
	"github.com/herros27/digital-scoring/internal/statemachine"
)

func gapRig(t *testing.T, gap *rules.PointGapRule) *testRig {
	t.Helper()
	return newRigWith(t, &rules.Ruleset{Name: "Uji", Scoring: rules.ScoringRule{
		TechniquePoints: 1, KnockdownPoints: 2, FallPoints: 3, RoundDuration: 120, BreakDuration: 60, TotalRounds: 3,
		PointGap: gap,
	}})
}

func confirm(r *testRig, target consensus.Target, round int) {
	r.o.applyScore(&consensus.ConsensusResult{Accepted: true, ScoreType: "technique", Target: target, Round: round, Count: 2})
}

func toRound2(t *testing.T, r *testRig) {
	t.Helper()
	advanceToBreak(t, r)
	r.o.onBreakExpired()
	if r.o.sm.Current() != statemachine.StateRound2 {
		t.Fatalf("state = %s, want round_2", r.o.sm.Current())
	}
}

// WMP: tidak berlaku di babak 1; mulai babak 2 yang dihitung adalah TOTAL poin semua babak, dan pemenangnya
// sudut yang unggul dengan result_type "wmp".
func TestPointGap_AppliesFromRound2OnTotals(t *testing.T) {
	r := gapRig(t, &rules.PointGapRule{Margin: 5})
	r.start(t)
	r.o.scores[1].RedTechnical = 8 // selisih sudah melewati batas, tetapi masih babak 1
	confirm(r, consensus.TargetRed, 1)
	if r.o.sm.IsFinished() {
		t.Fatal("WMP tidak boleh berlaku pada babak 1")
	}

	toRound2(t, r)
	confirm(r, consensus.TargetBlue, 2) // total merah 9 vs biru 1 = selisih 8 >= 5
	if !r.o.sm.IsFinished() {
		t.Fatal("selisih total mencapai batas pada babak 2, pertandingan harus dihentikan")
	}
	if r.o.result["corner"] != "red" || r.o.result["result_type"] != "wmp" {
		t.Errorf("result = %v, want corner red / wmp", r.o.result)
	}
	if find(drain(r.tvCh), hub.MsgMatchResult) == nil {
		t.Error("hasil WMP harus disiarkan ke layar")
	}
}

func TestPointGap_BelowMarginKeepsPlaying(t *testing.T) {
	r := gapRig(t, &rules.PointGapRule{Margin: 5})
	r.start(t)
	toRound2(t, r)
	r.o.scores[2].RedTechnical = 3
	confirm(r, consensus.TargetRed, 2) // selisih 4 < 5
	if r.o.sm.IsFinished() {
		t.Fatal("selisih di bawah batas tidak boleh menghentikan pertandingan")
	}
}

func TestPointGap_DefaultIs30AndZeroDisables(t *testing.T) {
	r := gapRig(t, nil)
	r.start(t)
	toRound2(t, r)
	r.o.scores[2].BlueTechnical = 28
	confirm(r, consensus.TargetBlue, 2) // 29
	if r.o.sm.IsFinished() {
		t.Fatal("29 < 30 belum menghentikan pertandingan")
	}
	confirm(r, consensus.TargetBlue, 2) // 30
	if !r.o.sm.IsFinished() || r.o.result["corner"] != "blue" {
		t.Fatalf("selisih 30 harus menghentikan dan biru menang: %v", r.o.result)
	}

	off := gapRig(t, &rules.PointGapRule{Margin: 0})
	off.start(t)
	toRound2(t, off)
	off.o.scores[2].RedTechnical = 500
	confirm(off, consensus.TargetRed, 2)
	if off.o.sm.IsFinished() {
		t.Fatal("margin 0 menonaktifkan WMP")
	}
}

// Sanksi mengurangi nilai sudut yang dikenai, jadi lawannya bisa menang WMP karenanya.
func TestPointGap_TriggeredByPenalty(t *testing.T) {
	r := gapRig(t, &rules.PointGapRule{Margin: 8})
	r.start(t)
	toRound2(t, r)
	r.o.scores[2].RedTechnical = 3
	r.send(r.referee, hub.MsgPenalty, map[string]any{"target": "blue", "type": "peringatan_1"}) // biru -5 => selisih 8
	if !r.o.sm.IsFinished() || r.o.result["corner"] != "red" || r.o.result["result_type"] != "wmp" {
		t.Fatalf("sanksi harus memicu WMP untuk merah: finished=%v result=%v", r.o.sm.IsFinished(), r.o.result)
	}
}
