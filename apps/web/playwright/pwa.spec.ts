import { expect, test } from '@playwright/test'
import { adminToken } from '../cypress/support/match/auth.ts'
import { buildKit } from '../cypress/support/match/kit.ts'

/**
 * PWA untuk tablet juri: dapat dipasang ke layar utama, cangkang aplikasi tersedia offline, dan service worker tidak
 * pernah menyajikan data pertandingan dari cache. Dijalankan pada build produksi (lihat playwright.pwa.config.ts).
 */
test.describe('PWA', () => {
  test('manifest lengkap dan semua ikon terunduh dengan ukuran yang dijanjikan', async ({
    page,
  }) => {
    await page.goto('/judge/login')
    await expect(page.locator('link[rel="manifest"]')).toHaveCount(1)
    await expect(page.locator('meta[name="theme-color"]')).toHaveAttribute('content', '#0b1220')
    await expect(page.locator('link[rel="apple-touch-icon"]')).toHaveCount(1)

    const href = await page.locator('link[rel="manifest"]').getAttribute('href')
    const res = await page.request.get(href!)
    expect(res.status()).toBe(200)
    const m = await res.json()
    expect(m).toMatchObject({
      name: 'Digital Scoring Silat',
      short_name: 'DS Silat',
      start_url: '/judge',
      scope: '/',
      display: 'standalone',
      orientation: 'landscape',
      lang: 'id',
    })
    expect(m.shortcuts.map((s: { url: string }) => s.url)).toEqual([
      '/judge/login',
      '/referee',
      '/scoreboard',
    ])

    const purposes = m.icons.map(
      (i: { sizes: string; purpose: string }) => `${i.sizes}/${i.purpose}`,
    )
    expect(purposes).toEqual(
      expect.arrayContaining(['192x192/any', '512x512/any', '512x512/maskable']),
    )
    for (const icon of m.icons as Array<{ src: string; sizes: string }>) {
      const size = await page.evaluate(
        (src) =>
          new Promise<string>((resolve, reject) => {
            const img = new Image()
            img.onload = () => resolve(`${img.naturalWidth}x${img.naturalHeight}`)
            img.onerror = () => reject(new Error(`ikon tidak terunduh: ${src}`))
            img.src = src
          }),
        icon.src,
      )
      expect(size, icon.src).toBe(icon.sizes)
    }
    // favicon yang dirujuk index.html benar-benar ada (sebelumnya 404)
    expect((await page.request.get('/favicon.svg')).status()).toBe(200)
  })

  test('tidak bergantung pada internet: semua permintaan satu asal dan font lokal termuat', async ({
    page,
    baseURL,
  }) => {
    const outside: string[] = []
    page.on('request', (r) => {
      const u = r.url()
      if (
        !u.startsWith('data:') &&
        !u.startsWith('blob:') &&
        new URL(u).origin !== new URL(baseURL!).origin
      )
        outside.push(u)
    })
    for (const path of ['/judge/login', '/scoreboard', '/bagan']) {
      await page.goto(path)
      await page.waitForLoadState('networkidle')
    }
    expect(outside, `permintaan ke luar: ${outside.join(', ')}`).toEqual([])

    // font dimuat saat dipakai: kumpulkan dari beberapa halaman (login memakai sans, scoreboard memakai mono)
    const loaded = new Set<string>()
    for (const path of ['/judge/login', '/scoreboard']) {
      await page.goto(path)
      const fonts = await page.evaluate(async () => {
        await document.fonts.ready
        return [...document.fonts]
          .filter((f) => f.status === 'loaded')
          .map((f) => f.family.replace(/"/g, ''))
      })
      fonts.forEach((f) => loaded.add(f))
    }
    const fonts = [...loaded]
    expect(fonts).toEqual(expect.arrayContaining(['Space Grotesk', 'JetBrains Mono']))
  })

  test('service worker terdaftar dan aktif, dan Chrome menganggap aplikasi dapat dipasang (installable)', async ({
    page,
    context,
  }) => {
    await page.goto('/judge/login')
    await page.evaluate(() => navigator.serviceWorker.ready)
    await expect
      .poll(() =>
        page.evaluate(async () => (await navigator.serviceWorker.getRegistration())?.active?.state),
      )
      .toBe('activated')
    const scope = await page.evaluate(
      async () => (await navigator.serviceWorker.getRegistration())?.scope,
    )
    expect(scope).toMatch(/\/$/)

    const cdp = await context.newCDPSession(page)
    const { installabilityErrors } = await cdp.send('Page.getInstallabilityErrors')
    // Konteks uji berupa mode penyamaran sehingga Chrome selalu melaporkan "in-incognito"; itu bukan cacat aplikasi.
    // Setiap kriteria lain (manifest, ikon, service worker, konteks aman) harus terpenuhi.
    const real = installabilityErrors.filter((e) => e.errorId !== 'in-incognito')
    expect(real, JSON.stringify(real)).toEqual([])
  })

  test('offline: cangkang aplikasi tetap terbuka dari cache, tetapi data API tidak pernah disajikan dari cache', async ({
    page,
    context,
  }) => {
    await page.goto('/judge/login')
    await page.evaluate(() => navigator.serviceWorker.ready)
    // muat ulang agar halaman dikendalikan service worker, lalu putuskan jaringan
    await page.reload()
    await expect.poll(() => page.evaluate(() => !!navigator.serviceWorker.controller)).toBe(true)

    await context.setOffline(true)
    await page.goto('/judge/login')
    await expect(page.locator('button.keypad-btn').first()).toBeVisible()
    await expect(page.getByRole('button', { name: /^\s*Masuk\s*$/ })).toBeVisible()
    // rute klien lain juga jatuh ke cangkang aplikasi
    await page.goto('/scoreboard')
    await expect(page.locator('.sbl__head')).toBeVisible()

    // API tidak punya cache: tanpa jaringan harus GAGAL, bukan memberi data usang
    const outcome = await page.evaluate(async () => {
      try {
        const r = await fetch('/api/v1/public/competitions')
        return `status ${r.status}`
      } catch {
        return 'gagal'
      }
    })
    expect(outcome).toBe('gagal')
    await context.setOffline(false)
  })

  test('tablet juri yang memakai PWA tetap bisa login PIN dan terhubung realtime (API dan WebSocket tidak terhalang SW)', async ({
    page,
  }) => {
    test.setTimeout(90_000)
    const kit = await buildKit(await adminToken(), { label: 'PW PWA', roundSeconds: 30 })
    try {
      const match = await kit.createMatch()
      await kit.loadMatch(match)
      const pinRes = await fetch('http://127.0.0.1:8080/api/v1/auth/create-pin', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json', Authorization: `Bearer ${kit.adminToken}` },
        body: JSON.stringify({
          competition_id: kit.competitionId,
          arena_id: kit.arenaId,
          role: 'juri',
          seat_number: 5,
          ttl_hours: 2,
        }),
      })
      const { data } = (await pinRes.json()) as { data: { pin: string } }

      await page.goto('/judge/login')
      await page.evaluate(() => navigator.serviceWorker.ready)
      await page.reload()
      for (const d of data.pin)
        await page.locator('button.keypad-btn', { hasText: new RegExp(`^\\s*${d}\\s*$`) }).click()
      await page.getByRole('button', { name: /^\s*Masuk\s*$/ }).click()
      await expect(page).toHaveURL(/\/judge(\?|$)/)
      await expect(page.locator('.judge-main-title')).toContainText('JURI 5')
      await expect(page.locator('.judge-online-pill')).toContainText('ONLINE')
      await expect(page.locator('.judge-athlete-name').first()).toHaveText(match.redName)
    } finally {
      await kit.teardown().catch(() => undefined)
    }
  })
})
