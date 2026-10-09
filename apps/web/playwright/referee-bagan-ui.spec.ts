import { expect, test } from '@playwright/test'
import { adminToken } from '../cypress/support/match/auth.ts'
import { enrollAndVerify, generateBracket } from '../cypress/support/match/competition.ts'
import { buildKit } from '../cypress/support/match/kit.ts'
import type { MatchKit } from '../cypress/support/match/kit.ts'
import { openAs } from './fixtures.ts'

/**
 * Meja referee memakai bagan kategori (bukan per gelanggang): partai satu kategori tersebar di beberapa gelanggang,
 * jadi referee gelanggang utama tetap melihat seluruh bagan, dengan partai gelanggangnya saja yang ditandai.
 */
let kit: MatchKit

test.beforeAll(async () => {
  test.setTimeout(120_000)
  kit = await buildKit(await adminToken(), {
    label: 'PW Referee Bagan',
    roundSeconds: 90,
    breakSeconds: 3,
    totalRounds: 2,
  })
  await enrollAndVerify(
    kit,
    ['Rani', 'Sita', 'Tari', 'Umi'].map((n, i) => ({ name: `${n} RB`, team: `Tim ${i}` })),
  )
  await generateBracket(kit)
})
test.afterAll(async () => {
  await kit?.teardown().catch(() => undefined)
})

test('referee gelanggang utama melihat bagan kategori lintas gelanggang, partainya ditandai', async ({
  browser,
  baseURL,
}) => {
  const ctx = await browser.newContext({ baseURL, viewport: { width: 1600, height: 1000 } })
  const page = await openAs(ctx, kit.wasit, '/referee', undefined)
  await page.getByText('Bagan & Antrean').first().click()

  await expect(page.locator('.ref-select')).toContainText('Dewasa Putra')
  const cards = page.locator('.t-round-column .t-match-card')
  await expect(cards.first()).toBeVisible()
  // dua partai babak 1 di dua gelanggang + final: seluruh pohon tampil, bukan hanya gelanggang ini
  expect(await cards.count()).toBe(3)
  await expect(page.locator('.t-arena-tag', { hasText: 'GELANGGANG UTAMA' })).not.toHaveCount(0)
  await expect(page.locator('.t-arena-tag', { hasText: 'GELANGGANG LAIN' })).not.toHaveCount(0)
  // hanya partai gelanggang referee yang bertepi biru
  await expect(page.locator('.t-match-card--own-arena')).toHaveCount(1)
  await expect(page.locator('.t-match-card--own-arena .t-arena-tag')).toHaveText('GELANGGANG UTAMA')
  await expect(page.getByTestId('ref-bracket-note')).toContainText('Gelanggang Utama')
  await ctx.close()
})
