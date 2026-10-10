import { ADMIN, DEWAN } from '../../support/accounts'
import {
  apiCall,
  apiToken,
  birthDateForAge,
  cancelCompetition,
  contingentToken,
  createCategory,
  createCompetition,
  registerAthlete,
  type ApiResponse,
  type Competition,
} from '../../support/api'

/**
 * Matriks hak akses (RBAC) untuk endpoint kategori, registrasi, verifikasi, dan bagan.
 * Sumber kebenaran: internal/api/router.go (adminOrDewan, canRegisterAthlete).
 * Dijalankan terhadap backend nyata lewat cy.request - UI hanya menyembunyikan menu, sedangkan
 * keamanan sebenarnya ada di API.
 */
type Actor = 'anonim' | 'kontingen' | 'dewan' | 'admin'

interface Ctx {
  compId: string
  categoryId: string
  participantId: string
}

interface Endpoint {
  name: string
  /** status yang diharapkan per peran; 'ok' = 2xx, 'deny' = 403, 'auth' = 401 */
  expect: Record<Actor, 'ok' | 'deny' | 'auth'>
  request: (ctx: Ctx, token: string | null) => Cypress.Chainable<ApiResponse<unknown>>
}

const call = (
  method: 'GET' | 'POST' | 'PUT' | 'DELETE',
  path: string,
  token: string | null,
  body?: object,
) => apiCall<unknown>(method, path, token, body)

const endpoints: Endpoint[] = [
  {
    name: 'GET daftar kategori',
    expect: { anonim: 'auth', kontingen: 'ok', dewan: 'ok', admin: 'ok' },
    request: (c, t) => call('GET', `/competitions/${c.compId}/categories`, t),
  },
  {
    name: 'POST buat kategori',
    expect: { anonim: 'auth', kontingen: 'deny', dewan: 'ok', admin: 'ok' },
    request: (c, t) =>
      call('POST', `/competitions/${c.compId}/categories`, t, {
        name: `RBAC ${Date.now()}`,
        gender: 'male',
        weight_class: `R${Math.floor(Math.random() * 1e6)}`,
      }),
  },
  {
    name: 'DELETE reset kategori (kejuaraan kosong)',
    expect: { anonim: 'auth', kontingen: 'deny', dewan: 'ok', admin: 'ok' },
    // kejuaraan terpisah tanpa peserta agar reset tidak 409
    request: (_c, t) =>
      apiToken(ADMIN).then((adminTk) =>
        createCompetition(adminTk, 'RBAC Reset').then((tmp) =>
          call('DELETE', `/competitions/${tmp.id}/categories`, t).then((res) =>
            // jangan campur nilai sinkron dengan command Cypress: bungkus lewat chain
            cancelCompetition(adminTk, tmp.id).then(() => res),
          ),
        ),
      ),
  },
  {
    name: 'POST daftarkan atlet',
    expect: { anonim: 'auth', kontingen: 'ok', dewan: 'ok', admin: 'ok' },
    request: (c, t) =>
      call('POST', `/competitions/${c.compId}/athletes`, t, {
        name: `RBAC Atlet ${Date.now()}`,
        gender: 'male',
        birth_date: birthDateForAge(25),
        category_id: c.categoryId,
      }),
  },
  {
    name: 'PUT verifikasi peserta',
    expect: { anonim: 'auth', kontingen: 'deny', dewan: 'ok', admin: 'ok' },
    request: (c, t) =>
      call('PUT', `/competitions/${c.compId}/participants/${c.participantId}/verify`, t, {
        status: 'rejected',
      }),
  },
  {
    name: 'POST generate bagan',
    expect: { anonim: 'auth', kontingen: 'deny', dewan: 'ok', admin: 'ok' },
    // ok = lolos otorisasi; bisnis boleh 400 (peserta < 2), jadi hanya 401/403 yang dibedakan
    request: (c, t) =>
      call('POST', `/competitions/${c.compId}/brackets/generate`, t, { category_id: c.categoryId }),
  },
]

describe('Matriks hak akses - kategori, registrasi, verifikasi, bagan', () => {
  let adminTk: string
  const tokens: Record<Actor, string | null> = {
    anonim: null,
    kontingen: null,
    dewan: null,
    admin: null,
  }
  let comp: Competition
  const ctx = {} as Ctx

  before(() => {
    apiToken(ADMIN).then((t) => {
      adminTk = t
      tokens.admin = t
      createCompetition(t, 'RBAC').then((c) => {
        comp = c
        ctx.compId = c.id
        createCategory(t, c.id, {
          name: 'RBAC Dewasa',
          gender: 'male',
          weight_class: 'A',
          age_group: 'dewasa',
        }).then((cat) => {
          ctx.categoryId = cat.id
          registerAthlete(t, c.id, {
            name: 'RBAC Peserta',
            gender: 'male',
            birth_date: birthDateForAge(25),
            category_id: cat.id,
          }).then((r) => (ctx.participantId = r.participantId))
        })
      })
    })
    apiToken(DEWAN).then((t) => (tokens.dewan = t))
    contingentToken().then((t) => (tokens.kontingen = t))
  })

  after(() => {
    if (comp) cancelCompetition(adminTk, comp.id)
  })

  const actors: Actor[] = ['anonim', 'kontingen', 'dewan', 'admin']

  endpoints.forEach((ep) => {
    describe(ep.name, () => {
      actors.forEach((actor) => {
        const want = ep.expect[actor]
        it(`${actor} => ${want === 'ok' ? 'diizinkan' : want === 'deny' ? '403 Forbidden' : '401 Unauthorized'}`, () => {
          ep.request(ctx, tokens[actor]).then((res) => {
            if (want === 'auth') expect(res.status).to.eq(401)
            else if (want === 'deny') expect(res.status).to.eq(403)
            else expect(res.status, 'bukan 401/403').to.not.be.oneOf([401, 403])
          })
        })
      })
    })
  })
})
