import { expect, test } from './fixtures.ts'

const redScore = (p: import('@playwright/test').Page) =>
  p.locator('.sb-corner--red .sb-giant-score')
const blueScore = (p: import('@playwright/test').Page) =>
  p.locator('.sb-corner--blue .sb-giant-score')

/**
 * Layar scoreboard TV (publik, tanpa login) di browser nyata: mengikuti partai lewat WebSocket penonton. Aksi dilakukan
 * wasit lewat konsol sungguhan dan juri lewat soket; yang diverifikasi adalah apa yang DILIHAT penonton.
 */
test.describe('Scoreboard TV', () => {
  test('daftar publik "Sedang Berlangsung" memuat partai live dan membuka layar gelanggang', async ({
    arena,
    browser,
    baseURL,
  }) => {
    await arena.start()
    const anon = await (await browser.newContext({ baseURL })).newPage()
    await anon.goto('/scoreboard')
    const card = anon.locator('.sbl-card').filter({ hasText: arena.match.redName })
    await expect(card).toBeVisible({ timeout: 15_000 })
    await expect(card).toContainText(arena.match.blueName)
    await expect(card).toContainText('0 : 0')
    await card.click()
    await expect(anon).toHaveURL(new RegExp(`/scoreboard/${arena.kit.arenaId}`))
    await expect(anon.locator('.sb-athlete-name').first()).toHaveText(arena.match.redName)
  })

  test('layar menampilkan atlet, kontingen, jumlah babak sesuai ruleset (2), dan timer berjalan turun', async ({
    arena,
  }) => {
    const sb = arena.scoreboard
    await expect(sb.locator('.sb-athlete-name').nth(1)).toHaveText(arena.match.blueName)
    await expect(sb.locator('.sb-corner--red')).toContainText('Klub Merah')
    await expect(sb.locator('.sb-corner--blue')).toContainText('Klub Biru')
    // ruleset uji hanya 2 babak: tidak boleh ada babak ke-3 hantu
    await expect(sb.locator('.sb-round-chip')).toHaveCount(2)
    await expect(sb.locator('.sb-round-card')).toHaveCount(4) // 2 per sudut
    await expect(sb.locator('.sb-round-chip').first()).toContainText('MENUNGGU')

    await arena.start()
    await expect(sb.locator('.sb-round-chip').first()).toContainText('AKTIF')
    const t1 = await sb.locator('.sb-giant-timer').innerText()
    await expect(sb.locator('.sb-giant-timer')).not.toHaveText(t1, { timeout: 4_000 })
  })

  test('poin juri muncul realtime di sudut yang benar dan banner pemimpin berubah', async ({
    arena,
  }) => {
    const sb = arena.scoreboard
    await arena.start()
    await expect(sb.locator('.sb-consensus-box')).toContainText('SKOR IMBANG')

    await arena.score('red', 'knockdown') // +2 merah
    await expect(redScore(sb)).toHaveText('2')
    await expect(blueScore(sb)).toHaveText('0')
    await expect(sb.locator('.sb-corner--red .sb-round-card').first()).toContainText('2')
    await expect(sb.locator('.sb-consensus-box')).toContainText('MERAH MEMIMPIN')

    await arena.score('blue', 'fall') // +3 biru
    await expect(blueScore(sb)).toHaveText('3')
    await expect(redScore(sb)).toHaveText('2')
    await expect(sb.locator('.sb-consensus-box')).toContainText('BIRU MEMIMPIN')
  })

  test('sanksi dari konsol wasit: potongan tampil di scoreboard dan total sudut berkurang', async ({
    arena,
  }) => {
    const sb = arena.scoreboard
    await arena.start()
    await arena.score('red', 'fall') // merah 3
    await expect(redScore(sb)).toHaveText('3')

    await arena.wasit.getByRole('button', { name: 'Sanksi & Disiplin' }).click()
    await arena.wasit
      .getByRole('button', { name: /Teguran 1/ })
      .first()
      .click() // kolom merah, -1
    await expect(sb.locator('.sb-corner--red .sb-pen-chip')).toContainText('-1')
    await expect(redScore(sb)).toHaveText('2')
    await expect(sb.locator('.sb-corner--blue .sb-pen-chip')).toContainText('0')
    await expect(blueScore(sb)).toHaveText('0')
  })

  test('jeda dan lanjut dari wasit: chip babak berubah DIJEDA lalu AKTIF dan timer membeku', async ({
    arena,
  }) => {
    const sb = arena.scoreboard
    await arena.start()
    await arena.wasit.locator('button.ref-cmd-btn--pause').click()
    await expect(sb.locator('.sb-round-chip').first()).toContainText('DIJEDA')
    const frozen = await sb.locator('.sb-giant-timer').innerText()
    await sb.waitForTimeout(1_600)
    await expect(sb.locator('.sb-giant-timer')).toHaveText(frozen)
    await arena.wasit.locator('button.ref-cmd-btn--resume').click()
    await expect(sb.locator('.sb-round-chip').first()).toContainText('AKTIF')
  })

  test('sinyal dokter dari wasit tampil sebagai banner di scoreboard', async ({ arena }) => {
    const sb = arena.scoreboard
    await arena.start()
    await expect(sb.locator('.sb-signal-banner')).toHaveCount(0)
    await arena.wasit.getByRole('button', { name: /DOKTER PERTANDINGAN/ }).click()
    await expect(sb.locator('.sb-signal-banner')).toBeVisible()
  })

  test('SELESAIKAN oleh wasit: layar penonton menampilkan pemenang, skor akhir, dan dapat ditutup', async ({
    arena,
  }) => {
    const sb = arena.scoreboard
    await arena.start()
    await arena.score('blue', 'knockdown')
    await expect(blueScore(sb)).toHaveText('2')
    await arena.wasit.locator('button.ref-cmd-btn--finish').click()
    await arena.wasit.locator('button.ref-modal-confirm').first().click()

    const overlay = sb.locator('.sb-winner-overlay')
    await expect(overlay).toBeVisible()
    await expect(overlay).toContainText('PEMENANG PERTANDINGAN')
    await expect(overlay.locator('.sb-winner-name')).toHaveText(arena.match.blueName)
    await expect(overlay).toContainText('SUDUT BLUE')
    await expect(overlay.locator('.sb-winner-scores')).toContainText('0')
    await expect(overlay.locator('.sb-winner-scores')).toContainText('2')
    await overlay.locator('.sb-winner-close').click()
    await expect(overlay).toHaveCount(0)
    // setelah ditutup skor akhir tetap terlihat
    await expect(blueScore(sb)).toHaveText('2')
  })
})
