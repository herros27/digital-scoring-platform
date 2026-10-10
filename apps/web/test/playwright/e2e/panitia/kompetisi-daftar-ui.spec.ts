import { expect, test } from '@playwright/test'
import { adminToken } from '../../../cypress/support/match/auth.ts'
import { api, must, type Json } from '../../../cypress/support/match/http.ts'
import { viewport, openAs } from '../../support/fixtures.ts'

/**
 * Halaman Kompetisi: angka pada tab dan kartu ringkasan harus sama di semua filter (hitungan global dari server),
 * daftar dipaginasi (tidak berhenti di 20 pertama), tab Dibatalkan ada, dan pencarian mencakup seluruh data.
 */
test.describe.configure({ mode: 'serial' })

const tag = `PWDaftar${Date.now().toString(36)}`
let admin = ''
const ids: string[] = []

test.beforeAll(async () => {
  test.setTimeout(120_000)
  admin = await adminToken()
  const stats = await must<Json>('stats', api('GET', '/competitions/stats', admin))
  // pastikan ada lebih dari satu halaman (20/halaman) dan setidaknya satu dari tiap status uji
  const need = Math.max(3, 22 - stats.total)
  for (let i = 0; i < need; i++) {
    const c = await must<Json>(
      `buat ${i}`,
      api('POST', '/competitions', admin, {
        name: `${tag} ${i}`,
        location: 'Arena Uji',
        start_date: '2026-06-01',
        end_date: '2026-06-02',
      }),
    )
    ids.push(c.id)
  }
  await must('batal', api('POST', `/competitions/${ids[0]}/cancel`, admin))
})

test.afterAll(async () => {
  for (const id of ids)
    await api('POST', `/competitions/${id}/cancel`, admin).catch(() => undefined)
})

async function openKompetisi(browser: import('@playwright/test').Browser, baseURL?: string) {
  const ctx = await browser.newContext({ baseURL, viewport: viewport() })
  const page = await openAs(
    ctx,
    { token: admin, role: 'admin', arenaId: '', competitionId: '' },
    '/ops?tab=kompetisi',
    undefined,
  )
  await expect(page.locator('.tab-btn').first()).toBeVisible()
  return page
}

test('angka tab tetap sama di semua filter dan cocok dengan server', async ({
  browser,
  baseURL,
}) => {
  const stats = await must<Json>('stats', api('GET', '/competitions/stats', admin))
  expect(stats.draft + stats.active + stats.paused + stats.completed + stats.cancelled).toBe(
    stats.total,
  )
  const page = await openKompetisi(browser, baseURL)
  const tabs = page.locator('.tab-btn')
  const snapshot = async () =>
    (await tabs.allTextContents()).map((t) => t.replace(/\s+/g, ' ').trim())
  const awal = await snapshot()
  expect(awal[0]).toContain(`Semua (${stats.total})`)
  expect(awal).toContain(`Dibatalkan (${stats.cancelled})`)
  expect(awal).toContain(`Draft (${stats.draft})`)

  for (const label of ['Draft', 'Dibatalkan', 'Aktif', 'Semua']) {
    await tabs.filter({ hasText: new RegExp(`\\b${label} \\(`) }).click()
    await expect(tabs.filter({ hasText: new RegExp(`\\b${label} \\(`) })).toHaveClass(
      /tab-btn--active/,
    )
    expect(await snapshot()).toEqual(awal)
  }
  await page.context().close()
})

test('daftar dipaginasi: halaman 2 berisi data lain dan total sesuai server', async ({
  browser,
  baseURL,
}) => {
  const stats = await must<Json>('stats', api('GET', '/competitions/stats', admin))
  expect(stats.total).toBeGreaterThan(20)
  const page = await openKompetisi(browser, baseURL)
  const cards = page.locator('.comp-card')
  await expect(cards).toHaveCount(20)
  await expect(page.getByTestId('comp-pager-info')).toContainText(`1–20 dari ${stats.total}`)
  const first = await cards.first().innerText()
  await expect(page.getByTestId('comp-pager-prev')).toBeDisabled()
  await page.getByTestId('comp-pager-next').click()
  await expect(page.getByTestId('comp-pager-info')).toContainText(`21–`)
  await expect(cards.first()).not.toHaveText(first)
  await page.getByTestId('comp-pager-prev').click()
  await expect(page.getByTestId('comp-pager-info')).toContainText('1–20')
  await page.context().close()
})

test('tab Dibatalkan menampilkan kejuaraan batal dan pencarian mencari di seluruh data', async ({
  browser,
  baseURL,
}) => {
  const page = await openKompetisi(browser, baseURL)
  await page
    .locator('.tab-btn')
    .filter({ hasText: /Dibatalkan \(/ })
    .click()
  await page.getByPlaceholder('Cari kejuaraan, venue, atau ID...').fill(`${tag} 0`)
  const cards = page.locator('.comp-card')
  await expect(cards).toHaveCount(1)
  await expect(cards.first()).toContainText(`${tag} 0`)
  // pindah ke Semua: pencarian tetap, hasil tetap satu; angka tab tidak berubah
  await page
    .locator('.tab-btn')
    .filter({ hasText: /Semua \(/ })
    .click()
  await expect(cards).toHaveCount(1)
  // pencarian menjangkau kejuaraan di luar halaman pertama
  await page.getByPlaceholder('Cari kejuaraan, venue, atau ID...').fill(`${tag} 2`)
  await expect(cards.first()).toContainText(`${tag} 2`)
  await page.context().close()
})
