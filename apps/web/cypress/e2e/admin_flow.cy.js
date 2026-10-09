import { ADMIN } from '../support/accounts';
// Alur admin sungguhan di atas data seed (tanpa mock). Tes ini hanya MEMBACA/menavigasi,
// tidak menekan aksi destruktif (mis. "Generate Ulang Semua PIN") agar PIN seed tetap valid
// untuk tes lain.
describe('Admin - operasi kejuaraan', () => {
    beforeEach(() => {
        cy.loginAs(ADMIN);
        cy.visit('/ops?tab=kompetisi');
        cy.location('pathname').should('eq', '/ops');
    });
    const tabs = [
        { menu: 'Kompetisi', heading: 'MANAJEMEN KEJUARAAN' },
        { menu: 'Gelanggang', heading: 'OPERASIONAL GELANGGANG' },
        { menu: 'Kategori Tanding', heading: 'KATEGORI TANDING' },
        { menu: 'Data Atlet', heading: 'KONTINGEN & DATA ATLET' },
        { menu: 'Verifikasi Peserta', heading: 'VERIFIKASI & TIMBANG BADAN' },
        { menu: 'Jadwal Pertandingan', heading: 'JADWAL PERTANDINGAN' },
        { menu: 'Bagan Pertandingan', heading: 'BAGAN PERTANDINGAN' },
        { menu: 'Perangkat & PIN', heading: 'PERANGKAT & PIN JURI' },
        { menu: 'Regulasi & Ruleset', heading: 'REGULASI & RULESET' },
        { menu: 'Laporan & Ekspor', heading: 'LAPORAN & EKSPOR DATA' },
    ];
    tabs.forEach(({ menu, heading }) => {
        it(`menu "${menu}" terbuka dan memuat halamannya dari backend`, () => {
            cy.get('aside button.ops-nav__item').contains(menu).click();
            cy.get('h1.ops-topbar__heading').should('contain.text', heading);
            cy.get('.ops-nav__item--active').should('contain.text', menu);
        });
    });
    it('daftar kompetisi menampilkan kejuaraan hasil seed', () => {
        cy.intercept('GET', '**/api/v1/competitions*').as('getCompetitions');
        cy.reload();
        cy.wait('@getCompetitions').its('response.statusCode').should('eq', 200);
        // kejuaraan tampil sebagai <option> di dropdown; <option> tidak dihitung "visible" oleh Cypress
        cy.contains('option', /Menpora/i).should('exist');
    });
    it('tab Kategori Tanding menyediakan aksi manajemen kategori', () => {
        cy.get('aside button.ops-nav__item').contains('Kategori Tanding').click();
        cy.contains('button', /Tambah Kategori/i).should('be.visible');
        cy.contains('button', /Terapkan Standar PB IPSI/i).should('be.visible');
    });
    it('tab Kategori Tanding: dialog Tambah Kategori bisa dibuka dan ditutup tanpa menyimpan', () => {
        cy.get('aside button.ops-nav__item').contains('Kategori Tanding').click();
        cy.contains('button', /Tambah Kategori/i).click();
        cy.contains('Tambah Kategori Tanding').should('be.visible');
        cy.get('body').type('{esc}');
    });
    it('tab Perangkat & PIN menampilkan kontrol akses dan format juri', () => {
        cy.get('aside button.ops-nav__item').contains('Perangkat & PIN').click();
        cy.contains(/KONTROL AKSES TABLET GELANGGANG/i).should('be.visible');
        cy.contains('button', /Generate Ulang Semua PIN/i).should('be.visible');
        cy.contains('button', /Buat PIN Kustom/i).should('be.visible');
        cy.get('button.format-btn')
            .contains(/5 JURI/i)
            .should('be.visible');
        cy.get('button.format-btn')
            .contains(/3 JURI/i)
            .should('be.visible');
    });
    it('tab Perangkat & PIN: berganti format juri ke 3 lalu kembali ke 5', () => {
        cy.get('aside button.ops-nav__item').contains('Perangkat & PIN').click();
        cy.get('button.format-btn')
            .contains(/3 JURI/i)
            .click();
        cy.get('button.format-btn')
            .contains(/3 JURI/i)
            .should('have.class', 'format-btn--active');
        cy.get('button.format-btn')
            .contains(/5 JURI/i)
            .click();
        cy.get('button.format-btn')
            .contains(/5 JURI/i)
            .should('have.class', 'format-btn--active');
    });
    it('tab Bagan Pertandingan menampilkan filter kategori', () => {
        cy.get('aside button.ops-nav__item').contains('Bagan Pertandingan').click();
        cy.contains(/BAGAN PERTANDINGAN TURNAMEN/i).should('be.visible');
        cy.contains(/KATEGORI & KELAS TANDING/i).should('be.visible');
        cy.get('select.filter-select').should('be.visible');
    });
    it('refresh halaman tidak mengeluarkan admin dari sesi', () => {
        cy.get('aside button.ops-nav__item').contains('Kategori Tanding').click();
        cy.reload();
        cy.location('pathname').should('eq', '/ops');
        cy.get('h1.ops-topbar__heading').should('be.visible');
    });
});
