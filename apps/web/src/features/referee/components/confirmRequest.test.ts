import { describe, expect, it, vi } from 'vitest'
import type { RefereeConsole } from '../composables/useRefereeConsole'
import { finishRequest, isConfirmRequest, manualTieRequest, outcomeRequest } from './confirmRequest'

function fakeConsole(over: Partial<Record<string, unknown>> = {}) {
  const c = {
    isTied: false,
    tiePending: false,
    redCompetitor: { name: 'Intan' },
    blueCompetitor: { name: 'Andri' },
    tiebreak: { manual_methods: ['extra_round', 'coin_toss'] },
    finish: vi.fn(),
    finishByTiebreak: vi.fn(async () => false),
    ...over,
  }
  return c as unknown as RefereeConsole & typeof c
}

describe('finishRequest', () => {
  it('tidak seri: konfirmasi biasa tanpa pilihan', () => {
    const c = fakeConsole()
    const r = finishRequest(c)
    expect(r.choices).toBeUndefined()
    r.action()
    expect(c.finish).toHaveBeenCalledWith()
  })

  it('seri: menjalankan aturan seri otomatis; selesai tanpa dialog lanjutan', async () => {
    const c = fakeConsole({ isTied: true })
    const r = finishRequest(c)
    expect(r.title).toMatch(/Aturan Seri/)
    expect(isConfirmRequest(await r.action())).toBe(false)
    expect(c.finishByTiebreak).toHaveBeenCalled()
  })

  it('seri yang tidak terselesaikan otomatis: lanjut ke dialog keputusan manual', async () => {
    const c = fakeConsole({ isTied: true, finishByTiebreak: vi.fn(async () => true) })
    const next = await finishRequest(c).action()
    expect(isConfirmRequest(next)).toBe(true)
    expect((next as ReturnType<typeof manualTieRequest>).title).toMatch(/Seri Penuh/)
  })

  it('waktu habis seri (tie_pending): langsung dialog manual', () => {
    const r = finishRequest(fakeConsole({ isTied: true, tiePending: true }))
    expect(r.methods).toBeDefined()
  })
})

describe('manualTieRequest', () => {
  it('menawarkan keputusan wasit + metode ruleset dan meneruskan pemenang serta metode', () => {
    const c = fakeConsole()
    const r = manualTieRequest(c)
    expect(r.methods?.map((m) => m.value)).toEqual(['referee_decision', 'extra_round', 'coin_toss'])
    expect(r.choices?.map((x) => x.value)).toEqual(['red', 'blue'])
    r.action('blue', 'coin_toss')
    expect(c.finish).toHaveBeenCalledWith('blue', 'coin_toss')
  })
})

describe('outcomeRequest', () => {
  it('meminta pemenang dan menyatakan hasil khusus dengan jenis yang dipilih', () => {
    const c = fakeConsole({ declareOutcome: vi.fn() })
    const r = outcomeRequest(c, { type: 'tko', label: 'Menang Teknik (TKO)' })
    expect(r.danger).toBe(true)
    expect(r.choices?.map((x) => x.value)).toEqual(['red', 'blue'])
    r.action('blue')
    expect(c.declareOutcome).toHaveBeenCalledWith('tko', 'blue')
  })

  it('tanpa pemilihan sudut tidak memanggil server', () => {
    const c = fakeConsole({ declareOutcome: vi.fn() })
    outcomeRequest(c, { type: 'ko', label: 'KO' }).action(undefined)
    expect(c.declareOutcome).not.toHaveBeenCalled()
  })
})
