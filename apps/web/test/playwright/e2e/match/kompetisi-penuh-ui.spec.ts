import { test as base, expect, type Page } from '@playwright/test'
import { ADMIN } from '../../../cypress/support/accounts.ts'
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
import { openCrew } from '../../support/fixtures.ts'

/**
 * SATU kompetisi penuh yang seluruh sisi pengguna-nya lewat LAYAR sungguhan:
 *   panitia login lewat form → mengaktifkan kejuaraan lewat tombol → penonton membuka /bagan di browser →
 *   tiap partai dimainkan dengan konsol wasit + 3 layar juri (klik tombol, bukan WebSocket) → penonton melihat juara.
 * Penyiapan data (kontingen/atlet/verifikasi/pembentukan bagan) dan "muat partai ke gelanggang" lewat API.
 */
const test = base
let kit: MatchKit
let entrants: Entrant[] = []
let compName = ''

test.beforeAll(async () => {
  test.setTimeout(120_000)
  kit = await buildKit(await adminToken(), {
    label: 'PW Kompetisi UI',
    roundSeconds: 90,
    breakSeconds: 3,
    totalRounds: 2,
  })
  const comp = await must(
    'kejuaraan',
    api('GET', `/competitions/${kit.competitionId}`, kit.adminToken),
  )
  compName = comp.name
  entrants = await enrollAndVerify(kit, [
    { name: 'Andi Garuda', team: 'Kontingen Garuda' },
    { name: 'Budi Garuda', team: 'Kontingen Garuda' },
    { name: 'Cahyo Rajawali', team: 'Kontingen Rajawali' },
    { name: 'Dimas Elang', team: 'Kontingen Elang' },
  ])
  expect((await generateBracket(kit)).status).toBe(201)
})
test.afterAll(async () => {
  await kit?.teardown().catch(() => undefined)
})

const nameOf = (pid: string) => entrants.find((e) => e.participantId === pid)!.name

/** Pada /bagan publik, pilih kejuaraan uji bila ada lebih dari satu kejuaraan aktif. */
async function chooseCompetition(page: Page) {
  const select = page.locator('#bgl-comp')
  if (await select.count()) await select.selectOption({ label: compName })
}

test('kompetisi penuh lewat layar: aktivasi panitia → bagan publik → 3 partai dengan wasit + 3 juri → juara', async ({
  browser,
  baseURL,
}, testInfo) => {
  test.setTimeout(300_000)
  const headed = testInfo.project.use.headless === false
  const admin = await (await browser.newContext({ baseURL })).newPage()
  const publik = await (await browser.newContext({ baseURL })).newPage()

  await test.step('bagan publik belum tampil sebelum kejuaraan diaktifkan', async () => {
    await publik.goto('/bagan')
    await expect(publik.locator('.bgl__head h1')).toHaveText('BAGAN PERTANDINGAN')
    await publik.waitForLoadState('networkidle')
    await expect(publik.locator('body')).not.toContainText(compName)
  })

  await test.step('panitia login lewat form dan masuk ke ruang operasi', async () => {
    await admin.goto('/login')
    await admin.locator('#login-email input').fill(ADMIN.email)
    await admin.locator('#login-password input').fill(ADMIN.password)
    await admin.getByRole('button', { name: 'Masuk ke Akun' }).click()
    await expect(admin).toHaveURL(/\/ops/)
  })

  await test.step('panitia mengaktifkan kejuaraan lewat tombol dan dialog konfirmasi', async () => {
    await admin.goto('/ops?tab=kompetisi')
    await admin.getByPlaceholder(/Cari kejuaraan/).fill(compName)
    const card = admin.locator('.comp-card', { hasText: compName })
    await expect(card).toHaveCount(1)
    await card.getByRole('button', { name: 'Aktifkan' }).click()
    await expect(admin.getByRole('alertdialog')).toContainText('Aktifkan Kejuaraan')
    await admin.getByRole('button', { name: 'Aktifkan Sekarang' }).click()
    await expect(admin.locator('.comp-toast')).toContainText('kini berstatus Aktif')
    // tombol Aktifkan berganti kontrol lain (jeda/selesai): kejuaraan sudah aktif
    await expect(card.getByRole('button', { name: 'Aktifkan' })).toHaveCount(0)
    const rec = await must(
      'status',
      api('GET', `/competitions/${kit.competitionId}`, kit.adminToken),
    )
    expect(rec.status).toBe('active')
  })

  await test.step('penonton membuka /bagan: kejuaraan dan bagan tampil, bagan berisi keempat atlet tanpa juara', async () => {
    await publik.goto('/bagan')
    await chooseCompetition(publik)
    const card = publik.locator('[data-test="bracket-card"]')
    await expect(card).toHaveCount(1)
    await card.click()
    await expect(publik).toHaveURL(new RegExp(`/bagan/${kit.competitionId}/${kit.categoryId}`))
    for (const e of entrants) await expect(publik.locator('main')).toContainText(e.name)
    await expect(publik.locator('[data-test="champion"]')).toHaveCount(0)
  })

  const results: Array<{ winner: string; loser: string; round: number }> = []
  const slots = await bracketSlots(kit)
  const roundOf = (slotId: string) => slots.find((s) => s.id === slotId)!.round as number

  for (const label of ['semifinal 1', 'semifinal 2', 'final']) {
    await test.step(`${label}: dimainkan dengan konsol wasit dan tiga layar juri sungguhan`, async () => {
      const match = (await listMatches(kit)).find(
        (m: Json) => m.status === 'idle' && slots.some((s) => s.id === m.bracket_slot_id),
      )
      expect(match, `partai ${label} tersedia`).toBeTruthy()
      const redName = nameOf(match.athlete_red_id)
      await must(
        'load-arena',
        kit.loadMatch({ id: match.id, redId: '', blueId: '', redName: '', blueName: '' }),
      )
      const crew = await openCrew(browser, kit, { baseURL, headed, redName })
      try {
        const { wasit, juri } = crew
        await wasit.locator('button.ref-cmd-btn--start').click()
        for (const p of juri) {
          await expect(p.locator('.judge-round-badge')).toContainText(/BABAK 1[\s\S]*BERJALAN/)
        }
        // dua juri menekan TENDANGAN merah bersamaan → konsensus → +2 untuk merah
        await Promise.all([
          juri[0].locator('.judge-btn--red').nth(1).click(),
          juri[1].locator('.judge-btn--red').nth(1).click(),
        ])
        await expect(wasit.locator('.ref-score-display--red .ref-score-num')).toHaveText('2')
        await wasit.locator('button.ref-cmd-btn--finish').click()
        await expect(wasit.locator('.ref-modal-box')).toBeVisible()
        await wasit.locator('button.ref-modal-confirm').first().click()
        await expect(wasit.locator('.ref-chrono-round')).toContainText('SELESAI')
        for (const p of juri) {
          await expect(p.locator('.judge-round-badge')).toContainText('SELESAI')
          await expect(p.locator('.judge-btn--red').first()).toBeDisabled()
        }
      } finally {
        await crew.close()
      }
      results.push({
        winner: match.athlete_red_id,
        loser: match.athlete_blue_id,
        round: roundOf(match.bracket_slot_id),
      })
    })
  }

  await test.step('penonton melihat juara di bagan (halaman publik di-cache sebentar, muat ulang sampai terbarui)', async () => {
    const champion = results[2].winner
    expect(results.map((r) => r.round)).toEqual([1, 1, 2])
    await expect(async () => {
      await publik.reload()
      await expect(publik.locator('[data-test="champion"]')).toContainText(nameOf(champion), {
        timeout: 2_000,
      })
    }).toPass({ timeout: 40_000, intervals: [2_000] })
  })

  await test.step('medali: 1 emas, 1 perak, 2 perunggu dengan tim yang benar', async () => {
    const rows = await must(
      'medali',
      api('GET', `/competitions/${kit.competitionId}/report/medals`, kit.adminToken),
    )
    const team = (pid: string) => entrants.find((e) => e.participantId === pid)!.team
    const sum = (k: string) => rows.reduce((n: number, r: Json) => n + r[k], 0)
    expect([sum('gold'), sum('silver'), sum('bronze')]).toEqual([1, 1, 2])
    const by = Object.fromEntries(rows.map((r: Json) => [r.team, r]))
    expect(by[team(results[2].winner)].gold).toBeGreaterThanOrEqual(1)
    expect(by[team(results[2].loser)].silver).toBeGreaterThanOrEqual(1)
  })
})
