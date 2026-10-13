package tests

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/herros27/digital-scoring/internal/api/handler"
	"github.com/herros27/digital-scoring/internal/domain"
	"github.com/herros27/digital-scoring/internal/rules"
	"github.com/herros27/digital-scoring/internal/service"
	"github.com/herros27/digital-scoring/internal/store"
)

// Layar juri sebelum ada partai dimuat memakai daftar teknik dari ruleset kejuaraan: 6 teknik di ruleset = 6 tombol.
func TestCompetitionTechniques_FollowsCompetitionRuleset(t *testing.T) {
	a := newAdminDB(t)
	rs := &store.RulesetStore{DB: a.db}
	scoring := rules.PresetScoring()
	scoring.Techniques = []rules.TechniqueRule{
		{Type: "technique", Label: "Pukulan", Points: 1},
		{Type: "knockdown", Label: "Tendangan", Points: 2},
		{Type: "fall", Label: "Jatuhan", Points: 3},
		{Type: "catch_fall", Label: "Tangkap Jatuh", Points: 4},
		{Type: "combo_1_2", Label: "Kombinasi 1-2", Points: 3},
		{Type: "combo_1_1", Label: "Kombinasi 1-1", Points: 2},
	}
	r := &domain.Ruleset{Name: fmt.Sprintf("Enam Teknik %d", time.Now().UnixNano()), Version: "1", Scoring: scoring, Status: "active"}
	if err := rs.Create(r); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = a.db.Exec(`UPDATE competitions SET ruleset_id = NULL WHERE ruleset_id = $1`, r.ID)
		_, _ = a.db.Exec(`DELETE FROM rulesets WHERE id = $1`, r.ID)
	})
	_, _ = a.db.Exec(`UPDATE competitions SET ruleset_id = $1 WHERE id = $2`, r.ID, a.comp)

	h := &handler.RulesetHandler{
		Service:      service.NewRulesetService(rs),
		Competitions: service.NewCompetitionService(&store.CompetitionStore{DB: a.db}),
	}
	call := func(compID string) (int, []rules.TechniqueRule) {
		req := httptest.NewRequest(http.MethodGet, "/x", nil)
		req.SetPathValue("comp_id", compID)
		w := httptest.NewRecorder()
		h.CompetitionTechniques(w, req)
		var env struct {
			Data []rules.TechniqueRule `json:"data"`
		}
		_ = json.Unmarshal(w.Body.Bytes(), &env)
		return w.Code, env.Data
	}

	code, list := call(a.comp)
	if code != http.StatusOK || len(list) != 6 {
		t.Fatalf("ruleset 6 teknik: status=%d jumlah=%d, want 200/6", code, len(list))
	}
	if code, _ := call("00000000-0000-0000-0000-000000000000"); code != http.StatusNotFound {
		t.Errorf("kejuaraan tidak ada = %d, want 404", code)
	}
}
