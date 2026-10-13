import { describe, expect, it } from 'vitest'
import type { WsPenaltyEntry } from '@/composables/useWebSocket'
import { hasPenalty, lastPenalty, penaltiesOf, penaltySummary, penaltyTotal } from './penalties'

const p = (id: number, target: 'red' | 'blue', type: string, points: number): WsPenaltyEntry => ({
  id,
  round: 1,
  target,
  type,
  label: type,
  points,
  clock_ms: 0,
})

const list = [
  p(1, 'red', 'binaan_1', 0),
  p(2, 'red', 'teguran_1', 1),
  p(3, 'blue', 'peringatan_1', 5),
  p(4, 'red', 'peringatan_1', 5),
]

describe('penalties', () => {
  it('menyaring per sudut dan menjumlahkan potongan', () => {
    expect(penaltiesOf(list, 'red')).toHaveLength(3)
    expect(penaltyTotal(list, 'red')).toBe(6)
    expect(penaltyTotal(list, 'blue')).toBe(5)
    expect(penaltyTotal([], 'red')).toBe(0)
  })

  it('merangkum per golongan', () => {
    expect(penaltySummary(list, 'red')).toEqual({ binaan: 1, teguran: 1, peringatan: 1 })
    expect(penaltySummary(list, 'blue')).toEqual({ binaan: 0, teguran: 0, peringatan: 1 })
  })

  it('mendeteksi sanksi yang sudah dijatuhkan', () => {
    expect(hasPenalty(list, 'red', 'teguran_1')).toBe(true)
    expect(hasPenalty(list, 'blue', 'teguran_1')).toBe(false)
  })

  it('sanksi terakhir = yang paling akhir dijatuhkan pada sudut itu', () => {
    expect(lastPenalty(list, 'red')?.id).toBe(4)
    expect(lastPenalty(list, 'blue')?.id).toBe(3)
    expect(lastPenalty([], 'red')).toBeUndefined()
  })
})
