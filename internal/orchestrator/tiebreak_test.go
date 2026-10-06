package orchestrator

import (
	"testing"

	"github.com/herros27/digital-scoring/internal/hub"
	"github.com/herros27/digital-scoring/internal/rules"
)

// setTiedScores membuat skor bersih seri 5-5 dengan rincian yang diatur pemanggil.
func setTiedScores(r *testRig, redPen, bluePen int) {
	r.o.mu.Lock()
	defer r.o.mu.Unlock()
	r.o.scores[1].RedTechnical = 5 + redPen
	r.o.scores[1].RedPenalties = redPen
	r.o.scores[1].BlueTechnical = 5 + bluePen
	r.o.scores[1].BluePenalties = bluePen
}

func resultOf(r *testRig) map[string]any { return r.o.Snapshot()["result"].(map[string]any) }

func TestTiebreak_FewestPenaltiesWinsAutomatically(t *testing.T) {
	r := newRig(t)
	r.start(t)
	setTiedScores(r, 0, 2) // 5-5: biru kena potongan 2, merah bersih

	snap := r.o.Snapshot()
	auto := snap["tiebreak"].(map[string]any)["auto_result"]
	if auto == nil {
		t.Fatal("pratinjau hasil otomatis harus ada saat skor seri dan dapat diselesaikan")
	}
	if err := r.o.Finish(); err != nil {
		t.Fatalf("seri dengan sanksi berbeda harus selesai otomatis: %v", err)
	}
	if res := resultOf(r); res["corner"] != "red" || res["result_type"] != "tiebreak_penalties" {
		t.Fatalf("hasil = %+v, want red/tiebreak_penalties", res)
	}
}

func TestTiebreak_TechniquePriorityOrder(t *testing.T) {
	preset, err := rules.LoadRuleset("ipsi") // memuat catch_fall (1+3) sebagai teknik paling bernilai
	if err != nil {
		t.Fatal(err)
	}
	r := newRigWith(t, preset)
	r.start(t)
	setTiedScores(r, 1, 1) // potongan sama
	r.o.mu.Lock()
	// merah: 1 tangkapan+jatuhan; biru: 3 jatuhan dan 5 tendangan — jenis paling bernilai menentukan
	r.o.countTechnique("red", "catch_fall")
	for i := 0; i < 3; i++ {
		r.o.countTechnique("blue", "fall")
	}
	for i := 0; i < 5; i++ {
		r.o.countTechnique("blue", "knockdown")
	}
	r.o.mu.Unlock()

	if err := r.o.Finish(); err != nil {
		t.Fatal(err)
	}
	if res := resultOf(r); res["corner"] != "red" || res["result_type"] != "tiebreak_technique" {
		t.Fatalf("hasil = %+v, want red/tiebreak_technique (catch_fall paling bernilai)", res)
	}
}

func TestTiebreak_NextTechniqueWhenTopTypeEqual(t *testing.T) {
	r := newRig(t)
	r.start(t)
	setTiedScores(r, 0, 0)
	r.o.mu.Lock()
	r.o.countTechnique("red", "fall")
	r.o.countTechnique("blue", "fall")
	r.o.countTechnique("blue", "knockdown")
	r.o.mu.Unlock()
	if err := r.o.Finish(); err != nil {
		t.Fatal(err)
	}
	if res := resultOf(r); res["corner"] != "blue" {
		t.Fatalf("hasil = %+v, want blue (tendangan lebih banyak setelah jatuhan sama)", res)
	}
}

func TestTiebreak_FullyTiedNeedsManualMethod(t *testing.T) {
	r := newRig(t)
	r.start(t)
	setTiedScores(r, 1, 1)

	if err := r.o.Finish(); err != ErrTieNeedsDecision {
		t.Fatalf("seri penuh: err = %v, want ErrTieNeedsDecision", err)
	}
	if err := r.o.FinishWithDecision("red", "ngawur"); err == nil {
		t.Error("metode tidak dikenal harus ditolak")
	}
	if r.o.sm.IsFinished() {
		t.Fatal("belum boleh selesai")
	}
	if err := r.o.FinishWithDecision("blue", rules.MethodCoinToss); err != nil {
		t.Fatal(err)
	}
	if res := resultOf(r); res["corner"] != "blue" || res["result_type"] != "tiebreak_coin_toss" {
		t.Fatalf("hasil = %+v, want blue/tiebreak_coin_toss", res)
	}
}

func TestTiebreak_Round3ExpiryResolvesAutomatically(t *testing.T) {
	r := newRig(t)
	r.start(t)
	advanceToRound3(t, r)
	r.o.mu.Lock()
	r.o.scores[1].RedTechnical, r.o.scores[1].BlueTechnical = 5, 6
	r.o.scores[1].BluePenalties = 1 // 5-5, biru kena potongan
	r.o.mu.Unlock()
	r.o.timer.Stop()
	r.o.timer.Reset(0)

	r.o.onRoundExpired()

	if !r.o.sm.IsFinished() {
		t.Fatal("seri yang dapat diselesaikan aturan seri harus berakhir otomatis")
	}
	if res := resultOf(r); res["corner"] != "red" || res["result_type"] != "tiebreak_penalties" {
		t.Fatalf("hasil = %+v", res)
	}
}

func TestTiebreak_RefereeOverrideStillAllowedAndLabelled(t *testing.T) {
	r := newRig(t)
	r.start(t)
	setTiedScores(r, 0, 2)
	if err := r.o.FinishWithWinner("blue"); err != nil {
		t.Fatal(err)
	}
	if res := resultOf(r); res["corner"] != "blue" || res["result_type"] != "referee_decision" {
		t.Fatalf("hasil = %+v", res)
	}
}

func TestTiebreak_ViaWebSocketFinishWithMethod(t *testing.T) {
	r := newRig(t)
	r.start(t)
	setTiedScores(r, 1, 1)
	r.send(r.referee, hub.MsgStateChange, map[string]any{"action": "finish", "winner": "red", "method": "lighter_weight"})
	if res := resultOf(r); res["result_type"] != "tiebreak_lighter_weight" {
		t.Fatalf("hasil = %+v", res)
	}
}

func TestTiebreak_CountsOnlyConfirmedScores(t *testing.T) {
	r := newRigWith(t, customRuleset()) // satu juri cukup
	r.start(t)
	r.send(r.judge, hub.MsgJudgeInput, map[string]any{"score_type": "catch_fall", "target": "red", "event_id": "x1"})
	r.send(r.judge, hub.MsgJudgeInput, map[string]any{"score_type": "undo", "target": "red", "event_id": "x2"})
	r.o.mu.Lock()
	defer r.o.mu.Unlock()
	if got := r.o.techCount["red"]["catch_fall"]; got != 1 {
		t.Errorf("catch_fall = %d, want 1", got)
	}
	if _, ok := r.o.techCount["red"]["undo"]; ok {
		t.Error("undo tidak boleh dihitung sebagai teknik")
	}
}
