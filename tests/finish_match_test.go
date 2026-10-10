package tests

import (
	"database/sql"
	"fmt"
	"testing"
	"time"

	"github.com/herros27/digital-scoring/internal/service"
	"github.com/herros27/digital-scoring/internal/store"
)

// FinishMatch menutup partai dan memajukan bagan dalam satu transaksi: pemenang naik, partai babak berikutnya
// terbentuk; bila kemajuan bagan gagal, status dan pemenang partai tidak ikut berubah.
func TestFinishMatch_AtomicWithBracketAdvance(t *testing.T) {
	a := newAdminDB(t)
	arenas := &store.ArenaStore{DB: a.db}
	matches := &store.MatchStore{DB: a.db}
	brackets := &store.BracketStore{DB: a.db}
	bsvc := service.NewBracketService(brackets).WithMatchStore(matches).WithArenaStore(arenas).WithCategoryArenas(a.categories)

	if err := arenas.Create(&store.Arena{CompetitionID: a.comp, Name: "Gelanggang Uji"}); err != nil {
		t.Fatal(err)
	}
	cat := a.category(t, "Kat Selesai")
	var ids []string
	for i := 0; i < 4; i++ {
		ids = append(ids, a.participant(t, cat, "", "verified"))
	}
	if _, err := bsvc.GenerateCategoryBracket(a.comp, cat.ID, cat.WeightClass, cat.Gender, "", ids); err != nil {
		t.Fatal(err)
	}

	type row struct{ id, red, blue, slot string }
	var round1 []row
	rows, err := a.db.Query(`SELECT m.id, m.athlete_red_id, m.athlete_blue_id, m.bracket_slot_id
		FROM matches m JOIN bracket_slots b ON b.id = m.bracket_slot_id
		WHERE b.category_id = $1 ORDER BY b.slot_index`, cat.ID)
	if err != nil {
		t.Fatal(err)
	}
	for rows.Next() {
		var r row
		if err := rows.Scan(&r.id, &r.red, &r.blue, &r.slot); err != nil {
			t.Fatal(err)
		}
		round1 = append(round1, r)
	}
	rows.Close()
	if len(round1) != 2 {
		t.Fatalf("partai babak 1 = %d, want 2", len(round1))
	}

	status := func(id string) (st string, winner sql.NullString) {
		if err := a.db.QueryRow(`SELECT status, winner_id::text FROM matches WHERE id = $1`, id).Scan(&st, &winner); err != nil {
			t.Fatal(err)
		}
		return
	}
	matchCount := func() (n int) {
		_ = a.db.QueryRow(`SELECT count(*) FROM matches m JOIN bracket_slots b ON b.id = m.bracket_slot_id WHERE b.category_id = $1`, cat.ID).Scan(&n)
		return
	}

	// kemajuan bagan dipaksa gagal: trigger menolak pengisian slot berikutnya
	fn, trg := fmt.Sprintf("tolak_maju_%d", time.Now().UnixNano()), fmt.Sprintf("trg_tolak_maju_%d", time.Now().UnixNano())
	for _, q := range []string{
		`CREATE FUNCTION ` + fn + `() RETURNS trigger AS $$ BEGIN RAISE EXCEPTION 'maju ditolak'; END $$ LANGUAGE plpgsql`,
		`CREATE TRIGGER ` + trg + ` BEFORE UPDATE OF participant_id ON bracket_slots FOR EACH ROW WHEN (NEW.category_id = '` + cat.ID + `') EXECUTE FUNCTION ` + fn + `()`,
	} {
		if _, err := a.db.Exec(q); err != nil {
			t.Fatal(err)
		}
	}
	dropTrigger := func() {
		_, _ = a.db.Exec(`DROP TRIGGER IF EXISTS ` + trg + ` ON bracket_slots`)
		_, _ = a.db.Exec(`DROP FUNCTION IF EXISTS ` + fn + `()`)
	}
	t.Cleanup(dropTrigger)

	if _, err := matches.FinishMatch(round1[0].id, "red"); err == nil {
		t.Fatal("kemajuan bagan gagal, FinishMatch harus mengembalikan error")
	}
	if st, w := status(round1[0].id); st == "finished" || w.Valid {
		t.Fatalf("transaksi harus dibatalkan seluruhnya, got status=%s winner=%v", st, w)
	}

	dropTrigger()
	winner, err := matches.FinishMatch(round1[0].id, "red")
	if err != nil || winner != round1[0].red {
		t.Fatalf("FinishMatch = %q, %v; want %q", winner, err, round1[0].red)
	}
	if st, w := status(round1[0].id); st != "finished" || w.String != round1[0].red {
		t.Fatalf("status=%s winner=%v", st, w)
	}
	if got := matchCount(); got != 2 {
		t.Fatalf("belum ada lawan di final, jumlah partai = %d, want 2", got)
	}

	// pemenang partai kedua melengkapi final: partai babak berikutnya terbentuk dalam transaksi yang sama
	if _, err := matches.FinishMatch(round1[1].id, "blue"); err != nil {
		t.Fatal(err)
	}
	if got := matchCount(); got != 3 {
		t.Fatalf("final harus terbentuk, jumlah partai = %d, want 3", got)
	}

	if _, err := matches.FinishMatch(round1[0].id, "purple"); err == nil {
		t.Error("sudut pemenang tidak dikenal harus ditolak")
	}
}
