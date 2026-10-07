import { describe, expect, it } from 'vitest'
import {
  ageGroupForAge,
  ageGroupForBirthDate,
  ageOn,
  checkWeight,
  isEligible,
  missingDocs,
  requiredDocs,
} from './ageGroups'

describe('ageOn', () => {
  it('menghitung tahun penuh dan menghormati ulang tahun', () => {
    expect(ageOn('2020-06-15', '2026-06-15')).toBe(6)
    expect(ageOn('2020-06-16', '2026-06-15')).toBe(5)
    expect(ageOn('1990-12-31', '2026-06-15')).toBe(35)
  })
  it('menolak tanggal tidak valid', () => {
    expect(ageOn('2010-02-30', '2026-01-01')).toBeNull()
    expect(ageOn('abc', '2026-01-01')).toBeNull()
  })
})

describe('ageGroupForAge', () => {
  it('IPSI: batas atas inklusif', () => {
    const cases: Array<[number, string]> = [
      [5, 'pra_usia_dini'],
      [6, 'usia_dini_1'],
      [8, 'usia_dini_1'],
      [9, 'usia_dini_2'],
      [11, 'usia_dini_2'],
      [12, 'pra_remaja'],
      [14, 'pra_remaja'],
      [15, 'remaja'],
      [17, 'remaja'],
      [18, 'dewasa'],
      [35, 'dewasa'],
      [36, 'master_a'],
      [45, 'master_a'],
      [46, 'master_b'],
      [80, 'master_b'],
    ]
    for (const [age, want] of cases) expect(ageGroupForAge(age, 'ipsi')).toBe(want)
  })
  it('PERSILAT 2026', () => {
    expect(ageGroupForAge(2, 'persilat_2026')).toBeNull()
    expect(ageGroupForAge(3, 'persilat_2026')).toBe('singa')
    expect(ageGroupForAge(9, 'persilat_2026')).toBe('macan')
    expect(ageGroupForAge(10, 'persilat_2026')).toBe('pre_teen')
    expect(ageGroupForAge(16, 'persilat_2026')).toBe('remaja')
    expect(ageGroupForAge(45, 'persilat_2026')).toBe('dewasa')
    expect(ageGroupForAge(46, 'persilat_2026')).toBeNull()
  })
  it('dari tanggal lahir', () => {
    expect(ageGroupForBirthDate('2016-05-01', '2026-06-15')).toBe('usia_dini_2')
  })
})

describe('isEligible', () => {
  const cat = { gender: 'putra', age_group: 'remaja', preset_standard: 'ipsi' as const }
  it('cocok gender + umur', () => {
    expect(isEligible({ gender: 'male', birth_date: '2010-03-01' }, cat, '2026-06-15')).toBe(true)
  })
  it('kategori lama tanpa age_group dianggap Dewasa, anak tidak lolos', () => {
    const legacy = { gender: 'putra' }
    expect(isEligible({ gender: 'male', birth_date: '2024-07-01' }, legacy, '2026-10-07')).toBe(
      false,
    )
    expect(isEligible({ gender: 'male', birth_date: '2000-01-01' }, legacy, '2026-10-07')).toBe(
      true,
    )
  })
  it('menolak umur/gender/tanpa tanggal lahir', () => {
    expect(isEligible({ gender: 'male', birth_date: '2000-01-01' }, cat, '2026-06-15')).toBe(false)
    expect(isEligible({ gender: 'female', birth_date: '2010-03-01' }, cat, '2026-06-15')).toBe(
      false,
    )
    expect(isEligible({ gender: 'male' }, cat, '2026-06-15')).toBe(false)
  })
})

describe('checkWeight (tanpa toleransi)', () => {
  it('batas inklusif, selisih 0.1 kg melanggar', () => {
    expect(checkWeight(50, 45, 50)).toBe('ok')
    expect(checkWeight(45, 45, 50)).toBe('ok')
    expect(checkWeight(50.1, 45, 50)).toBe('overweight')
    expect(checkWeight(44.9, 45, 50)).toBe('underweight')
    expect(checkWeight(50.01, 45, 50)).toBe('overweight')
  })
  it('kelas terbuka & tanpa batas', () => {
    expect(checkWeight(130, 95, null)).toBe('ok')
    expect(checkWeight(30, null, 39)).toBe('ok')
    expect(checkWeight(60, null, null)).toBe('unspecified')
  })
})

describe('berkas verifikasi', () => {
  it('dewasa tidak wajib, di bawah umur wajib 3 berkas', () => {
    expect(requiredDocs('dewasa')).toEqual([])
    expect(requiredDocs('usia_dini_1')).toHaveLength(3)
    expect(missingDocs('remaja', { identity: true })).toEqual(['parental', 'health'])
    expect(missingDocs('remaja', { identity: true, parental: true, health: true })).toEqual([])
  })
})
