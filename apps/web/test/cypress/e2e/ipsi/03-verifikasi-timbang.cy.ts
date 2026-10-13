import {
  adminToken,
  birthDateForAge,
  cancelCompetition,
  contingentToken,
  createCategory,
  createCompetition,
  DOCS_BASE,
  DOCS_FEMALE_SENIOR,
  listParticipants,
  registerAthlete,
  verifyParticipantRaw,
  type Category,
  type Competition,
} from '../../support/api'

/**
 * Verifikasi peserta & timbang badan (regulasi PB IPSI 2025):
 *  - bukti umur + surat sehat wajib untuk semua atlet; surat tidak hamil khusus putri Remaja & Dewasa
 *  - surat izin orang tua TIDAK pernah memblokir
 *  - timbang badan TANPA toleransi (0 kg); batas kelas bersifat inklusif
 */
describe('Verifikasi peserta - berkas & timbang badan', () => {
  let token: string
  let comp: Competition
  const cat: Record<string, Category> = {}

  before(() => {
    adminToken().then((t) => {
      token = t
      createCompetition(t, 'Verifikasi').then((c) => {
        comp = c
        const defs: Array<[string, Parameters<typeof createCategory>[2]]> = [
          [
            'remajaPa',
            {
              name: 'Remaja Putra A',
              gender: 'male',
              weight_class: 'A',
              age_group: 'remaja',
              min_weight: 39,
              max_weight: 43,
            },
          ],
          [
            'remajaPi',
            {
              name: 'Remaja Putri A',
              gender: 'female',
              weight_class: 'A',
              age_group: 'remaja',
              min_weight: 39,
              max_weight: 43,
            },
          ],
          [
            'dewasaPa',
            {
              name: 'Dewasa Putra A',
              gender: 'male',
              weight_class: 'A',
              age_group: 'dewasa',
              min_weight: 45,
              max_weight: 50,
            },
          ],
          [
            'dewasaPi',
            {
              name: 'Dewasa Putri A',
              gender: 'female',
              weight_class: 'A',
              age_group: 'dewasa',
              min_weight: 45,
              max_weight: 50,
            },
          ],
          [
            'praRemajaPi',
            {
              name: 'Pra Remaja Putri A',
              gender: 'female',
              weight_class: 'A',
              age_group: 'pra_remaja',
              min_weight: 30,
              max_weight: 33,
            },
          ],
          [
            'dini1Pa',
            {
              name: 'Usia Dini 1 Putra',
              gender: 'male',
              weight_class: 'MINI',
              age_group: 'usia_dini_1',
            },
          ],
        ]
        defs.forEach(([key, def]) => createCategory(t, c.id, def).then((k) => (cat[key] = k)))
      })
    })
  })

  after(() => {
    if (comp) cancelCompetition(token, comp.id)
  })

  /** Daftarkan atlet baru ke kategori dan kembalikan id pesertanya. */
  const newParticipant = (key: string, age: number, gender: 'male' | 'female') =>
    registerAthlete(token, comp.id, {
      name: `Atlet ${key} ${Date.now().toString(36)}`,
      gender,
      birth_date: birthDateForAge(age),
      category_id: cat[key].id,
    }).then((r) => r.participantId)

  const verify = (pid: string, body: Parameters<typeof verifyParticipantRaw>[3]) =>
    verifyParticipantRaw(token, comp.id, pid, body)

  describe('berkas wajib', () => {
    it('menolak verifikasi bila tidak ada berkas dicentang, dan menyebut berkas yang kurang', () => {
      newParticipant('remajaPa', 16, 'male').then((pid) =>
        verify(pid, { status: 'verified', actual_weight: 41 }).then((res) => {
          expect(res.status).to.eq(400)
          expect(res.body.error?.message)
            .to.match(/berkas belum lengkap/i)
            .and.include('identity')
            .and.include('health')
        }),
      )
    })

    it('hanya menyebut berkas yang benar-benar kurang', () => {
      newParticipant('remajaPa', 16, 'male').then((pid) =>
        verify(pid, {
          status: 'verified',
          actual_weight: 41,
          docs_verified: { identity: true },
        }).then((res) => {
          expect(res.status).to.eq(400)
          expect(res.body.error?.message).to.include('health').and.not.include('identity')
        }),
      )
    })

    it('surat izin orang tua tidak disyaratkan (tidak pernah memblokir)', () => {
      newParticipant('remajaPa', 16, 'male').then((pid) =>
        verify(pid, {
          status: 'verified',
          actual_weight: 41,
          docs_verified: { ...DOCS_BASE },
        }).then((res) => {
          expect(res.status).to.eq(200)
          expect(res.body.data.status).to.eq('verified')
        }),
      )
    })

    it('putri Dewasa wajib surat pernyataan tidak hamil', () => {
      newParticipant('dewasaPi', 25, 'female').then((pid) => {
        verify(pid, {
          status: 'verified',
          actual_weight: 47,
          docs_verified: { ...DOCS_BASE },
        }).then((res) => {
          expect(res.status).to.eq(400)
          expect(res.body.error?.message).to.include('pregnancy')
        })
        verify(pid, {
          status: 'verified',
          actual_weight: 47,
          docs_verified: { ...DOCS_FEMALE_SENIOR },
        })
          .its('body.data.status')
          .should('eq', 'verified')
      })
    })

    it('putri Remaja wajib surat pernyataan tidak hamil', () => {
      newParticipant('remajaPi', 16, 'female').then((pid) =>
        verify(pid, {
          status: 'verified',
          actual_weight: 41,
          docs_verified: { ...DOCS_BASE },
        }).then((res) => {
          expect(res.status).to.eq(400)
          expect(res.body.error?.message).to.include('pregnancy')
        }),
      )
    })

    it('putri Pra Remaja TIDAK wajib surat tidak hamil', () => {
      newParticipant('praRemajaPi', 13, 'female').then((pid) =>
        verify(pid, { status: 'verified', actual_weight: 31, docs_verified: { ...DOCS_BASE } })
          .its('body.data.status')
          .should('eq', 'verified'),
      )
    })

    it('putra Dewasa TIDAK wajib surat tidak hamil', () => {
      newParticipant('dewasaPa', 25, 'male').then((pid) =>
        verify(pid, { status: 'verified', actual_weight: 47, docs_verified: { ...DOCS_BASE } })
          .its('body.data.status')
          .should('eq', 'verified'),
      )
    })

    it('berkas yang sudah dicentang tersimpan dan dikembalikan di daftar peserta', () => {
      newParticipant('remajaPa', 16, 'male').then((pid) => {
        verify(pid, {
          status: 'verified',
          actual_weight: 40,
          docs_verified: { ...DOCS_BASE, parental: true },
        })
        listParticipants(token, comp.id).then((res) => {
          const p = res.body.data.find((x) => x.id === pid)
          expect(p?.docs_verified).to.deep.include({ identity: true, health: true })
        })
      })
    })
  })

  describe('timbang badan tanpa toleransi (kelas 39-43 kg)', () => {
    const cases: Array<{ weight: number; expected: string; note: string }> = [
      { weight: 38.99, expected: 'underweight', note: 'kurang 0,01 kg dari batas bawah' },
      { weight: 39, expected: 'verified', note: 'tepat batas bawah (inklusif)' },
      { weight: 41, expected: 'verified', note: 'di tengah kelas' },
      { weight: 43, expected: 'verified', note: 'tepat batas atas (inklusif)' },
      { weight: 43.01, expected: 'overweight', note: 'lebih 0,01 kg dari batas atas' },
      { weight: 43.1, expected: 'overweight', note: 'lebih 0,1 kg - tidak ada toleransi' },
    ]

    cases.forEach(({ weight, expected, note }) => {
      it(`${weight} kg => ${expected} (${note})`, () => {
        newParticipant('remajaPa', 16, 'male').then((pid) =>
          verify(pid, {
            status: 'verified',
            actual_weight: weight,
            docs_verified: { ...DOCS_BASE },
          }).then((res) => {
            expect(res.status).to.eq(200)
            expect(res.body.data.status).to.eq(expected)
            if (expected !== 'verified') expect(res.body.data.reason).to.match(/tanpa toleransi/i)
          }),
        )
      })
    })

    it('hasil timbang wajib diisi untuk kategori bertingkat bobot', () => {
      newParticipant('remajaPa', 16, 'male').then((pid) =>
        verify(pid, { status: 'verified', docs_verified: { ...DOCS_BASE } }).then((res) => {
          expect(res.status).to.eq(400)
          expect(res.body.error?.message).to.match(/actual_weight/)
        }),
      )
    })

    it('peserta overweight bisa ditimbang ulang dan menjadi verified (koreksi)', () => {
      newParticipant('remajaPa', 16, 'male').then((pid) => {
        verify(pid, { status: 'verified', actual_weight: 44, docs_verified: { ...DOCS_BASE } })
          .its('body.data.status')
          .should('eq', 'overweight')
        verify(pid, { status: 'verified', actual_weight: 42.5 })
          .its('body.data.status')
          .should('eq', 'verified')
      })
    })

    it('Usia Dini 1 tanpa kelas bobot lolos tanpa angka timbangan', () => {
      newParticipant('dini1Pa', 7, 'male').then((pid) =>
        verify(pid, { status: 'verified', docs_verified: { ...DOCS_BASE } })
          .its('body.data.status')
          .should('eq', 'verified'),
      )
    })

    const invalidWeights = [0, -5, 301]
    invalidWeights.forEach((w) => {
      it(`menolak actual_weight tidak masuk akal (${w} kg)`, () => {
        newParticipant('remajaPa', 16, 'male').then((pid) =>
          verify(pid, {
            status: 'verified',
            actual_weight: w,
            docs_verified: { ...DOCS_BASE },
          }).then((res) => {
            expect(res.status).to.eq(400)
            expect(res.body.error?.message).to.match(/actual_weight/)
          }),
        )
      })
    })
  })

  describe('status akhir & penolakan', () => {
    it('penolakan (rejected) tidak mensyaratkan berkas maupun timbangan', () => {
      newParticipant('remajaPa', 16, 'male').then((pid) =>
        verify(pid, { status: 'rejected' }).then((res) => {
          expect(res.status).to.eq(200)
          expect(res.body.data.status).to.eq('rejected')
        }),
      )
    })

    it('menolak nilai status di luar verified/rejected', () => {
      newParticipant('remajaPa', 16, 'male').then((pid) =>
        verify(pid, { status: 'approved' }).then((res) => {
          expect(res.status).to.eq(400)
          expect(res.body.error?.message).to.match(/verified.*rejected/)
        }),
      )
    })

    it('peserta yang tidak ada memberi 404', () => {
      verify('00000000-0000-4000-8000-000000000000', { status: 'rejected' }).then((res) => {
        expect(res.status).to.eq(404)
        expect(res.body.error?.code).to.eq('NOT_FOUND')
        expect(res.body.error?.message).to.not.match(/sql|rows in result/i)
      })
    })

    it('id peserta berformat salah memberi 400 tanpa membocorkan pesan SQL', () => {
      verify('bukan-uuid', { status: 'rejected' }).then((res) => {
        expect(res.status).to.eq(400)
        expect(res.body.error?.message).to.not.match(/sql|uuid|syntax/i)
      })
    })

    it('filter ?status= hanya mengembalikan peserta dengan status tersebut', () => {
      // Siapkan satu peserta untuk tiap hasil agar tes tidak bergantung pada tes lain.
      const outcomes: Array<{ status: string; body: Parameters<typeof verifyParticipantRaw>[3] }> =
        [
          {
            status: 'verified',
            body: { status: 'verified', actual_weight: 41, docs_verified: { ...DOCS_BASE } },
          },
          {
            status: 'overweight',
            body: { status: 'verified', actual_weight: 44, docs_verified: { ...DOCS_BASE } },
          },
          {
            status: 'underweight',
            body: { status: 'verified', actual_weight: 38, docs_verified: { ...DOCS_BASE } },
          },
          { status: 'rejected', body: { status: 'rejected' } },
        ]
      outcomes.forEach(({ body }) => {
        newParticipant('remajaPa', 16, 'male').then((pid) => verify(pid, body))
      })
      outcomes.forEach(({ status }) => {
        listParticipants(token, comp.id, status).then((res) => {
          expect(res.status).to.eq(200)
          expect(res.body.data.length, `ada peserta ${status}`).to.be.greaterThan(0)
          res.body.data.forEach((p) => expect(p.registration_status).to.eq(status))
        })
      })
    })
  })

  describe('hak akses', () => {
    it('manajer kontingen tidak boleh memverifikasi (403)', () => {
      contingentToken().then((ct) =>
        newParticipant('remajaPa', 16, 'male').then((pid) =>
          verifyParticipantRaw(ct, comp.id, pid, { status: 'verified' }).then((res) => {
            expect(res.status).to.eq(403)
            expect(res.body.error?.code).to.eq('FORBIDDEN')
          }),
        ),
      )
    })

    it('tanpa token ditolak (401)', () => {
      newParticipant('remajaPa', 16, 'male').then((pid) =>
        verifyParticipantRaw(null, comp.id, pid, { status: 'verified' })
          .its('status')
          .should('eq', 401),
      )
    })
  })
})
