import { describe, expect, it } from 'vitest'
import {
  buildPresets,
  createCategorySchema,
  groupNote,
  IPSI_STANDARD_PRESETS,
  isJurusOnly,
  presetToPayload,
} from './category.schema'

const find = (age: string, sex: string, code: string) =>
  IPSI_STANDARD_PRESETS.find((p) => p.ageGroup === age && p.gender === sex && p.code === code)

describe('preset kelas bobot resmi PB IPSI', () => {
  it('Usia Dini 2: A 26–28 … S >62–64, Open >64–68', () => {
    expect(find('usia_dini_2', 'male', 'A')).toMatchObject({ minWeight: 26, maxWeight: 28 })
    expect(find('usia_dini_2', 'female', 'S')).toMatchObject({ minWeight: 62, maxWeight: 64 })
    expect(find('usia_dini_2', 'male', 'OPEN')).toMatchObject({ minWeight: 64, maxWeight: 68 })
    expect(find('usia_dini_2', 'male', 'T')).toBeUndefined()
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
  it('Pra Usia Dini tidak punya kategori Tanding (hanya Jurus)', () => {
    expect(IPSI_STANDARD_PRESETS.some((p) => p.ageGroup === 'pra_usia_dini')).toBe(false)
    expect(isJurusOnly('ipsi', 'pra_usia_dini')).toBe(true)
    expect(groupNote('ipsi', 'pra_usia_dini')).toMatch(/Jurus/)
  })
  it('Usia Dini 1: satu Kelas Bebas per gender, tanpa batas bobot (tanpa timbang)', () => {
    const ud1 = IPSI_STANDARD_PRESETS.filter((p) => p.ageGroup === 'usia_dini_1')
    expect(ud1).toHaveLength(2)
    expect(
      ud1.every((p) => p.code === 'BEBAS' && p.minWeight === null && p.maxWeight === null),
    ).toBe(true)
    expect(ud1[0].note).toMatch(/1 kg/)
  })
  it('Master A & B memakai tabel Dewasa (13 putra + 11 putri)', () => {
    for (const g of ['master_a', 'master_b']) {
      const list = IPSI_STANDARD_PRESETS.filter((p) => p.ageGroup === g)
      expect(list.filter((p) => p.gender === 'male')).toHaveLength(13)
      expect(list.filter((p) => p.gender === 'female')).toHaveLength(11)
    }
    expect(find('master_b', 'male', 'OPEN2')).toMatchObject({ minWeight: 110, maxWeight: null })
  })
  it('jumlah preset IPSI sesuai regulasi', () => {
    // UD2 40 + Pra-Remaja 34 + Remaja 28 + Dewasa 24 + Master A 24 + Master B 24 + UD1 2
    expect(IPSI_STANDARD_PRESETS).toHaveLength(40 + 34 + 28 + 24 + 24 + 24 + 2)
  })
  it('semua preset memakai sistem gugur (pemasalan hanya pilihan manual)', () => {
    expect(IPSI_STANDARD_PRESETS.every((p) => p.bracketMode === 'single_elimination')).toBe(true)
  })
  it('PERSILAT 2026: tabel bobot Pre-Teen..Senior, Singa/Macan/Master tanpa kelas bobot', () => {
    const P = buildPresets('persilat_2026')
    const pf = (age: string, sex: string, code: string) =>
      P.find((p) => p.ageGroup === age && p.gender === sex && p.code === code)
    expect(pf('pre_teen', 'male', 'A')).toMatchObject({ minWeight: 26, maxWeight: 28 })
    expect(pf('pre_teen', 'female', 'OPEN')).toMatchObject({ minWeight: 64, maxWeight: 68 })
    expect(pf('pra_remaja', 'male', 'P')).toMatchObject({ minWeight: 75, maxWeight: 78 })
    expect(pf('remaja', 'female', 'OPEN2')).toMatchObject({ minWeight: 92, maxWeight: null })
    expect(pf('dewasa', 'male', 'UNDER')).toMatchObject({ maxWeight: 45 })
    for (const g of ['singa', 'macan', 'master_a', 'master_b']) {
      const list = P.filter((p) => p.ageGroup === g)
      expect(list).toHaveLength(2)
      expect(list.every((p) => p.code === 'BEBAS' && p.maxWeight === null && !!p.note)).toBe(true)
    }
    expect(P.filter((p) => p.ageGroup === 'pre_teen')).toHaveLength(40)
    expect(P.filter((p) => p.ageGroup === 'pra_remaja')).toHaveLength(34)
    expect(P.filter((p) => p.ageGroup === 'remaja')).toHaveLength(28)
    expect(P.filter((p) => p.ageGroup === 'dewasa')).toHaveLength(24)
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
