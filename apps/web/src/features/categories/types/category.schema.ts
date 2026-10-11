import { z } from 'zod'
import { AGE_BANDS, ageGroupLabel, type AgeGroupCode, type PresetStandard } from '../lib/ageGroups'

export type BracketMode = 'single_elimination' | 'pemasalan_4'

export const BRACKET_MODE_LABELS: Record<BracketMode, string> = {
  single_elimination: 'Gugur Tunggal (Prestasi)',
  pemasalan_4: 'Pemasalan — Mini Bagan 4',
}

const ageGroupCodes = [
  'pra_usia_dini',
  'usia_dini_1',
  'usia_dini_2',
  'pra_remaja',
  'remaja',
  'dewasa',
  'master_a',
  'master_b',
  'singa',
  'macan',
  'pre_teen',
] as const

export const createCategorySchema = z
  .object({
    name: z.string().min(2, 'Nama kategori minimal 2 karakter').max(100, 'Maksimal 100 karakter'),
    gender: z.enum(['male', 'female', 'putra', 'putri']),
    weight_class: z.string().min(1, 'Kelas tanding wajib dipilih'),
    ruleset_id: z.string().optional(),
    age_group: z.enum(ageGroupCodes).default('dewasa'),
    preset_standard: z.enum(['ipsi', 'persilat_2026']).default('ipsi'),
    min_weight: z.number().min(0, 'Bobot minimal tidak boleh negatif').nullable().optional(),
    max_weight: z.number().positive('Bobot maksimal harus positif').nullable().optional(),
    bracket_mode: z.enum(['single_elimination', 'pemasalan_4']).default('single_elimination'),
  })
  .refine(
    (v) =>
      v.min_weight == null ||
      v.max_weight == null ||
      (v.min_weight as number) <= (v.max_weight as number),
    { message: 'Bobot minimal tidak boleh melebihi bobot maksimal', path: ['max_weight'] },
  )

export const categoryFilterSchema = z.object({
  gender: z.enum(['all', 'male', 'female', 'putra', 'putri']).default('all'),
  search: z.string().optional(),
})

export type CreateCategoryFormValues = z.input<typeof createCategorySchema>
export type CategoryFilterValues = z.infer<typeof categoryFilterSchema>

export interface Category {
  id: string
  competition_id: string
  name: string
  gender: 'male' | 'female' | 'putra' | 'putri'
  weight_class: string
  ruleset_id?: string
  age_group?: AgeGroupCode
  preset_standard?: PresetStandard
  min_weight?: number | null
  max_weight?: number | null
  bracket_mode?: BracketMode
  /** Draw bagan dikunci panitia (disimpan di server): bagan tidak bisa diacak ulang */
  bracket_locked?: boolean
  /** Gelanggang asal: semua partai kategori dimainkan di sini. Kosong sampai bagan pertama dibuat atau dipilih panitia. */
  home_arena_id?: string | null
  bracket_locked_at?: string
  created_at: string
  // UI extended metadata
  quota?: number
  participant_count?: number
  status?: 'ready' | 'draft' | 'in_progress' | 'completed'
}

/** Penetapan ruleset khusus: pilih salah satu age_group atau category_id. */
export interface RulesetAssignment {
  age_group?: string
  category_id?: string
  ruleset_id: string | null
}

export interface StandardClassPreset {
  code: string
  label: string
  /** null = tanpa batas bawah (kelas "Under") */
  minWeight: number | null
  /** null = tanpa batas atas (kelas OPEN terakhir) */
  maxWeight: number | null
  gender: 'male' | 'female'
  categoryName: string
  ageGroup: AgeGroupCode
  standard: PresetStandard
  bracketMode: BracketMode
  /** Catatan pasangan/aturan khusus golongan (mis. golongan tanpa kelas bobot). */
  note?: string
}

// ---------- Tabel kelas bobot resmi PB IPSI ----------

const LETTERS = 'ABCDEFGHIJKLMNOPQRSTUVWXYZ'

interface WeightClassDef {
  code: string
  label: string
  min: number | null
  max: number | null
}

const kg = (n: number) => `${n}`.replace('.', ',')

/**
 * Kelas bertingkat A, B, C… dengan lebar `step` kg mulai `start`.
 * Kelas A = start–(start+step); selanjutnya >lower–upper.
 */
function stepClasses(start: number, step: number, count: number): WeightClassDef[] {
  return Array.from({ length: count }, (_, i) => {
    const lo = start + i * step
    const hi = lo + step
    return {
      code: LETTERS[i],
      label:
        i === 0
          ? `Kelas ${LETTERS[i]} (${kg(lo)}–${kg(hi)} Kg)`
          : `Kelas ${LETTERS[i]} (>${kg(lo)}–${kg(hi)} Kg)`,
      min: lo,
      max: hi,
    }
  })
}

const under = (limit: number): WeightClassDef => ({
  code: 'UNDER',
  label: `Under ${limit} Kg`,
  min: null,
  max: limit,
})

const open = (code: string, lo: number, hi: number | null): WeightClassDef => ({
  code,
  label: hi === null ? `Kelas Open (>${kg(lo)} Kg)` : `Kelas Open (>${kg(lo)}–${kg(hi)} Kg)`,
  min: lo,
  max: hi,
})

type Sex = 'male' | 'female'

// Tabel kelas bobot resmi — identik pada regulasi PB IPSI 2025 (Pasal 3–7) dan
// IPSF-PERSILAT Feb 2026 (Pasal 2 butir 3).
const SD_TABLE: WeightClassDef[] = [...stepClasses(26, 2, 19), open('OPEN', 64, 68)] // A–S + Open
const SMP_TABLE: WeightClassDef[] = [...stepClasses(30, 3, 16), open('OPEN', 78, 84)] // A–P + Open
const REMAJA_TABLE: Record<Sex, WeightClassDef[]> = {
  male: [under(39), ...stepClasses(39, 4, 12), open('OPEN1', 87, 100), open('OPEN2', 100, null)],
  female: [under(39), ...stepClasses(39, 4, 10), open('OPEN1', 79, 92), open('OPEN2', 92, null)],
}
const DEWASA_TABLE: Record<Sex, WeightClassDef[]> = {
  male: [under(45), ...stepClasses(45, 5, 10), open('OPEN1', 95, 110), open('OPEN2', 110, null)],
  female: [under(45), ...stepClasses(45, 5, 8), open('OPEN1', 85, 100), open('OPEN2', 100, null)],
}
const both = (t: WeightClassDef[]): Record<Sex, WeightClassDef[]> => ({ male: t, female: t })

/** Golongan yang memakai tabel kelas bobot resmi, per standar. */
const WEIGHT_TABLES: Record<
  PresetStandard,
  Partial<Record<AgeGroupCode, Record<Sex, WeightClassDef[]>>>
> = {
  ipsi: {
    usia_dini_2: both(SD_TABLE),
    pra_remaja: both(SMP_TABLE),
    remaja: REMAJA_TABLE,
    dewasa: DEWASA_TABLE,
    // Master A & B memakai tabel yang sama dengan Dewasa (13 putra + 11 putri).
    master_a: DEWASA_TABLE,
    master_b: DEWASA_TABLE,
  },
  persilat_2026: {
    pre_teen: both(SD_TABLE),
    pra_remaja: both(SMP_TABLE),
    remaja: REMAJA_TABLE,
    dewasa: DEWASA_TABLE,
  },
}

/**
 * Golongan tanpa kelas bobot: atlet dipasangkan menurut pedoman selisih. Dibuat satu
 * "Kelas Bebas" per gender tanpa batas bobot (tanpa kewajiban timbang).
 */
const MATCHING_GUIDELINES: Record<PresetStandard, Partial<Record<AgeGroupCode, string>>> = {
  ipsi: {
    usia_dini_1:
      'Tanpa kelas bobot & tanpa timbang. Pasangan: selisih usia 1 th, tinggi 3 cm, berat 1 kg.',
  },
  persilat_2026: {
    singa: 'Tanpa kelas bobot. Pasangan: selisih usia 1 th, tinggi 3 cm, berat 2 kg.',
    macan: 'Tanpa kelas bobot. Pasangan: selisih usia 1 th, tinggi 3 cm, berat 2 kg.',
    master_a: 'Tanpa kelas bobot. Pasangan: selisih berat 5 kg.',
    master_b: 'Tanpa kelas bobot. Pasangan: selisih berat 5 kg.',
  },
}

/** Golongan yang tidak mempertandingkan kategori Tanding (hanya Jurus) menurut regulasi. */
const NO_TANDING: Record<PresetStandard, AgeGroupCode[]> = {
  ipsi: ['pra_usia_dini'],
  persilat_2026: [],
}

const SEX_LABEL: Record<Sex, string> = { male: 'Putra', female: 'Putri' }

/** Catatan golongan untuk UI (pedoman pasangan / hanya Jurus), atau undefined. */
export function groupNote(standard: PresetStandard, code: AgeGroupCode): string | undefined {
  if (NO_TANDING[standard].includes(code)) {
    return 'Hanya kategori Jurus (Jurus Tunggal Bebas) — tidak ada kelas Tanding.'
  }
  return MATCHING_GUIDELINES[standard][code]
}

/** True bila golongan tidak punya kategori Tanding sama sekali. */
export function isJurusOnly(standard: PresetStandard, code: AgeGroupCode): boolean {
  return NO_TANDING[standard].includes(code)
}

/**
 * Bangun preset kategori Tanding untuk satu standar sesuai regulasi resmi. Golongan tanpa kelas
 * bobot mendapat satu "Kelas Bebas" per gender; golongan Jurus-saja tidak mendapat preset.
 */
export function buildPresets(
  standard: PresetStandard,
  ageGroups?: AgeGroupCode[],
): StandardClassPreset[] {
  const out: StandardClassPreset[] = []
  for (const band of AGE_BANDS[standard]) {
    if (ageGroups && !ageGroups.includes(band.code)) continue
    if (NO_TANDING[standard].includes(band.code)) continue
    const table = WEIGHT_TABLES[standard][band.code]
    const note = MATCHING_GUIDELINES[standard][band.code]
    for (const sex of ['male', 'female'] as Sex[]) {
      const classes: WeightClassDef[] = table?.[sex] ?? [
        { code: 'BEBAS', label: 'Kelas Bebas', min: null, max: null },
      ]
      for (const c of classes) {
        out.push({
          code: c.code,
          label: c.label,
          minWeight: c.min,
          maxWeight: c.max,
          gender: sex,
          ageGroup: band.code,
          standard,
          // Regulasi memakai sistem gugur; mode pemasalan 4 hanya pilihan manual panitia.
          bracketMode: 'single_elimination',
          note,
          categoryName: `Tanding ${ageGroupLabel(band.code, standard)} ${SEX_LABEL[sex]} - ${c.label}`,
        })
      }
    }
  }
  return out
}

export const IPSI_STANDARD_PRESETS: StandardClassPreset[] = buildPresets('ipsi')
export const PERSILAT_2026_PRESETS: StandardClassPreset[] = buildPresets('persilat_2026')

export function presetsFor(
  standard: PresetStandard,
  ageGroups?: AgeGroupCode[],
): StandardClassPreset[] {
  return buildPresets(standard, ageGroups)
}

/** Bangun payload create-category dari satu preset. */
export function presetToPayload(p: StandardClassPreset): CreateCategoryFormValues {
  return {
    name: p.categoryName,
    gender: p.gender,
    weight_class: p.code,
    age_group: p.ageGroup,
    preset_standard: p.standard,
    min_weight: p.minWeight,
    max_weight: p.maxWeight,
    bracket_mode: p.bracketMode,
  }
}

/**
 * Nama kategori untuk perbandingan: huruf kecil, spasi ganda dirapikan, dan semua jenis tanda hubung (- – — −) disamakan.
 * Tanpa ini "45-50 Kg" dan "45–50 Kg" dianggap kategori berbeda dan peserta terbelah di antara keduanya.
 */
export function normalizeCategoryName(name: string): string {
  return name
    .replace(/[‐-―−]/g, '-')
    .split(/\s+/)
    .filter(Boolean)
    .join(' ')
    .toLowerCase()
}
