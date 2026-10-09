import { ADMIN } from '../../support/accounts'
import {
  adminToken,
  apiCall,
  birthDateForAge,
  cancelCompetition,
  contingentToken,
  createCategory,
  createCompetition,
  DOCS_BASE,
  listCategories,
  listParticipants,
  registerAthlete,
  updateAthleteProfile,
  verifyParticipantRaw,
  type Category,
  type Competition,
} from '../../support/api'

/**
 * Pemeriksaan umur untuk peserta berkategori (terutama data lama tanpa/ dengan tanggal lahir keliru) dan
 * kelengkapan data seed (batas bobot kategori, tanggal lahir atlet).
 */
describe('Peninjauan umur peserta (age_check)', () => {
  let admin: string
  let comp: Competition
  let cat: Category

  before(() => {
    adminToken().then((t) => {
      admin = t
      createCompetition(t, 'Tinjau Umur').then((c) => {
        comp = c
        createCategory(t, c.id, {
          name: 'Dewasa Putra Tinjau',
          gender: 'male',
          weight_class: `T${Date.now().toString(36)}`,
          age_group: 'dewasa',
        }).then((k) => (cat = k))
      })
    })
  })

  after(() => {
    if (comp) cancelCompetition(admin, comp.id)
  })

  const newAthlete = (name: string) =>
    registerAthlete(admin, comp.id, {
      name,
      gender: 'male',
      birth_date: birthDateForAge(25),
      category_id: cat.id,
    })

  const ageCheckOf = (athleteId: string) =>
    listParticipants(admin, comp.id).then(
      (res) => res.body.data.find((p) => p.athlete_id === athleteId)?.age_check ?? '',
    )

  it('atlet dengan tanggal lahir sesuai golongan tidak diberi tanda', () => {
    newAthlete('Umur Sesuai').then(({ athleteId }) => {
      ageCheckOf(athleteId).should('eq', '')
    })
  })

  it('tanggal lahir yang keliru memunculkan age_mismatch dan memblokir verifikasi', () => {
    newAthlete('Umur Keliru').then(({ athleteId, participantId }) => {
      updateAthleteProfile(admin, athleteId, { birth_date: birthDateForAge(10) })
        .its('status')
        .should('eq', 200)
      ageCheckOf(athleteId).should('eq', 'age_mismatch')

      verifyParticipantRaw(admin, comp.id, participantId, {
        status: 'verified',
        docs_verified: { ...DOCS_BASE },
      }).then((res) => {
        expect(res.status).to.eq(400)
        expect(res.body.error?.message).to.match(/tidak sesuai golongan umur/i)
      })

      // penolakan tidak diblokir pemeriksaan umur
      verifyParticipantRaw(admin, comp.id, participantId, { status: 'rejected' })
        .its('body.data.status')
        .should('eq', 'rejected')
    })
  })

  it('memperbaiki tanggal lahir menghapus tanda dan membuka kembali verifikasi', () => {
    newAthlete('Umur Diperbaiki').then(({ athleteId, participantId }) => {
      updateAthleteProfile(admin, athleteId, { birth_date: birthDateForAge(10) })
      ageCheckOf(athleteId).should('eq', 'age_mismatch')

      updateAthleteProfile(admin, athleteId, { birth_date: birthDateForAge(25) })
        .its('status')
        .should('eq', 200)
      ageCheckOf(athleteId).should('eq', '')
      verifyParticipantRaw(admin, comp.id, participantId, {
        status: 'verified',
        docs_verified: { ...DOCS_BASE },
      })
        .its('body.data.status')
        .should('eq', 'verified')
    })
  })

  describe('PUT /athletes/{id}', () => {
    it('menolak body kosong, format tanggal salah, tanggal masa depan, dan NIK terlalu panjang', () => {
      newAthlete('Validasi Profil').then(({ athleteId }) => {
        updateAthleteProfile(admin, athleteId, {}).its('status').should('eq', 400)
        updateAthleteProfile(admin, athleteId, { birth_date: '31-12-2000' })
          .its('status')
          .should('eq', 400)
        updateAthleteProfile(admin, athleteId, { birth_date: '2099-01-01' })
          .its('status')
          .should('eq', 400)
        updateAthleteProfile(admin, athleteId, { nik: '9'.repeat(51) })
          .its('status')
          .should('eq', 400)
      })
    })

    it('menyimpan NIK tanpa mengubah tanggal lahir', () => {
      newAthlete('Hanya NIK').then(({ athleteId }) => {
        updateAthleteProfile(admin, athleteId, { nik: '3201999900001111' }).then((res) => {
          expect(res.status).to.eq(200)
          expect(res.body.data.nik).to.eq('3201999900001111')
          expect(res.body.data.birth_date).to.eq(birthDateForAge(25))
        })
      })
    })

    it('atlet yang tidak ada memberi 404 dan tanpa token 401', () => {
      updateAthleteProfile(admin, '00000000-0000-4000-8000-000000000000', {
        birth_date: '2000-01-01',
      })
        .its('status')
        .should('eq', 404)
      updateAthleteProfile(null, '00000000-0000-4000-8000-000000000000', {
        birth_date: '2000-01-01',
      })
        .its('status')
        .should('eq', 401)
    })

    it('manajer kontingen hanya boleh mengubah atlet miliknya', () => {
      contingentToken().then((k1) => {
        // atlet buatan kontingen (milik k1) vs atlet buatan admin (bukan milik k1)
        registerAthlete(k1, comp.id, {
          name: 'Milik Kontingen',
          gender: 'male',
          birth_date: birthDateForAge(25),
          category_id: cat.id,
        }).then(({ athleteId: own }) => {
          updateAthleteProfile(k1, own, { birth_date: birthDateForAge(26) })
            .its('status')
            .should('eq', 200)
        })
        newAthlete('Bukan Milik Kontingen').then(({ athleteId: other }) => {
          updateAthleteProfile(k1, other, { birth_date: birthDateForAge(26) })
            .its('status')
            .should('eq', 404)
          ageCheckOf(other).should('eq', '')
        })
      })
    })
  })

  describe('antarmuka Verifikasi Peserta', () => {
    it('peserta bertanda umur menampilkan peninjauan; tanggal lahir yang benar menghapusnya', () => {
      newAthlete('Umur UI').then(({ athleteId }) => {
        updateAthleteProfile(admin, athleteId, { birth_date: birthDateForAge(10) })
      })
      cy.loginAs(ADMIN)
      cy.visit('/ops?tab=kompetisi')
      cy.get('select.comp-dropdown').select(comp.id)
      cy.get('aside button.ops-nav__item').contains('Verifikasi Peserta').click()

      // kartu dicari lewat nama: peserta lain di kejuaraan ini juga bertanda umur
      cy.contains('.verif-card', 'Umur UI').within(() => {
        cy.get('[data-test="age-review-age_mismatch"]')
          .should('be.visible')
          .and('contain.text', 'tidak sesuai golongan')
        cy.contains('button', 'Simpan').should('be.disabled')
        cy.get('input[type="date"]').type(birthDateForAge(25))
        cy.contains('button', 'Simpan').click()
      })
      cy.contains('.verif-card', 'Umur UI').find('[data-test^="age-review"]').should('not.exist')
    })
  })
})

describe('Kelengkapan data seed (kejuaraan Menpora)', () => {
  let admin: string
  let compId: string

  before(() => {
    adminToken().then((t) => {
      admin = t
      apiCall<Competition[]>('GET', '/competitions', t).then((res) => {
        const seeded = res.body.data.find((c) => /menpora/i.test(c.name))
        expect(seeded, 'kejuaraan seed Menpora').to.not.equal(undefined)
        compId = seeded!.id
      })
    })
  })

  it('kategori Tanding Dewasa punya batas bobot sesuai rentang di namanya (inklusif, tanpa toleransi)', () => {
    listCategories(admin, compId).then((res) => {
      const ranged = res.body.data.filter((c) => /\((\d+)-(\d+) Kg\)/.test(c.name))
      expect(ranged.length, 'kategori seed berrentang').to.be.greaterThan(0)
      ranged.forEach((c) => {
        const [, lo, hi] = c.name.match(/\((\d+)-(\d+) Kg\)/)!
        expect(c.min_weight, `${c.name} min`).to.eq(Number(lo))
        expect(c.max_weight, `${c.name} max`).to.eq(Number(hi))
      })
    })
  })

  it('peserta seed bertanggal lahir dan tidak ada yang ditandai age_check', () => {
    listParticipants(admin, compId).then((res) => {
      expect(res.body.data.length).to.be.greaterThan(0)
      res.body.data.forEach((p) => {
        expect(p.birth_date, `${p.name} tanggal lahir`)
          .to.be.a('string')
          .and.have.length.greaterThan(0)
        expect(p.age_check ?? '', `${p.name} age_check`).to.eq('')
      })
    })
  })

  it('peserta seed tertaut ke kategori (bukan bagan jalur lama)', () => {
    listParticipants(admin, compId).then((res) => {
      res.body.data.forEach((p) =>
        expect(p.category_id, `${p.name} category_id`).to.be.a('string').and.not.eq(''),
      )
    })
  })
})
