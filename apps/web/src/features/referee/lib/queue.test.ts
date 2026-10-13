import { describe, expect, it } from 'vitest'
import type { MatchDTO } from '@/features/matches/api/matchService'
import type { BracketSlotRecord, ParticipantDTO } from '@/features/tournament/api/bracketService'
import { buildQueue, classLabel, competitorMap, groupBracket, roundLabel } from './queue'

const part = (id: string, name: string, weight: string, gender: string): ParticipantDTO => ({
  id,
  competition_id: 'c',
  name,
  club: `Tim ${name}`,
  weight_class: weight,
  gender,
  registration_status: 'verified',
  created_at: '',
})

const match = (
  id: string,
  red: string,
  blue: string,
  status: MatchDTO['status'] = 'idle',
  winner?: string,
): MatchDTO => ({
  id,
  competition_id: 'c',
  arena_id: 'a',
  athlete_red_id: red,
  athlete_blue_id: blue,
  current_round: 1,
  status,
  winner_id: winner,
  created_at: '',
})

const people = [
  part('p1', 'Eko', 'C', 'male'),
  part('p2', 'Fajar', 'C', 'putra'),
  part('p3', 'Sari', 'A', 'female'),
]
const comps = competitorMap(people)

describe('classLabel & competitorMap', () => {
  it('memberi label kelas dan jenis kelamin dalam bahasa Indonesia', () => {
    expect(classLabel(people[0])).toBe('Kelas C Putra')
    expect(classLabel(people[2])).toBe('Kelas A Putri')
  })
  it('memetakan id peserta ke nama dan kontingen', () => {
    expect(comps.get('p1')).toMatchObject({
      name: 'Eko',
      team: 'Tim Eko',
      classLabel: 'Kelas C Putra',
    })
  })
})

describe('buildQueue', () => {
  const matches = [
    match('m1', 'p1', 'p2', 'finished', 'p2'),
    match('m2', 'p1', 'p2'),
    match('m3', 'p1', 'p2'),
    match('m4', 'p1', 'p2', 'cancelled'),
  ]

  it('menomori antrean, menentukan pemenang, dan status baris', () => {
    const rows = buildQueue(matches, comps, null)
    expect(rows.map((r) => r.number)).toEqual([1, 2, 3, 4])
    expect(rows.map((r) => r.state)).toEqual(['finished', 'waiting', 'waiting', 'cancelled'])
    expect(rows[0].winner).toBe('blue')
    expect(rows[1].red?.name).toBe('Eko')
  })

  it('hanya partai menunggu yang boleh dimuat bila gelanggang kosong', () => {
    const rows = buildQueue(matches, comps, null)
    expect(rows.map((r) => r.canLoad)).toEqual([false, true, true, false])
  })

  it('saat ada partai aktif: ditandai "loaded" dan tidak ada partai lain yang boleh dimuat', () => {
    const rows = buildQueue(matches, comps, 'm2')
    expect(rows[1].state).toBe('loaded')
    expect(rows.every((r) => !r.canLoad)).toBe(true)
  })

  it('peserta tak dikenal menghasilkan null, bukan error', () => {
    const rows = buildQueue([match('x', 'hilang', 'p1')], comps, null)
    expect(rows[0].red).toBeNull()
    expect(rows[0].blue?.name).toBe('Eko')
  })
})

describe('groupBracket & roundLabel', () => {
  const slot = (id: string, round: number, idx: number, pid: string | null): BracketSlotRecord => ({
    id,
    competition_id: 'c',
    weight_class: 'C',
    gender: 'male',
    round,
    slot_index: idx,
    participant_id: pid,
    is_bye: false,
    winner_id: null,
    next_slot_id: null,
  })

  it('mengelompokkan per babak, mengurutkan slot, dan menautkan peserta', () => {
    const rounds = groupBracket(
      [slot('c', 2, 0, null), slot('b', 1, 1, 'p2'), slot('a', 1, 0, 'p1')],
      comps,
    )
    expect(rounds.map((r) => r.round)).toEqual([1, 2])
    expect(rounds[0].slots.map((s) => s.id)).toEqual(['a', 'b'])
    expect(rounds[0].slots[0].competitor?.name).toBe('Eko')
    expect(rounds[1].slots[0].competitor).toBeNull()
  })

  it('memberi nama babak akhir', () => {
    expect(roundLabel(3, 3)).toBe('FINAL')
    expect(roundLabel(2, 3)).toBe('SEMIFINAL')
    expect(roundLabel(1, 3)).toBe('BABAK 1')
  })
})
