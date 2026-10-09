import { ADMIN } from './accounts';
Cypress.Commands.add('loginAs', (account) => {
    cy.session(['email', account.email], () => {
        cy.visit('/login');
        cy.get('#login-email').type(account.email);
        cy.get('#login-password').type(account.password, { log: false });
        cy.contains('button[type="submit"]', 'Masuk ke Akun').click();
        cy.location('pathname', { timeout: 15000 }).should('match', /^\/(ops|referee)$/);
    }, {
        validate() {
            cy.window().then((win) => {
                const token = win.sessionStorage.getItem('ds_token') ?? '';
                expect(token.length, 'token sesi tersimpan').to.be.greaterThan(0);
            });
        },
        cacheAcrossSpecs: true,
    });
});
Cypress.Commands.add('loginWithPin', (pin) => {
    cy.visit('/judge/login');
    for (const digit of pin.split('')) {
        cy.contains('button.keypad-btn', new RegExp(`^\\s*${digit}\\s*$`)).click();
    }
    cy.contains('button', /^\s*Masuk\s*$/).click();
});
Cypress.Commands.add('fetchArenaPins', () => {
    return cy
        .request('POST', '/api/v1/auth/login', { email: ADMIN.email, password: ADMIN.password })
        .then((login) => {
        const headers = { Authorization: `Bearer ${login.body.data.token}` };
        return cy.request({ url: '/api/v1/competitions', headers }).then((comps) => {
            const list = comps.body.data;
            expect(list.length, 'ada kompetisi hasil seed').to.be.greaterThan(0);
            const comp = list.find((c) => /menpora/i.test(c.name)) ?? list[0];
            return cy
                .request({ url: `/api/v1/competitions/${comp.id}/arenas`, headers })
                .then((res) => {
                const arenas = res.body.data;
                expect(arenas.length, 'ada gelanggang hasil seed').to.be.greaterThan(0);
                const arenaId = arenas[0].id;
                return cy
                    .request({
                    url: `/api/v1/auth/pins?competition_id=${comp.id}&arena_id=${arenaId}`,
                    headers,
                })
                    .then((p) => {
                    const pins = p.body.data;
                    const result = {
                        competitionId: comp.id,
                        arenaId,
                        pins,
                        find(role, seat) {
                            const found = pins.find((x) => x.role === role && x.seat_number === seat);
                            expect(found, `PIN aktif ${role} kursi ${seat}`).to.not.equal(undefined);
                            return found.pin;
                        },
                    };
                    return result;
                });
            });
        });
    });
});
Cypress.Commands.add('visitWithSession', (path, session) => {
    cy.visit(path, {
        onBeforeLoad(win) {
            win.sessionStorage.setItem('ds_auth_session', JSON.stringify({
                token: session.token,
                role: session.role,
                seatNumber: session.seat,
                arenaId: session.arenaId,
                competitionId: session.competitionId,
            }));
            win.sessionStorage.setItem('ds_token', session.token);
        },
    });
});
