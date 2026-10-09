import { describe, expect, it } from 'vitest'
import { assignableRoles, canManageRole, isSelf } from './permissions'

describe('canManageRole', () => {
  it('admin mengelola semua peran', () => {
    for (const r of ['admin', 'dewan', 'wasit', 'operator'])
      expect(canManageRole('admin', r)).toBe(true)
  })
  it('dewan hanya wasit, operator, dan juri', () => {
    expect(canManageRole('dewan', 'wasit')).toBe(true)
    expect(canManageRole('dewan', 'operator')).toBe(true)
    expect(canManageRole('dewan', 'admin')).toBe(false)
    expect(canManageRole('dewan', 'dewan')).toBe(false)
  })
  it('peran lain tidak mengelola siapa pun', () => {
    for (const a of ['wasit', 'operator', 'contingent', 'judge', '']) {
      expect(canManageRole(a, 'operator')).toBe(false)
    }
  })
})

describe('assignableRoles', () => {
  it('admin dapat memberi keempat peran, dewan hanya wasit dan operator', () => {
    expect(assignableRoles('admin')).toEqual(['admin', 'dewan', 'wasit', 'operator'])
    expect(assignableRoles('dewan')).toEqual(['wasit', 'operator'])
    expect(assignableRoles('operator')).toEqual([])
  })
})

describe('isSelf', () => {
  it('memakai id bila ada', () => {
    expect(isSelf({ userId: 'a' }, { id: 'a', email: 'x@y.z' })).toBe(true)
    expect(isSelf({ userId: 'a', email: 'x@y.z' }, { id: 'b', email: 'x@y.z' })).toBe(false)
  })
  it('jatuh ke email (tanpa beda huruf besar) untuk sesi lama tanpa id', () => {
    expect(isSelf({ email: 'Admin@Silat.id' }, { id: 'b', email: 'admin@silat.id' })).toBe(true)
    expect(isSelf({}, { id: 'b', email: 'admin@silat.id' })).toBe(false)
  })
})
