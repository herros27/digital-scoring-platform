import { ADMIN, type EmailAccount } from './accounts'

/** Sesi aplikasi (bentuk AuthSession pada src/stores/auth.ts) untuk perangkat berbasis PIN. */
export interface PinSession {
  token: string
  role: string
  seat?: number
  arenaId: string
  competitionId: string
}

export interface PinItem {
  role: string
  seat_number: number
  pin: string
}
export interface ArenaPins {
  competitionId: string
  arenaId: string
  pins: PinItem[]
  find(role: 'judge' | 'wasit', seat: number): string
}

declare global {
  // eslint-disable-next-line @typescript-eslint/no-namespace
  namespace Cypress {
    interface Chainable {
      /** Login lewat form email/password seperti pengguna sungguhan (sesi di-cache per akun). */
      loginAs(account: EmailAccount): Chainable<void>
      /** Login tablet lewat keypad PIN seperti juri/wasit sungguhan. */
      loginWithPin(pin: string): Chainable<void>
      /**
       * Membuka halaman sebagai perangkat yang SUDAH login PIN (token dari API), dengan menyuntik sesi ke
       * sessionStorage sebelum aplikasi dimuat. Dipakai tes pertandingan agar tidak menghabiskan batas login
       * 10/menit; alur form PIN-nya sendiri diuji di judge_pin_login.cy.ts.
       */
      visitWithSession(path: string, session: PinSession): Chainable<void>
      /**
       * Ambil kompetisi, gelanggang, dan PIN aktif langsung dari backend nyata (setup data).
       * ID tidak ditulis di kode karena berbeda di tiap database hasil seed.
       * Dipanggil SEKALI per spec (before) karena endpoint login dibatasi 10 request/menit.
       */
      fetchArenaPins(): Chainable<ArenaPins>
    }
  }
}

Cypress.Commands.add('loginAs', (account) => {
  cy.session(
    ['email', account.email],
    () => {
      cy.visit('/login')
      cy.get('#login-email').type(account.email)
      cy.get('#login-password').type(account.password, { log: false })
      cy.contains('button[type="submit"]', 'Masuk ke Akun').click()
      cy.location('pathname', { timeout: 15000 }).should('match', /^\/(ops|referee)$/)
    },
    {
      validate() {
        cy.window().then((win) => {
          const token = win.sessionStorage.getItem('ds_token') ?? ''
          expect(token.length, 'token sesi tersimpan').to.be.greaterThan(0)
        })
      },
      cacheAcrossSpecs: true,
    },
  )
})

Cypress.Commands.add('loginWithPin', (pin) => {
  cy.visit('/judge/login')
  for (const digit of pin.split('')) {
    cy.contains('button.keypad-btn', new RegExp(`^\\s*${digit}\\s*$`)).click()
  }
  cy.contains('button', /^\s*Masuk\s*$/).click()
})

Cypress.Commands.add('fetchArenaPins', () => {
  return cy
    .request('POST', '/api/v1/auth/login', { email: ADMIN.email, password: ADMIN.password })
    .then((login) => {
      const headers = { Authorization: `Bearer ${login.body.data.token as string}` }
      return cy.request({ url: '/api/v1/competitions', headers }).then((comps) => {
        const list = comps.body.data as Array<{ id: string; name: string }>
        expect(list.length, 'ada kompetisi hasil seed').to.be.greaterThan(0)
        const comp = list.find((c) => /menpora/i.test(c.name)) ?? list[0]
        return cy
          .request({ url: `/api/v1/competitions/${comp.id}/arenas`, headers })
          .then((res) => {
            const arenas = res.body.data as Array<{ id: string }>
            expect(arenas.length, 'ada gelanggang hasil seed').to.be.greaterThan(0)
            const arenaId = arenas[0].id
            return cy
              .request({
                url: `/api/v1/auth/pins?competition_id=${comp.id}&arena_id=${arenaId}`,
                headers,
              })
              .then((p) => {
                const pins = p.body.data as PinItem[]
                const result: ArenaPins = {
                  competitionId: comp.id,
                  arenaId,
                  pins,
                  find(role, seat) {
                    const found = pins.find((x) => x.role === role && x.seat_number === seat)
                    expect(found, `PIN aktif ${role} kursi ${seat}`).to.not.equal(undefined)
                    return found!.pin
                  },
                }
                return result
              })
          })
      })
    })
})

Cypress.Commands.add('visitWithSession', (path, session) => {
  cy.visit(path, {
    onBeforeLoad(win) {
      win.sessionStorage.setItem(
        'ds_auth_session',
        JSON.stringify({
          token: session.token,
          role: session.role,
          seatNumber: session.seat,
          arenaId: session.arenaId,
          competitionId: session.competitionId,
        }),
      )
      win.sessionStorage.setItem('ds_token', session.token)
    },
  })
})
