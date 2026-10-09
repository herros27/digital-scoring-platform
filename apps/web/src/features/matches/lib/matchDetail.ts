import type { Participant } from '@/features/athletes/types/athlete.schema'
import type { RoundScoreDTO } from '../api/matchService'

export interface RoundSummary {
  round: number
  red: number
  blue: number
  redPenalty: number
  bluePenalty: number
  redNet: number
  blueNet: number
  /** Pemenang babak berdasarkan skor bersih; 'draw' bila seri. */
  winner: 'red' | 'blue' | 'draw'
}

/** Ringkasan per babak, berurutan; skor bersih = poin dikurangi potongan sanksi (sama dengan total server). */
export function summarizeRounds(rounds: RoundScoreDTO[]): RoundSummary[] {
  return [...rounds]
    .sort((a, b) => a.round - b.round)
    .map((r) => {
      const redNet = r.athlete_red_score - r.red_penalties
      const blueNet = r.athlete_blue_score - r.blue_penalties
      return {
        round: r.round,
        red: r.athlete_red_score,
        blue: r.athlete_blue_score,
        redPenalty: r.red_penalties,
        bluePenalty: r.blue_penalties,
        redNet,
        blueNet,
        winner: redNet === blueNet ? 'draw' : redNet > blueNet ? 'red' : 'blue',
      }
    })
}

/** Peserta yang dapat dipertandingkan secara manual: hanya yang sudah terverifikasi. */
export function selectablePlayers(participants: Participant[]): Participant[] {
  return participants
    .filter((p) => p.registration_status === 'verified')
    .sort((a, b) => (a.name || '').localeCompare(b.name || '', undefined, { numeric: true }))
}

const sameGender = (a: string, b: string): boolean => {
  const m = (g: string) => (g === 'male' || g === 'putra' ? 'm' : 'f')
  return m(a) === m(b)
}

/**
 * Peringatan (bukan larangan) untuk pasangan yang dipilih: server hanya mewajibkan peserta terverifikasi dan
 * berbeda, tetapi panitia sebaiknya tahu bila gender, golongan umur, atau kelas bobot tidak sama.
 */
export function matchupWarnings(red?: Participant, blue?: Participant): string[] {
  if (!red || !blue) return []
  const warnings: string[] = []
  if (!sameGender(red.gender, blue.gender)) warnings.push('Jenis kelamin kedua atlet berbeda.')
  if (red.age_group && blue.age_group && red.age_group !== blue.age_group) {
    warnings.push('Golongan umur kedua atlet berbeda.')
  }
  if (red.weight_class && blue.weight_class && red.weight_class !== blue.weight_class) {
    warnings.push(`Kelas bobot berbeda (${red.weight_class} vs ${blue.weight_class}).`)
  }
  return warnings
}
