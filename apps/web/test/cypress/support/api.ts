/**
 * Klien API bertipe untuk setup data & pengujian level kontrak (cy.request, tanpa mock).
 *
 * Prinsip:
 *  - Setup data lewat API, bukan klik UI: cepat, deterministik, dan antar-spec tidak saling bergantung.
 *  - Setiap spec membuat KOMPETISI SENDIRI (nama unik + tanggal mulai tetap) sehingga tidak
 *    menyentuh data seed dan perhitungan umur selalu deterministik.
 *  - Semua helper memakai `failOnStatusCode: false` agar kasus negatif (4xx) bisa di-assert.
 */
import { ADMIN, KONTINGEN, type EmailAccount } from './accounts'

/** Tanggal mulai kejuaraan uji; dipakai backend sebagai tanggal acuan hitung golongan umur. */
export const REF_DATE = '2026-06-01'

export interface ApiError {
  code: string
  message: string
}
export interface ApiEnvelope<T> {
  success: boolean
  data: T
  error?: ApiError
}
export type ApiResponse<T> = Cypress.Response<ApiEnvelope<T>>

export interface Competition {
  id: string
  name: string
  status: string
  start_date: string
}
export interface Category {
  id: string
  competition_id: string
  name: string
  gender: string
  weight_class: string
  age_group: string
  preset_standard: string
  min_weight?: number | null
  max_weight?: number | null
  bracket_mode: string
}
export interface Participant {
  id: string
  athlete_id: string
  name: string
  gender: string
  registration_status: string
  category_id?: string
  category_name?: string
  age_group?: string
  birth_date?: string
  weight_class: string
  docs_verified?: Record<string, boolean>
  actual_weight?: number
  age_check?: 'missing_birth_date' | 'age_mismatch'
}
export interface Registration {
  athlete_id: string
  category_id: string
  competition_id: string
  contingent_id: string
  status: string
  weight_class: string
}
export interface VerifyResult {
  participant_id: string
  status: string
  reason: string
}
export interface BracketSlot {
  id: string
  round: number
  slot_index: number
  category_id?: string
  participant_id: string | null
  is_bye: boolean
  next_slot_id: string | null
}

export interface CategoryInput {
  name: string
  gender: 'male' | 'female' | 'putra' | 'putri'
  weight_class: string
  age_group?: string
  preset_standard?: string
  min_weight?: number
  max_weight?: number
  bracket_mode?: string
}
export interface RegisterInput {
  name?: string
  gender?: string
  team?: string
  birth_date?: string
  nik?: string
  category_id?: string
  weight_class?: string
  athlete_id?: string
}
export interface VerifyInput {
  status: string
  actual_weight?: number
  docs_verified?: Record<string, boolean>
}

/** Berkas minimum yang diwajibkan regulasi IPSI untuk semua atlet. */
export const DOCS_BASE = { identity: true, health: true } as const
/** Tambahan bagi putri Remaja & Dewasa. */
export const DOCS_FEMALE_SENIOR = { ...DOCS_BASE, pregnancy: true } as const

type Method = 'GET' | 'POST' | 'PUT' | 'DELETE'

/** Backend membatasi 60 req/detik (burst 100) per IP; 429 dicoba ulang dengan jeda agar suite tidak flaky. */
const MAX_RATE_LIMIT_RETRIES = 5
const RATE_LIMIT_BACKOFF_MS = 1000

/** Panggilan API mentah dengan token opsional; status 4xx/5xx dikembalikan apa adanya untuk di-assert. */
export function apiCall<T>(
  method: Method,
  path: string,
  token: string | null,
  body?: object,
  attempt = 0,
): Cypress.Chainable<ApiResponse<T>> {
  return cy
    .request({
      method,
      url: `/api/v1${path}`,
      headers: token ? { Authorization: `Bearer ${token}` } : {},
      body,
      failOnStatusCode: false,
    })
    .then((res) => {
      if (res.status === 429 && attempt < MAX_RATE_LIMIT_RETRIES) {
        cy.log(`429 rate limit pada ${method} ${path}; coba ulang #${attempt + 1}`)
        return cy
          .wait(RATE_LIMIT_BACKOFF_MS * (attempt + 1), { log: false })
          .then(() => apiCall<T>(method, path, token, body, attempt + 1))
      }
      return cy.wrap(res as ApiResponse<T>, { log: false })
    }) as Cypress.Chainable<ApiResponse<T>>
}

const call = apiCall

// ---------------------------------------------------------------------------------------------
// Autentikasi
// ---------------------------------------------------------------------------------------------

function loginRequest(account: EmailAccount): Cypress.Chainable<string> {
  const url = account.role === 'contingent' ? '/api/v1/auth/contingent/login' : '/api/v1/auth/login'
  return cy
    .request({ method: 'POST', url, body: { email: account.email, password: account.password } })
    .then((res) => {
      const token = (res.body as ApiEnvelope<{ token: string }>).data.token
      expect(token, `token ${account.email}`).to.be.a('string').and.have.length.greaterThan(0)
      return cy.task('tokenCache:set', { key: account.email, token }).then(() => token)
    })
}

/**
 * Token API untuk satu akun. Memakai cache lintas-spec (task Node) dan hanya login ulang bila token
 * belum ada atau sudah ditolak server, supaya batas 10 login/menit tidak terlampaui.
 */
export function apiToken(account: EmailAccount): Cypress.Chainable<string> {
  return cy.task<string | null>('tokenCache:get', account.email).then((cached) => {
    if (!cached) return loginRequest(account)
    return call('GET', '/competitions', cached).then((res) =>
      res.status === 200 ? cy.wrap(cached, { log: false }) : loginRequest(account),
    )
  })
}

export const adminToken = () => apiToken(ADMIN)
export const contingentToken = () => apiToken(KONTINGEN)

// ---------------------------------------------------------------------------------------------
// Kompetisi
// ---------------------------------------------------------------------------------------------

let seq = 0
/** Nama unik per pemanggilan, mudah dikenali & dibersihkan: `E2E <label> <waktu>-<urut>`. */
export function uniqueName(label: string): string {
  seq += 1
  return `E2E ${label} ${Date.now().toString(36)}-${seq}`
}

/** Buat kompetisi uji terisolasi (draft) dengan tanggal mulai tetap {@link REF_DATE}. */
export function createCompetition(token: string, label: string): Cypress.Chainable<Competition> {
  return call<Competition>('POST', '/competitions', token, {
    name: uniqueName(label),
    location: 'Cypress E2E',
    start_date: REF_DATE,
    end_date: '2026-06-03',
  }).then((res) => {
    expect(res.status, 'buat kompetisi').to.eq(201)
    const comp = res.body.data
    // bagan menolak kejuaraan tanpa gelanggang (bracket_service.generate), jadi sediakan satu di sini
    return call('POST', `/competitions/${comp.id}/arenas`, token, { name: 'Gelanggang Utama' }).then(
      (arena) => {
        expect(arena.status, 'buat gelanggang').to.eq(201)
        return comp
      },
    )
  })
}

/** Batalkan kompetisi uji agar tidak mengotori daftar (best effort, tidak menggagalkan tes). */
export function cancelCompetition(token: string, competitionId: string) {
  return call('POST', `/competitions/${competitionId}/cancel`, token)
}

// ---------------------------------------------------------------------------------------------
// Kategori
// ---------------------------------------------------------------------------------------------

export const createCategoryRaw = (token: string | null, compId: string, input: object) =>
  call<Category>('POST', `/competitions/${compId}/categories`, token, input)

/** Buat kategori dan pastikan berhasil (201). */
export function createCategory(
  token: string,
  compId: string,
  input: CategoryInput,
): Cypress.Chainable<Category> {
  return createCategoryRaw(token, compId, input).then((res) => {
    expect(res.status, `buat kategori "${input.name}"`).to.eq(201)
    return res.body.data
  })
}

export const listCategories = (token: string | null, compId: string) =>
  call<Category[]>('GET', `/competitions/${compId}/categories`, token)

export const getCategory = (token: string | null, compId: string, id: string) =>
  call<Category>('GET', `/competitions/${compId}/categories/${id}`, token)

export const resetCategories = (token: string | null, compId: string) =>
  call<{ deleted: number }>('DELETE', `/competitions/${compId}/categories`, token)

// ---------------------------------------------------------------------------------------------
// Atlet & peserta
// ---------------------------------------------------------------------------------------------

export const registerAthleteRaw = (token: string | null, compId: string, input: RegisterInput) =>
  call<Registration>('POST', `/competitions/${compId}/athletes`, token, input)

/** Daftarkan atlet baru ke kategori; mengembalikan {athlete_id, participantId}. */
export function registerAthlete(
  token: string,
  compId: string,
  input: RegisterInput,
): Cypress.Chainable<{ athleteId: string; participantId: string; status: string }> {
  return registerAthleteRaw(token, compId, input).then((res) => {
    expect(res.status, `daftarkan atlet "${input.name ?? input.athlete_id}"`).to.eq(201)
    const athleteId = res.body.data.athlete_id
    return listParticipants(token, compId).then((list) => {
      const found = list.body.data.find((p) => p.athlete_id === athleteId)
      expect(found, 'peserta muncul di daftar peserta').to.not.equal(undefined)
      return { athleteId, participantId: found!.id, status: res.body.data.status }
    })
  })
}

export const updateAthleteProfile = (
  token: string | null,
  athleteId: string,
  body: { birth_date?: string; nik?: string },
) =>
  apiCall<{ id: string; birth_date?: string; nik?: string }>(
    'PUT',
    `/athletes/${athleteId}`,
    token,
    body,
  )

export const listParticipants = (token: string | null, compId: string, status?: string) =>
  call<Participant[]>(
    'GET',
    `/competitions/${compId}/participants${status ? `?status=${status}` : ''}`,
    token,
  )

export const verifyParticipantRaw = (
  token: string | null,
  compId: string,
  participantId: string,
  input: VerifyInput,
) =>
  call<VerifyResult>(
    'PUT',
    `/competitions/${compId}/participants/${participantId}/verify`,
    token,
    input,
  )

// ---------------------------------------------------------------------------------------------
// Bagan
// ---------------------------------------------------------------------------------------------

export const generateBracketRaw = (token: string | null, compId: string, input: object) =>
  call<BracketSlot[]>('POST', `/competitions/${compId}/brackets/generate`, token, input)

export const getBracket = (token: string | null, compId: string, categoryId: string) =>
  call<BracketSlot[]>('GET', `/competitions/${compId}/brackets?category_id=${categoryId}`, token)

// ---------------------------------------------------------------------------------------------
// Utilitas data uji
// ---------------------------------------------------------------------------------------------

function toIso(d: Date): string {
  return d.toISOString().slice(0, 10)
}

/**
 * Tanggal lahir yang menghasilkan usia TEPAT `age` tahun penuh pada {@link REF_DATE}.
 *  - `edge: 'birthday'`   : ulang tahun ke-`age` jatuh PERSIS di hari acuan (sudah genap `age`).
 *  - `edge: 'eve'`        : sehari sebelum ulang tahun ke-`age`+1 (masih `age`, batas atas golongan).
 *  - `edge: 'mid'` (def.) : pertengahan tahun, jauh dari batas.
 */
export function birthDateForAge(age: number, edge: 'mid' | 'birthday' | 'eve' = 'mid'): string {
  const ref = new Date(`${REF_DATE}T00:00:00Z`)
  if (edge === 'birthday') {
    return toIso(
      new Date(Date.UTC(ref.getUTCFullYear() - age, ref.getUTCMonth(), ref.getUTCDate())),
    )
  }
  if (edge === 'eve') {
    // lahir (age+1) tahun lalu + 1 hari => besok genap age+1, hari ini masih age
    const d = new Date(
      Date.UTC(ref.getUTCFullYear() - (age + 1), ref.getUTCMonth(), ref.getUTCDate() + 1),
    )
    return toIso(d)
  }
  // 15 Januari: ulang tahun tahun ini sudah lewat pada tanggal acuan (1 Juni)
  return toIso(new Date(Date.UTC(ref.getUTCFullYear() - age, 0, 15)))
}

// ---------------------------------------------------------------------------------------------
// Kejuaraan aktif & bagan siap pakai
// ---------------------------------------------------------------------------------------------

export const activateCompetition = (token: string, compId: string) =>
  apiCall<{ status: string }>('PUT', `/competitions/${compId}/activate`, token)

export const pauseCompetition = (token: string, compId: string) =>
  apiCall<{ status: string }>('POST', `/competitions/${compId}/pause`, token)

export interface SeedBracketInput {
  name: string
  gender?: 'male' | 'female'
  ageGroup?: string
  /** Usia atlet (tahun) pada REF_DATE; harus berada dalam golongan umur. */
  age?: number
  athletes?: number
  /** Awalan nama atlet; hasil: `<prefix> 1`, `<prefix> 2`, ... dengan kontingen `Klub <prefix> n`. */
  prefix?: string
}

export interface SeededBracket {
  category: Category
  /** Nama atlet yang masuk bagan (untuk di-assert di UI/API). */
  athleteNames: string[]
  /** NIK & tanggal lahir yang TIDAK boleh muncul di respons publik. */
  secrets: string[]
}

/**
 * Buat satu kategori, daftarkan + verifikasi `athletes` atlet, lalu generate bagannya.
 * Semua atlet diberi NIK dan tanggal lahir khas supaya kebocorannya mudah dideteksi.
 */
export function seedBracket(
  token: string,
  compId: string,
  input: SeedBracketInput,
): Cypress.Chainable<SeededBracket> {
  const gender = input.gender ?? 'male'
  const ageGroup = input.ageGroup ?? 'dewasa'
  const age = input.age ?? 25
  const count = input.athletes ?? 3
  const prefix = input.prefix ?? input.name
  const docs =
    gender === 'female' && (ageGroup === 'remaja' || ageGroup === 'dewasa')
      ? DOCS_FEMALE_SENIOR
      : DOCS_BASE

  seq += 1
  return createCategory(token, compId, {
    name: input.name,
    gender,
    weight_class: `B${Date.now().toString(36)}${seq}`,
    age_group: ageGroup,
  }).then((category) => {
    const athleteNames: string[] = []
    const secrets: string[] = []
    for (let i = 1; i <= count; i += 1) {
      const name = `${prefix} ${i}`
      const nik = `99${String(seq).padStart(4, '0')}${String(i).padStart(10, '0')}`
      const birth = birthDateForAge(age)
      athleteNames.push(name)
      secrets.push(nik)
      registerAthlete(token, compId, {
        name,
        gender,
        team: `Klub ${prefix} ${i}`,
        nik,
        birth_date: birth,
        category_id: category.id,
      }).then(({ participantId }) =>
        verifyParticipantRaw(token, compId, participantId, {
          status: 'verified',
          docs_verified: { ...docs },
        })
          .its('body.data.status')
          .should('eq', 'verified'),
      )
    }
    return generateBracketRaw(token, compId, { category_id: category.id }).then((res) => {
      expect(res.status, `generate bagan "${input.name}"`).to.eq(201)
      secrets.push(birthDateForAge(age))
      return { category, athleteNames, secrets }
    })
  })
}
