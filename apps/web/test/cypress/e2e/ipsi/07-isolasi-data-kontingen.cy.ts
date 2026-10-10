import { ADMIN, KONTINGEN, KONTINGEN_PSHT } from '../../support/accounts'
import {
  apiCall,
  apiToken,
  birthDateForAge,
  cancelCompetition,
  createCategory,
  createCompetition,
  DOCS_BASE,
  listParticipants,
  registerAthleteRaw,
  verifyParticipantRaw,
  type Category,
  type Competition,
  type Registration,
} from '../../support/api'

/**
 * Manajer kontingen hanya boleh melihat/menyentuh data yang ia daftarkan sendiri beserta status
 * verifikasinya. Data kontingen lain, atlet yang didaftarkan admin, dan profil kontingen lain tidak boleh
 * terbaca. Admin/dewan tetap melihat semuanya.
 */
describe('Isolasi data antar kontingen', () => {
  let admin: string
  let k1: string // Tapak Suci
  let k2: string // PSHT
  let comp: Competition
  let cat: Category
  const own: Record<'k1' | 'k2' | 'admin', { athleteId: string; participantId: string }> = {
    k1: { athleteId: '', participantId: '' },
    k2: { athleteId: '', participantId: '' },
    admin: { athleteId: '', participantId: '' },
  }

  const register = (token: string, name: string) =>
    registerAthleteRaw(token, comp.id, {
      name,
      gender: 'male',
      birth_date: birthDateForAge(25),
      category_id: cat.id,
    })

  before(() => {
    apiToken(ADMIN).then((t) => (admin = t))
    apiToken(KONTINGEN).then((t) => (k1 = t))
    apiToken(KONTINGEN_PSHT).then((t) => (k2 = t))

    cy.then(() => {
      createCompetition(admin, 'Isolasi Kontingen').then((c) => {
        comp = c
        createCategory(admin, c.id, {
          name: 'Dewasa Putra',
          gender: 'male',
          weight_class: 'A',
          age_group: 'dewasa',
        }).then((k) => {
          cat = k
          ;(['k1', 'k2', 'admin'] as const).forEach((who) => {
            const token = who === 'k1' ? k1 : who === 'k2' ? k2 : admin
            register(token, `Atlet ${who}`).then((res) => {
              expect(res.status).to.eq(201)
              own[who].athleteId = res.body.data.athlete_id
            })
          })
          // ambil id peserta dari daftar admin (kontingen hanya melihat miliknya)
          listParticipants(admin, c.id).then((res) => {
            ;(['k1', 'k2', 'admin'] as const).forEach((who) => {
              const row = res.body.data.find((p) => p.athlete_id === own[who].athleteId)
              expect(row, `peserta ${who}`).to.not.equal(undefined)
              own[who].participantId = row!.id
            })
          })
        })
      })
    })
  })

  after(() => {
    if (comp) cancelCompetition(admin, comp.id)
  })

  describe('daftar peserta', () => {
    it('kontingen hanya melihat peserta miliknya beserta status verifikasinya', () => {
      listParticipants(k1, comp.id).then((res) => {
        expect(res.status).to.eq(200)
        const ids = res.body.data.map((p) => p.id)
        expect(ids).to.deep.eq([own.k1.participantId])
        expect(res.body.data[0].registration_status).to.eq('pending')
      })
    })

    it('kontingen lain tidak melihat peserta k1 (dan sebaliknya)', () => {
      listParticipants(k2, comp.id).then((res) => {
        expect(res.body.data.map((p) => p.id)).to.deep.eq([own.k2.participantId])
      })
    })

    it('peserta yang didaftarkan admin tidak tampil bagi kontingen mana pun', () => {
      ;[k1, k2].forEach((tk) =>
        listParticipants(tk, comp.id).then((res) => {
          expect(res.body.data.map((p) => p.id)).to.not.include(own.admin.participantId)
        }),
      )
    })

    it('filter ?status= tidak membuka data milik kontingen lain', () => {
      // status "registered" hanya dimiliki peserta yang didaftarkan admin
      listParticipants(k1, comp.id, 'registered').its('body.data').should('have.length', 0)
    })

    it('pemilik melihat status terbaru setelah admin memverifikasi', () => {
      verifyParticipantRaw(admin, comp.id, own.k1.participantId, {
        status: 'verified',
        docs_verified: { ...DOCS_BASE },
      })
        .its('body.data.status')
        .should('eq', 'verified')

      listParticipants(k1, comp.id).then((res) => {
        expect(res.body.data).to.have.length(1)
        expect(res.body.data[0].registration_status).to.eq('verified')
      })
      // kontingen lain tetap tidak melihat peserta yang baru terverifikasi itu
      listParticipants(k2, comp.id).then((res) => {
        expect(res.body.data.map((p) => p.id)).to.not.include(own.k1.participantId)
      })
    })

    it('admin tetap melihat semua peserta', () => {
      listParticipants(admin, comp.id).then((res) => {
        const ids = res.body.data.map((p) => p.id)
        expect(ids).to.include.members([
          own.k1.participantId,
          own.k2.participantId,
          own.admin.participantId,
        ])
      })
    })
  })

  describe('daftar atlet kejuaraan (halaman Data Atlet)', () => {
    const listAthletes = (token: string | null) =>
      apiCall<Array<{ id: string; nik?: string }>>(
        'GET',
        `/competitions/${comp.id}/athletes`,
        token,
      )

    it('tanpa login ditolak 401 (data atlet memuat tanggal lahir & NIK)', () => {
      listAthletes(null).its('status').should('eq', 401)
      apiCall('GET', '/athletes', null).its('status').should('eq', 401)
    })

    it('kontingen hanya menerima atlet kontingennya sendiri', () => {
      listAthletes(k1).then((res) => {
        expect(res.status).to.eq(200)
        expect(res.body.data.map((a) => a.id)).to.deep.eq([own.k1.athleteId])
      })
      listAthletes(k2).then((res) => {
        expect(res.body.data.map((a) => a.id)).to.deep.eq([own.k2.athleteId])
      })
    })

    it('respons admin tidak bocor ke kontingen lewat cache (urutan admin -> kontingen)', () => {
      listAthletes(admin).its('body.data').should('have.length.at.least', 3)
      listAthletes(k1).then((res) => {
        const ids = res.body.data.map((a) => a.id)
        expect(ids).to.not.include.members([own.k2.athleteId])
        expect(ids).to.not.include(own.admin.athleteId)
      })
    })

    it('admin melihat atlet semua kontingen', () => {
      listAthletes(admin).then((res) => {
        expect(res.body.data.map((a) => a.id)).to.include.members([
          own.k1.athleteId,
          own.k2.athleteId,
          own.admin.athleteId,
        ])
      })
    })

    it('daftar master /athletes hanya untuk admin/dewan (kontingen 403)', () => {
      apiCall('GET', '/athletes?limit=1', k1).its('status').should('eq', 403)
      apiCall('GET', '/athletes?limit=1', admin).its('status').should('eq', 200)
    })
  })

  describe('detail atlet', () => {
    const getAthlete = (token: string, id: string) => apiCall('GET', `/athletes/${id}`, token)

    it('kontingen boleh membuka atlet miliknya', () => {
      getAthlete(k1, own.k1.athleteId).its('status').should('eq', 200)
    })

    it('atlet kontingen lain dan atlet buatan admin disamarkan sebagai 404', () => {
      getAthlete(k1, own.k2.athleteId).its('status').should('eq', 404)
      getAthlete(k1, own.admin.athleteId).its('status').should('eq', 404)
    })

    it('admin boleh membuka atlet kontingen mana pun', () => {
      getAthlete(admin, own.k2.athleteId).its('status').should('eq', 200)
    })
  })

  describe('pendaftaran & pembuatan data', () => {
    it('kontingen tidak bisa mendaftarkan atlet milik kontingen lain (404)', () => {
      registerAthleteRaw(k1, comp.id, { athlete_id: own.k2.athleteId, category_id: cat.id }).then(
        (res) => {
          expect(res.status).to.eq(404)
        },
      )
      listParticipants(admin, comp.id).then((res) => {
        const rows = res.body.data.filter((p) => p.athlete_id === own.k2.athleteId)
        expect(rows, 'atlet k2 tetap satu baris').to.have.length(1)
      })
    })

    it('contingent_id pada body POST /athletes diabaikan: atlet selalu milik pembuat', () => {
      apiCall<{ id: string; contingent_id: string }>('POST', '/athletes', k1, {
        name: `Palsu ${Date.now()}`,
        gender: 'male',
        team: 'Cypress',
        // mencoba mengatasnamakan kontingen lain
        contingent_id: '00000000-0000-4000-8000-000000000001',
      }).then((created) => {
        expect(created.status).to.eq(201)
        // atlet itu harus bisa dibuka pembuatnya (kepemilikan = k1), bukan milik id palsu
        apiCall('GET', `/athletes/${created.body.data.id}`, k1).its('status').should('eq', 200)
        apiCall('GET', `/athletes/${created.body.data.id}`, k2).its('status').should('eq', 404)
      })
    })

    it('pendaftaran kontingen tetap berstatus pending dan terikat kontingennya', () => {
      register(k1, `Tambahan ${Date.now()}`).then((res) => {
        expect(res.status).to.eq(201)
        const reg: Registration = res.body.data
        expect(reg.status).to.eq('pending')
        expect(reg.contingent_id).to.be.a('string').and.have.length.greaterThan(0)
      })
    })
  })

  describe('profil kontingen', () => {
    it('kontingen boleh membaca profilnya sendiri tetapi tidak profil kontingen lain (403)', () => {
      apiCall<{ id: string }>('GET', '/contingents', admin).then((list) => {
        expect(list.status).to.eq(200)
        const all = list.body.data as unknown as Array<{ id: string; email: string }>
        const mine = all.find((c) => c.email === KONTINGEN.email)!
        const other = all.find((c) => c.email === KONTINGEN_PSHT.email)!
        apiCall('GET', `/contingents/${mine.id}`, k1).its('status').should('eq', 200)
        apiCall('GET', `/contingents/${other.id}`, k1).its('status').should('eq', 403)
        apiCall('GET', '/contingents', k1).its('status').should('eq', 403)
      })
    })
  })
})
