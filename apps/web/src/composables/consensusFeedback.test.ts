import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { useConsensusFeedback } from './consensusFeedback'

const techniques = [
  { type: 'technique', label: 'Pukulan', points: 1 },
  { type: 'catch_fall', label: 'Tangkapan + Jatuhan', points: 4 },
]

describe('useConsensusFeedback', () => {
  beforeEach(() => vi.useFakeTimers())
  afterEach(() => vi.useRealTimers())

  it('jumlah lampu mengikuti jumlah juri ruleset', () => {
    const f = useConsensusFeedback(() => techniques)
    expect(f.dots('red').value).toHaveLength(3)
    f.configure(5, 3)
    expect(f.dots('red').value).toHaveLength(5)
    expect(f.needed.value).toBe(3)
  })

  it('kemajuan menyalakan lampu sesuai jumlah juri yang sepakat, lalu padam setelah window', () => {
    const f = useConsensusFeedback(() => techniques)
    f.onProgress({ target: 'red', score_type: 'technique', count: 1, needed: 2, window_ms: 3000 })
    expect(f.dots('red').value).toEqual([true, false, false])
    expect(f.dots('blue').value).toEqual([false, false, false])
    expect(f.sides.red.text).toBe('PUKULAN 1/2 • MENUNGGU JURI LAIN')
    expect(f.sides.red.confirmed).toBe(false)

    vi.advanceTimersByTime(2999)
    expect(f.sides.red.lit).toBe(1)
    vi.advanceTimersByTime(2)
    expect(f.sides.red.lit).toBe(0)
    expect(f.sides.red.text).toBe('')
  })

  it('tekanan berikutnya memperpanjang window dan memperbarui hitungan', () => {
    const f = useConsensusFeedback(() => techniques)
    f.onProgress({ target: 'blue', score_type: 'technique', count: 1, needed: 3, window_ms: 3000 })
    vi.advanceTimersByTime(2000)
    f.onProgress({ target: 'blue', score_type: 'technique', count: 2, needed: 3, window_ms: 3000 })
    vi.advanceTimersByTime(2000)
    expect(f.sides.blue.lit).toBe(2)
  })

  it('terkonfirmasi: lampu penuh sesuai juri yang sepakat, label teknik dan poin, lalu padam', () => {
    const f = useConsensusFeedback(() => techniques)
    f.configure(5, 3)
    f.onConfirmed({ target: 'red', score_type: 'catch_fall', points: 4, count: 3 })
    expect(f.dots('red').value).toEqual([true, true, true, false, false])
    expect(f.sides.red.confirmed).toBe(true)
    expect(f.sides.red.text).toBe('TANGKAPAN + JATUHAN +4 TERKONFIRMASI')
    vi.advanceTimersByTime(1600)
    expect(f.sides.red.lit).toBe(0)
    expect(f.sides.red.confirmed).toBe(false)
  })

  it('target tidak dikenal diabaikan; reset memadamkan semua', () => {
    const f = useConsensusFeedback(() => techniques)
    f.onProgress({ target: 'ungu', score_type: 'technique', count: 1, needed: 2, window_ms: 3000 })
    expect(f.sides.red.lit + f.sides.blue.lit).toBe(0)
    f.onProgress({ target: 'red', score_type: 'technique', count: 2, needed: 2, window_ms: 3000 })
    f.reset()
    expect(f.sides.red.lit).toBe(0)
  })

  it('jumlah lampu menyala tidak melebihi jumlah juri', () => {
    const f = useConsensusFeedback(() => techniques)
    f.onProgress({ target: 'red', score_type: 'technique', count: 9, needed: 2, window_ms: 3000 })
    expect(f.sides.red.lit).toBe(3)
  })
})
