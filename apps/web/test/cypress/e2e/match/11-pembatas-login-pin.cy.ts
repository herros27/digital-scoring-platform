import { adminToken } from '../../support/match/auth.ts'
import { appOrigin, must } from '../../support/match/http.ts'
import { buildKit } from '../../support/match/kit.ts'
import type { MatchKit } from '../../support/match/kit.ts'
import { runAsync } from '../../support/match/run.ts'
import { api } from '../../support/match/http.ts'

/**
 * Pelindung tebak-PIN berlapis (perangkat 5x, IP 25x, global 200x per 15 menit; hanya PIN SALAH yang dihitung).
 * Tiap tes memakai alamat IP uji sendiri (header X-Forwarded-For, rentang dokumentasi 203.0.113.0/24) agar kunci
 * yang sengaja dipicu di sini tidak menjatuhkan IP runner dan tes lain.
 */
const WRONG = '000001'

async function pinLogin(pin: string, opts: { ip: string; device?: string }) {
  const res = await fetch(`${appOrigin()}/api/v1/auth/judge-login`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      'X-Forwarded-For': opts.ip,
      ...(opts.device ? { 'X-Device-Id': opts.device } : {}),
    },
    body: JSON.stringify({ pin }),
  })
  const body = await res.json().catch(() => null)
  return {
    status: res.status,
    retryAfter: res.headers.get('Retry-After'),
    code: body?.error?.code as string | undefined,
    scope: body?.error?.details?.scope as string | undefined,
    message: body?.error?.message as string | undefined,
  }
}

describe('Pelindung tebak-PIN: satu tablet tidak mengunci tablet lain', () => {
  let kit: MatchKit
  let goodPin = ''
  const uid = Date.now().toString(36)
  const device = (n: string) => `e2e-${uid}-${n}` // 8-64 karakter alfanumerik/strip

  before(() =>
    runAsync(async () => {
      kit = await buildKit(await adminToken(), { label: 'Pembatas PIN' })
      const created = await must(
        'buat PIN juri kursi 4',
        api('POST', '/auth/create-pin', kit.adminToken, {
          competition_id: kit.competitionId,
          arena_id: kit.arenaId,
          role: 'juri',
          seat_number: 4,
          ttl_hours: 2,
        }),
      )
      goodPin = created.pin
    }),
  )
  after(() => runAsync(async () => kit?.teardown()))

  it('tablet A salah PIN 5x terkunci (walau lalu memakai PIN benar), tablet B di IP yang sama tetap bisa login', () =>
    runAsync(async () => {
      const ip = '203.0.113.11'
      for (let i = 1; i <= 5; i += 1) {
        expect((await pinLogin(WRONG, { ip, device: device('a') })).status, `salah ke-${i}`).to.eq(
          401,
        )
      }
      const locked = await pinLogin(goodPin, { ip, device: device('a') })
      expect(locked).to.include({ status: 429, code: 'TOO_MANY_ATTEMPTS', scope: 'device' })
      expect(Number(locked.retryAfter)).to.be.greaterThan(0)
      expect(locked.message).to.match(/perangkat ini/)

      const b = await pinLogin(goodPin, { ip, device: device('b') })
      expect(b.status, 'tablet B tidak boleh ikut terkunci').to.eq(200)
    }))

  it('mengganti-ganti ID perangkat tidak menembus: 25 PIN salah dari satu IP mengunci IP itu, IP lain tetap normal', () =>
    runAsync(async () => {
      const ip = '203.0.113.12'
      for (let i = 0; i < 25; i += 1) {
        const r = await pinLogin(WRONG, { ip, device: device(`rot-${i}`) })
        expect(r.status, `tebakan ${i + 1}`).to.eq(401)
      }
      const fresh = await pinLogin(goodPin, { ip, device: device('baru-sekali') })
      expect(fresh).to.include({ status: 429, scope: 'ip' })
      expect(fresh.message).to.match(/jaringan ini/)

      const otherIp = await pinLogin(goodPin, { ip: '203.0.113.99', device: device('ip-lain') })
      expect(otherIp.status, 'IP lain tidak terdampak').to.eq(200)
    }))

  it('login berhasil di tengah tebakan tidak menghapus hitungan IP (penyerang dengan satu PIN sah tidak bisa menebak tanpa batas)', () =>
    runAsync(async () => {
      const ip = '203.0.113.13'
      for (let i = 0; i < 20; i += 1) {
        expect((await pinLogin(WRONG, { ip, device: device(`m-${i}`) })).status).to.eq(401)
      }
      expect((await pinLogin(goodPin, { ip, device: device('sah') })).status).to.eq(200)
      for (let i = 0; i < 5; i += 1) {
        expect((await pinLogin(WRONG, { ip, device: device(`n-${i}`) })).status).to.eq(401)
      }
      // total 25 salah meski ada satu login berhasil di tengah: IP terkunci
      expect(await pinLogin(goodPin, { ip, device: device('setelah') })).to.include({
        status: 429,
        scope: 'ip',
      })
    }))

  it('ID perangkat yang tidak valid diabaikan: klien tanpa ID dikunci per IP seperti perangkat tunggal', () =>
    runAsync(async () => {
      const ip = '203.0.113.14'
      for (let i = 0; i < 5; i += 1) await pinLogin(WRONG, { ip, device: 'x' }) // terlalu pendek -> dianggap tanpa ID
      expect(await pinLogin(goodPin, { ip })).to.include({ status: 429, scope: 'device' })
    }))
})
