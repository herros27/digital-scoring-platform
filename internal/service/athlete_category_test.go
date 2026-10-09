package service

import (
	"database/sql"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/herros27/digital-scoring/internal/domain"
)

type fakeAthleteRepo struct {
	athletes map[string]*domain.Athlete
	parts    map[string]*domain.Participant
	saved    map[string]string
}

func (f *fakeAthleteRepo) Create(a *domain.Athlete) error { f.athletes[a.ID] = a; return nil }
func (f *fakeAthleteRepo) GetByID(id string) (*domain.Athlete, error) {
	return f.athletes[id], nil
}
func (f *fakeAthleteRepo) List() ([]domain.Athlete, error) { return nil, nil }
func (f *fakeAthleteRepo) RegisterToCompetition(string, string, string, string) error {
	return nil
}
func (f *fakeAthleteRepo) ListByCompetition(string) ([]domain.Athlete, error) { return nil, nil }
func (f *fakeAthleteRepo) RegisterParticipantInCategory(compID, athID, contID, catID, wc, gender, status string) error {
	f.parts["p1"] = &domain.Participant{ID: "p1", CompetitionID: compID, AthleteID: athID, CategoryID: catID, WeightClass: wc, Gender: gender, RegistrationStatus: status}
	return nil
}
func (f *fakeAthleteRepo) GetParticipantByID(id string) (*domain.Participant, error) {
	p, ok := f.parts[id]
	if !ok {
		return nil, sql.ErrNoRows
	}
	// seperti join ke tabel atlet di store sungguhan: tanggal lahir ikut terbaca
	if a := f.athletes[p.AthleteID]; a != nil && p.BirthDate == "" {
		cp := *p
		cp.BirthDate = a.BirthDate
		return &cp, nil
	}
	return p, nil
}

func (f *fakeAthleteRepo) UpdateProfile(id string, birthDate, nik *string) error {
	a, ok := f.athletes[id]
	if !ok {
		return domain.ErrNotFound
	}
	if birthDate != nil {
		a.BirthDate = *birthDate
	}
	if nik != nil {
		a.NIK = *nik
	}
	return nil
}
func (f *fakeAthleteRepo) SaveVerification(id, status string, w *float64, docs map[string]bool) error {
	f.saved[id] = status
	f.parts[id].RegistrationStatus = status
	f.parts[id].ActualWeight = w
	f.parts[id].DocsVerified = docs
	return nil
}

type fakeCategoryRepo struct{ cats map[string]*domain.Category }

func (f *fakeCategoryRepo) Create(c *domain.Category) error { f.cats[c.ID] = c; return nil }
func (f *fakeCategoryRepo) GetByID(id string) (*domain.Category, error) {
	c, ok := f.cats[id]
	if !ok {
		return nil, sql.ErrNoRows
	}
	return c, nil
}
func (f *fakeCategoryRepo) ListByCompetition(string) ([]domain.Category, error) { return nil, nil }

func fl(v float64) *float64 { return &v }

func setup() (*AthleteService, *fakeAthleteRepo) {
	ar := &fakeAthleteRepo{athletes: map[string]*domain.Athlete{}, parts: map[string]*domain.Participant{}, saved: map[string]string{}}
	cr := &fakeCategoryRepo{cats: map[string]*domain.Category{
		"cat-sd": {ID: "cat-sd", CompetitionID: "c1", Name: "Usia Dini 2 Putra A", Gender: "putra", WeightClass: "A",
			AgeGroup: domain.AgeUsiaDini2, PresetStandard: domain.PresetIPSI, MinWeight: fl(26), MaxWeight: fl(28)},
		"cat-dew": {ID: "cat-dew", CompetitionID: "c1", Name: "Dewasa Putra A", Gender: "putra", WeightClass: "A",
			AgeGroup: domain.AgeDewasa, PresetStandard: domain.PresetIPSI, MinWeight: fl(45), MaxWeight: fl(50)},
	}}
	ar.athletes["kid"] = &domain.Athlete{ID: "kid", Name: "Kid", Gender: "male", BirthDate: "2016-05-01"} // 10 th
	ar.athletes["adult"] = &domain.Athlete{ID: "adult", Name: "Adult", Gender: "male", BirthDate: "2000-01-01"}
	return NewAthleteService(ar).WithCategoryRepo(cr), ar
}

var refDate = time.Date(2026, 6, 15, 0, 0, 0, 0, time.UTC)

func TestRegisterToCategory_AgeGuard(t *testing.T) {
	svc, _ := setup()
	if _, err := svc.RegisterParticipantToCategory("c1", "kid", "", "cat-sd", "pending", refDate); err != nil {
		t.Fatalf("kid should enter usia dini 2: %v", err)
	}
	if _, err := svc.RegisterParticipantToCategory("c1", "adult", "", "cat-sd", "pending", refDate); err == nil {
		t.Fatal("adult must not enter usia dini 2")
	}
	if _, err := svc.RegisterParticipantToCategory("c1", "kid", "", "cat-dew", "pending", refDate); err == nil {
		t.Fatal("kid must not enter dewasa")
	}
	if _, err := svc.RegisterParticipantToCategory("other-comp", "kid", "", "cat-sd", "pending", refDate); err == nil {
		t.Fatal("category from another competition must be rejected")
	}
}

func TestVerify_WeightNoTolerance(t *testing.T) {
	svc, ar := setup()
	if _, err := svc.RegisterParticipantToCategory("c1", "adult", "", "cat-dew", "pending", refDate); err != nil {
		t.Fatal(err)
	}
	docs := map[string]bool{domain.DocIdentity: true, domain.DocHealth: true}
	res, err := svc.VerifyParticipantDetailed("p1", VerifyInput{Status: "verified", ActualWeight: fl(50.1), Docs: docs})
	if err != nil || res.Status != domain.ParticipantOverweight {
		t.Fatalf("want overweight, got %v %v", res, err)
	}
	res, _ = svc.VerifyParticipantDetailed("p1", VerifyInput{Status: "verified", ActualWeight: fl(44.9), Docs: docs})
	if res.Status != domain.ParticipantUnderweight {
		t.Fatalf("want underweight, got %s", res.Status)
	}
	res, err = svc.VerifyParticipantDetailed("p1", VerifyInput{Status: "verified", ActualWeight: fl(50), Docs: docs})
	if err != nil || res.Status != "verified" || ar.parts["p1"].RegistrationStatus != "verified" {
		t.Fatalf("exact limit must pass: %v %v", res, err)
	}
	if _, err := svc.VerifyParticipantDetailed("p1", VerifyInput{Status: "verified"}); err != nil {
		t.Fatalf("previously stored weight should be reused: %v", err)
	}
}

func TestVerify_WeightRequiredAndDocs(t *testing.T) {
	svc, ar := setup()
	if _, err := svc.RegisterParticipantToCategory("c1", "kid", "", "cat-sd", "pending", refDate); err != nil {
		t.Fatal(err)
	}
	_, err := svc.VerifyParticipantDetailed("p1", VerifyInput{Status: "verified", ActualWeight: fl(27)})
	if err == nil || !strings.Contains(err.Error(), "berkas") {
		t.Fatalf("minor without docs must be blocked, got %v", err)
	}
	docs := map[string]bool{domain.DocIdentity: true, domain.DocHealth: true}
	if _, err := svc.VerifyParticipantDetailed("p1", VerifyInput{Status: "verified", Docs: docs}); err == nil {
		t.Fatal("weight must be required for weight-bound category")
	}
	res, err := svc.VerifyParticipantDetailed("p1", VerifyInput{Status: "verified", ActualWeight: fl(27), Docs: docs})
	if err != nil || res.Status != "verified" {
		t.Fatalf("complete docs + valid weight must verify: %v %v", res, err)
	}
	if !ar.parts["p1"].DocsVerified[domain.DocHealth] {
		t.Error("docs not persisted")
	}
	// Reject tidak mensyaratkan berkas/bobot.
	if res, err := svc.VerifyParticipantDetailed("p1", VerifyInput{Status: "rejected"}); err != nil || res.Status != "rejected" {
		t.Fatalf("reject failed: %v %v", res, err)
	}
}

func TestNormalizeCategory(t *testing.T) {
	c := &domain.Category{CompetitionID: "c1", Name: "x", Gender: "male"}
	if err := normalizeCategory(c); err != nil || c.AgeGroup != domain.AgeDewasa || c.PresetStandard != domain.PresetIPSI || c.BracketMode != domain.BracketSingleElim {
		t.Fatalf("defaults wrong: %+v %v", c, err)
	}
	bad := []*domain.Category{
		{AgeGroup: "singa", PresetStandard: domain.PresetIPSI},
		{PresetStandard: "foo"},
		{BracketMode: "round_robin"},
		{MinWeight: fl(50), MaxWeight: fl(40)},
	}
	for i, b := range bad {
		if normalizeCategory(b) == nil {
			t.Errorf("case %d should be rejected", i)
		}
	}
	p := &domain.Category{AgeGroup: "singa", PresetStandard: domain.PresetPersilat26}
	if err := normalizeCategory(p); err != nil {
		t.Errorf("singa valid for persilat: %v", err)
	}
}

func TestCreateCategory_OneStandardPerCompetition(t *testing.T) {
	repo := &mockCategoryRepo{categories: map[string]*domain.Category{}}
	svc := NewCategoryService(repo)
	ipsi := &domain.Category{ID: "a", CompetitionID: "c1", Name: "IPSI A", Gender: "male", WeightClass: "A", PresetStandard: domain.PresetIPSI}
	if err := svc.CreateCategory(ipsi); err != nil {
		t.Fatal(err)
	}
	pers := &domain.Category{ID: "b", CompetitionID: "c1", Name: "PERSILAT", Gender: "male", WeightClass: "BEBAS", AgeGroup: domain.AgeSinga, PresetStandard: domain.PresetPersilat26}
	err := svc.CreateCategory(pers)
	if !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("mixing standards must conflict, got %v", err)
	}
	// Kejuaraan lain bebas memilih standar sendiri.
	other := &domain.Category{ID: "c", CompetitionID: "c2", Name: "PERSILAT", Gender: "male", WeightClass: "BEBAS", AgeGroup: domain.AgeSinga, PresetStandard: domain.PresetPersilat26}
	if err := svc.CreateCategory(other); err != nil {
		t.Fatalf("other competition: %v", err)
	}
	if _, err := svc.ResetCategories("c1"); err == nil {
		t.Fatal("mock repo has no reset support, expected error")
	}
}

func TestVerifyBlockedWhenBirthDateMissingOrAgeMismatch(t *testing.T) {
	svc, ar := setup()
	docs := map[string]bool{domain.DocIdentity: true, domain.DocHealth: true}
	ref := time.Date(2026, 6, 15, 0, 0, 0, 0, time.UTC)

	// peserta lama: atlet tanpa tanggal lahir
	ar.athletes["legacy"] = &domain.Athlete{ID: "legacy", Name: "Lama", Gender: "male"}
	ar.parts["p1"] = &domain.Participant{ID: "p1", AthleteID: "legacy", CategoryID: "cat-sd", Gender: "male"}
	_, err := svc.VerifyParticipantDetailed("p1", VerifyInput{Status: "verified", ActualWeight: fl(27), Docs: docs, Ref: ref})
	if err == nil || !strings.Contains(err.Error(), "tanggal lahir") {
		t.Fatalf("tanpa tanggal lahir harus ditolak, got %v", err)
	}

	// melengkapi tanggal lahir yang cocok golongan Usia Dini 2 -> lolos
	if _, err := svc.UpdateAthleteProfile("legacy", sp("2016-05-01"), nil); err != nil {
		t.Fatal(err)
	}
	res, err := svc.VerifyParticipantDetailed("p1", VerifyInput{Status: "verified", ActualWeight: fl(27), Docs: docs, Ref: ref})
	if err != nil || res.Status != "verified" {
		t.Fatalf("setelah dilengkapi harus lolos, got %v %v", res, err)
	}

	// tanggal lahir yang salah (dewasa) untuk kategori Usia Dini 2 -> ditolak
	if _, err := svc.UpdateAthleteProfile("legacy", sp("1990-01-01"), nil); err != nil {
		t.Fatal(err)
	}
	_, err = svc.VerifyParticipantDetailed("p1", VerifyInput{Status: "verified", ActualWeight: fl(27), Docs: docs, Ref: ref})
	if err == nil || !strings.Contains(err.Error(), "tidak sesuai golongan umur") {
		t.Fatalf("usia tidak sesuai harus ditolak, got %v", err)
	}

	// penolakan tidak diblokir pemeriksaan umur
	if res, err := svc.VerifyParticipantDetailed("p1", VerifyInput{Status: "rejected", Ref: ref}); err != nil || res.Status != "rejected" {
		t.Fatalf("rejected harus tetap bisa, got %v %v", res, err)
	}
}

func sp(s string) *string { return &s }

func TestUpdateAthleteProfileValidation(t *testing.T) {
	svc, ar := setup()
	if _, err := svc.UpdateAthleteProfile("kid", nil, nil); !errors.Is(err, domain.ErrInvalidInput) {
		t.Fatalf("tanpa field harus ErrInvalidInput, got %v", err)
	}
	for _, bad := range []string{"01/01/2000", "2099-01-01", "bukan-tanggal"} {
		if _, err := svc.UpdateAthleteProfile("kid", sp(bad), nil); err == nil {
			t.Fatalf("birth_date %q seharusnya ditolak", bad)
		}
	}
	if _, err := svc.UpdateAthleteProfile("kid", nil, sp(strings.Repeat("9", 51))); err == nil {
		t.Fatal("nik > 50 karakter harus ditolak")
	}
	if _, err := svc.UpdateAthleteProfile("tidak-ada", sp("2000-01-01"), nil); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("atlet tidak ada harus ErrNotFound, got %v", err)
	}
	got, err := svc.UpdateAthleteProfile("kid", sp(" 2016-06-01 "), sp("3201"))
	if err != nil || got.BirthDate != "2016-06-01" || ar.athletes["kid"].NIK != "3201" {
		t.Fatalf("pembaruan valid gagal: %v %+v", err, got)
	}
}
