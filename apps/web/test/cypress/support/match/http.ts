/**
 * Klien HTTP berbasis fetch untuk skenario pertandingan. Sengaja tidak memakai `cy.request` agar modul
 * ini berjalan sama persis di Cypress (spec window) dan di runner Node headless (scripts/run-match-flow.ts).
 * Semua permintaan lewat origin yang sama dengan aplikasi (proxy Vite `/api`), seperti pengguna sungguhan.
 */

/**
 * Muatan JSON dari server (dinamis). Dibiarkan longgar di batas I/O ini dan dipersempit oleh asersi di tiap tes;
 * satu-satunya tempat `any` diizinkan di suite ini.
 */
// eslint-disable-next-line @typescript-eslint/no-explicit-any
export type Json = any

export interface ApiError {
  code: string
  message: string
}

export interface ApiResult<T = Json> {
  status: number
  /** `data` pada amplop sukses; undefined bila gagal */
  data: T
  error?: ApiError
  body: Json
}

type Method = 'GET' | 'POST' | 'PUT' | 'DELETE'

/** Origin aplikasi: spec window di Cypress, atau E2E_BASE_URL (bawaan dev server) di Node. */
export function appOrigin(): string {
  const loc = (globalThis as { location?: { origin?: string } }).location
  if (loc?.origin && loc.origin !== 'null') return loc.origin
  const env = (globalThis as { process?: { env?: Record<string, string | undefined> } }).process
    ?.env
  return env?.E2E_BASE_URL ?? 'http://127.0.0.1:5173'
}

export const wsOrigin = (): string => appOrigin().replace(/^http/, 'ws')

export const sleep = (ms: number) => new Promise<void>((resolve) => setTimeout(resolve, ms))

const MAX_429_RETRIES = 8

/**
 * Panggilan API. Status 4xx/5xx dikembalikan apa adanya untuk di-assert. 429 (rate limit login 10/menit
 * dan 60 req/dtk) dicoba ulang dengan jeda bertahap supaya suite tidak flaky.
 */
export async function api<T = Json>(
  method: Method,
  path: string,
  token: string | null,
  body?: object,
  attempt = 0,
): Promise<ApiResult<T>> {
  const res = await fetch(`${appOrigin()}/api/v1${path}`, {
    method,
    headers: {
      'Content-Type': 'application/json',
      ...(token ? { Authorization: `Bearer ${token}` } : {}),
    },
    body: body === undefined ? undefined : JSON.stringify(body),
  })
  if (res.status === 429 && attempt < MAX_429_RETRIES) {
    await sleep(1500 * (attempt + 1))
    return api<T>(method, path, token, body, attempt + 1)
  }
  let json: Json = null
  try {
    json = await res.json()
  } catch {
    // respons tanpa badan JSON
  }
  return { status: res.status, data: json?.data, error: json?.error, body: json }
}

/** Panggilan yang wajib sukses (2xx); selain itu melempar galat berisi konteks. */
export async function must<T = Json>(
  what: string,
  call: Promise<ApiResult<T>>,
  expected: number | number[] = [200, 201],
): Promise<T> {
  const res = await call
  const ok = Array.isArray(expected) ? expected : [expected]
  if (!ok.includes(res.status)) {
    throw new Error(
      `${what}: HTTP ${res.status} ${res.error?.code ?? ''} ${res.error?.message ?? ''}`.trim(),
    )
  }
  return res.data
}
