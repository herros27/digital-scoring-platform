/**
 * Runner headless untuk spec alur pertandingan (test/cypress/e2e/match/*.cy.ts yang tidak berawalan `ui-`).
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
import './load-env.mjs'

import { readdirSync } from 'node:fs'
import { dirname, join } from 'node:path'
import { fileURLToPath, pathToFileURL } from 'node:url'
import { expect } from 'chai'

type Fn = () => unknown | Promise<unknown>
interface Suite {
  title: string
  parent: Suite | null
  before: Fn[]
  after: Fn[]
  beforeEach: Fn[]
  afterEach: Fn[]
  children: Array<Suite | Test>
  skip: boolean
}
interface Test {
  title: string
  fn: Fn
  skip: boolean
  only: boolean
  suite: Suite
}

const root: Suite = {
  title: '',
  parent: null,
  before: [],
  after: [],
  beforeEach: [],
  afterEach: [],
  children: [],
  skip: false,
}
let current = root
let hasOnly = false

function define(title: string, body: () => void, skip = false) {
  const suite: Suite = {
    title,
    parent: current,
    before: [],
    after: [],
    beforeEach: [],
    afterEach: [],
    children: [],
    skip,
  }
  current.children.push(suite)
  const prev = current
  current = suite
  body()
  current = prev
}
// describe(title, fn) atau describe(title, opsi, fn) seperti Mocha/Cypress (opsi mis. { retries: 0 } diabaikan di sini)
type Body = () => void
const pick = (a: Body | object, b?: Body): Body =>
  typeof a === 'function' ? (a as Body) : (b as Body)
const describeFn = Object.assign((t: string, a: Body | object, b?: Body) => define(t, pick(a, b)), {
  skip: (t: string, a: Body | object, b?: Body) => define(t, pick(a, b), true),
})
function itFn(title: string, fn: Fn) {
  current.children.push({ title, fn, skip: false, only: false, suite: current })
}
itFn.skip = (title: string, fn: Fn) =>
  current.children.push({ title, fn, skip: true, only: false, suite: current })
itFn.only = (title: string, fn: Fn) => {
  hasOnly = true
  current.children.push({ title, fn, skip: false, only: true, suite: current })
}
Object.assign(globalThis, {
  describe: describeFn,
  context: describeFn,
  it: itFn,
  specify: itFn,
  before: (fn: Fn) => current.before.push(fn),
  after: (fn: Fn) => current.after.push(fn),
  beforeEach: (fn: Fn) => current.beforeEach.push(fn),
  afterEach: (fn: Fn) => current.afterEach.push(fn),
  expect,
})

const TEST_TIMEOUT_MS = 150_000
function withTimeout<T>(p: Promise<T>, what: string): Promise<T> {
  let t: ReturnType<typeof setTimeout>
  return Promise.race([
    p,
    new Promise<never>((_, rej) => {
      t = setTimeout(
        () => rej(new Error(`timeout ${TEST_TIMEOUT_MS / 1000}s: ${what}`)),
        TEST_TIMEOUT_MS,
      )
    }),
  ]).finally(() => clearTimeout(t)) as Promise<T>
}
const run = async (fn: Fn, what: string) => {
  await withTimeout(Promise.resolve().then(fn), what)
}

const chain = (s: Suite): Suite[] => (s.parent ? [...chain(s.parent), s] : [s])
const path = (s: Suite) =>
  chain(s)
    .map((x) => x.title)
    .filter(Boolean)
    .join(' › ')

const results = { passed: 0, failed: 0, skipped: 0 }
const failures: string[] = []

async function runSuite(suite: Suite): Promise<void> {
  const skipped = suite.skip
  let hookFailed = false
  if (!skipped) {
    for (const h of suite.before) {
      try {
        await run(h, `before (${path(suite)})`)
      } catch (e) {
        hookFailed = true
        failures.push(`before-hook "${path(suite)}": ${(e as Error).message}`)
        console.log(`  ✗ [before-hook] ${path(suite)}: ${(e as Error).message}`)
        break
      }
    }
  }
  for (const child of suite.children) {
    if ('children' in child) {
      await runSuite(child)
      continue
    }
    const name = `${path(suite)} › ${child.title}`
    if (skipped || child.skip || hookFailed || (hasOnly && !child.only)) {
      results.skipped += 1
      console.log(`  - ${name} (dilewati)`)
      continue
    }
    const started = Date.now()
    try {
      for (const s of chain(suite))
        for (const h of s.beforeEach) await run(h, `beforeEach (${path(s)})`)
      await run(child.fn, name)
      results.passed += 1
      console.log(`  ✓ ${name} (${((Date.now() - started) / 1000).toFixed(1)}s)`)
    } catch (e) {
      results.failed += 1
      const msg = (e as Error).stack?.split('\n').slice(0, 4).join('\n') ?? String(e)
      failures.push(`${name}\n    ${msg}`)
      console.log(`  ✗ ${name}\n      ${(e as Error).message}`)
    } finally {
      for (const s of [...chain(suite)].reverse()) {
        for (const h of s.afterEach) {
          try {
            await run(h, `afterEach (${path(s)})`)
          } catch (e) {
            console.log(`  ! afterEach gagal: ${(e as Error).message}`)
          }
        }
      }
    }
  }
  if (!skipped) {
    for (const h of suite.after) {
      try {
        await run(h, `after (${path(suite)})`)
      } catch (e) {
        console.log(`  ! after-hook gagal (${path(suite)}): ${(e as Error).message}`)
      }
    }
  }
}

const here = dirname(fileURLToPath(import.meta.url))
const specDir = join(here, '..', 'e2e', 'match')
const filter = process.argv.slice(2).filter((a) => !a.startsWith('-'))
const files = readdirSync(specDir)
  .filter((f) => f.endsWith('.cy.ts') && !f.startsWith('ui-'))
  .filter((f) => filter.length === 0 || filter.some((x) => f.includes(x)))
  .sort()

if (files.length === 0) {
  console.error('Tidak ada spec yang cocok.')
  process.exit(2)
}

// Hook restart server untuk spec pemulihan: dipasang hanya bila E2E_RESTART_CMD diberikan lingkungan
if (process.env.E2E_RESTART_CMD) {
  const { execSync } = await import('node:child_process')
  ;(globalThis as { __e2eRestart?: () => Promise<void> }).__e2eRestart = async () => {
    execSync(process.env.E2E_RESTART_CMD as string, { stdio: 'inherit', timeout: 120_000 })
  }
}

const base = process.env.E2E_BASE_URL ?? 'http://127.0.0.1:5173'
const health = await fetch(`${base}/api/v1/health`).catch(() => null)
if (!health?.ok) {
  console.error(
    `Backend/Vite tidak merespons di ${base}. Jalankan backend (go run ./cmd/server) dan 'npm run dev'.`,
  )
  process.exit(2)
}

const t0 = Date.now()
for (const file of files) {
  console.log(`\n▶ ${file}`)
  current = root
  root.children = []
  await import(pathToFileURL(join(specDir, file)).href)
  await runSuite(root)
}
console.log(
  `\n${results.passed} lulus, ${results.failed} gagal, ${results.skipped} dilewati — ${((Date.now() - t0) / 1000).toFixed(0)} dtk`,
)
if (failures.length) {
  console.log('\nKEGAGALAN:')
  failures.forEach((f) => console.log(' • ' + f))
}
process.exit(results.failed > 0 ? 1 : 0)
