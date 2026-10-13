import type { RefereeConsole } from '../composables/useRefereeConsole'
import { TIEBREAK_METHOD_LABEL } from '@/composables/matchResult'
import { TARGET_LABEL } from '../lib/matchLog'

/** Permintaan dialog konfirmasi dari panel ke cangkang konsol. */
export default interface ConfirmRequest {
  title: string
  message: string
  confirmLabel: string
  /** true untuk aksi yang tidak bisa dibatalkan (tombol merah) */
  danger?: boolean
  /** Bila ada, pengguna wajib memilih salah satu sebelum dapat mengonfirmasi. */
  choices?: Array<{ value: string; label: string }>
  /** Pilihan kedua (mis. dasar keputusan); wajib dipilih bila diisi. */
  methods?: Array<{ value: string; label: string }>
  /**
   * Menerima pilihan (jika `choices`/`methods` diisi). Bila mengembalikan ConfirmRequest,
   * dialog berikutnya langsung dibuka (mis. dari aturan seri otomatis ke keputusan manual).
   */
  action: (choice?: string, method?: string) => unknown
}

function isRequest(v: unknown): v is ConfirmRequest {
  return !!v && typeof v === 'object' && 'title' in v && 'confirmLabel' in v
}
export { isRequest as isConfirmRequest }

const fighterChoices = (c: RefereeConsole) =>
  (['red', 'blue'] as const).map((t) => ({
    value: t,
    label:
      `Sudut ${TARGET_LABEL[t]} — ${(t === 'red' ? c.redCompetitor : c.blueCompetitor)?.name ?? ''}`.trim(),
  }))

/** Seri yang tidak terselesaikan aturan otomatis: wasit memilih dasar keputusan dan pemenang. */
export function manualTieRequest(c: RefereeConsole): ConfirmRequest {
  const manual = c.tiebreak?.manual_methods ?? ['extra_round', 'lighter_weight', 'coin_toss']
  return {
    title: 'Seri Penuh — Tentukan Pemenang',
    message:
      'Skor, sanksi, dan teknik kedua sudut sama. Pilih dasar keputusan (babak tambahan, timbang ulang, undian koin, atau keputusan operator meja) lalu pemenangnya.',
    confirmLabel: 'Tetapkan Pemenang & Selesaikan',
    methods: ['referee_decision', ...manual].map((m) => ({
      value: m,
      label: TIEBREAK_METHOD_LABEL[m] ?? m,
    })),
    choices: fighterChoices(c),
    action: (choice, method) =>
      c.finish(choice === 'red' || choice === 'blue' ? choice : undefined, method),
  }
}

/**
 * Dialog "Selesaikan Pertandingan". Saat skor seri, server menerapkan aturan seri otomatis
 * (sanksi paling sedikit, lalu teknik terbanyak); wasit hanya memutuskan manual bila masih seri.
 */
export function finishRequest(c: RefereeConsole): ConfirmRequest {
  if (c.tiePending) return manualTieRequest(c)
  if (c.isTied) {
    return {
      title: 'Skor Seri — Terapkan Aturan Seri',
      message:
        'Skor kedua sudut sama. Sistem menentukan pemenang dari sanksi paling sedikit, lalu teknik terbanyak. Bila masih seri, Anda akan diminta menentukan pemenang secara manual.',
      confirmLabel: 'Terapkan Aturan Seri',
      action: async () => ((await c.finishByTiebreak()) ? manualTieRequest(c) : undefined),
    }
  }
  return {
    title: 'Selesaikan Pertandingan?',
    message: 'Pertandingan akan diakhiri sekarang dan pemenang dihitung otomatis dari skor.',
    confirmLabel: 'Selesaikan',
    action: () => c.finish(),
  }
}

/** Dialog hasil khusus (KO/TKO/WO/undur diri): wasit memilih sudut pemenang. */
export function outcomeRequest(
  c: RefereeConsole,
  outcome: { type: string; label: string },
): ConfirmRequest {
  return {
    title: `${outcome.label} — Tentukan Pemenang`,
    message:
      'Pertandingan langsung berakhir dengan hasil ini dan tidak dapat dibatalkan. Pilih sudut yang MENANG.',
    confirmLabel: `Nyatakan ${outcome.label}`,
    danger: true,
    choices: fighterChoices(c),
    action: (choice) => {
      if (choice === 'red' || choice === 'blue') return c.declareOutcome(outcome.type, choice)
    },
  }
}
