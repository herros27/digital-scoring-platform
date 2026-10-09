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
	// techCount jumlah poin sah per sudut dan jenis teknik (untuk aturan seri)
	techCount map[string]map[string]int

	// pemulihan setelah server restart (lihat runtime.go)
	runtimeStore RuntimeStore
	recovered    bool // dipulihkan dari keadaan tersimpan dan belum dilanjutkan wasit
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
		engine:       newEngine(ruleset),
		h:            h,
		matchStore:   matchStore,
		eventStore:   eventStore,
		scoreStore:   scoreStore,
		bracketStore: bracketStore,
		seenEvents:   make(map[string]time.Time),
		lastInput:    make(map[int]judgeAction),
		inputCount:   make(map[int]int),
		techCount:    make(map[string]map[string]int),
	}
	// Pasang callback: setiap message dari hub masuk ke sini
	h.Attach(o, o.handleMessage, o.broadcastPresence)
	return o
}

// BroadcastSnapshot menyiarkan keadaan terkini ke semua klien arena sebagai sync_response
// (dipakai saat partai baru dimuat agar layar yang sudah tersambung langsung menyesuaikan diri).
func (o *Orchestrator) BroadcastSnapshot() {
	o.h.Broadcast(hub.Message{Type: hub.MsgSyncResponse, Payload: o.Snapshot()})
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
	Winner  string `json:"winner,omitempty"`   // finish: pemenang keputusan wasit (wajib bila seri dan langkah otomatis masih seri)
	Outcome string `json:"outcome,omitempty"`  // declare_outcome: ko | tko | walkover | withdrawal
	Note    string `json:"note,omitempty"`     // declare_outcome: catatan wasit (opsional, maks 200 karakter)
	Method  string `json:"method,omitempty"`   // finish: dasar keputusan (coin_toss, lighter_weight, extra_round; kosong = keputusan wasit)
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
		if err := o.doFinish(p.Winner, p.Method); err != nil {
			o.sendError(client, "INVALID_STATE_TRANSITION", err.Error())
		}
	case "cancel":
		if err := o.doCancel(); err != nil {
			o.sendError(client, "INVALID_STATE_TRANSITION", err.Error())
		}
	case "declare_outcome":
		if err := o.doDeclareOutcome(p.Outcome, p.Winner, p.Note); err != nil {
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

	o.recovered = false
	// Status 'running' disimpan agar daftar partai, badge LIVE, dan penghitung "sedang tanding" di UI
	// (yang membaca status dari database) benar; sebelumnya partai berjalan tetap tercatat 'idle'.
	if o.matchStore != nil {
		_ = o.matchStore.UpdateStatus(o.matchID, "running")
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
	// Status 'paused' disimpan agar daftar partai di UI panitia menampilkan Lanjutkan (bukan Jeda lagi)
	if o.matchStore != nil {
		_ = o.matchStore.UpdateStatus(o.matchID, "paused")
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
	o.recovered = false
	if o.matchStore != nil {
		_ = o.matchStore.UpdateStatus(o.matchID, "running")
	}
	o.appendEvent("match_resumed", nil, "")
	o.broadcastState()
	return nil
}

// doFinish mengakhiri pertandingan. Bila skor seri, wasit wajib menentukan pemenang
// (`winner` = "red"/"blue", keputusan wasit); tanpa itu pertandingan tidak diakhiri.
// doFinish mengakhiri pertandingan. `winner` (keputusan wasit) menimpa hasil; `method` menjelaskan
// dasar keputusan (kosong = keputusan wasit; selain itu salah satu metode manual ruleset).
// Tanpa `winner` dan skor seri, aturan seri otomatis dijalankan; bila masih seri, ErrTieNeedsDecision.
func (o *Orchestrator) doFinish(winner, method string) error {
	o.mu.Lock()
	defer o.mu.Unlock()

	if o.sm.IsFinished() {
		return fmt.Errorf("match is already finished")
	}
	// Partai yang belum dimulai tidak boleh "diselesaikan": tanpa pertandingan tidak ada hasil, dan
	// finalize() akan menetapkan pemenang, memajukan bagan, serta menyiarkan hasil sementara state
	// machine tetap idle. WO/undur diri sebelum mulai memakai declare_outcome; batal memakai cancel.
	if o.sm.Current() == statemachine.StateIdle {
		return fmt.Errorf("cannot finish match in idle state: match has not started (use declare_outcome for walkover or cancel)")
	}
	if winner != "" {
		if !validTarget(winner) {
			return fmt.Errorf("pemenang harus red atau blue")
		}
		if method != "" && method != rules.MethodRefereeDecision {
			allowed := false
			for _, m := range o.ruleset.Scoring.EffectiveTiebreak().ManualMethods {
				if m == method {
					allowed = true
				}
			}
			if !allowed {
				return fmt.Errorf("metode %q tidak tersedia pada ruleset ini", method)
			}
		}
		resultType := manualResultType(method)
		o.forced = &forcedResult{Corner: winner, ResultType: resultType}
		o.appendEvent("referee_decision", map[string]any{"winner_corner": winner, "method": resultType}, "")
	} else if o.sm.CurrentRoundNumber() > 0 && o.tiedLocked() {
		res, ok := o.resolveTiebreakLocked()
		if !ok {
			return ErrTieNeedsDecision
		}
		o.forceTiebreakLocked(res)
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
	o.clearRuntimeLocked()
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

// scoringOpen nil bila skor juri boleh masuk: babak berjalan dan jam tidak sedang dijeda. Partai yang dijeda
// (atau baru dipulihkan setelah restart) menolak input agar skor tidak masuk saat wasit menghentikan laga.
func (o *Orchestrator) scoringOpen() error {
	if !o.sm.IsInRound() {
		return fmt.Errorf("scoring not allowed in state %s", o.sm.Current())
	}
	o.mu.Lock()
	t := o.timer
	o.mu.Unlock()
	if t != nil && !t.IsRunning() {
		return fmt.Errorf("pertandingan sedang dijeda")
	}
	return nil
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
	if err := o.scoringOpen(); err != nil {
		o.sendError(client, "SCORING_NOT_ALLOWED", err.Error())
		return
	}

	var p judgeInputPayload
	if err := decodePayload(msg.Payload, &p); err != nil {
		o.sendError(client, "BAD_PAYLOAD", err.Error())
		return
	}

	if !o.validScoreType(p.ScoreType) {
		o.sendError(client, "BAD_PAYLOAD", fmt.Sprintf("score_type %q tidak dikenal pada ruleset ini", p.ScoreType))
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

	if p.ScoreType == undoScoreType {
		o.handleUndo(client, p)
		return
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
		o.notifyConsensusProgress(result)
		return
	}

	// Consensus tercapai → hitung poin
	o.applyScore(result)
}

// SubmitJudgeInput memungkinkan pengiriman scoring event via REST dengan idempotency key check
func (o *Orchestrator) SubmitJudgeInput(input consensus.JudgeInput, idempotencyKey string) (*consensus.ConsensusResult, bool, error) {
	if err := o.scoringOpen(); err != nil {
		return nil, false, err
	}
	if input.ScoreType == undoScoreType {
		return nil, false, fmt.Errorf("pembatalan hanya tersedia lewat WebSocket tablet juri")
	}
	if !o.validScoreType(string(input.ScoreType)) {
		return nil, false, fmt.Errorf("score_type %q tidak dikenal pada ruleset ini", input.ScoreType)
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
	} else {
		o.notifyConsensusProgress(result)
	}

	return result, false, nil
}

const undoScoreType = "undo"

// handleUndo membatalkan input juri ini yang masih menunggu juri lain (belum jadi poin) pada sudut tertentu.
// Poin yang sudah terkonfirmasi tidak dibatalkan dari tablet; itu wewenang wasit. Tanpa input yang bisa
// dibatalkan, juri mendapat UNDO_NOT_POSSIBLE agar tidak mengira pembatalan berhasil.
func (o *Orchestrator) handleUndo(client *hub.Client, p judgeInputPayload) {
	target := consensus.Target(p.Target)
	if target != consensus.TargetRed && target != consensus.TargetBlue {
		o.sendError(client, "BAD_PAYLOAD", "target pembatalan harus red atau blue")
		return
	}
	retracted, ok := o.engine.Retract(client.ID, target, o.sm.CurrentRoundNumber())
	if !ok {
		o.sendError(client, "UNDO_NOT_POSSIBLE", "tidak ada input yang menunggu juri lain; poin terkonfirmasi tidak dapat dibatalkan dari tablet")
		return
	}
	o.mu.Lock()
	o.appendEvent("judge_undo", map[string]any{
		"judge_id":    client.ID,
		"seat_number": client.SeatNumber,
		"score_type":  retracted.ScoreType,
		"target":      retracted.Target,
		"round":       retracted.Round,
	}, client.ID)
	o.mu.Unlock()
	// count 0 memadamkan lampu konsensus di tablet; count>0 menurunkan hitungan juri yang tersisa
	o.h.SendToRole(hub.RoleJudge, hub.Message{
		Type: hub.MsgConsensusProgress,
		Payload: map[string]any{
			"target":     retracted.Target,
			"score_type": retracted.ScoreType,
			"round":      retracted.Round,
			"count":      retracted.Remaining,
			"needed":     o.engine.Majority(),
			"window_ms":  o.engine.Window().Milliseconds(),
		},
	})
}

// notifyConsensusProgress memberi tahu tablet juri berapa juri yang sudah menekan teknik dan sudut
// yang sama dalam window (anonim: tanpa identitas kursi, agar juri tidak saling terpengaruh).
func (o *Orchestrator) notifyConsensusProgress(res *consensus.ConsensusResult) {
	if res == nil || res.Accepted || res.Count == 0 {
		return
	}
	o.h.SendToRole(hub.RoleJudge, hub.Message{
		Type: hub.MsgConsensusProgress,
		Payload: map[string]any{
			"target":     res.Target,
			"score_type": res.ScoreType,
			"round":      res.Round,
			"count":      res.Count,
			"needed":     o.engine.Majority(),
			"window_ms":  o.engine.Window().Milliseconds(),
		},
	})
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
	if pts > 0 {
		o.countTechnique(string(result.Target), string(result.ScoreType))
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

// pointsFor nilai teknik menurut ruleset; 0 untuk jenis yang tidak dikenal (mis. "undo").
func (o *Orchestrator) pointsFor(st consensus.ScoreType) int {
	if t, ok := o.ruleset.Scoring.Technique(string(st)); ok {
		return t.Points
	}
	return 0
}

// newEngine engine konsensus sesuai aturan ruleset (jumlah juri sepakat dan window waktu).
func newEngine(rs *rules.Ruleset) *consensus.Engine {
	c := rs.Scoring.EffectiveConsensus()
	return consensus.NewEngineWith(c.Window(), c.Majority)
}

// validScoreType jenis input juri yang diterima: teknik ruleset atau "undo" (pembatalan dari tablet).
func (o *Orchestrator) validScoreType(t string) bool {
	if t == "undo" {
		return true
	}
	_, ok := o.ruleset.Scoring.Technique(t)
	return ok
}

func (o *Orchestrator) handleSyncRequest(client *hub.Client) {
	o.mu.Lock()
	payload := o.snapshotLocked()
	o.mu.Unlock()

	o.h.SendToConn(client, hub.Message{
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
		"round":       o.roundNumberLocked(),
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

// isFinalRoundLocked true bila ronde yang berjalan adalah ronde terakhir menurut ruleset:
// 3 babak (default) atau 2 babak (mis. Usia Dini, total_rounds=2). Pemanggil memegang o.mu.
func (o *Orchestrator) isFinalRoundLocked() bool {
	cur := o.sm.Current()
	if cur == statemachine.StateRound3 {
		return true
	}
	return cur == statemachine.StateRound2 && o.ruleset.Scoring.TotalRounds == 2
}

// roundNumberLocked nomor ronde yang ditampilkan, dibatasi total_rounds ruleset
// (pertandingan 2 babak yang selesai tidak dilaporkan sebagai ronde 3).
func (o *Orchestrator) roundNumberLocked() int {
	n := o.sm.CurrentRoundNumber()
	if tr := o.ruleset.Scoring.TotalRounds; tr > 0 && n > tr {
		return tr
	}
	return n
}

func (o *Orchestrator) onRoundExpired() {
	o.mu.Lock()
	defer o.mu.Unlock()

	// Ronde terakhir selesai → langsung finish, kecuali skor seri: pertandingan menunggu keputusan wasit
	if o.isFinalRoundLocked() {
		finalRound := o.roundNumberLocked()
		if o.tiedLocked() {
			res, ok := o.resolveTiebreakLocked()
			if !ok {
				o.appendEvent("tie_pending", map[string]any{"round": finalRound}, "")
				o.broadcastState()
				return
			}
			o.forceTiebreakLocked(res)
		}
		o.sm.Transition(statemachine.EventFinish)
		o.appendEvent("round_ended", map[string]any{"round": finalRound}, "")
		o.finalize()
		return
	}

	// Ronde 1 atau 2 selesai → masuk break
	roundNum := o.sm.CurrentRoundNumber()
	o.sm.Transition(statemachine.EventRoundEnd)
	o.appendEvent("round_ended", map[string]any{"round": roundNum}, "")

	// Setelah break, mulai ronde berikutnya otomatis. Timer istirahat dibuat SEBELUM siaran agar
	// state_change break_N membawa hitung mundur istirahat (sisa waktu + tenggat); bila disiarkan lebih
	// dulu, semua layar menerima timer ronde yang sudah habis (0 dan tidak berjalan) sampai timer_sync berikutnya.
	breakDur := time.Duration(o.ruleset.Scoring.BreakDuration) * time.Second
	o.timer = o.newTimer(breakDur, o.onBreakExpired)
	o.timer.Start()
	o.broadcastState()
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
	// Siarkan juga state_change "finished" (timer berhenti) sebelum hasil: layar juri hanya mengikuti
	// state_change, jadi tanpa ini tablet juri tetap menampilkan "BERJALAN" dan tombol skor tetap aktif
	// setelah partai ditutup.
	o.broadcastState()
	o.h.Broadcast(hub.Message{
		Type: hub.MsgMatchResult,
		Payload: map[string]any{
			"winner":      winnerID,
			"corner":      winnerCorner,
			"result_type": resultType,
			"scores":      o.scores[1:4],
		},
	})
	o.clearRuntimeLocked()
}

// Hitung total skor semua ronde, tentukan pemenang
func (o *Orchestrator) determineWinner() (winnerID string, resultType string) {
	var redTotal, blueTotal int
	for i := 1; i <= 3; i++ {
		redTotal += o.scores[i].RedTechnical - o.scores[i].RedPenalties
		blueTotal += o.scores[i].BlueTechnical - o.scores[i].BluePenalties
	}

	// Seri tidak sampai di sini: finalize dipanggil setelah aturan seri atau keputusan wasit menetapkan
	// o.forced. "draw" hanya cadangan bila seri tak terselesaikan.
	switch {
	case redTotal > blueTotal:
		return "red", "point_win"
	case blueTotal > redTotal:
		return "blue", "point_win"
	default:
		return "red", "draw"
	}
}

func (o *Orchestrator) broadcastState() {
	payload := timerPayload(o.timer)
	payload["state"] = string(o.sm.Current())
	payload["round"] = o.sm.CurrentRoundNumber()
	payload["tie_pending"] = o.tiePendingLocked()
	o.h.Broadcast(hub.Message{Type: hub.MsgStateChange, Payload: payload})
	o.persistLocked()
}

func (o *Orchestrator) broadcastScoreUpdate(round int) {
	defer o.persistLocked()
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
	o.h.SendToConn(client, hub.Message{
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
	return o.doFinish("", "")
}

// FinishWithWinner mengakhiri pertandingan dengan pemenang yang ditentukan wasit (wajib bila seri).
func (o *Orchestrator) FinishWithWinner(winner string) error {
	return o.doFinish(winner, "")
}

// DeclareOutcome menyatakan hasil khusus (KO, TKO, WO, undur diri) dengan `winner` sebagai sudut pemenang.
func (o *Orchestrator) DeclareOutcome(outcome, winner, note string) error {
	return o.doDeclareOutcome(outcome, winner, note)
}

// FinishWithDecision seperti FinishWithWinner dengan metode keputusan (mis. coin_toss, lighter_weight).
func (o *Orchestrator) FinishWithDecision(winner, method string) error {
	return o.doFinish(winner, method)
}

func (o *Orchestrator) Cancel() error {
	return o.doCancel()
}
