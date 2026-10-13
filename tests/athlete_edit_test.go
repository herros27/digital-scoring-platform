package tests

import (
	"errors"
	"testing"
	"time"

	"github.com/herros27/digital-scoring/internal/domain"
)

func strPtr(s string) *string { return &s }

// Nama dan perguruan atlet dapat diperbaiki; nilai tidak valid ditolak dan atlet tak dikenal dijawab ErrNotFound.
func TestUpdateAthleteIdentity(t *testing.T) {
	a := newAdminDB(t)
	cat := a.category(t, "Kat Edit Atlet")
	pid := a.participant(t, cat, "", "pending")
	part, err := a.athletes.GetParticipantByID(pid)
	if err != nil {
		t.Fatal(err)
	}

	if err := a.athSvc.UpdateAthleteIdentity(part.AthleteID, strPtr("  Nama Baru  "), strPtr("Perguruan Baru")); err != nil {
		t.Fatalf("ubah identitas: %v", err)
	}
	got, _ := a.athletes.GetParticipantByID(pid)
	if got.Name != "Nama Baru" || got.Club != "Perguruan Baru" {
		t.Errorf("peserta harus mengikuti data atlet: %q / %q", got.Name, got.Club)
	}
	if err := a.athSvc.UpdateAthleteIdentity(part.AthleteID, strPtr("x"), nil); !errors.Is(err, domain.ErrInvalidInput) {
		t.Errorf("nama 1 huruf: %v", err)
	}
	if err := a.athSvc.UpdateAthleteIdentity(part.AthleteID, nil, nil); !errors.Is(err, domain.ErrInvalidInput) {
		t.Errorf("tanpa field: %v", err)
	}
	if err := a.athSvc.UpdateAthleteIdentity("00000000-0000-0000-0000-000000000000", strPtr("Nama Valid"), nil); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("atlet tidak ada: %v", err)
	}
}

// Hapus permanen (purge) menghapus peserta dan data atletnya; tanpa purge atlet tetap tersimpan.
func TestRemoveParticipant_Purge(t *testing.T) {
	a := newAdminDB(t)
	cat := a.category(t, "Kat Hapus Atlet")
	athleteCount := func(athleteID string) (n int) {
		_ = a.db.QueryRow(`SELECT COUNT(*) FROM athletes WHERE id = $1`, athleteID).Scan(&n)
		return
	}

	keep := a.participant(t, cat, "", "pending")
	keepPart, _ := a.athletes.GetParticipantByID(keep)
	if err := a.athSvc.RemoveParticipant(a.comp, keep, "", false, false); err != nil {
		t.Fatal(err)
	}
	if athleteCount(keepPart.AthleteID) != 1 {
		t.Error("tanpa purge, data atlet harus tetap ada")
	}

	gone := a.participant(t, cat, "", "rejected")
	gonePart, _ := a.athletes.GetParticipantByID(gone)
	if err := a.athSvc.RemoveParticipant(a.comp, gone, "", false, true); err != nil {
		t.Fatal(err)
	}
	if athleteCount(gonePart.AthleteID) != 0 {
		t.Error("dengan purge, data atlet harus ikut terhapus")
	}
}

// Pindah kategori: umur/gender divalidasi, peserta yang sudah diverifikasi kembali menunggu verifikasi,
// dan peserta yang sudah masuk bagan ditolak (ErrConflict).
func TestChangeParticipantCategory(t *testing.T) {
	r := newBracketRig(t)
	from, to := r.category(t, "Kat Asal"), r.category(t, "Kat Tujuan")
	female := &domain.Category{CompetitionID: r.comp, Name: "Kat Putri", Gender: "female", WeightClass: "B"}
	if err := r.catSvc.CreateCategory(female); err != nil {
		t.Fatal(err)
	}
	pid := r.participant(t, from, "", "verified")
	ref := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	move := func(catID string) error {
		return r.athSvc.ChangeParticipantCategory(r.comp, pid, catID, "", false, ref)
	}

	if err := move(female.ID); err == nil {
		t.Error("kategori beda gender harus ditolak")
	}
	if err := move(to.ID); err != nil {
		t.Fatalf("pindah kategori: %v", err)
	}
	got, _ := r.athletes.GetParticipantByID(pid)
	if got.CategoryID != to.ID || got.RegistrationStatus != "pending" {
		t.Errorf("setelah pindah: kategori=%s status=%s, want %s/pending", got.CategoryID, got.RegistrationStatus, to.ID)
	}
	if err := r.athSvc.ChangeParticipantCategory(r.comp, pid, to.ID, "orang-lain", true, ref); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("manajer kontingen lain: %v", err)
	}

	other := r.participant(t, to, "", "verified")
	if _, err := r.h.Service.GenerateCategoryBracket(r.comp, to.ID, to.WeightClass, to.Gender, "", []string{pid, other}); err != nil {
		t.Fatal(err)
	}
	if err := move(from.ID); !errors.Is(err, domain.ErrConflict) {
		t.Errorf("peserta yang sudah di bagan harus ditolak: %v", err)
	}
}
