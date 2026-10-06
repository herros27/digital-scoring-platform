import { describe, expect, it } from 'vitest'
import type { RulesetDTO } from '../types'
import {
  copyDraft,
  draftFromRuleset,
  newDraft,
  normalizedPriority,
  payloadFromDraft,
  priorityByValue,
  summarize,
  validateDraft,
} from './rulesetDraft'

const legacy: RulesetDTO = {
  id: 'r1',
  name: 'IPSI Lama',
  version: '1',
  status: 'active',
  scoring: {
    technique_points: 1,
    knockdown_points: 2,
    fall_points: 3,
    max_penalties: 3,
    penalty_threshold: 1,
    round_duration: 120,
    break_duration: 60,
    total_rounds: 3,
  },
}

describe('draftFromRuleset', () => {
  it('ruleset lama tanpa bagian opsional diisi bawaan yang sama dengan server', () => {
    const d = draftFromRuleset(legacy)
    expect(d.techniques.map((t) => t.type)).toEqual(['technique', 'knockdown', 'fall'])
    expect(d.penalties.find((p) => p.type === 'teguran_2')?.points).toBe(2)
    expect(d.penalties.find((p) => p.type === 'peringatan_3')?.disqualify).toBe(true)
    expect(d.consensus).toEqual({ judges: 3, majority: 2, windowMs: 3000 })
    expect(d.tiebreak.priority).toEqual(['fall', 'knockdown', 'technique'])
    expect(d.outcomes.every((o) => o.enabled)).toBe(true)
    expect(validateDraft({ ...d })).toEqual({})
  })

  it('hasil khusus yang tidak tercantum ditandai nonaktif', () => {
    const d = draftFromRuleset({
      ...legacy,
      scoring: { ...legacy.scoring, outcomes: [{ type: 'ko', label: 'KO', phase: 'in_match' }] },
    })
    expect(d.outcomes.filter((o) => o.enabled).map((o) => o.type)).toEqual(['ko'])
  })
})

describe('payloadFromDraft', () => {
  it('menurunkan nilai skalar warisan dari teknik dan menulis semua bagian', () => {
    const d = newDraft()
    d.name = 'Kejurda'
    d.techniques[2].points = 4 // jatuhan 4
    const p = payloadFromDraft(d)
    expect(p.scoring.fall_points).toBe(4)
    expect(p.scoring.techniques).toHaveLength(6)
    expect(p.scoring.penalties?.at(-1)).toEqual({
      type: 'peringatan_3',
      label: 'Peringatan 3',
      points: 0,
      action: 'disqualify',
    })
    expect(p.scoring.consensus).toEqual({ judges: 3, majority: 2, window_ms: 3000 })
    expect(p.scoring.tiebreak?.steps).toEqual(['fewest_penalties', 'technique_priority'])
    expect(p.scoring.outcomes).toHaveLength(4)
  })

  it('langkah seri dan hasil khusus nonaktif tidak ikut dikirim', () => {
    const d = newDraft()
    d.name = 'X1'
    d.tiebreak.techniquePriority = false
    d.outcomes[0].enabled = false
    const p = payloadFromDraft(d)
    expect(p.scoring.tiebreak?.steps).toEqual(['fewest_penalties'])
    expect(p.scoring.outcomes?.map((o) => o.type)).not.toContain('ko')
  })

  it('round-trip: draft -> payload -> draft tidak mengubah isi', () => {
    const d = newDraft()
    d.name = 'Round Trip'
    const back = draftFromRuleset({ id: 'x', ...payloadFromDraft(d) } as RulesetDTO)
    expect(payloadFromDraft(back)).toEqual(payloadFromDraft(d))
  })
})

describe('priority', () => {
  it('urut menurut nilai tertinggi', () => {
    // urutan regulasi: 1+3, 3, 1+2, 2, 1+1, 1
    expect(priorityByValue(newDraft().techniques)).toEqual([
      'catch_fall',
      'fall',
      'combo_1_2',
      'knockdown',
      'combo_1_1',
      'technique',
    ])
  })
  it('membuang kode yang dihapus dan menambah kode baru di akhir', () => {
    const d = newDraft()
    d.tiebreak.priority = ['fall', 'hilang', 'technique']
    expect(normalizedPriority(d)).toEqual([
      'fall',
      'technique',
      'knockdown',
      'catch_fall',
      'combo_1_2',
      'combo_1_1',
    ])
  })
})

describe('validateDraft', () => {
  const valid = () => ({ ...newDraft(), name: 'Valid' })
  it('draft bawaan valid', () => expect(validateDraft(valid())).toEqual({}))
  it('menolak nama pendek, kode ganda, kode dicadangkan, nilai 0, mayoritas > juri, window kecil', () => {
    const d = valid()
    d.name = 'x'
    d.techniques[1].type = 'technique'
    d.techniques[2].type = 'undo'
    d.techniques[3].points = 0
    d.penalties[0].points = -1
    d.consensus = { judges: 3, majority: 4, windowMs: 100 }
    const e = validateDraft(d)
    expect(Object.keys(e).sort()).toEqual(
      [
        'consensus.majority',
        'consensus.windowMs',
        'name',
        'penalties.0.points',
        'techniques.1.type',
        'techniques.2.type',
        'techniques.3.points',
      ].sort(),
    )
  })
  it('kode harus huruf kecil dan diawali huruf', () => {
    const d = valid()
    d.techniques[0].type = 'Tendang 1'
    d.penalties[0].type = '1x'
    const e = validateDraft(d)
    expect(e['techniques.0.type']).toBeDefined()
    expect(e['penalties.0.type']).toBeDefined()
  })
})

describe('copyDraft & summarize', () => {
  it('salinan tanpa id dengan nama berakhiran (salinan)', () => {
    const c = copyDraft(draftFromRuleset(legacy))
    expect(c.id).toBeUndefined()
    expect(c.name).toBe('IPSI Lama (salinan)')
  })
  it('ringkasan memuat babak, juri, teknik, sanksi', () => {
    expect(summarize(legacy)).toBe('3 babak × 2 mnt • 3 juri (2 sepakat) • 3 teknik • 7 sanksi')
  })
})
