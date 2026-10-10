import {
  adminToken,
  birthDateForAge,
  cancelCompetition,
  contingentToken,
  createCategory,
  createCompetition,
  listParticipants,
  registerAthleteRaw,
  REF_DATE,
  type Category,
  type Competition,
} from '../../support/api'

/**
 * Registrasi atlet ke kategori golongan umur.
 * Usia dihitung pada TANGGAL MULAI KEJUARAAN (REF_DATE), bukan hari ini, sehingga hasilnya stabil
 * kapan pun tes dijalankan.
 */

// Batas golongan umur PB IPSI (docs/rencana_kategori_dan_registrasi_ipsi.md): usia tahun penuh, inklusif.
const IPSI_BANDS = [
  { code: 'pra_usia_dini', label: 'Pra Usia Dini', min: 0, max: 5 },
  { code: 'usia_dini_1', label: 'Usia Dini 1', min: 6, max: 8 },
  { code: 'usia_dini_2', label: 'Usia Dini 2', min: 9, max: 11 },
  { code: 'pra_remaja', label: 'Pra Remaja', min: 12, max: 14 },
  { code: 'remaja', label: 'Remaja', min: 15, max: 17 },
  { code: 'dewasa', label: 'Dewasa', min: 18, max: 35 },
  { code: 'master_a', label: 'Master A', min: 36, max: 45 },
  { code: 'master_b', label: 'Master B', min: 46, max: null },
] as const

describe('Registrasi atlet - golongan umur otomatis (batas usia IPSI)', () => {
  let token: string
  let comp: Competition
  const cats: Record<string, Category> = {}

  before(() => {
    adminToken().then((t) => {
      token = t
      createCompetition(t, 'Umur').then((c) => {
        comp = c
        IPSI_BANDS.forEach((b, i) => {
          createCategory(t, c.id, {
            name: `${b.label} Putra`,
            gender: 'male',
            weight_class: `K${i}`,
            age_group: b.code,
          }).then((cat) => (cats[b.code] = cat))
        })
      })
    })
  })

  after(() => {
    if (comp) cancelCompetition(token, comp.id)
  })

  const register = (band: string, name: string, birth: string) =>
    registerAthleteRaw(token, comp.id, {
      name,
      gender: 'male',
      birth_date: birth,
      category_id: cats[band].id,
    })

  IPSI_BANDS.forEach((b) => {
    describe(`${b.label} (${b.min}${b.max === null ? '+' : `-${b.max}`} tahun)`, () => {
      it(`menerima usia batas bawah ${b.min}`, () => {
        register(b.code, `${b.label} min`, birthDateForAge(b.min, 'birthday'))
          .its('status')
          .should('eq', 201)
      })

      it(`menerima usia batas atas ${b.max ?? 80}`, () => {
        const age = b.max ?? 80
        register(b.code, `${b.label} max`, birthDateForAge(age, 'eve'))
          .its('status')
          .should('eq', 201)
      })

      if (b.min > 0) {
        it(`menolak usia ${b.min - 1} (satu tahun di bawah batas)`, () => {
          register(b.code, `${b.label} kurang`, birthDateForAge(b.min - 1, 'eve')).then((res) => {
            expect(res.status).to.eq(400)
            expect(res.body.error?.message).to.match(/tidak sesuai golongan umur/i)
          })
        })
      }

      if (b.max !== null) {
        it(`menolak usia ${b.max + 1} (satu tahun di atas batas)`, () => {
          register(b.code, `${b.label} lebih`, birthDateForAge(b.max! + 1, 'birthday')).then(
            (res) => {
              expect(res.status).to.eq(400)
              expect(res.body.error?.message).to.match(/tidak sesuai golongan umur/i)
            },
          )
        })
      }
    })
  })

  it('ulang tahun tepat di hari pertama kejuaraan sudah dihitung genap (17 -> 18 = Dewasa)', () => {
    // lahir 2008-06-01: pada 2026-06-01 berusia 18 => Remaja ditolak, Dewasa diterima
    register('remaja', 'Genap 18 remaja', '2008-06-01').its('status').should('eq', 400)
    register('dewasa', 'Genap 18 dewasa', '2008-06-01').its('status').should('eq', 201)
  })

  it('sehari sebelum ulang tahun ke-18 masih Remaja', () => {
    register('remaja', 'H-1 remaja', '2008-06-02').its('status').should('eq', 201)
    register('dewasa', 'H-1 dewasa', '2008-06-02').its('status').should('eq', 400)
  })

  it(`tanggal acuan memakai tanggal mulai kejuaraan (${REF_DATE}), bukan tanggal hari ini`, () => {
    // Lahir 2014-07-01: berusia 11 pada 2026-06-01 (Usia Dini 2, 9-11) tetapi sudah 12 (Pra Remaja)
    // pada tanggal eksekusi tes. Diterima di Usia Dini 2 hanya bila backend memakai tanggal mulai kejuaraan.
    register('usia_dini_2', 'Acuan kejuaraan', '2014-07-01').its('status').should('eq', 201)
    register('pra_remaja', 'Acuan kejuaraan 2', '2014-07-01').its('status').should('eq', 400)
  })
})

describe('Registrasi atlet - validasi input', () => {
  let token: string
  let comp: Competition
  let male: Category
  let female: Category

  before(() => {
    adminToken().then((t) => {
      token = t
      createCompetition(t, 'Validasi').then((c) => {
        comp = c
        createCategory(t, c.id, {
          name: 'Dewasa Putra',
          gender: 'male',
          weight_class: 'A',
          age_group: 'dewasa',
        }).then((cat) => (male = cat))
        createCategory(t, c.id, {
          name: 'Dewasa Putri',
          gender: 'female',
          weight_class: 'A',
          age_group: 'dewasa',
        }).then((cat) => (female = cat))
      })
    })
  })

  after(() => {
    if (comp) cancelCompetition(token, comp.id)
  })

  const base = () => ({ name: 'Atlet Validasi', gender: 'male', birth_date: birthDateForAge(25) })

  it('menolak gender atlet yang tidak sesuai kategori', () => {
    registerAthleteRaw(token, comp.id, { ...base(), gender: 'female', category_id: male.id }).then(
      (res) => {
        expect(res.status).to.eq(400)
        expect(res.body.error?.message).to.match(/gender/i)
      },
    )
  })

  it('menerima alias putra/putri yang setara dengan male/female', () => {
    registerAthleteRaw(token, comp.id, { ...base(), gender: 'putri', category_id: female.id })
      .its('status')
      .should('eq', 201)
    registerAthleteRaw(token, comp.id, { ...base(), gender: 'putra', category_id: male.id })
      .its('status')
      .should('eq', 201)
  })

  it('birth_date wajib untuk kategori berbasis golongan umur', () => {
    registerAthleteRaw(token, comp.id, {
      name: 'Tanpa Tanggal',
      gender: 'male',
      category_id: male.id,
    }).then((res) => {
      expect(res.status).to.eq(400)
      expect(res.body.error?.message).to.match(/birth_date/i)
    })
  })

  it('menolak format tanggal lahir selain YYYY-MM-DD', () => {
    registerAthleteRaw(token, comp.id, {
      ...base(),
      birth_date: '01/01/2000',
      category_id: male.id,
    }).then((res) => {
      expect(res.status).to.eq(400)
      expect(res.body.error?.message).to.match(/YYYY-MM-DD/)
    })
  })

  it('menolak tanggal lahir di masa depan', () => {
    registerAthleteRaw(token, comp.id, {
      ...base(),
      birth_date: '2099-01-01',
      category_id: male.id,
    }).then((res) => {
      expect(res.status).to.eq(400)
      expect(res.body.error?.message).to.match(/future/i)
    })
  })

  it('menolak NIK lebih dari 50 karakter', () => {
    registerAthleteRaw(token, comp.id, {
      ...base(),
      nik: '9'.repeat(51),
      category_id: male.id,
    }).then((res) => {
      expect(res.status).to.eq(400)
      expect(res.body.error?.message).to.match(/nik/i)
    })
  })

  it('menolak kategori yang tidak ada', () => {
    registerAthleteRaw(token, comp.id, {
      ...base(),
      category_id: '00000000-0000-4000-8000-000000000000',
    })
      .its('status')
      .should('eq', 400)
  })

  it('menolak pendaftaran tanpa category_id maupun weight_class', () => {
    registerAthleteRaw(token, comp.id, base()).then((res) => {
      expect(res.status).to.eq(400)
      expect(res.body.error?.message).to.match(/weight_class/i)
    })
  })

  it('pendaftaran yang ditolak tidak meninggalkan peserta', () => {
    listParticipants(token, comp.id).then((res) => {
      const names = res.body.data.map((p) => p.name)
      expect(names).to.not.include('Tanpa Tanggal')
    })
  })

  it('menolak mendaftarkan atlet yang sama dua kali ke kejuaraan yang sama (409)', () => {
    registerAthleteRaw(token, comp.id, { ...base(), name: 'Ganda', category_id: male.id }).then(
      (first) => {
        expect(first.status).to.eq(201)
        registerAthleteRaw(token, comp.id, {
          athlete_id: first.body.data.athlete_id,
          category_id: male.id,
        }).then((dup) => {
          expect(dup.status).to.eq(409)
          expect(dup.body.error?.code).to.eq('CONFLICT')
        })
        listParticipants(token, comp.id).then((res) => {
          const rows = res.body.data.filter((p) => p.athlete_id === first.body.data.athlete_id)
          expect(rows, 'hanya satu baris peserta per atlet').to.have.length(1)
        })
      },
    )
  })
})

describe('Registrasi atlet - peran pendaftar', () => {
  let admin: string
  let contingent: string
  let comp: Competition
  let cat: Category

  before(() => {
    adminToken().then((t) => {
      admin = t
      createCompetition(t, 'Peran').then((c) => {
        comp = c
        createCategory(t, c.id, {
          name: 'Dewasa Putra',
          gender: 'male',
          weight_class: 'A',
          age_group: 'dewasa',
        }).then((k) => (cat = k))
      })
    })
    contingentToken().then((t) => (contingent = t))
  })

  after(() => {
    if (comp) cancelCompetition(admin, comp.id)
  })

  it('pendaftaran oleh admin langsung berstatus "registered" tanpa kontingen', () => {
    registerAthleteRaw(admin, comp.id, {
      name: 'Dari Admin',
      gender: 'male',
      birth_date: birthDateForAge(25),
      category_id: cat.id,
    }).then((res) => {
      expect(res.status).to.eq(201)
      expect(res.body.data.status).to.eq('registered')
      expect(res.body.data.contingent_id).to.eq('')
    })
  })

  it('pendaftaran oleh manajer kontingen berstatus "pending" dan terikat kontingennya', () => {
    registerAthleteRaw(contingent, comp.id, {
      name: 'Dari Kontingen',
      gender: 'male',
      birth_date: birthDateForAge(25),
      category_id: cat.id,
    }).then((res) => {
      expect(res.status).to.eq(201)
      expect(res.body.data.status).to.eq('pending')
      expect(res.body.data.contingent_id).to.be.a('string').and.have.length.greaterThan(0)
    })
  })

  it('daftar peserta menampilkan kategori, golongan umur, dan tanggal lahir', () => {
    listParticipants(admin, comp.id).then((res) => {
      const p = res.body.data.find((x) => x.name === 'Dari Kontingen')
      expect(p, 'peserta kontingen').to.not.equal(undefined)
      expect(p).to.include({
        category_id: cat.id,
        category_name: 'Dewasa Putra',
        age_group: 'dewasa',
      })
      expect(p!.registration_status).to.eq('pending')
      expect(p!.birth_date).to.eq(birthDateForAge(25))
    })
  })

  it('peserta berstatus pending belum muncul pada filter status=verified', () => {
    listParticipants(admin, comp.id, 'verified').its('body.data').should('have.length', 0)
  })

  it('tanpa token ditolak 401', () => {
    registerAthleteRaw(null, comp.id, {
      name: 'Anonim',
      gender: 'male',
      birth_date: birthDateForAge(25),
      category_id: cat.id,
    })
      .its('status')
      .should('eq', 401)
  })
})
