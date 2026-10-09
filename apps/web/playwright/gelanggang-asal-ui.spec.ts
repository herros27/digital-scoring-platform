import { expect, test } from '@playwright/test'
import { adminToken } from '../cypress/support/match/auth.ts'
import { enrollAndVerify, generateBracket } from '../cypress/support/match/competition.ts'
import { api, must, type Json } from '../cypress/support/match/http.ts'
import { buildKit } from '../cypress/support/match/kit.ts'
import type { MatchKit } from '../cypress/support/match/kit.ts'
import { openAs } from './fixtures.ts'

/**
 * Gelanggang asal kategori: terisi otomatis saat bagan dibuat, semua partai (termasuk babak lanjutan) ikut gelanggang itu,
 * panitia dapat mengubahnya dari kartu kategori sebelum partai berjalan, wasit tidak dapat memuat partai gelanggang lain,
 * dan bagan tanpa gelanggang ditolak.
 */
let kit: MatchKit
let admin = ''

test.beforeAll(async () => {
  test.setTimeout(120_000)
  admin = await adminToken()
  kit = await buildKit(admin, {
    label: 'PW Gelanggang Asal',
    roundSeconds: 90,
    breakSeconds: 3,
    totalRounds: 2,
    otherArenaCrew: true,
  })
  await enrollAndVerify(
    kit,
    ['Ayu', 'Bima', 'Cici', 'Dodi'].map((n, i) => ({ name: `${n} GA`, team: `Tim ${i}` })),
  )
})
test.afterAll(async () => {
  await kit?.teardown().catch(() => undefined)
})

const category = async () =>
  (
    await must<Json[]>(
      'kategori',
      api('GET', `/competitions/${kit.competitionId}/categories`, admin),
    )
  ).find((c) => c.id === kit.categoryId)!
const matchArenas = async () =>
  (
    await must<Json[]>('partai', api('GET', `/competitions/${kit.competitionId}/matches`, admin))
  ).map((m) => m.arena_id as string)

test('gelanggang asal otomatis, dapat diubah dari kartu kategori, dan wasit lain ditolak', async ({
  browser,
  baseURL,
}) => {
  expect((await category()).home_arena_id ?? null).toBeNull()
  expect((await generateBracket(kit)).status).toBe(201)

  // otomatis: gelanggang bernama paling awal (utama); semua partai di sana
  expect((await category()).home_arena_id).toBe(kit.arenaId)
  expect(new Set(await matchArenas())).toEqual(new Set([kit.arenaId]))

  const ctx = await browser.newContext({ baseURL, viewport: { width: 1600, height: 1000 } })
  const page = await openAs(
    ctx,
    { token: admin, role: 'admin', arenaId: '', competitionId: kit.competitionId },
    '/ops?tab=kategori',
    undefined,
  )
  const select = page
    .locator('article.cat-card', { hasText: 'Dewasa Putra' })
    .locator('[data-test="category-home-arena"]')
  await expect(select).toHaveValue(kit.arenaId)

  await select.selectOption(kit.otherArenaId)
  await expect
    .poll(async () => (await category()).home_arena_id, { timeout: 10_000 })
    .toBe(kit.otherArenaId)
  expect(new Set(await matchArenas())).toEqual(new Set([kit.otherArenaId]))
  await ctx.close()

  // wasit gelanggang utama tidak bisa memuat partai yang kini di gelanggang lain
  const [first] = await must<Json[]>(
    'partai',
    api('GET', `/competitions/${kit.competitionId}/matches`, admin),
  )
  const res = await api(
    'POST',
    `/competitions/${kit.competitionId}/matches/${first.id}/load-arena`,
    kit.wasit.token,
    {
      arena_id: kit.arenaId,
      ruleset_path: kit.rulesetId,
    },
  )
  expect(res.status).toBe(403)
})
