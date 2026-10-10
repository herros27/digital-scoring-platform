package orchestrator

import (
	"fmt"

	"github.com/herros27/digital-scoring/internal/rules"
)

// tiebreakResult hasil langkah seri otomatis.
type tiebreakResult struct {
	Corner     string `json:"winner_corner"`
	ResultType string `json:"result_type"`
	Step       string `json:"step"`
	Detail     string `json:"detail"`
}

// Jenis hasil (result_type) bila pemenang ditentukan lewat aturan seri.
const (
	resultTiebreakPenalties = "tiebreak_penalties"
	resultTiebreakTechnique = "tiebreak_technique"
	// resultWMP: Wasit Menghentikan Pertandingan karena selisih poin mencapai batas ruleset.
	resultWMP = "wmp"
)

// manualResultType result_type untuk keputusan manual wasit saat seri.
func manualResultType(method string) string {
	if method == "" || method == rules.MethodRefereeDecision {
		return rules.MethodRefereeDecision
	}
	return "tiebreak_" + method
}

// penaltyTotalLocked total potongan sanksi satu sudut di seluruh ronde. Pemanggil memegang o.mu.
func (o *Orchestrator) penaltyTotalLocked(corner string) int {
	total := 0
	for i := 1; i <= 3; i++ {
		if corner == "red" {
			total += o.scores[i].RedPenalties
		} else {
			total += o.scores[i].BluePenalties
		}
	}
	return total
}

// resolveTiebreakLocked menjalankan langkah seri otomatis menurut ruleset. ok=false bila skor tidak
// seri atau semua langkah otomatis masih seri (wasit harus memutuskan). Pemanggil memegang o.mu.
func (o *Orchestrator) resolveTiebreakLocked() (res tiebreakResult, ok bool) {
	if !o.tiedLocked() {
		return tiebreakResult{}, false
	}
	tb := o.ruleset.Scoring.EffectiveTiebreak()
	for _, step := range tb.Steps {
		switch step {
		case rules.StepFewestPenalties:
			red, blue := o.penaltyTotalLocked("red"), o.penaltyTotalLocked("blue")
			if red != blue {
				corner := "red"
				if blue < red {
					corner = "blue"
				}
				return tiebreakResult{
					Corner: corner, ResultType: resultTiebreakPenalties, Step: step,
					Detail: fmt.Sprintf("potongan sanksi paling sedikit (merah %d, biru %d)", red, blue),
				}, true
			}
		case rules.StepTechniquePriority:
			for _, tech := range tb.TechniquePriority {
				red, blue := o.techCount["red"][tech], o.techCount["blue"][tech]
				if red == blue {
					continue
				}
				corner := "red"
				if blue > red {
					corner = "blue"
				}
				label := tech
				if t, found := o.ruleset.Scoring.Technique(tech); found {
					label = t.Label
				}
				return tiebreakResult{
					Corner: corner, ResultType: resultTiebreakTechnique, Step: step,
					Detail: fmt.Sprintf("%s terbanyak (merah %d, biru %d)", label, red, blue),
				}, true
			}
		}
	}
	return tiebreakResult{}, false
}

// countTechnique mencatat satu poin sah pada sudut untuk penghitungan seri. Pemanggil memegang o.mu.
func (o *Orchestrator) countTechnique(corner, tech string) {
	if o.techCount[corner] == nil {
		o.techCount[corner] = map[string]int{}
	}
	o.techCount[corner][tech]++
}

// forceTiebreakLocked menetapkan hasil dari langkah seri otomatis dan mencatat kejadiannya.
func (o *Orchestrator) forceTiebreakLocked(res tiebreakResult) {
	o.forced = &forcedResult{Corner: res.Corner, ResultType: res.ResultType}
	o.appendEvent("tiebreak_resolved", map[string]any{
		"winner_corner": res.Corner, "step": res.Step, "detail": res.Detail,
	}, "")
}

// tiebreakSnapshotLocked aturan seri dan pratinjau hasil otomatis saat skor sedang seri.
func (o *Orchestrator) tiebreakSnapshotLocked() map[string]any {
	tb := o.ruleset.Scoring.EffectiveTiebreak()
	out := map[string]any{
		"steps":              tb.Steps,
		"technique_priority": tb.TechniquePriority,
		"manual_methods":     tb.ManualMethods,
		"auto_result":        nil,
	}
	if res, ok := o.resolveTiebreakLocked(); ok {
		out["auto_result"] = res
	}
	return out
}
