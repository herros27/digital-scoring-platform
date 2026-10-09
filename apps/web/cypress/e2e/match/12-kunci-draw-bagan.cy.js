import { adminToken, dewanToken } from '../../support/match/auth.ts';
import { bracketSlots, enrollAndVerify, generateBracket, listMatches, playMatch, } from '../../support/match/competition.ts';
import { api, must } from '../../support/match/http.ts';
import { buildKit } from '../../support/match/kit.ts';
import { runAsync } from '../../support/match/run.ts';
const LONG = 120_000;
/**
 * Kunci draw bagan DI SERVER (sebelumnya hanya state di browser panitia): terkunci berarti brackets/generate untuk
 * kategori itu ditolak 409 walau dipanggil langsung lewat API, status kunci tersimpan, hanya admin/dewan yang boleh
 * mengubahnya, dan kunci tidak menghalangi pertandingan berjalan. Bagan yang sudah dimainkan juga tidak boleh
 * dibentuk ulang (akan memutus partai dari slotnya) walau tidak dikunci.
 */
describe('Kunci draw bagan ditegakkan server', { retries: 0 }, () => {
    let kit;
    const lockUrl = () => `/competitions/${kit.competitionId}/categories/${kit.categoryId}/bracket-lock`;
    const category = async () => {
        const list = await must('kategori', api('GET', `/competitions/${kit.competitionId}/categories`, kit.adminToken));
        return list.find((c) => c.id === kit.categoryId);
    };
    const slotIds = async () => (await bracketSlots(kit)).map((s) => s.id).sort();
    before(() => runAsync(async () => {
        kit = await buildKit(await adminToken(), { label: 'Kunci Draw', roundSeconds: 20 });
        await enrollAndVerify(kit, [1, 2, 3, 4].map((i) => ({ name: `Atlet ${i}`, team: `Tim ${i}` })));
    }, LONG));
    after(() => runAsync(async () => kit?.teardown()));
    it('kategori baru terbuka; generate dan acak ulang diizinkan', () => runAsync(async () => {
        expect((await category()).bracket_locked).to.eq(false);
        expect((await generateBracket(kit)).status).to.eq(201);
        const first = await slotIds();
        expect((await generateBracket(kit)).status, 'acak ulang saat terbuka').to.eq(201);
        expect(await slotIds(), 'acak ulang membentuk slot baru').to.not.deep.eq(first);
    }, LONG));
    it('dikunci: status tersimpan di server dan generate (walau lewat API langsung) ditolak 409 tanpa mengubah bagan', () => runAsync(async () => {
        const before = await slotIds();
        const res = await api('PUT', lockUrl(), kit.adminToken, { locked: true });
        expect(res.status).to.eq(200);
        expect(res.data).to.include({ bracket_locked: true });
        expect(res.data.bracket_locked_at).to.be.a('string');
        const persisted = await category();
        expect(persisted.bracket_locked).to.eq(true);
        const refused = await generateBracket(kit);
        expect(refused.status).to.eq(409);
        expect(refused.error?.code).to.eq('BRACKET_LOCKED');
        expect(await slotIds(), 'bagan tidak berubah').to.deep.eq(before);
        expect(await listMatches(kit)).to.have.length(2);
        // idempoten: mengunci lagi tidak mengubah apa pun
        const again = await api('PUT', lockUrl(), kit.adminToken, { locked: true });
        expect(again.status).to.eq(200);
        expect(again.data.bracket_locked_at).to.eq(persisted.bracket_locked_at);
    }, LONG));
    it('hanya admin/dewan yang boleh mengubah kunci; wasit, juri, anonim, dan kategori kejuaraan lain ditolak', () => runAsync(async () => {
        const body = { locked: false };
        expect((await api('PUT', lockUrl(), kit.wasit.token, body)).status, 'wasit').to.eq(403);
        expect((await api('PUT', lockUrl(), kit.judges[0].token, body)).status, 'juri').to.eq(403);
        expect((await api('PUT', lockUrl(), null, body)).status, 'anonim').to.eq(401);
        expect((await category()).bracket_locked, 'tidak ada yang berubah').to.eq(true);
        const otherComp = await must('kejuaraan lain', api('POST', '/competitions', kit.adminToken, {
            name: `Lain ${Date.now()}`,
            location: 'x',
            start_date: '2026-06-01',
            end_date: '2026-06-02',
        }));
        try {
            const foreign = await api('PUT', `/competitions/${otherComp.id}/categories/${kit.categoryId}/bracket-lock`, kit.adminToken, body);
            expect(foreign.status, 'kategori milik kejuaraan lain').to.eq(404);
            expect((await api('PUT', lockUrl(), kit.adminToken, {})).status, 'tanpa field locked').to.eq(400);
        }
        finally {
            await api('POST', `/competitions/${otherComp.id}/cancel`, kit.adminToken);
        }
    }, LONG));
    it('kunci tidak menghalangi pertandingan: partai di bagan terkunci tetap bisa dimainkan dan pemenang maju', () => runAsync(async () => {
        expect((await category()).bracket_locked).to.eq(true);
        const m = (await listMatches(kit)).find((x) => x.status === 'idle');
        const { result } = await playMatch(kit, m.id, [{ target: 'red', type: 'knockdown' }]);
        expect(result.corner).to.eq('red');
        const final = (await bracketSlots(kit)).filter((s) => s.round === 2);
        expect(final.map((s) => s.participant_id)).to.include(m.athlete_red_id);
    }, LONG));
    it('bagan yang sudah dimainkan tidak bisa dibentuk ulang walau kunci dibuka (BRACKET_IN_PROGRESS), dan hasil tetap utuh', () => runAsync(async () => {
        const dewan = await dewanToken();
        expect((await api('PUT', lockUrl(), dewan, { locked: false })).status, 'dewan boleh membuka').to.eq(200);
        expect((await category()).bracket_locked).to.eq(false);
        const before = await slotIds();
        const refused = await generateBracket(kit);
        expect(refused.status).to.eq(409);
        expect(refused.error?.code).to.eq('BRACKET_IN_PROGRESS');
        expect(await slotIds()).to.deep.eq(before);
        const finished = (await listMatches(kit)).filter((x) => x.status === 'finished');
        expect(finished, 'hasil partai yang sudah selesai tidak boleh hilang').to.have.length(1);
    }, LONG));
});
