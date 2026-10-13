import { describe, expect, it } from 'vitest'
import type { MatchDTO } from '@/features/matches/api/matchService'
import type { BracketSlotRecord } from '@/features/tournament/api/bracketService'
import { buildRefereeRounds } from './bracketView'

const slot = (
  id: string,
  round: number,
  idx: number,
  pid: string | null,
  extra: Partial<BracketSlotRecord> = {},
): BracketSlotRecord => ({
  id,
  competition_id: 'c',
  weight_class: 'A',
  gender: 'male',
  round,
  slot_index: idx,
  participant_id: pid,
  is_bye: false,
  winner_id: null,
  next_slot_id: null,
  ...extra,
})
const m = (
  id: string,
  slotId: string,
  arena: string,
  status: MatchDTO['status'] = 'idle',
): MatchDTO => ({
  id,
  competition_id: 'c',
  arena_id: arena,
  bracket_slot_id: slotId,
  athlete_red_id: 'x',
  athlete_blue_id: 'y',
  current_round: 1,
  status,
  created_at: '',
})
const comps = new Map(
  ['p1', 'p2', 'p3', 'p4'].map((id) => [
    id,
    { id, name: id.toUpperCase(), team: 'T', classLabel: '' },
  ]),
)
const arenas = [
  { id: 'A', name: 'Gelanggang A' },
  { id: 'B', name: 'Gelanggang B' },
] as never

describe('buildRefereeRounds', () => {
  const slots = [
    slot('s1', 1, 0, 'p1'),
    slot('s2', 1, 1, 'p2'),
    slot('s3', 1, 2, 'p3'),
    slot('s4', 1, 3, 'p4'),
    slot('f1', 2, 0, null),
    slot('f2', 2, 1, null),
  ]
  const matches = [m('m1', 's1', 'A'), m('m2', 's3', 'B')]

  it('menampilkan seluruh pohon dan menandai hanya partai gelanggang referee', () => {
    const rounds = buildRefereeRounds({
      slots,
      competitors: comps,
      matches,
      arenas,
      arenaId: 'A',
      queueNumbers: new Map([['m1', 7]]),
    })
    expect(rounds.map((r) => r.title)).toEqual(['BABAK SEMIFINAL', 'BABAK FINAL'])
    const [first, second] = rounds[0].matches
    expect(first.isOwnArena).toBe(true)
    expect(first.partaiNumber).toBe(7)
    expect(first.arenaLabel).toBe('GELANGGANG A')
    expect(second.isOwnArena).toBe(false)
    expect(second.arenaLabel).toBe('GELANGGANG B')
    expect(rounds[1].matches[0].red.name).toBe('Pemenang Babak Sebelumnya')
  })

  it('partai belum dijadwalkan diberi label netral dan tidak ditandai', () => {
    const [r1] = buildRefereeRounds({
      slots,
      competitors: comps,
      matches: [],
      arenas,
      arenaId: 'A',
      queueNumbers: new Map(),
    })
    expect(r1.matches[0].arenaLabel).toBe('BELUM DIJADWALKAN')
    expect(r1.matches[0].isOwnArena).toBe(false)
  })

  it('pemenang dan partai live tercermin pada kartu', () => {
    const [r1] = buildRefereeRounds({
      slots: [
        slot('s1', 1, 0, 'p1', { winner_id: 'p1' }),
        slot('s2', 1, 1, 'p2', { winner_id: 'p1' }),
      ],
      competitors: comps,
      matches: [m('m1', 's1', 'A', 'finished')],
      arenas,
      arenaId: 'A',
      queueNumbers: new Map(),
    })
    expect(r1.matches[0].status).toBe('selesai')
    expect(r1.matches[0].red.isWinner).toBe(true)
    expect(r1.matches[0].blue.isWinner).toBe(false)
    const [live] = buildRefereeRounds({
      slots: [slot('s1', 1, 0, 'p1'), slot('s2', 1, 1, 'p2')],
      competitors: comps,
      matches: [m('m1', 's1', 'B', 'running')],
      arenas,
      arenaId: 'A',
      queueNumbers: new Map(),
    })
    expect(live.matches[0].isLive).toBe(true)
  })
})
