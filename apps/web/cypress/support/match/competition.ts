import { api, must, type Json } from './http.ts'
import type { Devices, MatchKit } from './kit.ts'
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
): Promise<Entrant[]> {
  const out: Entrant[] = []
  for (const a of list) {
    const reg = await must(
      `daftar ${a.name}`,
      api('POST', `/competitions/${kit.competitionId}/athletes`, kit.adminToken, {
        name: a.name,
        gender: 'male',
        team: a.team,
        birth_date: '1998-01-01',
        category_id: kit.categoryId,
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
        { status: 'verified', actual_weight: 60, docs_verified: DOCS },
      ),
    )
  }
  return out
}

export const generateBracket = (kit: MatchKit) =>
  api('POST', `/competitions/${kit.competitionId}/brackets/generate`, kit.adminToken, {
    category_id: kit.categoryId,
  })

export const listMatches = (kit: MatchKit): Promise<Json[]> =>
  must('daftar partai', api('GET', `/competitions/${kit.competitionId}/matches`, kit.adminToken))

export const bracketSlots = (kit: MatchKit): Promise<Json[]> =>
  must(
    'bagan',
    api(
      'GET',
      `/competitions/${kit.competitionId}/brackets?category_id=${kit.categoryId}`,
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
  opts: { untilTimeout?: boolean } = {},
): Promise<{ result: Json; devices: Devices }> {
  const match = { id: matchId, redId: '', blueId: '', redName: '', blueName: '' }
  const devices = await kit.openDevices()
  try {
    await must('load-arena', kit.loadMatch(match))
    await eachSocket(devices.all, (s) =>
      s.waitFor((m) => m.type === 'sync_response' && m.payload?.match_id === matchId, {
        what: 'snapshot partai',
      }),
    )
    await must('start', kit.control(match, 'start'))
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
      await must('finish', kit.control(match, 'finish'))
      await waitState(devices.wasit, 'finished', marks[0])
    }
    const result = (await snapshot(devices.wasit)).result
    return { result, devices }
  } finally {
    await devices.closeAll()
  }
}
