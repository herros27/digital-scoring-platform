import { api, must, sleep, type ApiResult, type Json } from './http.ts'
import { ArenaSocket } from './socket.ts'

/** Sesi perangkat hasil login PIN: token + konteks gelanggang (bentuk yang sama dengan sesi aplikasi). */
export interface Session {
  token: string
  role: string
  seat?: number
  arenaId: string
  competitionId: string
}

export interface KitOptions {
  /** Label unik pada nama kejuaraan/ruleset (memudahkan pembersihan). */
  label: string
  roundSeconds?: number
  breakSeconds?: number
  totalRounds?: number
  /** Jumlah juri yang dibuatkan PIN (kursi 1..n); bawaan 3 sesuai konsensus 2-dari-3. */
  judges?: number
  /** Aturan konsensus ruleset; bawaan 2-dari-3 dalam 3 dtk. Jumlah PIN juri mengikuti `judges` di atas. */
  majority?: number
  consensusWindowMs?: number
  /** Buat juga PIN wasit untuk gelanggang lain (hanya dipakai uji otorisasi lintas gelanggang). */
  outsideWasit?: boolean
  /** Lengkapi gelanggang kedua dengan wasit dan semua juri (kursi 1..judges) agar partai dapat dimainkan di sana. */
  otherArenaCrew?: boolean
}

export interface MatchRef {
  id: string
  redId: string
  blueId: string
  redName: string
  blueName: string
}

export interface MatchKit {
  adminToken: string
  competitionId: string
  arenaId: string
  /** Kategori resmi tunggal kejuaraan uji (Dewasa Putra A) tempat bagan dibentuk. */
  categoryId: string
  /** Gelanggang kedua di kejuaraan yang sama (untuk uji isolasi antar gelanggang). */
  otherArenaId: string
  rulesetId: string
  wasit: Session
  judges: Session[]
  /** Juri yang PIN-nya terikat ke gelanggang LAIN. */
  outsideJudge: Session
  /** Wasit yang PIN-nya terikat ke gelanggang LAIN. */
  outsideWasit: Session
  roundMs: number
  breakMs: number
  totalRounds: number
  /** Membuat partai idle baru (dua peserta terverifikasi) untuk gelanggang utama. */
  createMatch(): Promise<MatchRef>
  /** Awak (wasit + juri) satu gelanggang. `other` hanya lengkap bila opsi otherArenaCrew diberikan. */
  crew(arena: ArenaKey): Crew
  /** Membuat kategori tambahan di kejuaraan kit; mengembalikan id kategori. */
  createCategory(def: CategoryDef): Promise<string>
  /** Memuat partai ke gelanggang sebagai wasit (ruleset kit); bawaan gelanggang utama. */
  loadMatch(match: MatchRef, token?: string, arena?: ArenaKey): Promise<ApiResult>
  /** Perintah kontrol partai via REST sebagai wasit (atau token lain). */
  control(match: MatchRef, action: string, body?: object, token?: string): Promise<ApiResult>
  /** Membuka soket wasit + semua juri di gelanggang utama. */
  openDevices(opts?: { skipSeats?: number[]; arena?: ArenaKey }): Promise<Devices>
  /** Membatalkan semua partai yang dibuat kit lalu kejuaraan dan ruleset (aman dipanggil berulang). */
  teardown(): Promise<void>
}

export type ArenaKey = 'main' | 'other'

export interface Crew {
  arenaId: string
  wasit: Session
  judges: Session[]
}

export interface CategoryDef {
  name: string
  gender: 'male' | 'female'
  weight_class: string
  age_group: string
}

export interface Devices {
  wasit: ArenaSocket
  judges: ArenaSocket[]
  all: ArenaSocket[]
  closeAll(): Promise<void>
}

let counter = 0
const uid = () => `${Date.now().toString(36)}${(counter += 1)}`

/** Skema skor ruleset uji: durasi singkat, tiga teknik, sanksi resmi, konsensus 2-dari-3 dalam 3 dtk. */
function scoring(
  o: Required<
    Pick<
      KitOptions,
      'roundSeconds' | 'breakSeconds' | 'totalRounds' | 'judges' | 'majority' | 'consensusWindowMs'
    >
  >,
) {
  return {
    technique_points: 1,
    knockdown_points: 2,
    fall_points: 3,
    max_penalties: 3,
    penalty_threshold: 1,
    round_duration: o.roundSeconds,
    break_duration: o.breakSeconds,
    total_rounds: o.totalRounds,
    techniques: [
      { type: 'technique', label: 'Pukulan', points: 1 },
      { type: 'knockdown', label: 'Tendangan', points: 2 },
      { type: 'fall', label: 'Jatuhan', points: 3 },
    ],
    penalties: [
      { type: 'binaan_1', label: 'Binaan 1', points: 0 },
      { type: 'teguran_1', label: 'Teguran 1', points: 1 },
      { type: 'teguran_2', label: 'Teguran 2', points: 2 },
      { type: 'peringatan_3', label: 'Peringatan 3', points: 0, action: 'disqualify' },
    ],
    consensus: { judges: o.judges, majority: o.majority, window_ms: o.consensusWindowMs },
  }
}

async function pinSession(
  adminToken: string,
  competitionId: string,
  arenaId: string,
  role: 'wasit' | 'juri',
  seat: number,
): Promise<Session> {
  const created = await must(
    `buat PIN ${role}${seat || ''}`,
    api('POST', '/auth/create-pin', adminToken, {
      competition_id: competitionId,
      arena_id: arenaId,
      role,
      seat_number: seat,
      ttl_hours: 2,
    }),
  )
  const login = await must(
    `login PIN ${role}${seat || ''}`,
    api('POST', '/auth/pin', null, { pin: created.pin }),
  )
  return {
    token: login.token,
    role: login.role,
    seat: login.seat_number,
    arenaId: login.arena_id,
    competitionId: login.competition_id,
  }
}

/**
 * Menyiapkan satu kejuaraan uji lengkap dengan DATA NYATA di backend: gelanggang (utama + lain), ruleset
 * cepat, dua peserta terverifikasi, PIN + login untuk wasit dan tiap juri, serta juri gelanggang lain.
 * Tidak ada mock: semua lewat API publik yang sama dengan aplikasi.
 */
export async function buildKit(adminToken: string, options: KitOptions): Promise<MatchKit> {
  const opt = {
    roundSeconds: options.roundSeconds ?? 8,
    breakSeconds: options.breakSeconds ?? 3,
    totalRounds: options.totalRounds ?? 2,
    judges: options.judges ?? 3,
    majority: options.majority ?? 2,
    consensusWindowMs: options.consensusWindowMs ?? 3000,
  }
  const tag = `E2E ${options.label} ${uid()}`

  const comp = await must(
    'buat kejuaraan',
    api('POST', '/competitions', adminToken, {
      name: tag,
      location: 'Cypress E2E',
      start_date: '2026-06-01',
      end_date: '2026-06-03',
    }),
  )
  const arena = await must(
    'buat gelanggang',
    api('POST', `/competitions/${comp.id}/arenas`, adminToken, { name: 'Gelanggang Utama' }),
  )
  const other = await must(
    'buat gelanggang lain',
    api('POST', `/competitions/${comp.id}/arenas`, adminToken, { name: 'Gelanggang Lain' }),
  )
  const ruleset = await must(
    'buat ruleset uji',
    api('POST', '/rulesets', adminToken, { name: tag, version: '1', scoring: scoring(opt) }),
  )
  // kejuaraan sungguhan selalu punya ruleset; tanpa ini "Muat ke Gelanggang" dari UI panitia gagal ("ruleset '' not found")
  await must(
    'pasang ruleset ke kejuaraan',
    api('PUT', `/competitions/${comp.id}`, adminToken, {
      name: tag,
      location: 'Cypress E2E',
      start_date: '2026-06-01',
      end_date: '2026-06-03',
      ruleset_id: ruleset.id,
    }),
  )
  const category = await must(
    'buat kategori',
    api('POST', `/competitions/${comp.id}/categories`, adminToken, {
      name: 'Dewasa Putra',
      gender: 'male',
      weight_class: 'A',
      age_group: 'dewasa',
    }),
  )

  const wasit = await pinSession(adminToken, comp.id, arena.id, 'wasit', 0)
  const judges: Session[] = []
  for (let seat = 1; seat <= opt.judges; seat += 1) {
    judges.push(await pinSession(adminToken, comp.id, arena.id, 'juri', seat))
  }
  const outsideJudge = await pinSession(adminToken, comp.id, other.id, 'juri', 1)
  const needOtherWasit = options.outsideWasit || options.otherArenaCrew
  const outsideWasit = needOtherWasit
    ? await pinSession(adminToken, comp.id, other.id, 'wasit', 0)
    : outsideJudge
  const otherJudges: Session[] = [outsideJudge]
  if (options.otherArenaCrew) {
    for (let seat = 2; seat <= opt.judges; seat += 1) {
      otherJudges.push(await pinSession(adminToken, comp.id, other.id, 'juri', seat))
    }
  }

  const createdMatches: string[] = []
  const kit: MatchKit = {
    adminToken,
    competitionId: comp.id,
    arenaId: arena.id,
    categoryId: category.id,
    otherArenaId: other.id,
    rulesetId: ruleset.id,
    wasit,
    judges,
    outsideJudge,
    outsideWasit,
    roundMs: opt.roundSeconds * 1000,
    breakMs: opt.breakSeconds * 1000,
    totalRounds: opt.totalRounds,

    async createMatch() {
      const names: string[] = []
      const ids: string[] = []
      for (const n of ['Merah', 'Biru']) {
        const name = `${n} ${uid()}`
        const reg = await must(
          `daftarkan atlet ${n}`,
          api('POST', `/competitions/${comp.id}/athletes`, adminToken, {
            name,
            gender: 'male',
            team: `Klub ${n}`,
            birth_date: '1998-01-01',
            category_id: category.id,
          }),
        )
        const parts = await must(
          'daftar peserta',
          api('GET', `/competitions/${comp.id}/participants`, adminToken),
        )
        ids.push(parts.find((p: Json) => p.athlete_id === reg.athlete_id).id)
        names.push(name)
      }
      const m = await must(
        'buat partai',
        api('POST', `/competitions/${comp.id}/matches`, adminToken, {
          arena_id: arena.id,
          athlete_red_id: ids[0],
          athlete_blue_id: ids[1],
        }),
      )
      createdMatches.push(m.id)
      return { id: m.id, redId: ids[0], blueId: ids[1], redName: names[0], blueName: names[1] }
    },

    crew(key) {
      return key === 'main'
        ? { arenaId: arena.id, wasit, judges }
        : { arenaId: other.id, wasit: outsideWasit, judges: otherJudges }
    },

    async createCategory(def) {
      const c = await must(
        `buat kategori ${def.name}`,
        api('POST', `/competitions/${comp.id}/categories`, adminToken, def),
      )
      return c.id
    },

    loadMatch(match, token = wasit.token, key = 'main') {
      return api('POST', `/competitions/${comp.id}/matches/${match.id}/load-arena`, token, {
        arena_id: key === 'main' ? arena.id : other.id,
        ruleset_path: ruleset.id,
      })
    },

    control(match, action, body, token = wasit.token) {
      return api('POST', `/competitions/${comp.id}/matches/${match.id}/${action}`, token, body)
    },

    async openDevices(opts = {}) {
      const skip = new Set(opts.skipSeats ?? [])
      const crew = kit.crew(opts.arena ?? 'main')
      const w = await ArenaSocket.open(crew.arenaId, crew.wasit.token, 'wasit')
      const js: ArenaSocket[] = []
      for (const j of crew.judges) {
        if (!skip.has(j.seat ?? 0))
          js.push(await ArenaSocket.open(crew.arenaId, j.token, `juri-${j.seat}`))
      }
      // pastikan semua perangkat sudah terdaftar di hub sebelum tes mengirim perintah
      await sleep(150)
      const devices: Devices = {
        wasit: w,
        judges: js,
        all: [w, ...js],
        // menutup perangkat yang berlaku SAAT DIPANGGIL: tes pemulihan boleh mengganti soket
        closeAll: async () => {
          await Promise.all(devices.all.map((s) => s.close()))
        },
      }
      return devices
    },

    async teardown() {
      for (const id of createdMatches) {
        await api('POST', `/competitions/${comp.id}/matches/${id}/cancel`, wasit.token)
      }
      await api('POST', `/competitions/${comp.id}/cancel`, adminToken)
      await api('DELETE', `/rulesets/${ruleset.id}`, adminToken)
    },
  }
  return kit
}

/** Login akun email (admin/dewan/operator) dan kontingen; token dipakai untuk uji otorisasi. */
export async function loginOfficial(email: string, password: string): Promise<string> {
  const d = await must(`login ${email}`, api('POST', '/auth/login', null, { email, password }))
  return d.token
}

export async function loginContingent(email: string, password: string): Promise<string> {
  const d = await must(
    `login kontingen ${email}`,
    api('POST', '/auth/contingent/login', null, { email, password }),
  )
  return d.token
}
