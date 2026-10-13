import type {
  BracketRoundDisplay,
  MatchCard,
} from '@/features/tournament/components/TournamentTreeBracket.vue'
import type { PublicBracketSlot } from '../api/publicBracketService'

export function roundTitle(round: number, maxRound: number): string {
  const dist = maxRound - round
  if (dist === 0) return 'BABAK FINAL'
  if (dist === 1) return 'BABAK SEMIFINAL'
  if (dist === 2) return 'BABAK PEREMPAT FINAL'
  if (dist === 3) return 'BABAK 16 BESAR'
  return `BABAK ${Math.pow(2, dist + 1)} BESAR`
}

const WAITING = 'Pemenang Babak Sebelumnya'
const BYE = 'BYE (Lolos Otomatis)'

/**
 * Mengubah slot bagan publik menjadi ronde untuk <TournamentTreeBracket read-only>.
 * Dua slot berurutan dalam satu ronde (0&1, 2&3, …) membentuk satu partai. Tampilan publik tidak punya
 * nomor gelanggang maupun status live, jadi kartu hanya menampilkan SIAP / BYE / SELESAI.
 */
export function buildPublicBracketRounds(slots: PublicBracketSlot[]): BracketRoundDisplay[] {
  if (slots.length === 0) return []

  const byRound = new Map<number, PublicBracketSlot[]>()
  for (const s of slots) {
    const list = byRound.get(s.round) ?? []
    list.push(s)
    byRound.set(s.round, list)
  }
  const maxRound = Math.max(...byRound.keys())

  const rounds: BracketRoundDisplay[] = []
  for (let round = 1; round <= maxRound; round++) {
    const list = [...(byRound.get(round) ?? [])].sort((a, b) => a.slot_index - b.slot_index)
    const matches: MatchCard[] = []

    for (let i = 0; i < list.length; i += 2) {
      const red = list[i]
      const blue = list[i + 1]
      const finished = !!(red?.is_winner || blue?.is_winner)
      const hasBye = !!(red?.is_bye || blue?.is_bye)

      matches.push({
        id: `slot-${red?.id ?? i}`,
        partaiNumber: i / 2 + 1,
        arenaLabel: '',
        status: finished ? 'selesai' : 'ready',
        statusBadge: finished ? 'SELESAI' : hasBye ? 'BYE' : 'SIAP',
        statusVariant: 'neutral',
        red: corner(red, finished),
        blue: corner(blue, finished),
      })
    }

    const dist = maxRound - round
    rounds.push({
      roundNumber: round,
      title: roundTitle(round, maxRound),
      badgeVariant: dist === 0 ? 'emerald' : dist === 1 ? 'red' : dist === 2 ? 'blue' : 'neutral',
      matches,
    })
  }
  return rounds
}

function corner(slot: PublicBracketSlot | undefined, matchFinished: boolean) {
  return {
    name: slot?.participant?.name || (slot?.is_bye ? BYE : WAITING),
    contingent: slot?.participant?.team || '-',
    score: matchFinished ? (slot?.is_winner ? 'W' : 'L') : '-',
    isWinner: !!slot?.is_winner,
  }
}
