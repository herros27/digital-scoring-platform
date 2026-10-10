import { adminToken } from '../../support/match/auth.ts'
import {
  bracketSlots,
  enrollAndVerify,
  generateBracket,
  listMatches,
  playMatch,
  type Entrant,
} from '../../support/match/competition.ts'
import { api, must, type Json } from '../../support/match/http.ts'
import { buildKit } from '../../support/match/kit.ts'
import type { MatchKit } from '../../support/match/kit.ts'
import { runAsync } from '../../support/match/run.ts'

const LONG = 240_000

/**
 * SATU kompetisi penuh dari awal sampai juara: kejuaraan + kategori → pendaftaran 4 atlet dari 3 kontingen →
 * verifikasi/timbang → bagan → semifinal 1 & 2 (wasit + 3 juri sungguhan lewat WebSocket) → final otomatis terbentuk →
 * final dua babak habis waktu → juara di bagan publik → klasemen medali.
 * Langkah SENGAJA berurutan dan saling bergantung (state dibangun bertahap), jadi retry dimatikan.
 */
describe('Kompetisi penuh: dari pendaftaran sampai juara', { retries: 0 }, () => {
  let kit: MatchKit
  let entrants: Entrant[] = []
  const name = (e: Json) => entrants.find((x) => x.participantId === e)?.name
  let semi: Json[] = []
  let finalId = ''
  const winners: string[] = []
  const losers: string[] = []

  before(() =>
    runAsync(async () => {
      kit = await buildKit(await adminToken(), {
        label: 'Kompetisi Penuh',
        roundSeconds: 8,
        breakSeconds: 2,
        totalRounds: 2,
      })
    }),
  )
  after(() => runAsync(async () => kit?.teardown()))

  it('1. empat atlet dari tiga kontingen terdaftar dan lolos verifikasi/timbang', () =>
    runAsync(async () => {
      entrants = await enrollAndVerify(kit, [
        { name: 'Andi Garuda', team: 'Kontingen Garuda' },
        { name: 'Budi Garuda', team: 'Kontingen Garuda' },
        { name: 'Cahyo Rajawali', team: 'Kontingen Rajawali' },
        { name: 'Dimas Elang', team: 'Kontingen Elang' },
      ])
      const parts = await must(
        'peserta',
        api(
          'GET',
          `/competitions/${kit.competitionId}/participants?status=verified`,
          kit.adminToken,
        ),
      )
      expect(parts).to.have.length(4)
    }, LONG))

  it('2. bagan dibentuk: 4 peserta di babak 1, final kosong, dan 2 partai semifinal siap (belum ada final)', () =>
    runAsync(async () => {
      const gen = await generateBracket(kit)
      expect(gen.status).to.eq(201)
      const slots = await bracketSlots(kit)
      const round1 = slots.filter((s) => s.round === 1)
      expect(round1.map((s) => s.participant_id)).to.have.members(
        entrants.map((e) => e.participantId),
      )
      expect(slots.filter((s) => s.round === 2).every((s) => !s.participant_id)).to.eq(true)

      semi = await listMatches(kit)
      expect(semi, 'hanya dua semifinal yang dibuat otomatis').to.have.length(2)
      semi.forEach((m) => expect(m.status).to.eq('idle'))
      // setiap atlet muncul tepat sekali di semifinal
      const used = semi.flatMap((m) => [m.athlete_red_id, m.athlete_blue_id])
      expect(used).to.have.members(entrants.map((e) => e.participantId))
    }, LONG))

  it('3. bagan publik tersembunyi sebelum kejuaraan diaktifkan, lalu terbuka (tanpa login) dengan empat atlet', () =>
    runAsync(async () => {
      const url = `/public/competitions/${kit.competitionId}/brackets/${kit.categoryId}`
      expect((await api('GET', url, null)).status, 'belum aktif → tidak tampil').to.eq(404)
      await must(
        'aktifkan',
        api('PUT', `/competitions/${kit.competitionId}/activate`, kit.adminToken),
      )
      const res = await api('GET', url, null)
      expect(res.status).to.eq(200)
      const text = JSON.stringify(res.data)
      entrants.forEach((e) => expect(text).to.include(e.name))
    }, LONG))

  it('4. semifinal 1: merah unggul → wasit menyelesaikan, pemenang tercatat dan maju ke final', () =>
    runAsync(async () => {
      const m = semi[0]
      const { result } = await playMatch(kit, m.id, [{ target: 'red', type: 'knockdown' }])
      expect(result.corner).to.eq('red')
      winners.push(m.athlete_red_id)
      losers.push(m.athlete_blue_id)

      const rec = await must(
        'partai',
        api('GET', `/competitions/${kit.competitionId}/matches/${m.id}`, kit.adminToken),
      )
      expect(rec.status).to.eq('finished')
      expect(rec.winner_id).to.eq(m.athlete_red_id)
      const final = (await bracketSlots(kit)).filter((s) => s.round === 2)
      expect(final.map((s) => s.participant_id)).to.include(m.athlete_red_id)
    }, LONG))

  it('5. semifinal 2: biru unggul → final otomatis terbentuk dengan kedua pemenang', () =>
    runAsync(async () => {
      const m = semi[1]
      const { result } = await playMatch(kit, m.id, [{ target: 'blue', type: 'fall' }])
      expect(result.corner).to.eq('blue')
      winners.push(m.athlete_blue_id)
      losers.push(m.athlete_red_id)

      const all = await listMatches(kit)
      expect(all, 'semifinal 2 + final').to.have.length(3)
      const final = all.find((x) => !semi.some((s) => s.id === x.id))!
      finalId = final.id
      expect(final.status).to.eq('idle')
      expect([final.athlete_red_id, final.athlete_blue_id]).to.have.members(winners)
    }, LONG))

  it('6. final dua babak habis waktu: skor babak 1 + babak 2 dijumlah dan pemenang ditentukan otomatis', () =>
    runAsync(async () => {
      const final = (await listMatches(kit)).find((x) => x.id === finalId)!
      // babak 1: merah +3 (jatuhan); babak 2: biru +2 (tendangan) → merah 3 : biru 2
      const { result } = await playMatch(
        kit,
        finalId,
        [
          { target: 'red', type: 'fall' },
          { target: 'blue', type: 'knockdown' },
        ],
        { untilTimeout: true },
      )
      expect(result.corner).to.eq('red')
      const rec = await must(
        'final',
        api('GET', `/competitions/${kit.competitionId}/matches/${finalId}`, kit.adminToken),
      )
      expect(rec.status).to.eq('finished')
      expect(rec.winner_id).to.eq(final.athlete_red_id)
      winners.push(final.athlete_red_id)
      losers.push(final.athlete_blue_id)
    }, LONG))

  it('7. bagan akhir: juara tercatat, ketiga partai selesai, tidak ada partai tersisa', () =>
    runAsync(async () => {
      const all = await listMatches(kit)
      expect(all.every((m) => m.status === 'finished')).to.eq(true)
      expect(all).to.have.length(3)
      const slots = await bracketSlots(kit)
      const champion = winners[2]
      expect(
        slots.filter((s) => s.round === 2).some((s) => s.winner_id === champion),
        `juara ${name(champion)} tercatat pada slot final`,
      ).to.eq(true)
      // publik juga melihat juara
      const pub = await api(
        'GET',
        `/public/competitions/${kit.competitionId}/brackets/${kit.categoryId}`,
        null,
      )
      expect(JSON.stringify(pub.data)).to.include(name(champion)!)
    }, LONG))

  it('8. klasemen medali kontingen: tepat 1 emas (juara), 1 perak (finalis), 2 perunggu (kalah di semifinal)', () =>
    runAsync(async () => {
      const medals = await must(
        'medali',
        api('GET', `/competitions/${kit.competitionId}/report/medals`, kit.adminToken),
      )
      const teamOf = (pid: string) => entrants.find((e) => e.participantId === pid)!.team
      const expected: Record<string, { gold: number; silver: number; bronze: number }> = {}
      const add = (pid: string, k: 'gold' | 'silver' | 'bronze') => {
        const t = teamOf(pid)
        expected[t] ??= { gold: 0, silver: 0, bronze: 0 }
        expected[t][k] += 1
      }
      add(winners[2], 'gold')
      add(losers[2], 'silver')
      add(losers[0], 'bronze')
      add(losers[1], 'bronze')

      const actual: Record<string, Json> = {}
      for (const row of medals)
        actual[row.team] = { gold: row.gold, silver: row.silver, bronze: row.bronze }
      expect(actual).to.deep.eq(expected)
      const sum = (k: string) => Object.values(actual).reduce((n, v) => n + v[k], 0)
      expect([sum('gold'), sum('silver'), sum('bronze')]).to.deep.eq([1, 1, 2])
    }, LONG))
})
