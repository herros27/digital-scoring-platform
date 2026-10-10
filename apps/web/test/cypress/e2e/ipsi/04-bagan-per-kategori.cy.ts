import {
  adminToken,
  birthDateForAge,
  cancelCompetition,
  contingentToken,
  createCategory,
  createCompetition,
  DOCS_BASE,
  generateBracketRaw,
  getBracket,
  registerAthlete,
  verifyParticipantRaw,
  type BracketSlot,
  type Category,
  type Competition,
} from '../../support/api'

/**
 * Bagan pertandingan terisolasi per kategori. Hanya peserta berstatus VERIFIED yang boleh masuk bagan.
 * Struktur yang diverifikasi: ronde 1 berisi 2^k slot (k = ceil(log2 N)) dengan N peserta + sisa bye,
 * dan jumlah ronde = k.
 */
describe('Bagan per kategori', () => {
  let token: string
  let comp: Competition

  before(() => {
    adminToken().then((t) => {
      token = t
      createCompetition(t, 'Bagan').then((c) => (comp = c))
    })
  })

  after(() => {
    if (comp) cancelCompetition(token, comp.id)
  })

  /** Buat kategori Dewasa Putra tanpa batas bobot (agar verifikasi tidak butuh timbangan). */
  let classSeq = 0
  const makeCategory = (name: string): Cypress.Chainable<Category> => {
    classSeq += 1
    // weight_class unik per kategori agar tidak ada tabrakan kunci (weight_class, gender) di bagan.
    return createCategory(token, comp.id, {
      name,
      gender: 'male',
      weight_class: `W${classSeq}`,
      age_group: 'dewasa',
    })
  }

  /** Daftarkan lalu verifikasi `n` atlet; mengembalikan id peserta. */
  const verifiedAthletes = (cat: Category, n: number, prefix: string) => {
    const ids: string[] = []
    for (let i = 0; i < n; i += 1) {
      registerAthlete(token, comp.id, {
        name: `${prefix} ${i + 1}`,
        gender: 'male',
        birth_date: birthDateForAge(20 + i),
        category_id: cat.id,
      }).then(({ participantId }) => {
        verifyParticipantRaw(token, comp.id, participantId, {
          status: 'verified',
          docs_verified: { ...DOCS_BASE },
        })
          .its('body.data.status')
          .should('eq', 'verified')
        ids.push(participantId)
      })
    }
    return cy.then(() => ids)
  }

  const isPow2 = (n: number) => n > 0 && (n & (n - 1)) === 0
  const nextPow2 = (n: number) => 2 ** Math.ceil(Math.log2(n))

  describe('struktur bagan menurut jumlah peserta', () => {
    ;[2, 3, 4, 5, 8, 9].forEach((n) => {
      it(`${n} peserta terverifikasi`, () => {
        makeCategory(`Struktur ${n}`).then((cat) => {
          verifiedAthletes(cat, n, `S${n}`).then((ids) => {
            generateBracketRaw(token, comp.id, { category_id: cat.id }).then((res) => {
              expect(res.status).to.eq(201)
              const slots: BracketSlot[] = res.body.data
              const round1 = slots.filter((s) => s.round === 1)
              const size = nextPow2(n)

              expect(round1, 'jumlah slot ronde 1').to.have.length(size)
              expect(isPow2(round1.length)).to.eq(true)
              expect(
                round1.filter((s) => s.is_bye),
                'jumlah bye',
              ).to.have.length(size - n)

              const placed = round1.map((s) => s.participant_id).filter((p): p is string => !!p)
              expect(placed.sort(), 'setiap peserta tepat satu kali').to.deep.eq([...ids].sort())

              expect(Math.max(...slots.map((s) => s.round)), 'jumlah ronde').to.eq(Math.log2(size))
              slots.forEach((s) => expect(s.category_id).to.eq(cat.id))
            })
          })
        })
      })
    })
  })

  describe('hanya peserta terverifikasi yang masuk bagan', () => {
    it('mengecualikan pending, overweight, underweight, rejected, dan registered', () => {
      createCategory(token, comp.id, {
        name: 'Filter Status',
        gender: 'male',
        weight_class: 'FS',
        age_group: 'dewasa',
        min_weight: 60,
        max_weight: 65,
      }).then((cat) => {
        const mk = (name: string, via: 'admin' | 'contingent') =>
          (via === 'admin' ? cy.wrap(token) : contingentToken()).then((tk) =>
            registerAthlete(tk, comp.id, {
              name,
              gender: 'male',
              birth_date: birthDateForAge(25),
              category_id: cat.id,
            }).then((r) => ({ ...r, name })),
          )

        const verify = (pid: string, weight: number) =>
          verifyParticipantRaw(token, comp.id, pid, {
            status: 'verified',
            actual_weight: weight,
            docs_verified: { ...DOCS_BASE },
          })

        const expectedIn: string[] = []
        const expectedOut: string[] = []

        mk('Lolos A', 'admin').then((p) =>
          verify(p.participantId, 62).then(() => expectedIn.push(p.participantId)),
        )
        mk('Lolos B', 'admin').then((p) =>
          verify(p.participantId, 65).then(() => expectedIn.push(p.participantId)),
        )
        mk('Kelebihan', 'admin').then((p) =>
          verify(p.participantId, 65.1).then(() => expectedOut.push(p.participantId)),
        )
        mk('Kekurangan', 'admin').then((p) =>
          verify(p.participantId, 59.9).then(() => expectedOut.push(p.participantId)),
        )
        mk('Ditolak', 'admin').then((p) =>
          verifyParticipantRaw(token, comp.id, p.participantId, { status: 'rejected' }).then(() =>
            expectedOut.push(p.participantId),
          ),
        )
        mk('Belum Timbang', 'contingent').then((p) => expectedOut.push(p.participantId)) // pending
        mk('Baru Terdaftar', 'admin').then((p) => expectedOut.push(p.participantId)) // registered

        cy.then(() =>
          generateBracketRaw(token, comp.id, { category_id: cat.id }).then((res) => {
            expect(res.status).to.eq(201)
            const placed = res.body.data
              .map((s) => s.participant_id)
              .filter((p): p is string => !!p)
            expect(placed.sort()).to.deep.eq([...expectedIn].sort())
            expectedOut.forEach((id) => expect(placed).to.not.include(id))
          }),
        )
      })
    })

    it('menolak (400) bila peserta terverifikasi kurang dari 2', () => {
      makeCategory('Hanya Satu').then((cat) => {
        verifiedAthletes(cat, 1, 'Tunggal')
        generateBracketRaw(token, comp.id, { category_id: cat.id }).then((res) => {
          expect(res.status).to.eq(400)
          expect(res.body.error?.message).to.match(/at least 2 verified participants/i)
        })
      })
    })

    it('kategori tanpa peserta ditolak dengan pesan jelas, bukan 5xx', () => {
      makeCategory('Kosong').then((cat) =>
        generateBracketRaw(token, comp.id, { category_id: cat.id }).then((res) => {
          expect(res.status).to.eq(400)
        }),
      )
    })
  })

  describe('isolasi antar kategori', () => {
    it('bagan kategori A dan B tidak saling berisi peserta', () => {
      makeCategory('Isolasi A').then((a) =>
        makeCategory('Isolasi B').then((b) => {
          verifiedAthletes(a, 3, 'IsoA').then((idsA) =>
            verifiedAthletes(b, 4, 'IsoB').then((idsB) => {
              generateBracketRaw(token, comp.id, { category_id: a.id })
                .its('status')
                .should('eq', 201)
              generateBracketRaw(token, comp.id, { category_id: b.id })
                .its('status')
                .should('eq', 201)

              const placedIn = (cat: Category) =>
                getBracket(token, comp.id, cat.id).then((res) => {
                  expect(res.status).to.eq(200)
                  res.body.data.forEach((s) => expect(s.category_id).to.eq(cat.id))
                  // hanya ronde 1: pemenang bye sudah terisi di ronde berikutnya sehingga muncul dua kali di semua slot
                  return res.body.data
                    .filter((s) => s.round === 1)
                    .map((s) => s.participant_id)
                    .filter((p): p is string => !!p)
                })

              placedIn(a).then((placed) => {
                expect(placed.sort()).to.deep.eq([...idsA].sort())
                idsB.forEach((id) => expect(placed).to.not.include(id))
              })
              placedIn(b).then((placed) => {
                expect(placed.sort()).to.deep.eq([...idsB].sort())
                idsA.forEach((id) => expect(placed).to.not.include(id))
              })
            }),
          )
        }),
      )
    })

    it('generate ulang menggantikan bagan lama (tidak menggandakan slot)', () => {
      makeCategory('Ulang').then((cat) => {
        verifiedAthletes(cat, 4, 'Ulang')
        generateBracketRaw(token, comp.id, { category_id: cat.id }).its('status').should('eq', 201)
        getBracket(token, comp.id, cat.id).then((first) => {
          generateBracketRaw(token, comp.id, { category_id: cat.id })
            .its('status')
            .should('eq', 201)
          getBracket(token, comp.id, cat.id).then((second) => {
            expect(second.body.data).to.have.length(first.body.data.length)
            const placed = second.body.data.map((s) => s.participant_id).filter(Boolean)
            expect(new Set(placed).size, 'tidak ada peserta ganda').to.eq(placed.length)
          })
        })
      })
    })

    it('kategori dari kejuaraan lain ditolak (404)', () => {
      createCompetition(token, 'Bagan Lain').then((other) => {
        createCategory(token, other.id, {
          name: 'Kategori Lain',
          gender: 'male',
          weight_class: 'X',
        }).then((foreign) => {
          generateBracketRaw(token, comp.id, { category_id: foreign.id })
            .its('status')
            .should('eq', 404)
          cancelCompetition(token, other.id)
        })
      })
    })
  })

  describe('hak akses', () => {
    it('manajer kontingen tidak boleh membuat bagan (403)', () => {
      makeCategory('Akses').then((cat) => {
        verifiedAthletes(cat, 2, 'Akses')
        contingentToken().then((ct) =>
          generateBracketRaw(ct, comp.id, { category_id: cat.id }).its('status').should('eq', 403),
        )
      })
    })

    it('tanpa token tidak bisa membuat bagan (401), tetapi bagan bisa dilihat publik', () => {
      makeCategory('Publik').then((cat) => {
        verifiedAthletes(cat, 2, 'Publik')
        generateBracketRaw(null, comp.id, { category_id: cat.id }).its('status').should('eq', 401)
        generateBracketRaw(token, comp.id, { category_id: cat.id }).its('status').should('eq', 201)
        getBracket(null, comp.id, cat.id).then((res) => {
          expect(res.status).to.eq(200)
          expect(res.body.data).to.have.length(2)
        })
      })
    })
  })
})
