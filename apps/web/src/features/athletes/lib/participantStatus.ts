import type { ParticipantStatus } from '../types/athlete.schema'

export const PARTICIPANT_STATUS_LABEL: Record<string, string> = {
  registered: 'TERDAFTAR',
  pending: 'MENUNGGU VERIFIKASI',
  verified: 'TERVERIFIKASI',
  rejected: 'DITOLAK',
  overweight: 'KELEBIHAN BOBOT',
  underweight: 'KURANG BOBOT',
}

export const participantStatusLabel = (s: string): string =>
  PARTICIPANT_STATUS_LABEL[s] ?? s.toUpperCase()

/** Nada status untuk penanda warna: ok (hijau), wait (kuning), bad (merah). */
export function participantStatusTone(s: ParticipantStatus | string): 'ok' | 'wait' | 'bad' {
  if (s === 'verified') return 'ok'
  if (s === 'rejected' || s === 'overweight' || s === 'underweight') return 'bad'
  return 'wait'
}

/**
 * Cermin aturan server untuk tombol "Batal Daftar": admin/dewan selalu (server menolak bila sudah di bagan/partai);
 * manajer kontingen hanya selama peserta belum diverifikasi.
 */
export function canUnregister(role: string, status: ParticipantStatus | string): boolean {
  if (role === 'admin' || role === 'dewan') return true
  return role === 'contingent' && status !== 'verified'
}
