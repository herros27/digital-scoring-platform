import { expect, test as base, type Page } from '@playwright/test'
import { adminToken } from '../../../cypress/support/match/auth.ts'
import {
  bracketSlots,
  enrollAndVerify,
  generateBracket,
  listMatches,
  type Entrant,
} from '../../../cypress/support/match/competition.ts'
import { api, must, type Json } from '../../../cypress/support/match/http.ts'
import { buildKit } from '../../../cypress/support/match/kit.ts'
import type { MatchKit } from '../../../cypress/support/match/kit.ts'
import { viewport, openAs, test } from '../../support/fixtures.ts'

type Side = 'red' | 'blue'

/** Hasil khusus (KO/TKO/WO/Undur diri): tombol di konsol → dialog pilih sudut pemenang → konfirmasi. */
async function declare(wasit: Page, button: RegExp, winner: Side) {
  await wasit.getByRole('button', { name: button }).click()
  await expect(wasit.locator('.ref-modal-box')).toBeVisible()
  await wasit.locator(`.ref-modal-box input[value="${winner}"]`).check()
  await wasit.locator('.ref-modal-box .ref-modal-confirm').click()
}

const sanctionsTab = (wasit: Page) =>
  wasit.getByRole('button', { name: 'Sanksi & Disiplin' }).click()

/**
 * Sanksi dan hasil khusus lewat konsol wasit sungguhan. Efeknya diverifikasi dari server (skor, daftar sanksi, pemenang)
 * dan dari tampilan penonton, bukan hanya dari konsol itu sendiri.
 */
test.describe('Sanksi dari konsol wasit', () => {
  test('Binaan tidak memotong poin; Teguran 1 memotong 1 dan Teguran 2 memotong 2 (menumpuk)', async ({
    arena,
  }) => {
    await arena.start()
    await arena.score('blue', 'fall') // biru 3
    await sanctionsTab(arena.wasit)

    const blueCol = (name: RegExp) => arena.wasit.getByRole('button', { name }).nth(1) // kolom kedua = biru
    await blueCol(/Binaan 1/).click()
    await expect.poll(async () => (await arena.state()).blue_total).toBe(3)

    await blueCol(/Teguran 1/).click()
    await expect.poll(async () => (await arena.state()).blue_total).toBe(2)
    await blueCol(/Teguran 2/).click()
    await expect.poll(async () => (await arena.state()).blue_total).toBe(0) // 3 - 1 - 2
    await expect(arena.scoreboard.locator('.sb-corner--blue .sb-pen-chip')).toContainText('-3')
    expect((await arena.state()).red_total).toBe(0)
  })

  test('sanksi yang sama tidak bisa dijatuhkan dua kali dan tombolnya menandai yang sudah dipakai', async ({
    arena,
  }) => {
    await arena.start()
    await sanctionsTab(arena.wasit)
    const teguran = arena.wasit.getByRole('button', { name: /Teguran 1/ }).first()
    await teguran.click()
    await expect(teguran).toContainText('✓')
    await teguran.click({ force: true }).catch(() => undefined)
    const penalties = (await arena.state()).penalties as Json[]
    expect(penalties.filter((p) => p.target === 'red' && p.type === 'teguran_1')).toHaveLength(1)
    expect((await arena.state()).red_total).toBe(-1)
  })

  test('BATALKAN SANKSI mengembalikan potongan terakhir', async ({ arena }) => {
    await arena.start()
    await sanctionsTab(arena.wasit)
    await arena.wasit
      .getByRole('button', { name: /Teguran 2/ })
      .first()
      .click()
    await expect.poll(async () => (await arena.state()).red_total).toBe(-2)
    const undo = arena.wasit.locator('.ref-undo-sanc-btn').first() // kolom merah
    await expect(undo).toContainText('BATALKAN: TEGURAN 2')
    await undo.click()
    await expect(undo).toBeDisabled() // tidak ada sanksi tersisa
    await expect.poll(async () => (await arena.state()).red_total).toBe(0)
    await expect(arena.scoreboard.locator('.sb-corner--red .sb-pen-chip')).toContainText('0')
  })

  test('Peringatan 3 (diskualifikasi) pada sudut merah: partai berakhir dan sudut biru menang', async ({
    arena,
  }) => {
    await arena.start()
    await arena.score('red', 'fall') // merah unggul, tetap kalah karena didiskualifikasi
    await sanctionsTab(arena.wasit)
    await arena.wasit
      .getByRole('button', { name: /Peringatan 3/ })
      .first()
      .click()
    const confirm = arena.wasit.locator('.ref-modal-box .ref-modal-confirm')
    if (await confirm.isVisible().catch(() => false)) await confirm.click()

    await expect.poll(async () => (await arena.state()).state, { timeout: 10_000 }).toBe('finished')
    const result = (await arena.state()).result
    expect(result.corner).toBe('blue')
    expect(String(result.result_type)).toMatch(/disqual|diskual/i)
    const rec = await must(
      'partai',
      api(
        'GET',
        `/competitions/${arena.kit.competitionId}/matches/${arena.match.id}`,
        arena.kit.adminToken,
      ),
    )
    expect(rec.status).toBe('finished')
    expect(rec.winner_id).toBe(await winnerParticipant(arena.kit, arena.match.id, 'blue'))
    await expect(arena.scoreboard.locator('.sb-winner-overlay .sb-winner-name')).toHaveText(
      arena.match.blueName,
    )
  })

  test('tombol DISKUALIFIKASI BIRU: merah menang', async ({ arena }) => {
    await arena.start()
    await sanctionsTab(arena.wasit)
    await arena.wasit.getByRole('button', { name: 'DISKUALIFIKASI BIRU' }).click()
    const confirm = arena.wasit.locator('.ref-modal-box .ref-modal-confirm')
    if (await confirm.isVisible().catch(() => false)) await confirm.click()
    await expect.poll(async () => (await arena.state()).state, { timeout: 10_000 }).toBe('finished')
    expect((await arena.state()).result.corner).toBe('red')
  })
})

/** ID peserta pada sudut tertentu sebuah partai. */
async function winnerParticipant(kit: MatchKit, matchId: string, side: Side): Promise<string> {
  const m = await must(
    'partai',
    api('GET', `/competitions/${kit.competitionId}/matches/${matchId}`, kit.adminToken),
  )
  return side === 'red' ? m.athlete_red_id : m.athlete_blue_id
}

test.describe('Hasil khusus dari konsol wasit', () => {
  const cases: Array<{ type: string; button: RegExp; winner: Side; started: boolean }> = [
    { type: 'ko', button: /MENANG MUTLAK/, winner: 'blue', started: true },
    { type: 'tko', button: /MENANG TEKNIK/, winner: 'red', started: true },
    { type: 'withdrawal', button: /UNDUR DIRI/, winner: 'blue', started: true },
    { type: 'walkover', button: /MENANG WO/, winner: 'red', started: false },
  ]
  for (const c of cases) {
    test(`${c.type}: sudut ${c.winner} menang, tercatat di server, konsol, dan layar penonton`, async ({
      arena,
    }) => {
      if (c.started) await arena.start()
      await declare(arena.wasit, c.button, c.winner)

      await expect(arena.wasit.locator('.ref-result-banner')).toContainText(
        `PEMENANG SUDUT ${c.winner === 'red' ? 'MERAH' : 'BIRU'}`,
      )
      const snap = await arena.state()
      expect(snap.state).toBe('finished')
      expect(snap.result.corner).toBe(c.winner)
      expect(snap.result.result_type).toBe(c.type)
      const rec = await must(
        'partai',
        api(
          'GET',
          `/competitions/${arena.kit.competitionId}/matches/${arena.match.id}`,
          arena.kit.adminToken,
        ),
      )
      expect(rec.status).toBe('finished')
      expect(rec.winner_id).toBe(await winnerParticipant(arena.kit, arena.match.id, c.winner))
      const name = c.winner === 'red' ? arena.match.redName : arena.match.blueName
      await expect(arena.scoreboard.locator('.sb-winner-overlay .sb-winner-name')).toHaveText(name)
    })
  }

  test('KO tidak tersedia sebelum partai dimulai dan WO tidak tersedia saat partai berjalan', async ({
    arena,
  }) => {
    await expect(arena.wasit.getByRole('button', { name: /MENANG MUTLAK/ })).toBeDisabled()
    await expect(arena.wasit.getByRole('button', { name: /MENANG WO/ })).toBeEnabled()
    await arena.start()
    await expect(arena.wasit.getByRole('button', { name: /MENANG MUTLAK/ })).toBeEnabled()
    await expect(arena.wasit.getByRole('button', { name: /MENANG WO/ })).toBeDisabled()
  })

  test('dialog hasil khusus: tombol konfirmasi terkunci sampai pemenang dipilih, dan Batal tidak mengubah apa pun', async ({
    arena,
  }) => {
    await arena.start()
    await arena.wasit.getByRole('button', { name: /MENANG TEKNIK/ }).click()
    const confirm = arena.wasit.locator('.ref-modal-box .ref-modal-confirm')
    await expect(confirm).toBeDisabled()
    await arena.wasit.locator('.ref-modal-box .ref-modal-cancel').click()
    await expect(arena.wasit.locator('.ref-modal-box')).toHaveCount(0)
    expect((await arena.state()).state).toBe('round_1')
  })
})

/**
 * Hasil khusus memengaruhi bagan dan medali: pemenang KO/WO/Undur diri maju ke babak berikutnya dan medali
 * mengikuti. Kejuaraan sendiri dengan 4 peserta (2 semifinal + final).
 */
const turnamen = base
turnamen.describe('Hasil khusus dalam bagan: pemenang maju dan medali dihitung', () => {
  let kit: MatchKit
  let entrants: Entrant[] = []

  turnamen.beforeAll(async () => {
    turnamen.setTimeout(120_000)
    kit = await buildKit(await adminToken(), {
      label: 'PW Hasil Khusus',
      roundSeconds: 90,
      breakSeconds: 3,
      totalRounds: 2,
    })
    entrants = await enrollAndVerify(kit, [
      { name: 'Andi Garuda', team: 'Kontingen Garuda' },
      { name: 'Budi Garuda', team: 'Kontingen Garuda' },
      { name: 'Cahyo Rajawali', team: 'Kontingen Rajawali' },
      { name: 'Dimas Elang', team: 'Kontingen Elang' },
    ])
    expect((await generateBracket(kit)).status).toBe(201)
  })
  turnamen.afterAll(async () => {
    await kit?.teardown().catch(() => undefined)
  })

  turnamen(
    'semifinal KO + semifinal WO + final undur diri → juara, finalis, dan dua perunggu',
    async ({ browser, baseURL }) => {
      turnamen.setTimeout(180_000)
      const slots = await bracketSlots(kit)
      const team = (pid: string) => entrants.find((e) => e.participantId === pid)!.team

      /** Memainkan satu partai bagan yang sedang idle dan mengembalikan pemenang/pecundangnya. */
      async function play(how: 'ko' | 'wo' | 'withdrawal', winner: Side) {
        const m = (await listMatches(kit)).find(
          (x: Json) => x.status === 'idle' && slots.some((s) => s.id === x.bracket_slot_id),
        )
        expect(m, 'ada partai idle di bagan').toBeTruthy()
        await must(
          'load',
          kit.loadMatch({ id: m.id, redId: '', blueId: '', redName: '', blueName: '' }),
        )
        const ctx = await browser.newContext({ baseURL, viewport: viewport() })
        const wasit = await openAs(ctx, kit.wasit, '/referee', undefined)
        await expect(wasit.locator('.ref-fighter-card--red')).toBeVisible({ timeout: 15_000 })
        if (how !== 'wo') await wasit.locator('button.ref-cmd-btn--start').click()
        await declare(
          wasit,
          how === 'ko' ? /MENANG MUTLAK/ : how === 'wo' ? /MENANG WO/ : /UNDUR DIRI/,
          winner,
        )
        await expect(wasit.locator('.ref-result-banner')).toContainText('PEMENANG SUDUT')
        await ctx.close()
        const won = winner === 'red' ? m.athlete_red_id : m.athlete_blue_id
        const lost = winner === 'red' ? m.athlete_blue_id : m.athlete_red_id
        return { id: m.id, won, lost }
      }

      const s1 = await play('ko', 'red')
      let finalSlots = (await bracketSlots(kit)).filter((s) => s.round === 2)
      expect(
        finalSlots.map((s) => s.participant_id),
        'pemenang KO maju ke final',
      ).toContain(s1.won)

      const s2 = await play('wo', 'blue')
      finalSlots = (await bracketSlots(kit)).filter((s) => s.round === 2)
      expect(finalSlots.map((s) => s.participant_id)).toEqual(
        expect.arrayContaining([s1.won, s2.won]),
      )

      const idle = (await listMatches(kit)).filter((m: Json) => m.status === 'idle')
      expect(idle, 'final terbentuk otomatis tepat satu').toHaveLength(1)
      expect([idle[0].athlete_red_id, idle[0].athlete_blue_id].sort()).toEqual(
        [s1.won, s2.won].sort(),
      )

      const fin = await play('withdrawal', 'blue')
      const slotsEnd = (await bracketSlots(kit)).filter((s) => s.round === 2)
      expect(
        slotsEnd.some((s) => s.winner_id === fin.won),
        'juara tercatat di bagan',
      ).toBe(true)

      const rows = await must(
        'medali',
        api('GET', `/competitions/${kit.competitionId}/report/medals`, kit.adminToken),
      )
      const expected: Record<string, { gold: number; silver: number; bronze: number }> = {}
      const add = (pid: string, k: 'gold' | 'silver' | 'bronze') => {
        expected[team(pid)] ??= { gold: 0, silver: 0, bronze: 0 }
        expected[team(pid)][k] += 1
      }
      add(fin.won, 'gold')
      add(fin.lost, 'silver')
      add(s1.lost, 'bronze')
      add(s2.lost, 'bronze')
      const actual = Object.fromEntries(
        rows.map((r: Json) => [r.team, { gold: r.gold, silver: r.silver, bronze: r.bronze }]),
      )
      expect(actual).toEqual(expected)
    },
  )
})
