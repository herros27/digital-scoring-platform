package tests

import (
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/herros27/digital-scoring/internal/domain"
	"github.com/herros27/digital-scoring/internal/rules"
	"github.com/herros27/digital-scoring/internal/service"
	"github.com/herros27/digital-scoring/internal/store"
)

// Mengganti ruleset kejuaraan harus ikut memindahkan kategori yang terikat ke ruleset lama; kalau tidak, kategori tetap
// memakai ruleset lama dan ruleset lama tidak bisa diarsipkan. Ruleset khusus kategori yang berbeda tetap utuh.
func TestCompetitionRulesetSwitch_ReleasesCategories(t *testing.T) {
	a := newAdminDB(t)
	rs := &store.RulesetStore{DB: a.db}
	comps := &store.CompetitionStore{DB: a.db}
	csvc := service.NewCompetitionService(comps)
	rsvc := service.NewRulesetService(rs)

	mk := func(name string) *domain.Ruleset {
		r := &domain.Ruleset{Name: fmt.Sprintf("%s %d", name, time.Now().UnixNano()), Version: "1", Scoring: rules.PresetScoring(), Status: "active"}
		if err := rs.Create(r); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() {
			_, _ = a.db.Exec(`UPDATE categories SET ruleset_id = NULL WHERE ruleset_id = $1`, r.ID)
			_, _ = a.db.Exec(`UPDATE competitions SET ruleset_id = NULL WHERE ruleset_id = $1`, r.ID)
			_, _ = a.db.Exec(`DELETE FROM rulesets WHERE id = $1`, r.ID)
		})
		return r
	}
	oldRS, newRS, special := mk("Lama"), mk("Baru"), mk("Khusus")

	catPinned := a.category(t, "Terikat Lama")
	catSpecial := a.category(t, "Khusus Usia Dini")
	_, _ = a.db.Exec(`UPDATE competitions SET ruleset_id = $1 WHERE id = $2`, oldRS.ID, a.comp)
	_, _ = a.db.Exec(`UPDATE categories SET ruleset_id = $1 WHERE id = $2`, oldRS.ID, catPinned.ID)
	_, _ = a.db.Exec(`UPDATE categories SET ruleset_id = $1 WHERE id = $2`, special.ID, catSpecial.ID)

	// selama kategori masih terikat, ruleset lama tidak bisa diarsipkan; pesannya menyebut kategori, bukan hanya kejuaraan
	in := service.CreateRulesetInput{Name: oldRS.Name, Version: "1", Scoring: rules.PresetScoring(), Status: "archived"}
	if _, err := rsvc.UpdateRuleset(oldRS.ID, in); !errors.Is(err, domain.ErrConflict) || !strings.Contains(err.Error(), "kategori di") {
		t.Fatalf("arsip ruleset yang masih dirujuk harus ditolak dengan pesan kategori: %v", err)
	}

	if _, err := csvc.UpdateCompetition(a.comp, &domain.Competition{RulesetID: &newRS.ID}); err != nil {
		t.Fatalf("ganti ruleset kejuaraan: %v", err)
	}
	var pinned, spec *string
	_ = a.db.QueryRow(`SELECT ruleset_id::text FROM categories WHERE id = $1`, catPinned.ID).Scan(&pinned)
	_ = a.db.QueryRow(`SELECT ruleset_id::text FROM categories WHERE id = $1`, catSpecial.ID).Scan(&spec)
	if pinned != nil {
		t.Errorf("kategori yang terikat ke ruleset lama harus mengikuti kejuaraan (NULL), dapat %s", *pinned)
	}
	if spec == nil || *spec != special.ID {
		t.Errorf("ruleset khusus kategori tidak boleh berubah: %v", spec)
	}
	if _, err := rsvc.UpdateRuleset(oldRS.ID, in); err != nil {
		t.Fatalf("setelah dilepas, ruleset lama harus bisa diarsipkan: %v", err)
	}
}
