import { expect, test as base, type Browser, type Page } from '@playwright/test'
import { adminToken } from '../cypress/support/match/auth.ts'
import { confirmScore } from '../cypress/support/match/helpers.ts'
import {
  enrollAndVerify,
  listMatches,
  bracketSlots,
  type Entrant,
} from '../cypress/support/match/competition.ts'
import { api, must, type Json } from '../cypress/support/match/http.ts'
import { buildKit } from '../cypress/support/match/kit.ts'
import type { MatchKit } from '../cypress/support/match/kit.ts'
import { openAs } from './fixtures.ts'

/**
 * Ruang operasi PANITIA di browser nyata: membentuk bagan, mengelola jadwal (muat ke gelanggang, mulai, jeda,
 * selesaikan), dan menerbitkan/mengatur ulang PIN perangkat. Penyiapan atlet dan verifikasi lewat API; semua aksi
 * panitia lewat tombol di layar, dan hasilnya diperiksa dari server.
 */
const test = base
let kit: MatchKit
let adminTok = ''
let entrants: Entrant[] = []

test.beforeAll(async () => {
  test.setTimeout(120_000)
  adminTok = await adminToken()
  kit = await buildKit(adminTok, {
    label: 'PW Panitia',
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
})
test.afterAll(async () => {
  await kit?.teardown().catch(() => undefined)
})

/** Membuka ruang operasi sebagai admin pada tab tertentu dan memilih kejuaraan uji. */
async function openOps(browser: Browser, baseURL: string | undefined, tab: string): Promise<Page> {
  const ctx = await browser.newContext({ baseURL, viewport: { width: 1600, height: 1000 } })
  const page = await openAs(
    ctx,
    { token: adminTok, role: 'admin', arenaId: '', competitionId: '' },
    `/ops?tab=${tab}`,
    undefined,
  )
  await page.locator('.comp-dropdown').selectOption(kit.competitionId)
  return page
}

test.describe.configure({ mode: 'serial' })

test.describe('Panitia: bagan, jadwal, dan PIN', () => {
  test('bagan: draw terkunci sejak awal; buka kunci, Generate Bagan Acak membentuk bagan dan dua semifinal; acak ulang tidak menggandakan; kunci lagi mencegah acak', async ({
    browser,
    baseURL,
  }) => {
    const page = await openOps(browser, baseURL, 'bagan')
    await expect(page.locator('h1.ops-topbar__heading')).toContainText('BAGAN PERTANDINGAN')
    await page.locator('.filter-select').selectOption({ label: '[Dewasa Putra] - Kelas A' })
    await expect(page.getByText('Bagan Pertandingan Belum Di-Generate')).toBeVisible({
      timeout: 15_000,
    })

    // status kunci terbaca jelas dari label tombol
    await expect(page.locator('[data-test="draw-lock"]')).toContainText('Draw Terkunci')
    // draw terkunci sejak awal: Generate ditolak dengan peringatan dan tidak membuat apa pun
    await page.getByRole('button', { name: 'Generate Bagan Acak' }).first().click()
    await expect(page.getByText('Bagan Terkunci')).toBeVisible()
    expect(await listMatches(kit)).toHaveLength(0)

    // panitia membuka kunci (dengan konfirmasi), lalu generate (dengan konfirmasi)
    await page.locator('[data-test="draw-lock"]').click()
    await expect(page.getByText('Buka Kunci Legalitas Draw?')).toBeVisible()
    await page.getByRole('button', { name: 'Buka Kunci', exact: true }).last().click()
    await expect(page.getByText('Kunci Dibuka')).toBeVisible()
    await expect(page.locator('[data-test="draw-lock"]')).toContainText('Draw Terbuka')

    const generate = async () => {
      await page.getByRole('button', { name: 'Generate Bagan Acak' }).first().click()
      await expect(page.getByText('Acak Ulang Bagan Pertandingan?')).toBeVisible()
      await page.getByRole('button', { name: 'Ya, Generate Bagan' }).click()
      await expect(page.getByText('Bagan Berhasil Di-Generate')).toBeVisible({ timeout: 15_000 })
    }
    await generate()

    // bagan tampil di layar dengan semua atlet, dan tersimpan di server
    for (const n of ['Andi Garuda', 'Budi Garuda', 'Cahyo Rajawali', 'Dimas Elang']) {
      await expect(page.locator('main')).toContainText(n, { timeout: 15_000 })
    }
    expect((await bracketSlots(kit)).filter((s) => s.participant_id)).toHaveLength(4)
    expect(await listMatches(kit)).toHaveLength(2)

    // acak ulang tidak menggandakan partai
    await generate()
    expect(await listMatches(kit)).toHaveLength(2)

    // dikunci kembali: acak ditolak lagi
    await expect(page.locator('[data-test="draw-lock"]')).toContainText('Draw Terbuka') // status jelas
    await page.locator('[data-test="draw-lock"]').click()
    await expect(page.locator('[data-test="draw-lock"]')).toContainText('Draw Terkunci')
    await expect(page.getByText('Bagan Dikunci')).toBeVisible()
    await page.getByRole('button', { name: 'Generate Bagan Acak' }).first().click()
    await expect(page.getByText('Bagan Terkunci')).toBeVisible()
    await page.context().close()
  })

  test('dock bagan: "Kirim ke Gelanggang" memuat partai antrean ke gelanggang milik partai itu', async ({
    browser,
    baseURL,
  }) => {
    const page = await openOps(browser, baseURL, 'bagan')
    await page.locator('.filter-select').selectOption({ label: '[Dewasa Putra] - Kelas A' })
    await expect(page.locator('main')).toContainText('Andi Garuda', { timeout: 15_000 })
    const dockTitle = await page.getByText(/^PARTAI \d+:/).innerText()
    expect(dockTitle).toContain(' vs ')

    await page.getByRole('button', { name: 'Kirim ke Gelanggang' }).click()
    const dialog = await page.getByText(/^Muat Partai ke /).innerText()
    // partai yang tampil di dock adalah partai pertama; gelanggang di dialog harus gelanggang partai itu
    // cocokkan partai di dock berdasarkan nama atletnya, lalu bandingkan gelanggang di dialog dengan gelanggang partai itu
    const names = dockTitle
      .replace(/^PARTAI \d+:\s*/, '')
      .split(' vs ')
      .map((n) => n.replace(/\s*\(.*\)\s*$/, '').trim())
    const pid = (name: string) => entrants.find((e) => e.name === name)!.participantId
    const own = (await listMatches(kit)).find(
      (m: Json) => m.athlete_red_id === pid(names[0]) && m.athlete_blue_id === pid(names[1]),
    )
    expect(own, `partai ${names.join(' vs ')}`).toBeTruthy()
    const arenas = await must(
      'gelanggang',
      api('GET', `/competitions/${kit.competitionId}/arenas`, adminTok),
    )
    const ownArena = arenas.find((a: Json) => a.id === own.arena_id)
    expect(dialog, `partai ${names.join(' vs ')} milik ${ownArena?.name}`).toContain(ownArena.name)
    await page.context().close()
  })

  test('jadwal: hitung dan filter, muat ke gelanggang, mulai, jeda, lalu selesaikan partai', async ({
    browser,
    baseURL,
  }) => {
    test.setTimeout(120_000)
    const page = await openOps(browser, baseURL, 'jadwal')
    page.on('dialog', (d) => void d.accept()) // konfirmasi bawaan browser untuk "Selesaikan Partai"
    await expect(page.locator('h1.ops-topbar__heading')).toContainText('JADWAL PERTANDINGAN')
    await expect(page.getByText('Semua (2)', { exact: true })).toBeVisible({ timeout: 15_000 })
    await expect(page.getByText('Antrean (2)', { exact: true })).toBeVisible()
    await expect(page.getByText('Live (0)', { exact: true })).toBeVisible()

    // filter per gelanggang: hanya partai Gelanggang Utama yang tampil
    await page
      .locator('select')
      .filter({ hasText: 'Semua Gelanggang' })
      .selectOption({ label: 'Gelanggang Utama' })
    await expect(page.getByRole('button', { name: 'Muat ke Gelanggang' })).toHaveCount(1)
    const arenas = await must(
      'gelanggang',
      api('GET', `/competitions/${kit.competitionId}/arenas`, adminTok),
    )
    const utama = arenas.find((a: Json) => a.name === 'Gelanggang Utama')
    const first = (await listMatches(kit)).find((m: Json) => m.arena_id === utama.id)
    const card = page
      .locator('article, .sched-card, [class*=match-card]')
      .filter({ hasText: 'PARTAI' })
      .first()
    await expect(card).toBeVisible()

    // mulai sebelum dimuat ke gelanggang: ditolak dengan pesan, partai tetap antre
    await page.getByRole('button', { name: 'Mulai Partai' }).first().click()
    await expect(page.getByText('Gagal Memulai')).toBeVisible({ timeout: 15_000 })
    await expect(page.getByText('Antrean (2)', { exact: true })).toBeVisible()

    // muat ke gelanggang (tunggu konfirmasi), lalu mulai lewat tombol panitia
    await page.getByRole('button', { name: 'Muat ke Gelanggang' }).first().click()
    await expect(page.getByText('Dimuat ke Gelanggang', { exact: true })).toBeVisible({
      timeout: 15_000,
    })
    const devices = await kit.openDevices()
    try {
      await page.getByRole('button', { name: 'Mulai Partai' }).first().click()
      await devices.wasit.waitFor(
        (m) => m.type === 'state_change' && m.payload.state === 'round_1',
        {
          what: 'partai dimulai dari ruang operasi',
        },
      )
      await expect(page.getByText('Live (1)', { exact: true })).toBeVisible({
        timeout: 15_000,
      })
      await expect(page.getByRole('button', { name: 'Jeda (Pause)' })).toBeVisible()

      await page.getByRole('button', { name: 'Jeda (Pause)' }).click()
      await devices.wasit.waitFor(
        (m) => m.type === 'state_change' && m.payload.timer_running === false,
        { what: 'jeda dari ruang operasi' },
      )

      // skor juri ditolak selama dijeda; panitia melanjutkan lalu merah unggul agar penyelesaian tidak seri
      await page.getByRole('button', { name: 'Lanjutkan Partai' }).click()
      await devices.wasit.waitFor(
        (m) => m.type === 'state_change' && m.payload.timer_running === true,
        { since: devices.wasit.mark() - 1, what: 'lanjut dari ruang operasi' },
      )
      const before = devices.wasit.mark()
      await confirmScore(devices, 'red', 'knockdown')
      await devices.wasit.waitFor('score_confirmed', { since: before, what: 'poin merah' })
      await page.getByRole('button', { name: 'Selesaikan Partai' }).first().click()
      await devices.wasit.waitFor((m) => m.type === 'match_result', { what: 'hasil partai' })
      await expect(page.getByText('Selesai (1)', { exact: true })).toBeVisible({
        timeout: 15_000,
      })
    } finally {
      await devices.closeAll()
    }
    // partai tercatat selesai di server
    const rec = await must(
      'partai',
      api('GET', `/competitions/${kit.competitionId}/matches/${first.id}`, adminTok),
    )
    expect(rec.status).toBe('finished')
    await page.context().close()
  })

  test('PIN: panitia menerbitkan PIN juri kustom, tablet bisa login dengannya, Reset PIN mencabut PIN lama', async ({
    browser,
    baseURL,
  }) => {
    test.setTimeout(120_000)
    const page = await openOps(browser, baseURL, 'pin')
    await expect(page.locator('h1.ops-topbar__heading')).toContainText('PERANGKAT & PIN JURI')
    const pin = String(100000 + Math.floor(Math.random() * 899999))

    await page.getByRole('button', { name: 'Buat PIN Kustom' }).click()
    const modal = page.locator('.pin-modal')
    await expect(modal).toContainText('BUAT PIN OTORISASI BARU')
    await modal.locator('select').nth(0).selectOption({ label: 'GELANGGANG UTAMA' })
    await modal.locator('select').nth(1).selectOption({ label: 'Juri Lapangan' })
    await modal.locator('select').nth(2).selectOption({ label: 'Kursi Juri 4' })
    await modal.locator('.pin-form-input').fill(pin)
    await modal.getByRole('button', { name: 'Simpan & Terapkan PIN' }).click()
    const spaced = `${pin.slice(0, 3)} ${pin.slice(3)}` // kartu menampilkan PIN berspasi
    const cardOf = () => page.locator('.pin-device-card').filter({ hasText: spaced })
    await expect(cardOf()).toBeVisible({
      timeout: 15_000,
    })

    // status NYATA: belum ada tablet yang terhubung, jadi kartu OFFLINE dan header menunjukkan 0 juri online
    const status = cardOf().locator('[data-test="seat-status"]')
    await expect(status).toHaveText('OFFLINE', { timeout: 15_000 })
    await expect(page.locator('[data-test="juri-online"]')).toHaveText('0 JURI ONLINE')
    await expect(page.getByText('SERVER ONLINE')).toBeVisible()

    // tablet memakai PIN itu di keypad dan masuk sebagai juri kursi 4
    const tablet = await (await browser.newContext({ baseURL })).newPage()
    await loginWithPin(tablet, pin)
    await expect(tablet).toHaveURL(/\/judge(\?|$)/)
    await expect(tablet.locator('.judge-main-title')).toContainText('JURI 4')
    await expect(tablet.locator('.judge-online-pill')).toContainText('ONLINE')

    // kartu dan header beralih ONLINE sendiri (polling kehadiran 5 dtk), kursi lain tetap OFFLINE
    await expect(status).toHaveText('ONLINE', { timeout: 20_000 })
    await expect(page.locator('[data-test="juri-online"]')).toHaveText('1 JURI ONLINE')
    await expect(
      page
        .locator('.pin-device-card')
        .filter({ hasText: 'KURSI JURI 1' })
        .locator('[data-test="seat-status"]'),
    ).toHaveText('OFFLINE')

    // tablet ditutup: kembali OFFLINE
    await tablet.context().close()
    await expect(status).toHaveText('OFFLINE', { timeout: 20_000 })
    await expect(page.locator('[data-test="juri-online"]')).toHaveText('0 JURI ONLINE')

    // Reset PIN mengganti PIN pada kartu itu dan PIN lama tidak berlaku lagi
    await expect(cardOf()).toContainText('KURSI JURI 4')
    await cardOf().getByRole('button', { name: 'Reset PIN' }).click()
    await expect(page.getByText('PIN Diperbarui')).toBeVisible({ timeout: 15_000 })
    await expect(cardOf()).toHaveCount(0)

    const old = await (await browser.newContext({ baseURL })).newPage()
    await loginWithPin(old, pin)
    await expect(old).not.toHaveURL(/\/judge(\?|$)/)
    await expect(old.locator('body')).toContainText(/PIN|tidak valid|salah|kedaluwarsa/i)
    await old.context().close()
    await page.context().close()
  })
})

async function loginWithPin(page: Page, pin: string) {
  await page.goto('/judge/login')
  for (const d of pin)
    await page.locator('button.keypad-btn', { hasText: new RegExp(`^\\s*${d}\\s*$`) }).click()
  await page.getByRole('button', { name: /^\s*Masuk\s*$/ }).click()
  await page.waitForTimeout(1500)
}

export type { Json }
