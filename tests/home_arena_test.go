package tests

import (
	"errors"
	"strings"
	"testing"

	"github.com/herros27/digital-scoring/internal/domain"
	"github.com/herros27/digital-scoring/internal/service"
	"github.com/herros27/digital-scoring/internal/store"
)

// Gelanggang asal kategori: dipilih otomatis (beban paling ringan), stabil saat bagan diacak ulang, dipakai juga untuk
// babak lanjutan, dapat diubah sebelum partai berjalan, dan gelanggang yang dipakai tidak bisa dihapus.
func TestHomeArena_AssignmentAdvanceChangeAndGuards(t *testing.T) {
	a := newAdminDB(t)
	arenas := &store.ArenaStore{DB: a.db}
	matches := &store.MatchStore{DB: a.db}
	brackets := &store.BracketStore{DB: a.db}
	bsvc := service.NewBracketService(brackets).WithMatchStore(matches).WithArenaStore(arenas).WithCategoryArenas(a.categories)
	asvc := service.NewArenaService(arenas, nil)

	four := func(cat *domain.Category) []string {
		var ids []string
		for i := 0; i < 4; i++ {
			ids = append(ids, a.participant(t, cat, "", "verified"))
		}
		return ids
	}
	catA := a.category(t, "Kat Asal A")
	idsA := four(catA)
	catB := a.category(t, "Kat Asal B")
	idsB := four(catB)

	// tanpa gelanggang: bagan ditolak dengan pesan jelas, tidak ada slot yang terbentuk
	if _, err := bsvc.GenerateCategoryBracket(a.comp, catA.ID, catA.WeightClass, catA.Gender, "", idsA); err == nil || !strings.Contains(err.Error(), "gelanggang") {
		t.Fatalf("tanpa gelanggang harus ditolak: %v", err)
	}
	if slots, _ := brackets.GetByCategoryID(a.comp, catA.ID); len(slots) != 0 {
		t.Fatalf("tidak boleh ada slot tersisa: %d", len(slots))
	}

	arA := &store.Arena{CompetitionID: a.comp, Name: "Gelanggang A"}
	arB := &store.Arena{CompetitionID: a.comp, Name: "Gelanggang B"}
	for _, x := range []*store.Arena{arB, arA} { // sengaja dibuat terbalik: urutan nama yang menentukan, bukan urutan dibuat
		if err := arenas.Create(x); err != nil {
			t.Fatal(err)
		}
	}

	arenaOfCategory := func(cat *domain.Category) []string {
		rows, err := a.db.Query(`SELECT COALESCE(m.arena_id::text, '') FROM matches m JOIN bracket_slots b ON b.id = m.bracket_slot_id WHERE b.category_id = $1`, cat.ID)
		if err != nil {
			t.Fatal(err)
		}
		defer rows.Close()
		var out []string
		for rows.Next() {
			var id string
			_ = rows.Scan(&id)
			out = append(out, id)
		}
		return out
	}
	all := func(ids []string, want string) bool {
		for _, id := range ids {
			if id != want {
				return false
			}
		}
		return len(ids) > 0
	}

	// kategori pertama: beban seri -> gelanggang bernama paling awal; semua partainya di sana
	if _, err := bsvc.GenerateCategoryBracket(a.comp, catA.ID, catA.WeightClass, catA.Gender, "", idsA); err != nil {
		t.Fatal(err)
	}
	if got := arenaOfCategory(catA); len(got) != 2 || !all(got, arA.ID) {
		t.Fatalf("kategori A harus di Gelanggang A: %v", got)
	}
	// kategori kedua: Gelanggang A sudah terbebani -> Gelanggang B
	if _, err := bsvc.GenerateCategoryBracket(a.comp, catB.ID, catB.WeightClass, catB.Gender, "", idsB); err != nil {
		t.Fatal(err)
	}
	if got := arenaOfCategory(catB); !all(got, arB.ID) {
		t.Fatalf("kategori B harus di Gelanggang B: %v", got)
	}
	// acak ulang: gelanggang asal tetap
	if _, err := bsvc.GenerateCategoryBracket(a.comp, catA.ID, catA.WeightClass, catA.Gender, "", idsA); err != nil {
		t.Fatal(err)
	}
	if got := arenaOfCategory(catA); !all(got, arA.ID) {
		t.Fatalf("acak ulang tidak boleh memindahkan gelanggang: %v", got)
	}

	// pemenang maju: partai babak berikutnya di gelanggang asal yang sama
	slots, _ := brackets.GetByCategoryID(a.comp, catB.ID)
	for _, s := range slots {
		if s.Round == 1 && s.ParticipantID != nil && s.SlotIndex%2 == 0 {
			if err := brackets.AdvanceWinner(s.ID, *s.ParticipantID); err != nil {
				t.Fatal(err)
			}
		}
	}
	if got := arenaOfCategory(catB); len(got) != 3 || !all(got, arB.ID) {
		t.Fatalf("babak lanjutan harus di gelanggang asal (B): %v", got)
	}

	// gelanggang yang dipakai tidak boleh dihapus
	if err := asvc.DeleteArena(arA.ID); !errors.Is(err, domain.ErrConflict) {
		t.Errorf("hapus gelanggang terpakai harus ErrConflict: %v", err)
	}

	// panitia mengubah gelanggang asal sebelum ada partai berjalan: partai idle ikut pindah
	cat, err := a.catSvc.SetHomeArena(a.comp, catA.ID, arB.ID)
	if err != nil || cat.HomeArenaID == nil || *cat.HomeArenaID != arB.ID {
		t.Fatalf("ubah gelanggang asal: %v %+v", err, cat)
	}
	if got := arenaOfCategory(catA); !all(got, arB.ID) {
		t.Fatalf("partai idle harus ikut pindah: %v", got)
	}
	// gelanggang bukan milik kejuaraan ditolak
	if _, err := a.catSvc.SetHomeArena(a.comp, catA.ID, "00000000-0000-0000-0000-000000000000"); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("gelanggang asing harus ErrNotFound: %v", err)
	}
	// setelah ada partai kategori yang dimuat, memindahkan gelanggang asal ditolak (409)
	var loaded string
	_ = a.db.QueryRow(`SELECT m.id FROM matches m JOIN bracket_slots b ON b.id = m.bracket_slot_id WHERE b.category_id = $1 LIMIT 1`, catA.ID).Scan(&loaded)
	if err := matches.UpdateStatus(loaded, "ready"); err != nil {
		t.Fatal(err)
	}
	if _, err := a.catSvc.SetHomeArena(a.comp, catA.ID, arA.ID); !errors.Is(err, domain.ErrConflict) {
		t.Errorf("pindah setelah partai dimuat harus ErrConflict: %v", err)
	}
}
