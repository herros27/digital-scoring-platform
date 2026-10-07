import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { useTimer } from './useTimer'

// Jam server buatan: dikendalikan tes, bukan jam sistem
let serverNow = 0
const now = () => serverNow
const make = () => useTimer({ now, tickMs: 100 })

beforeEach(() => {
  vi.useFakeTimers()
  serverNow = 1_000_000
})
afterEach(() => vi.useRealTimers())

/** Majukan jam server dan jalankan satu ketukan interval. */
function advance(ms: number) {
  serverNow += ms
  vi.advanceTimersByTime(100)
}

describe('useTimer (berbasis tenggat)', () => {
  it('menghitung sisa waktu dari tenggat dan jam server', () => {
    const t = make()
    t.sync(120_000, true, serverNow + 120_000)
    expect(t.remainingMs.value).toBe(120_000)
    advance(30_000)
    expect(t.remainingMs.value).toBe(90_000)
    expect(t.formattedTime.value).toBe('01:30')
  })

  it('tidak melenceng walau ketukan interval hilang/diperlambat (tab di latar belakang)', () => {
    const t = make()
    t.sync(120_000, true, serverNow + 120_000)
    // 60 detik berlalu tetapi hanya SATU ketukan yang sempat berjalan
    advance(60_000)
    expect(t.remainingMs.value).toBe(60_000)
    // penghitung lama (kurangi 100 per ketukan) akan menjawab 119.900 di sini
  })

  it('dua layar yang menerima tenggat sama menampilkan waktu sama walau mulai di saat berbeda', () => {
    const endsAt = serverNow + 120_000
    const wasit = make()
    wasit.sync(120_000, true, endsAt)
    advance(700) // juri baru menerima pesan 0,7 detik kemudian
    const juri = make()
    juri.sync(119_300, true, endsAt)
    const scoreboard = make()
    scoreboard.sync(119_300, true, endsAt)

    for (const step of [0, 400, 1_000, 5_000, 20_000]) {
      advance(step)
      expect(juri.formattedTime.value).toBe(wasit.formattedTime.value)
      expect(scoreboard.formattedTime.value).toBe(wasit.formattedTime.value)
      expect(juri.remainingMs.value).toBe(wasit.remainingMs.value)
    }
  })

  it('menghormati selisih jam: tenggat dalam waktu server dibaca dengan jam server', () => {
    // jam klien tertinggal 2 detik dari server; `now` sudah memasukkan selisih itu
    let clientNow = 500_000
    const offset = 2_000
    const t = useTimer({ now: () => clientNow + offset, tickMs: 100 })
    t.sync(10_000, true, 500_000 + offset + 10_000)
    clientNow += 4_000
    vi.advanceTimersByTime(100)
    expect(t.remainingMs.value).toBe(6_000)
  })

  it('saat tidak berjalan: nilai tetap, tidak ada ketukan', () => {
    const t = make()
    t.sync(98_000, false)
    advance(10_000)
    expect(t.remainingMs.value).toBe(98_000)
    expect(t.isRunning.value).toBe(false)
    expect(vi.getTimerCount()).toBe(0)
  })

  it('habis: berhenti di 0 dan interval dibersihkan', () => {
    const t = make()
    t.sync(1_000, true, serverNow + 1_000)
    advance(1_500)
    expect(t.remainingMs.value).toBe(0)
    expect(t.isRunning.value).toBe(false)
    expect(t.formattedTime.value).toBe('00:00')
    expect(vi.getTimerCount()).toBe(0)
  })

  it('tanpa tenggat (server lama): tenggat dihitung dari sisa waktu + jam sekarang', () => {
    const t = make()
    t.sync(5_000, true)
    advance(2_000)
    expect(t.remainingMs.value).toBe(3_000)
  })

  it('jeda membekukan sisa waktu saat itu', () => {
    const t = make()
    t.sync(60_000, true, serverNow + 60_000)
    advance(10_000)
    t.pause()
    const frozen = t.remainingMs.value
    expect(frozen).toBe(50_000)
    advance(20_000)
    expect(t.remainingMs.value).toBe(frozen)
    expect(t.isRunning.value).toBe(false)
  })

  it('sinkronisasi ulang menggeser tenggat (koreksi waktu wasit)', () => {
    const t = make()
    t.sync(120_000, true, serverNow + 120_000)
    advance(5_000)
    t.sync(105_000, true, serverNow + 105_000) // wasit -10 detik
    expect(t.remainingMs.value).toBe(105_000)
    advance(5_000)
    expect(t.remainingMs.value).toBe(100_000)
  })

  it('pembulatan ke atas: 119,9 detik tampil 02:00, berganti setelah satu detik penuh', () => {
    const t = make()
    t.sync(119_900, false)
    expect(t.formattedTime.value).toBe('02:00')
    t.sync(119_000, false)
    expect(t.formattedTime.value).toBe('01:59')
    t.sync(0, false)
    expect(t.formattedTime.value).toBe('00:00')
  })
})
