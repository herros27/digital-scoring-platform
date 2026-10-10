import { api, must, type Json } from './http.ts'
import type { ArenaKey, Devices, MatchKit } from './kit.ts'
import { confirmScore, eachSocket, snapshot, waitState } from './helpers.ts'

/** Peserta yang didaftarkan lewat API dengan identitas lengkap (atlet + peserta kejuaraan). */
export interface Entrant {
  name: string
  team: string
  participantId: string
  athleteId: string
}

const DOCS = { identity: true, health: true }

/** Mendaftarkan atlet ke kategori kit, lalu memverifikasi/menimbang agar layak masuk bagan. */
export async function enrollAndVerify(
  kit: MatchKit,
  list: Array<{ name: string; team: string; weight?: number }>,
  opts: { categoryId?: string; gender?: 'male' | 'female'; docs?: Record<string, boolean> } = {},
): Promise<Entrant[]> {
  const categoryId = opts.categoryId ?? kit.categoryId
  const out: Entrant[] = []
  for (const a of list) {
    const reg = await must(
      `daftar ${a.name}`,
      api('POST', `/competitions/${kit.competitionId}/athletes`, kit.adminToken, {
        name: a.name,
        gender: opts.gender ?? 'male',
        team: a.team,
        birth_date: '1998-01-01',
        category_id: categoryId,
      }),
    )
    out.push({ name: a.name, team: a.team, athleteId: reg.athlete_id, participantId: '' })
  }
  const parts = await must(
    'daftar peserta',
    api('GET', `/competitions/${kit.competitionId}/participants`, kit.adminToken),
  )
  for (const e of out) {
    e.participantId = parts.find((p: Json) => p.athlete_id === e.athleteId).id
    await must(
      `verifikasi ${e.name}`,
      api(
        'PUT',
        `/competitions/${kit.competitionId}/participants/${e.participantId}/verify`,
        kit.adminToken,
        { status: 'verified', actual_weight: 60, docs_verified: opts.docs ?? DOCS },
      ),
    )
  }
  return out
}

export const generateBracket = (kit: MatchKit, categoryId = kit.categoryId) =>
  api('POST', `/competitions/${kit.competitionId}/brackets/generate`, kit.adminToken, {
    category_id: categoryId,
  })

export const listMatches = (kit: MatchKit): Promise<Json[]> =>
  must('daftar partai', api('GET', `/competitions/${kit.competitionId}/matches`, kit.adminToken))

export const bracketSlots = (kit: MatchKit, categoryId = kit.categoryId): Promise<Json[]> =>
  must(
    'bagan',
    api(
      'GET',
      `/competitions/${kit.competitionId}/brackets?category_id=${categoryId}`,
      kit.adminToken,
    ),
  )

/**
 * Menjalankan satu partai sampai selesai seperti di arena: muat ke gelanggang, buka perangkat (wasit + 3 juri),
 * mulai, juri memberi poin sesuai `scoring` per babak, lalu partai ditutup (wasit `finish`, atau dibiarkan habis
 * waktu bila `untilTimeout`). Mengembalikan hasil akhir dari `match_result` yang diterima semua perangkat.
 */
export async function playMatch(
  kit: MatchKit,
  matchId: string,
  scoring: Array<{ target: 'red' | 'blue'; type?: string }>,
  opts: { untilTimeout?: boolean; arena?: ArenaKey } = {},
): Promise<{ result: Json; devices: Devices }> {
  const match = { id: matchId, redId: '', blueId: '', redName: '', blueName: '' }
  // Partai hanya boleh dimuat di gelanggang tempat ia dijadwalkan (gelanggang asal kategori): ikuti data server.
  let arena: ArenaKey = opts.arena ?? 'main'
  if (!opts.arena) {
    const rec = await must<Json>(
      'partai',
      api('GET', `/competitions/${kit.competitionId}/matches/${matchId}`, kit.adminToken),
    )
    arena = rec.arena_id === kit.otherArenaId ? 'other' : 'main'
  }
  const token = kit.crew(arena).wasit.token // wasit PIN terikat ke gelanggangnya
  const devices = await kit.openDevices({ arena })
  try {
    await must('load-arena', kit.loadMatch(match, token, arena))
    await eachSocket(devices.all, (s) =>
      s.waitFor((m) => m.type === 'sync_response' && m.payload?.match_id === matchId, {
        what: 'snapshot partai',
      }),
    )
    await must('start', kit.control(match, 'start', undefined, token))
    await waitState(devices.wasit, 'round_1')

    const marks = devices.all.map((s) => s.mark())
    for (const [i, step] of scoring.entries()) {
      if (i > 0 && opts.untilTimeout) {
        // poin babak berikutnya baru boleh masuk setelah istirahat selesai
        await waitState(devices.wasit, `round_${i + 1}`, marks[0], 15_000)
      }
      const before = devices.wasit.mark()
      await confirmScore(devices, step.target, step.type ?? 'knockdown')
      await devices.wasit.waitFor('score_confirmed', { since: before, what: `poin ${i + 1}` })
    }

    if (opts.untilTimeout) {
      await waitState(devices.wasit, 'finished', marks[0], 40_000)
    } else {
      await must('finish', kit.control(match, 'finish', undefined, token))
      await waitState(devices.wasit, 'finished', marks[0])
    }
    const result = (await snapshot(devices.wasit)).result
    return { result, devices }
  } finally {
    await devices.closeAll()
  }
}

export interface PlayedMatch {
  id: string
  round: number
  winner: string
  loser: string
}

/**
 * Memainkan seluruh bagan sampai tidak ada partai idle tersisa (pemenang = sudut merah). Partai babak berikutnya
 * terbentuk otomatis saat pemenang maju, jadi daftar partai dibaca ulang tiap putaran.
 */
export async function playBracketToEnd(
  kit: MatchKit,
  opts: { categoryId?: string; arena?: ArenaKey; maxPasses?: number } = {},
): Promise<PlayedMatch[]> {
  const categoryId = opts.categoryId ?? kit.categoryId
  const slots = await bracketSlots(kit, categoryId)
  const inCategory = new Set(slots.map((s) => s.id))
  const roundOf = (slotId: string) => slots.find((s) => s.id === slotId)?.round as number
  const done = new Set<string>()
  const played: PlayedMatch[] = []
  for (let pass = 0; pass < (opts.maxPasses ?? 12); pass += 1) {
    const idle = (await listMatches(kit)).filter(
      (m) => m.status === 'idle' && !done.has(m.id) && inCategory.has(m.bracket_slot_id),
    )
    if (idle.length === 0) break
    for (const m of idle) {
      await playMatch(kit, m.id, [{ target: 'red', type: 'knockdown' }], { arena: opts.arena })
      done.add(m.id)
      played.push({
        id: m.id,
        round: roundOf(m.bracket_slot_id),
        winner: m.athlete_red_id,
        loser: m.athlete_blue_id,
      })
    }
  }
  return played
}
