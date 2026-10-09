/**
 * E2E Playwright: 3 layar juri + 1 konsol wasit, masing-masing browser context terpisah (sesi sendiri),
 * semuanya UI sungguhan terhadap backend + Vite yang berjalan. Berbeda dari Cypress (satu tab per tes), di sini
 * ketiga juri menekan tombol yang SAMA bersamaan. Data dibuat lewat API (tanpa mock) memakai support/match.
 *
 *   npm run test:pw-match
 *   PW_CHROMIUM=/path/chrome E2E_BASE_URL=http://127.0.0.1:5173 npm run test:pw-match
 */
import { chromium, type Browser, type Page } from 'playwright-core'
import { adminToken } from '../cypress/support/match/auth.ts'
import { must, sleep } from '../cypress/support/match/http.ts'
import { buildKit, type Session } from '../cypress/support/match/kit.ts'

const BASE = process.env.E2E_BASE_URL ?? 'http://127.0.0.1:5173'
const CHROMIUM = process.env.PW_CHROMIUM ?? '/opt/pw-browsers/chromium-1194/chrome-linux/chrome'

let failed = 0
async function step(name: string, fn: () => Promise<void>) {
  try {
    await fn()
    console.log('  ✓', name)
  } catch (e) {
    failed += 1
    console.log('  ✗', name, '\n     ', String((e as Error).message).split('\n')[0])
  }
}
function assert(cond: unknown, msg: string): asserts cond {
  if (!cond) throw new Error(msg)
}

/** Menunggu teks elemen cocok pola (polling), dengan pesan yang menyebut isi terakhir bila gagal. */
async function waitText(page: Page, sel: string, re: RegExp, timeout = 8000): Promise<string> {
  const t0 = Date.now()
  for (;;) {
    const t = (
      await page
        .locator(sel)
        .first()
        .innerText()
        .catch(() => '')
    ).replace(/\s+/g, ' ')
    if (re.test(t)) return t
    if (Date.now() - t0 > timeout) throw new Error(`${sel}: "${t}" tidak cocok ${re}`)
    await sleep(120)
  }
}
const allPages = (pages: Page[], fn: (p: Page, i: number) => Promise<unknown>) =>
  Promise.all(pages.map(fn))

async function openAs(browser: Browser, s: Session, path: string): Promise<Page> {
  const ctx = await browser.newContext({ viewport: { width: 1280, height: 800 } })
  await ctx.addInitScript((sess) => {
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
  const page = await ctx.newPage()
  await page.goto(BASE + path, { waitUntil: 'domcontentloaded' })
  return page
}

const kit = await buildKit(await adminToken(), {
  label: 'PW 3 Juri',
  roundSeconds: 90,
  breakSeconds: 3,
  totalRounds: 2,
})
const browser = await chromium.launch({ executablePath: CHROMIUM, args: ['--no-sandbox'] })
const match = await kit.createMatch()
await must('load-arena', kit.loadMatch(match))

console.log('== Playwright: 3 juri + 1 wasit (UI sungguhan) ==')
const wasit = await openAs(browser, kit.wasit, '/referee')
const juri = await Promise.all(kit.judges.map((j) => openAs(browser, j, '/judge')))
const red = (p: Page) => p.locator('.judge-btn--red').first()
const blue = (p: Page) => p.locator('.judge-btn--blue').first()
const redScore = () => waitText(wasit, '.ref-score-display--red .ref-score-num', /\d+/)

try {
  await step('semua perangkat online: 3 layar juri ONLINE dan wasit melihat 3 juri', async () => {
    await allPages(juri, (p) => waitText(p, '.judge-online-pill', /ONLINE/, 10000))
    await waitText(
      wasit,
      '.ref-fighter-card--red .ref-fighter-name',
      new RegExp(match.redName),
      10000,
    )
    await waitText(wasit, 'body', /JURI ONLINE: 3\//, 8000)
    await allPages(juri, async (p, i) => {
      const title = await p.locator('.judge-main-title').innerText()
      assert(title.includes(`JURI ${i + 1}`), `judul layar ${i + 1}: ${title}`)
    })
  })

  await step(
    'wasit klik MULAI RONDE → ketiga layar juri BABAK 1 BERJALAN dengan timer turun',
    async () => {
      await wasit.locator('button.ref-cmd-btn--start').click()
      await allPages(juri, (p) => waitText(p, '.judge-round-badge', /BABAK 1[\s\S]*BERJALAN/))
      await waitText(wasit, '.ref-chrono-round', /BERJALAN/)
      const before = await allPages(juri, (p) => p.locator('.judge-timer-badge').innerText())
      await sleep(1600)
      const after = await allPages(juri, (p) => p.locator('.judge-timer-badge').innerText())
      before.forEach((b, i) => assert(b !== after[i], `timer juri ${i + 1} tidak bergerak (${b})`))
    },
  )

  await step(
    'tiga juri menekan tombol merah yang sama BERSAMAAN → tepat SATU poin (tekanan ketiga tidak jadi poin kedua)',
    async () => {
      await allPages(juri, (p) => red(p).click())
      await waitText(wasit, '.ref-score-display--red .ref-score-num', /^1$/)
      // tekanan ketiga hanya menunggu di window lalu hangus; tidak boleh menjadi poin kedua
      await sleep(3500)
      assert((await redScore()) === '1', 'tiga tekanan harus menjadi SATU poin')
    },
  )

  await step(
    'hanya satu juri menekan biru → layar semua juri "MENUNGGU JURI LAIN", tidak ada poin, lampu padam sendiri',
    async () => {
      await sleep(1700) // tunggu tanda terkonfirmasi sebelumnya hilang
      await blue(juri[1]).click()
      await allPages(juri, (p) =>
        waitText(p, '[data-test="consensus-text-blue"]', /MENUNGGU JURI LAIN/, 5000),
      )
      assert(
        (await wasit.locator('.ref-score-display--blue .ref-score-num').innerText()).trim() === '0',
        'skor biru harus 0',
      )
      await allPages(juri, async (p) => {
        const t0 = Date.now()
        while ((await p.locator('[data-test="consensus-text-blue"]').count()) > 0) {
          assert(Date.now() - t0 < 6000, 'lampu konsensus tidak padam setelah window')
          await sleep(150)
        }
      })
      assert(
        (await wasit.locator('.ref-score-display--blue .ref-score-num').innerText()).trim() === '0',
        'biru berubah',
      )
    },
  )

  await step(
    'BATAL: juri 3 menekan lalu BATAL → lampu padam di semua layar dan juri lain tidak menghasilkan poin',
    async () => {
      await blue(juri[2]).click()
      await allPages(juri, (p) => waitText(p, '[data-test="consensus-text-blue"]', /1\/2/, 5000))
      await juri[2].locator('.judge-batal-btn').click()
      await allPages(juri, async (p) => {
        const t0 = Date.now()
        while ((await p.locator('[data-test="consensus-text-blue"]').count()) > 0) {
          assert(Date.now() - t0 < 4000, 'lampu tidak padam setelah BATAL')
          await sleep(100)
        }
      })
      await blue(juri[0]).click() // satu suara lagi: tetap 1/2 karena yang dibatalkan tidak dihitung
      await waitText(juri[1], '[data-test="consensus-text-blue"]', /1\/2/, 5000)
      assert(
        (await wasit.locator('.ref-score-display--blue .ref-score-num').innerText()).trim() === '0',
        'biru berubah',
      )
      await sleep(3200)
    },
  )

  await step(
    'wasit JEDA → ketiga layar juri terkunci dan menampilkan DIJEDA; LANJUTKAN membukanya lagi',
    async () => {
      await wasit.locator('button.ref-cmd-btn--pause').click()
      await allPages(juri, (p) => waitText(p, '.judge-round-badge', /DIJEDA/))
      await allPages(juri, async (p, i) => {
        assert(await p.locator('.judge-paused-note').isVisible(), `keterangan jeda juri ${i + 1}`)
        assert(await red(p).isDisabled(), `tombol merah juri ${i + 1} harus terkunci`)
        assert(await blue(p).isDisabled(), `tombol biru juri ${i + 1} harus terkunci`)
      })
      await wasit.locator('button.ref-cmd-btn--resume').click()
      await allPages(juri, (p) => waitText(p, '.judge-round-badge', /BERJALAN/))
      await allPages(juri, async (p) =>
        assert(!(await red(p).isDisabled()), 'tombol harus aktif lagi'),
      )
    },
  )

  await step(
    'dua juri menekan biru bersamaan → poin biru; skor wasit merah 1 : biru 1',
    async () => {
      await Promise.all([blue(juri[0]).click(), blue(juri[2]).click()])
      await allPages(juri, (p) =>
        waitText(p, '[data-test="consensus-text-blue"]', /TERKONFIRMASI/, 6000),
      )
      await waitText(wasit, '.ref-score-display--blue .ref-score-num', /^1$/)
      assert((await redScore()) === '1', 'merah harus tetap 1')
    },
  )

  await step(
    'dua juri menekan merah bersamaan → merah 2 : biru 1 (menang jelas, tanpa seri)',
    async () => {
      await sleep(1700)
      await Promise.all([red(juri[1]).click(), red(juri[2]).click()])
      await waitText(wasit, '.ref-score-display--red .ref-score-num', /^2$/)
    },
  )

  await step('wasit SELESAIKAN → ketiga layar juri SELESAI dan tombol terkunci', async () => {
    await wasit.locator('button.ref-cmd-btn--finish').click()
    await wasit.locator('.ref-modal-box').waitFor({ state: 'visible' })
    await wasit.locator('button.ref-modal-confirm').first().click()
    await allPages(juri, (p) => waitText(p, '.judge-round-badge', /SELESAI/, 8000))
    await waitText(wasit, '.ref-chrono-round', /SELESAI/)
    await allPages(juri, async (p) =>
      assert(await red(p).isDisabled(), 'tombol harus terkunci setelah selesai'),
    )
  })
} finally {
  await browser.close()
  await kit.control(match, 'cancel').catch(() => undefined)
  await kit.teardown().catch(() => undefined)
}
console.log(failed ? `GAGAL: ${failed} langkah` : 'SEMUA LANGKAH LULUS')
process.exit(failed ? 1 : 0)
