// End-to-End Match Simulation Script for DigitalScoring
// Verifies: REST API Auth + WebSocket Hub + Consensus Engine + Scoreboard + Referee + 3 Judges
import { randomUUID } from 'crypto'

const BASE_HTTP = 'http://localhost:8080'
const BASE_WS = 'ws://localhost:8080'

function sleep(ms) {
  return new Promise((resolve) => setTimeout(resolve, ms))
}

// Setiap pemeriksaan dicatat; skrip keluar dengan kode 1 bila ada yang gagal.
const failures = []
function check(name, condition, detail = '') {
  if (condition) {
    console.log(`   ✔ ${name}`)
  } else {
    console.log(`   ✘ ${name} ${detail}`)
    failures.push(name)
  }
}

function connect(url) {
  return new Promise((resolve, reject) => {
    const ws = new WebSocket(url)
    const events = []
    ws.onmessage = (event) => events.push(JSON.parse(event.data))
    ws.onopen = () => resolve({ ws, events })
    ws.onerror = () => reject(new Error(`WS gagal terhubung: ${url.split('?')[0]}`))
  })
}

const last = (events, type) => [...events].reverse().find((e) => e.type === type)

async function postJson(path, body, headers = {}) {
  const res = await fetch(`${BASE_HTTP}${path}`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json', ...headers },
    body: JSON.stringify(body),
  })
  const text = await res.text()
  let data
  try {
    data = JSON.parse(text)
  } catch {
    throw new Error(`Invalid JSON from ${path}: ${text}`)
  }
  if (!res.ok || data.success === false) {
    throw new Error(`HTTP ${res.status}: ${data.error?.message || text}`)
  }
  return data
}

async function getJson(path, headers = {}) {
  const res = await fetch(`${BASE_HTTP}${path}`, {
    headers: { 'Content-Type': 'application/json', ...headers },
  })
  const data = await res.json()
  if (!res.ok || data.success === false) {
    throw new Error(`HTTP ${res.status}: ${data.error?.message || JSON.stringify(data)}`)
  }
  return data
}

async function runSimulation() {
  console.log('🥋 === DIGITAL SCORING END-TO-END SIMULATION START === 🥋\n')

  // 1. Health check
  const health = await getJson('/api/v1/health')
  console.log('1. Backend Health:', health.data)

  // 2. Admin Login
  console.log('2. Logging in as Admin...')
  const adminLogin = await postJson('/api/v1/auth/login', {
    email: 'admin@silat.id',
    password: 'DemoPass_2026',
  })
  const adminToken = adminLogin.data.token
  const authHeaders = { Authorization: `Bearer ${adminToken}` }
  console.log('   Admin logged in successfully! Role:', adminLogin.data.user.role)

  // 3. Fetch active competition and arena
  console.log('3. Fetching competitions and arenas...')
  const comps = await getJson('/api/v1/competitions', authHeaders)
  const comp = comps.data[0]
  console.log(`   Found Competition: "${comp.name}" (ID: ${comp.id})`)

  const arenas = await getJson(`/api/v1/competitions/${comp.id}/arenas`, authHeaders)
  const arena = arenas.data[0]
  console.log(`   Using Arena: "${arena.name}" (ID: ${arena.id})`)

  // 4. Generate PINs for 3 Judges and 1 Wasit
  console.log('4. Generating PINs for 3 Judges...')
  const judges = []
  for (let seat = 1; seat <= 3; seat++) {
    const pin = `99900${seat}`
    await postJson(
      '/api/v1/auth/create-pin',
      {
        competition_id: comp.id,
        arena_id: arena.id,
        role: 'juri',
        seat_number: seat,
        pin,
        ttl_hours: 12,
      },
      authHeaders
    )

    // Login Judge using PIN
    const judgeLogin = await postJson('/api/v1/auth/judge-login', { pin })
    judges.push({
      seat,
      token: judgeLogin.data.token,
    })
    console.log(`   Judge ${seat} registered & logged in with PIN ${pin}`)
  }

  // 5. Fetch matches and load match into Arena
  console.log('5. Loading match into Arena...')
  const matchesRes = await getJson(`/api/v1/competitions/${comp.id}/matches`, authHeaders)
  const targetMatch = matchesRes.data[0]
  console.log(`   Target Match ID: ${targetMatch.id}`)

  const rulesetsRes = await getJson('/api/v1/rulesets', authHeaders)
  const ruleset = rulesetsRes.data[0]
  console.log(`   Using Ruleset: ${ruleset.name} (${ruleset.id})`)

  try {
    await postJson(
      `/api/v1/competitions/${comp.id}/matches/${targetMatch.id}/load-arena`,
      {
        arena_id: arena.id,
        ruleset_path: ruleset.id,
      },
      authHeaders
    )
    console.log('   Match successfully loaded into Arena!')
  } catch (err) {
    console.log('   Match load info:', err.message)
  }

  // 6. Establish WebSocket Connections
  console.log('\n6. Connecting WebSockets for all match officials...')

  // Wasit login via PIN (alur aplikasi: token terikat arena, role "wasit")
  await postJson(
    '/api/v1/auth/create-pin',
    { competition_id: comp.id, arena_id: arena.id, role: 'wasit', seat_number: 0, pin: '99900000', ttl_hours: 12 },
    authHeaders
  )
  // Login hanya dengan PIN: server menentukan turnamen, gelanggang, role, dan kursi
  const wasitLogin = await postJson('/api/v1/auth/judge-login', { pin: '99900000' })
  check('Login PIN menentukan arena dari PIN', wasitLogin.data.arena_id === arena.id && wasitLogin.data.competition_id === comp.id)
  const wasitToken = wasitLogin.data.token
  const wrong = await fetch(`${BASE_HTTP}/api/v1/auth/judge-login`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ pin: '00000000' }),
  })
  check('PIN salah ditolak 401', wrong.status === 401, String(wrong.status))
  const malformed = await fetch(`${BASE_HTTP}/api/v1/auth/judge-login`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ pin: '123' }),
  })
  check('PIN berformat salah ditolak 400', malformed.status === 400, String(malformed.status))
  const taken = await fetch(`${BASE_HTTP}/api/v1/auth/create-pin`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json', ...authHeaders },
    body: JSON.stringify({ competition_id: comp.id, arena_id: arena.id, role: 'juri', seat_number: 4, pin: '999001', ttl_hours: 1 }),
  })
  check('PIN kustom yang bentrok ditolak 409', taken.status === 409, String(taken.status))
  check('Login PIN wasit mengembalikan role "wasit"', wasitLogin.data.role === 'wasit', wasitLogin.data.role)

  // A. Scoreboard TV: publik, tanpa token
  const sb = await connect(`${BASE_WS}/ws/arena/${arena.id}`)
  check('Scoreboard TV publik (tanpa token) bisa terhubung', true)

  // B. Konsol wasit (token PIN, role wasit)
  const ref = await connect(`${BASE_WS}/ws/arena/${arena.id}?token=${wasitToken}`)
  check('Konsol wasit (token PIN) bisa terhubung', true)

  // C. 3 tablet juri
  const judgeSockets = []
  for (const j of judges) {
    judgeSockets.push(await connect(`${BASE_WS}/ws/arena/${arena.id}?token=${j.token}`))
  }
  check('3 tablet juri terhubung', judgeSockets.length === 3)

  // D. Token arena lain ditolak, arena tanpa hub ditolak untuk penonton
  const otherArena = arenas.data.find((a) => a.id !== arena.id)
  if (otherArena) {
    await connect(`${BASE_WS}/ws/arena/${otherArena.id}?token=${wasitToken}`).then(
      () => check('Token arena lain ditolak', false),
      () => check('Token arena lain ditolak', true)
    )
  }
  await connect(`${BASE_WS}/ws/arena/${randomUUID()}`).then(
    () => check('Penonton ke arena tanpa pertandingan ditolak', false),
    () => check('Penonton ke arena tanpa pertandingan ditolak', true)
  )

  await sleep(400)

  // 7. Hak akses perintah: juri & penonton tidak boleh mengendalikan pertandingan
  console.log('\n7. Access control for match control commands...')
  judgeSockets[0].ws.send(JSON.stringify({ type: 'state_change', payload: { action: 'start' } }))
  sb.ws.send(JSON.stringify({ type: 'state_change', payload: { action: 'start' } }))
  await sleep(400)
  check('Juri ditolak (FORBIDDEN) saat mencoba start', last(judgeSockets[0].events, 'error')?.payload?.code === 'FORBIDDEN')
  check('Penonton ditolak (FORBIDDEN) saat mencoba start', last(sb.events, 'error')?.payload?.code === 'FORBIDDEN')
  check('Pertandingan belum mulai setelah percobaan ilegal', !last(sb.events, 'state_change'))

  // 8. Wasit memulai pertandingan
  console.log('\n8. Referee triggers match START...')
  ref.ws.send(JSON.stringify({ type: 'state_change', payload: { action: 'start' } }))
  await sleep(600)
  const started = last(sb.events, 'state_change')
  check('Scoreboard menerima state_change round_1', started?.payload?.state === 'round_1', JSON.stringify(started?.payload))
  check('state_change membawa timer_running=true', started?.payload?.timer_running === true)

  const live = await getJson(`/api/v1/live/scoreboard/${arena.id}`)
  check('REST /live/scoreboard menampilkan pertandingan aktif', live.data.status === 'round_1' && live.data.match_id === targetMatch.id, JSON.stringify(live.data))
  check('REST /live/scoreboard memuat nama atlet', !!live.data.red_athlete && !!live.data.blue_athlete)

  // 9. Konsensus: 2 dari 3 juri menilai sama dalam jendela waktu
  console.log('\n9. Simulating Judge Scoring (Red Technique +1, 2/3 majority)...')
  for (const idx of [0, 1]) {
    judgeSockets[idx].ws.send(JSON.stringify({
      type: 'judge_input',
      payload: { event_id: randomUUID(), idempotency_key: randomUUID(), score_type: 'technique', target: 'red', round: 1 },
    }))
    await sleep(300)
  }
  await sleep(1000)

  const confirmed = last(sb.events, 'score_confirmed')
  check('Scoreboard menerima score_confirmed', confirmed?.payload?.target === 'red' && confirmed?.payload?.points === 1, JSON.stringify(confirmed?.payload))
  check('Wasit menerima score_confirmed', !!last(ref.events, 'score_confirmed'))
  const update = last(sb.events, 'score_update')
  check('score_update: merah=1, biru=0', update?.payload?.red_technical === 1 && update?.payload?.blue_technical === 0, JSON.stringify(update?.payload))

  // 9b. Fitur meja wasit: sanksi, pembatalan, sinyal arena, kehadiran, log
  console.log('\n9b. Referee console features (penalties, signals, presence, events)...')
  const send = (client, type, payload) => client.ws.send(JSON.stringify({ type, payload }))
  const lastMsg = (client, type) => last(client.events, type)
  const errorsOf = (client) => client.events.filter((e) => e.type === 'error').map((e) => e.payload?.code)

  send(ref, 'penalty', { target: 'red', type: 'teguran_1' })
  send(ref, 'penalty', { target: 'red', type: 'peringatan_1' })
  await sleep(400)
  const pens = lastMsg(sb, 'penalties')?.payload?.penalties ?? []
  check('Sanksi disiarkan ke scoreboard (2 sanksi merah)', pens.filter((x) => x.target === 'red').length === 2, JSON.stringify(pens))
  check('Poin sanksi ditentukan server (teguran 1 = -1, peringatan 1 = -5)', pens.map((x) => x.points).join() === '1,5', pens.map((x) => x.points).join())
  const afterPen = lastMsg(sb, 'score_update')?.payload
  check('Skor merah dipotong sanksi (teknik 1, potongan 6)', afterPen?.red_technical === 1 && afterPen?.red_penalties === 6, JSON.stringify(afterPen))

  send(ref, 'penalty', { target: 'red', type: 'teguran_1' })
  send(judgeSockets[0], 'penalty', { target: 'blue', type: 'teguran_1' })
  send(ref, 'penalty', { target: 'red', type: 'ngawur' })
  await sleep(400)
  check('Sanksi ganda ditolak (PENALTY_ALREADY_APPLIED)', errorsOf(ref).includes('PENALTY_ALREADY_APPLIED'), errorsOf(ref).join())
  check('Juri tidak boleh menjatuhkan sanksi (FORBIDDEN)', errorsOf(judgeSockets[0]).includes('FORBIDDEN'))
  check('Jenis sanksi tidak dikenal ditolak (BAD_PAYLOAD)', errorsOf(ref).includes('BAD_PAYLOAD'))

  send(ref, 'penalty_revoke', { target: 'red' })
  await sleep(400)
  const revoked = lastMsg(sb, 'penalties')?.payload?.penalties ?? []
  check('Batalkan sanksi terakhir (peringatan_1 hilang, teguran_1 tetap)', revoked.length === 1 && revoked[0].type === 'teguran_1', JSON.stringify(revoked))
  check('Poin potongan dikembalikan (potongan 1)', lastMsg(sb, 'score_update')?.payload?.red_penalties === 1)

  send(ref, 'arena_signal', { signal: 'gong' })
  send(ref, 'arena_signal', { signal: 'medic', active: true })
  send(ref, 'arena_signal', { signal: 'var', active: true })
  await sleep(400)
  const signals = sb.events.filter((e) => e.type === 'arena_signal').map((e) => `${e.payload.signal}:${e.payload.active}`)
  check('Scoreboard menerima sinyal gong, dokter, dan VAR', ['gong:true', 'medic:true', 'var:true'].every((x) => signals.includes(x)), signals.join())
  const snapNow = await getJson(`/api/v1/live/scoreboard/${arena.id}`)
  check('Status dokter/VAR tampil di snapshot REST', snapNow.data.medic_active === true && snapNow.data.var_active === true)
  send(ref, 'arena_signal', { signal: 'medic', active: false })
  send(ref, 'arena_signal', { signal: 'var', active: false })
  send(judgeSockets[0], 'arena_signal', { signal: 'gong' })
  await sleep(300)
  check('Juri tidak boleh membunyikan gong (FORBIDDEN)', errorsOf(judgeSockets[0]).filter((c) => c === 'FORBIDDEN').length >= 2)

  const presence = lastMsg(ref, 'presence')?.payload
  const online = (presence?.judges ?? []).filter((j) => j.connected).map((j) => j.seat)
  check('Kehadiran: juri kursi 1-3 online', [1, 2, 3].every((n) => online.includes(n)) && online.length === 3, online.join())
  check('Kehadiran: 1 layar TV dan 1 wasit', presence?.scoreboards === 1 && presence?.referees === 1, JSON.stringify({ tv: presence?.scoreboards, ref: presence?.referees }))
  const j1 = (presence?.judges ?? []).find((j) => j.seat === 1)
  check('Aktivitas juri 1 tercatat (input terakhir + jumlah)', j1?.inputs >= 1 && j1?.last_input?.target === 'red', JSON.stringify(j1))

  const eventsRes = await getJson(`/api/v1/competitions/${comp.id}/matches/${targetMatch.id}/events`, authHeaders)
  const types = eventsRes.data.map((e) => e.event_type)
  check('Log REST memuat match_started, score_confirmed, penalty_applied, penalty_revoked, arena_signal',
    ['match_started', 'score_confirmed', 'penalty_applied', 'penalty_revoked', 'arena_signal'].every((t) => types.includes(t)), types.join())
  const liveEv = ref.events.filter((e) => e.type === 'match_event').map((e) => e.payload.event_type)
  check('Log real-time (match_event) sampai ke meja wasit', liveEv.includes('penalty_applied') && liveEv.includes('arena_signal'))
  check('Penonton publik tidak menerima log internal', !sb.events.some((e) => e.type === 'match_event'))
  check('Setiap kejadian membawa clock_ms dan round', ref.events.filter((e) => e.type === 'match_event').every((e) => typeof e.payload.payload.clock_ms === 'number' && typeof e.payload.payload.round === 'number'))

  const sync = await new Promise((resolve) => {
    const before = ref.events.length
    send(ref, 'sync_request', {})
    const t = setInterval(() => {
      const m = ref.events.slice(before).find((e) => e.type === 'sync_response')
      if (m) { clearInterval(t); resolve(m.payload) }
    }, 50)
  })
  check('sync_response memuat katalog sanksi, sanksi aktif, dan info ruleset',
    sync.penalty_catalog?.length === 7 && sync.penalties?.length === 1 && sync.round_duration_ms === 120000 && sync.total_rounds === 3,
    JSON.stringify({ cat: sync.penalty_catalog?.length, pen: sync.penalties?.length, dur: sync.round_duration_ms }))
  const kinds = Object.fromEntries((sync.penalty_catalog ?? []).map((k) => [k.type, k]))
  check('Katalog: Teguran II = -2, Peringatan III = diskualifikasi',
    kinds.teguran_2?.points === 2 && kinds.peringatan_3?.action === 'disqualify', JSON.stringify(kinds.teguran_2) + JSON.stringify(kinds.peringatan_3))
  check('sync_response memuat teknik ruleset (termasuk Tangkapan + Jatuhan = 4), jumlah juri, dan konsensus',
    ['technique', 'knockdown', 'fall', 'catch_fall'].every((t) => sync.techniques?.some((x) => x.type === t)) &&
      sync.techniques.find((x) => x.type === 'catch_fall')?.points === 4 && sync.judge_count === 3 && sync.consensus?.majority === 2,
    JSON.stringify({ tech: sync.techniques, judges: sync.judge_count, c: sync.consensus }))

  const queueRes = await getJson(`/api/v1/competitions/${comp.id}/matches?arena_id=${arena.id}&limit=100`, authHeaders)
  check('Antrean per gelanggang: semua partai berada di arena ini', queueRes.data.length >= 1 && queueRes.data.every((m) => m.arena_id === arena.id), String(queueRes.data.length))

  // 10. Pause & resume
  console.log('\n10. Referee tests PAUSE and RESUME...')
  ref.ws.send(JSON.stringify({ type: 'state_change', payload: { action: 'pause' } }))
  await sleep(400)
  check('Pause: timer_running=false', last(sb.events, 'state_change')?.payload?.timer_running === false)
  ref.ws.send(JSON.stringify({ type: 'state_change', payload: { action: 'resume' } }))
  await sleep(400)
  check('Resume: timer_running=true', last(sb.events, 'state_change')?.payload?.timer_running === true)

  // Koreksi waktu & reset babak
  const remBefore = last(sb.events, 'state_change').payload.timer_remaining_ms
  send(ref, 'state_change', { action: 'adjust_time', delta_ms: -10000 })
  await sleep(400)
  const remAfter = last(sb.events, 'state_change').payload.timer_remaining_ms
  check('Koreksi waktu -10 detik disiarkan ke scoreboard', Math.abs(remBefore - 10000 - remAfter) <= 2000, `${remBefore} -> ${remAfter}`)
  send(ref, 'state_change', { action: 'reset_round' })
  await sleep(300)
  check('Reset babak saat berjalan ditolak', errorsOf(ref).includes('INVALID_STATE_TRANSITION'), errorsOf(ref).join())
  send(ref, 'state_change', { action: 'pause' })
  await sleep(300)
  send(ref, 'state_change', { action: 'reset_round' })
  await sleep(400)
  const remReset = last(sb.events, 'state_change').payload
  check('Reset babak saat dijeda mengembalikan waktu penuh (02:00)', remReset.timer_remaining_ms === 120000 && remReset.timer_running === false, JSON.stringify(remReset))
  send(ref, 'state_change', { action: 'resume' })
  await sleep(300)

  // 11. Selesai. Skor akhir seri (merah 1-1 sanksi, biru 0) -> wasit wajib memutuskan pemenang
  console.log('\n11. Finishing match (tie requires referee decision)...')
  const errorsBefore = errorsOf(ref).length
  send(ref, 'state_change', { action: 'finish' })
  await sleep(500)
  check('Selesai saat skor seri tanpa keputusan ditolak', errorsOf(ref).length > errorsBefore && !last(sb.events, 'match_result'), errorsOf(ref).join())
  const restTie = await fetch(`${BASE_HTTP}/api/v1/competitions/${comp.id}/matches/${targetMatch.id}/finish`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json', ...authHeaders },
  })
  const restTieBody = await restTie.json()
  check('REST finish saat seri -> 400 TIE_NEEDS_DECISION', restTie.status === 400 && restTieBody.error?.code === 'TIE_NEEDS_DECISION', JSON.stringify(restTieBody))

  const restFinish = await fetch(`${BASE_HTTP}/api/v1/competitions/${comp.id}/matches/${targetMatch.id}/finish`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json', ...authHeaders },
    body: JSON.stringify({ winner: 'blue' }),
  })
  check('REST finish dengan keputusan wasit diterima', restFinish.status === 200, String(restFinish.status))
  await sleep(500)

  send(ref, 'penalty', { target: 'blue', type: 'binaan_1' })
  await sleep(300)
  check('Sanksi setelah pertandingan selesai ditolak', errorsOf(ref).includes('PENALTY_NOT_ALLOWED'), errorsOf(ref).join())

  const matchResult = last(sb.events, 'match_result')
  check('match_result: sudut biru menang lewat keputusan wasit', matchResult?.payload?.corner === 'blue' && matchResult?.payload?.result_type === 'referee_decision', JSON.stringify(matchResult?.payload))
  const after = await getJson(`/api/v1/live/scoreboard/${arena.id}`)
  check('REST /live/scoreboard kembali "no active match"', after.data.status === 'no active match', JSON.stringify(after.data))

  const sbWs = sb.ws
  const refWs = ref.ws

  // Close all sockets
  sbWs.close()
  refWs.close()
  judgeSockets.forEach(({ ws }) => ws.close())

  console.log('\n✅ All WebSocket connections cleanly closed.')
  if (failures.length > 0) {
    console.error(`\n❌ ${failures.length} pemeriksaan gagal:\n - ${failures.join('\n - ')}`)
    process.exit(1)
  }
  console.log('🥋 === DIGITAL SCORING END-TO-END SIMULATION FINISHED SUCCESSFULLY === 🥋\n')
}

runSimulation().catch(err => {
  console.error('❌ Simulation Error:', err)
  process.exit(1)
})
