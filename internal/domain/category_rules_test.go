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
	cases := map[int]string{2: "", 3: AgeSinga, 6: AgeSinga, 7: AgeMacan, 10: AgePreTeen, 12: AgePraRemaja, 14: AgeRemaja, 17: AgeDewasa, 45: AgeDewasa, 46: ""}
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
	if len(MissingDocs(AgeDewasa, nil)) != 0 {
		t.Error("adults need no protection docs")
	}
	if got := MissingDocs(AgeUsiaDini1, map[string]bool{DocIdentity: true}); len(got) != 2 {
		t.Errorf("want 2 missing, got %v", got)
	}
	all := map[string]bool{DocIdentity: true, DocParental: true, DocHealth: true}
	if len(MissingDocs(AgeRemaja, all)) != 0 {
		t.Error("complete docs should pass")
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
