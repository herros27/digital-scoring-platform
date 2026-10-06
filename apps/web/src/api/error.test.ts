import { describe, expect, it } from 'vitest'
import { AppApiError, getErrorMessage, isNotFoundError } from './error'

const apiError = (statusCode: number, message = 'x') =>
  new AppApiError({ statusCode, code: 'E', message })

describe('getErrorMessage', () => {
  it('memakai pesan dari Error', () => {
    expect(getErrorMessage(new Error('gagal'), 'cadangan')).toBe('gagal')
  })
  it('memakai string apa adanya', () => {
    expect(getErrorMessage('oops', 'cadangan')).toBe('oops')
  })
  it('jatuh ke pesan cadangan untuk nilai lain atau pesan kosong', () => {
    expect(getErrorMessage(undefined, 'cadangan')).toBe('cadangan')
    expect(getErrorMessage({ message: 'bukan Error' }, 'cadangan')).toBe('cadangan')
    expect(getErrorMessage(new Error(''), 'cadangan')).toBe('cadangan')
  })
})

describe('isNotFoundError', () => {
  it('true untuk AppApiError 404', () => {
    expect(isNotFoundError(apiError(404))).toBe(true)
  })
  it('true untuk pesan "tidak ditemukan"', () => {
    expect(isNotFoundError(new Error('Gelanggang tidak ditemukan'))).toBe(true)
  })
  it('false untuk error lain dan nilai non-Error', () => {
    expect(isNotFoundError(apiError(500))).toBe(false)
    expect(isNotFoundError(null)).toBe(false)
  })
})
