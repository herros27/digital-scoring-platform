import { ADMIN } from '../../support/accounts'
import {
  adminToken,
  cancelCompetition,
  createCompetition,
  seedBracket,
  type Competition,
  type SeededBracket,
} from '../../support/api'

/**
 * Menu Bagan Pertandingan (admin): dropdown "Kategori & Kelas" harus selalu terisi dengan kategori yang
 * benar-benar ada, sehingga bagan yang tampil adalah bagan kategori itu (bukan jalur lama per kelas bobot
 * yang dulu muncul saat dropdown kosong).
 */
describe('Admin - Bagan Pertandingan per kategori', () => {
  let admin: string
  let comp: Competition
  let seeded: SeededBracket

  before(() => {
    adminToken().then((t) => {
      admin = t
      createCompetition(t, 'Bagan Ops').then((c) => {
        comp = c
        seedBracket(t, c.id, { name: 'Tanding Dewasa Putra - Kelas A', prefix: 'Ops' }).then(
          (s) => (seeded = s),
        )
      })
    })
  })

  after(() => {
    if (comp) cancelCompetition(admin, comp.id)
  })

  beforeEach(() => {
    cy.loginAs(ADMIN)
    cy.visit('/ops?tab=kompetisi')
    cy.get('select.comp-dropdown').select(comp.id)
    cy.get('aside button.ops-nav__item').contains('Bagan Pertandingan').click()
  })

  it('dropdown kategori terisi otomatis dengan kategori pertama (tidak kosong)', () => {
    cy.get('select.filter-select')
      .first()
      .should(($select) => {
        const el = $select[0] as HTMLSelectElement
        expect(el.selectedIndex, 'ada opsi terpilih').to.be.greaterThan(-1)
        expect(el.options[el.selectedIndex].text).to.match(/^\[Dewasa Putra\] - Kelas /)
      })
  })

  it('menampilkan bagan milik kategori terpilih, bukan keadaan kosong', () => {
    cy.get('.bracket-empty-state').should('not.exist')
    cy.contains('.t-round-title', 'BABAK').should('be.visible')
    seeded.athleteNames.forEach((name) => cy.contains('.t-fighter-name', name).should('exist'))
  })

  it('memilih kategori lain yang belum punya bagan menampilkan keadaan kosong, lalu kembali menampilkan bagan', () => {
    // kategori kedua tanpa bagan dibuat lewat API
    cy.then(() =>
      cy.request({
        method: 'POST',
        url: `/api/v1/competitions/${comp.id}/categories`,
        headers: { Authorization: `Bearer ${admin}` },
        body: {
          name: 'Kategori Kosong',
          gender: 'female',
          weight_class: `E${Date.now().toString(36)}`,
        },
      }),
    )
    cy.reload()
    cy.get('select.comp-dropdown').select(comp.id)
    cy.get('aside button.ops-nav__item').contains('Bagan Pertandingan').click()

    cy.get('select.filter-select').first().select('Kategori Kosong')
    cy.get('.bracket-empty-state').should('be.visible')

    cy.get('select.filter-select').first().select(seeded.category.name)
    cy.get('.bracket-empty-state').should('not.exist')
    cy.contains('.t-round-title', 'BABAK').should('be.visible')
  })
})
