import { describe, expect, it } from 'vitest'
import type { PublicBracketSlot } from '../api/publicBracketService'
import { buildPublicBracketRounds, roundTitle } from './publicBracketRounds'

function slot(
  p: Partial<PublicBracketSlot> & Pick<PublicBracketSlot, 'id' | 'round' | 'slot_index'>,
) {
  return {
    is_bye: false,
    next_slot_id: null,
    is_winner: false,
    participant: null,
    ...p,
  } satisfies PublicBracketSlot
}
const athlete = (name: string, team = 'Klub') => ({ name, team })

describe('roundTitle', () => {
  it('menamai ronde dari final ke belakang', () => {
    expect(roundTitle(3, 3)).toBe('BABAK FINAL')
    expect(roundTitle(2, 3)).toBe('BABAK SEMIFINAL')
    expect(roundTitle(1, 3)).toBe('BABAK PEREMPAT FINAL')
    expect(roundTitle(1, 4)).toBe('BABAK 16 BESAR')
    expect(roundTitle(1, 5)).toBe('BABAK 32 BESAR')
  })
})

describe('buildPublicBracketRounds', () => {
  it('mengembalikan kosong bila tidak ada slot', () => {
    expect(buildPublicBracketRounds([])).toEqual([])
  })

  it('memasangkan slot berurutan menjadi partai dan menampilkan nama + kontingen', () => {
    const rounds = buildPublicBracketRounds([
      slot({ id: 'a', round: 1, slot_index: 0, participant: athlete('Budi', 'Tapak Suci') }),
      slot({ id: 'b', round: 1, slot_index: 1, participant: athlete('Candra', 'PSHT') }),
      slot({ id: 'c', round: 2, slot_index: 0 }),
      slot({ id: 'd', round: 2, slot_index: 1 }),
    ])
    expect(rounds).toHaveLength(2)
    expect(rounds[0].title).toBe('BABAK SEMIFINAL')
    expect(rounds[1].title).toBe('BABAK FINAL')

    const match = rounds[0].matches[0]
    expect(match.red).toMatchObject({ name: 'Budi', contingent: 'Tapak Suci', isWinner: false })
    expect(match.blue).toMatchObject({ name: 'Candra', contingent: 'PSHT' })
    expect(match.statusBadge).toBe('SIAP')
    // ronde berikutnya belum terisi
    expect(rounds[1].matches[0].red.name).toBe('Pemenang Babak Sebelumnya')
  })

  it('menandai pemenang dan partai selesai', () => {
    const [round] = buildPublicBracketRounds([
      slot({ id: 'a', round: 1, slot_index: 0, participant: athlete('Budi'), is_winner: true }),
      slot({ id: 'b', round: 1, slot_index: 1, participant: athlete('Candra') }),
    ])
    const match = round.matches[0]
    expect(match.statusBadge).toBe('SELESAI')
    expect(match.status).toBe('selesai')
    expect(match.red).toMatchObject({ isWinner: true, score: 'W' })
    expect(match.blue).toMatchObject({ isWinner: false, score: 'L' })
  })

  it('menampilkan bye sebagai lolos otomatis', () => {
    const [round] = buildPublicBracketRounds([
      slot({ id: 'a', round: 1, slot_index: 0, participant: athlete('Budi'), is_winner: true }),
      slot({ id: 'b', round: 1, slot_index: 1, is_bye: true }),
    ])
    expect(round.matches[0].blue.name).toBe('BYE (Lolos Otomatis)')
  })

  it('mengurutkan slot berdasarkan slot_index meski datang acak', () => {
    const [round] = buildPublicBracketRounds([
      slot({ id: 'b', round: 1, slot_index: 1, participant: athlete('Kedua') }),
      slot({ id: 'a', round: 1, slot_index: 0, participant: athlete('Pertama') }),
    ])
    expect(round.matches[0].red.name).toBe('Pertama')
    expect(round.matches[0].blue.name).toBe('Kedua')
  })

  it('tidak menampilkan nomor gelanggang (data tidak tersedia untuk publik)', () => {
    const [round] = buildPublicBracketRounds([
      slot({ id: 'a', round: 1, slot_index: 0 }),
      slot({ id: 'b', round: 1, slot_index: 1 }),
    ])
    expect(round.matches[0].arenaLabel).toBe('')
  })
})
