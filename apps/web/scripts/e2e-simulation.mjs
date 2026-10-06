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

  // 10. Pause & resume
  console.log('\n10. Referee tests PAUSE and RESUME...')
  ref.ws.send(JSON.stringify({ type: 'state_change', payload: { action: 'pause' } }))
  await sleep(400)
  check('Pause: timer_running=false', last(sb.events, 'state_change')?.payload?.timer_running === false)
  ref.ws.send(JSON.stringify({ type: 'state_change', payload: { action: 'resume' } }))
  await sleep(400)
  check('Resume: timer_running=true', last(sb.events, 'state_change')?.payload?.timer_running === true)

  // 11. Selesai
  console.log('\n11. Finishing match...')
  ref.ws.send(JSON.stringify({ type: 'state_change', payload: { action: 'finish' } }))
  await sleep(600)

  const matchResult = last(sb.events, 'match_result')
  check('match_result: sudut merah menang angka', matchResult?.payload?.corner === 'red' && matchResult?.payload?.result_type === 'point_win', JSON.stringify(matchResult?.payload))
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
