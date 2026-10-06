package orchestrator

import (
	"testing"

	"github.com/herros27/digital-scoring/internal/hub"
	"github.com/herros27/digital-scoring/internal/rules"
	"github.com/herros27/digital-scoring/internal/statemachine"
)

type testRig struct {
	o       *Orchestrator
	h       *hub.Hub
	referee *hub.Client
	refCh   <-chan hub.Message
	judge   *hub.Client
	judgeCh <-chan hub.Message
	tv      *hub.Client
	tvCh    <-chan hub.Message
}

func newRig(t *testing.T) *testRig {
	t.Helper()
	h := hub.NewHub("arena-1")
	rs := &rules.Ruleset{Name: "Uji", Scoring: rules.ScoringRule{
		TechniquePoints: 1, KnockdownPoints: 2, FallPoints: 3, RoundDuration: 120, BreakDuration: 60, TotalRounds: 3,
	}}
	o := New("match-1", "arena-1", statemachine.New("match-1"), rs, h, nil, nil, nil, nil)

	r := &testRig{o: o, h: h}
	r.referee, r.refCh = hub.NewClient("ref-1", hub.RoleReferee, "arena-1")
	r.judge, r.judgeCh = hub.NewClient("judge-2", hub.RoleJudge, "arena-1")
	r.judge.SeatNumber = 2
	r.tv, r.tvCh = hub.NewClient("tv-1", hub.RoleScoreboard, "arena-1")
	h.Register(r.referee)
	h.Register(r.judge)
	h.Register(r.tv)
	t.Cleanup(func() {
		if o.timer != nil {
			o.timer.Stop()
		}
	})
	return r
}

func (r *testRig) send(c *hub.Client, typ hub.MessageType, payload map[string]any) {
	r.o.handleMessage("arena-1", c, hub.Message{Type: typ, Payload: payload})
}

func (r *testRig) start(t *testing.T) {
	t.Helper()
	r.send(r.referee, hub.MsgStateChange, map[string]any{"action": "start"})
	if got := r.o.sm.Current(); got != statemachine.StateRound1 {
		t.Fatalf("state setelah start = %s", got)
	}
	drain(r.refCh)
}

func drain(ch <-chan hub.Message) []hub.Message {
	var out []hub.Message
	for {
		select {
		case m := <-ch:
			out = append(out, m)
		default:
			return out
		}
	}
}

func find(msgs []hub.Message, typ hub.MessageType) *hub.Message {
	for i := len(msgs) - 1; i >= 0; i-- {
		if msgs[i].Type == typ {
			return &msgs[i]
		}
	}
	return nil
}

func errCode(msgs []hub.Message) string {
	if m := find(msgs, hub.MsgError); m != nil {
		return m.Payload.(map[string]string)["code"]
	}
	return ""
}

func TestPenalty_AppliedFromCatalogAndReflectedInSnapshot(t *testing.T) {
	r := newRig(t)
	r.start(t)

	r.send(r.referee, hub.MsgPenalty, map[string]any{"target": "red", "type": "teguran_1"})
	r.send(r.referee, hub.MsgPenalty, map[string]any{"target": "red", "type": "peringatan_1"})
	msgs := drain(r.refCh)

	snap := r.o.Snapshot()
	if got := snap["red_total"]; got != -6 {
		t.Fatalf("red_total = %v, want -6 (teguran -1, peringatan -5)", got)
	}
	pens := snap["penalties"].([]penaltyEntry)
	if len(pens) != 2 || pens[0].Type != "teguran_1" || pens[1].Points != 5 || pens[0].ID == pens[1].ID {
		t.Fatalf("daftar sanksi salah: %+v", pens)
	}
	if m := find(msgs, hub.MsgPenaltyUpdate); m == nil {
		t.Fatal("daftar sanksi harus disiarkan")
	}
	if m := find(msgs, hub.MsgScoreUpdate); m == nil || m.Payload.(map[string]any)["red_penalties"] != 6 {
		t.Fatalf("score_update harus memuat potongan 6: %+v", m)
	}
	// penonton publik juga menerima daftar sanksi
	if find(drain(r.tvCh), hub.MsgPenaltyUpdate) == nil {
		t.Error("scoreboard harus menerima daftar sanksi")
	}
	if len(snap["penalty_catalog"].([]penaltyKind)) != len(penaltyCatalog) {
		t.Error("katalog sanksi harus ada pada snapshot")
	}
}

func TestPenalty_Validation(t *testing.T) {
	r := newRig(t)

	// belum mulai
	r.send(r.referee, hub.MsgPenalty, map[string]any{"target": "red", "type": "teguran_1"})
	if c := errCode(drain(r.refCh)); c != "PENALTY_NOT_ALLOWED" {
		t.Fatalf("sebelum mulai: %q", c)
	}
	r.start(t)

	cases := []struct {
		name    string
		by      *hub.Client
		payload map[string]any
		want    string
	}{
		{"juri tidak berhak", r.judge, map[string]any{"target": "red", "type": "teguran_1"}, "FORBIDDEN"},
		{"jenis tidak dikenal", r.referee, map[string]any{"target": "red", "type": "ngawur"}, "BAD_PAYLOAD"},
		{"target tidak valid", r.referee, map[string]any{"target": "hijau", "type": "teguran_1"}, "BAD_PAYLOAD"},
		{"tanpa type (format lama)", r.referee, map[string]any{"target": "red", "level": 5}, "BAD_PAYLOAD"},
	}
	for _, c := range cases {
		r.send(c.by, hub.MsgPenalty, c.payload)
		var ch <-chan hub.Message = r.refCh
		if c.by == r.judge {
			ch = r.judgeCh
		}
		if got := errCode(drain(ch)); got != c.want {
			t.Errorf("%s: kode = %q, want %q", c.name, got, c.want)
		}
	}

	r.send(r.referee, hub.MsgPenalty, map[string]any{"target": "blue", "type": "binaan_1"})
	drain(r.refCh)
	r.send(r.referee, hub.MsgPenalty, map[string]any{"target": "blue", "type": "binaan_1"})
	if c := errCode(drain(r.refCh)); c != "PENALTY_ALREADY_APPLIED" {
		t.Errorf("sanksi ganda: %q", c)
	}
	if got := r.o.Snapshot()["blue_total"]; got != 0 {
		t.Errorf("binaan tidak boleh memotong poin, blue_total = %v", got)
	}
}

func TestPenaltyRevoke_RemovesLastAndRestoresPoints(t *testing.T) {
	r := newRig(t)
	r.start(t)

	r.send(r.referee, hub.MsgPenalty, map[string]any{"target": "red", "type": "teguran_1"})
	r.send(r.referee, hub.MsgPenalty, map[string]any{"target": "red", "type": "peringatan_1"})
	r.send(r.referee, hub.MsgPenalty, map[string]any{"target": "blue", "type": "teguran_1"})
	drain(r.refCh)

	r.send(r.referee, hub.MsgPenaltyRevoke, map[string]any{"target": "red"})
	msgs := drain(r.refCh)
	snap := r.o.Snapshot()
	if got := snap["red_total"]; got != -1 {
		t.Fatalf("setelah membatalkan peringatan_1, red_total = %v, want -1", got)
	}
	if got := snap["blue_total"]; got != -1 {
		t.Errorf("sanksi sudut biru tidak boleh ikut terbatalkan: %v", got)
	}
	pens := snap["penalties"].([]penaltyEntry)
	if len(pens) != 2 {
		t.Fatalf("sisa sanksi = %d, want 2", len(pens))
	}
	if find(msgs, hub.MsgPenaltyUpdate) == nil {
		t.Error("daftar sanksi harus disiarkan ulang")
	}

	// sanksi yang sama boleh dijatuhkan lagi setelah dibatalkan
	r.send(r.referee, hub.MsgPenalty, map[string]any{"target": "red", "type": "peringatan_1"})
	if c := errCode(drain(r.refCh)); c != "" {
		t.Errorf("penjatuhan ulang ditolak: %q", c)
	}

	r.send(r.referee, hub.MsgPenaltyRevoke, map[string]any{"target": "red"})
	r.send(r.referee, hub.MsgPenaltyRevoke, map[string]any{"target": "red"})
	drain(r.refCh)
	r.send(r.referee, hub.MsgPenaltyRevoke, map[string]any{"target": "red"})
	if c := errCode(drain(r.refCh)); c != "NOTHING_TO_REVOKE" {
		t.Errorf("tanpa sanksi: %q", c)
	}
	r.send(r.judge, hub.MsgPenaltyRevoke, map[string]any{"target": "blue"})
	if c := errCode(drain(r.judgeCh)); c != "FORBIDDEN" {
		t.Errorf("juri membatalkan sanksi: %q", c)
	}
}

func TestDisqualify_OpponentWinsAndMatchEnds(t *testing.T) {
	r := newRig(t)

	r.send(r.referee, hub.MsgStateChange, map[string]any{"action": "disqualify", "target": "red"})
	if c := errCode(drain(r.refCh)); c != "INVALID_STATE_TRANSITION" {
		t.Fatalf("DQ sebelum mulai harus ditolak: %q", c)
	}

	r.start(t)
	r.send(r.referee, hub.MsgStateChange, map[string]any{"action": "disqualify", "target": "red"})
	msgs := drain(r.tvCh)

	if !r.o.sm.IsFinished() {
		t.Fatal("pertandingan harus selesai setelah diskualifikasi")
	}
	m := find(msgs, hub.MsgMatchResult)
	if m == nil {
		t.Fatal("match_result harus disiarkan")
	}
	p := m.Payload.(map[string]any)
	if p["corner"] != "blue" || p["result_type"] != "disqualification" {
		t.Fatalf("hasil = %+v, want blue/disqualification", p)
	}
	res := r.o.Snapshot()["result"].(map[string]any)
	if res["corner"] != "blue" || res["result_type"] != "disqualification" {
		t.Errorf("snapshot harus menyimpan hasil akhir: %+v", res)
	}

	r.send(r.referee, hub.MsgStateChange, map[string]any{"action": "disqualify", "target": "blue"})
	if c := errCode(drain(r.refCh)); c != "INVALID_STATE_TRANSITION" {
		t.Errorf("DQ setelah selesai: %q", c)
	}
	r.send(r.referee, hub.MsgStateChange, map[string]any{"action": "disqualify", "target": "ungu"})
	if c := errCode(drain(r.refCh)); c == "" {
		t.Error("target tidak valid harus ditolak")
	}
}

func TestAdjustTimeAndResetRound(t *testing.T) {
	r := newRig(t)

	r.send(r.referee, hub.MsgStateChange, map[string]any{"action": "adjust_time", "delta_ms": -10000})
	if c := errCode(drain(r.refCh)); c != "INVALID_STATE_TRANSITION" {
		t.Fatalf("koreksi waktu sebelum ronde: %q", c)
	}

	r.start(t)
	r.send(r.referee, hub.MsgStateChange, map[string]any{"action": "pause"})
	drain(r.refCh)

	rem := func() int64 { return r.o.Snapshot()["timer_remaining_ms"].(int64) }
	r.send(r.referee, hub.MsgStateChange, map[string]any{"action": "adjust_time", "delta_ms": -10000})
	if got := rem(); got != 110000 {
		t.Fatalf("setelah -10 detik = %d, want 110000", got)
	}
	if m := find(drain(r.tvCh), hub.MsgStateChange); m == nil || m.Payload.(map[string]any)["timer_remaining_ms"] != int64(110000) {
		t.Error("koreksi waktu harus disiarkan lewat state_change")
	}

	r.send(r.referee, hub.MsgStateChange, map[string]any{"action": "adjust_time", "delta_ms": 50000})
	if got := rem(); got != 120000 {
		t.Errorf("tidak boleh melebihi durasi ronde: %d", got)
	}
	r.send(r.referee, hub.MsgStateChange, map[string]any{"action": "adjust_time", "delta_ms": -999999})
	if got := rem(); got != 1000 {
		t.Errorf("batas bawah 1 detik: %d", got)
	}

	r.send(r.referee, hub.MsgStateChange, map[string]any{"action": "reset_round"})
	if got := rem(); got != 120000 {
		t.Errorf("reset ronde = %d, want 120000", got)
	}

	r.send(r.referee, hub.MsgStateChange, map[string]any{"action": "resume"})
	drain(r.refCh)
	r.send(r.referee, hub.MsgStateChange, map[string]any{"action": "reset_round"})
	if c := errCode(drain(r.refCh)); c != "INVALID_STATE_TRANSITION" {
		t.Errorf("reset saat berjalan harus ditolak: %q", c)
	}
	r.send(r.judge, hub.MsgStateChange, map[string]any{"action": "adjust_time", "delta_ms": 1000})
	if c := errCode(drain(r.judgeCh)); c != "FORBIDDEN" {
		t.Errorf("juri koreksi waktu: %q", c)
	}
}

func TestArenaSignals(t *testing.T) {
	r := newRig(t)

	r.send(r.referee, hub.MsgArenaSignal, map[string]any{"signal": "gong"})
	m := find(drain(r.tvCh), hub.MsgArenaSignal)
	if m == nil || m.Payload.(map[string]any)["signal"] != "gong" {
		t.Fatalf("gong harus disiarkan ke scoreboard: %+v", m)
	}

	r.send(r.referee, hub.MsgArenaSignal, map[string]any{"signal": "medic", "active": true})
	r.send(r.referee, hub.MsgArenaSignal, map[string]any{"signal": "var", "active": true})
	snap := r.o.Snapshot()
	if snap["medic_active"] != true || snap["var_active"] != true {
		t.Fatalf("status medic/var harus tersimpan: %v %v", snap["medic_active"], snap["var_active"])
	}
	r.send(r.referee, hub.MsgArenaSignal, map[string]any{"signal": "medic", "active": false})
	if r.o.Snapshot()["medic_active"] != false {
		t.Error("medic harus bisa dimatikan")
	}
	drain(r.refCh)

	r.send(r.referee, hub.MsgArenaSignal, map[string]any{"signal": "medic"})
	if c := errCode(drain(r.refCh)); c != "BAD_PAYLOAD" {
		t.Errorf("medic tanpa active: %q", c)
	}
	r.send(r.referee, hub.MsgArenaSignal, map[string]any{"signal": "sirine"})
	if c := errCode(drain(r.refCh)); c != "BAD_PAYLOAD" {
		t.Errorf("sinyal tidak dikenal: %q", c)
	}
	r.send(r.judge, hub.MsgArenaSignal, map[string]any{"signal": "gong"})
	if c := errCode(drain(r.judgeCh)); c != "FORBIDDEN" {
		t.Errorf("juri membunyikan gong: %q", c)
	}
}

func TestPresenceAndJudgeActivity(t *testing.T) {
	r := newRig(t)
	r.start(t)

	pres := r.o.Snapshot()["presence"].(map[string]any)
	judges := pres["judges"].([]map[string]any)
	if len(judges) != 5 || judges[1]["connected"] != true || judges[0]["connected"] != false {
		t.Fatalf("hanya kursi 2 yang online: %+v", judges)
	}
	if pres["scoreboards"] != 1 || pres["referees"] != 1 {
		t.Errorf("hitungan perangkat salah: %+v", pres)
	}

	r.send(r.judge, hub.MsgJudgeInput, map[string]any{"score_type": "technique", "target": "red", "round": 1, "event_id": "e1"})
	msgs := drain(r.refCh)
	pres = r.o.Snapshot()["presence"].(map[string]any)
	j2 := pres["judges"].([]map[string]any)[1]
	last, ok := j2["last_input"].(judgeAction)
	if !ok || last.ScoreType != "technique" || last.Target != "red" || j2["inputs"] != 1 {
		t.Fatalf("aktivitas juri 2 tidak tercatat: %+v", j2)
	}
	if find(msgs, hub.MsgPresence) == nil {
		t.Error("meja wasit harus menerima pembaruan kehadiran")
	}
}

func TestMatchEventsBroadcastToRefereeOnly(t *testing.T) {
	r := newRig(t)
	r.start(t)
	r.send(r.referee, hub.MsgStateChange, map[string]any{"action": "pause"})
	refMsgs := drain(r.refCh)

	ev := find(refMsgs, hub.MsgMatchEvent)
	if ev == nil {
		t.Fatal("meja wasit harus menerima match_event")
	}
	p := ev.Payload.(map[string]any)
	if p["event_type"] != "match_paused" {
		t.Errorf("event_type = %v", p["event_type"])
	}
	inner := p["payload"].(map[string]any)
	if _, ok := inner["clock_ms"]; !ok {
		t.Error("setiap kejadian harus membawa clock_ms")
	}
	if inner["round"] != 1 {
		t.Errorf("round = %v, want 1", inner["round"])
	}
	if find(drain(r.tvCh), hub.MsgMatchEvent) != nil {
		t.Error("penonton publik tidak boleh menerima log internal")
	}
	if find(drain(r.judgeCh), hub.MsgMatchEvent) != nil {
		t.Error("juri tidak boleh menerima log internal")
	}
}

// advanceTo memajukan mesin status ke ronde 3 (tanpa menunggu timer sungguhan).
func advanceToRound3(t *testing.T, r *testRig) {
	t.Helper()
	for _, ev := range []statemachine.Event{
		statemachine.EventRoundEnd, statemachine.EventBreakEnd,
		statemachine.EventRoundEnd, statemachine.EventBreakEnd,
	} {
		if err := r.o.sm.Transition(ev); err != nil {
			t.Fatalf("transisi %s: %v", ev, err)
		}
	}
	if r.o.sm.Current() != statemachine.StateRound3 {
		t.Fatalf("state = %s, want round_3", r.o.sm.Current())
	}
}

func TestFinish_TieRequiresRefereeDecision(t *testing.T) {
	r := newRig(t)
	r.start(t)

	// skor 0-0 = seri: selesai tanpa pemenang ditolak, partai tetap berjalan
	if err := r.o.Finish(); err != ErrTieNeedsDecision {
		t.Fatalf("seri tanpa keputusan: err = %v, want ErrTieNeedsDecision", err)
	}
	if r.o.sm.IsFinished() {
		t.Fatal("pertandingan tidak boleh berakhir tanpa keputusan saat seri")
	}

	if err := r.o.FinishWithWinner("ungu"); err == nil {
		t.Error("pemenang tidak valid harus ditolak")
	}

	if err := r.o.FinishWithWinner("blue"); err != nil {
		t.Fatalf("keputusan wasit: %v", err)
	}
	res := r.o.Snapshot()["result"].(map[string]any)
	if res["corner"] != "blue" || res["result_type"] != "referee_decision" {
		t.Errorf("hasil = %+v, want blue/referee_decision", res)
	}
}

func TestFinish_NotTiedNeedsNoDecision(t *testing.T) {
	r := newRig(t)
	r.start(t)
	r.send(r.referee, hub.MsgPenalty, map[string]any{"target": "blue", "type": "teguran_1"}) // merah unggul 1 poin
	if err := r.o.Finish(); err != nil {
		t.Fatalf("tidak seri harus bisa diselesaikan: %v", err)
	}
	res := r.o.Snapshot()["result"].(map[string]any)
	if res["corner"] != "red" || res["result_type"] != "point_win" {
		t.Errorf("hasil = %+v, want red/point_win", res)
	}
}

func TestRound3Expiry_TieWaitsForDecisionInsteadOfAwardingRed(t *testing.T) {
	r := newRig(t)
	r.start(t)
	advanceToRound3(t, r)
	r.o.timer.Reset(0)
	r.o.timer.Stop()
	drain(r.tvCh)

	r.o.onRoundExpired()

	if r.o.sm.IsFinished() {
		t.Fatal("seri saat waktu habis tidak boleh langsung memberi kemenangan")
	}
	snap := r.o.Snapshot()
	if snap["tie_pending"] != true {
		t.Fatalf("tie_pending = %v, want true", snap["tie_pending"])
	}
	st := find(drain(r.tvCh), hub.MsgStateChange)
	if st == nil || st.Payload.(map[string]any)["tie_pending"] != true {
		t.Errorf("state_change harus membawa tie_pending: %+v", st)
	}

	// wasit memutuskan -> selesai
	if err := r.o.FinishWithWinner("red"); err != nil {
		t.Fatal(err)
	}
	if !r.o.sm.IsFinished() {
		t.Error("harus selesai setelah keputusan wasit")
	}
	if r.o.Snapshot()["tie_pending"] != false {
		t.Error("tie_pending harus padam setelah selesai")
	}
}

func TestRound3Expiry_NotTiedFinishesAutomatically(t *testing.T) {
	r := newRig(t)
	r.start(t)
	r.send(r.referee, hub.MsgPenalty, map[string]any{"target": "red", "type": "teguran_1"}) // biru unggul
	advanceToRound3(t, r)
	r.o.timer.Reset(0)
	r.o.timer.Stop()

	r.o.onRoundExpired()

	if !r.o.sm.IsFinished() {
		t.Fatal("tidak seri: pertandingan harus selesai otomatis")
	}
	res := r.o.Snapshot()["result"].(map[string]any)
	if res["corner"] != "blue" || res["result_type"] != "point_win" {
		t.Errorf("hasil = %+v", res)
	}
}
