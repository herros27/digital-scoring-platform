import type { Arena } from '@/features/arena/types/arena.schema'
import type { MatchDTO } from '@/features/matches/api/matchService'
import type { BracketSlotRecord } from '@/features/tournament/api/bracketService'
import type {
  BracketRoundDisplay,
  MatchCard,
} from '@/features/tournament/components/TournamentTreeBracket.vue'
import type { Competitor } from './queue'

export function roundTitle(roundNum: number, maxRound: number): string {
  const dist = maxRound - roundNum
  if (dist === 0) return 'BABAK FINAL'
  if (dist === 1) return 'BABAK SEMIFINAL'
  if (dist === 2) return 'BABAK PEREMPAT FINAL'
  if (dist === 3) return 'BABAK 16 BESAR'
  return `BABAK ${Math.pow(2, dist + 1)} BESAR`
}

const isLiveStatus = (s?: string) => s === 'running' || s === 'paused'

export interface RefereeBracketInput {
  slots: BracketSlotRecord[]
  competitors: Map<string, Competitor>
  /** Semua partai kejuaraan (semua gelanggang): kartu bagan memakai gelanggang tiap partai. */
  matches: MatchDTO[]
  arenas: Arena[]
  /** Gelanggang yang sedang dibuka referee: partainya ditandai. */
  arenaId: string
  /** Nomor antrean per ID partai untuk gelanggang ini. */
  queueNumbers: Map<string, number>
  /** Partai yang sedang dimuat di gelanggang ini. */
  loadedMatchId?: string | null
}

/**
 * Bagan satu kategori untuk referee. Bagan milik kategori, bukan gelanggang: partainya bisa tersebar di beberapa
 * gelanggang, sehingga seluruh pohon ditampilkan dan partai gelanggang referee ditandai (`isOwnArena`).
 */
export function buildRefereeRounds(input: RefereeBracketInput): BracketRoundDisplay[] {
  const byRound = new Map<number, BracketSlotRecord[]>()
  for (const s of input.slots) {
    const list = byRound.get(s.round) ?? []
    list.push(s)
    byRound.set(s.round, list)
  }
  const roundNumbers = [...byRound.keys()].sort((a, b) => a - b)
  const maxRound = roundNumbers.length ? roundNumbers[roundNumbers.length - 1] : 0
  const arenaName = (id?: string) => input.arenas.find((a) => a.id === id)?.name

  return roundNumbers.map((r) => {
    const slots = (byRound.get(r) ?? []).sort((a, b) => a.slot_index - b.slot_index)
    const matches: MatchCard[] = []
    for (let i = 0; i < slots.length; i += 2) {
      const red = slots[i]
      const blue = slots[i + 1]
      const pair = Math.floor(i / 2) + 1
      const compRed = red?.participant_id ? input.competitors.get(red.participant_id) : undefined
      const compBlue = blue?.participant_id ? input.competitors.get(blue.participant_id) : undefined

      const m = input.matches.find(
        (x) =>
          x.status !== 'cancelled' &&
          ((red?.id && x.bracket_slot_id === red.id) ||
            (blue?.id && x.bracket_slot_id === blue.id)),
      )
      const finished = !!(red?.winner_id || blue?.winner_id)
      const redWin = !!red?.winner_id && red.winner_id === red.participant_id
      const blueWin = !!blue?.winner_id && blue.winner_id === blue.participant_id
      const live = !!m && (isLiveStatus(m.status) || m.id === input.loadedMatchId)
      const bye = !!(red?.is_bye || blue?.is_bye)
      const own = !!m && m.arena_id === input.arenaId

      matches.push({
        id: `slot-${red?.id ?? i}`,
        partaiNumber: (m && input.queueNumbers.get(m.id)) || pair,
        arenaLabel: (arenaName(m?.arena_id) ?? 'BELUM DIJADWALKAN').toUpperCase(),
        status: finished ? 'selesai' : live ? 'live' : 'ready',
        statusBadge: finished ? 'SELESAI' : live ? 'LIVE' : bye ? 'BYE' : m ? 'SIAP' : 'MENUNGGU',
        statusVariant: finished ? 'neutral' : live ? 'emerald' : 'neutral',
        isLive: live,
        isOwnArena: own,
        red: {
          name:
            compRed?.name ?? (red?.is_bye ? 'BYE (Lolos Otomatis)' : 'Pemenang Babak Sebelumnya'),
          contingent: compRed?.team || '-',
          score: finished ? (redWin ? 'W' : 'L') : '-',
          isWinner: redWin,
        },
        blue: {
          name:
            compBlue?.name ?? (blue?.is_bye ? 'BYE (Lolos Otomatis)' : 'Pemenang Babak Sebelumnya'),
          contingent: compBlue?.team || '-',
          score: finished ? (blueWin ? 'W' : 'L') : '-',
          isWinner: blueWin,
        },
      })
    }
    const dist = maxRound - r
    return {
      roundNumber: r,
      title: roundTitle(r, maxRound),
      badgeVariant: dist === 0 ? 'emerald' : dist === 1 ? 'red' : dist === 2 ? 'blue' : 'neutral',
      matches,
    }
  })
}
