import { describe, expect, it } from 'vitest'
import { buildPresets, createCategorySchema, IPSI_STANDARD_PRESETS, presetToPayload } from './category.schema'

const find = (age: string, sex: string, code: string) =>
  IPSI_STANDARD_PRESETS.find((p) => p.ageGroup === age && p.gender === sex && p.code === code)

describe('preset kelas bobot resmi PB IPSI', () => {
  it('Usia Dini: A 26–28 … S >62–64, Open >64–68', () => {
    expect(find('usia_dini_2', 'male', 'A')).toMatchObject({ minWeight: 26, maxWeight: 28 })
    expect(find('usia_dini_2', 'female', 'S')).toMatchObject({ minWeight: 62, maxWeight: 64 })
    expect(find('usia_dini_1', 'male', 'OPEN')).toMatchObject({ minWeight: 64, maxWeight: 68 })
    expect(find('usia_dini_1', 'male', 'T')).toBeUndefined()
  })
  it('Pra-Remaja: A 30–33 … P >75–78, Open >78–84', () => {
    expect(find('pra_remaja', 'male', 'A')).toMatchObject({ minWeight: 30, maxWeight: 33 })
    expect(find('pra_remaja', 'male', 'P')).toMatchObject({ minWeight: 75, maxWeight: 78 })
    expect(find('pra_remaja', 'female', 'OPEN')).toMatchObject({ minWeight: 78, maxWeight: 84 })
  })
  it('Remaja putra/putri berbeda jumlah kelas', () => {
    expect(find('remaja', 'male', 'UNDER')).toMatchObject({ minWeight: null, maxWeight: 39 })
    expect(find('remaja', 'male', 'L')).toMatchObject({ minWeight: 83, maxWeight: 87 })
    expect(find('remaja', 'male', 'OPEN1')).toMatchObject({ minWeight: 87, maxWeight: 100 })
    expect(find('remaja', 'male', 'OPEN2')).toMatchObject({ minWeight: 100, maxWeight: null })
    expect(find('remaja', 'female', 'J')).toMatchObject({ minWeight: 75, maxWeight: 79 })
    expect(find('remaja', 'female', 'K')).toBeUndefined()
    expect(find('remaja', 'female', 'OPEN1')).toMatchObject({ minWeight: 79, maxWeight: 92 })
  })
  it('Dewasa putra J 90–95 / putri H 80–85, open sesuai tabel', () => {
    expect(find('dewasa', 'male', 'UNDER')).toMatchObject({ maxWeight: 45 })
    expect(find('dewasa', 'male', 'J')).toMatchObject({ minWeight: 90, maxWeight: 95 })
    expect(find('dewasa', 'male', 'OPEN1')).toMatchObject({ minWeight: 95, maxWeight: 110 })
    expect(find('dewasa', 'female', 'H')).toMatchObject({ minWeight: 80, maxWeight: 85 })
    expect(find('dewasa', 'female', 'I')).toBeUndefined()
    expect(find('dewasa', 'female', 'OPEN2')).toMatchObject({ minWeight: 100, maxWeight: null })
  })
  it('bagan pemasalan untuk Pra Usia Dini & Usia Dini 1 saja', () => {
    expect(find('usia_dini_1', 'male', 'A')?.bracketMode).toBe('pemasalan_4')
    expect(find('usia_dini_2', 'male', 'A')?.bracketMode).toBe('single_elimination')
    expect(IPSI_STANDARD_PRESETS.find((p) => p.ageGroup === 'pra_usia_dini')?.bracketMode).toBe('pemasalan_4')
  })
  it('golongan tanpa tabel resmi mendapat Kelas Bebas tanpa batas bobot', () => {
    const master = IPSI_STANDARD_PRESETS.filter((p) => p.ageGroup === 'master_a')
    expect(master).toHaveLength(2)
    expect(master[0]).toMatchObject({ code: 'BEBAS', minWeight: null, maxWeight: null })
  })
  it('PERSILAT 2026 memakai golongan umur sendiri', () => {
    const groups = new Set(buildPresets('persilat_2026').map((p) => p.ageGroup))
    expect([...groups]).toEqual(['singa', 'macan', 'pre_teen', 'pra_remaja', 'remaja', 'dewasa'])
  })
  it('filter golongan & nama unik', () => {
    const only = buildPresets('ipsi', ['dewasa'])
    expect(only.every((p) => p.ageGroup === 'dewasa')).toBe(true)
    const names = IPSI_STANDARD_PRESETS.map((p) => p.categoryName)
    expect(new Set(names).size).toBe(names.length)
  })
  it('payload preset lolos skema', () => {
    for (const p of IPSI_STANDARD_PRESETS) {
      expect(createCategorySchema.safeParse(presetToPayload(p)).success).toBe(true)
    }
  })
})
