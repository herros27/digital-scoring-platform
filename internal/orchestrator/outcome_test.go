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

func TestFinish_RejectedBeforeMatchStarts(t *testing.T) {
	r := newRig(t) // partai baru dimuat: masih idle
	drain(r.tvCh)

	for _, body := range []map[string]any{
		{"action": "finish"},
		{"action": "finish", "winner": "red"},
	} {
		r.send(r.referee, hub.MsgStateChange, body)
		if c := errCode(drain(r.refCh)); c != "INVALID_STATE_TRANSITION" {
			t.Fatalf("finish %v sebelum mulai harus ditolak, got %q", body, c)
		}
	}
	if r.o.sm.IsFinished() {
		t.Fatal("partai tidak boleh selesai")
	}
	if m := find(drain(r.tvCh), hub.MsgMatchResult); m != nil {
		t.Fatalf("tidak boleh ada match_result untuk partai yang belum dimulai: %+v", m.Payload)
	}

	// setelah dimulai, finish normal berjalan
	r.start(t)
	r.send(r.referee, hub.MsgStateChange, map[string]any{"action": "finish", "winner": "blue"})
	if !r.o.sm.IsFinished() {
		t.Fatal("setelah dimulai, finish harus menyelesaikan partai")
	}
}

type recordingMatchStore struct{ statuses []string }

func (f *recordingMatchStore) UpdateStatus(_ string, status string) error {
	f.statuses = append(f.statuses, status)
	return nil
}
func (f *recordingMatchStore) FinishMatch(_, corner string) (string, error) { return corner, nil }

// Status 'running' harus tersimpan saat partai dimulai: UI membaca status dari database untuk badge LIVE
// dan penghitung "sedang tanding"; sebelumnya partai berjalan tetap tercatat 'idle'.
func TestStart_PersistsRunningStatusOnce(t *testing.T) {
	r := newRig(t)
	ms := &recordingMatchStore{}
	r.o.matchStore = ms

	r.send(r.referee, hub.MsgStateChange, map[string]any{"action": "start"})
	if len(ms.statuses) != 1 || ms.statuses[0] != "running" {
		t.Fatalf("statuses = %v, want [running]", ms.statuses)
	}

	// start kedua ditolak dan tidak menulis ulang status
	r.send(r.referee, hub.MsgStateChange, map[string]any{"action": "start"})
	if len(ms.statuses) != 1 {
		t.Fatalf("start ditolak tidak boleh menulis status: %v", ms.statuses)
	}
}

// state_change break_N harus membawa hitung mundur istirahat (berjalan + tenggat), bukan timer ronde yang
// sudah habis; kalau tidak, layar juri menampilkan 00:00 sampai timer_sync berikutnya.
func TestBreakStateChange_CarriesBreakCountdown(t *testing.T) {
	r := newRig(t) // ronde 120 dtk, istirahat 60 dtk, 3 babak
	r.start(t)
	drain(r.judgeCh)

	r.o.onRoundExpired()

	var got map[string]any
	for _, m := range drain(r.judgeCh) {
		if m.Type != hub.MsgStateChange {
			continue
		}
		if p := m.Payload.(map[string]any); p["state"] == "break_1" {
			got = p
		}
	}
	if got == nil {
		t.Fatal("juri harus menerima state_change break_1")
	}
	if got["timer_running"] != true {
		t.Fatalf("timer istirahat harus berjalan: %+v", got)
	}
	rem, _ := got["timer_remaining_ms"].(int64)
	if rem < 55_000 || rem > 60_000 {
		t.Fatalf("sisa waktu istirahat = %d ms, want ~60000", rem)
	}
	if ends, _ := got["timer_ends_at_ms"].(int64); ends <= 0 {
		t.Fatalf("timer_ends_at_ms harus terisi: %+v", got)
	}
}

// Setiap cara partai berakhir harus menyiarkan state_change "finished" ke juri, bukan hanya match_result:
// layar juri hanya mengikuti state_change.
func TestFinish_BroadcastsFinishedStateToJudges(t *testing.T) {
	r := newRig(t)
	r.start(t)
	drain(r.judgeCh)

	r.send(r.referee, hub.MsgStateChange, map[string]any{"action": "finish", "winner": "red"})

	var states []string
	var resultSeen bool
	for _, m := range drain(r.judgeCh) {
		switch m.Type {
		case hub.MsgStateChange:
			p := m.Payload.(map[string]any)
			states = append(states, p["state"].(string))
			if p["state"] == "finished" && p["timer_running"] != false {
				t.Fatalf("timer harus berhenti pada state finished: %+v", p)
			}
		case hub.MsgMatchResult:
			resultSeen = true
			if len(states) == 0 || states[len(states)-1] != "finished" {
				t.Fatalf("state_change finished harus mendahului match_result, states=%v", states)
			}
		}
	}
	if !resultSeen {
		t.Fatal("match_result harus tetap disiarkan")
	}
	if len(states) == 0 || states[len(states)-1] != "finished" {
		t.Fatalf("juri harus menerima state_change finished, got %v", states)
	}
}
