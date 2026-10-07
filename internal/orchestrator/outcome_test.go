package orchestrator

import (
	"strings"
	"testing"

	"github.com/herros27/digital-scoring/internal/hub"
	"github.com/herros27/digital-scoring/internal/rules"
)

func declare(r *testRig, by *hub.Client, outcome, winner string) {
	r.send(by, hub.MsgStateChange, map[string]any{"action": "declare_outcome", "outcome": outcome, "winner": winner})
}

func TestOutcome_KOandTKOInMatch(t *testing.T) {
	for _, outcome := range []string{"ko", "tko"} {
		r := newRig(t)
		r.start(t)
		drain(r.tvCh)
		declare(r, r.referee, outcome, "blue")
		if !r.o.sm.IsFinished() {
			t.Fatalf("%s: pertandingan harus selesai", outcome)
		}
		m := find(drain(r.tvCh), hub.MsgMatchResult)
		if m == nil {
			t.Fatalf("%s: match_result harus disiarkan", outcome)
		}
		p := m.Payload.(map[string]any)
		if p["corner"] != "blue" || p["result_type"] != outcome {
			t.Fatalf("%s: hasil = %+v", outcome, p)
		}
	}
}

func TestOutcome_WalkoverOnlyBeforeStart(t *testing.T) {
	r := newRig(t) // partai baru dimuat: belum dimulai
	declare(r, r.referee, "ko", "red")
	if c := errCode(drain(r.refCh)); c != "INVALID_STATE_TRANSITION" {
		t.Fatalf("KO sebelum mulai harus ditolak: %q", c)
	}
	declare(r, r.referee, "walkover", "red")
	if !r.o.sm.IsFinished() {
		t.Fatal("WO sebelum mulai harus menyelesaikan partai")
	}
	if res := resultOf(r); res["corner"] != "red" || res["result_type"] != "walkover" {
		t.Fatalf("hasil = %+v", res)
	}

	r2 := newRig(t)
	r2.start(t)
	declare(r2, r2.referee, "walkover", "red")
	if c := errCode(drain(r2.refCh)); c != "INVALID_STATE_TRANSITION" {
		t.Fatalf("WO setelah mulai harus ditolak: %q", c)
	}
	if r2.o.sm.IsFinished() {
		t.Fatal("partai tidak boleh selesai")
	}
}

func TestOutcome_WithdrawalAnyTime(t *testing.T) {
	for _, started := range []bool{false, true} {
		r := newRig(t)
		if started {
			r.start(t)
		}
		declare(r, r.referee, "withdrawal", "blue")
		if res := resultOf(r); res["corner"] != "blue" || res["result_type"] != "withdrawal" {
			t.Fatalf("started=%v: hasil = %+v", started, res)
		}
	}
}

func TestOutcome_Validation(t *testing.T) {
	r := newRig(t)
	r.start(t)
	drain(r.refCh)

	declare(r, r.judge, "ko", "red")
	if c := errCode(drain(r.judgeCh)); c != "FORBIDDEN" {
		t.Fatalf("juri tidak berhak: %q", c)
	}
	declare(r, r.referee, "sulap", "red")
	if c := errCode(drain(r.refCh)); c != "INVALID_STATE_TRANSITION" {
		t.Fatalf("jenis tak dikenal: %q", c)
	}
	declare(r, r.referee, "ko", "ungu")
	if c := errCode(drain(r.refCh)); c != "INVALID_STATE_TRANSITION" {
		t.Fatalf("pemenang tak valid: %q", c)
	}
	if err := r.o.DeclareOutcome("ko", "red", strings.Repeat("x", 201)); err == nil {
		t.Error("catatan terlalu panjang harus ditolak")
	}
	if r.o.sm.IsFinished() {
		t.Fatal("tidak ada yang boleh menyelesaikan partai")
	}

	if err := r.o.DeclareOutcome("tko", "red", "pelatih melempar handuk"); err != nil {
		t.Fatal(err)
	}
	if err := r.o.DeclareOutcome("ko", "blue", ""); err == nil {
		t.Error("hasil setelah selesai harus ditolak")
	}
}

func TestOutcome_OverridesScoreAndSnapshotListsOutcomes(t *testing.T) {
	r := newRig(t)
	r.start(t)
	r.send(r.referee, hub.MsgPenalty, map[string]any{"target": "blue", "type": "teguran_1"}) // merah unggul
	if got := len(r.o.Snapshot()["outcomes"].([]rules.OutcomeRule)); got != 4 {
		t.Errorf("outcomes pada snapshot = %d, want 4", got)
	}
	if err := r.o.DeclareOutcome("tko", "blue", ""); err != nil { // biru menang TKO walau skor kalah
		t.Fatal(err)
	}
	if res := resultOf(r); res["corner"] != "blue" || res["result_type"] != "tko" {
		t.Fatalf("hasil = %+v", res)
	}
}
