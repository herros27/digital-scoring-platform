import { expect, test } from '@playwright/test'
import { adminToken } from '../../../cypress/support/match/auth.ts'
import { api, must, type Json } from '../../../cypress/support/match/http.ts'
import { buildKit } from '../../../cypress/support/match/kit.ts'
import type { MatchKit } from '../../../cypress/support/match/kit.ts'
import { openAs, viewport } from '../../support/fixtures.ts'

/**
 * Waktu per golongan umur di editor ruleset: panitia bebas menentukan (tidak wajib 2 babak x 1,5 menit); golongan yang
 * tidak dicentang ikut waktu utama. Hasil diperiksa dari server.
 */
let kit: MatchKit
let admin = ''
const stamp = Date.now().toString(36)
const created: string[] = []

test.beforeAll(async () => {
  test.setTimeout(120_000)
  admin = await adminToken()
  kit = await buildKit(admin, {
    label: 'PW Waktu Umur',
    roundSeconds: 90,
    breakSeconds: 3,
    totalRounds: 2,
  })
})
test.afterAll(async () => {
  for (const id of created) await api('DELETE', `/rulesets/${id}`, admin).catch(() => undefined)
  await kit?.teardown().catch(() => undefined)
})

const findRuleset = async (name: string) =>
  (await must<Json[]>('daftar ruleset', api('GET', '/rulesets', admin))).find(
    (r) => r.name === name,
  )

test('panitia menentukan waktu Usia Dini sendiri (2 babak × 2 menit) di ruleset', async ({
  browser,
  baseURL,
}) => {
  const name = `Waktu Umur ${stamp}`
  const ctx = await browser.newContext({ baseURL, viewport: viewport() })
  const page = await openAs(
    ctx,
    { token: admin, role: 'admin', arenaId: '', competitionId: kit.competitionId },
    '/ops?tab=regulasi',
    undefined,
  )
  await page.locator('[data-test="new-ruleset"]').click()
  await page.locator('[data-test="name"]').fill(name)

  const row = page.locator('[data-test="age-usia_dini_1"]')
  await expect(row.locator('input[type="number"]')).toHaveCount(0) // belum dicentang: ikut waktu utama
  await row.locator('input[type="checkbox"]').check()
  const fields = row.locator('input[type="number"]')
  await expect(fields).toHaveCount(3)
  await fields.nth(0).fill('120') // durasi babak (detik)
  await fields.nth(1).fill('30') // istirahat
  await fields.nth(2).fill('2') // jumlah babak
  await page.locator('[data-test="save"]').click()

  await expect.poll(async () => !!(await findRuleset(name)), { timeout: 10_000 }).toBe(true)
  const rs = (await findRuleset(name))!
  created.push(rs.id)
  expect(rs.scoring.age_timings).toEqual({
    usia_dini_1: { round_duration: 120, break_duration: 30, total_rounds: 2 },
  })
  // golongan lain tidak dicantumkan: ikut waktu utama ruleset
  expect(Object.keys(rs.scoring.age_timings)).toEqual(['usia_dini_1'])
  await ctx.close()
})

test('usulan IPSI mengisi Usia Dini 1 dan 2 tetapi nilainya tetap bisa diubah', async ({
  browser,
  baseURL,
}) => {
  const ctx = await browser.newContext({ baseURL, viewport: viewport() })
  const page = await openAs(
    ctx,
    { token: admin, role: 'admin', arenaId: '', competitionId: kit.competitionId },
    '/ops?tab=regulasi',
    undefined,
  )
  await page.locator('[data-test="new-ruleset"]').click()
  await page.locator('[data-test="age-suggest"]').click()
  for (const code of ['usia_dini_1', 'usia_dini_2']) {
    const f = page.locator(`[data-test="age-${code}"] input[type="number"]`)
    await expect(f.nth(0)).toHaveValue('90')
    await expect(f.nth(2)).toHaveValue('2')
  }
  await page.locator('[data-test="age-usia_dini_2"] input[type="number"]').nth(0).fill('150')
  await expect(
    page.locator('[data-test="age-usia_dini_2"] input[type="number"]').nth(0),
  ).toHaveValue('150')
  await expect(page.locator('[data-test="age-dewasa"] input[type="number"]')).toHaveCount(0)
  await ctx.close()
})
