/**
 * Runner headless untuk spec alur pertandingan (cypress/e2e/match/*.cy.ts yang tidak berawalan `ui-`).
 *
 * Spec tersebut hanya memakai `runAsync` + modul `support/match/*` (fetch + WebSocket), sehingga dapat
 * dijalankan TANPA browser/binary Cypress terhadap backend dan Vite dev server yang sama:
 *
 *   npm run test:match-flow            # semua spec alur
 *   npm run test:match-flow -- 02      # hanya spec yang namanya memuat "02"
 *
 * Ini pelengkap, bukan pengganti, `cypress run`: spec UI (`ui-*.cy.ts`) hanya berjalan di Cypress.
 * Runner mengimplementasikan subset Mocha yang dipakai spec: describe/it/before/after/beforeEach/afterEach
 * (+ .skip/.only), dengan `expect` dari chai (sama seperti `expect` global Cypress).
 */
import './load-env.mjs';
import { readdirSync } from 'node:fs';
import { dirname, join } from 'node:path';
import { fileURLToPath, pathToFileURL } from 'node:url';
import { expect } from 'chai';
const root = {
    title: '',
    parent: null,
    before: [],
    after: [],
    beforeEach: [],
    afterEach: [],
    children: [],
    skip: false,
};
let current = root;
let hasOnly = false;
function define(title, body, skip = false) {
    const suite = {
        title,
        parent: current,
        before: [],
        after: [],
        beforeEach: [],
        afterEach: [],
        children: [],
        skip,
    };
    current.children.push(suite);
    const prev = current;
    current = suite;
    body();
    current = prev;
}
const pick = (a, b) => typeof a === 'function' ? a : b;
const describeFn = Object.assign((t, a, b) => define(t, pick(a, b)), {
    skip: (t, a, b) => define(t, pick(a, b), true),
});
function itFn(title, fn) {
    current.children.push({ title, fn, skip: false, only: false, suite: current });
}
itFn.skip = (title, fn) => current.children.push({ title, fn, skip: true, only: false, suite: current });
itFn.only = (title, fn) => {
    hasOnly = true;
    current.children.push({ title, fn, skip: false, only: true, suite: current });
};
Object.assign(globalThis, {
    describe: describeFn,
    context: describeFn,
    it: itFn,
    specify: itFn,
    before: (fn) => current.before.push(fn),
    after: (fn) => current.after.push(fn),
    beforeEach: (fn) => current.beforeEach.push(fn),
    afterEach: (fn) => current.afterEach.push(fn),
    expect,
});
const TEST_TIMEOUT_MS = 150_000;
function withTimeout(p, what) {
    let t;
    return Promise.race([
        p,
        new Promise((_, rej) => {
            t = setTimeout(() => rej(new Error(`timeout ${TEST_TIMEOUT_MS / 1000}s: ${what}`)), TEST_TIMEOUT_MS);
        }),
    ]).finally(() => clearTimeout(t));
}
const run = async (fn, what) => {
    await withTimeout(Promise.resolve().then(fn), what);
};
const chain = (s) => (s.parent ? [...chain(s.parent), s] : [s]);
const path = (s) => chain(s)
    .map((x) => x.title)
    .filter(Boolean)
    .join(' › ');
const results = { passed: 0, failed: 0, skipped: 0 };
const failures = [];
async function runSuite(suite) {
    const skipped = suite.skip;
    let hookFailed = false;
    if (!skipped) {
        for (const h of suite.before) {
            try {
                await run(h, `before (${path(suite)})`);
            }
            catch (e) {
                hookFailed = true;
                failures.push(`before-hook "${path(suite)}": ${e.message}`);
                console.log(`  ✗ [before-hook] ${path(suite)}: ${e.message}`);
                break;
            }
        }
    }
    for (const child of suite.children) {
        if ('children' in child) {
            await runSuite(child);
            continue;
        }
        const name = `${path(suite)} › ${child.title}`;
        if (skipped || child.skip || hookFailed || (hasOnly && !child.only)) {
            results.skipped += 1;
            console.log(`  - ${name} (dilewati)`);
            continue;
        }
        const started = Date.now();
        try {
            for (const s of chain(suite))
                for (const h of s.beforeEach)
                    await run(h, `beforeEach (${path(s)})`);
            await run(child.fn, name);
            results.passed += 1;
            console.log(`  ✓ ${name} (${((Date.now() - started) / 1000).toFixed(1)}s)`);
        }
        catch (e) {
            results.failed += 1;
            const msg = e.stack?.split('\n').slice(0, 4).join('\n') ?? String(e);
            failures.push(`${name}\n    ${msg}`);
            console.log(`  ✗ ${name}\n      ${e.message}`);
        }
        finally {
            for (const s of [...chain(suite)].reverse()) {
                for (const h of s.afterEach) {
                    try {
                        await run(h, `afterEach (${path(s)})`);
                    }
                    catch (e) {
                        console.log(`  ! afterEach gagal: ${e.message}`);
                    }
                }
            }
        }
    }
    if (!skipped) {
        for (const h of suite.after) {
            try {
                await run(h, `after (${path(suite)})`);
            }
            catch (e) {
                console.log(`  ! after-hook gagal (${path(suite)}): ${e.message}`);
            }
        }
    }
}
const here = dirname(fileURLToPath(import.meta.url));
const specDir = join(here, '..', 'e2e', 'match');
const filter = process.argv.slice(2).filter((a) => !a.startsWith('-'));
const files = readdirSync(specDir)
    .filter((f) => f.endsWith('.cy.ts') && !f.startsWith('ui-'))
    .filter((f) => filter.length === 0 || filter.some((x) => f.includes(x)))
    .sort();
if (files.length === 0) {
    console.error('Tidak ada spec yang cocok.');
    process.exit(2);
}
// Hook restart server untuk spec pemulihan: dipasang hanya bila E2E_RESTART_CMD diberikan lingkungan
if (process.env.E2E_RESTART_CMD) {
    const { execSync } = await import('node:child_process');
    globalThis.__e2eRestart = async () => {
        execSync(process.env.E2E_RESTART_CMD, { stdio: 'inherit', timeout: 120_000 });
    };
}
const base = process.env.E2E_BASE_URL ?? 'http://127.0.0.1:5173';
const health = await fetch(`${base}/api/v1/health`).catch(() => null);
if (!health?.ok) {
    console.error(`Backend/Vite tidak merespons di ${base}. Jalankan backend (go run ./cmd/server) dan 'npm run dev'.`);
    process.exit(2);
}
const t0 = Date.now();
for (const file of files) {
    console.log(`\n▶ ${file}`);
    current = root;
    root.children = [];
    await import(pathToFileURL(join(specDir, file)).href);
    await runSuite(root);
}
console.log(`\n${results.passed} lulus, ${results.failed} gagal, ${results.skipped} dilewati — ${((Date.now() - t0) / 1000).toFixed(0)} dtk`);
if (failures.length) {
    console.log('\nKEGAGALAN:');
    failures.forEach((f) => console.log(' • ' + f));
}
process.exit(results.failed > 0 ? 1 : 0);
