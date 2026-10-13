import { beforeEach, describe, expect, it } from 'vitest'
import router from './index'

function login(role: string) {
  sessionStorage.setItem('ds_token', 'tkn')
  sessionStorage.setItem('ds_auth_session', JSON.stringify({ token: 'tkn', role }))
}

async function visit(path: string) {
  await router.push(path)
  return router.currentRoute.value
}

describe('router guard', () => {
  beforeEach(async () => {
    sessionStorage.clear()
    await router.push('/login')
  })

  it('mengizinkan rute publik tanpa token', async () => {
    expect((await visit('/scoreboard/a1')).name).toBe('scoreboard')
    expect((await visit('/scoreboard')).name).toBe('scoreboard-list')
  })

  it('halaman bagan publik terbuka tanpa token', async () => {
    expect((await visit('/bagan')).name).toBe('bagan-list')
    expect((await visit('/bagan/c1/k1')).name).toBe('bagan')
  })

  it('base URL diarahkan ke daftar scoreboard live', async () => {
    expect((await visit('/')).name).toBe('scoreboard-list')
  })

  it('/ops tanpa token diarahkan ke /login dengan redirect', async () => {
    const route = await visit('/ops')
    expect(route.path).toBe('/login')
    expect(route.query.redirect).toBe('/ops')
  })

  it.each(['admin', 'dewan', 'operator', 'contingent'])(
    '/ops diizinkan untuk role %s',
    async (role) => {
      login(role)
      expect((await visit('/ops')).path).toBe('/ops')
    },
  )

  it('/ops ditolak untuk role judge dan dialihkan ke /judge', async () => {
    login('judge')
    expect((await visit('/ops')).path).toBe('/judge')
  })

  it('/referee diizinkan untuk wasit tetapi ditolak untuk judge', async () => {
    login('wasit')
    expect((await visit('/referee')).path).toBe('/referee')
    await visit('/login')
    login('judge')
    expect((await visit('/referee')).path).toBe('/judge')
  })

  it('/admin/pins diarahkan ke /ops?tab=pin', async () => {
    login('admin')
    expect((await visit('/admin/pins')).path).toBe('/ops')
  })

  it('sesi rusak diperlakukan tanpa role', async () => {
    sessionStorage.setItem('ds_token', 'tkn')
    sessionStorage.setItem('ds_auth_session', '{rusak')
    expect((await visit('/ops')).path).toBe('/login')
  })
})
