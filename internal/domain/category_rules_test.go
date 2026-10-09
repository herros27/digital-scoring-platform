package domain

import (
	"testing"
	"time"
)

func d(s string) time.Time { t, _ := time.Parse("2006-01-02", s); return t }

func TestAgeAt(t *testing.T) {
	ref := d("2026-06-15")
	cases := map[string]int{"2020-06-15": 6, "2020-06-16": 5, "2008-01-01": 18, "1990-12-31": 35}
	for birth, want := range cases {
		if got := AgeAt(d(birth), ref); got != want {
			t.Errorf("AgeAt(%s)=%d want %d", birth, got, want)
		}
	}
}

func TestAgeGroupForAge_IPSI(t *testing.T) {
	cases := map[int]string{
		0: AgePraUsiaDini, 5: AgePraUsiaDini, 6: AgeUsiaDini1, 8: AgeUsiaDini1,
		9: AgeUsiaDini2, 11: AgeUsiaDini2, 12: AgePraRemaja, 14: AgePraRemaja,
		15: AgeRemaja, 17: AgeRemaja, 18: AgeDewasa, 35: AgeDewasa,
		36: AgeMasterA, 45: AgeMasterA, 46: AgeMasterB, 70: AgeMasterB,
	}
	for age, want := range cases {
		if got := AgeGroupForAge(age, PresetIPSI); got != want {
			t.Errorf("IPSI age %d = %q want %q", age, got, want)
		}
	}
}

func TestAgeGroupForAge_Persilat(t *testing.T) {
	cases := map[int]string{2: "", 3: AgeSinga, 6: AgeSinga, 7: AgeMacan, 10: AgePreTeen, 12: AgePraRemaja, 14: AgeRemaja, 17: AgeDewasa, 45: AgeDewasa, 46: AgeMasterA, 60: AgeMasterA, 61: AgeMasterB}
	for age, want := range cases {
		if got := AgeGroupForAge(age, PresetPersilat26); got != want {
			t.Errorf("PERSILAT age %d = %q want %q", age, got, want)
		}
	}
}

func f(v float64) *float64 { return &v }

func TestCheckWeight_NoTolerance(t *testing.T) {
	c := &Category{MinWeight: f(45), MaxWeight: f(50)}
	cases := []struct {
		w    float64
		want string
	}{{45, WeightOK}, {50, WeightOK}, {47.5, WeightOK}, {50.1, WeightOver}, {44.9, WeightUnder}, {50.01, WeightOver}}
	for _, tc := range cases {
		if got := c.CheckWeight(tc.w); got != tc.want {
			t.Errorf("CheckWeight(%v)=%s want %s", tc.w, got, tc.want)
		}
	}
	open := &Category{MinWeight: f(95)}
	if open.CheckWeight(130) != WeightOK || open.CheckWeight(94.9) != WeightUnder {
		t.Error("open-ended class bounds wrong")
	}
	if (&Category{}).CheckWeight(60) != WeightUnspecified {
		t.Error("category without bounds must be unspecified")
	}
}

func TestMissingDocs(t *testing.T) {
	// Semua atlet wajib bukti umur + surat sehat (IPSI & PERSILAT), termasuk dewasa.
	if got := MissingDocs(AgeDewasa, "male", nil); len(got) != 2 {
		t.Errorf("adult male needs identity+health, missing=%v", got)
	}
	// Surat izin orang tua tidak disyaratkan regulasi.
	ok := map[string]bool{DocIdentity: true, DocHealth: true}
	if len(MissingDocs(AgeUsiaDini1, "male", ok)) != 0 {
		t.Error("parental letter must not be required")
	}
	// Pernyataan tidak hamil: hanya putri Remaja & Dewasa.
	if got := MissingDocs(AgeRemaja, "putri", ok); len(got) != 1 || got[0] != DocPregnancy {
		t.Errorf("female remaja must declare not pregnant, missing=%v", got)
	}
	if len(MissingDocs(AgeDewasa, "female", ok)) != 1 {
		t.Error("female dewasa must declare not pregnant")
	}
	if len(MissingDocs(AgePraRemaja, "female", ok)) != 0 || len(MissingDocs(AgeRemaja, "male", ok)) != 0 {
		t.Error("pregnancy declaration only for female remaja/dewasa")
	}
}

func TestCheckEligibility(t *testing.T) {
	cat := &Category{Name: "Remaja Putra A", Gender: "putra", AgeGroup: AgeRemaja, PresetStandard: PresetIPSI}
	ref := d("2026-06-15")
	ok := &Athlete{Name: "A", Gender: "male", BirthDate: "2010-03-01"} // 16
	if err := CheckEligibility(ok, cat, ref); err != nil {
		t.Errorf("unexpected: %v", err)
	}
	old := &Athlete{Name: "B", Gender: "male", BirthDate: "2000-01-01"}
	if CheckEligibility(old, cat, ref) == nil {
		t.Error("adult must not enter remaja")
	}
	female := &Athlete{Name: "C", Gender: "female", BirthDate: "2010-03-01"}
	if CheckEligibility(female, cat, ref) == nil {
		t.Error("gender mismatch must fail")
	}
	noBirth := &Athlete{Name: "D", Gender: "male"}
	if CheckEligibility(noBirth, cat, ref) == nil {
		t.Error("missing birth_date must fail")
	}
}

func TestParseBirthDate(t *testing.T) {
	if _, err := ParseBirthDate("2010-02-30"); err == nil {
		t.Error("invalid date accepted")
	}
	if _, err := ParseBirthDate("2999-01-01"); err == nil {
		t.Error("future date accepted")
	}
	if _, err := ParseBirthDate("2010-02-03"); err != nil {
		t.Error(err)
	}
}

func TestAgeCheck(t *testing.T) {
	ref := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	cases := []struct {
		name, birth, group, preset, want string
	}{
		{"sesuai dewasa", "2000-01-15", AgeDewasa, PresetIPSI, ""},
		{"batas bawah dewasa (genap 18 di hari acuan)", "2008-06-01", AgeDewasa, PresetIPSI, ""},
		{"sehari sebelum 18 masih remaja", "2008-06-02", AgeDewasa, PresetIPSI, AgeCheckMismatch},
		{"remaja untuk kategori dewasa", "2010-01-01", AgeDewasa, PresetIPSI, AgeCheckMismatch},
		{"tanggal lahir kosong", "", AgeDewasa, PresetIPSI, AgeCheckMissing},
		{"format salah dianggap belum diisi", "01/01/2000", AgeDewasa, PresetIPSI, AgeCheckMissing},
		{"kategori tanpa golongan umur dilewati", "", "", PresetIPSI, ""},
		{"persilat: usia 14 = remaja", "2012-01-01", AgeRemaja, PresetPersilat26, ""},
		{"ipsi: usia 14 = pra remaja, bukan remaja", "2012-01-01", AgeRemaja, PresetIPSI, AgeCheckMismatch},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := AgeCheck(tc.birth, tc.group, tc.preset, ref); got != tc.want {
				t.Fatalf("AgeCheck(%q,%q,%q) = %q, want %q", tc.birth, tc.group, tc.preset, got, tc.want)
			}
		})
	}
}
