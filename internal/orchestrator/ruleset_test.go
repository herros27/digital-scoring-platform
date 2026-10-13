package orchestrator

import (
	"testing"

	"github.com/herros27/digital-scoring/internal/hub"
	"github.com/herros27/digital-scoring/internal/rules"
)

func customRuleset() *rules.Ruleset {
	return &rules.Ruleset{Name: "Custom", Scoring: rules.ScoringRule{
		RoundDuration: 120, BreakDuration: 60, TotalRounds: 3,
		Techniques: []rules.TechniqueRule{
			{Type: "technique", Label: "Pukulan", Points: 1},
			{Type: "catch_fall", Label: "Tangkapan + Jatuhan", Points: 4},
		},
		Penalties: []rules.PenaltyRule{
			{Type: "teguran_x", Label: "Teguran X", Points: 7},
			{Type: "dq", Label: "Pelanggaran Berat", Action: rules.ActionDisqualify},
		},
		// satu juri sudah cukup (mis. latihan atau format 1 juri)
		Consensus: &rules.ConsensusRule{Judges: 1, Majority: 1, WindowMs: 2000},
	}}
}

func TestDefaultPenaltyTable_Teguran2IsMinus2(t *testing.T) {
	r := newRig(t)
	r.start(t)
	r.send(r.referee, hub.MsgPenalty, map[string]any{"target": "red", "type": "teguran_1"})
	r.send(r.referee, hub.MsgPenalty, map[string]any{"target": "red", "type": "teguran_2"})
	if got := r.o.Snapshot()["red_total"]; got != -3 {
		t.Fatalf("red_total = %v, want -3 (teguran I -1, teguran II -2)", got)
	}
}

func TestDefaultPenaltyTable_Peringatan3Disqualifies(t *testing.T) {
	r := newRig(t)
	r.start(t)
	r.send(r.referee, hub.MsgPenalty, map[string]any{"target": "red", "type": "peringatan_3"})
	if !r.o.sm.IsFinished() {
		t.Fatal("peringatan III harus mengakhiri pertandingan")
	}
	res := r.o.Snapshot()["result"].(map[string]any)
	if res["corner"] != "blue" || res["result_type"] != "disqualification" {
		t.Fatalf("hasil = %+v, want blue/disqualification", res)
	}
}

func TestRuleset_TechniquesAndConsensusFromRuleset(t *testing.T) {
	r := newRigWith(t, customRuleset())
	r.start(t)

	// satu juri cukup, dan teknik khusus ruleset bernilai 4
	r.send(r.judge, hub.MsgJudgeInput, map[string]any{"score_type": "catch_fall", "target": "red", "event_id": "e1"})
	if got := r.o.Snapshot()["red_total"]; got != 4 {
		t.Fatalf("red_total = %v, want 4", got)
	}

	// teknik yang tidak ada pada ruleset ditolak (fall tidak didefinisikan)
	drain(r.judgeCh)
	r.send(r.judge, hub.MsgJudgeInput, map[string]any{"score_type": "fall", "target": "red", "event_id": "e2"})
	if c := errCode(drain(r.judgeCh)); c != "BAD_PAYLOAD" {
		t.Fatalf("teknik tak dikenal: %q", c)
	}
	if got := r.o.Snapshot()["red_total"]; got != 4 {
		t.Fatalf("skor tidak boleh berubah, red_total = %v", got)
	}

	// sanksi khusus ruleset
	r.send(r.referee, hub.MsgPenalty, map[string]any{"target": "blue", "type": "teguran_x"})
	if got := r.o.Snapshot()["blue_total"]; got != -7 {
		t.Fatalf("blue_total = %v, want -7", got)
	}
	// sanksi bawaan yang tidak ada pada ruleset ini ditolak
	drain(r.refCh)
	r.send(r.referee, hub.MsgPenalty, map[string]any{"target": "blue", "type": "binaan_1"})
	if c := errCode(drain(r.refCh)); c != "BAD_PAYLOAD" {
		t.Fatalf("sanksi tak dikenal: %q", c)
	}
}

func TestSnapshot_ExposesRulesetCatalogs(t *testing.T) {
	r := newRigWith(t, customRuleset())
	snap := r.o.Snapshot()
	if got := len(snap["techniques"].([]rules.TechniqueRule)); got != 2 {
		t.Errorf("techniques = %d, want 2", got)
	}
	if got := len(snap["penalty_catalog"].([]rules.PenaltyRule)); got != 2 {
		t.Errorf("penalty_catalog = %d, want 2", got)
	}
	if snap["judge_count"] != 1 {
		t.Errorf("judge_count = %v, want 1", snap["judge_count"])
	}
}
