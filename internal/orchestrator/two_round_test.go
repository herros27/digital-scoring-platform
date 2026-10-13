package orchestrator

import (
	"testing"

	"github.com/herros27/digital-scoring/internal/rules"
	"github.com/herros27/digital-scoring/internal/statemachine"
)

func twoRoundRuleset(totalRounds int) *rules.Ruleset {
	return &rules.Ruleset{Name: "Uji", Scoring: rules.ScoringRule{
		TechniquePoints: 1, KnockdownPoints: 2, FallPoints: 3, RoundDuration: 90, BreakDuration: 60, TotalRounds: totalRounds,
	}}
}

// Pertandingan 2 babak (Usia Dini) selesai setelah ronde 2, tidak lanjut ke istirahat/ronde 3.
func TestTwoRoundMatch_FinishesAfterRound2(t *testing.T) {
	r := newRigWith(t, twoRoundRuleset(2))
	r.start(t)
	r.o.mu.Lock()
	r.o.scores[1].RedTechnical = 4 // merah unggul agar tidak seri
	r.o.mu.Unlock()

	advanceToBreak(t, r) // ronde 1 -> istirahat 1
	r.o.timer.Stop()
	r.o.onBreakExpired() // ronde 2 dimulai
	if got := r.o.sm.Current(); got != statemachine.StateRound2 {
		t.Fatalf("state = %s, want round_2", got)
	}

	r.o.timer.Stop()
	r.o.timer.Reset(0)
	r.o.onRoundExpired()
	if got := r.o.sm.Current(); got != statemachine.StateFinished {
		t.Fatalf("2-round match must finish after round 2, state = %s", got)
	}

	r.o.mu.Lock()
	n := r.o.roundNumberLocked()
	r.o.mu.Unlock()
	if n != 2 {
		t.Errorf("round number after finish = %d, want 2 (clamped to total_rounds)", n)
	}
}

// Pertandingan 3 babak tetap lanjut ke istirahat 2 setelah ronde 2 (perilaku lama tidak berubah).
func TestThreeRoundMatch_StillGoesToBreak2(t *testing.T) {
	r := newRigWith(t, twoRoundRuleset(3))
	r.start(t)
	advanceToBreak(t, r)
	r.o.timer.Stop()
	r.o.onBreakExpired()
	r.o.timer.Stop()
	r.o.timer.Reset(0)
	r.o.onRoundExpired()
	if got := r.o.sm.Current(); got != statemachine.StateBreak2 {
		t.Fatalf("3-round match after round 2: state = %s, want break_2", got)
	}
}

// Seri di ronde terakhir pertandingan 2 babak menunggu keputusan wasit (bukan langsung selesai).
func TestTwoRoundMatch_TieWaitsForRefereeDecision(t *testing.T) {
	r := newRigWith(t, twoRoundRuleset(2))
	r.start(t)
	advanceToBreak(t, r)
	r.o.timer.Stop()
	r.o.onBreakExpired()
	r.o.timer.Stop()
	r.o.timer.Reset(0)
	r.o.onRoundExpired() // skor 0-0 -> seri
	if r.o.sm.Current() == statemachine.StateFinished {
		t.Fatal("tied 2-round match must wait for referee decision")
	}
	r.o.mu.Lock()
	pending := r.o.tiePendingLocked()
	r.o.mu.Unlock()
	if !pending {
		t.Error("tie must be pending in final round of a 2-round match")
	}
}
