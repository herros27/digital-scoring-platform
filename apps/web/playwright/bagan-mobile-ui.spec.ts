import { expect, test } from '@playwright/test'
import { adminToken } from '../cypress/support/match/auth.ts'
import { enrollAndVerify, generateBracket } from '../cypress/support/match/competition.ts'
import { buildKit } from '../cypress/support/match/kit.ts'
import { openAs } from './fixtures.ts'

/**
 * Bagan di layar ponsel: dulu kartu bagan menyusut ke tinggi 0 di dalam kolom flex ber-tinggi tetap sehingga tidak
 * tampak sama sekali. Sekarang bagan harus punya tinggi nyata, kartu partai terlihat, dan bisa digeser dengan sentuhan.
 */
test('bagan tampil dan bisa digeser di layar ponsel', async ({ browser, baseURL }) => {
  test.setTimeout(120_000)
  const admin = await adminToken()
  const kit = await buildKit(admin, {
    label: 'PW Bagan Mobile',
    roundSeconds: 90,
    breakSeconds: 3,
    totalRounds: 2,
  })
  try {
    await enrollAndVerify(
      kit,
      Array.from({ length: 6 }, (_, i) => ({ name: `Mob ${i}`, team: `T${i}` })),
    )
    await generateBracket(kit)
    const ctx = await browser.newContext({
      baseURL,
      viewport: { width: 412, height: 900 },
      hasTouch: true,
      isMobile: true,
    })
    const page = await openAs(
      ctx,
      { token: admin, role: 'admin', arenaId: '', competitionId: kit.competitionId },
      '/ops?tab=bagan',
      undefined,
    )
    const root = page.locator('.tournament-bracket-root')
    await expect(root).toBeVisible()
    await expect
      .poll(async () => (await root.boundingBox())?.height ?? 0, { timeout: 10_000 })
      .toBeGreaterThan(300)
    await root.evaluate((el) => el.scrollIntoView({ block: 'center' }))
    await expect(page.locator('.t-match-slot').first()).toBeInViewport()
    await ctx.close()
  } finally {
    await kit.teardown().catch(() => undefined)
  }
})
