import { describe, expect, it } from 'vitest'
import { canUnregister, participantStatusLabel, participantStatusTone } from './participantStatus'

describe('canUnregister', () => {
  it('admin dan dewan selalu boleh (server yang menolak bila sudah di bagan)', () => {
    for (const s of ['pending', 'verified', 'rejected']) {
      expect(canUnregister('admin', s)).toBe(true)
      expect(canUnregister('dewan', s)).toBe(true)
    }
  })
  it('kontingen hanya sebelum diverifikasi', () => {
    expect(canUnregister('contingent', 'pending')).toBe(true)
    expect(canUnregister('contingent', 'rejected')).toBe(true)
    expect(canUnregister('contingent', 'verified')).toBe(false)
  })
  it('peran lain tidak boleh', () => {
    for (const r of ['wasit', 'operator', 'judge', ''])
      expect(canUnregister(r, 'pending')).toBe(false)
  })
})

describe('label dan nada status', () => {
  it('menerjemahkan status dan memberi nada', () => {
    expect(participantStatusLabel('verified')).toBe('TERVERIFIKASI')
    expect(participantStatusLabel('aneh')).toBe('ANEH')
    expect(participantStatusTone('verified')).toBe('ok')
    expect(participantStatusTone('overweight')).toBe('bad')
    expect(participantStatusTone('pending')).toBe('wait')
  })
})
