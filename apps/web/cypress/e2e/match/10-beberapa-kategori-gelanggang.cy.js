import { adminToken } from '../../support/match/auth.ts';
import { bracketSlots, enrollAndVerify, generateBracket, listMatches, playBracketToEnd, } from '../../support/match/competition.ts';
import { api, must } from '../../support/match/http.ts';
import { buildKit } from '../../support/match/kit.ts';
import { runAsync } from '../../support/match/run.ts';
const LONG = 300_000;
const TEAMS = ['Kontingen Garuda', 'Kontingen Rajawali', 'Kontingen Elang', 'Kontingen Merpati'];
const DOCS_FEMALE = { identity: true, health: true, pregnancy: true };
/**
 * Dua kategori (Putra dan Putri) yang berjalan BERSAMAAN di dua gelanggang, masing-masing dengan wasit dan 3 juri
 * sendiri, dengan kontingen yang sama di kedua kategori. Memastikan bagan tiap kategori terisolasi, dua partai
 * berlangsung serentak tanpa saling mengganggu, partai dapat berpindah gelanggang antar babak (semifinal di satu
 * gelanggang, final di gelanggang lain), dan medali kontingen dijumlah dari kedua kategori.
 */
describe('Dua kategori di dua gelanggang berjalan bersamaan', { retries: 0 }, () => {
    let kit;
    let putraId;
    let putriId;
    let putra = [];
    let putri = [];
    const played = [];
    before(() => runAsync(async () => {
        kit = await buildKit(await adminToken(), {
            label: 'Dua Kategori',
            roundSeconds: 20,
            breakSeconds: 2,
            totalRounds: 2,
            otherArenaCrew: true,
        });
        putraId = kit.categoryId;
        putriId = await kit.createCategory({
            name: 'Dewasa Putri A',
            gender: 'female',
            weight_class: 'A',
            age_group: 'dewasa',
        });
        putra = await enrollAndVerify(kit, TEAMS.map((team, i) => ({ name: `Putra ${i + 1}`, team })));
        putri = await enrollAndVerify(kit, TEAMS.map((team, i) => ({ name: `Putri ${i + 1}`, team })), { categoryId: putriId, gender: 'female', docs: DOCS_FEMALE });
    }, LONG));
    after(() => runAsync(async () => kit?.teardown()));
    it('1. bagan tiap kategori terisolasi: hanya berisi pesertanya sendiri, masing-masing 2 semifinal', () => runAsync(async () => {
        expect((await generateBracket(kit, putraId)).status).to.eq(201);
        expect((await generateBracket(kit, putriId)).status).to.eq(201);
        for (const [cat, list] of [
            [putraId, putra],
            [putriId, putri],
        ]) {
            const ids = (await bracketSlots(kit, cat)).map((s) => s.participant_id).filter(Boolean);
            expect(ids).to.have.members(list.map((e) => e.participantId));
        }
        const matches = await listMatches(kit);
        expect(matches, '2 semifinal per kategori').to.have.length(4);
        const slotsPutra = new Set((await bracketSlots(kit, putraId)).map((s) => s.id));
        const ofPutra = matches.filter((m) => slotsPutra.has(m.bracket_slot_id));
        expect(ofPutra).to.have.length(2);
        ofPutra.forEach((m) => expect([m.athlete_red_id, m.athlete_blue_id].every((id) => putra.some((e) => e.participantId === id))).to.eq(true));
    }, LONG));
    it('2. semifinal Putra di gelanggang utama dan semifinal Putri di gelanggang lain berlangsung SERENTAK', () => runAsync(async () => {
        const [a, b] = await Promise.all([
            playBracketToEnd(kit, { categoryId: putraId, arena: 'main', maxPasses: 1 }),
            playBracketToEnd(kit, { categoryId: putriId, arena: 'other', maxPasses: 1 }),
        ]);
        expect(a).to.have.length(2);
        expect(b).to.have.length(2);
        played.push(...a, ...b);
        // tiap partai tercatat di gelanggang tempat ia dimainkan
        const all = await listMatches(kit);
        for (const [list, arena] of [
            [a, 'main'],
            [b, 'other'],
        ]) {
            for (const p of list) {
                const rec = all.find((m) => m.id === p.id);
                expect(rec.status).to.eq('finished');
                expect(rec.arena_id).to.eq(kit.crew(arena).arenaId);
            }
        }
        // final kedua kategori terbentuk otomatis, kategorinya tidak tertukar
        const idle = all.filter((m) => m.status === 'idle');
        expect(idle, 'satu final per kategori').to.have.length(2);
    }, LONG));
    it('3. final bertukar gelanggang (Putra di gelanggang lain, Putri di gelanggang utama) dan juga serentak', () => runAsync(async () => {
        const [a, b] = await Promise.all([
            playBracketToEnd(kit, { categoryId: putraId, arena: 'other', maxPasses: 1 }),
            playBracketToEnd(kit, { categoryId: putriId, arena: 'main', maxPasses: 1 }),
        ]);
        expect(a).to.have.length(1);
        expect(b).to.have.length(1);
        played.push(...a, ...b);
        const all = await listMatches(kit);
        expect(all).to.have.length(6);
        expect(all.every((m) => m.status === 'finished')).to.eq(true);
        expect(all.find((m) => m.id === a[0].id).arena_id).to.eq(kit.crew('other').arenaId);
        expect(all.find((m) => m.id === b[0].id).arena_id).to.eq(kit.crew('main').arenaId);
    }, LONG));
    it('4. masing-masing kategori punya juara sendiri di bagannya', () => runAsync(async () => {
        for (const [cat, list] of [
            [putraId, putra],
            [putriId, putri],
        ]) {
            const slots = await bracketSlots(kit, cat);
            const top = Math.max(...slots.map((s) => s.round));
            const finalists = slots.filter((s) => s.round === top);
            const winner = finalists.find((s) => s.winner_id)?.winner_id;
            expect(list.some((e) => e.participantId === winner), 'juara berasal dari kategorinya').to.eq(true);
        }
    }, LONG));
    it('5. medali kontingen dijumlah dari KEDUA kategori: 2 emas, 2 perak, 4 perunggu di seluruh tim', () => runAsync(async () => {
        const everyone = [...putra, ...putri];
        const team = (pid) => everyone.find((e) => e.participantId === pid).team;
        const expected = {};
        const add = (pid, k) => {
            const t = team(pid);
            expected[t] ??= { gold: 0, silver: 0, bronze: 0 };
            expected[t][k] += 1;
        };
        for (const list of [putra, putri]) {
            const ids = new Set(list.map((e) => e.participantId));
            const mine = played.filter((p) => ids.has(p.winner));
            const top = Math.max(...mine.map((p) => p.round));
            const final = mine.find((p) => p.round === top);
            add(final.winner, 'gold');
            add(final.loser, 'silver');
            mine.filter((p) => p.round === top - 1).forEach((p) => add(p.loser, 'bronze'));
        }
        const rows = await must('medali', api('GET', `/competitions/${kit.competitionId}/report/medals`, kit.adminToken));
        const actual = {};
        rows.forEach((r) => (actual[r.team] = { gold: r.gold, silver: r.silver, bronze: r.bronze }));
        expect(actual).to.deep.eq(expected);
        const sum = (k) => rows.reduce((n, r) => n + r[k], 0);
        expect([sum('gold'), sum('silver'), sum('bronze')]).to.deep.eq([2, 2, 4]);
    }, LONG));
    it('6. bagan publik: kejuaraan aktif menampilkan dua bagan, masing-masing hanya berisi atletnya', () => runAsync(async () => {
        await must('aktifkan', api('PUT', `/competitions/${kit.competitionId}/activate`, kit.adminToken));
        const list = await api('GET', `/public/competitions/${kit.competitionId}/brackets`, null);
        expect(list.status).to.eq(200);
        expect(list.data).to.have.length(2);
        for (const [cat, mine, other] of [
            [putraId, putra, putri],
            [putriId, putri, putra],
        ]) {
            const res = await api('GET', `/public/competitions/${kit.competitionId}/brackets/${cat}`, null);
            expect(res.status).to.eq(200);
            const text = JSON.stringify(res.data);
            mine.forEach((e) => expect(text).to.include(e.name));
            other.forEach((e) => expect(text).to.not.include(e.name));
        }
    }, LONG));
});
