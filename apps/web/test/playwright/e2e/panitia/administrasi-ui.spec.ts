import { expect, test, type Browser, type Page } from '@playwright/test'
import { adminToken } from '../../../cypress/support/match/auth.ts'
import {
  enrollAndVerify,
  generateBracket,
  playMatch,
} from '../../../cypress/support/match/competition.ts'
import { api, must, type Json } from '../../../cypress/support/match/http.ts'
import { buildKit } from '../../../cypress/support/match/kit.ts'
import type { MatchKit } from '../../../cypress/support/match/kit.ts'
import { viewport, openAs } from '../../support/fixtures.ts'

/**
 * Administrasi di ruang operasi (browser nyata): ubah dan hapus satu kategori, batal daftar atlet (admin dan manajer
 * kontingen), reset password kontingen, tambah partai manual, dan rincian skor per partai. Data disiapkan lewat API;
 * semua aksi lewat tombol di layar; hasil akhir dicek ke server.
 */
test.describe.configure({ mode: 'serial' })

const stamp = Date.now().toString(36)
let kit: MatchKit
let admin = ''

test.beforeAll(async () => {
  test.setTimeout(120_000)
  admin = await adminToken()
  kit = await buildKit(admin, {
    label: 'PW Administrasi',
    roundSeconds: 90,
    breakSeconds: 3,
    totalRounds: 2,
  })
})
test.afterAll(async () => {
  await kit?.teardown().catch(() => undefined)
})

async function openOps(
  browser: Browser,
  baseURL: string | undefined,
  tab: string,
  who: { token: string; role: string } = { token: admin, role: 'admin' },
): Promise<Page> {
  const ctx = await browser.newContext({ baseURL, viewport: viewport() })
  const page = await openAs(
    ctx,
    { token: who.token, role: who.role, arenaId: '', competitionId: '' },
    `/ops?tab=${tab}`,
    undefined,
  )
  await page.locator('.comp-dropdown').selectOption(kit.competitionId)
  return page
}

const createCategory = (name: string, extra: Json = {}): Promise<Json> =>
  must(
    `kategori ${name}`,
    api('POST', `/competitions/${kit.competitionId}/categories`, admin, {
      name,
      gender: 'male',
      weight_class: 'B',
      age_group: 'dewasa',
      min_weight: 50,
      max_weight: 70,
      ...extra,
    }),
  )

const listCategories = (): Promise<Json[]> =>
  must('daftar kategori', api('GET', `/competitions/${kit.competitionId}/categories`, admin))
const participants = (token = admin): Promise<Json[]> =>
  must('peserta', api('GET', `/competitions/${kit.competitionId}/participants`, token))
const catCard = (page: Page, name: string) => page.locator('article.cat-card', { hasText: name })
const atlCard = (page: Page, name: string) => page.locator('article.atl-card', { hasText: name })

test.describe('Kategori: ubah dan hapus satu kategori', () => {
  const name = `Uji Kategori ${stamp}`
  let cat: Json

  test('ubah nama dan batas bobot kategori kosong lewat form', async ({ browser, baseURL }) => {
    cat = await createCategory(name)
    const page = await openOps(browser, baseURL, 'kategori')
    await catCard(page, name).locator('[data-test="category-edit"]').click()
    const form = page.locator('[data-test="category-form"]')
    await expect(form).toBeVisible()
    await expect(page.getByRole('heading', { name: 'Ubah Kategori Tanding' })).toBeVisible()
    await page.locator('#cat-name-input').fill(`${name} Baru`)
    await page.locator('#cat-weight-class-input').fill('C')
    await page.locator('#cat-max-weight').fill('75')
    await page.locator('[data-test="category-save"]').click()

    await expect(catCard(page, `${name} Baru`)).toBeVisible()
    const now = (await listCategories()).find((c) => c.id === cat.id)!
    expect(now).toMatchObject({ name: `${name} Baru`, weight_class: 'C', max_weight: 75 })
  })

  test('setelah ada peserta hanya nama yang boleh berubah; server menolak batas bobot dengan pesan jelas', async ({
    browser,
    baseURL,
  }) => {
    await enrollAndVerify(kit, [{ name: `Atlet Kat ${stamp}`, team: 'Tim Uji' }], {
      categoryId: cat.id,
    })
    const page = await openOps(browser, baseURL, 'kategori')
    await catCard(page, `${name} Baru`).locator('[data-test="category-edit"]').click()

    await page.locator('#cat-max-weight').fill('80')
    await page.locator('[data-test="category-save"]').click()
    await expect(page.locator('[data-test="category-form-error"]')).toContainText('hanya nama')
    expect((await listCategories()).find((c) => c.id === cat.id)!.max_weight).toBe(75)

    await page.locator('#cat-max-weight').fill('75') // kembalikan, ganti nama saja
    await page.locator('#cat-name-input').fill(`${name} Final`)
    await page.locator('[data-test="category-save"]').click()
    await expect(catCard(page, `${name} Final`)).toBeVisible()
  })

  test('hapus kategori berpeserta ditolak; setelah peserta dibatalkan, kategori terhapus', async ({
    browser,
    baseURL,
  }) => {
    const page = await openOps(browser, baseURL, 'kategori')
    await catCard(page, `${name} Final`).locator('[data-test="category-delete"]').click()
    await page.locator('[data-test="category-delete-confirm"]').click()
    await expect(page.locator('[data-test="category-delete-error"]')).toContainText('peserta')
    expect((await listCategories()).some((c) => c.id === cat.id)).toBe(true)

    // batalkan pendaftaran lewat API (alur layar diuji di bagian berikut), lalu hapus lewat layar
    const mine = (await participants()).find((p) => p.category_id === cat.id)!
    await must(
      'batal daftar',
      api('DELETE', `/competitions/${kit.competitionId}/participants/${mine.id}`, admin),
    )
    await page.locator('[data-test="category-delete-confirm"]').click()
    await expect(catCard(page, `${name} Final`)).toHaveCount(0)
    expect((await listCategories()).some((c) => c.id === cat.id)).toBe(false)
  })
})

test.describe('Batal daftar atlet', () => {
  test('admin membatalkan pendaftaran atlet terverifikasi lewat dialog; atlet hilang dari daftar', async ({
    browser,
    baseURL,
  }) => {
    const [e] = await enrollAndVerify(kit, [{ name: `Atlet Batal ${stamp}`, team: 'Tim Batal' }])
    const page = await openOps(browser, baseURL, 'atlet')
    const card = atlCard(page, e.name)
    await expect(card.locator('[data-test="athlete-status"]')).toContainText('TERVERIFIKASI')

    await card.locator('[data-test="athlete-unregister"]').click()
    await expect(page.locator('[data-test="unregister-dialog"]')).toContainText(e.name)
    await page.getByRole('button', { name: 'Kembali' }).click()
    await expect(card).toBeVisible() // kembali = tidak terhapus

    await card.locator('[data-test="athlete-unregister"]').click()
    await page.locator('[data-test="unregister-confirm"]').click()
    await expect(atlCard(page, e.name)).toHaveCount(0)
    expect((await participants()).some((p) => p.athlete_id === e.athleteId)).toBe(false)
  })

  test('peserta yang sudah masuk bagan tidak dapat dibatalkan; pesan menjelaskan langkahnya', async ({
    browser,
    baseURL,
  }) => {
    const cat = await createCategory(`Uji Bagan ${stamp}`)
    const [a] = await enrollAndVerify(
      kit,
      [
        { name: `Bagan A ${stamp}`, team: 'Tim 1' },
        { name: `Bagan B ${stamp}`, team: 'Tim 2' },
      ],
      { categoryId: cat.id },
    )
    expect((await generateBracket(kit, cat.id)).status).toBe(201)

    const page = await openOps(browser, baseURL, 'atlet')
    await atlCard(page, a.name).locator('[data-test="athlete-unregister"]').click()
    await page.locator('[data-test="unregister-confirm"]').click()
    await expect(page.locator('[data-test="unregister-error"]')).toContainText('bagan')
    expect((await participants()).some((p) => p.athlete_id === a.athleteId)).toBe(true)
  })

  test('manajer kontingen: bisa membatalkan atlet miliknya yang belum diverifikasi; yang sudah diverifikasi terkunci', async ({
    browser,
    baseURL,
  }) => {
    const mail = `kont-${stamp}@lomba.test`
    await must(
      'daftar kontingen',
      api('POST', '/auth/contingent/register', null, {
        name: `Kontingen Uji ${stamp}`,
        pic_name: 'Pak PIC',
        email: mail,
        phone: '081234567',
        password: 'password-awal-1',
      }),
    )
    const login = await must(
      'login kontingen',
      api('POST', '/auth/contingent/login', null, { email: mail, password: 'password-awal-1' }),
    )
    const ctoken: string = login.token
    const reg = (name: string) =>
      must(
        `daftar ${name}`,
        api('POST', `/competitions/${kit.competitionId}/athletes`, ctoken, {
          name,
          gender: 'male',
          team: 'Kontingen Uji',
          birth_date: '1998-01-01',
          category_id: kit.categoryId,
        }),
      )
    const pending = await reg(`Kont Menunggu ${stamp}`)
    const verified = await reg(`Kont Terverifikasi ${stamp}`)
    const vp = (await participants(ctoken)).find((p) => p.athlete_id === verified.athlete_id)!
    await must(
      'verifikasi',
      api('PUT', `/competitions/${kit.competitionId}/participants/${vp.id}/verify`, admin, {
        status: 'verified',
        actual_weight: 60,
        docs_verified: { identity: true, health: true },
      }),
    )

    const page = await openOps(browser, baseURL, 'atlet', { token: ctoken, role: 'contingent' })
    const waiting = atlCard(page, `Kont Menunggu ${stamp}`)
    await expect(waiting.locator('[data-test="athlete-status"]')).toContainText(
      'MENUNGGU VERIFIKASI',
    )
    await expect(
      atlCard(page, `Kont Terverifikasi ${stamp}`).locator('[data-test="athlete-unregister"]'),
    ).toHaveCount(0)

    await waiting.locator('[data-test="athlete-unregister"]').click()
    await page.locator('[data-test="unregister-confirm"]').click()
    await expect(atlCard(page, `Kont Menunggu ${stamp}`)).toHaveCount(0)
    expect((await participants()).some((p) => p.athlete_id === pending.athlete_id)).toBe(false)
    expect((await participants()).some((p) => p.id === vp.id)).toBe(true)
  })
})

test.describe('Akun kontingen', () => {
  test('admin mereset password manajer kontingen lewat tab Manajer Kontingen', async ({
    browser,
    baseURL,
  }) => {
    const mail = `kont-reset-${stamp}@lomba.test`
    await must(
      'daftar kontingen',
      api('POST', '/auth/contingent/register', null, {
        name: `Kontingen Reset ${stamp}`,
        pic_name: 'Bu PIC',
        email: mail,
        phone: '081234567',
        password: 'password-awal-1',
      }),
    )
    const page = await openOps(browser, baseURL, 'petugas')
    await page.locator('[data-test="tab-kontingen"]').click()
    const row = page.locator(`[data-test="contingent-row"][data-email="${mail}"]`)
    await expect(row).toBeVisible()

    await row.locator('[data-test="contingent-reset"]').click()
    await page.locator('[data-test="contingent-reset-password"]').fill('pendek')
    await page.locator('[data-test="contingent-reset-submit"]').click()
    await expect(page.locator('[data-test="contingent-reset-error"]')).toContainText('minimal 8')

    await page.locator('[data-test="contingent-reset-password"]').fill('password-baru-9')
    await page.locator('[data-test="contingent-reset-submit"]').click()
    await expect(page.locator('[data-test="contingent-reset-dialog"]')).toBeHidden()

    const login = (password: string) =>
      api('POST', '/auth/contingent/login', null, { email: mail, password })
    expect((await login('password-awal-1')).status).toBe(401)
    expect((await login('password-baru-9')).status).toBe(200)
  })
})

test.describe('Jadwal: tanpa tambah partai manual, dan rincian skor', () => {
  const names = [`Manual Merah ${stamp}`, `Manual Biru ${stamp}`]
  let ids: string[] = []

  test('jadwal tidak punya tombol tambah partai; partai dari API tetap tampil', async ({
    browser,
    baseURL,
  }) => {
    const entrants = await enrollAndVerify(
      kit,
      names.map((n) => ({ name: n, team: 'Tim Manual' })),
    )
    ids = (await participants()).filter((p) => names.includes(p.name)).map((p) => p.id)
    // satu peserta belum diverifikasi: tidak boleh muncul sebagai pilihan
    await must(
      'daftar pending',
      api('POST', `/competitions/${kit.competitionId}/athletes`, admin, {
        name: `Manual Pending ${stamp}`,
        gender: 'male',
        team: 'Tim X',
        birth_date: '1998-01-01',
        category_id: kit.categoryId,
      }),
    )
    expect(entrants).toHaveLength(2)

    // Jadwal hanya menampilkan partai dari bagan: tidak ada tombol tambah partai manual di UI.
    await must(
      'buat partai',
      api('POST', `/competitions/${kit.competitionId}/matches`, admin, {
        arena_id: kit.arenaId,
        athlete_red_id: ids[0],
        athlete_blue_id: ids[1],
      }),
    )
    const page = await openOps(browser, baseURL, 'jadwal')
    await expect(page.locator('[data-test="match-add"]')).toHaveCount(0)

    const card = page.locator('article.sch-card', { hasText: names[0] })
    await expect(card).toBeVisible()
    await expect(card).toContainText(names[1])
    const matches = await must(
      'partai',
      api('GET', `/competitions/${kit.competitionId}/matches`, admin),
    )
    const created = matches.find(
      (m: Json) => m.athlete_red_id === ids[0] && m.athlete_blue_id === ids[1],
    )
    expect(created).toBeTruthy()
    expect(created.arena_id).toBe(kit.arenaId)
  })

  test('rincian skor: partai belum dimainkan kosong; setelah dimainkan menampilkan skor per babak, total, dan riwayat', async ({
    browser,
    baseURL,
  }) => {
    test.setTimeout(120_000)
    const page = await openOps(browser, baseURL, 'jadwal')
    const card = page.locator('article.sch-card', { hasText: names[0] })

    await card.locator('[data-test="match-scores"]').click()
    const dialog = page.locator('[data-test="score-dialog"]')
    await expect(dialog).toBeVisible()
    await expect(dialog.locator('[data-test="no-rounds"]')).toBeVisible()
    await dialog.getByRole('button', { name: 'Tutup' }).click()

    const matches = await must(
      'partai',
      api('GET', `/competitions/${kit.competitionId}/matches`, admin),
    )
    const m = matches.find((x: Json) => x.athlete_red_id === ids[0])
    await playMatch(kit, m.id, [
      { target: 'red', type: 'knockdown' },
      { target: 'red', type: 'knockdown' },
    ])

    await page.reload()
    await page.locator('.comp-dropdown').selectOption(kit.competitionId)
    const finished = page.locator('article.sch-card', { hasText: names[0] })
    await finished.locator('[data-test="match-scores"]').click()
    await expect(dialog).toBeVisible()
    await expect(dialog.locator('[data-test="score-round-row"]').first()).toBeVisible()
    await expect(dialog.locator('[data-test="total-red"]')).not.toHaveText('0')
    await expect(dialog.locator('[data-test="total-blue"]')).toHaveText('0')
    await expect(dialog.locator('[data-test="score-events"]')).toContainText('Pertandingan selesai')
    await expect(dialog.getByText('MENANG', { exact: true })).toBeVisible()
  })
})
