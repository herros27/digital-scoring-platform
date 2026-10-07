package orchestrator

import (
	"fmt"
	"time"

	"github.com/herros27/digital-scoring/internal/hub"
	"github.com/herros27/digital-scoring/internal/rules"
	"github.com/herros27/digital-scoring/internal/statemachine"
	"github.com/herros27/digital-scoring/internal/store"
)

// penaltyEntry satu sanksi yang sudah dijatuhkan pada pertandingan ini.
type penaltyEntry struct {
	ID      int    `json:"id"`
	Round   int    `json:"round"`
	Target  string `json:"target"`
	Type    string `json:"type"`
	Label   string `json:"label"`
	Points  int    `json:"points"`
	ClockMS int64  `json:"clock_ms"`
}

// judgeAction input juri terakhir (untuk panel verifikasi juri).
type judgeAction struct {
	ScoreType string `json:"score_type"`
	Target    string `json:"target"`
	Round     int    `json:"round"`
	ClockMS   int64  `json:"clock_ms"`
	At        string `json:"at"`
}

// forcedResult hasil yang ditentukan wasit (diskualifikasi) dan menimpa perhitungan skor.
type forcedResult struct {
	Corner     string
	ResultType string
}

const (
	maxJudgeSeats = 5
	minRoundClock = time.Second
)

// ErrTieNeedsDecision: skor seri, wasit harus menentukan pemenang (keputusan wasit).
var ErrTieNeedsDecision = fmt.Errorf("skor seri: tentukan pemenang sebagai keputusan wasit")

// tiedLocked true bila total skor bersih kedua sudut sama. Pemanggil memegang o.mu.
func (o *Orchestrator) tiedLocked() bool {
	var red, blue int
	for i := 1; i <= 3; i++ {
		red += o.scores[i].RedTechnical - o.scores[i].RedPenalties
		blue += o.scores[i].BlueTechnical - o.scores[i].BluePenalties
	}
	return red == blue
}

// tiePendingLocked true saat waktu ronde terakhir habis dengan skor seri dan menunggu keputusan wasit.
func (o *Orchestrator) tiePendingLocked() bool {
	if !o.isFinalRoundLocked() ||
		o.timer == nil || o.timer.Remaining() > 0 || o.timer.IsRunning() || !o.tiedLocked() {
		return false
	}
	_, resolvable := o.resolveTiebreakLocked()
	return !resolvable
}

func opposite(target string) string {
	if target == "red" {
		return "blue"
	}
	return "red"
}

func validTarget(t string) bool { return t == "red" || t == "blue" }

// canControl: hanya referee dan admin yang boleh mengendalikan pertandingan.
func canControl(c *hub.Client) bool {
	return c.Role == hub.RoleReferee || c.Role == hub.RoleAdmin
}

// clockMS sisa waktu timer saat ini dalam ms. Pemanggil memegang o.mu.
func (o *Orchestrator) clockMS() int64 {
	if o.timer == nil {
		return 0
	}
	return o.timer.Remaining().Milliseconds()
}

// persistRound menyimpan skor satu ronde ke database. Pemanggil memegang o.mu.
func (o *Orchestrator) persistRound(round int) {
	if o.scoreStore == nil {
		return
	}
	_ = o.scoreStore.Upsert(&store.MatchScore{
		MatchID:          o.matchID,
		Round:            round,
		AthleteRedScore:  o.scores[round].RedTechnical,
		AthleteBlueScore: o.scores[round].BlueTechnical,
		RedPenalties:     o.scores[round].RedPenalties,
		BluePenalties:    o.scores[round].BluePenalties,
	})
}

func (o *Orchestrator) addPenaltyPoints(round int, target string, delta int) {
	if target == "red" {
		o.scores[round].RedPenalties += delta
	} else {
		o.scores[round].BluePenalties += delta
	}
}

// penaltiesLocked salinan daftar sanksi (tidak pernah nil agar JSON berupa []).
func (o *Orchestrator) penaltiesLocked() []penaltyEntry {
	out := make([]penaltyEntry, len(o.penalties))
	copy(out, o.penalties)
	return out
}

func (o *Orchestrator) broadcastPenalties() {
	defer o.persistLocked()
	o.h.Broadcast(hub.Message{
		Type:    hub.MsgPenaltyUpdate,
		Payload: map[string]any{"penalties": o.penaltiesLocked()},
	})
}

// handlePenalty: {"target":"red","type":"teguran_1"}. Server menentukan poin dari katalog.
func (o *Orchestrator) handlePenalty(client *hub.Client, msg hub.Message) {
	if !canControl(client) {
		o.sendError(client, "FORBIDDEN", "only referee can issue penalty")
		return
	}
	var p struct {
		Target string `json:"target"`
		Type   string `json:"type"`
	}
	if err := decodePayload(msg.Payload, &p); err != nil {
		o.sendError(client, "BAD_PAYLOAD", err.Error())
		return
	}
	kind, ok := o.ruleset.Scoring.Penalty(p.Type)
	if !validTarget(p.Target) || !ok {
		o.sendError(client, "BAD_PAYLOAD", "target harus red/blue dan type harus salah satu jenis sanksi resmi")
		return
	}

	o.mu.Lock()
	defer o.mu.Unlock()

	round := o.sm.CurrentRoundNumber()
	if round == 0 || o.sm.IsFinished() {
		o.sendError(client, "PENALTY_NOT_ALLOWED", "sanksi hanya dapat dijatuhkan saat pertandingan berlangsung")
		return
	}
	for _, e := range o.penalties {
		if e.Target == p.Target && e.Type == p.Type {
			o.sendError(client, "PENALTY_ALREADY_APPLIED", kind.Label+" sudah dijatuhkan pada sudut ini")
			return
		}
	}

	o.nextPenaltyID++
	entry := penaltyEntry{
		ID: o.nextPenaltyID, Round: round, Target: p.Target,
		Type: kind.Type, Label: kind.Label, Points: kind.Points, ClockMS: o.clockMS(),
	}
	o.penalties = append(o.penalties, entry)
	o.addPenaltyPoints(round, p.Target, kind.Points)
	o.persistRound(round)

	o.appendEvent("penalty_applied", map[string]any{
		"id": entry.ID, "target": p.Target, "type": kind.Type, "label": kind.Label,
		"points": kind.Points, "round": round, "by": client.ID,
	}, client.ID)
	o.broadcastScoreUpdate(round)
	o.broadcastPenalties()

	if kind.Action == rules.ActionDisqualify {
		o.forced = &forcedResult{Corner: opposite(p.Target), ResultType: "disqualification"}
		o.appendEvent("disqualified", map[string]any{
			"target": p.Target, "winner_corner": o.forced.Corner, "reason": kind.Type,
		}, client.ID)
		o.finalize()
	}
}

// handlePenaltyRevoke: {"target":"red"} membatalkan sanksi TERAKHIR pada sudut itu.
func (o *Orchestrator) handlePenaltyRevoke(client *hub.Client, msg hub.Message) {
	if !canControl(client) {
		o.sendError(client, "FORBIDDEN", "only referee can revoke penalty")
		return
	}
	var p struct {
		Target string `json:"target"`
	}
	if err := decodePayload(msg.Payload, &p); err != nil || !validTarget(p.Target) {
		o.sendError(client, "BAD_PAYLOAD", "target harus red atau blue")
		return
	}

	o.mu.Lock()
	defer o.mu.Unlock()

	if o.sm.IsFinished() {
		o.sendError(client, "PENALTY_NOT_ALLOWED", "pertandingan sudah selesai")
		return
	}
	idx := -1
	for i := len(o.penalties) - 1; i >= 0; i-- {
		if o.penalties[i].Target == p.Target {
			idx = i
			break
		}
	}
	if idx < 0 {
		o.sendError(client, "NOTHING_TO_REVOKE", "tidak ada sanksi yang dapat dibatalkan pada sudut ini")
		return
	}
	entry := o.penalties[idx]
	o.penalties = append(o.penalties[:idx], o.penalties[idx+1:]...)
	o.addPenaltyPoints(entry.Round, entry.Target, -entry.Points)
	o.persistRound(entry.Round)

	o.appendEvent("penalty_revoked", map[string]any{
		"id": entry.ID, "target": entry.Target, "type": entry.Type, "label": entry.Label,
		"points": entry.Points, "round": entry.Round, "by": client.ID,
	}, client.ID)
	o.broadcastScoreUpdate(entry.Round)
	o.broadcastPenalties()
}

// handleArenaSignal: {"signal":"gong"} atau {"signal":"medic"|"var","active":true|false}.
func (o *Orchestrator) handleArenaSignal(client *hub.Client, msg hub.Message) {
	if !canControl(client) {
		o.sendError(client, "FORBIDDEN", "only referee can send arena signals")
		return
	}
	var p struct {
		Signal string `json:"signal"`
		Active *bool  `json:"active"`
	}
	if err := decodePayload(msg.Payload, &p); err != nil {
		o.sendError(client, "BAD_PAYLOAD", err.Error())
		return
	}

	o.mu.Lock()
	defer o.mu.Unlock()

	active := true
	switch p.Signal {
	case "gong":
		// sinyal sesaat, tanpa status
	case "medic", "var":
		if p.Active == nil {
			o.sendError(client, "BAD_PAYLOAD", "active wajib diisi untuk sinyal medic/var")
			return
		}
		active = *p.Active
		if p.Signal == "medic" {
			o.medicActive = active
		} else {
			o.varActive = active
		}
	default:
		o.sendError(client, "BAD_PAYLOAD", "signal harus gong, medic, atau var")
		return
	}

	o.persistLocked()
	o.appendEvent("arena_signal", map[string]any{"signal": p.Signal, "active": active, "by": client.ID}, client.ID)
	o.h.Broadcast(hub.Message{
		Type:    hub.MsgArenaSignal,
		Payload: map[string]any{"signal": p.Signal, "active": active},
	})
}

// doDisqualify mengakhiri pertandingan: sudut lawan menang dengan result_type "disqualification".
func (o *Orchestrator) doDisqualify(target string) error {
	if !validTarget(target) {
		return fmt.Errorf("target harus red atau blue")
	}
	o.mu.Lock()
	defer o.mu.Unlock()

	if o.sm.IsFinished() {
		return fmt.Errorf("match is already finished")
	}
	if o.sm.CurrentRoundNumber() == 0 {
		return fmt.Errorf("pertandingan belum dimulai")
	}
	o.forced = &forcedResult{Corner: opposite(target), ResultType: "disqualification"}
	o.appendEvent("disqualified", map[string]any{"target": target, "winner_corner": o.forced.Corner}, "")
	o.finalize()
	return nil
}

// maxNoteLen panjang maksimum catatan wasit pada hasil khusus.
const maxNoteLen = 200

// doDeclareOutcome menyatakan hasil khusus (ko/tko/walkover/withdrawal): `winner` menang dengan
// result_type = jenis hasil. Kapan boleh dinyatakan diatur `phase` pada ruleset.
func (o *Orchestrator) doDeclareOutcome(outcome, winner, note string) error {
	if !validTarget(winner) {
		return fmt.Errorf("pemenang harus red atau blue")
	}
	if len([]rune(note)) > maxNoteLen {
		return fmt.Errorf("catatan maksimal %d karakter", maxNoteLen)
	}
	o.mu.Lock()
	defer o.mu.Unlock()

	rule, ok := o.ruleset.Scoring.Outcome(outcome)
	if !ok {
		return fmt.Errorf("jenis hasil %q tidak tersedia pada ruleset ini", outcome)
	}
	if o.sm.IsFinished() {
		return fmt.Errorf("match is already finished")
	}
	started := o.sm.CurrentRoundNumber() > 0
	switch {
	case rule.Phase == rules.PhaseBeforeStart && started:
		return fmt.Errorf("%s hanya dapat dinyatakan sebelum pertandingan dimulai", rule.Label)
	case rule.Phase == rules.PhaseInMatch && !started:
		return fmt.Errorf("%s hanya dapat dinyatakan setelah pertandingan dimulai", rule.Label)
	}

	// partai yang baru dimuat masih idle: majukan ke ready agar dapat diselesaikan
	if o.sm.Current() == statemachine.StateIdle {
		if err := o.sm.Transition(statemachine.EventStart); err != nil {
			return err
		}
	}
	o.forced = &forcedResult{Corner: winner, ResultType: rule.Type}
	o.appendEvent("outcome_declared", map[string]any{
		"outcome": rule.Type, "label": rule.Label, "winner_corner": winner, "note": note,
	}, "")
	o.finalize()
	return nil
}

// doAdjustTime koreksi waktu ronde (+/- ms) saat pertandingan berada di dalam ronde.
func (o *Orchestrator) doAdjustTime(deltaMS int64) error {
	o.mu.Lock()
	defer o.mu.Unlock()

	if !o.sm.IsInRound() || o.timer == nil {
		return fmt.Errorf("waktu hanya dapat dikoreksi saat berada di dalam ronde")
	}
	max := time.Duration(o.ruleset.Scoring.RoundDuration) * time.Second
	remaining := o.timer.Adjust(time.Duration(deltaMS)*time.Millisecond, minRoundClock, max)
	o.appendEvent("time_adjusted", map[string]any{"delta_ms": deltaMS, "remaining_ms": remaining.Milliseconds()}, "")
	o.broadcastState()
	return nil
}

// doResetRound mengembalikan sisa waktu ronde berjalan ke durasi penuh. Hanya saat dijeda.
func (o *Orchestrator) doResetRound() error {
	o.mu.Lock()
	defer o.mu.Unlock()

	if !o.sm.IsInRound() || o.timer == nil {
		return fmt.Errorf("ronde hanya dapat di-reset saat berada di dalam ronde")
	}
	if o.timer.IsRunning() {
		return fmt.Errorf("jeda pertandingan terlebih dahulu sebelum me-reset ronde")
	}
	o.timer.Reset(time.Duration(o.ruleset.Scoring.RoundDuration) * time.Second)
	o.appendEvent("round_reset", map[string]any{"round": o.sm.CurrentRoundNumber()}, "")
	o.broadcastState()
	return nil
}

// recordJudgeInput mencatat input juri terakhir per kursi. Pemanggil memegang o.mu.
func (o *Orchestrator) recordJudgeInput(seat int, scoreType, target string, round int) {
	o.lastInput[seat] = judgeAction{
		ScoreType: scoreType, Target: target, Round: round,
		ClockMS: o.clockMS(), At: time.Now().UTC().Format(time.RFC3339),
	}
	o.inputCount[seat]++
}

// presenceLocked kehadiran perangkat + aktivitas juri. Pemanggil memegang o.mu.
func (o *Orchestrator) presenceLocked() map[string]any {
	pr := o.h.Presence()
	judges := make([]map[string]any, 0, maxJudgeSeats)
	for seat := 1; seat <= maxJudgeSeats; seat++ {
		j := map[string]any{
			"seat":       seat,
			"connected":  pr.JudgeSeats[seat],
			"inputs":     o.inputCount[seat],
			"last_input": nil,
		}
		if a, ok := o.lastInput[seat]; ok {
			j["last_input"] = a
		}
		judges = append(judges, j)
	}
	return map[string]any{
		"judges":      judges,
		"referees":    pr.Referees,
		"admins":      pr.Admins,
		"scoreboards": pr.Scoreboards,
	}
}

// broadcastPresence memberi tahu meja wasit/admin saat perangkat terhubung atau terputus.
func (o *Orchestrator) broadcastPresence() {
	o.mu.Lock()
	payload := o.presenceLocked()
	o.mu.Unlock()
	msg := hub.Message{Type: hub.MsgPresence, Payload: payload}
	o.h.SendToRole(hub.RoleReferee, msg)
	o.h.SendToRole(hub.RoleAdmin, msg)
}

// extraSnapshotLocked bidang tambahan snapshot untuk konsol wasit. Pemanggil memegang o.mu.
func (o *Orchestrator) extraSnapshotLocked() map[string]any {
	return map[string]any{
		"ruleset_name":      o.ruleset.Name,
		"round_duration_ms": int64(o.ruleset.Scoring.RoundDuration) * 1000,
		"break_duration_ms": int64(o.ruleset.Scoring.BreakDuration) * 1000,
		"total_rounds":      o.ruleset.Scoring.TotalRounds,
		"penalties":         o.penaltiesLocked(),
		"penalty_catalog":   o.ruleset.Scoring.EffectivePenalties(),
		"techniques":        o.ruleset.Scoring.EffectiveTechniques(),
		"judge_count":       o.ruleset.Scoring.EffectiveConsensus().Judges,
		"consensus":         o.ruleset.Scoring.EffectiveConsensus(),
		"tiebreak":          o.tiebreakSnapshotLocked(),
		"recovered":         o.recovered,
		"outcomes":          o.ruleset.Scoring.EffectiveOutcomes(),
		"medic_active":      o.medicActive,
		"var_active":        o.varActive,
		"presence":          o.presenceLocked(),
		"tie_pending":       o.tiePendingLocked(),
		"result":            o.resultLocked(),
	}
}

// resultLocked hasil akhir bila pertandingan sudah selesai, selain itu nil.
func (o *Orchestrator) resultLocked() any {
	if o.result == nil {
		return nil
	}
	return o.result
}
