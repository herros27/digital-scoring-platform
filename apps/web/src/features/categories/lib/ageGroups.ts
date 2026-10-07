/**
 * Golongan umur & aturan kelayakan kategori — cermin dari internal/domain/category_rules.go.
 * Usia dihitung dalam tahun penuh pada tanggal acuan (hari pertama kejuaraan).
 */

export type PresetStandard = 'ipsi' | 'persilat_2026'

export type AgeGroupCode =
  | 'pra_usia_dini'
  | 'usia_dini_1'
  | 'usia_dini_2'
  | 'pra_remaja'
  | 'remaja'
  | 'dewasa'
  | 'master_a'
  | 'master_b'
  | 'singa'
  | 'macan'
  | 'pre_teen'

export interface AgeBand {
  code: AgeGroupCode
  label: string
  /** Keterangan jenjang / rentang umur untuk UI. */
  hint: string
  minAge: number
  /** null = tanpa batas atas */
  maxAge: number | null
}

export const AGE_BANDS: Record<PresetStandard, AgeBand[]> = {
  ipsi: [
    { code: 'pra_usia_dini', label: 'Pra Usia Dini', hint: '≤ 5 th · TK / Festival', minAge: 0, maxAge: 5 },
    { code: 'usia_dini_1', label: 'Usia Dini 1', hint: '6–8 th · SD kelas awal', minAge: 6, maxAge: 8 },
    { code: 'usia_dini_2', label: 'Usia Dini 2', hint: '9–11 th · SD kelas 3–6', minAge: 9, maxAge: 11 },
    { code: 'pra_remaja', label: 'Pra-Remaja', hint: '12–14 th · SMP', minAge: 12, maxAge: 14 },
    { code: 'remaja', label: 'Remaja', hint: '15–17 th · SMA', minAge: 15, maxAge: 17 },
    { code: 'dewasa', label: 'Dewasa', hint: '18–35 th · Umum & Mahasiswa', minAge: 18, maxAge: 35 },
    { code: 'master_a', label: 'Master A', hint: '36–45 th', minAge: 36, maxAge: 45 },
    { code: 'master_b', label: 'Master B', hint: '> 45 th', minAge: 46, maxAge: null },
  ],
  persilat_2026: [
    { code: 'singa', label: 'Singa', hint: '3–6 th', minAge: 3, maxAge: 6 },
    { code: 'macan', label: 'Macan', hint: '7–9 th', minAge: 7, maxAge: 9 },
    { code: 'pre_teen', label: 'Usia Dini / Pre-Teen', hint: '10–11 th', minAge: 10, maxAge: 11 },
    { code: 'pra_remaja', label: 'Pra-Remaja', hint: '12–13 th', minAge: 12, maxAge: 13 },
    { code: 'remaja', label: 'Remaja', hint: '14–16 th', minAge: 14, maxAge: 16 },
    { code: 'dewasa', label: 'Dewasa', hint: '17–45 th', minAge: 17, maxAge: 45 },
  ],
}

export const PRESET_LABELS: Record<PresetStandard, string> = {
  ipsi: 'PB IPSI Terbaru',
  persilat_2026: 'PERSILAT 2026',
}

export function ageGroupLabel(code: string | undefined, standard: PresetStandard = 'ipsi'): string {
  if (!code) return '—'
  const band =
    AGE_BANDS[standard].find((b) => b.code === code) ??
    AGE_BANDS.ipsi.find((b) => b.code === code) ??
    AGE_BANDS.persilat_2026.find((b) => b.code === code)
  return band?.label ?? code
}

/** Usia tahun penuh pada tanggal acuan (keduanya YYYY-MM-DD / Date). null jika tanggal lahir tidak valid. */
export function ageOn(birthDate: string | Date, refDate: string | Date = new Date()): number | null {
  const birth = parseDate(birthDate)
  const ref = parseDate(refDate)
  if (!birth || !ref) return null
  let age = ref.getUTCFullYear() - birth.getUTCFullYear()
  const beforeBirthday =
    ref.getUTCMonth() < birth.getUTCMonth() ||
    (ref.getUTCMonth() === birth.getUTCMonth() && ref.getUTCDate() < birth.getUTCDate())
  if (beforeBirthday) age--
  return age
}

function parseDate(v: string | Date): Date | null {
  if (v instanceof Date) {
    return Number.isNaN(v.getTime())
      ? null
      : new Date(Date.UTC(v.getFullYear(), v.getMonth(), v.getDate()))
  }
  const m = /^(\d{4})-(\d{2})-(\d{2})/.exec(v ?? '')
  if (!m) return null
  const d = new Date(Date.UTC(Number(m[1]), Number(m[2]) - 1, Number(m[3])))
  // Tolak tanggal yang "meluap" (mis. 2010-02-30).
  if (d.getUTCMonth() !== Number(m[2]) - 1) return null
  return d
}

/** Golongan umur dari usia; null bila di luar semua band standar. */
export function ageGroupForAge(age: number, standard: PresetStandard = 'ipsi'): AgeGroupCode | null {
  if (age < 0) return null
  const band = AGE_BANDS[standard].find((b) => age >= b.minAge && (b.maxAge === null || age <= b.maxAge))
  return band?.code ?? null
}

export function ageGroupForBirthDate(
  birthDate: string,
  refDate: string | Date,
  standard: PresetStandard = 'ipsi',
): AgeGroupCode | null {
  const age = ageOn(birthDate, refDate)
  return age === null ? null : ageGroupForAge(age, standard)
}

export function normalizeGender(g: string | undefined): 'male' | 'female' | '' {
  const v = (g ?? '').toLowerCase().trim()
  if (v === 'male' || v === 'putra' || v === 'm') return 'male'
  if (v === 'female' || v === 'putri' || v === 'f') return 'female'
  return ''
}

export interface EligibilityCategory {
  gender: string
  age_group?: string
  preset_standard?: PresetStandard
}

/** Apakah atlet (gender + tanggal lahir) sah masuk kategori ini pada tanggal acuan. */
export function isEligible(
  athlete: { gender: string; birth_date?: string },
  category: EligibilityCategory,
  refDate: string | Date,
): boolean {
  if (normalizeGender(athlete.gender) !== normalizeGender(category.gender)) return false
  if (!category.age_group) return true
  if (!athlete.birth_date) return false
  return (
    ageGroupForBirthDate(athlete.birth_date, refDate, category.preset_standard ?? 'ipsi') ===
    category.age_group
  )
}

// ---------- Timbang badan & berkas ----------

export type WeightVerdict = 'ok' | 'overweight' | 'underweight' | 'unspecified'

/** Tanpa toleransi (0 kg). Batas min/max inklusif; perbandingan pada presisi 0.01 kg. */
export function checkWeight(
  actual: number,
  min: number | null | undefined,
  max: number | null | undefined,
): WeightVerdict {
  const hasMin = min !== null && min !== undefined
  const hasMax = max !== null && max !== undefined
  if (!hasMin && !hasMax) return 'unspecified'
  const a = Math.round(actual * 100)
  if (hasMax && a > Math.round((max as number) * 100)) return 'overweight'
  if (hasMin && a < Math.round((min as number) * 100)) return 'underweight'
  return 'ok'
}

export type DocKey = 'identity' | 'parental' | 'health'

export const DOC_CHECKLIST: { key: DocKey; label: string }[] = [
  { key: 'identity', label: 'Akta Kelahiran / KIA / Kartu Keluarga terverifikasi' },
  { key: 'parental', label: 'Surat Izin Orang Tua / Wali valid' },
  { key: 'health', label: 'Surat Keterangan Sehat Dokter ada' },
]

/** Berkas wajib: seluruh golongan di bawah Dewasa. Dewasa & Master tidak diwajibkan. */
export function requiredDocs(ageGroup: string | undefined): DocKey[] {
  if (!ageGroup || ['dewasa', 'master_a', 'master_b'].includes(ageGroup)) return []
  return DOC_CHECKLIST.map((d) => d.key)
}

export function missingDocs(ageGroup: string | undefined, docs: Partial<Record<string, boolean>> | undefined): DocKey[] {
  return requiredDocs(ageGroup).filter((k) => !docs?.[k])
}
