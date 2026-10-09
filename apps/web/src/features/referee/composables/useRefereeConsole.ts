import { computed, onMounted, onUnmounted, reactive, ref, shallowRef, watch } from 'vue'
import { AppApiError, getErrorMessage } from '@/api/error'
import {
  useWebSocket,
  type WsMatchEvent,
  type WsMatchOutcome,
  type WsPenaltyEntry,
  type WsPenaltyKind,
  type WsTechniqueKind,
  type WsTiebreak,
  type WsOutcomeKind,
  type WsPresence,
  type WsRoundScore,
  type WsSyncResponse,
  type WsTarget,
} from '@/composables/useWebSocket'
import { useTimer } from '@/composables/useTimer'
import { DEFAULT_OUTCOMES, DEFAULT_TECHNIQUES } from '@/composables/techniques'
import { playGong } from '@/composables/useGong'
import { matchStatusFromState, type MatchStatus } from '@/composables/matchStatus'
import { arenaService } from '@/features/arena/api/arenaService'
import type { Arena } from '@/features/arena/types/arena.schema'
import { categoryService } from '@/features/categories/api/categoryService'
import type { Category } from '@/features/categories/types/category.schema'
import { competitionService } from '@/features/competitions/api/competitionService'
import { matchService, type MatchDTO } from '@/features/matches/api/matchService'
import {
  bracketService,
  type BracketSlotRecord,
  type ParticipantDTO,
} from '@/features/tournament/api/bracketService'
import { countTechniques, toLogEntry, type LogEntry, type RawMatchEvent } from '../lib/matchLog'
import { buildQueue, competitorMap, type Competitor } from '../lib/queue'

export interface RefereeConsoleOptions {
  arenaId: string
  compId: string
  token?: string
}

export type ToastKind = 'info' | 'success' | 'error'

export interface ConsoleToast {
  id: number
  kind: ToastKind
  message: string
}

const emptyRounds = (): WsRoundScore[] =>
  [1, 2, 3].map(() => ({ RedTechnical: 0, BlueTechnical: 0, RedPenalties: 0, BluePenalties: 0 }))

/** Pesan ramah untuk kode error yang dikirim server lewat WebSocket. */
const ERROR_TEXT: Record<string, string> = {
  FORBIDDEN: 'Akun ini tidak berhak melakukan aksi tersebut.',
  PENALTY_NOT_ALLOWED: 'Sanksi hanya dapat dijatuhkan saat pertandingan berlangsung.',
  PENALTY_ALREADY_APPLIED: 'Sanksi tersebut sudah dijatuhkan pada sudut ini.',
  NOTHING_TO_REVOKE: 'Tidak ada sanksi yang dapat dibatalkan pada sudut ini.',
  BAD_PAYLOAD: 'Perintah tidak valid.',
}

/**
 * Seluruh state dan aksi konsol wasit. Sumber kebenaran adalah server: status, timer, skor,
 * sanksi, dan kehadiran perangkat dibaca dari `sync_response` dan siaran WebSocket; aksi
 * dikirim lewat REST (kontrol pertandingan) atau WebSocket (sanksi, koreksi waktu, sinyal).
 */
export function useRefereeConsole(opts: RefereeConsoleOptions) {
  const ws = useWebSocket(opts.arenaId, opts.token)
  const timer = useTimer()

  // --- konteks ---
  const arenaName = ref('')
  const competitionName = ref('')
  const rulesetName = ref('')

  // --- pertandingan di gelanggang ---
  const matchId = ref<string | null>(null)
  const matchState = ref('')
  const status = ref<MatchStatus>('idle')
  const round = ref(1)
  const totalRounds = ref(3)
  const roundDurationMs = ref(120000)
  const roundScores = ref<WsRoundScore[]>(emptyRounds())
  const penalties = ref<WsPenaltyEntry[]>([])
  const penaltyCatalog = ref<WsPenaltyKind[]>([])
  const techniqueCatalog = ref<WsTechniqueKind[]>(DEFAULT_TECHNIQUES)
  const judgeCount = ref(3)
  const tiebreak = ref<WsTiebreak | null>(null)
  const outcomeCatalog = ref<WsOutcomeKind[]>(DEFAULT_OUTCOMES)
  const medicActive = ref(false)
  const varActive = ref(false)
  const presence = ref<WsPresence | null>(null)
  const result = ref<WsMatchOutcome | null>(null)
  const lastConfirmed = ref<{ target: WsTarget; points: number } | null>(null)
  const tiePending = ref(false)
  const recovered = ref(false)

  // --- data pendukung (REST) ---
  const matchDto = shallowRef<MatchDTO | null>(null)
  const participants = ref<ParticipantDTO[]>([])
  const queueMatches = ref<MatchDTO[]>([])
  const events = shallowRef<RawMatchEvent[]>([])
  const bracketSlots = ref<BracketSlotRecord[]>([])
  const categories = ref<Category[]>([])
  const arenas = ref<Arena[]>([])
  const allMatches = ref<MatchDTO[]>([])
  let bracketCategoryId = ''
  const latencyMs = ref<number | null>(null)
  const busy = ref(false)
  const toasts = ref<ConsoleToast[]>([])

  const competitors = computed(() => competitorMap(participants.value))
  const hasMatch = computed(() => !!matchId.value)
  const isFinished = computed(() => status.value === 'finished')
  const isCancelled = computed(() => matchState.value === 'cancelled')
  const activeMatchId = computed(() => (matchId.value && !isFinished.value ? matchId.value : null))
  const redCompetitor = computed<Competitor | null>(() =>
    matchDto.value ? (competitors.value.get(matchDto.value.athlete_red_id) ?? null) : null,
  )
  const blueCompetitor = computed<Competitor | null>(() =>
    matchDto.value ? (competitors.value.get(matchDto.value.athlete_blue_id) ?? null) : null,
  )

  const redTotal = computed(() =>
    roundScores.value.reduce((s, r) => s + r.RedTechnical - r.RedPenalties, 0),
  )
  const blueTotal = computed(() =>
    roundScores.value.reduce((s, r) => s + r.BlueTechnical - r.BluePenalties, 0),
  )
  const isTied = computed(() => redTotal.value === blueTotal.value)

  const logEntries = computed<LogEntry[]>(() =>
    events.value.map(toLogEntry).filter((e): e is LogEntry => e !== null),
  )
  const techniques = computed(() =>
    countTechniques(
      events.value,
      techniqueCatalog.value.map((t) => t.type),
    ),
  )
  const queue = computed(() =>
    buildQueue(queueMatches.value, competitors.value, activeMatchId.value),
  )

  // --- aturan tombol (mengikuti mesin status di server) ---
  const inRound = computed(() => matchState.value.startsWith('round'))
  const canStart = computed(
    () => hasMatch.value && (status.value === 'idle' || status.value === 'ready'),
  )
  const canPause = computed(() => status.value === 'running')
  const canResume = computed(() => status.value === 'paused')
  const canFinish = computed(() => hasMatch.value && !isFinished.value)
  const canAdjustTime = computed(() => inRound.value)
  /** Pertandingan sudah dimulai (sedang ronde atau jeda antar ronde, belum selesai). */
  const matchStarted = computed(() => inRound.value || matchState.value.startsWith('break'))
  /** Apakah hasil khusus ini boleh dinyatakan sekarang (menurut `phase` pada ruleset). */
  const canDeclare = (outcome: WsOutcomeKind): boolean =>
    hasMatch.value &&
    !isFinished.value &&
    (outcome.phase === 'any' ||
      (outcome.phase === 'in_match' ? matchStarted.value : !matchStarted.value))
  const canResetRound = computed(() => status.value === 'paused')
  const canPenalize = computed(
    () =>
      hasMatch.value &&
      !isFinished.value &&
      round.value > 0 &&
      matchState.value !== 'idle' &&
      matchState.value !== 'ready',
  )
  const canSignal = computed(() => hasMatch.value)

  // --- toast ---
  let toastSeq = 0
  function notify(kind: ToastKind, message: string) {
    const id = ++toastSeq
    toasts.value = [...toasts.value, { id, kind, message }]
    setTimeout(() => {
      toasts.value = toasts.value.filter((t) => t.id !== id)
    }, 4500)
  }

  // --- menerapkan data server ---
  function applySnapshot(s: WsSyncResponse) {
    const previous = matchId.value
    if (s.match_id) matchId.value = s.match_id
    if (s.state !== undefined) matchState.value = s.state
    status.value = matchStatusFromState(s.state, s.timer_running) ?? status.value
    if (s.round) round.value = s.round
    if (s.total_rounds) totalRounds.value = s.total_rounds
    if (s.round_duration_ms) roundDurationMs.value = s.round_duration_ms
    if (s.ruleset_name) rulesetName.value = s.ruleset_name
    if (s.timer_remaining_ms !== undefined)
      timer.sync(s.timer_remaining_ms, s.timer_running ?? false, s.timer_ends_at_ms)
    if (s.scores) roundScores.value = s.scores.map((r) => ({ ...r }))
    if (s.penalties) penalties.value = s.penalties
    if (s.penalty_catalog) penaltyCatalog.value = s.penalty_catalog
    if (s.techniques?.length) techniqueCatalog.value = s.techniques
    if (s.judge_count) judgeCount.value = s.judge_count
    if (s.tiebreak) tiebreak.value = s.tiebreak
    if (s.outcomes?.length) outcomeCatalog.value = s.outcomes
    if (s.medic_active !== undefined) medicActive.value = s.medic_active
    if (s.var_active !== undefined) varActive.value = s.var_active
    if (s.presence) presence.value = s.presence
    tiePending.value = s.tie_pending ?? false
    const wasRecovered = recovered.value
    recovered.value = s.recovered ?? false
    // kejadian "server dimulai ulang" dicatat sebelum klien tersambung: ambil dari REST agar muncul di log
    if (recovered.value && !wasRecovered) void loadEvents()
    result.value = s.result ?? null
    if (s.match_id && s.match_id !== previous) void onMatchChanged()
  }

  async function onMatchChanged() {
    events.value = []
    lastConfirmed.value = null
    await Promise.all([loadMatchDetail(), loadEvents()])
  }

  async function loadMatchDetail() {
    if (!matchId.value || !opts.compId) return
    try {
      matchDto.value = await matchService.getMatch(opts.compId, matchId.value)
    } catch (err) {
      notify('error', getErrorMessage(err, 'Gagal memuat data partai.'))
    }
  }

  function mergeEvents(incoming: RawMatchEvent[]) {
    const known = new Set(events.value.map((e) => e.id))
    const fresh = incoming.filter((e) => !known.has(e.id))
    if (fresh.length) events.value = [...events.value, ...fresh]
  }

  async function loadEvents() {
    if (!matchId.value || !opts.compId) return
    try {
      mergeEvents(await matchService.getEvents(opts.compId, matchId.value))
    } catch (err) {
      notify('error', getErrorMessage(err, 'Gagal memuat log pertandingan.'))
    }
  }

  async function loadContext() {
    if (!opts.compId) return
    const [arenaList, comp, people, cats] = await Promise.allSettled([
      arenaService.getArenas(opts.compId),
      competitionService.getCompetition(opts.compId),
      bracketService.getParticipants(opts.compId),
      categoryService.getCategories(opts.compId),
    ])
    if (arenaList.status === 'fulfilled') {
      arenas.value = arenaList.value
      arenaName.value = arenaList.value.find((a) => a.id === opts.arenaId)?.name ?? ''
    }
    if (cats.status === 'fulfilled') categories.value = cats.value
    if (comp.status === 'fulfilled') {
      competitionName.value = comp.value.name
      if (!rulesetName.value) rulesetName.value = comp.value.ruleset_name
    }
    if (people.status === 'fulfilled') participants.value = people.value
    await refreshQueue()
  }

  async function refreshQueue() {
    if (!opts.compId) return
    try {
      queueMatches.value = await matchService.getArenaQueue(opts.compId, opts.arenaId)
      if (bracketCategoryId) void loadCategoryBracket(bracketCategoryId) // bagan ikut mutakhir (pemenang maju)
    } catch (err) {
      notify('error', getErrorMessage(err, 'Gagal memuat antrean partai.'))
    }
  }

  /** Bagan satu kategori (semua gelanggang) beserta seluruh partai kejuaraan untuk label gelanggang. */
  async function loadCategoryBracket(categoryId: string) {
    if (!opts.compId || !categoryId) return
    bracketCategoryId = categoryId
    try {
      const [slots, matches] = await Promise.all([
        bracketService.getBracketTreeByCategory(opts.compId, categoryId),
        matchService.getMatches(opts.compId),
      ])
      if (bracketCategoryId !== categoryId) return // pilihan sudah berganti
      bracketSlots.value = slots
      allMatches.value = matches
    } catch (err) {
      bracketSlots.value = []
      notify('error', getErrorMessage(err, 'Gagal memuat bagan.'))
    }
  }

  function refreshBracket() {
    return loadCategoryBracket(bracketCategoryId)
  }

  // --- pesan WebSocket ---
  let pingSentAt: number | null = null

  ws.onMessage('sync_response', (p) => {
    if (pingSentAt !== null) {
      latencyMs.value = Math.max(1, Math.round(performance.now() - pingSentAt))
      pingSentAt = null
    }
    applySnapshot(p)
  })

  ws.onMessage('state_change', (p) => {
    if (p.state !== undefined) matchState.value = p.state
    if (p.round) round.value = p.round
    status.value = matchStatusFromState(p.state, p.timer_running) ?? status.value
    tiePending.value = p.tie_pending ?? false
    if (p.timer_remaining_ms !== undefined)
      timer.sync(
        p.timer_remaining_ms,
        p.timer_running ?? status.value === 'running',
        p.timer_ends_at_ms,
      )
    if (status.value === 'finished') timer.pause()
    if (status.value === 'running') recovered.value = false
  })

  // koreksi berkala selama timer berjalan (server menyiarkan tenggat yang sama)
  ws.onMessage('timer_sync', (p) => {
    if (p.timer_remaining_ms !== undefined)
      timer.sync(p.timer_remaining_ms, p.timer_running ?? false, p.timer_ends_at_ms)
  })

  ws.onMessage('score_update', (p) => {
    if (!p.round) return
    const row = roundScores.value[p.round - 1]
    if (!row) return
    if (p.red_technical !== undefined) row.RedTechnical = p.red_technical
    if (p.blue_technical !== undefined) row.BlueTechnical = p.blue_technical
    if (p.red_penalties !== undefined) row.RedPenalties = p.red_penalties
    if (p.blue_penalties !== undefined) row.BluePenalties = p.blue_penalties
    roundScores.value = [...roundScores.value]
  })

  ws.onMessage('score_confirmed', (p) => {
    if (p.target === 'red' || p.target === 'blue')
      lastConfirmed.value = { target: p.target, points: p.points }
  })

  ws.onMessage('penalties', (p) => {
    penalties.value = p.penalties
  })

  ws.onMessage('presence', (p) => {
    presence.value = p
  })

  ws.onMessage('arena_signal', (p) => {
    if (p.signal === 'medic') medicActive.value = p.active
    if (p.signal === 'var') varActive.value = p.active
    if (p.signal === 'gong') playGong()
  })

  ws.onMessage('match_event', (p: WsMatchEvent) => {
    mergeEvents([
      {
        id: String(p.id),
        event_type: p.event_type,
        payload: p.payload,
        judge_id: p.judge_id,
        created_at: p.created_at,
      },
    ])
  })

  ws.onMessage('match_result', (p) => {
    status.value = 'finished'
    timer.pause()
    result.value = { winner: '', corner: p.corner ?? '', result_type: p.result_type ?? '' }
    void refreshQueue()
    void loadMatchDetail()
  })

  ws.onMessage('error', (p) => {
    notify('error', ERROR_TEXT[p.code] ?? p.message)
  })

  // --- aksi ---
  function sendWs(type: string, payload: Record<string, unknown>): boolean {
    const sent = ws.sendMessage(type, payload)
    if (!sent) notify('error', 'Tidak terhubung ke server. Periksa jaringan lalu coba lagi.')
    return sent
  }

  async function runControl(action: () => Promise<unknown>, failure: string) {
    if (busy.value) return
    busy.value = true
    try {
      await action()
    } catch (err) {
      notify('error', getErrorMessage(err, failure))
    } finally {
      busy.value = false
    }
  }

  const requireMatch = (): string | null => {
    if (!matchId.value || !opts.compId) {
      notify('error', 'Belum ada partai di gelanggang ini. Muat partai dari menu Bagan & Antrean.')
      return null
    }
    return matchId.value
  }

  const start = () =>
    runControl(async () => {
      const id = requireMatch()
      if (id) await matchService.startMatch(opts.compId, id)
    }, 'Gagal memulai pertandingan.')
  const pause = () =>
    runControl(async () => {
      const id = requireMatch()
      if (id) await matchService.pauseMatch(opts.compId, id)
    }, 'Gagal menjeda pertandingan.')
  const resume = () =>
    runControl(async () => {
      const id = requireMatch()
      if (id) await matchService.resumeMatch(opts.compId, id)
    }, 'Gagal melanjutkan pertandingan.')
  /** `winner` wajib saat skor seri (keputusan wasit). */
  const finish = (winner?: WsTarget, method?: string) =>
    runControl(async () => {
      const id = requireMatch()
      if (!id) return
      await matchService.finishMatch(opts.compId, id, winner, method)
      await refreshQueue()
    }, 'Gagal menyelesaikan pertandingan.')
  /**
   * Selesaikan saat skor seri tanpa pemenang: server menjalankan aturan seri otomatis
   * (sanksi paling sedikit, teknik terbanyak). Mengembalikan true bila masih seri dan wasit
   * harus menentukan pemenang secara manual.
   */
  async function finishByTiebreak(): Promise<boolean> {
    if (busy.value) return false
    const id = requireMatch()
    if (!id) return false
    busy.value = true
    try {
      await matchService.finishMatch(opts.compId, id)
      await refreshQueue()
      return false
    } catch (err) {
      if (err instanceof AppApiError && err.code === 'TIE_NEEDS_DECISION') return true
      notify('error', getErrorMessage(err, 'Gagal menyelesaikan pertandingan.'))
      return false
    } finally {
      busy.value = false
    }
  }
  /** Nyatakan hasil khusus (KO, TKO, WO, undur diri) dengan `winner` sebagai pemenang. */
  const declareOutcome = (outcome: string, winner: WsTarget, note?: string) =>
    runControl(async () => {
      const id = requireMatch()
      if (!id) return
      await matchService.declareOutcome(opts.compId, id, outcome, winner, note)
      await refreshQueue()
    }, 'Gagal menyatakan hasil pertandingan.')
  const cancelMatch = () =>
    runControl(async () => {
      const id = requireMatch()
      if (!id) return
      await matchService.cancelMatch(opts.compId, id)
      await refreshQueue()
    }, 'Gagal membatalkan pertandingan.')

  /** Kembalikan partai batal ke antrean (diulang dari 0-0); konsol dikosongkan bila partai itu sedang tampil. */
  const rescheduleMatch = (id: string) =>
    runControl(async () => {
      await matchService.rescheduleMatch(opts.compId, id)
      if (matchId.value === id) {
        matchId.value = null
        matchState.value = ''
        status.value = 'idle'
        round.value = 1
        roundScores.value = emptyRounds()
        penalties.value = []
        result.value = null
        tiePending.value = false
        lastConfirmed.value = null
        events.value = []
        matchDto.value = null
      }
      notify('success', 'Partai dikembalikan ke antrean dan dapat dimuat ulang dari 0-0.')
      await refreshQueue()
    }, 'Gagal menjadwalkan ulang partai.')

  function adjustTime(deltaSeconds: number) {
    sendWs('state_change', { action: 'adjust_time', delta_ms: deltaSeconds * 1000 })
  }
  const resetRound = () => sendWs('state_change', { action: 'reset_round' })
  const disqualify = (target: WsTarget) => sendWs('state_change', { action: 'disqualify', target })
  const applyPenalty = (target: WsTarget, type: string) => sendWs('penalty', { target, type })
  const revokePenalty = (target: WsTarget) => sendWs('penalty_revoke', { target })
  const ringGong = () => sendWs('arena_signal', { signal: 'gong' })
  const toggleMedic = () => sendWs('arena_signal', { signal: 'medic', active: !medicActive.value })
  const toggleVar = () => sendWs('arena_signal', { signal: 'var', active: !varActive.value })

  async function loadMatch(id: string) {
    await runControl(async () => {
      await matchService.loadArena(opts.compId, id, opts.arenaId)
      ws.sendMessage('sync_request', {})
      notify('success', 'Partai dimuat ke gelanggang.')
      await refreshQueue()
    }, 'Gagal memuat partai ke gelanggang.')
  }

  function ping() {
    pingSentAt = performance.now()
    ws.sendMessage('sync_request', {})
  }

  let pingTimer: ReturnType<typeof setInterval> | null = null
  watch(ws.isConnected, (connected) => {
    if (connected) ping()
  })
  onMounted(() => {
    void loadContext()
    pingTimer = setInterval(() => {
      if (ws.isConnected.value) ping()
    }, 10000)
  })
  onUnmounted(() => {
    if (pingTimer) clearInterval(pingTimer)
  })

  // reactive() membuka bungkus ref/computed agar panel membaca `c.status`, bukan `c.status.value`
  return reactive({
    // koneksi & konteks
    isConnected: ws.isConnected,
    latencyMs,
    arenaName,
    competitionName,
    rulesetName,
    // pertandingan
    matchId,
    matchState,
    status,
    round,
    totalRounds,
    roundDurationMs,
    timer,
    roundScores,
    redTotal,
    blueTotal,
    penalties,
    penaltyCatalog,
    techniqueCatalog,
    judgeCount,
    tiebreak,
    medicActive,
    varActive,
    presence,
    result,
    tiePending,
    recovered,
    isTied,
    lastConfirmed,
    matchDto,
    redCompetitor,
    blueCompetitor,
    // data pendukung
    participants,
    queue,
    events,
    logEntries,
    techniques,
    bracketSlots,
    arenaId: opts.arenaId,
    categories,
    arenas,
    allMatches,
    competitors,
    toasts,
    busy,
    // aturan tombol
    hasMatch,
    isFinished,
    isCancelled,
    canStart,
    canPause,
    canResume,
    canFinish,
    canAdjustTime,
    canResetRound,
    canPenalize,
    canSignal,
    // aksi
    start,
    pause,
    resume,
    finish,
    finishByTiebreak,
    declareOutcome,
    outcomeCatalog,
    canDeclare,
    cancelMatch,
    rescheduleMatch,
    adjustTime,
    resetRound,
    disqualify,
    applyPenalty,
    revokePenalty,
    ringGong,
    toggleMedic,
    toggleVar,
    loadMatch,
    refreshQueue,
    loadCategoryBracket,
    refreshBracket,
  })
}

export type RefereeConsole = ReturnType<typeof useRefereeConsole>
