import {
  test as base,
  expect,
  type Browser,
  type BrowserContext,
  type Page,
} from '@playwright/test'
import { adminToken } from '../../cypress/support/match/auth.ts'
import { must } from '../../cypress/support/match/http.ts'
import { buildKit } from '../../cypress/support/match/kit.ts'
import type {
  Devices as SocketDevices,
  MatchKit,
  MatchRef,
  Session,
} from '../../cypress/support/match/kit.ts'
import { confirmScore, snapshot } from '../../cypress/support/match/helpers.ts'
import type { Json } from '../../cypress/support/match/http.ts'

/** Ukuran layar desktop uji dari `.env` (PW_SCREEN_W/H). Bawaan 1280x800 bila tidak diset. */
export const SCREEN = {
  width: Number(process.env.PW_SCREEN_W ?? 1280),
  height: Number(process.env.PW_SCREEN_H ?? 800),
}

/** Viewport context baru: saat --headed ikut ukuran jendela (null) agar halaman tampil utuh, selain itu SCREEN. */
export const viewport = () => (base.info().project.use.headless === false ? null : SCREEN)

/** Empat perangkat pada satu partai: konsol wasit dan layar juri kursi 1-3. */
export interface Devices {
  kit: MatchKit
  match: MatchRef
  wasit: Page
  juri: Page[]
  /** Memulai partai lewat REST (bukan lewat klik) lalu menunggu ketiga layar juri BERJALAN. */
  startViaApi(): Promise<void>
}

export async function openAs(
  context: BrowserContext,
  s: Session,
  path: string,
  tile: ((page: Page) => Promise<void>) | undefined,
): Promise<Page> {
  await context.addInitScript((sess) => {
    sessionStorage.setItem(
      'ds_auth_session',
      JSON.stringify({
        token: sess.token,
        role: sess.role,
        seatNumber: sess.seat,
        arenaId: sess.arenaId,
        competitionId: sess.competitionId,
      }),
    )
    sessionStorage.setItem('ds_token', sess.token)
  }, s)
  const page = await context.newPage()
  await tile?.(page)
  await page.goto(path, { waitUntil: 'domcontentloaded' })
  return page
}

/** Konsol wasit + 3 layar juri (browser context terpisah) untuk gelanggang utama. */
export interface Crew {
  wasit: Page
  juri: Page[]
  close(): Promise<void>
}

/**
 * Membuka konsol wasit dan layar juri kursi 1-3. Mode berjendela (--headed): tiap jendela menempati satu kuadran
 * layar (grid 2x2). `redName` (bila diberikan) ditunggu muncul di konsol wasit sebagai tanda partai sudah dimuat.
 */
export async function openCrew(
  browser: Browser,
  kit: MatchKit,
  opts: { baseURL: string | undefined; headed: boolean; redName?: string },
): Promise<Crew> {
  let index = 0
  const tile = opts.headed
    ? async (page: Page) => {
        const slot = index++
        try {
          const { w, h } = await page.evaluate(() => ({
            w: window.screen.availWidth,
            h: window.screen.availHeight,
          }))
          const cdp = await page.context().newCDPSession(page)
          const { windowId } = await cdp.send('Browser.getWindowForTarget')
          await cdp.send('Browser.setWindowBounds', {
            windowId,
            bounds: {
              left: (slot % 2) * Math.floor(w / 2),
              top: Math.floor(slot / 2) * Math.floor(h / 2),
              width: Math.floor(w / 2),
              height: Math.floor(h / 2),
              windowState: 'normal',
            },
          })
        } catch {
          // penataan jendela opsional
        }
      }
    : undefined

  const contexts: BrowserContext[] = []
  const newContext = async () => {
    const c = await browser.newContext({
      baseURL: opts.baseURL,
      viewport: opts.headed ? null : SCREEN,
    })
    contexts.push(c)
    return c
  }

  const wasit = await openAs(await newContext(), kit.wasit, '/referee', tile)
  const juri: Page[] = []
  for (const j of kit.judges) juri.push(await openAs(await newContext(), j, '/judge', tile))

  if (opts.redName) {
    await expect(wasit.locator('.ref-fighter-card--red .ref-fighter-name')).toHaveText(
      opts.redName,
      { timeout: 15_000 },
    )
  }
  for (const p of juri) await expect(p.locator('.judge-online-pill')).toContainText('ONLINE')

  return {
    wasit,
    juri,
    close: async () => {
      await Promise.all(contexts.map((c) => c.close().catch(() => undefined)))
    },
  }
}

/**
 * Satu partai di gelanggang utama dengan konsol wasit (UI) + layar scoreboard publik (UI) + tiga juri lewat soket.
 * Untuk tes yang fokus pada tampilan wasit/penonton dan butuh poin juri tanpa membuka tiga browser lagi.
 */
export interface Arena {
  kit: MatchKit
  match: MatchRef
  wasit: Page
  scoreboard: Page
  sockets: SocketDevices
  /** Memulai partai lewat REST dan menunggu konsol wasit menampilkan BERJALAN. */
  start(): Promise<void>
  /** Dua juri menekan teknik yang sama pada sudut itu (konsensus) dan menunggu poin terkonfirmasi. */
  score(target: 'red' | 'blue', type?: string): Promise<void>
  /** Ringkasan keadaan partai dari server (sync_response). */
  state(): Promise<Json>
}

export const test = base.extend<{ devices: Devices; arena: Arena }, { kit: MatchKit }>({
  arena: async ({ browser, kit, baseURL }, use) => {
    const match = await kit.createMatch()
    await must('load-arena', kit.loadMatch(match))
    const sockets = await kit.openDevices()
    const wctx = await browser.newContext({ baseURL, viewport: viewport() })
    const wasit = await openAs(wctx, kit.wasit, '/referee', undefined)
    const sctx = await browser.newContext({ baseURL, viewport: viewport() })
    const scoreboard = await sctx.newPage()
    await scoreboard.goto(`/scoreboard/${kit.arenaId}`)
    await expect(wasit.locator('.ref-fighter-card--red .ref-fighter-name')).toHaveText(
      match.redName,
      { timeout: 15_000 },
    )
    await expect(scoreboard.locator('.sb-athlete-name').first()).toHaveText(match.redName, {
      timeout: 15_000,
    })

    await use({
      kit,
      match,
      wasit,
      scoreboard,
      sockets,
      async start() {
        await must('start', kit.control(match, 'start'))
        await expect(wasit.locator('.ref-chrono-round')).toContainText('BERJALAN')
      },
      async score(target, type = 'knockdown') {
        const before = sockets.wasit.mark()
        await confirmScore(sockets, target, type)
        await sockets.wasit.waitFor('score_confirmed', {
          since: before,
          what: 'poin terkonfirmasi',
        })
      },
      state: () => snapshot(sockets.wasit),
    })

    await sockets.closeAll()
    await Promise.all([wctx.close(), sctx.close()])
    await kit.control(match, 'cancel').catch(() => undefined)
  },

  // Satu kejuaraan + ruleset + PIN untuk seluruh sesi tes (dibersihkan di akhir)
  kit: [
    // Playwright mewajibkan pola destrukturisasi (kosong) untuk fixture tanpa dependensi
    // eslint-disable-next-line no-empty-pattern
    async ({}, use) => {
      const kit = await buildKit(await adminToken(), {
        label: 'PW 3 Juri',
        roundSeconds: 90,
        breakSeconds: 3,
        totalRounds: 2,
      })
      await use(kit)
      await kit.teardown().catch(() => undefined)
    },
    { scope: 'worker', timeout: 60_000 },
  ],

  // Tiap tes mendapat partai baru yang sudah dimuat ke gelanggang dan 4 browser context terpisah
  devices: async ({ browser, kit, baseURL }, use, testInfo) => {
    const headed = testInfo.project.use.headless === false
    const match = await kit.createMatch()
    await must('load-arena', kit.loadMatch(match))
    const crew = await openCrew(browser, kit, { baseURL, headed, redName: match.redName })
    const { wasit, juri } = crew

    await use({
      kit,
      match,
      wasit,
      juri,
      async startViaApi() {
        await must('start', kit.control(match, 'start'))
        for (const p of juri)
          await expect(p.locator('.judge-round-badge')).toContainText(/BABAK 1[\s\S]*BERJALAN/)
      },
    })

    await crew.close()
    await kit.control(match, 'cancel').catch(() => undefined)
  },
})

export { expect }
