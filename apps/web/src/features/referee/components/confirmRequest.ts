import type { RefereeConsole } from '../composables/useRefereeConsole'
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
  /** Menerima nilai pilihan (jika `choices` diisi). */
  action: (choice?: string) => unknown
}

/**
 * Dialog "Selesaikan Pertandingan". Saat skor seri, wasit wajib memilih pemenang
 * (keputusan wasit); selain itu pemenang dihitung otomatis dari skor.
 */
export function finishRequest(c: RefereeConsole): ConfirmRequest {
  if (c.isTied) {
    return {
      title: 'Skor Seri — Keputusan Wasit',
      message:
        'Skor kedua sudut sama. Tentukan pemenang berdasarkan keputusan wasit untuk mengakhiri pertandingan.',
      confirmLabel: 'Tetapkan Pemenang & Selesaikan',
      choices: (['red', 'blue'] as const).map((t) => ({
        value: t,
        label:
          `Sudut ${TARGET_LABEL[t]} — ${(t === 'red' ? c.redCompetitor : c.blueCompetitor)?.name ?? ''}`.trim(),
      })),
      action: (choice) => c.finish(choice === 'red' || choice === 'blue' ? choice : undefined),
    }
  }
  return {
    title: 'Selesaikan Pertandingan?',
    message: 'Pertandingan akan diakhiri sekarang dan pemenang dihitung otomatis dari skor.',
    confirmLabel: 'Selesaikan',
    action: () => c.finish(),
  }
}
