import type { WsPenaltyEntry, WsTarget } from '@/composables/useWebSocket'

export const penaltiesOf = (list: WsPenaltyEntry[], target: WsTarget): WsPenaltyEntry[] =>
  list.filter((p) => p.target === target)

/** Total poin yang dipotong dari satu sudut. */
export const penaltyTotal = (list: WsPenaltyEntry[], target: WsTarget): number =>
  penaltiesOf(list, target).reduce((sum, p) => sum + p.points, 0)

export interface PenaltySummary {
  binaan: number
  teguran: number
  peringatan: number
}

/** Jumlah sanksi per golongan (berdasarkan awalan `type`: binaan_*, teguran_*, peringatan_*). */
export function penaltySummary(list: WsPenaltyEntry[], target: WsTarget): PenaltySummary {
  const out: PenaltySummary = { binaan: 0, teguran: 0, peringatan: 0 }
  for (const p of penaltiesOf(list, target)) {
    if (p.type.startsWith('binaan')) out.binaan++
    else if (p.type.startsWith('teguran')) out.teguran++
    else if (p.type.startsWith('peringatan')) out.peringatan++
  }
  return out
}

export const hasPenalty = (list: WsPenaltyEntry[], target: WsTarget, type: string): boolean =>
  list.some((p) => p.target === target && p.type === type)

/** Sanksi terakhir pada satu sudut (yang akan dibatalkan oleh "Batalkan Sanksi"). */
export function lastPenalty(list: WsPenaltyEntry[], target: WsTarget): WsPenaltyEntry | undefined {
  const mine = penaltiesOf(list, target)
  return mine[mine.length - 1]
}
