package domain

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

// Standar preset kategori turnamen.
const (
	PresetIPSI        = "ipsi"
	PresetPersilat26  = "persilat_2026"
	BracketSingleElim = "single_elimination"
	BracketPemasalan4 = "pemasalan_4"
)

// Golongan umur (kode disimpan di categories.age_group).
const (
	AgePraUsiaDini = "pra_usia_dini"
	AgeUsiaDini1   = "usia_dini_1"
	AgeUsiaDini2   = "usia_dini_2"
	AgePraRemaja   = "pra_remaja"
	AgeRemaja      = "remaja"
	AgeDewasa      = "dewasa"
	AgeMasterA     = "master_a"
	AgeMasterB     = "master_b"
	// Khusus PERSILAT 2026
	AgeSinga   = "singa"
	AgeMacan   = "macan"
	AgePreTeen = "pre_teen"
)

// Status peserta tambahan hasil timbang badan (selain pending/verified/rejected).
const (
	ParticipantOverweight  = "overweight"
	ParticipantUnderweight = "underweight"
)

// Kunci checklist berkas verifikasi (participants.docs_verified).
const (
	// DocIdentity bukti umur & identitas. IPSI: Akte Kelahiran / Ijazah / Paspor (asli atau legalisir).
	// PERSILAT: kartu identitas / paspor asli berfoto.
	DocIdentity = "identity"
	// DocHealth surat keterangan sehat (IPSI: dokter RS / RS Tipe D / Puskesmas, maks. 1 minggu sebelum).
	DocHealth = "health"
	// DocPregnancy surat pernyataan tidak hamil (putri Remaja & Dewasa / Junior & Senior).
	DocPregnancy = "pregnancy"
	// DocParental surat izin orang tua / wali. TIDAK disyaratkan regulasi IPSI maupun PERSILAT;
	// opsional sesuai kebijakan panitia, tidak pernah memblokir verifikasi.
	DocParental = "parental"
)

// AgeBand adalah rentang usia (tahun penuh, inklusif) satu golongan umur.
// MaxAge == 0 berarti tanpa batas atas.
type AgeBand struct {
	Code   string
	MinAge int
	MaxAge int
}

var ipsiBands = []AgeBand{
	{AgePraUsiaDini, 0, 5},
	{AgeUsiaDini1, 6, 8},
	{AgeUsiaDini2, 9, 11},
	{AgePraRemaja, 12, 14},
	{AgeRemaja, 15, 17},
	{AgeDewasa, 18, 35},
	{AgeMasterA, 36, 45},
	{AgeMasterB, 46, 0},
}

var persilat26Bands = []AgeBand{
	{AgeSinga, 3, 6},
	{AgeMacan, 7, 9},
	{AgePreTeen, 10, 11},
	{AgePraRemaja, 12, 13},
	{AgeRemaja, 14, 16},
	{AgeDewasa, 17, 45},
	{AgeMasterA, 46, 60},
	{AgeMasterB, 61, 0},
}

// AgeBands mengembalikan daftar golongan umur untuk sebuah standar (default IPSI).
func AgeBands(standard string) []AgeBand {
	if standard == PresetPersilat26 {
		return persilat26Bands
	}
	return ipsiBands
}

// ValidAgeGroup memeriksa apakah kode golongan umur dikenal oleh salah satu standar.
func ValidAgeGroup(code string) bool {
	for _, std := range []string{PresetIPSI, PresetPersilat26} {
		for _, b := range AgeBands(std) {
			if b.Code == code {
				return true
			}
		}
	}
	return false
}

// AgeGroupForAge menentukan golongan umur dari usia (tahun penuh). Kosong jika di luar semua band.
func AgeGroupForAge(age int, standard string) string {
	if age < 0 {
		return ""
	}
	for _, b := range AgeBands(standard) {
		if age >= b.MinAge && (b.MaxAge == 0 || age <= b.MaxAge) {
			return b.Code
		}
	}
	return ""
}

const birthDateLayout = "2006-01-02"

// ParseBirthDate mem-parse tanggal lahir format YYYY-MM-DD dan menolak tanggal di masa depan.
func ParseBirthDate(s string) (time.Time, error) {
	t, err := time.Parse(birthDateLayout, strings.TrimSpace(s))
	if err != nil {
		return time.Time{}, errors.New("birth_date must use format YYYY-MM-DD")
	}
	if t.After(time.Now().AddDate(0, 0, 1)) {
		return time.Time{}, errors.New("birth_date cannot be in the future")
	}
	return t, nil
}

// AgeAt menghitung usia tahun penuh pada tanggal acuan (hari pertama kejuaraan).
func AgeAt(birth, ref time.Time) int {
	age := ref.Year() - birth.Year()
	if ref.Month() < birth.Month() || (ref.Month() == birth.Month() && ref.Day() < birth.Day()) {
		age--
	}
	return age
}

// NormalizeGender menyeragamkan putra/putri -> male/female.
func NormalizeGender(g string) string {
	switch strings.ToLower(strings.TrimSpace(g)) {
	case "putra", "male", "m":
		return "male"
	case "putri", "female", "f":
		return "female"
	}
	return strings.ToLower(strings.TrimSpace(g))
}

// Hasil timbang badan.
const (
	WeightOK          = "ok"
	WeightOver        = "overweight"
	WeightUnder       = "underweight"
	WeightUnspecified = "unspecified"
)

// HasWeightBounds true jika kategori memiliki batas bobot.
func (c *Category) HasWeightBounds() bool {
	return c != nil && (c.MinWeight != nil || c.MaxWeight != nil)
}

// CheckWeight menilai bobot aktual terhadap kelas. TIDAK ADA toleransi (0 kg):
// selisih 0.1 kg pun melanggar. Batas min/max bersifat inklusif.
func (c *Category) CheckWeight(actual float64) string {
	if !c.HasWeightBounds() {
		return WeightUnspecified
	}
	// Bandingkan pada presisi 0.01 kg (kolom NUMERIC(5,2)) agar tidak terpengaruh error float.
	a := int64(actual*100 + 0.5)
	if c.MaxWeight != nil && a > int64(*c.MaxWeight*100+0.5) {
		return WeightOver
	}
	if c.MinWeight != nil && a < int64(*c.MinWeight*100+0.5) {
		return WeightUnder
	}
	return WeightOK
}

// RequiredDocs berkas yang wajib terverifikasi sebelum atlet boleh ditimbang/bertanding.
// Regulasi IPSI (Pasal 2 butir 2 & 5) dan PERSILAT sama: bukti umur + surat sehat untuk semua
// atlet, ditambah surat pernyataan tidak hamil bagi putri Remaja & Dewasa.
func RequiredDocs(ageGroup, gender string) []string {
	docs := []string{DocIdentity, DocHealth}
	if NormalizeGender(gender) == "female" && (ageGroup == AgeRemaja || ageGroup == AgeDewasa) {
		docs = append(docs, DocPregnancy)
	}
	return docs
}

// MissingDocs mengembalikan berkas wajib yang belum dicentang.
func MissingDocs(ageGroup, gender string, docs map[string]bool) []string {
	var missing []string
	for _, d := range RequiredDocs(ageGroup, gender) {
		if !docs[d] {
			missing = append(missing, d)
		}
	}
	return missing
}

// CheckEligibility memvalidasi atlet terhadap kategori: gender dan golongan umur
// (dihitung dengan standar kategori pada tanggal acuan).
func CheckEligibility(a *Athlete, c *Category, ref time.Time) error {
	if a == nil || c == nil {
		return ErrInvalidInput
	}
	if c.Gender != "" && NormalizeGender(a.Gender) != NormalizeGender(c.Gender) {
		return fmt.Errorf("atlet %s tidak sesuai gender kategori %s", a.Name, c.Name)
	}
	if c.AgeGroup == "" {
		return nil
	}
	if a.BirthDate == "" {
		return errors.New("birth_date atlet wajib diisi untuk mendaftar ke kategori berbasis golongan umur")
	}
	birth, err := ParseBirthDate(a.BirthDate)
	if err != nil {
		return err
	}
	age := AgeAt(birth, ref)
	group := AgeGroupForAge(age, c.PresetStandard)
	if group != c.AgeGroup {
		return fmt.Errorf("usia atlet %d tahun tidak sesuai golongan umur kategori %s (%s)", age, c.Name, c.AgeGroup)
	}
	return nil
}
