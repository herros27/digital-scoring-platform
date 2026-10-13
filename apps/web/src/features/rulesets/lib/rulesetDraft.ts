import type {
  RulesetDTO,
  RulesetOutcome,
  RulesetPayload,
  RulesetPenalty,
  RulesetScoring,
  RulesetTechnique,
} from '../types'
import { AGE_BANDS } from '@/features/categories/lib/ageGroups'

/** Kode jenis hasil khusus yang dikenal server. */
export const OUTCOME_TYPES = ['ko', 'tko', 'walkover', 'withdrawal'] as const
export const TIEBREAK_METHODS = ['extra_round', 'lighter_weight', 'coin_toss'] as const

export const PHASE_LABEL: Record<RulesetOutcome['phase'], string> = {
  before_start: 'Hanya sebelum dimulai',
  in_match: 'Hanya setelah dimulai',
  any: 'Kapan saja',
}

export const OUTCOME_LABEL: Record<string, string> = {
  ko: 'Menang Mutlak (KO)',
  tko: 'Menang Teknik (TKO)',
  walkover: 'Menang WO',
  withdrawal: 'Undur Diri',
}

export const METHOD_LABEL: Record<string, string> = {
  extra_round: 'Babak tambahan',
  lighter_weight: 'Pesilat lebih ringan (timbang ulang)',
  coin_toss: 'Undian lempar koin',
}

export interface DraftPenalty {
  type: string
  label: string
  points: number
  disqualify: boolean
}

/** auto = server memilih menurut golongan umur kategori; custom = nilai di bawah; off = WMP dimatikan. */
export type PointGapMode = 'auto' | 'custom' | 'off'

export const POINT_GAP_MODE_LABEL: Record<PointGapMode, string> = {
  auto: 'Otomatis menurut golongan umur',
  custom: 'Atur sendiri',
  off: 'Nonaktif',
}

export interface DraftOutcome {
  type: string
  label: string
  phase: RulesetOutcome['phase']
  enabled: boolean
}

/** Waktu pertandingan khusus satu golongan umur; hanya dikirim ke server bila `enabled`. */
export interface DraftAgeTiming {
  code: string
  label: string
  enabled: boolean
  roundDuration: number
  breakDuration: number
  totalRounds: number
}

/** Usulan IPSI untuk Usia Dini (2 babak × 1,5 menit, istirahat 1 menit); hanya pengisi cepat, nilainya bebas diubah. */
export const IPSI_YOUTH_SUGGESTION = { roundDuration: 90, breakDuration: 60, totalRounds: 2 }
export const IPSI_YOUTH_CODES = ['usia_dini_1', 'usia_dini_2']

/** Semua golongan umur yang dikenal (gabungan standar IPSI dan PERSILAT), tanpa duplikat. */
export function ageGroupOptions(): { code: string; label: string }[] {
  const seen = new Set<string>()
  const out: { code: string; label: string }[] = []
  for (const std of ['ipsi', 'persilat_2026'] as const) {
    for (const b of AGE_BANDS[std]) {
      if (seen.has(b.code)) continue
      seen.add(b.code)
      out.push({ code: b.code, label: b.label })
    }
  }
  return out
}

function defaultAgeTimings(
  main: { roundDuration: number; breakDuration: number; totalRounds: number },
  saved?: RulesetScoring['age_timings'],
): DraftAgeTiming[] {
  return ageGroupOptions().map(({ code, label }) => {
    const t = saved?.[code]
    return t
      ? {
          code,
          label,
          enabled: true,
          roundDuration: t.round_duration,
          breakDuration: t.break_duration,
          totalRounds: t.total_rounds,
        }
      : { code, label, enabled: false, ...main }
  })
}

/** Keadaan formulir editor: semua bagian eksplisit (tidak ada "kosong = bawaan"). */
export interface RulesetDraft {
  id?: string
  name: string
  version: string
  status: string
  roundDuration: number
  breakDuration: number
  totalRounds: number
  techniques: RulesetTechnique[]
  penalties: DraftPenalty[]
  consensus: { judges: number; majority: number; windowMs: number }
  tiebreak: {
    fewestPenalties: boolean
    techniquePriority: boolean
    /** kode teknik dari yang paling bernilai */
    priority: string[]
    manual: string[]
  }
  outcomes: DraftOutcome[]
  pointGap: { mode: PointGapMode; margin: number; fromRound: number }
  /** pengecualian waktu per golongan umur (bebas ditentukan panitia) */
  ageTimings: DraftAgeTiming[]
  /** nilai skalar warisan yang dipertahankan saat menyimpan */
  legacy: { maxPenalties: number; penaltyThreshold: number }
}

export const defaultTechniques = (): RulesetTechnique[] => [
  { type: 'technique', label: 'Pukulan', points: 1 },
  { type: 'knockdown', label: 'Tendangan', points: 2 },
  { type: 'fall', label: 'Jatuhan', points: 3 },
  { type: 'catch_fall', label: 'Tangkapan + Jatuhan', points: 4 },
  { type: 'combo_1_2', label: 'Kombinasi Pukulan + Tendangan', points: 3 },
  { type: 'combo_1_1', label: 'Kombinasi 2 Pukulan', points: 2 },
]

export const defaultPenalties = (): DraftPenalty[] => [
  { type: 'binaan_1', label: 'Binaan 1', points: 0, disqualify: false },
  { type: 'binaan_2', label: 'Binaan 2', points: 0, disqualify: false },
  { type: 'teguran_1', label: 'Teguran 1', points: 1, disqualify: false },
  { type: 'teguran_2', label: 'Teguran 2', points: 2, disqualify: false },
  { type: 'peringatan_1', label: 'Peringatan 1', points: 5, disqualify: false },
  { type: 'peringatan_2', label: 'Peringatan 2', points: 10, disqualify: false },
  { type: 'peringatan_3', label: 'Peringatan 3', points: 0, disqualify: true },
]

export const defaultOutcomes = (): DraftOutcome[] => [
  { type: 'ko', label: OUTCOME_LABEL.ko, phase: 'in_match', enabled: true },
  { type: 'tko', label: OUTCOME_LABEL.tko, phase: 'in_match', enabled: true },
  { type: 'walkover', label: OUTCOME_LABEL.walkover, phase: 'before_start', enabled: true },
  { type: 'withdrawal', label: OUTCOME_LABEL.withdrawal, phase: 'any', enabled: true },
]

/** Urutan prioritas bawaan: teknik dari nilai tertinggi ke terendah (urutan input dipertahankan bila sama). */
export function priorityByValue(techniques: RulesetTechnique[]): string[] {
  return techniques
    .map((t, i) => ({ t, i }))
    .sort((a, b) => b.t.points - a.t.points || a.i - b.i)
    .map(({ t }) => t.type)
}

/** Prioritas yang konsisten dengan daftar teknik: buang kode yang sudah dihapus, tambahkan yang baru di akhir. */
export function normalizedPriority(d: Pick<RulesetDraft, 'techniques' | 'tiebreak'>): string[] {
  const codes = d.techniques.map((t) => t.type)
  const kept = d.tiebreak.priority.filter((c, i, arr) => codes.includes(c) && arr.indexOf(c) === i)
  return [...kept, ...codes.filter((c) => !kept.includes(c))]
}

export function newDraft(): RulesetDraft {
  const techniques = defaultTechniques()
  return {
    name: '',
    version: '1.0',
    status: 'active',
    roundDuration: 120,
    breakDuration: 60,
    totalRounds: 3,
    techniques,
    penalties: defaultPenalties(),
    consensus: { judges: 3, majority: 2, windowMs: 3000 },
    tiebreak: {
      fewestPenalties: true,
      techniquePriority: true,
      priority: priorityByValue(techniques),
      manual: [...TIEBREAK_METHODS],
    },
    outcomes: defaultOutcomes(),
    pointGap: { mode: 'auto', margin: 30, fromRound: 2 },
    ageTimings: defaultAgeTimings({ roundDuration: 120, breakDuration: 60, totalRounds: 3 }),
    legacy: { maxPenalties: 3, penaltyThreshold: 1 },
  }
}

/** Ubah ruleset API menjadi draft; bagian yang kosong diisi bawaan yang sama dengan server. */
export function draftFromRuleset(r: RulesetDTO): RulesetDraft {
  const s = r.scoring
  const base = newDraft()

  const techniques: RulesetTechnique[] = s.techniques?.length
    ? s.techniques.map((t) => ({ ...t }))
    : [
        { type: 'technique', label: 'Pukulan', points: s.technique_points || 1 },
        { type: 'knockdown', label: 'Tendangan', points: s.knockdown_points || 2 },
        { type: 'fall', label: 'Jatuhan', points: s.fall_points || 3 },
      ]
  const penalties: DraftPenalty[] = s.penalties?.length
    ? s.penalties.map((p) => ({
        type: p.type,
        label: p.label,
        points: p.points ?? 0,
        disqualify: p.action === 'disqualify',
      }))
    : base.penalties

  const steps = s.tiebreak?.steps?.length
    ? s.tiebreak.steps
    : ['fewest_penalties', 'technique_priority']
  const configured = s.outcomes?.length ? s.outcomes : null
  const outcomes: DraftOutcome[] = defaultOutcomes().map((def) => {
    const found = configured?.find((o) => o.type === def.type)
    if (!configured) return def
    return found
      ? { type: found.type, label: found.label, phase: found.phase, enabled: true }
      : { ...def, enabled: false }
  })

  const draft: RulesetDraft = {
    id: r.id,
    name: r.name,
    version: r.version,
    status: r.status,
    roundDuration: s.round_duration,
    breakDuration: s.break_duration,
    totalRounds: s.total_rounds,
    techniques,
    penalties,
    consensus: {
      judges: s.consensus?.judges || 3,
      majority: s.consensus?.majority || 2,
      windowMs: s.consensus?.window_ms || 3000,
    },
    tiebreak: {
      fewestPenalties: steps.includes('fewest_penalties'),
      techniquePriority: steps.includes('technique_priority'),
      priority: s.tiebreak?.technique_priority?.length
        ? [...s.tiebreak.technique_priority]
        : priorityByValue(techniques),
      manual: s.tiebreak?.manual_methods?.length
        ? [...s.tiebreak.manual_methods]
        : [...TIEBREAK_METHODS],
    },
    outcomes,
    pointGap: s.point_gap
      ? {
          mode: s.point_gap.margin === 0 ? 'off' : 'custom',
          margin: s.point_gap.margin || base.pointGap.margin,
          fromRound: s.point_gap.from_round || base.pointGap.fromRound,
        }
      : base.pointGap,
    ageTimings: defaultAgeTimings(
      {
        roundDuration: s.round_duration,
        breakDuration: s.break_duration,
        totalRounds: s.total_rounds,
      },
      s.age_timings,
    ),
    legacy: { maxPenalties: s.max_penalties ?? 3, penaltyThreshold: s.penalty_threshold ?? 1 },
  }
  draft.tiebreak.priority = normalizedPriority(draft)
  return draft
}

/** Salinan untuk dibuat sebagai ruleset baru (tanpa id, nama diberi akhiran). */
export function copyDraft(d: RulesetDraft): RulesetDraft {
  const copy: RulesetDraft = JSON.parse(JSON.stringify(d))
  delete copy.id
  copy.name = `${d.name} (salinan)`
  copy.version = '1.0'
  copy.status = 'active'
  return copy
}

const pointsOf = (techs: RulesetTechnique[], type: string, fallback: number) =>
  techs.find((t) => t.type === type)?.points ?? fallback

/** Bangun badan permintaan API dari draft (bagian eksplisit, nilai skalar warisan diturunkan dari teknik). */
export function payloadFromDraft(d: RulesetDraft): RulesetPayload {
  const steps: string[] = []
  if (d.tiebreak.fewestPenalties) steps.push('fewest_penalties')
  if (d.tiebreak.techniquePriority) steps.push('technique_priority')

  const penalties: RulesetPenalty[] = d.penalties.map((p) => ({
    type: p.type.trim(),
    label: p.label.trim(),
    points: p.points,
    ...(p.disqualify ? { action: 'disqualify' as const } : {}),
  }))
  const outcomes: RulesetOutcome[] = d.outcomes
    .filter((o) => o.enabled)
    .map((o) => ({ type: o.type, label: o.label.trim(), phase: o.phase }))

  const scoring: RulesetScoring = {
    technique_points: pointsOf(d.techniques, 'technique', 1),
    knockdown_points: pointsOf(d.techniques, 'knockdown', 2),
    fall_points: pointsOf(d.techniques, 'fall', 3),
    max_penalties: d.legacy.maxPenalties,
    penalty_threshold: d.legacy.penaltyThreshold,
    round_duration: d.roundDuration,
    break_duration: d.breakDuration,
    total_rounds: d.totalRounds,
    techniques: d.techniques.map((t) => ({
      type: t.type.trim(),
      label: t.label.trim(),
      points: t.points,
    })),
    penalties,
    consensus: {
      judges: d.consensus.judges,
      majority: d.consensus.majority,
      window_ms: d.consensus.windowMs,
    },
    tiebreak: {
      steps,
      technique_priority: normalizedPriority(d),
      manual_methods: [...d.tiebreak.manual],
    },
    outcomes,
    ...(d.pointGap.mode === 'custom'
      ? { point_gap: { margin: d.pointGap.margin, from_round: d.pointGap.fromRound } }
      : d.pointGap.mode === 'off'
        ? { point_gap: { margin: 0 } }
        : {}),
    ...(d.ageTimings.some((a) => a.enabled)
      ? {
          age_timings: Object.fromEntries(
            d.ageTimings
              .filter((a) => a.enabled)
              .map((a) => [
                a.code,
                {
                  round_duration: a.roundDuration,
                  break_duration: a.breakDuration,
                  total_rounds: a.totalRounds,
                },
              ]),
          ),
        }
      : {}),
  }
  return { name: d.name.trim(), version: d.version.trim(), status: d.status, scoring }
}

const CODE_RE = /^[a-z][a-z0-9_]{0,31}$/

/**
 * Validasi sisi klien (mencerminkan `ScoringRule.Validate` di server). Kunci galat memakai jalur
 * bidang, mis. `name`, `techniques.2.points`, `consensus.majority`.
 */
export function validateDraft(d: RulesetDraft): Record<string, string> {
  const e: Record<string, string> = {}
  if (d.name.trim().length < 2) e.name = 'Nama minimal 2 karakter'
  if (!d.version.trim()) e.version = 'Versi wajib diisi'
  if (!Number.isInteger(d.roundDuration) || d.roundDuration <= 0)
    e.roundDuration = 'Durasi babak harus lebih dari 0 detik'
  if (!Number.isInteger(d.breakDuration) || d.breakDuration < 0)
    e.breakDuration = 'Durasi istirahat tidak boleh negatif'
  if (!Number.isInteger(d.totalRounds) || d.totalRounds < 1 || d.totalRounds > 3)
    e.totalRounds = 'Jumlah babak 1-3'

  if (!d.techniques.length) e.techniques = 'Minimal satu teknik'
  const techCodes = new Set<string>()
  d.techniques.forEach((t, i) => {
    const code = t.type.trim()
    if (!CODE_RE.test(code))
      e[`techniques.${i}.type`] = 'Kode: huruf kecil/angka/underscore, diawali huruf'
    else if (code === 'undo' || code === 'penalty')
      e[`techniques.${i}.type`] = 'Kode dicadangkan sistem'
    else if (techCodes.has(code)) e[`techniques.${i}.type`] = 'Kode ganda'
    techCodes.add(code)
    if (!t.label.trim()) e[`techniques.${i}.label`] = 'Label wajib diisi'
    if (!Number.isInteger(t.points) || t.points <= 0)
      e[`techniques.${i}.points`] = 'Nilai harus lebih dari 0'
  })

  const penCodes = new Set<string>()
  d.penalties.forEach((p, i) => {
    const code = p.type.trim()
    if (!CODE_RE.test(code))
      e[`penalties.${i}.type`] = 'Kode: huruf kecil/angka/underscore, diawali huruf'
    else if (penCodes.has(code)) e[`penalties.${i}.type`] = 'Kode ganda'
    penCodes.add(code)
    if (!p.label.trim()) e[`penalties.${i}.label`] = 'Label wajib diisi'
    if (!Number.isInteger(p.points) || p.points < 0)
      e[`penalties.${i}.points`] = 'Potongan tidak boleh negatif'
  })

  const c = d.consensus
  if (!Number.isInteger(c.judges) || c.judges < 1 || c.judges > 5)
    e['consensus.judges'] = 'Jumlah juri 1-5'
  if (!Number.isInteger(c.majority) || c.majority < 1 || c.majority > c.judges)
    e['consensus.majority'] = 'Mayoritas 1 sampai jumlah juri'
  if (!Number.isInteger(c.windowMs) || c.windowMs < 500 || c.windowMs > 10000)
    e['consensus.windowMs'] = 'Window 500-10000 ms'

  d.outcomes.forEach((o, i) => {
    if (o.enabled && !o.label.trim()) e[`outcomes.${i}.label`] = 'Label wajib diisi'
  })

  if (d.pointGap.mode === 'custom') {
    if (!Number.isInteger(d.pointGap.margin) || d.pointGap.margin < 1)
      e['pointGap.margin'] = 'Selisih minimal 1 poin (pilih Nonaktif untuk mematikan)'
    if (
      !Number.isInteger(d.pointGap.fromRound) ||
      d.pointGap.fromRound < 1 ||
      d.pointGap.fromRound > d.totalRounds
    )
      e['pointGap.fromRound'] = 'Babak mulai 1 sampai jumlah babak'
  }

  d.ageTimings.forEach((a, i) => {
    if (!a.enabled) return
    if (!Number.isInteger(a.roundDuration) || a.roundDuration <= 0)
      e[`ageTimings.${i}.roundDuration`] = 'Durasi babak harus lebih dari 0 detik'
    if (!Number.isInteger(a.breakDuration) || a.breakDuration < 0)
      e[`ageTimings.${i}.breakDuration`] = 'Istirahat tidak boleh negatif'
    if (!Number.isInteger(a.totalRounds) || a.totalRounds < 1 || a.totalRounds > 3)
      e[`ageTimings.${i}.totalRounds`] = 'Jumlah babak 1-3'
  })
  return e
}

/** Ringkasan satu baris untuk kartu daftar ruleset. */
export function summarize(r: RulesetDTO): string {
  const s = r.scoring
  const judges = s.consensus?.judges ?? 3
  const majority = s.consensus?.majority ?? 2
  const techniques = s.techniques?.length ?? 3
  const penalties = s.penalties?.length ?? 7
  const extra = Object.keys(s.age_timings ?? {}).length
  const age = extra ? ` • ${extra} golongan umur berwaktu khusus` : ''
  return `${s.total_rounds} babak × ${Math.round((s.round_duration / 60) * 10) / 10} mnt • ${judges} juri (${majority} sepakat) • ${techniques} teknik • ${penalties} sanksi${age}`
}
