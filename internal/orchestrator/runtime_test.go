package orchestrator

import (
	"sync"
	"testing"
	"time"

	"github.com/herros27/digital-scoring/internal/hub"
	"github.com/herros27/digital-scoring/internal/rules"
	"github.com/herros27/digital-scoring/internal/statemachine"
	"github.com/herros27/digital-scoring/internal/store"
)

// memRuntimeStore penyimpanan pemulihan di memori untuk tes.
type memRuntimeStore struct {
	mu    sync.Mutex
	rows  map[string]store.MatchRuntime
	saves int
}

func newMemRuntimeStore() *memRuntimeStore {
	return &memRuntimeStore{rows: map[string]store.MatchRuntime{}}
}
func (m *memRuntimeStore) Save(r *store.MatchRuntime) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	cp := *r
	cp.Data = append([]byte(nil), r.Data...)
	m.rows[r.MatchID] = cp
	m.saves++
	return nil
}
func (m *memRuntimeStore) Delete(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.rows, id)
	return nil
}
func (m *memRuntimeStore) get(id string) (store.MatchRuntime, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	r, ok := m.rows[id]
	return r, ok
}

// restoreFrom meniru server baru: hub baru, orchestrator dipulihkan dari baris tersimpan.
func restoreFrom(t *testing.T, rs *memRuntimeStore) (*Orchestrator, *testRig) {
	t.Helper()
	rec, ok := rs.get("match-1")
	if !ok {
		t.Fatal("tidak ada keadaan tersimpan untuk match-1")
	}
	h := hub.NewHub("arena-1")
	o, err := Restore(rec, h, nil, nil, nil, nil, rs)
	if err != nil {
		t.Fatalf("Restore: %v", err)
	}
	r := &testRig{o: o, h: h}
	r.referee, r.refCh = hub.NewClient("ref-1", hub.RoleReferee, "arena-1")
	r.judge, r.judgeCh = hub.NewClient("judge-2", hub.RoleJudge, "arena-1")
	r.judge.SeatNumber = 2
	h.Register(r.referee)
	h.Register(r.judge)
	t.Cleanup(func() {
		if o.timer != nil {
			o.timer.Stop()
		}
	})
	return o, r
}

func rigWithStore(t *testing.T, rs *rules.Ruleset, mem *memRuntimeStore) *testRig {
	t.Helper()
	r := newRigWith(t, rs)
	r.o.SetRuntimeStore(mem)
	return r
}

// advanceToBreak mengakhiri ronde 1 lewat kedaluwarsa timer sungguhan sehingga masuk istirahat.
func advanceToBreak(t *testing.T, r *testRig) {
	t.Helper()
	r.o.timer.Stop()
	r.o.timer.Reset(0)
	r.o.onRoundExpired()
	if r.o.sm.Current() != statemachine.StateBreak1 {
		t.Fatalf("state = %s, want break_1", r.o.sm.Current())
	}
}

func TestRuntime_RestoreRunningRoundComesBackPausedWithState(t *testing.T) {
	mem := newMemRuntimeStore()
	r := rigWithStore(t, customRuleset(), mem) // 1 juri cukup, teknik catch_fall = 4
	r.start(t)
	r.send(r.judge, hub.MsgJudgeInput, map[string]any{"score_type": "catch_fall", "target": "red", "event_id": "a"})
	r.send(r.referee, hub.MsgPenalty, map[string]any{"target": "blue", "type": "teguran_x"})
	r.send(r.referee, hub.MsgArenaSignal, map[string]any{"signal": "medic", "active": true})
	r.send(r.referee, hub.MsgPenalty, map[string]any{"target": "blue", "type": "teguran_x"}) // sudah dijatuhkan -> ditolak
	want := r.o.Snapshot()

	o2, _ := restoreFrom(t, mem) // "server restart"
	got := o2.Snapshot()

	if got["state"] != "round_1" || got["round"] != 1 {
		t.Fatalf("state/round = %v/%v", got["state"], got["round"])
	}
	if got["timer_running"] != false {
		t.Fatal("ronde yang berjalan harus dipulihkan dijeda")
	}
	if got["recovered"] != true {
		t.Fatal("snapshot harus menandai recovered")
	}
	for _, k := range []string{"red_total", "blue_total", "medic_active", "ruleset_name"} {
		if got[k] != want[k] {
			t.Errorf("%s = %v, want %v", k, got[k], want[k])
		}
	}
	if got["red_total"] != 4 || got["blue_total"] != -7 {
		t.Errorf("skor = %v/%v, want 4/-7", got["red_total"], got["blue_total"])
	}
	if len(got["penalties"].([]penaltyEntry)) != 1 {
		t.Errorf("sanksi tidak pulih: %+v", got["penalties"])
	}
	// ruleset kustom ikut pulih (bukan bawaan)
	if len(got["techniques"].([]rules.TechniqueRule)) != 2 || got["judge_count"] != 1 {
		t.Errorf("ruleset tidak pulih: %v / %v", got["techniques"], got["judge_count"])
	}
	// sisa waktu mendekati yang tersimpan (ronde 120 detik, baru berjalan sebentar)
	rem := got["timer_remaining_ms"].(int64)
	if rem < 118000 || rem > 120000 {
		t.Errorf("sisa waktu = %d ms", rem)
	}
	if o2.techCount["red"]["catch_fall"] != 1 {
		t.Errorf("hitungan teknik tidak pulih: %v", o2.techCount)
	}
}

func TestRuntime_RestoredMatchCanResumeScoreAndFinish(t *testing.T) {
	mem := newMemRuntimeStore()
	r := rigWithStore(t, customRuleset(), mem)
	r.start(t)
	r.send(r.judge, hub.MsgJudgeInput, map[string]any{"score_type": "catch_fall", "target": "red", "event_id": "a"})

	o2, r2 := restoreFrom(t, mem)
	r2.send(r2.referee, hub.MsgStateChange, map[string]any{"action": "resume"})
	snap := o2.Snapshot()
	if snap["timer_running"] != true || snap["recovered"] != false {
		t.Fatalf("setelah Lanjutkan: running=%v recovered=%v", snap["timer_running"], snap["recovered"])
	}
	// konsensus berjalan lagi dengan ruleset yang dipulihkan
	r2.send(r2.judge, hub.MsgJudgeInput, map[string]any{"score_type": "technique", "target": "blue", "event_id": "b"})
	if got := o2.Snapshot()["blue_total"]; got != 1 {
		t.Fatalf("blue_total = %v, want 1", got)
	}
	if err := o2.Finish(); err != nil {
		t.Fatal(err)
	}
	if res := o2.Snapshot()["result"].(map[string]any); res["corner"] != "red" {
		t.Fatalf("hasil = %+v", res)
	}
	if _, ok := mem.get("match-1"); ok {
		t.Fatal("keadaan tersimpan harus dihapus setelah partai selesai")
	}
}

func TestRuntime_CheckpointTracksPauseAndTimerAdjust(t *testing.T) {
	mem := newMemRuntimeStore()
	r := rigWithStore(t, customRuleset(), mem)
	r.start(t)
	r.send(r.referee, hub.MsgStateChange, map[string]any{"action": "pause"})
	r.send(r.referee, hub.MsgStateChange, map[string]any{"action": "adjust_time", "delta_ms": -30000})
	rec, _ := mem.get("match-1")
	if rec.TimerRunning {
		t.Error("catatan harus menyatakan jeda")
	}
	if rec.TimerRemainingMS < 89000 || rec.TimerRemainingMS > 90000 {
		t.Errorf("sisa waktu tercatat %d ms, want ~90000", rec.TimerRemainingMS)
	}
}

func TestRuntime_NotStartedAndBreakAndTerminalStates(t *testing.T) {
	// belum dimulai (dimuat ke gelanggang): dipulihkan apa adanya, tidak ditandai recovered
	mem := newMemRuntimeStore()
	r := rigWithStore(t, customRuleset(), mem)
	_ = r
	o2, _ := restoreFrom(t, mem)
	if snap := o2.Snapshot(); snap["state"] != "idle" || snap["recovered"] != false {
		t.Fatalf("idle: %v / %v", snap["state"], snap["recovered"])
	}

	// istirahat: dipulihkan dengan hitung mundur berjalan
	mem2 := newMemRuntimeStore()
	r2 := rigWithStore(t, customRuleset(), mem2)
	r2.start(t)
	advanceToBreak(t, r2)
	o3, _ := restoreFrom(t, mem2)
	snap := o3.Snapshot()
	if snap["state"] != "break_1" || snap["timer_running"] != true || snap["recovered"] != true {
		t.Fatalf("break: %v running=%v recovered=%v", snap["state"], snap["timer_running"], snap["recovered"])
	}

	// partai selesai tidak dapat dipulihkan
	rec, _ := mem2.get("match-1")
	rec.State = string(statemachine.StateFinished)
	if _, err := Restore(rec, hub.NewHub("arena-1"), nil, nil, nil, nil, mem2); err == nil {
		t.Fatal("partai selesai tidak boleh dipulihkan")
	}
	rec.State, rec.Data = "round_1", []byte("{rusak")
	if _, err := Restore(rec, hub.NewHub("arena-1"), nil, nil, nil, nil, mem2); err == nil {
		t.Fatal("data rusak harus ditolak")
	}
}

func TestRuntime_CancelAndDQClearStoredState(t *testing.T) {
	mem := newMemRuntimeStore()
	r := rigWithStore(t, customRuleset(), mem)
	r.start(t)
	if _, ok := mem.get("match-1"); !ok {
		t.Fatal("keadaan harus tersimpan selama partai berjalan")
	}
	if err := r.o.Cancel(); err != nil {
		t.Fatal(err)
	}
	if _, ok := mem.get("match-1"); ok {
		t.Fatal("keadaan harus dihapus setelah dibatalkan (dan tidak ditulis ulang oleh siaran sesudahnya)")
	}

	mem2 := newMemRuntimeStore()
	r2 := rigWithStore(t, customRuleset(), mem2)
	r2.start(t)
	r2.send(r2.referee, hub.MsgPenalty, map[string]any{"target": "red", "type": "dq"}) // sanksi diskualifikasi
	if _, ok := mem2.get("match-1"); ok {
		t.Fatal("keadaan harus dihapus setelah partai selesai karena diskualifikasi")
	}
}

func TestRuntime_ExpiredTimerAtRoundEndRestoresAsTiePending(t *testing.T) {
	mem := newMemRuntimeStore()
	r := rigWithStore(t, &rules.Ruleset{Name: "Uji", Scoring: rules.ScoringRule{
		TechniquePoints: 1, KnockdownPoints: 2, FallPoints: 3, RoundDuration: 120, BreakDuration: 60, TotalRounds: 3,
	}}, mem)
	r.start(t)
	advanceToRound3(t, r)
	r.o.timer.Stop()
	r.o.timer.Reset(0)
	r.o.onRoundExpired() // seri penuh -> menunggu keputusan wasit
	o2, _ := restoreFrom(t, mem)
	if o2.Snapshot()["tie_pending"] != true {
		t.Fatal("menunggu keputusan seri harus tetap menunggu setelah pemulihan")
	}
	if err := o2.FinishWithDecision("blue", rules.MethodCoinToss); err != nil {
		t.Fatal(err)
	}
}

func TestRuntime_ElapsedWallClockDoesNotChangeRestoredPausedTimer(t *testing.T) {
	mem := newMemRuntimeStore()
	r := rigWithStore(t, customRuleset(), mem)
	r.start(t)
	r.send(r.referee, hub.MsgStateChange, map[string]any{"action": "pause"})
	o2, _ := restoreFrom(t, mem)
	before := o2.Snapshot()["timer_remaining_ms"].(int64)
	time.Sleep(60 * time.Millisecond)
	if after := o2.Snapshot()["timer_remaining_ms"].(int64); after != before {
		t.Fatalf("timer dijeda tidak boleh berkurang: %d -> %d", before, after)
	}
}
