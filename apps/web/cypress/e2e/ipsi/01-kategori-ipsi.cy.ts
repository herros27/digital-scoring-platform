import {
  adminToken,
  cancelCompetition,
  createCategory,
  createCategoryRaw,
  createCompetition,
  getCategory,
  listCategories,
  listParticipants,
  registerAthlete,
  resetCategories,
  birthDateForAge,
  type Competition,
} from '../../support/api'

/**
 * Kategori resmi PB IPSI / PERSILAT 2026 - kontrak API manajemen kategori.
 * Setiap `describe` memakai kompetisi sendiri sehingga urutan tes tidak saling memengaruhi.
 */
describe('Kategori IPSI - pembuatan & validasi', () => {
  let token: string
  let comp: Competition

  before(() => {
    adminToken().then((t) => {
      token = t
      createCompetition(t, 'Kategori').then((c) => (comp = c))
    })
  })

  after(() => {
    if (comp) cancelCompetition(token, comp.id)
  })

  it('memakai default IPSI, golongan Dewasa, dan bagan gugur tunggal bila tidak diisi', () => {
    createCategory(token, comp.id, {
      name: 'Default Putra',
      gender: 'male',
      weight_class: 'A',
    }).then((cat) => {
      expect(cat).to.include({
        preset_standard: 'ipsi',
        age_group: 'dewasa',
        bracket_mode: 'single_elimination',
        competition_id: comp.id,
      })
    })
  })

  it('menyimpan golongan umur serta batas bobot desimal apa adanya', () => {
    createCategory(token, comp.id, {
      name: 'Remaja Putri B',
      gender: 'female',
      weight_class: 'B',
      age_group: 'remaja',
      min_weight: 43.5,
      max_weight: 47,
    }).then((cat) => {
      expect(cat).to.include({ age_group: 'remaja', min_weight: 43.5, max_weight: 47 })
      getCategory(token, comp.id, cat.id).then((res) => {
        expect(res.status).to.eq(200)
        expect(res.body.data).to.deep.include({ id: cat.id, min_weight: 43.5, max_weight: 47 })
      })
    })
  })

  it('kelas OPEN tanpa batas atas diperbolehkan (hanya min_weight)', () => {
    createCategory(token, comp.id, {
      name: 'Dewasa Putra OPEN 2',
      gender: 'male',
      weight_class: 'OPEN2',
      min_weight: 110,
    }).then((cat) => {
      expect(cat.min_weight).to.eq(110)
      expect(cat.max_weight ?? null).to.eq(null)
    })
  })

  it('menerima alias gender putra/putri', () => {
    createCategory(token, comp.id, {
      name: 'Alias Putri',
      gender: 'putri',
      weight_class: 'C',
    }).then((cat) => {
      expect(cat.gender).to.be.oneOf(['putri', 'female'])
    })
  })

  const invalid: Array<{ title: string; body: Record<string, unknown>; message: RegExp }> = [
    {
      title: 'min_weight lebih besar dari max_weight',
      body: {
        name: 'Salah Bobot',
        gender: 'male',
        weight_class: 'A',
        min_weight: 50,
        max_weight: 43,
      },
      message: /min_weight cannot exceed max_weight/i,
    },
    {
      title: 'min_weight negatif',
      body: { name: 'Bobot Negatif', gender: 'male', weight_class: 'A', min_weight: -1 },
      message: /min_weight cannot be negative/i,
    },
    {
      title: 'max_weight nol',
      body: { name: 'Bobot Nol', gender: 'male', weight_class: 'A', max_weight: 0 },
      message: /max_weight must be positive/i,
    },
    {
      title: 'golongan umur milik PERSILAT dipakai pada preset IPSI',
      body: { name: 'Singa Salah', gender: 'male', weight_class: 'A', age_group: 'singa' },
      message: /age_group 'singa' is not valid for preset ipsi/i,
    },
    {
      title: 'golongan umur tidak dikenal',
      body: { name: 'Umur Ngawur', gender: 'male', weight_class: 'A', age_group: 'balita' },
      message: /not valid/i,
    },
    {
      title: 'preset_standard tidak dikenal',
      body: { name: 'Preset Ngawur', gender: 'male', weight_class: 'A', preset_standard: 'fifa' },
      message: /preset_standard must be/i,
    },
    {
      title: 'bracket_mode tidak dikenal',
      body: { name: 'Mode Ngawur', gender: 'male', weight_class: 'A', bracket_mode: 'round_robin' },
      message: /bracket_mode must be/i,
    },
    {
      title: 'nama terlalu pendek',
      body: { name: 'A', gender: 'male', weight_class: 'A' },
      message: /name/i,
    },
    {
      title: 'gender tidak valid',
      body: { name: 'Gender Ngawur', gender: 'other', weight_class: 'A' },
      message: /gender/i,
    },
    {
      title: 'weight_class kosong',
      body: { name: 'Tanpa Kelas', gender: 'male' },
      message: /weight_class/i,
    },
  ]

  invalid.forEach(({ title, body, message }) => {
    it(`menolak ${title} (400)`, () => {
      createCategoryRaw(token, comp.id, body).then((res) => {
        expect(res.status).to.eq(400)
        expect(res.body.success).to.eq(false)
        expect(res.body.error?.message).to.match(message)
      })
    })
  })

  it('kategori yang ditolak tidak tersimpan', () => {
    listCategories(token, comp.id).then((res) => {
      const names = res.body.data.map((c) => c.name)
      invalid.forEach(({ body }) => expect(names).to.not.include(body.name as string))
    })
  })
})

describe('Kategori IPSI - satu standar per kejuaraan', () => {
  let token: string
  let comp: Competition

  before(() => {
    adminToken().then((t) => {
      token = t
      createCompetition(t, 'Standar').then((c) => (comp = c))
    })
  })

  after(() => {
    if (comp) cancelCompetition(token, comp.id)
  })

  it('PERSILAT 2026 menerima golongan umur khasnya (Singa/Macan/Pre-Teen)', () => {
    ;['singa', 'macan', 'pre_teen'].forEach((group) => {
      createCategory(token, comp.id, {
        name: `Persilat ${group}`,
        gender: 'male',
        weight_class: 'A',
        preset_standard: 'persilat_2026',
        age_group: group,
      }).then((cat) =>
        expect(cat).to.include({ preset_standard: 'persilat_2026', age_group: group }),
      )
    })
  })

  it('menolak mencampur IPSI dengan PERSILAT dalam satu kejuaraan (409)', () => {
    createCategoryRaw(token, comp.id, {
      name: 'IPSI Campuran',
      gender: 'male',
      weight_class: 'A',
      preset_standard: 'ipsi',
    }).then((res) => {
      expect(res.status).to.eq(409)
      expect(res.body.error?.code).to.eq('CONFLICT')
      expect(res.body.error?.message).to.match(/persilat_2026/i)
    })
  })

  it('reset kategori menghapus semuanya sehingga standar bisa diganti', () => {
    resetCategories(token, comp.id).then((res) => {
      expect(res.status).to.eq(200)
      expect(res.body.data.deleted).to.be.greaterThan(0)
    })
    listCategories(token, comp.id).its('body.data').should('have.length', 0)
    createCategory(token, comp.id, {
      name: 'IPSI Setelah Reset',
      gender: 'male',
      weight_class: 'A',
    }).then((cat) => expect(cat.preset_standard).to.eq('ipsi'))
  })

  it('reset ditolak (409) bila kategori masih dipakai peserta', () => {
    createCategory(token, comp.id, {
      name: 'Dewasa Terpakai',
      gender: 'male',
      weight_class: 'Z',
      age_group: 'dewasa',
    }).then((cat) => {
      registerAthlete(token, comp.id, {
        name: 'Peserta Penahan',
        gender: 'male',
        birth_date: birthDateForAge(25),
        category_id: cat.id,
      })
    })
    resetCategories(token, comp.id).then((res) => {
      expect(res.status).to.eq(409)
      expect(res.body.error?.code).to.eq('CONFLICT')
    })
    listParticipants(token, comp.id).its('body.data').should('have.length', 1)
  })
})

describe('Kategori IPSI - isolasi antar kejuaraan & akses', () => {
  let token: string
  let compA: Competition
  let compB: Competition

  before(() => {
    adminToken().then((t) => {
      token = t
      createCompetition(t, 'Isolasi A').then((c) => (compA = c))
      createCompetition(t, 'Isolasi B').then((c) => (compB = c))
    })
  })

  after(() => {
    ;[compA, compB].forEach((c) => c && cancelCompetition(token, c.id))
  })

  it('daftar kategori hanya berisi milik kejuaraannya', () => {
    createCategory(token, compA.id, { name: 'Milik A', gender: 'male', weight_class: 'A' })
    createCategory(token, compB.id, { name: 'Milik B', gender: 'male', weight_class: 'A' })

    listCategories(token, compA.id).then((res) => {
      const names = res.body.data.map((c) => c.name)
      expect(names).to.include('Milik A').and.not.include('Milik B')
    })
    listCategories(token, compB.id).then((res) => {
      const names = res.body.data.map((c) => c.name)
      expect(names).to.include('Milik B').and.not.include('Milik A')
    })
  })

  it('mengambil kategori dengan id yang tidak ada ditolak tanpa 5xx', () => {
    getCategory(token, compA.id, '00000000-0000-4000-8000-000000000000').then((res) => {
      expect(res.status).to.be.oneOf([400, 404])
    })
  })

  it('tanpa token ditolak 401', () => {
    listCategories(null, compA.id).its('status').should('eq', 401)
    createCategoryRaw(null, compA.id, { name: 'Anonim', gender: 'male', weight_class: 'A' })
      .its('status')
      .should('eq', 401)
  })
})
