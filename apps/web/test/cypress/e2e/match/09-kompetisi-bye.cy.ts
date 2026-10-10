import { adminToken } from '../../support/match/auth.ts'
import {
  bracketSlots,
  enrollAndVerify,
  generateBracket,
  listMatches,
  playBracketToEnd,
  type Entrant,
  type PlayedMatch,
} from '../../support/match/competition.ts'
import { api, must, type Json } from '../../support/match/http.ts'
import { buildKit } from '../../support/match/kit.ts'
import type { MatchKit } from '../../support/match/kit.ts'
import { runAsync } from '../../support/match/run.ts'

const LONG = 300_000

/**
 * Jumlah peserta yang BUKAN pangkat dua (BYE). Pada 5 dan 6 peserta bagan dulu macet: slot BYE ditaruh berurutan
 * sehingga ada pasangan BYE lawan BYE dan pemenang pasangan di sebelahnya tidak pernah mendapat lawan. Kini BYE
 * tersebar satu per pasangan. Tiap skenario dimainkan sampai juara: tidak boleh ada partai macet, total partai
 * harus N-1, dan medali mengikuti struktur (perunggu = yang kalah di semifinal).
 */
for (const n of [3, 5, 6, 7, 9, 12]) {
  describe(`Kompetisi ${n} peserta (BYE): dimainkan sampai juara`, { retries: 0 }, () => {
    let kit: MatchKit
    let entrants: Entrant[] = []
    let played: PlayedMatch[] = []

    before(() =>
      runAsync(async () => {
        kit = await buildKit(await adminToken(), {
          label: `Bye ${n}`,
          roundSeconds: 20,
          breakSeconds: 2,
          totalRounds: 2,
        })
        entrants = await enrollAndVerify(
          kit,
          Array.from({ length: n }, (_, i) => ({
            name: `Pesilat ${i + 1}`,
            team: `Kontingen ${String.fromCharCode(65 + i)}`,
          })),
        )
        expect((await generateBracket(kit)).status).to.eq(201)
      }, LONG),
    )
    after(() => runAsync(async () => kit?.teardown()))

    it('bagan: setiap peserta tepat sekali di ronde 1 dan tidak ada pasangan BYE lawan BYE', () =>
      runAsync(async () => {
        const r1 = (await bracketSlots(kit)).filter((s) => s.round === 1)
        r1.sort((a, b) => a.slot_index - b.slot_index)
        for (let i = 0; i < r1.length; i += 2) {
          expect(r1[i].is_bye && r1[i + 1].is_bye, `pasangan ${i}/${i + 1}`).to.eq(false)
        }
        const placed = r1.map((s) => s.participant_id).filter(Boolean)
        expect(placed).to.have.members(entrants.map((e) => e.participantId))
        expect(r1.filter((s) => s.is_bye)).to.have.length(r1.length - n)
      }, LONG))

    it('seluruh partai terbentuk dan dimainkan sampai habis: total N-1 partai, tidak ada yang macet', () =>
      runAsync(async () => {
        played = await playBracketToEnd(kit)
        expect(played, `${n - 1} partai untuk ${n} peserta`).to.have.length(n - 1)
        const all = await listMatches(kit)
        expect(all).to.have.length(n - 1)
        expect(all.every((m) => m.status === 'finished')).to.eq(true)
        // setiap peserta kalah paling banyak sekali, dan hanya satu yang tidak pernah kalah (juara)
        const losses = new Set(played.map((p) => p.loser))
        expect(losses.size).to.eq(n - 1)
        const champion = entrants.filter((e) => !losses.has(e.participantId))
        expect(champion).to.have.length(1)
      }, LONG))

    it('juara tercatat di bagan dan medali: 1 emas, 1 perak, perunggu = yang kalah di semifinal', () =>
      runAsync(async () => {
        const finalRound = Math.max(...played.map((p) => p.round))
        const final = played.find((p) => p.round === finalRound)!
        const slots = await bracketSlots(kit)
        const top = Math.max(...slots.map((s) => s.round))
        expect(
          slots.filter((s) => s.round === top).some((s) => s.winner_id === final.winner),
        ).to.eq(true)

        const team = (pid: string) => entrants.find((e) => e.participantId === pid)!.team
        const semiLosers = played.filter((p) => p.round === finalRound - 1).map((p) => p.loser)
        // 3 peserta: satu semifinal nyata (pemegang BYE langsung ke final); 4 peserta atau lebih: dua semifinal
        const expectedBronze = n === 3 ? 1 : 2
        const rows = await must(
          'medali',
          api('GET', `/competitions/${kit.competitionId}/report/medals`, kit.adminToken),
        )
        const by: Record<string, Json> = {}
        rows.forEach((r: Json) => (by[r.team] = r))
        const sum = (k: string) => rows.reduce((acc: number, r: Json) => acc + r[k], 0)
        expect([sum('gold'), sum('silver'), sum('bronze')]).to.deep.eq([1, 1, expectedBronze])
        expect(by[team(final.winner)]).to.include({ gold: 1 })
        expect(by[team(final.loser)]).to.include({ silver: 1 })
        expect(semiLosers).to.have.length(expectedBronze)
        semiLosers.forEach((pid) => expect(by[team(pid)]).to.include({ bronze: 1 }))
      }, LONG))
  })
}
