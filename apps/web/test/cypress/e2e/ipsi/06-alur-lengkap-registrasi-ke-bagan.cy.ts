import { ADMIN } from '../../support/accounts'
import {
  adminToken,
  birthDateForAge,
  cancelCompetition,
  contingentToken,
  createCategory,
  createCompetition,
  DOCS_BASE,
  DOCS_FEMALE_SENIOR,
  generateBracketRaw,
  getBracket,
  listParticipants,
  registerAthlete,
  verifyParticipantRaw,
  type Category,
  type Competition,
} from '../../support/api'

/**
 * Alur end-to-end: kategori -> registrasi kontingen -> verifikasi/timbang -> bagan -> tampilan publik.
 * Satu skenario berurutan (state dibangun bertahap), sehingga `it` di bawah SENGAJA saling bergantung
 * dan retry per-tes dimatikan agar kegagalan di tengah tidak menutupi akar masalah.
 */
describe('Alur lengkap: registrasi kontingen sampai bagan', { retries: 0 }, () => {
  let admin: string
  let contingent: string
  let comp: Competition
  let remaja: Category
  let putri: Category
  const ids: Record<string, string> = {}

  before(() => {
    adminToken().then((t) => (admin = t))
    contingentToken().then((t) => (contingent = t))
  })

  after(() => {
    if (comp) cancelCompetition(admin, comp.id)
  })

  it('1. admin menyiapkan kejuaraan dan dua kategori resmi', () => {
    createCompetition(admin, 'Alur Lengkap').then((c) => {
      comp = c
      createCategory(admin, c.id, {
        name: 'Remaja Putra A',
        gender: 'male',
        weight_class: 'A',
        age_group: 'remaja',
        min_weight: 39,
        max_weight: 43,
      }).then((k) => (remaja = k))
      createCategory(admin, c.id, {
        name: 'Dewasa Putri A',
        gender: 'female',
        weight_class: 'A',
        age_group: 'dewasa',
        min_weight: 45,
        max_weight: 50,
      }).then((k) => (putri = k))
    })
  })

  it('2. manajer kontingen mendaftarkan 4 atlet dan semuanya berstatus pending', () => {
    const athletes = [
      { key: 'budi', name: 'Budi Remaja', gender: 'male', age: 16, cat: () => remaja },
      { key: 'candra', name: 'Candra Remaja', gender: 'male', age: 15, cat: () => remaja },
      { key: 'dodi', name: 'Dodi Kelebihan', gender: 'male', age: 17, cat: () => remaja },
      { key: 'sari', name: 'Sari Dewasa', gender: 'female', age: 24, cat: () => putri },
    ]
    athletes.forEach((a) => {
      registerAthlete(contingent, comp.id, {
        name: a.name,
        gender: a.gender,
        birth_date: birthDateForAge(a.age),
        category_id: a.cat().id,
      }).then((r) => {
        expect(r.status).to.eq('pending')
        ids[a.key] = r.participantId
      })
    })
    listParticipants(admin, comp.id, 'pending').its('body.data').should('have.length', 4)
  })

  it('3. bagan belum bisa dibuat sebelum ada peserta terverifikasi', () => {
    generateBracketRaw(admin, comp.id, { category_id: remaja.id }).its('status').should('eq', 400)
  })

  it('4. admin memverifikasi: dua lolos, satu kelebihan bobot, satu putri lolos dengan surat tidak hamil', () => {
    const v = (key: string, weight: number, docs: Record<string, boolean>) =>
      verifyParticipantRaw(admin, comp.id, ids[key], {
        status: 'verified',
        actual_weight: weight,
        docs_verified: docs,
      })

    v('budi', 41, { ...DOCS_BASE })
      .its('body.data.status')
      .should('eq', 'verified')
    v('candra', 43, { ...DOCS_BASE })
      .its('body.data.status')
      .should('eq', 'verified')
    v('dodi', 43.5, { ...DOCS_BASE })
      .its('body.data.status')
      .should('eq', 'overweight')
    v('sari', 47, { ...DOCS_FEMALE_SENIOR })
      .its('body.data.status')
      .should('eq', 'verified')

    listParticipants(admin, comp.id, 'verified').its('body.data').should('have.length', 3)
    listParticipants(admin, comp.id, 'overweight').its('body.data').should('have.length', 1)
  })

  it('5. bagan Remaja Putra berisi tepat 2 peserta lolos dan tanpa peserta overweight', () => {
    generateBracketRaw(admin, comp.id, { category_id: remaja.id }).then((res) => {
      expect(res.status).to.eq(201)
      const placed = res.body.data.map((s) => s.participant_id).filter(Boolean)
      expect(placed).to.have.members([ids.budi, ids.candra])
      expect(placed).to.not.include(ids.dodi)
    })
  })

  it('6. kategori Putri dengan satu peserta tidak membentuk bagan (butuh minimal 2)', () => {
    generateBracketRaw(admin, comp.id, { category_id: putri.id }).its('status').should('eq', 400)
  })

  it('7. penonton publik melihat bagan tanpa login, terbatas pada kategorinya', () => {
    getBracket(null, comp.id, remaja.id).then((res) => {
      expect(res.status).to.eq(200)
      expect(res.body.data).to.have.length(2)
      res.body.data.forEach((s) => expect(s.category_id).to.eq(remaja.id))
    })
    getBracket(null, comp.id, putri.id).its('body.data').should('have.length', 0)
  })

  it('8. ringkasan laporan kejuaraan dapat diakses publik', () => {
    cy.request({ url: `/api/v1/competitions/${comp.id}/report/summary`, failOnStatusCode: false })
      .its('status')
      .should('eq', 200)
  })

  it('9. UI: meja verifikasi dimuat tanpa error untuk admin', () => {
    cy.loginAs(ADMIN)
    cy.visit('/ops?tab=verifikasi')
    cy.location('pathname').should('eq', '/ops')
    cy.get('h1.ops-topbar__heading').should('contain.text', 'VERIFIKASI & TIMBANG BADAN')
    cy.contains('Gagal Memuat Antrean Verifikasi').should('not.exist')
    cy.contains('LOLOS VERIFIKASI').should('be.visible')
    cy.contains('ANTREAN TIMBANG').should('be.visible')
  })
})
