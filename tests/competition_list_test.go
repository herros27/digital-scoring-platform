package tests

import (
	"fmt"
	"testing"
	"time"

	"github.com/herros27/digital-scoring/internal/domain"
	"github.com/herros27/digital-scoring/internal/service"
	"github.com/herros27/digital-scoring/internal/store"
)

// Daftar kejuaraan: total menurut filter, pencarian di server, urutan, dan hitungan per status yang tetap.
func TestCompetitionList_SearchPaginationStats(t *testing.T) {
	a := newAdminDB(t)
	cs := &store.CompetitionStore{DB: a.db}
	svc := service.NewCompetitionService(cs)

	tag := fmt.Sprintf("zz%d", time.Now().UnixNano())
	statuses := []string{"draft", "draft", "active", "cancelled", "cancelled", "cancelled", "completed"}
	var ids []string
	for i, st := range statuses {
		c := &domain.Competition{Name: fmt.Sprintf("%s Kejuaraan %d", tag, i), Location: "Kota_%", StartDate: "2026-06-01", EndDate: "2026-06-02", Status: st}
		if err := cs.Create(c); err != nil {
			t.Fatal(err)
		}
		ids = append(ids, c.ID)
	}
	t.Cleanup(func() {
		for _, id := range ids {
			_, _ = a.db.Exec(`DELETE FROM competitions WHERE id = $1`, id)
		}
	})

	all, total, err := svc.GetCompetitionsSearch("", tag, 1, 3)
	if err != nil || total != len(statuses) || len(all) != 3 {
		t.Fatalf("halaman 1: total=%d len=%d err=%v", total, len(all), err)
	}
	// urutan: active, draft, ... dibatalkan paling akhir
	if all[0].Status != "active" || all[1].Status != "draft" {
		t.Errorf("urutan: %s,%s", all[0].Status, all[1].Status)
	}
	p3, total3, _ := svc.GetCompetitionsSearch("", tag, 3, 3)
	if total3 != len(statuses) || len(p3) != 1 || p3[0].Status != "cancelled" {
		t.Errorf("halaman 3: total=%d len=%d", total3, len(p3))
	}
	if beyond, tb, _ := svc.GetCompetitionsSearch("", tag, 9, 3); len(beyond) != 0 || tb != len(statuses) {
		t.Errorf("halaman di luar jangkauan: len=%d total=%d", len(beyond), tb)
	}
	if _, tc, _ := svc.GetCompetitionsSearch("cancelled", tag, 1, 20); tc != 3 {
		t.Errorf("filter cancelled: %d", tc)
	}
	if _, tn, _ := svc.GetCompetitionsSearch("", tag+"-tidak-ada", 1, 20); tn != 0 {
		t.Errorf("tanpa hasil: %d", tn)
	}
	// karakter khusus LIKE diperlakukan literal: "%" saja tidak boleh cocok dengan semuanya
	if _, tp, _ := svc.GetCompetitionsSearch("", tag+"%", 1, 20); tp != 0 {
		t.Errorf("%% harus literal: %d", tp)
	}
	if _, tl, _ := svc.GetCompetitionsSearch("", "Kota_%", 1, 200); tl < len(statuses) {
		t.Errorf("pencarian lokasi literal: %d", tl)
	}
	if byID, tid, _ := svc.GetCompetitionsSearch("", ids[0], 1, 20); tid != 1 || len(byID) != 1 {
		t.Errorf("cari lewat ID: %d", tid)
	}

	stats, err := svc.CompetitionStats()
	if err != nil {
		t.Fatal(err)
	}
	sum := 0
	for _, k := range []string{"draft", "active", "paused", "completed", "cancelled"} {
		sum += stats[k]
	}
	if stats["total"] != sum {
		t.Errorf("total %d != jumlah status %d", stats["total"], sum)
	}
	if stats["cancelled"] < 3 || stats["draft"] < 2 {
		t.Errorf("hitungan per status: %v", stats)
	}
}
