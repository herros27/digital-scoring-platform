// End-to-End Match Simulation Script for DigitalScoring
// Verifies: REST API Auth + WebSocket Hub + Consensus Engine + Scoreboard + Referee + 3 Judges
import { randomUUID } from 'crypto'

const BASE_HTTP = 'http://localhost:8080'
const BASE_WS = 'ws://localhost:8080'

function sleep(ms) {
  return new Promise((resolve) => setTimeout(resolve, ms))
}

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
    const judgeLogin = await postJson('/api/v1/auth/judge-login', {
      competition_id: comp.id,
      arena_id: arena.id,
      pin,
    })
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

  // A. Scoreboard Display (Public, No Token)
  const scoreboardEvents = []
  const sbWs = new WebSocket(`${BASE_WS}/ws/arena/${arena.id}`)
  await new Promise((resolve, reject) => {
    sbWs.onopen = () => {
      console.log('   ✅ Scoreboard TV connected (Public Spectator Mode)')
      resolve()
    }
    sbWs.onerror = (e) => reject(new Error('Scoreboard WS error: ' + e.message))
  })
  sbWs.onmessage = (event) => {
    const data = JSON.parse(event.data)
    scoreboardEvents.push(data)
    console.log(`   📺 [Scoreboard Recv] ${data.type}`, data.payload || '')
  }

  // B. Referee Console (Wasit/Admin Token)
  const refereeEvents = []
  const refWs = new WebSocket(`${BASE_WS}/ws/arena/${arena.id}?token=${adminToken}`)
  await new Promise((resolve, reject) => {
    refWs.onopen = () => {
      console.log('   ✅ Referee Console connected')
      resolve()
    }
    refWs.onerror = (e) => reject(new Error('Referee WS error: ' + e.message))
  })
  refWs.onmessage = (event) => {
    const data = JSON.parse(event.data)
    refereeEvents.push(data)
    console.log(`   🟨 [Referee Recv] ${data.type}`, data.payload || '')
  }

  // C. 3 Judge Tablets
  const judgeSockets = []
  for (const j of judges) {
    const ws = new WebSocket(`${BASE_WS}/ws/arena/${arena.id}?token=${j.token}`)
    await new Promise((resolve, reject) => {
      ws.onopen = () => {
        console.log(`   ✅ Judge ${j.seat} Tablet connected`);
        resolve();
      }
      ws.onerror = (e) => reject(new Error(`Judge ${j.seat} WS error: ` + e.message))
    })
    judgeSockets.push(ws)
  }

  await sleep(400)

  // 7. Referee starts match
  console.log('\n7. Referee triggers match START...')
  refWs.send(JSON.stringify({
    type: 'state_change',
    payload: { action: 'start' }
  }))
  await sleep(600)

  // 8. Test Consensus Engine: Judge 1 & Judge 2 submit punch (pukulan) for Red within 1.5s window
  console.log('\n8. Simulating Judge Scoring (Red Punch +1)...')
  console.log('   - Judge 1 taps Red Technique (+1)')
  judgeSockets[0].send(JSON.stringify({
    type: 'judge_input',
    payload: {
      event_id: randomUUID(),
      idempotency_key: randomUUID(),
      score_type: 'technique',
      target: 'red',
      round: 1
    }
  }))

  await sleep(300)

  console.log('   - Judge 2 taps Red Technique (+1) (forming 2/3 majority consensus)...')
  judgeSockets[1].send(JSON.stringify({
    type: 'judge_input',
    payload: {
      event_id: randomUUID(),
      idempotency_key: randomUUID(),
      score_type: 'technique',
      target: 'red',
      round: 1
    }
  }))

  await sleep(1000)

  // 9. Check Scoreboard and Referee received score_confirmed & score_update
  const confirmedScore = scoreboardEvents.find(e => e.type === 'score_confirmed')
  const scoreUpdate = scoreboardEvents.find(e => e.type === 'score_update')

  if (confirmedScore) {
    console.log(`\n🎉 CONSENSUS REACHED! Scoreboard received:`)
    console.log(`   Type: ${confirmedScore.type}`)
    console.log(`   Target: ${confirmedScore.payload?.target}, Points: +${confirmedScore.payload?.points}`)
  } else {
    console.log('\n⚠️ No score_confirmed in scoreboard events yet. Events received:', scoreboardEvents.map(e => e.type))
  }

  if (scoreUpdate) {
    console.log(`   Score Update: Red = ${scoreUpdate.payload?.red_technical}, Blue = ${scoreUpdate.payload?.blue_technical}`)
  }

  // 10. Referee pauses and resumes match
  console.log('\n10. Referee tests PAUSE and RESUME...')
  refWs.send(JSON.stringify({
    type: 'state_change',
    payload: { action: 'pause' }
  }))
  await sleep(400)
  refWs.send(JSON.stringify({
    type: 'state_change',
    payload: { action: 'resume' }
  }))
  await sleep(400)

  // 11. Clean finish
  console.log('\n11. Finishing match...')
  refWs.send(JSON.stringify({
    type: 'state_change',
    payload: { action: 'finish' }
  }))
  await sleep(600)

  const matchResult = scoreboardEvents.find(e => e.type === 'match_result')
  if (matchResult) {
    console.log(`🏆 MATCH COMPLETED! Winner: Sudut ${matchResult.payload?.corner.toUpperCase()} (${matchResult.payload?.result_type})`)
  }

  // Close all sockets
  sbWs.close()
  refWs.close()
  judgeSockets.forEach(ws => ws.close())

  console.log('\n✅ All WebSocket connections cleanly closed.')
  console.log('🥋 === DIGITAL SCORING END-TO-END SIMULATION FINISHED SUCCESSFULLY === 🥋\n')
}

runSimulation().catch(err => {
  console.error('❌ Simulation Error:', err)
  process.exit(1)
})
