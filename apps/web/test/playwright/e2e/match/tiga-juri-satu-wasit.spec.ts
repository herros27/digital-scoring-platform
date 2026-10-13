import { expect, test } from '../../support/fixtures.ts'
import type { Page } from '@playwright/test'

const red = (p: Page) => p.locator('.judge-btn--red').first()
const blue = (p: Page) => p.locator('.judge-btn--blue').first()
const consensus = (p: Page, side: 'red' | 'blue') =>
  p.locator(`[data-test="consensus-text-${side}"]`)
const score = (wasit: Page, side: 'red' | 'blue') =>
  wasit.locator(`.ref-score-display--${side} .ref-score-num`)

/**
 * Pertandingan dengan 3 layar juri + 1 konsol wasit (browser context terpisah, UI sungguhan, backend nyata).
 * Tiap tes berdiri sendiri (partai baru), jadi dapat dipilih dan dijalankan satu per satu di UI Mode.
 */
test.describe('Pertandingan: 3 juri + 1 wasit', () => {
  test('semua perangkat online: tiga layar juri ONLINE dan wasit melihat 3 juri', async ({
    devices,
  }) => {
    const { wasit, juri } = devices
    await expect(wasit.locator('body')).toContainText(/JURI ONLINE: 3\//)
    for (const [i, p] of juri.entries()) {
      await expect(p.locator('.judge-main-title')).toContainText(`JURI ${i + 1}`)
      await expect(p.locator('.judge-round-badge')).toContainText('MENUNGGU')
    }
  })

  test('wasit klik MULAI RONDE: ketiga layar juri BABAK 1 BERJALAN dan timer turun', async ({
    devices,
  }) => {
    const { wasit, juri } = devices
    await wasit.locator('button.ref-cmd-btn--start').click()
    for (const p of juri)
      await expect(p.locator('.judge-round-badge')).toContainText(/BABAK 1[\s\S]*BERJALAN/)
    await expect(wasit.locator('.ref-chrono-round')).toContainText('BERJALAN')
    const before = await Promise.all(juri.map((p) => p.locator('.judge-timer-badge').innerText()))
    for (const [i, p] of juri.entries())
      await expect(p.locator('.judge-timer-badge')).not.toHaveText(before[i], { timeout: 4_000 })
  })

  test('tiga juri menekan tombol merah yang sama bersamaan: tepat SATU poin', async ({
    devices,
  }) => {
    const { wasit, juri } = devices
    await devices.startViaApi()
    await Promise.all(juri.map((p) => red(p).click()))
    await expect(score(wasit, 'red')).toHaveText('1')
    // tekanan ketiga hanya menunggu di window 3 detik lalu hangus; tidak boleh jadi poin kedua
    await wasit.waitForTimeout(3_500)
    await expect(score(wasit, 'red')).toHaveText('1')
  })

  test('satu juri saja menekan biru: semua layar "MENUNGGU JURI LAIN", tanpa poin, lampu padam sendiri', async ({
    devices,
  }) => {
    const { wasit, juri } = devices
    await devices.startViaApi()
    await blue(juri[1]).click()
    for (const p of juri) await expect(consensus(p, 'blue')).toContainText('MENUNGGU JURI LAIN')
    await expect(score(wasit, 'blue')).toHaveText('0')
    for (const p of juri) await expect(consensus(p, 'blue')).toHaveCount(0, { timeout: 6_000 })
    await expect(score(wasit, 'blue')).toHaveText('0')
  })

  test('BATAL menarik tekanan yang menunggu: lampu padam di semua layar dan suara itu tidak dihitung', async ({
    devices,
  }) => {
    const { wasit, juri } = devices
    await devices.startViaApi()
    await blue(juri[2]).click()
    for (const p of juri) await expect(consensus(p, 'blue')).toContainText('1/2')
    await juri[2].locator('.judge-batal-btn').click()
    for (const p of juri) await expect(consensus(p, 'blue')).toHaveCount(0)
    await blue(juri[0]).click() // baru 1 suara: yang dibatalkan tidak ikut dihitung
    await expect(consensus(juri[1], 'blue')).toContainText('1/2')
    await expect(score(wasit, 'blue')).toHaveText('0')
  })

  test('BATAL tanpa tekanan menunggu: layar menampilkan pesan, bukan diam', async ({ devices }) => {
    const { juri } = devices
    await devices.startViaApi()
    await juri[0].locator('.judge-batal-btn').click()
    await expect(juri[0].locator('.judge-notice')).toContainText('dibatalkan')
  })

  test('wasit JEDA: ketiga layar terkunci dan menampilkan DIJEDA; LANJUTKAN membukanya lagi', async ({
    devices,
  }) => {
    const { wasit, juri } = devices
    await devices.startViaApi()
    await wasit.locator('button.ref-cmd-btn--pause').click()
    for (const p of juri) {
      await expect(p.locator('.judge-round-badge')).toContainText('DIJEDA')
      await expect(p.locator('.judge-paused-note')).toBeVisible()
      await expect(red(p)).toBeDisabled()
      await expect(blue(p)).toBeDisabled()
    }
    await wasit.locator('button.ref-cmd-btn--resume').click()
    for (const p of juri) {
      await expect(p.locator('.judge-round-badge')).toContainText('BERJALAN')
      await expect(red(p)).toBeEnabled()
    }
  })

  test('dua juri menekan biru bersamaan: poin biru terkonfirmasi di semua layar', async ({
    devices,
  }) => {
    const { wasit, juri } = devices
    await devices.startViaApi()
    await Promise.all([blue(juri[0]).click(), blue(juri[2]).click()])
    for (const p of juri) await expect(consensus(p, 'blue')).toContainText('TERKONFIRMASI')
    await expect(score(wasit, 'blue')).toHaveText('1')
    await expect(score(wasit, 'red')).toHaveText('0')
  })

  test('wasit SELESAIKAN: ketiga layar juri SELESAI dan tombol skor terkunci', async ({
    devices,
  }) => {
    const { wasit, juri } = devices
    await devices.startViaApi()
    await Promise.all([red(juri[1]).click(), red(juri[2]).click()]) // merah unggul, tanpa seri
    await expect(score(wasit, 'red')).toHaveText('1')
    await wasit.locator('button.ref-cmd-btn--finish').click()
    await expect(wasit.locator('.ref-modal-box')).toBeVisible()
    await wasit.locator('button.ref-modal-confirm').first().click()
    await expect(wasit.locator('.ref-chrono-round')).toContainText('SELESAI')
    for (const p of juri) {
      await expect(p.locator('.judge-round-badge')).toContainText('SELESAI')
      await expect(red(p)).toBeDisabled()
      await expect(blue(p)).toBeDisabled()
    }
  })
})
