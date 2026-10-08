import { ADMIN, DEWAN, KONTINGEN } from '../support/accounts'

function submitLogin(email: string, password: string) {
  cy.visit('/login')
  if (email) cy.get('#login-email').type(email)
  if (password) cy.get('#login-password').type(password, { log: false })
  cy.contains('button[type="submit"]', 'Masuk ke Akun').click()
}

// Endpoint login dibatasi 10 request/menit per IP, jadi retry dimatikan dan jumlah tes login dijaga hemat.
describe('Login email (backend nyata)', { retries: 0 }, () => {
  beforeEach(() => {
    // spy saja (tanpa stub): request tetap dikirim ke backend asli
    cy.intercept('POST', '**/api/v1/auth/login').as('officialLogin')
    cy.intercept('POST', '**/api/v1/auth/contingent/login').as('contingentLogin')
  })

  it('admin bisa masuk dengan akun seed dan diarahkan ke panel operasional', () => {
    submitLogin(ADMIN.email, ADMIN.password)
    cy.wait('@officialLogin').its('response.statusCode').should('eq', 200)
    cy.location('pathname').should('eq', '/ops')
    cy.window().its('sessionStorage').invoke('getItem', 'ds_token').should('not.be.empty')
    cy.get('h1.ops-topbar__heading').should('contain.text', 'MANAJEMEN KEJUARAAN')
  })

  it('dewan bisa masuk dan melihat menu manajemen', () => {
    submitLogin(DEWAN.email, DEWAN.password)
    cy.wait('@officialLogin').its('response.statusCode').should('eq', 200)
    cy.location('pathname').should('eq', '/ops')
    cy.get('aside button.ops-nav__item').contains('Perangkat & PIN').should('be.visible')
  })

  it('manajer kontingen bisa masuk dan diarahkan ke Data Atlet', () => {
    submitLogin(KONTINGEN.email, KONTINGEN.password)
    cy.wait('@contingentLogin').its('response.statusCode').should('eq', 200)
    cy.location('pathname').should('eq', '/ops')
    cy.location('search').should('contain', 'tab=atlet')
    cy.get('h1.ops-topbar__heading').should('contain.text', 'KONTINGEN & DATA ATLET')
  })

  it('password salah ditolak dan pengguna tetap di halaman login', () => {
    submitLogin(ADMIN.email, 'PasswordSalah123!')
    cy.wait('@contingentLogin').its('response.statusCode').should('be.oneOf', [400, 401, 403, 422])
    cy.get('[role="alert"]').should('be.visible')
    cy.location('pathname').should('eq', '/login')
    cy.window().its('sessionStorage').invoke('getItem', 'ds_token').should('be.null')
  })

  it('form kosong menampilkan pesan validasi tanpa memanggil server', () => {
    submitLogin('', '')
    cy.get('[role="alert"]').should('contain.text', 'Harap isi alamat email dan kata sandi')
    cy.get('@officialLogin.all').should('have.length', 0)
  })

  it('pengguna yang belum login dialihkan dari /ops ke /login', () => {
    cy.visit('/ops')
    cy.location('pathname').should('eq', '/login')
    cy.location('search').should('contain', 'redirect')
  })

  it('logout mengakhiri sesi dan token dihapus', () => {
    submitLogin(ADMIN.email, ADMIN.password)
    cy.location('pathname').should('eq', '/ops')
    cy.get('button.ops-logout-btn').click()
    cy.contains('Konfirmasi Keluar').should('be.visible')
    cy.contains('button', 'Keluar Sekarang').click()
    cy.location('pathname').should('eq', '/login')
    cy.window().its('sessionStorage').invoke('getItem', 'ds_token').should('be.null')
    // setelah logout, /ops tidak bisa dibuka lagi
    cy.visit('/ops')
    cy.location('pathname').should('eq', '/login')
  })
})
