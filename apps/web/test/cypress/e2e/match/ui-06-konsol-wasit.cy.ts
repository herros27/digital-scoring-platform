import { adminToken } from '../../support/match/auth.ts'
import { release, waitState } from '../../support/match/helpers.ts'
import type { Arrangement } from '../../support/match/helpers.ts'
import { must } from '../../support/match/http.ts'
import { buildKit } from '../../support/match/kit.ts'
import type { MatchKit } from '../../support/match/kit.ts'
import { runAsync } from '../../support/match/run.ts'

/**
 * Konsol WASIT di browser nyata: tombol di layar memanggil REST sungguhan, dan perubahan yang dihasilkan
 * dibuktikan dari sisi LAIN (soket para juri), bukan hanya dari tampilan konsol itu sendiri.
 * (Hanya Cypress; alur API/WS-nya diuji di 01-wasit-kontrol-pertandingan.)
 */
describe('Konsol wasit (UI) - memulai dan mengendalikan pertandingan', () => {
  let kit: MatchKit
  let a: Arrangement | undefined

  before(() =>
    runAsync(async () => {
      kit = await buildKit(await adminToken(), {
        label: 'UI Wasit',
        roundSeconds: 20,
        breakSeconds: 3,
        totalRounds: 2,
        outsideWasit: true,
      })
    }),
  )
  after(() => runAsync(async () => kit?.teardown()))
  beforeEach(() =>
    runAsync(async () => {
      const match = await kit.createMatch()
      // tiga juri online lewat soket; wasit dilayani konsol sungguhan
      const devices = await kit.openDevices()
      await devices.wasit.close()
      await must('load-arena', kit.loadMatch(match))
      a = { match, devices }
    }),
  )
  afterEach(() => runAsync(async () => release(kit, a)))

  const openConsole = () => {
    cy.visitWithSession('/referee', kit.wasit)
    cy.contains('SISTEM ONLINE', { timeout: 10000 }).should('be.visible')
  }

  it('menampilkan partai yang dimuat, atlet, dan kehadiran tiga juri', () => {
    openConsole()
    cy.get('.ref-fighter-card--red .ref-fighter-name', { timeout: 10000 }).should(
      'have.text',
      a!.match.redName,
    )
    cy.get('.ref-fighter-card--blue .ref-fighter-name').should('have.text', a!.match.blueName)
    cy.contains('JURI ONLINE: 3/').should('be.visible')
    cy.get('.ref-chrono-round').should('contain.text', 'MENUNGGU')
    cy.contains('button.ref-cmd-btn--start', 'MULAI RONDE').should('not.be.disabled')
    cy.contains('button.ref-cmd-btn--pause', 'JEDA').should('be.disabled')
    cy.contains('button.ref-cmd-btn--resume', 'LANJUTKAN').should('be.disabled')
  })

  it('MULAI RONDE di layar memulai partai di server: seluruh juri menerima round_1 dengan tenggat yang sama', () => {
    openConsole()
    cy.get('.ref-fighter-card--red .ref-fighter-name', { timeout: 10000 }).should(
      'have.text',
      a!.match.redName,
    )
    cy.then(() => {
      a!.judgeMarks = a!.devices.judges.map((s) => s.mark())
    })
    cy.contains('button.ref-cmd-btn--start', 'MULAI RONDE').click()

    // bukti dari sisi juri (bukan hanya tampilan wasit)
    cy.then(() =>
      runAsync(async () => {
        const changes = await Promise.all(
          a!.devices.judges.map((s, i) => waitState(s, 'round_1', a!.judgeMarks![i])),
        )
        changes.forEach((m) => expect(m.payload).to.include({ round: 1, timer_running: true }))
        expect(new Set(changes.map((m) => m.payload.timer_ends_at_ms)).size).to.eq(1)
      }),
    )

    // konsol mengikuti: status berjalan, tombol mulai terkunci, jeda aktif, timer turun
    cy.get('.ref-chrono-round').should('contain.text', 'BABAK 1').and('contain.text', 'BERJALAN')
    cy.contains('button.ref-cmd-btn--start', 'MULAI RONDE').should('be.disabled')
    cy.contains('button.ref-cmd-btn--pause', 'JEDA').should('not.be.disabled')
    cy.get('.ref-chrono-time')
      .invoke('text')
      .then((t1) => {
        cy.wait(1500)
        cy.get('.ref-chrono-time')
          .invoke('text')
          .should((t2) => expect(t2).not.to.eq(t1))
      })
  })

  it('JEDA lalu LANJUTKAN di layar: juri menerima timer berhenti kemudian berjalan lagi', () => {
    openConsole()
    cy.get('.ref-fighter-card--red .ref-fighter-name', { timeout: 10000 }).should(
      'have.text',
      a!.match.redName,
    )
    cy.contains('button.ref-cmd-btn--start', 'MULAI RONDE').click()
    cy.get('.ref-chrono-round').should('contain.text', 'BERJALAN')
    cy.then(() => {
      a!.judgeMarks = a!.devices.judges.map((s) => s.mark())
    })

    cy.contains('button.ref-cmd-btn--pause', 'JEDA').click()
    cy.get('.ref-chrono-round').should('contain.text', 'DIJEDA')
    cy.then(() =>
      runAsync(async () => {
        for (const [i, s] of a!.devices.judges.entries()) {
          const m = await s.waitFor(
            (x) => x.type === 'state_change' && x.payload.timer_running === false,
            { since: a!.judgeMarks![i], what: 'jeda' },
          )
          expect(m.payload.state).to.eq('round_1')
        }
      }),
    )
    cy.contains('button.ref-cmd-btn--resume', 'LANJUTKAN').should('not.be.disabled').click()
    cy.get('.ref-chrono-round').should('contain.text', 'BERJALAN')
    cy.then(() =>
      runAsync(async () => {
        for (const [i, s] of a!.devices.judges.entries()) {
          await s.waitFor(
            (x) =>
              x.type === 'state_change' &&
              x.payload.timer_running === true &&
              x.seq > a!.judgeMarks![i],
            { since: a!.judgeMarks![i], what: 'lanjut' },
          )
        }
      }),
    )
  })

  it('poin konsensus dari juri tampil pada skor sudut dan SELESAIKAN menutup partai untuk semua perangkat', () => {
    openConsole()
    cy.get('.ref-fighter-card--red .ref-fighter-name', { timeout: 10000 }).should(
      'have.text',
      a!.match.redName,
    )
    cy.contains('button.ref-cmd-btn--start', 'MULAI RONDE').click()
    cy.get('.ref-chrono-round').should('contain.text', 'BERJALAN')

    cy.then(() =>
      runAsync(async () => {
        const [j1, j2] = a!.devices.judges
        j1.send('judge_input', {
          event_id: `w-${Date.now()}-1`,
          score_type: 'knockdown',
          target: 'red',
          round: 1,
        })
        j2.send('judge_input', {
          event_id: `w-${Date.now()}-2`,
          score_type: 'knockdown',
          target: 'red',
          round: 1,
        })
      }),
    )
    cy.get('.ref-score-display--red .ref-score-num', { timeout: 8000 }).should('have.text', '2')
    cy.get('.ref-score-display--red .ref-score-note').should('contain.text', 'KONSENSUS JURI')
    cy.get('.ref-score-display--blue .ref-score-num').should('have.text', '0')

    cy.then(() => {
      a!.judgeMarks = a!.devices.judges.map((s) => s.mark())
    })
    cy.contains('button.ref-cmd-btn--finish', 'SELESAIKAN').click()
    cy.get('.ref-modal-box').should('be.visible')
    cy.contains('button.ref-modal-confirm', /Selesaikan|Tetapkan|Akhiri/i).click()

    cy.then(() =>
      runAsync(async () => {
        for (const [i, s] of a!.devices.judges.entries()) {
          const r = await s.waitFor('match_result', {
            since: a!.judgeMarks![i],
            what: 'match_result',
          })
          expect(r.payload).to.include({ corner: 'red' })
        }
      }),
    )
    cy.get('.ref-chrono-round').should('contain.text', 'SELESAI')
    cy.contains('button.ref-cmd-btn--start', 'MULAI RONDE').should('be.disabled')
    cy.contains('button.ref-cmd-btn--finish', 'SELESAIKAN').should('be.disabled')
  })

  it('partai yang dimulai pihak lain lebih dulu: konsol ikut berubah tanpa aksi, dan MULAI tidak bisa dipakai lagi', () => {
    openConsole()
    cy.get('.ref-fighter-card--red .ref-fighter-name', { timeout: 10000 }).should(
      'have.text',
      a!.match.redName,
    )
    // dewan/admin memulai dari tempat lain
    cy.then(() =>
      runAsync(async () => {
        await must('start', kit.control(a!.match, 'start', undefined, kit.adminToken))
      }),
    )
    cy.get('.ref-chrono-round', { timeout: 8000 }).should('contain.text', 'BERJALAN')
    cy.contains('button.ref-cmd-btn--start', 'MULAI RONDE').should('be.disabled')
  })

  it('wasit gelanggang lain melihat konsol kosong: tidak ada partai dari gelanggang ini', () => {
    cy.visitWithSession('/referee', kit.outsideWasit)
    cy.contains('SISTEM ONLINE', { timeout: 10000 }).should('be.visible')
    cy.contains('Belum ada partai di gelanggang ini').should('be.visible')
    cy.get('.ref-fighter-name').should('not.exist')
  })

  it('akses halaman: juri tidak dapat membuka konsol wasit', () => {
    cy.visitWithSession('/referee', kit.judges[0])
    cy.location('pathname', { timeout: 8000 }).should('not.eq', '/referee')
  })
})
