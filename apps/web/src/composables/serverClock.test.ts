import { afterEach, describe, expect, it, vi } from 'vitest'
import { serverClock } from './serverClock'

afterEach(() => {
  serverClock.reset()
  vi.useRealTimers()
})

describe('serverClock', () => {
  it('tanpa pengamatan: selisih 0 dan now() = jam klien', () => {
    vi.useFakeTimers()
    vi.setSystemTime(1_000_000)
    expect(serverClock.offset()).toBe(0)
    expect(serverClock.now()).toBe(1_000_000)
  })

  it('memakai sampel terbesar (tundaan jaringan terkecil) sebagai selisih jam', () => {
    // jam server 500 ms lebih cepat; pesan tiba dengan tundaan 40, 5, 120 ms
    serverClock.observe(10_000 + 500, 10_040) // sampel 460
    serverClock.observe(20_000 + 500, 20_005) // sampel 495 (tundaan terkecil)
    serverClock.observe(30_000 + 500, 30_120) // sampel 380
    expect(serverClock.offset()).toBe(495)
  })

  it('now() = jam klien + selisih', () => {
    vi.useFakeTimers()
    vi.setSystemTime(50_000)
    serverClock.observe(50_300, 50_000)
    expect(serverClock.now()).toBe(50_300)
  })

  it('mengabaikan nilai tidak valid', () => {
    serverClock.observe(Number.NaN, 1)
    serverClock.observe(Number.POSITIVE_INFINITY, 1)
    expect(serverClock.offset()).toBe(0)
  })

  it('hanya mengingat jendela pengamatan terakhir (sampel lama yang ekstrem tersingkir)', () => {
    serverClock.observe(1_000_000, 0) // sampel sangat besar
    for (let i = 0; i < 30; i++) serverClock.observe(100 + i, i) // 30 sampel selisih 100
    expect(serverClock.offset()).toBe(100)
  })
})
