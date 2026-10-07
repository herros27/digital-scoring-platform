import { describe, expect, it } from 'vitest'
import { AppApiError } from '@/api/error'
import {
  formatPin,
  isCompletePin,
  pinLengthForRole,
  pinLoginErrorMessage,
  validateCustomPin,
} from './pin'

const apiError = (code: string, message = 'x') =>
  new AppApiError({ statusCode: 400, code, message })

describe('pinLengthForRole', () => {
  it('wasit 8 digit, role lain 6 digit', () => {
    expect(pinLengthForRole('wasit')).toBe(8)
    expect(pinLengthForRole('juri')).toBe(6)
    expect(pinLengthForRole('dewan')).toBe(6)
  })
})

describe('isCompletePin', () => {
  it('menerima 6 atau 8 digit angka saja', () => {
    expect(isCompletePin('123456')).toBe(true)
    expect(isCompletePin('12345678')).toBe(true)
    expect(isCompletePin('12345')).toBe(false)
    expect(isCompletePin('1234567')).toBe(false)
    expect(isCompletePin('12345a')).toBe(false)
    expect(isCompletePin('')).toBe(false)
  })
})

describe('validateCustomPin', () => {
  it('memvalidasi per role', () => {
    expect(validateCustomPin('juri', '123456')).toBeNull()
    expect(validateCustomPin('wasit', '12345678')).toBeNull()
    expect(validateCustomPin('wasit', '123456')).toMatch(/8 digit/)
    expect(validateCustomPin('juri', '12345678')).toMatch(/6 digit/)
    expect(validateCustomPin('juri', '12ab56')).toMatch(/angka/)
  })
})

describe('formatPin', () => {
  it('memberi spasi sesuai panjang', () => {
    expect(formatPin('123456')).toBe('123 456')
    expect(formatPin('11110001')).toBe('1111 0001')
    expect(formatPin('')).toBe('------')
  })
})

describe('pinLoginErrorMessage', () => {
  it('memakai pesan Indonesia untuk kode error dikenal', () => {
    expect(pinLoginErrorMessage(apiError('INVALID_PIN'))).toMatch(/salah/)
    expect(pinLoginErrorMessage(apiError('TOO_MANY_ATTEMPTS'))).toMatch(/Terlalu banyak/)
  })
  it('jatuh ke pesan umum untuk error lain', () => {
    expect(pinLoginErrorMessage(undefined)).toMatch(/terhubung/)
  })
})
