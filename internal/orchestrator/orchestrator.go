package orchestrator

import (
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/herros27/digital-scoring/internal/consensus"
	"github.com/herros27/digital-scoring/internal/hub"
	"github.com/herros27/digital-scoring/internal/rules"
	"github.com/herros27/digital-scoring/internal/statemachine"
	"github.com/herros27/digital-scoring/internal/store"
)

// Scores disimpan in-memory, di-persist ke DB setiap ada perubahan
type RoundScore struct {
	RedTechnical  int
	BlueTechnical int
	RedPenalties  int
	BluePenalties int
}

type BracketAdvancer interface {
	AdvanceWinner(slotID, winnerID string) error
}

type MatchStore interface {
	UpdateStatus(id, status string) error
	SetWinner(matchID, winnerID string) error
	GetByID(id string) (*store.Match, error)
}

type Orchestrator struct {
	mu      sync.Mutex
	matchID string
	arenaID string

	sm      *statemachine.StateMachine
	ruleset *rules.Ruleset
	engine  *consensus.Engine
	h       *hub.Hub

	matchStore   MatchStore
	eventStore   *store.MatchEventStore
	scoreStore   *store.MatchScoreStore
	bracketStore BracketAdvancer

	// skor per ronde: index = round number (1-based, [0] tidak dipakai)
	scores     [4]RoundScore
	timer      *Timer
	seenEvents map[string]time.Time

	// state meja wasit (lihat referee_ops.go)
	penalties     []penaltyEntry
	nextPenaltyID int
	medicActive   bool
	varActive     bool
	lastInput     map[int]judgeAction
	inputCount    map[int]int
	forced        *forcedResult
	result        map[string]any
}

func New(
	matchID, arenaID string,
	sm *statemachine.StateMachine,
	ruleset *rules.Ruleset,
	h *hub.Hub,
	matchStore MatchStore,
	eventStore *store.MatchEventStore,
	scoreStore *store.MatchScoreStore,
	bracketStore BracketAdvancer,
) *Orchestrator {
	o := &Orchestrator{
		matchID:      matchID,
		arenaID:      arenaID,
		sm:           sm,
		ruleset:      ruleset,
		engine:       consensus.NewEngine(),
		h:            h,
		matchStore:   matchStore,
		eventStore:   eventStore,
		scoreStore:   scoreStore,
		bracketStore: bracketStore,
		seenEvents:   make(map[string]time.Time),
		lastInput:    make(map[int]judgeAction),
		inputCount:   make(map[int]int),
	}
	// Pasang callback: setiap message dari hub masuk ke sini
	h.Attach(o, o.handleMessage, o.broadcastPresence)
	return o
}

// Release melepas orchestrator dari hub arena (dipakai saat partai dijadwalkan ulang).
func (o *Orchestrator) Release() {
	o.h.Release(o)
}

func decodePayload(payload interface{}, dest interface{}) error {
	switch v := payload.(type) {
	case []byte:
		return json.Unmarshal(v, dest)
	case map[string]interface{}:
		data, err := json.Marshal(v)
		if err != nil {
			return err
		}
		return json.Unmarshal(data, dest)
	default:
		return fmt.Errorf("unsupported payload type: %T", payload)
	}
}

func (o *Orchestrator) handleMessage(arenaID string, client *hub.Client, msg hub.Message) {
	switch msg.Type {
	case hub.MsgJudgeInput:
		o.handleJudgeInput(client, msg)
	case hub.MsgStateChange:
		o.handleControl(client, msg)
	case hub.MsgPenalty:
		o.handlePenalty(client, msg)
	case hub.MsgPenaltyRevoke:
		o.handlePenaltyRevoke(client, msg)
	case hub.MsgArenaSignal:
		o.handleArenaSignal(client, msg)
	case hub.MsgSyncRequest:
		o.handleSyncRequest(client)
	default:
		log.Printf("[orch:%s] unknown message type: %s", o.matchID, msg.Type)
	}
}

// payload untuk MsgStateChange: {"action": "start"} / "pause" / "resume"
type controlPayload struct {
	Action  string `json:"action"`
	Winner  string `json:"winner,omitempty"`   // finish: pemenang keputusan wasit (wajib bila seri)
	Target  string `json:"target,omitempty"`   // disqualify: sudut yang didiskualifikasi
	DeltaMS int64  `json:"delta_ms,omitempty"` // adjust_time: koreksi waktu (+/-)
}

func (o *Orchestrator) handleControl(client *hub.Client, msg hub.Message) {
	// hanya referee & admin boleh kontrol
	if client.Role != hub.RoleReferee && client.Role != hub.RoleAdmin {
		o.sendError(client, "FORBIDDEN", "only referee can control match")
		return
	}

	var p controlPayload
	if err := decodePayload(msg.Payload, &p); err != nil {
		o.sendError(client, "BAD_PAYLOAD", err.Error())
		return
	}

	switch p.Action {
	case "start":
		if err := o.doStart(); err != nil {
			o.sendError(client, "INVALID_STATE_TRANSITION", err.Error())
		}
	case "pause":
		if err := o.doPause(); err != nil {
			o.sendError(client, "INVALID_STATE_TRANSITION", err.Error())
		}
	case "resume":
		if err := o.doResume(); err != nil {
			o.sendError(client, "INVALID_STATE_TRANSITION", err.Error())
		}
	case "finish":
		if err := o.doFinish(p.Winner); err != nil {
			o.sendError(client, "INVALID_STATE_TRANSITION", err.Error())
		}
	case "cancel":
		if err := o.doCancel(); err != nil {
			o.sendError(client, "INVALID_STATE_TRANSITION", err.Error())
		}
	case "disqualify":
		if err := o.doDisqualify(p.Target); err != nil {
			o.sendError(client, "INVALID_STATE_TRANSITION", err.Error())
		}
	case "adjust_time":
		if err := o.doAdjustTime(p.DeltaMS); err != nil {
			o.sendError(client, "INVALID_STATE_TRANSITION", err.Error())
		}
	case "reset_round":
		if err := o.doResetRound(); err != nil {
			o.sendError(client, "INVALID_STATE_TRANSITION", err.Error())
		}
	default:
		o.sendError(client, "UNKNOWN_ACTION", p.Action)
	}
}

func (o *Orchestrator) doStart() error {
	o.mu.Lock()
	defer o.mu.Unlock()

	cur := o.sm.Current()
	if cur == statemachine.StateFinished || cur == statemachine.StateCancelled {
		return fmt.Errorf("cannot start match in %s state", cur)
	}
	if cur == statemachine.StateIdle {
		if err := o.sm.Transition(statemachine.EventStart); err != nil {
			log.Printf("[orch:%s] transition error: %v", o.matchID, err)
			return err
		}
	}
	if err := o.sm.Transition(statemachine.EventStart); err != nil {
		log.Printf("[orch:%s] transition error: %v", o.matchID, err)
		return err
	}

	o.appendEvent("match_started", map[string]any{"round": o.sm.CurrentRoundNumber()}, "")
	o.startRoundTimer()
	o.broadcastState()
	return nil
}

func (o *Orchestrator) doPause() error {
	o.mu.Lock()
	defer o.mu.Unlock()

	if !o.sm.IsInRound() {
		return fmt.Errorf("cannot pause match in %s state", o.sm.Current())
	}
	if o.timer != nil {
		o.timer.Pause()
	}
	var rem int64
	if o.timer != nil {
		rem = o.timer.Remaining().Milliseconds()
	}
	o.appendEvent("match_paused", map[string]any{
		"remaining_ms": rem,
	}, "")
	o.broadcastState()
	return nil
}

func (o *Orchestrator) doResume() error {
	o.mu.Lock()
	defer o.mu.Unlock()

	if !o.sm.IsInRound() {
		return fmt.Errorf("cannot resume match in %s state", o.sm.Current())
	}
	if o.timer != nil {
		o.timer.Start()
	}
	o.appendEvent("match_resumed", nil, "")
	o.broadcastState()
	return nil
}

// doFinish mengakhiri pertandingan. Bila skor seri, wasit wajib menentukan pemenang
// (`winner` = "red"/"blue", keputusan wasit); tanpa itu pertandingan tidak diakhiri.
func (o *Orchestrator) doFinish(winner string) error {
	o.mu.Lock()
	defer o.mu.Unlock()

	if o.sm.IsFinished() {
		return fmt.Errorf("match is already finished")
	}
	if winner != "" {
		if !validTarget(winner) {
			return fmt.Errorf("pemenang harus red atau blue")
		}
		o.forced = &forcedResult{Corner: winner, ResultType: "referee_decision"}
		o.appendEvent("referee_decision", map[string]any{"winner_corner": winner}, "")
	} else if o.sm.CurrentRoundNumber() > 0 && o.tiedLocked() {
		return ErrTieNeedsDecision
	}
	o.finalize()
	return nil
}

func (o *Orchestrator) doCancel() error {
	o.mu.Lock()
	defer o.mu.Unlock()

	if o.sm.IsFinished() {
		return fmt.Errorf("cannot cancel finished match")
	}
	if o.timer != nil {
		o.timer.Stop()
	}
	if err := o.sm.Transition(statemachine.EventCancel); err != nil {
		log.Printf("[orch:%s] cancel transition error: %v", o.matchID, err)
		return err
	}
	if o.matchStore != nil {
		_ = o.matchStore.UpdateStatus(o.matchID, string(statemachine.StateCancelled))
	}
	o.appendEvent("match_cancelled", nil, "")
	o.broadcastState()
	return nil
}

// payload untuk MsgJudgeInput: {"event_id": "...", "score_type": "technique", "target": "red", "round": 1}
type judgeInputPayload struct {
	EventID        string `json:"event_id,omitempty"`
	IdempotencyKey string `json:"idempotency_key,omitempty"`
	ScoreType      string `json:"score_type"`
	Target         string `json:"target"`
	Round          int    `json:"round"`
}

func (o *Orchestrator) handleJudgeInput(client *hub.Client, msg hub.Message) {
	// hanya judge yang boleh submit skor
	// if client.Role != hub.RoleJudge {
	// 	o.sendError(client, "FORBIDDEN", "only judges can submit scores")
	// 	return
	// }

	if !client.Role.IsJudge() {
		o.sendError(client, "FORBIDDEN", "only judges can submit scores")
		return
	}

	// scoring hanya boleh di state ROUND
	if !o.sm.IsInRound() {
		o.sendError(client, "SCORING_NOT_ALLOWED", fmt.Sprintf("state is %s", o.sm.Current()))
		return
	}

	var p judgeInputPayload
	if err := decodePayload(msg.Payload, &p); err != nil {
		o.sendError(client, "BAD_PAYLOAD", err.Error())
		return
	}

	key := p.EventID
	if key == "" {
		key = p.IdempotencyKey
	}
	if key != "" {
		o.mu.Lock()
		if _, exists := o.seenEvents[key]; exists {
			o.mu.Unlock()
			log.Printf("[orch:%s] duplicate score event ignored by idempotency key: %s", o.matchID, key)
			return
		}
		o.seenEvents[key] = time.Now()
		if len(o.seenEvents) > 500 {
			cutoff := time.Now().Add(-5 * time.Minute)
			for k, t := range o.seenEvents {
				if t.Before(cutoff) {
					delete(o.seenEvents, k)
				}
			}
		}
		o.mu.Unlock()
	}

	input := consensus.JudgeInput{
		JudgeID:    client.ID,
		Role:       consensus.JudgeRole(client.Role),
		SeatNumber: client.SeatNumber,
		MatchID:    o.matchID,
		Round:      o.sm.CurrentRoundNumber(),
		ScoreType:  consensus.ScoreType(p.ScoreType),
		Target:     consensus.Target(p.Target),
		Timestamp:  time.Now(),
	}

	// log raw judge input ke DB (audit trail) dan catat aktivitas kursi untuk panel juri
	o.mu.Lock()
	o.recordJudgeInput(client.SeatNumber, string(input.ScoreType), string(input.Target), input.Round)
	o.appendEvent("judge_input", map[string]any{
		"judge_id":    input.JudgeID,
		"seat_number": client.SeatNumber,
		"role":        client.Role,
		"score_type":  input.ScoreType,
		"target":      input.Target,
		"round":       input.Round,
	}, client.ID)
	o.mu.Unlock()
	o.broadcastPresence()

	result := o.engine.Submit(input)
	if result == nil || !result.Accepted {
		// belum majority, tidak apa-apa — tunggu juri lain
		return
	}

	// Consensus tercapai → hitung poin
	o.applyScore(result)
}

// SubmitJudgeInput memungkinkan pengiriman scoring event via REST dengan idempotency key check
func (o *Orchestrator) SubmitJudgeInput(input consensus.JudgeInput, idempotencyKey string) (*consensus.ConsensusResult, bool, error) {
	if !o.sm.IsInRound() {
		return nil, false, fmt.Errorf("scoring not allowed in state %s", o.sm.Current())
	}

	if idempotencyKey != "" {
		o.mu.Lock()
		if _, exists := o.seenEvents[idempotencyKey]; exists {
			o.mu.Unlock()
			return nil, true, nil
		}
		o.seenEvents[idempotencyKey] = time.Now()
		o.mu.Unlock()
	}

	input.MatchID = o.matchID
	input.Round = o.sm.CurrentRoundNumber()
	input.Timestamp = time.Now()

	o.mu.Lock()
	o.appendEvent("judge_input", map[string]any{
		"judge_id":   input.JudgeID,
		"score_type": input.ScoreType,
		"target":     input.Target,
		"round":      input.Round,
	}, input.JudgeID)
	o.mu.Unlock()

	result := o.engine.Submit(input)
	if result != nil && result.Accepted {
		o.applyScore(result)
	}

	return result, false, nil
}

func (o *Orchestrator) applyScore(result *consensus.ConsensusResult) {
	o.mu.Lock()
	defer o.mu.Unlock()

	round := result.Round
	pts := o.pointsFor(result.ScoreType)

	if result.Target == consensus.TargetRed {
		o.scores[round].RedTechnical += pts
	} else {
		o.scores[round].BlueTechnical += pts
	}

	// persist ke DB
	sc := &store.MatchScore{
		MatchID:          o.matchID,
		Round:            round,
		AthleteRedScore:  o.scores[round].RedTechnical,
		AthleteBlueScore: o.scores[round].BlueTechnical,
		RedPenalties:     o.scores[round].RedPenalties,
		BluePenalties:    o.scores[round].BluePenalties,
	}
	if o.scoreStore != nil {
		if err := o.scoreStore.Upsert(sc); err != nil {
			log.Printf("[orch:%s] score upsert error: %v", o.matchID, err)
		}
	}

	confirmed := map[string]any{
		"score_type": result.ScoreType,
		"target":     result.Target,
		"points":     pts,
		"count":      result.Count,
		"round":      round,
	}
	o.appendEvent("score_confirmed", confirmed, "")
	o.h.Broadcast(hub.Message{Type: hub.MsgScoreConfirmed, Payload: confirmed})

	o.broadcastScoreUpdate(round)

	// cek win condition (KO, point difference)
	o.checkMidMatchWin(round)
}

func (o *Orchestrator) pointsFor(st consensus.ScoreType) int {
	switch st {
	case consensus.ScoreTechnique:
		return o.ruleset.Scoring.TechniquePoints
	case consensus.ScoreKnockdown:
		return o.ruleset.Scoring.KnockdownPoints
	case consensus.ScoreFall:
		return o.ruleset.Scoring.FallPoints
	default:
		return 0
	}
}

func (o *Orchestrator) handleSyncRequest(client *hub.Client) {
	o.mu.Lock()
	payload := o.snapshotLocked()
	o.mu.Unlock()

	o.h.SendToClient(client.ID, hub.Message{
		Type:    hub.MsgSyncResponse,
		Payload: payload,
	})
}

// Snapshot mengembalikan keadaan pertandingan saat ini (dipakai sync_response dan REST scoreboard).
func (o *Orchestrator) Snapshot() map[string]any {
	o.mu.Lock()
	defer o.mu.Unlock()
	return o.snapshotLocked()
}

// snapshotLocked: pemanggil harus memegang o.mu.
func (o *Orchestrator) snapshotLocked() map[string]any {
	var redTotal, blueTotal int
	for i := 1; i <= 3; i++ {
		redTotal += o.scores[i].RedTechnical - o.scores[i].RedPenalties
		blueTotal += o.scores[i].BlueTechnical - o.scores[i].BluePenalties
	}

	snap := map[string]any{
		"match_id":    o.matchID,
		"arena_id":    o.arenaID,
		"state":       string(o.sm.Current()),
		"round":       o.sm.CurrentRoundNumber(),
		"scores":      o.scores[1:4],
		"red_total":   redTotal,
		"blue_total":  blueTotal,
		"server_time": time.Now().UTC().Format(time.RFC3339),
	}
	for k, v := range timerPayload(o.timer) {
		snap[k] = v
	}
	for k, v := range o.extraSnapshotLocked() {
		snap[k] = v
	}
	return snap
}

func (o *Orchestrator) startRoundTimer() {
	duration := time.Duration(o.ruleset.Scoring.RoundDuration) * time.Second
	o.timer = o.newTimer(duration, o.onRoundExpired)
	o.timer.Start()
}

func (o *Orchestrator) onRoundExpired() {
	o.mu.Lock()
	defer o.mu.Unlock()

	cur := o.sm.Current()

	// Ronde 3 selesai → langsung finish, kecuali skor seri: pertandingan menunggu keputusan wasit
	if cur == statemachine.StateRound3 {
		if o.tiedLocked() {
			o.appendEvent("tie_pending", map[string]any{"round": 3}, "")
			o.broadcastState()
			return
		}
		o.sm.Transition(statemachine.EventFinish)
		o.appendEvent("round_ended", map[string]any{"round": 3}, "")
		o.finalize()
		return
	}

	// Ronde 1 atau 2 selesai → masuk break
	roundNum := o.sm.CurrentRoundNumber()
	o.sm.Transition(statemachine.EventRoundEnd)
	o.appendEvent("round_ended", map[string]any{"round": roundNum}, "")
	o.broadcastState()

	// Setelah break, mulai ronde berikutnya otomatis
	breakDur := time.Duration(o.ruleset.Scoring.BreakDuration) * time.Second
	o.timer = o.newTimer(breakDur, o.onBreakExpired)
	o.timer.Start()
}

func (o *Orchestrator) onBreakExpired() {
	o.mu.Lock()
	defer o.mu.Unlock()

	o.sm.Transition(statemachine.EventBreakEnd)
	o.appendEvent("round_started", map[string]any{"round": o.sm.CurrentRoundNumber()}, "")
	o.startRoundTimer()
	o.broadcastState()
}

// Cek KO / point difference di tengah pertandingan
func (o *Orchestrator) checkMidMatchWin(round int) {
	sc := o.scores[round]
	// point difference threshold (hardcode 30, nanti ambil dari ruleset kalau ada)
	redTotal := sc.RedTechnical - sc.RedPenalties
	blueTotal := sc.BlueTechnical - sc.BluePenalties
	diff := redTotal - blueTotal
	if diff < 0 {
		diff = -diff
	}
	if diff >= 30 {
		o.sm.Transition(statemachine.EventFinish)
		o.finalize()
	}
}

// finalize dipanggil saat match benar-benar selesai
func (o *Orchestrator) finalize() {
	if o.timer != nil {
		o.timer.Stop()
	}

	if !o.sm.IsFinished() {
		_ = o.sm.Transition(statemachine.EventFinish)
	}

	winnerCorner, resultType := o.determineWinner()
	if o.forced != nil {
		winnerCorner, resultType = o.forced.Corner, o.forced.ResultType
	}
	winnerID := winnerCorner

	if o.matchStore != nil {
		_ = o.matchStore.UpdateStatus(o.matchID, string(statemachine.StateFinished))
		m, err := o.matchStore.GetByID(o.matchID)
		if err == nil && m != nil {
			if winnerCorner == "red" && m.AthleteRedID != "" {
				winnerID = m.AthleteRedID
			} else if winnerCorner == "blue" && m.AthleteBlueID != "" {
				winnerID = m.AthleteBlueID
			}
			_ = o.matchStore.SetWinner(o.matchID, winnerID)

			// Otomatisasi Bracket Advancement: majukan pemenang ke slot berikutnya
			if m.BracketSlotID != "" && o.bracketStore != nil {
				if err := o.bracketStore.AdvanceWinner(m.BracketSlotID, winnerID); err != nil {
					log.Printf("[orch:%s] failed to advance bracket winner: %v", o.matchID, err)
				}
			}
		} else {
			_ = o.matchStore.SetWinner(o.matchID, winnerID)
		}
	}

	o.appendEvent("match_finished", map[string]any{
		"winner":      winnerID,
		"corner":      winnerCorner,
		"result_type": resultType,
	}, "")

	o.result = map[string]any{
		"winner":      winnerID,
		"corner":      winnerCorner,
		"result_type": resultType,
	}
	o.h.Broadcast(hub.Message{
		Type: hub.MsgMatchResult,
		Payload: map[string]any{
			"winner":      winnerID,
			"corner":      winnerCorner,
			"result_type": resultType,
			"scores":      o.scores[1:4],
		},
	})
}

// Hitung total skor semua ronde, tentukan pemenang
func (o *Orchestrator) determineWinner() (winnerID string, resultType string) {
	var redTotal, blueTotal int
	for i := 1; i <= 3; i++ {
		redTotal += o.scores[i].RedTechnical - o.scores[i].RedPenalties
		blueTotal += o.scores[i].BlueTechnical - o.scores[i].BluePenalties
	}

	// ponytail: tiebreaker belum diimplementasi, draw = red menang (placeholder)
	switch {
	case redTotal > blueTotal:
		return "red", "point_win"
	case blueTotal > redTotal:
		return "blue", "point_win"
	default:
		return "red", "draw" // ponytail: tiebreaker rules belum ada
	}
}

func (o *Orchestrator) broadcastState() {
	payload := timerPayload(o.timer)
	payload["state"] = string(o.sm.Current())
	payload["round"] = o.sm.CurrentRoundNumber()
	payload["tie_pending"] = o.tiePendingLocked()
	o.h.Broadcast(hub.Message{Type: hub.MsgStateChange, Payload: payload})
}

func (o *Orchestrator) broadcastScoreUpdate(round int) {
	o.h.Broadcast(hub.Message{
		Type: hub.MsgScoreUpdate,
		Payload: map[string]any{
			"round":          round,
			"red_technical":  o.scores[round].RedTechnical,
			"blue_technical": o.scores[round].BlueTechnical,
			"red_penalties":  o.scores[round].RedPenalties,
			"blue_penalties": o.scores[round].BluePenalties,
		},
	})
}

func (o *Orchestrator) sendError(client *hub.Client, code, message string) {
	o.h.SendToClient(client.ID, hub.Message{
		Type: hub.MsgError,
		Payload: map[string]string{
			"code":    code,
			"message": message,
		},
	})
}

// appendEvent mencatat kejadian ke database dan menyiarkannya ke meja wasit/admin sebagai
// `match_event` (log real-time). Pemanggil WAJIB memegang o.mu. Setiap kejadian diberi
// `clock_ms` (sisa waktu timer saat itu) dan `round` bila belum ada.
func (o *Orchestrator) appendEvent(eventType string, payload map[string]any, judgeID string) {
	if payload == nil {
		payload = map[string]any{}
	}
	if _, ok := payload["clock_ms"]; !ok {
		payload["clock_ms"] = o.clockMS()
	}
	if _, ok := payload["round"]; !ok {
		payload["round"] = o.sm.CurrentRoundNumber()
	}

	raw, _ := json.Marshal(payload)
	ev := &store.MatchEvent{
		MatchID:   o.matchID,
		EventType: eventType,
		Payload:   string(raw),
		JudgeID:   judgeID,
		CreatedAt: time.Now().UTC(),
	}
	if o.eventStore != nil {
		if err := o.eventStore.Append(ev); err != nil {
			log.Printf("[orch:%s] event append error: %v", o.matchID, err)
		}
	}

	msg := hub.Message{Type: hub.MsgMatchEvent, Payload: map[string]any{
		"id":         ev.ID,
		"match_id":   ev.MatchID,
		"event_type": ev.EventType,
		"payload":    payload,
		"judge_id":   ev.JudgeID,
		"created_at": ev.CreatedAt,
	}}
	o.h.SendToRole(hub.RoleReferee, msg)
	o.h.SendToRole(hub.RoleAdmin, msg)
}

func (o *Orchestrator) GetMatchID() string {
	return o.matchID
}

func (o *Orchestrator) GetArenaID() string {
	return o.arenaID
}

func (o *Orchestrator) IsActive() bool {
	o.mu.Lock()
	defer o.mu.Unlock()
	st := o.sm.Current()
	return st != statemachine.StateFinished && st != statemachine.StateCancelled
}

func (o *Orchestrator) Start() error {
	return o.doStart()
}

func (o *Orchestrator) Pause() error {
	return o.doPause()
}

func (o *Orchestrator) Resume() error {
	return o.doResume()
}

func (o *Orchestrator) Finish() error {
	return o.doFinish("")
}

// FinishWithWinner mengakhiri pertandingan dengan pemenang yang ditentukan wasit (wajib bila seri).
func (o *Orchestrator) FinishWithWinner(winner string) error {
	return o.doFinish(winner)
}

func (o *Orchestrator) Cancel() error {
	return o.doCancel()
}
