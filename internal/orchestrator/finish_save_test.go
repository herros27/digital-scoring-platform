package orchestrator

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/herros27/digital-scoring/internal/consensus"
	"github.com/herros27/digital-scoring/internal/hub"
	"github.com/herros27/digital-scoring/internal/statemachine"
)

// flakyFinisher MatchStore palsu yang bisa menutup partai secara atomik; gagal sebanyak failures kali dulu.
type flakyFinisher struct {
	failures int
	corners  []string
}

func (f *flakyFinisher) UpdateStatus(string, string) error { return nil }
func (f *flakyFinisher) FinishMatch(_, corner string) (string, error) {
	f.corners = append(f.corners, corner)
	if f.failures > 0 {
		f.failures--
		return "", errors.New("koneksi database putus")
	}
	return "atlet-" + corner, nil
}

// Hasil akhir yang gagal ditulis tidak boleh hilang diam-diam: wasit diberi tahu, keadaan pemulihan dipertahankan,
// dan menekan Selesai lagi mengulang penulisan yang sama.
func TestFinish_UnsavedResultIsReportedKeptAndRetried(t *testing.T) {
	r := newRig(t)
	mem := newMemRuntimeStore()
	r.o.SetRuntimeStore(mem)
	fin := &flakyFinisher{failures: 1}
	r.o.matchStore = fin
	r.start(t)

	if err := r.o.FinishWithWinner("red"); !errors.Is(err, ErrResultNotSaved) {
		t.Fatalf("finish dengan database gagal = %v, want ErrResultNotSaved", err)
	}
	if errCode(drain(r.refCh)) != "RESULT_NOT_SAVED" {
		t.Error("wasit harus menerima RESULT_NOT_SAVED")
	}
	if find(drain(r.tvCh), hub.MsgMatchResult) == nil {
		t.Error("layar scoreboard tetap harus menerima hasil pertandingan")
	}
	if _, ok := mem.get("match-1"); !ok {
		t.Fatal("keadaan pemulihan tidak boleh dihapus sebelum hasil tersimpan")
	}

	if err := r.o.Finish(); err != nil {
		t.Fatalf("ulangi finish: %v", err)
	}
	if len(fin.corners) != 2 || fin.corners[0] != "red" || fin.corners[1] != "red" {
		t.Fatalf("penulisan ulang harus memakai pemenang yang sama: %v", fin.corners)
	}
	if _, ok := mem.get("match-1"); ok {
		t.Error("setelah tersimpan, keadaan pemulihan harus dihapus")
	}
	if got := r.o.result["winner"]; got != "atlet-red" {
		t.Errorf("winner = %v, want atlet-red", got)
	}
	if err := r.o.Finish(); err == nil {
		t.Error("finish ketiga harus ditolak: partai sudah selesai dan tersimpan")
	}
}

// Konsensus juri yang tiba setelah partai ditutup (bel ronde terakhir lebih dulu mengambil kunci) tidak boleh
// mengubah skor yang sudah menentukan pemenang, mengubah baris skor, atau menyiarkan poin baru.
func TestApplyScore_IgnoredAfterMatchFinished(t *testing.T) {
	r := newRig(t)
	r.start(t)
	if err := r.o.FinishWithWinner("red"); err != nil {
		t.Fatal(err)
	}
	drain(r.refCh)
	drain(r.tvCh)
	before := r.o.Snapshot()["red_total"]

	r.o.applyScore(&consensus.ConsensusResult{Accepted: true, ScoreType: "technique", Target: consensus.TargetBlue, Round: 1, Count: 2})

	if got := r.o.Snapshot()["red_total"]; got != before || r.o.scores[1].BlueTechnical != 0 {
		t.Errorf("skor berubah setelah partai selesai: red_total %v -> %v, biru ronde 1 = %d", before, got, r.o.scores[1].BlueTechnical)
	}
	if find(drain(r.tvCh), hub.MsgScoreConfirmed) != nil {
		t.Error("poin terlambat tidak boleh disiarkan")
	}
}

// failingStatusStore MatchStore palsu yang menolak menulis status "cancelled" sebanyak failures kali.
type failingStatusStore struct{ failures int }

func (f *failingStatusStore) UpdateStatus(_, status string) error {
	if status == "cancelled" && f.failures > 0 {
		f.failures--
		return errors.New("koneksi database putus")
	}
	return nil
}
func (f *failingStatusStore) FinishMatch(_, corner string) (string, error) { return corner, nil }

// Pembatalan yang gagal ditulis ke database tidak boleh diam-diam menutup partai di layar: partai tetap berjalan,
// pemulihan dipertahankan, dan pembatalan bisa diulang.
func TestCancel_DatabaseFailureLeavesMatchIntactAndRetryable(t *testing.T) {
	r := newRig(t)
	mem := newMemRuntimeStore()
	r.o.SetRuntimeStore(mem)
	r.o.matchStore = &failingStatusStore{failures: 1}
	r.start(t)

	if err := r.o.Cancel(); err == nil {
		t.Fatal("cancel dengan database gagal harus mengembalikan error")
	}
	if r.o.sm.Current() != statemachine.StateRound1 || !r.o.timer.IsRunning() {
		t.Fatalf("partai harus tetap berjalan: state=%s running=%v", r.o.sm.Current(), r.o.timer.IsRunning())
	}
	if _, ok := mem.get("match-1"); !ok {
		t.Fatal("keadaan pemulihan tidak boleh dihapus")
	}

	if err := r.o.Cancel(); err != nil {
		t.Fatalf("ulangi cancel: %v", err)
	}
	if r.o.sm.Current() != statemachine.StateCancelled {
		t.Errorf("state = %s, want cancelled", r.o.sm.Current())
	}
	if _, ok := mem.get("match-1"); ok {
		t.Error("setelah tersimpan, keadaan pemulihan harus dihapus")
	}
}

// Kunci idempotensi: kiriman ulang ditolak, dan kunci tua dibuang agar peta tidak tumbuh tanpa batas.
func TestFirstTime_RejectsRepeatAndPrunesOldKeys(t *testing.T) {
	r := newRig(t)
	if !r.o.firstTime("a") || r.o.firstTime("a") {
		t.Fatal("kunci yang sama hanya boleh lolos sekali")
	}
	for i := 0; i < 500; i++ {
		r.o.seenEvents[fmt.Sprintf("lama-%d", i)] = time.Now().Add(-10 * time.Minute)
	}
	if !r.o.firstTime("baru") {
		t.Fatal("kunci baru harus lolos")
	}
	if n := len(r.o.seenEvents); n > 3 {
		t.Errorf("kunci kedaluwarsa harus dibuang, tersisa %d", n)
	}
}
