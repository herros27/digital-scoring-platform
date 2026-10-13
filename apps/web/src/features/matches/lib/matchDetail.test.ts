import { describe, expect, it } from 'vitest'
import type { Participant } from '@/features/athletes/types/athlete.schema'
import type { RoundScoreDTO } from '../api/matchService'
import { matchupWarnings, selectablePlayers, summarizeRounds } from './matchDetail'

const round = (n: number, red: number, blue: number, rp = 0, bp = 0): RoundScoreDTO => ({
  id: `r${n}`,
  match_id: 'm',
  round: n,
  athlete_red_score: red,
  athlete_blue_score: blue,
  red_penalties: rp,
  blue_penalties: bp,
  updated_at: '',
})

const player = (over: Partial<Participant>): Participant => ({
  id: 'p',
  competition_id: 'c',
  athlete_id: 'a',
  name: 'Nama',
  weight_class: 'B',
  gender: 'male',
  registration_status: 'verified',
  created_at: '',
  ...over,
})

describe('summarizeRounds', () => {
  it('mengurutkan babak, menghitung skor bersih, dan menentukan pemenang babak', () => {
    const out = summarizeRounds([round(2, 4, 4), round(1, 10, 6, 2, 0), round(3, 3, 5, 0, 1)])
    expect(out.map((r) => r.round)).toEqual([1, 2, 3])
    expect(out[0]).toMatchObject({ redNet: 8, blueNet: 6, winner: 'red' })
    expect(out[1].winner).toBe('draw')
    expect(out[2]).toMatchObject({ redNet: 3, blueNet: 4, winner: 'blue' })
  })
  it('daftar kosong menghasilkan daftar kosong', () => {
    expect(summarizeRounds([])).toEqual([])
  })
})

describe('selectablePlayers', () => {
  it('hanya peserta terverifikasi, berurutan menurut nama', () => {
    const out = selectablePlayers([
      player({ id: '1', name: 'Zaki' }),
      player({ id: '2', name: 'Andi', registration_status: 'pending' }),
      player({ id: '3', name: 'Budi' }),
      player({ id: '4', name: 'Citra', registration_status: 'rejected' }),
    ])
    expect(out.map((p) => p.name)).toEqual(['Budi', 'Zaki'])
  })
})

describe('matchupWarnings', () => {
  it('tanpa peringatan bila pasangan sepadan atau belum lengkap', () => {
    expect(matchupWarnings(player({}), player({}))).toEqual([])
    expect(matchupWarnings(undefined, player({}))).toEqual([])
  })
  it('memperingatkan gender, golongan umur, dan kelas bobot yang berbeda (putra = male)', () => {
    const w = matchupWarnings(
      player({ gender: 'putra', age_group: 'dewasa', weight_class: 'B' }),
      player({ gender: 'female', age_group: 'remaja', weight_class: 'C' }),
    )
    expect(w).toHaveLength(3)
    expect(matchupWarnings(player({ gender: 'putra' }), player({ gender: 'male' }))).toEqual([])
  })
})
