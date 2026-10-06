import { describe, expect, it } from 'vitest'
import { cn } from './utils'
import { generateUUID } from './uuid'

describe('cn', () => {
  it('menggabungkan kelas dan menyelesaikan konflik Tailwind', () => {
    expect(cn('p-2', 'p-4', undefined, 'text-sm')).toBe('p-4 text-sm')
  })
})

describe('generateUUID', () => {
  it('menghasilkan UUID v4 yang unik', () => {
    const a = generateUUID()
    expect(a).toMatch(/^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/)
    expect(generateUUID()).not.toBe(a)
  })
})
