import { expect, test, type Page } from '@playwright/test'
import { api, must } from '../../../cypress/support/match/http.ts'
import { ADMIN, DEWAN } from '../../../cypress/support/accounts.ts'
import { adminToken, dewanToken } from '../../../cypress/support/match/auth.ts'
import { loginOfficial } from '../../../cypress/support/match/kit.ts'

/**
 * Layar "Akun Petugas" di ruang operasi: tambah, ubah peran, reset password, hapus, dan ganti password sendiri,
 * termasuk batas wewenang Dewan Juri (tidak boleh mengelola admin/dewan). Login lewat formulir sungguhan; hasil
 * akhir selalu dicek ke server (bisa/tidaknya akun masuk dengan password tertentu).
 */
test.describe.configure({ mode: 'serial' })

const stamp = Date.now()
const email = (n: string) => `pw-${n}-${stamp}@silat.test`
const created: string[] = [] // id akun yang dibuat tes, dibersihkan di akhir
let admin = ''

test.beforeAll(async () => {
  admin = await adminToken()
})
test.afterAll(async () => {
  for (const id of created) await api('DELETE', `/officials/${id}`, admin).catch(() => undefined)
})

async function loginViaForm(page: Page, who: { email: string; password: string }) {
  await page.goto('/login', { waitUntil: 'domcontentloaded' })
  await page.locator('#login-email input').fill(who.email)
  await page.locator('#login-password input').fill(who.password)
  await page.locator('button[type="submit"]').click()
  await expect(page).toHaveURL(/\/ops/)
}

/** Membuka ruang operasi dengan sesi dari token (login dibatasi 10/menit, jadi tidak semua tes lewat formulir). */
async function openAsOfficial(
  page: Page,
  who: { token: string; role: string; email: string; name: string },
) {
  const list = await must<{ id: string; email: string }[]>(
    'daftar petugas',
    api('GET', '/officials', who.token),
  )
  const userId = list.find((o) => o.email === who.email)?.id ?? ''
  await page.addInitScript(
    (sess) => {
      sessionStorage.setItem('ds_auth_session', JSON.stringify(sess))
      sessionStorage.setItem('ds_token', sess.token)
    },
    { ...who, userId },
  )
  await page.goto('/ops?tab=kompetisi', { waitUntil: 'domcontentloaded' })
  await openOfficials(page)
}

async function openOfficials(page: Page) {
  await page.locator('[data-test="nav-petugas"]').click()
  await expect(page.locator('[data-test="officials-view"]')).toBeVisible()
  await expect(page.locator('[data-test="official-row"]').first()).toBeVisible()
}

const row = (page: Page, mail: string) =>
  page.locator(`[data-test="official-row"][data-email="${mail}"]`)

const canLogin = async (mail: string, password: string) =>
  (await api('POST', '/auth/login', null, { email: mail, password })).status === 200

async function idOf(mail: string): Promise<string> {
  const list = await must<{ id: string; email: string }[]>(
    'daftar petugas',
    api('GET', '/officials', admin),
  )
  return list.find((o) => o.email === mail)!.id
}

test('admin melihat akunnya sendiri bertanda ANDA dan tidak bisa menghapusnya', async ({
  page,
}) => {
  await loginViaForm(page, ADMIN)
  await openOfficials(page)
  const me = row(page, ADMIN.email)
  await expect(me).toContainText('ANDA')
  await expect(me.locator('[data-test="official-delete"]')).toBeDisabled()
})

test('admin menambah operator lewat form; akun itu bisa masuk dengan password awalnya', async ({
  page,
}) => {
  const mail = email('op')
  await openAsOfficial(page, {
    token: admin,
    role: 'admin',
    email: ADMIN.email,
    name: ADMIN.name ?? 'Admin',
  })

  await page.locator('[data-test="official-add"]').click()
  await page.locator('[data-test="official-name"]').fill('Operator Uji')
  await page.locator('[data-test="official-email"]').fill(mail)
  await page.locator('[data-test="official-password"]').fill('mock_pass_init')
  await page.locator('[data-test="official-role"]').selectOption('operator')
  await page.locator('[data-test="official-save"]').click()

  await expect(row(page, mail)).toBeVisible()
  await expect(row(page, mail)).toContainText('Staf Administrasi')
  created.push(await idOf(mail))
  expect(await canLogin(mail, 'mock_pass_init')).toBe(true)
})

test('form menolak password pendek dan email ganda dengan pesan yang jelas', async ({ page }) => {
  const mail = email('op')
  await openAsOfficial(page, {
    token: admin,
    role: 'admin',
    email: ADMIN.email,
    name: ADMIN.name ?? 'Admin',
  })

  await page.locator('[data-test="official-add"]').click()
  await page.locator('[data-test="official-name"]').fill('Calon Pendek')
  await page.locator('[data-test="official-email"]').fill(email('pendek'))
  await page.locator('[data-test="official-password"]').fill('abc')
  await page.locator('[data-test="official-save"]').click()
  await expect(page.locator('[data-test="official-form-error"]')).toContainText('minimal 8')
  expect(await canLogin(email('pendek'), 'abc')).toBe(false)

  await page.locator('[data-test="official-email"]').fill(mail) // email yang sudah dipakai
  await page.locator('[data-test="official-password"]').fill('mock_pass_dup')
  await page.locator('[data-test="official-save"]').click()
  await expect(page.locator('[data-test="official-form-error"]')).toBeVisible()
  const list = await must<{ email: string }[]>('daftar', api('GET', '/officials', admin))
  expect(list.filter((o) => o.email === mail)).toHaveLength(1)
})

test('admin mengubah peran operator menjadi wasit', async ({ page }) => {
  const mail = email('op')
  await openAsOfficial(page, {
    token: admin,
    role: 'admin',
    email: ADMIN.email,
    name: ADMIN.name ?? 'Admin',
  })

  await row(page, mail).locator('[data-test="official-edit"]').click()
  await page.locator('[data-test="official-role"]').selectOption('wasit')
  await page.locator('[data-test="official-save"]').click()
  await expect(row(page, mail)).toContainText('Operator Meja')
  const list = await must<{ email: string; role: string }[]>(
    'daftar',
    api('GET', '/officials', admin),
  )
  expect(list.find((o) => o.email === mail)?.role).toBe('wasit')
})

test('reset password: password lama tidak berlaku lagi, password baru berlaku', async ({
  page,
}) => {
  const mail = email('op')
  await openAsOfficial(page, {
    token: admin,
    role: 'admin',
    email: ADMIN.email,
    name: ADMIN.name ?? 'Admin',
  })

  await row(page, mail).locator('[data-test="official-reset"]').click()
  await page.locator('[data-test="official-reset-password"]').fill('pendek')
  await page.locator('[data-test="official-reset-submit"]').click()
  await expect(page.locator('[data-test="official-reset-error"]')).toContainText('minimal 8')

  await page.locator('[data-test="official-reset-password"]').fill('mock_pass_reset')
  await page.locator('[data-test="official-reset-submit"]').click()
  await expect(page.locator('[data-test="official-reset-dialog"]')).toBeHidden()
  expect(await canLogin(mail, 'mock_pass_init')).toBe(false)
  expect(await canLogin(mail, 'mock_pass_reset')).toBe(true)
})

test('hapus meminta konfirmasi; setelah dikonfirmasi akun tidak bisa masuk lagi', async ({
  page,
}) => {
  const mail = email('op')
  await openAsOfficial(page, {
    token: admin,
    role: 'admin',
    email: ADMIN.email,
    name: ADMIN.name ?? 'Admin',
  })

  await row(page, mail).locator('[data-test="official-delete"]').click()
  await page.getByRole('button', { name: 'Batal' }).click()
  await expect(row(page, mail)).toBeVisible() // batal = tidak terhapus

  await row(page, mail).locator('[data-test="official-delete"]').click()
  await page.locator('[data-test="official-delete-confirm"]').click()
  await expect(row(page, mail)).toHaveCount(0)
  expect(await canLogin(mail, 'mock_pass_reset')).toBe(false)
})

test('Pengawas Pertandingan: hanya Operator Meja/Staf Administrasi yang bisa dikelola; akun Admin dan Dewan terkunci', async ({
  page,
}) => {
  await openAsOfficial(page, {
    token: await dewanToken(),
    role: 'dewan',
    email: DEWAN.email,
    name: DEWAN.name ?? 'Dewan',
  })

  await expect(row(page, ADMIN.email).locator('[data-test="official-edit"]')).toHaveCount(0)
  await expect(row(page, ADMIN.email)).toContainText('Hanya Admin')
  await expect(row(page, DEWAN.email).locator('[data-test="official-edit"]')).toHaveCount(0)

  await page.locator('[data-test="official-add"]').click()
  const options = await page.locator('[data-test="official-role"] option').allTextContents()
  expect(options).toEqual(['Operator Meja', 'Staf Administrasi'])
})

test('ganti password sendiri: password lama salah tidak mengeluarkan sesi; yang benar berhasil', async ({
  page,
}) => {
  // akun dewan sementara agar password akun seed tidak berubah
  const mail = email('dewan')
  const made = await must<{ id: string }>(
    'buat dewan',
    api('POST', '/officials', admin, {
      name: 'Dewan Uji',
      email: mail,
      password: 'mock_pass_alpha',
      role: 'dewan',
    }),
  )
  created.push(made.id)

  await openAsOfficial(page, {
    token: await loginOfficial(mail, 'mock_pass_alpha'),
    role: 'dewan',
    email: mail,
    name: 'Dewan Uji',
  })
  await page.locator('[data-test="ops-change-password"]').click()
  const dialog = page.locator('[data-test="own-password-dialog"]')
  await expect(dialog).toBeVisible()

  await page.locator('[data-test="own-password-current"]').fill('mock_pass_wrong')
  await page.locator('[data-test="own-password-new"]').fill('mock_pass_beta')
  await page.locator('[data-test="own-password-confirm"]').fill('mock_pass_beta')
  await page.locator('[data-test="own-password-submit"]').click()
  await expect(page.locator('[data-test="own-password-error"]')).toContainText(
    'password lama salah',
  )
  // sesi tetap hidup: dialog masih terbuka dan halaman tidak dialihkan ke login
  await expect(dialog).toBeVisible()
  await expect(page).toHaveURL(/\/ops/)

  await page.locator('[data-test="own-password-current"]').fill('mock_pass_alpha')
  await page.locator('[data-test="own-password-submit"]').click()
  await expect(dialog).toBeHidden()
  expect(await canLogin(mail, 'mock_pass_alpha')).toBe(false)
  expect(await canLogin(mail, 'mock_pass_beta')).toBe(true)
})
