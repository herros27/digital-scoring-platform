import { describe, expect, it } from 'vitest'
import { countTechniques, formatClock, parsePayload, toLogEntry } from './matchLog'

const ev = (event_type: string, payload: Record<string, unknown>, id = '1') => ({
  id,
  event_type,
  payload,
})

describe('formatClock', () => {
  it('memformat milidetik menjadi mm:ss', () => {
    expect(formatClock(98000)).toBe('01:38')
    expect(formatClock(0)).toBe('00:00')
    expect(formatClock(120000)).toBe('02:00')
    expect(formatClock(-5)).toBe('00:00')
  })
})

describe('parsePayload', () => {
  it('menerima string JSON (REST) dan objek (WebSocket)', () => {
    expect(parsePayload('{"a":1}')).toEqual({ a: 1 })
    expect(parsePayload({ a: 1 })).toEqual({ a: 1 })
  })
  it('mengembalikan objek kosong untuk masukan rusak', () => {
    expect(parsePayload('{rusak')).toEqual({})
    expect(parsePayload(null)).toEqual({})
    expect(parsePayload('[1,2]')).toEqual({})
  })
})

describe('toLogEntry', () => {
  it('poin sah memuat sudut, nilai, dan teknik', () => {
    const e = toLogEntry(
      ev('score_confirmed', {
        target: 'red',
        points: 2,
        score_type: 'knockdown',
        count: 2,
        round: 1,
        clock_ms: 75000,
      }),
    )
    expect(e).toMatchObject({
      category: 'poin',
      title: 'Poin sah Sudut MERAH +2',
      round: 1,
      clockMs: 75000,
      target: 'red',
    })
    expect(e?.detail).toContain('Tendangan')
  })

  it('sanksi dan pembatalannya', () => {
    const applied = toLogEntry(
      ev('penalty_applied', { target: 'blue', label: 'Teguran 1', points: 1 }),
    )
    expect(applied).toMatchObject({ category: 'sanksi', title: 'Teguran 1 Sudut BIRU (-1)' })
    const revoked = toLogEntry(
      ev('penalty_revoked', { target: 'blue', label: 'Teguran 1', points: 1 }),
    )
    expect(revoked?.title).toBe('Teguran 1 Sudut BIRU dibatalkan')
  })

  it('diskualifikasi menyebut pemenang', () => {
    const e = toLogEntry(ev('disqualified', { target: 'red', winner_corner: 'blue' }))
    expect(e).toMatchObject({ category: 'sanksi', title: 'DISKUALIFIKASI Sudut MERAH' })
    expect(e?.detail).toContain('BIRU')
  })

  it('kontrol: koreksi waktu bertanda +/-', () => {
    expect(toLogEntry(ev('time_adjusted', { delta_ms: 10000, remaining_ms: 100000 }))?.title).toBe(
      'Waktu dikoreksi +10 detik',
    )
    expect(toLogEntry(ev('time_adjusted', { delta_ms: -10000, remaining_ms: 90000 }))?.title).toBe(
      'Waktu dikoreksi -10 detik',
    )
  })

  it('sinyal arena membedakan aktif dan selesai', () => {
    expect(toLogEntry(ev('arena_signal', { signal: 'gong', active: true }))?.title).toBe(
      'Gong dibunyikan',
    )
    expect(toLogEntry(ev('arena_signal', { signal: 'medic', active: true }))?.title).toBe(
      'Tim medis dipanggil',
    )
    expect(toLogEntry(ev('arena_signal', { signal: 'medic', active: false }))?.title).toBe(
      'Panggilan medis selesai',
    )
    expect(toLogEntry(ev('arena_signal', { signal: 'var', active: true }))?.category).toBe('sinyal')
    expect(toLogEntry(ev('arena_signal', { signal: 'sirine' }))).toBeNull()
  })

  it('input mentah juri masuk kategori juri', () => {
    const e = toLogEntry(ev('judge_input', { seat_number: 3, score_type: 'fall', target: 'blue' }))
    expect(e).toMatchObject({ category: 'juri', title: 'Juri 3: Jatuhan Sudut BIRU' })
  })

  it('pertandingan selesai memuat pemenang dan jenis hasil', () => {
    const e = toLogEntry(ev('match_finished', { corner: 'red', result_type: 'point_win' }))
    expect(e?.detail).toBe('Pemenang Sudut MERAH • point_win')
  })

  it('jenis kejadian tidak dikenal diabaikan', () => {
    expect(toLogEntry(ev('sesuatu_baru', {}))).toBeNull()
  })

  it('menerima payload string JSON dari REST', () => {
    const e = toLogEntry({
      id: '9',
      event_type: 'match_started',
      payload: '{"round":1,"clock_ms":120000}',
    })
    expect(e).toMatchObject({ title: 'Pertandingan dimulai', round: 1, clockMs: 120000 })
  })
})

describe('toLogEntry hasil khusus', () => {
  it('outcome_declared memuat label, pemenang, dan catatan', () => {
    const e = toLogEntry(
      ev('outcome_declared', {
        label: 'Menang Teknik (TKO)',
        winner_corner: 'blue',
        note: 'handuk',
      }),
    )
    expect(e?.title).toBe('Menang Teknik (TKO)')
    expect(e?.detail).toBe('Pemenang Sudut BIRU • handuk')
  })
})

describe('countTechniques', () => {
  it('menghitung poin sah per teknik dan sudut, mengabaikan kejadian lain', () => {
    const events = [
      ev('score_confirmed', { target: 'red', score_type: 'technique' }),
      ev('score_confirmed', { target: 'red', score_type: 'technique' }),
      ev('score_confirmed', { target: 'red', score_type: 'knockdown' }),
      ev('score_confirmed', { target: 'blue', score_type: 'fall' }),
      ev('judge_input', { target: 'blue', score_type: 'fall' }),
      ev('score_confirmed', { target: 'hijau', score_type: 'fall' }),
      ev('score_confirmed', { target: 'blue', score_type: 'aneh' }),
    ]
    const c = countTechniques(events)
    expect(c.red).toEqual({ byType: { technique: 2, knockdown: 1 }, total: 3 })
    expect(c.blue).toEqual({ byType: { fall: 1 }, total: 1 })
  })

  it('memakai daftar teknik ruleset bila diberikan', () => {
    const events = [
      ev('score_confirmed', { target: 'red', score_type: 'catch_fall' }),
      ev('score_confirmed', { target: 'red', score_type: 'fall' }),
      ev('score_confirmed', { target: 'red', score_type: 'sapuan' }),
    ]
    const c = countTechniques(events, ['catch_fall', 'sapuan'])
    expect(c.red).toEqual({ byType: { catch_fall: 1, sapuan: 1 }, total: 2 })
  })
})
