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
  created_at: string
  // UI extended metadata
  quota?: number
  participant_count?: number
  status?: 'ready' | 'draft' | 'in_progress' | 'completed'
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

/** Tabel resmi per golongan umur → kelas bobot per gender. Golongan tanpa tabel resmi tidak ada di sini. */
const WEIGHT_TABLES: Partial<Record<AgeGroupCode, Record<Sex, WeightClassDef[]>>> = (() => {
  // Usia Dini 1 & 2 (SD): A 26–28 … S (>62–64), Open >64–68 — sama untuk putra/putri.
  const sd = [...stepClasses(26, 2, 19), open('OPEN', 64, 68)]
  // Pra-Remaja (SMP): A 30–33 … P (>75–78), Open >78–84.
  const smp = [...stepClasses(30, 3, 16), open('OPEN', 78, 84)]
  return {
    usia_dini_1: { male: sd, female: sd },
    usia_dini_2: { male: sd, female: sd },
    pra_remaja: { male: smp, female: smp },
    remaja: {
      male: [
        under(39),
        ...stepClasses(39, 4, 12),
        open('OPEN1', 87, 100),
        open('OPEN2', 100, null),
      ],
      female: [
        under(39),
        ...stepClasses(39, 4, 10),
        open('OPEN1', 79, 92),
        open('OPEN2', 92, null),
      ],
    },
    dewasa: {
      male: [
        under(45),
        ...stepClasses(45, 5, 10),
        open('OPEN1', 95, 110),
        open('OPEN2', 110, null),
      ],
      female: [
        under(45),
        ...stepClasses(45, 5, 8),
        open('OPEN1', 85, 100),
        open('OPEN2', 100, null),
      ],
    },
  }
})()

/** Pra Usia Dini & Usia Dini 1 umumnya memakai bagan pemasalan (mini bagan 4). */
const PEMASALAN_GROUPS: AgeGroupCode[] = ['pra_usia_dini', 'usia_dini_1']

const SEX_LABEL: Record<Sex, string> = { male: 'Putra', female: 'Putri' }

/**
 * Bangun preset kategori untuk satu standar. Golongan tanpa tabel bobot resmi
 * (Pra Usia Dini, Master, serta golongan PERSILAT) mendapat satu kategori "Kelas Bebas"
 * per gender tanpa batas bobot, yang dapat diubah panitia.
 */
export function buildPresets(
  standard: PresetStandard,
  ageGroups?: AgeGroupCode[],
): StandardClassPreset[] {
  const out: StandardClassPreset[] = []
  for (const band of AGE_BANDS[standard]) {
    if (ageGroups && !ageGroups.includes(band.code)) continue
    const bracketMode: BracketMode =
      standard === 'ipsi' && PEMASALAN_GROUPS.includes(band.code)
        ? 'pemasalan_4'
        : 'single_elimination'
    const table = standard === 'ipsi' ? WEIGHT_TABLES[band.code] : undefined
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
          bracketMode,
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
