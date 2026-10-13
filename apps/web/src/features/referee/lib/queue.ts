import type { MatchDTO } from '@/features/matches/api/matchService'
import type { ParticipantDTO, BracketSlotRecord } from '@/features/tournament/api/bracketService'

export interface Competitor {
  id: string
  name: string
  team: string
  classLabel: string
}

const GENDER_LABEL: Record<string, string> = {
  male: 'Putra',
  putra: 'Putra',
  female: 'Putri',
  putri: 'Putri',
}

export function classLabel(p: Pick<ParticipantDTO, 'weight_class' | 'gender'>): string {
  const gender = GENDER_LABEL[p.gender] ?? p.gender
  return `Kelas ${p.weight_class} ${gender}`.trim()
}

/** Peta id peserta → data tampilan (nama, kontingen, kelas). */
export function competitorMap(participants: ParticipantDTO[]): Map<string, Competitor> {
  return new Map(
    participants.map((p) => [
      p.id,
      { id: p.id, name: p.name || 'Tanpa nama', team: p.club ?? '', classLabel: classLabel(p) },
    ]),
  )
}

export type QueueState = 'loaded' | 'waiting' | 'finished' | 'cancelled'

export interface QueueRow {
  number: number
  match: MatchDTO
  red: Competitor | null
  blue: Competitor | null
  winner: 'red' | 'blue' | null
  state: QueueState
  /** Boleh dimuat ke gelanggang sekarang (belum selesai/batal dan tidak ada partai lain yang aktif). */
  canLoad: boolean
}

export const QUEUE_STATE_LABEL: Record<QueueState, string> = {
  loaded: 'DI GELANGGANG',
  waiting: 'MENUNGGU',
  finished: 'SELESAI',
  cancelled: 'BATAL',
}

/**
 * Susun antrean partai satu gelanggang (urutan dari server = urutan antrean).
 * `activeMatchId` = partai yang sedang dimuat/berlangsung di gelanggang (bila ada).
 */
export function buildQueue(
  matches: MatchDTO[],
  competitors: Map<string, Competitor>,
  activeMatchId: string | null,
): QueueRow[] {
  return matches.map((match, i) => {
    const finished = match.status === 'finished'
    const cancelled = match.status === 'cancelled'
    const loaded = !!activeMatchId && match.id === activeMatchId
    const state: QueueState = loaded
      ? 'loaded'
      : finished
        ? 'finished'
        : cancelled
          ? 'cancelled'
          : 'waiting'
    let winner: 'red' | 'blue' | null = null
    if (finished && match.winner_id) {
      if (match.winner_id === match.athlete_red_id) winner = 'red'
      else if (match.winner_id === match.athlete_blue_id) winner = 'blue'
    }
    return {
      number: i + 1,
      match,
      red: competitors.get(match.athlete_red_id) ?? null,
      blue: competitors.get(match.athlete_blue_id) ?? null,
      winner,
      state,
      canLoad: state === 'waiting' && !activeMatchId,
    }
  })
}

export interface BracketRound {
  round: number
  slots: Array<BracketSlotRecord & { competitor: Competitor | null }>
}

/** Kelompokkan slot bagan per babak (urut `slot_index`), lengkap dengan data peserta. */
export function groupBracket(
  slots: BracketSlotRecord[],
  competitors: Map<string, Competitor>,
): BracketRound[] {
  const rounds = new Map<number, BracketRound['slots']>()
  for (const s of slots) {
    const list = rounds.get(s.round) ?? []
    list.push({
      ...s,
      competitor: s.participant_id ? (competitors.get(s.participant_id) ?? null) : null,
    })
    rounds.set(s.round, list)
  }
  return [...rounds.entries()]
    .sort(([a], [b]) => a - b)
    .map(([round, list]) => ({ round, slots: list.sort((a, b) => a.slot_index - b.slot_index) }))
}

export function roundLabel(round: number, totalRounds: number): string {
  if (round === totalRounds) return 'FINAL'
  if (round === totalRounds - 1) return 'SEMIFINAL'
  return `BABAK ${round}`
}
