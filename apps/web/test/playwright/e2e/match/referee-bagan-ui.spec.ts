import { expect, test } from '@playwright/test'
import { adminToken } from '../../../cypress/support/match/auth.ts'
import { enrollAndVerify, generateBracket } from '../../../cypress/support/match/competition.ts'
import { api, must } from '../../../cypress/support/match/http.ts'
import { buildKit } from '../../../cypress/support/match/kit.ts'
import type { MatchKit } from '../../../cypress/support/match/kit.ts'
import { viewport, openAs } from '../../support/fixtures.ts'

/**
 * Meja referee memakai bagan kategori (bukan per gelanggang): partai satu kategori tersebar di beberapa gelanggang,
 * jadi referee gelanggang utama tetap melihat seluruh bagan, dengan partai gelanggangnya saja yang ditandai.
 */
let kit: MatchKit
let admin = ''

test.beforeAll(async () => {
  test.setTimeout(120_000)
  admin = await adminToken()
  kit = await buildKit(admin, {
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

test('referee melihat bagan kategorinya; partai gelanggangnya ditandai, gelanggang lain hanya dilihat', async ({
  browser,
  baseURL,
}) => {
  const ctx = await browser.newContext({ baseURL, viewport: viewport() })
  const page = await openAs(ctx, kit.wasit, '/referee', undefined)
  await page.getByText('Bagan & Antrean').first().click()

  await expect(page.locator('.ref-select')).toContainText('Dewasa Putra')
  const cards = page.locator('.t-round-column .t-match-card')
  await expect(cards.first()).toBeVisible()
  // seluruh pohon tampil: 2 semifinal + final
  expect(await cards.count()).toBe(3)
  // kategori ini di gelanggang asal = gelanggang referee: kedua partai babak 1 bertepi biru, final belum dijadwalkan
  await expect(page.locator('.t-match-card--own-arena')).toHaveCount(2)
  await expect(page.locator('.t-match-card--own-arena .t-arena-tag').first()).toHaveText(
    'GELANGGANG UTAMA',
  )
  await expect(page.getByTestId('ref-bracket-note')).toContainText('Gelanggang Utama')

  // panitia memindahkan gelanggang asal kategori: bagan yang sama tampil di referee, tapi tidak lagi miliknya
  await must(
    'pindah gelanggang',
    api(
      'PUT',
      `/competitions/${kit.competitionId}/categories/${kit.categoryId}/home-arena`,
      admin,
      {
        arena_id: kit.otherArenaId,
      },
    ),
  )
  await page.reload()
  await page.getByText('Bagan & Antrean').first().click()
  await expect(page.locator('.t-round-column .t-match-card').first()).toBeVisible()
  await expect(page.locator('.t-match-card--own-arena')).toHaveCount(0)
  await expect(page.locator('.t-arena-tag', { hasText: 'GELANGGANG UTAMA 2' })).not.toHaveCount(0)
  await ctx.close()
})
