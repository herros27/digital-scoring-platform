import { adminToken } from '../../support/match/auth.ts';
import { bracketSlots, enrollAndVerify, generateBracket, listMatches, playMatch, } from '../../support/match/competition.ts';
import { api, must } from '../../support/match/http.ts';
import { buildKit } from '../../support/match/kit.ts';
import { runAsync } from '../../support/match/run.ts';
const LONG = 300_000;
/**
 * Bagan 8 peserta (perempat final → semifinal → final) dimainkan sampai juara. Memastikan partai berikutnya
 * terbentuk otomatis di tiap babak, tidak ada partai yang terlewat, dan medali hanya untuk final dan semifinal:
 * pemenang/pecundang perempat final TIDAK mendapat medali. Delapan kontingen berbeda agar medali tiap tim jelas.
 */
describe('Kompetisi 8 peserta: perempat final sampai juara', { retries: 0 }, () => {
    let kit;
    let entrants = [];
    const played = [];
    before(() => runAsync(async () => {
        kit = await buildKit(await adminToken(), {
            label: 'Kompetisi 8',
            roundSeconds: 20,
            breakSeconds: 2,
            totalRounds: 2,
        });
    }));
    after(() => runAsync(async () => kit?.teardown()));
    it('1. delapan atlet dari delapan kontingen terverifikasi dan bagan terbentuk dengan 4 perempat final', () => runAsync(async () => {
        entrants = await enrollAndVerify(kit, Array.from({ length: 8 }, (_, i) => ({
            name: `Pesilat ${i + 1}`,
            team: `Kontingen ${String.fromCharCode(65 + i)}`,
        })));
        expect((await generateBracket(kit)).status).to.eq(201);
        const matches = await listMatches(kit);
        expect(matches, 'hanya perempat final yang dibuat di awal').to.have.length(4);
        const used = matches.flatMap((m) => [m.athlete_red_id, m.athlete_blue_id]);
        expect(used).to.have.members(entrants.map((e) => e.participantId));
    }, LONG));
    it('2. seluruh partai dimainkan babak demi babak (merah selalu unggul): 4 perempat → 2 semi → 1 final', () => runAsync(async () => {
        const slots = await bracketSlots(kit);
        const roundOf = (m) => slots.find((s) => s.id === m.bracket_slot_id).round;
        const done = new Set();
        for (let guard = 0; guard < 10; guard += 1) {
            const idle = (await listMatches(kit)).filter((m) => m.status === 'idle' && !done.has(m.id));
            if (idle.length === 0)
                break;
            for (const m of idle) {
                const { result } = await playMatch(kit, m.id, [{ target: 'red', type: 'knockdown' }]);
                expect(result.corner).to.eq('red');
                done.add(m.id);
                played.push({
                    round: roundOf(m),
                    winner: m.athlete_red_id,
                    loser: m.athlete_blue_id,
                });
            }
        }
        expect(played, '4 + 2 + 1 partai').to.have.length(7);
        const perRound = (r) => played.filter((p) => p.round === r).length;
        expect([perRound(1), perRound(2), perRound(3)]).to.deep.eq([4, 2, 1]);
        const all = await listMatches(kit);
        expect(all.every((m) => m.status === 'finished')).to.eq(true);
    }, LONG));
    it('3. juara tercatat di bagan akhir', () => runAsync(async () => {
        const champion = played.find((p) => p.round === 3).winner;
        const slots = await bracketSlots(kit);
        const finalRound = Math.max(...slots.map((s) => s.round));
        expect(slots.filter((s) => s.round === finalRound).some((s) => s.winner_id === champion)).to.eq(true);
    }, LONG));
    it('4. medali: 1 emas (juara), 1 perak (finalis), 2 perunggu (kalah semifinal); babak perempat tanpa medali', () => runAsync(async () => {
        const team = (pid) => entrants.find((e) => e.participantId === pid).team;
        const final = played.find((p) => p.round === 3);
        const semiLosers = played.filter((p) => p.round === 2).map((p) => team(p.loser));
        const quarterTeams = played
            .filter((p) => p.round === 1)
            .flatMap((p) => [team(p.winner), team(p.loser)]);
        const rows = await must('medali', api('GET', `/competitions/${kit.competitionId}/report/medals`, kit.adminToken));
        const by = {};
        rows.forEach((r) => (by[r.team] = r));
        expect(by[team(final.winner)]).to.include({ gold: 1 });
        expect(by[team(final.loser)]).to.include({ silver: 1 });
        semiLosers.forEach((t) => expect(by[t]).to.include({ bronze: 1 }));
        const sum = (k) => rows.reduce((n, r) => n + r[k], 0);
        expect([sum('gold'), sum('silver'), sum('bronze'), sum('total')]).to.deep.eq([1, 1, 2, 4]);
        // tim yang hanya bermain di perempat final dan tidak masuk final/semifinal tidak muncul sama sekali
        const medalists = new Set([team(final.winner), team(final.loser), ...semiLosers]);
        quarterTeams
            .filter((t) => !medalists.has(t))
            .forEach((t) => expect(by[t], `${t} tanpa medali`).to.eq(undefined));
    }, LONG));
});
