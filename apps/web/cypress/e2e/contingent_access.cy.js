import { KONTINGEN } from '../support/accounts';
describe('Manajer kontingen - hak akses nyata', () => {
    beforeEach(() => {
        cy.loginAs(KONTINGEN);
        cy.visit('/ops?tab=atlet');
    });
    it('melihat menu yang relevan untuk kontingen', () => {
        cy.get('h1.ops-topbar__heading').should('contain.text', 'KONTINGEN & DATA ATLET');
        cy.get('aside button.ops-nav__item').contains('Data Atlet').should('be.visible');
        cy.get('aside button.ops-nav__item').contains('Jadwal Pertandingan').should('be.visible');
        cy.get('aside button.ops-nav__item').contains('Bagan Pertandingan').should('be.visible');
    });
    it('tidak melihat menu khusus admin/dewan', () => {
        const adminOnly = [
            'Kompetisi',
            'Kategori Tanding',
            'Verifikasi Peserta',
            'Perangkat & PIN',
            'Regulasi & Ruleset',
        ];
        adminOnly.forEach((menu) => {
            cy.get('aside').should('not.contain.text', menu);
        });
    });
    it('membuka ?tab=pin langsung tidak memberi akses ke halaman PIN', () => {
        cy.visit('/ops?tab=pin');
        cy.get('h1.ops-topbar__heading').should('not.contain.text', 'PERANGKAT & PIN JURI');
        cy.contains('KONTROL AKSES TABLET GELANGGANG').should('not.exist');
    });
    it('tidak bisa membuka konsol wasit', () => {
        cy.visit('/referee');
        cy.location('pathname').should('not.eq', '/referee');
    });
});
