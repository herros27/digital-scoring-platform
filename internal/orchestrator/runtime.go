package orchestrator

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/herros27/digital-scoring/internal/hub"
	"github.com/herros27/digital-scoring/internal/rules"
	"github.com/herros27/digital-scoring/internal/statemachine"
	"github.com/herros27/digital-scoring/internal/store"
)

// RuntimeStore penyimpanan keadaan partai aktif agar dapat dipulihkan setelah server restart.
type RuntimeStore interface {
	Save(r *store.MatchRuntime) error
	Delete(matchID string) error
}

// runtimeData isi keadaan partai di luar kolom state/timer. Ruleset ikut disimpan: mengubah ruleset di
// editor tidak boleh mengubah partai yang sedang berjalan, juga setelah server restart.
type runtimeData struct {
	Ruleset       rules.Ruleset             `json:"ruleset"`
	Scores        [4]RoundScore             `json:"scores"`
	Penalties     []penaltyEntry            `json:"penalties"`
	NextPenaltyID int                       `json:"next_penalty_id"`
	MedicActive   bool                      `json:"medic_active"`
	VarActive     bool                      `json:"var_active"`
	Forced        *forcedResult             `json:"forced,omitempty"`
	TechCount     map[string]map[string]int `json:"tech_count"`
	LastInput     map[int]judgeAction       `json:"last_input"`
	InputCount    map[int]int               `json:"input_count"`
}

// SetRuntimeStore memasang penyimpanan pemulihan dan langsung menyimpan keadaan awal partai.
func (o *Orchestrator) SetRuntimeStore(rs RuntimeStore) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.runtimeStore = rs
	o.persistLocked()
}

// persistLocked menyimpan keadaan terbaru. Pemanggil memegang o.mu. Tidak berbuat apa-apa bila partai
// sudah selesai/dibatalkan (barisnya sudah/akan dihapus) atau penyimpanan tidak dipasang. Kegagalan
// hanya dicatat: partai tetap berjalan, hanya pemulihan yang tidak terjamin.
func (o *Orchestrator) persistLocked() {
	if o.runtimeStore == nil || o.sm.IsFinished() {
		return
	}
	data, err := json.Marshal(runtimeData{
		Ruleset:       *o.ruleset,
		Scores:        o.scores,
		Penalties:     o.penalties,
		NextPenaltyID: o.nextPenaltyID,
		MedicActive:   o.medicActive,
		VarActive:     o.varActive,
		Forced:        o.forced,
		TechCount:     o.techCount,
		LastInput:     o.lastInput,
		InputCount:    o.inputCount,
	})
	if err != nil {
		log.Printf("[orch:%s] runtime marshal error: %v", o.matchID, err)
		return
	}
	var remaining int64
	var running bool
	if o.timer != nil {
		remaining = o.timer.Remaining().Milliseconds()
		running = o.timer.IsRunning()
	}
	if err := o.runtimeStore.Save(&store.MatchRuntime{
		MatchID: o.matchID, ArenaID: o.arenaID, State: string(o.sm.Current()),
		TimerRemainingMS: remaining, TimerRunning: running, Data: data,
	}); err != nil {
		log.Printf("[orch:%s] runtime save error: %v", o.matchID, err)
	}
}

// clearRuntimeLocked menghapus keadaan tersimpan (partai selesai atau dibatalkan). Pemanggil memegang o.mu.
func (o *Orchestrator) clearRuntimeLocked() {
	if o.runtimeStore == nil {
		return
	}
	if err := o.runtimeStore.Delete(o.matchID); err != nil {
		log.Printf("[orch:%s] runtime delete error: %v", o.matchID, err)
	}
}

// Restore membangun kembali orchestrator dari keadaan tersimpan setelah server restart.
// Ronde yang sedang berjalan dipulihkan DIJEDA dengan sisa waktu catatan terakhir (wasit menekan
// Lanjutkan); istirahat antar ronde dilanjutkan otomatis dengan sisa waktunya. Input juri yang belum
// mencapai konsensus tidak ikut dipulihkan.
func Restore(
	rec store.MatchRuntime,
	h *hub.Hub,
	matchStore MatchStore,
	eventStore *store.MatchEventStore,
	scoreStore *store.MatchScoreStore,
	rs RuntimeStore,
) (*Orchestrator, error) {
	var d runtimeData
	if err := json.Unmarshal(rec.Data, &d); err != nil {
		return nil, fmt.Errorf("data pemulihan rusak: %w", err)
	}
	state := statemachine.State(rec.State)
	switch state {
	case statemachine.StateFinished, statemachine.StateCancelled:
		return nil, fmt.Errorf("partai sudah %s", state)
	}

	sm := statemachine.NewWithState(rec.MatchID, state)
	ruleset := d.Ruleset
	o := New(rec.MatchID, rec.ArenaID, sm, &ruleset, h, matchStore, eventStore, scoreStore)

	o.mu.Lock()
	defer o.mu.Unlock()
	o.scores = d.Scores
	o.penalties = d.Penalties
	o.nextPenaltyID = d.NextPenaltyID
	o.medicActive = d.MedicActive
	o.varActive = d.VarActive
	o.forced = d.Forced
	if d.TechCount != nil {
		o.techCount = d.TechCount
	}
	if d.LastInput != nil {
		o.lastInput = d.LastInput
	}
	if d.InputCount != nil {
		o.inputCount = d.InputCount
	}

	remaining := time.Duration(rec.TimerRemainingMS) * time.Millisecond
	switch {
	case sm.IsInRound():
		o.timer = o.newTimer(remaining, o.onRoundExpired) // sengaja tidak dijalankan: dipulihkan dijeda
		o.recovered = true
		if matchStore != nil {
			_ = matchStore.UpdateStatus(rec.MatchID, "paused") // daftar partai di UI menampilkan Lanjutkan
		}
	case state == statemachine.StateBreak1 || state == statemachine.StateBreak2:
		o.timer = o.newTimer(remaining, o.onBreakExpired)
		o.timer.Start()
		o.recovered = true
	default: // idle / ready: belum dimulai
		o.recovered = false
	}

	o.runtimeStore = rs
	if o.recovered {
		o.appendEvent("match_recovered", map[string]any{
			"state": rec.State, "remaining_ms": remaining.Milliseconds(),
		}, "")
	}
	o.persistLocked()
	return o, nil
}
