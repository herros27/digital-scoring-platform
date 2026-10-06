import { describe, expect, it } from 'vitest'
import { MATCH_STATUS_LABEL, matchStatusFromState } from './matchStatus'

describe('matchStatusFromState', () => {
  it('memetakan state server ke status tampilan', () => {
    expect(matchStatusFromState('idle')).toBe('idle')
    expect(matchStatusFromState('ready')).toBe('ready')
    expect(matchStatusFromState('round_2', true)).toBe('running')
    expect(matchStatusFromState('round_1')).toBe('running')
    expect(matchStatusFromState('break_1', false)).toBe('break')
    expect(matchStatusFromState('finished')).toBe('finished')
  })
  it('jeda: round_N dengan timer_running=false menjadi paused', () => {
    expect(matchStatusFromState('round_1', false)).toBe('paused')
  })
  it('cancelled diperlakukan terminal seperti finished', () => {
    expect(matchStatusFromState('cancelled')).toBe('finished')
  })
  it('state kosong atau tidak dikenal tidak mengubah status', () => {
    expect(matchStatusFromState(undefined)).toBeUndefined()
    expect(matchStatusFromState('')).toBeUndefined()
    expect(matchStatusFromState('aneh')).toBeUndefined()
  })
})

describe('MATCH_STATUS_LABEL', () => {
  it('punya label untuk setiap status', () => {
    const statuses = ['idle', 'ready', 'running', 'paused', 'break', 'finished'] as const
    for (const s of statuses) expect(MATCH_STATUS_LABEL[s]).toMatch(/^[A-Z]+$/)
  })
})
