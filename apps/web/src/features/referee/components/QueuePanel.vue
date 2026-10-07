<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import type { RefereeConsole } from '../composables/useRefereeConsole'
import { bracketCategories, QUEUE_STATE_LABEL } from '../lib/queue'
import type { BracketSlotRecord } from '@/features/tournament/api/bracketService'
import TournamentTreeBracket, {
  type BracketRoundDisplay,
  type MatchCard,
} from '@/features/tournament/components/TournamentTreeBracket.vue'

const props = defineProps<{ c: RefereeConsole }>()
const emit = defineEmits<{ goto: [tab: string] }>()

const categories = computed(() => bracketCategories(props.c.participants))
const selectedKey = ref('')

const selected = computed(() => categories.value.find((x) => x.key === selectedKey.value) ?? null)

watch(
  categories,
  (list) => {
    if (!selectedKey.value && list.length) selectedKey.value = list[0].key
  },
  { immediate: true },
)

watch(
  selected,
  (cat) => {
    if (cat) void props.c.loadBracket(cat.weightClass, cat.gender)
  },
  { immediate: true },
)

// Group slots by round
const slotsByRound = computed(() => {
  const slots: BracketSlotRecord[] = props.c.bracketSlots || []
  const grouped: Record<number, BracketSlotRecord[]> = {}
  for (const s of slots) {
    if (!grouped[s.round]) grouped[s.round] = []
    grouped[s.round].push(s)
  }
  for (const r in grouped) {
    grouped[r].sort((a, b) => a.slot_index - b.slot_index)
  }
  return grouped
})

const maxRound = computed(() => {
  const keys = Object.keys(slotsByRound.value).map(Number)
  return keys.length > 0 ? Math.max(...keys) : 0
})

function getRoundTitle(roundNum: number, maxR: number): string {
  const dist = maxR - roundNum
  if (dist === 0) return 'BABAK FINAL'
  if (dist === 1) return 'BABAK SEMIFINAL'
  if (dist === 2) return 'BABAK PEREMPAT FINAL'
  if (dist === 3) return 'BABAK 16 BESAR'
  return `BABAK ${Math.pow(2, dist + 1)} BESAR`
}

function buildMatchesForRound(roundNum: number): MatchCard[] {
  const slots = slotsByRound.value[roundNum] || []
  if (slots.length === 0) return []

  const result: MatchCard[] = []
  for (let i = 0; i < slots.length; i += 2) {
    const slotRed = slots[i]
    const slotBlue = slots[i + 1]
    const pairIndex = Math.floor(i / 2) + 1

    const compRed = slotRed?.participant_id ? props.c.competitors.get(slotRed.participant_id) : null
    const compBlue = slotBlue?.participant_id ? props.c.competitors.get(slotBlue.participant_id) : null

    const hasRedBye = slotRed?.is_bye
    const hasBlueBye = slotBlue?.is_bye

    const isFinished = !!(slotRed?.winner_id || slotBlue?.winner_id)
    const redWinner = !!slotRed?.winner_id && slotRed.winner_id === slotRed.participant_id
    const blueWinner = !!slotBlue?.winner_id && slotBlue.winner_id === slotBlue.participant_id

    // Check if match corresponds to an active or queued match in props.c.queue
    const matchedQueue = props.c.queue.find(
      (q) =>
        (slotRed?.id && q.match.bracket_slot_id === slotRed.id) ||
        (slotBlue?.id && q.match.bracket_slot_id === slotBlue.id) ||
        (compRed?.id && q.match.athlete_red_id === compRed.id && compBlue?.id && q.match.athlete_blue_id === compBlue.id),
    )

    const isLive = matchedQueue?.state === 'loaded' || (!!matchedQueue && matchedQueue.match.id === props.c.matchId)
    const arenaName = props.c.arenaName || 'GELANGGANG'

    result.push({
      id: `slot-${slotRed?.id || i}`,
      partaiNumber: matchedQueue?.number || pairIndex,
      arenaLabel: arenaName.toUpperCase(),
      status: isFinished ? 'selesai' : isLive ? 'live' : 'ready',
      statusBadge: isFinished ? 'SELESAI' : isLive ? 'LIVE' : hasRedBye || hasBlueBye ? 'BYE' : 'SIAP',
      statusVariant: isFinished ? 'neutral' : isLive ? 'emerald' : 'neutral',
      isLive,
      red: {
        name: compRed?.name || (hasRedBye ? 'BYE (Lolos Otomatis)' : 'Pemenang Babak Sebelumnya'),
        contingent: compRed?.team || '-',
        score: isFinished ? (redWinner ? 'W' : 'L') : '-',
        isWinner: redWinner,
      },
      blue: {
        name: compBlue?.name || (hasBlueBye ? 'BYE (Lolos Otomatis)' : 'Pemenang Babak Sebelumnya'),
        contingent: compBlue?.team || '-',
        score: isFinished ? (blueWinner ? 'W' : 'L') : '-',
        isWinner: blueWinner,
      },
    })
  }
  return result
}

const bracketRounds = computed<BracketRoundDisplay[]>(() => {
  if (maxRound.value === 0) return []

  const list: BracketRoundDisplay[] = []
  for (let r = 1; r <= maxRound.value; r++) {
    const title = getRoundTitle(r, maxRound.value)
    const matches = buildMatchesForRound(r)
    const dist = maxRound.value - r
    const badgeVariant: 'neutral' | 'blue' | 'red' | 'emerald' =
      dist === 0 ? 'emerald' : dist === 1 ? 'red' : dist === 2 ? 'blue' : 'neutral'

    list.push({
      roundNumber: r,
      title,
      badgeVariant,
      matches,
    })
  }
  return list
})

function handleSelectBracketMatch(m: MatchCard) {
  const queued = props.c.queue.find((q) => q.number === m.partaiNumber)
  if (queued && queued.canLoad) {
    props.c.loadMatch(queued.match.id)
  }
}
</script>

<template>
  <div>
    <!-- SECTION 1: ANTREAN PARTAI GELANGGANG -->
    <section class="ref-card ref-section">
      <div class="ref-card__head font-mono">
        <span>ANTREAN PARTAI GELANGGANG</span>
        <button class="ref-chip-btn" @click="c.refreshQueue()">Muat ulang</button>
      </div>

      <div v-if="!c.queue.length" class="ref-empty">
        <strong>Belum ada partai terjadwal untuk gelanggang ini</strong>
        <span
          >Jadwalkan partai ke gelanggang ini dari menu Jadwal Pertandingan pada panel
          operasional.</span
        >
      </div>

      <div v-else class="ref-queue font-mono">
        <div
          v-for="row in c.queue"
          :key="row.match.id"
          class="ref-queue-row"
          :class="{ 'ref-queue-row--loaded': row.state === 'loaded' }"
        >
          <span class="ref-queue-no">#{{ row.number }}</span>
          <div class="ref-queue-names">
            <span class="ref-red" :class="{ 'ref-win': row.winner === 'red' }">{{
              row.red?.name ?? 'Peserta tidak dikenal'
            }}</span>
            <span class="ref-blue" :class="{ 'ref-win': row.winner === 'blue' }">{{
              row.blue?.name ?? 'Peserta tidak dikenal'
            }}</span>
            <small>{{ row.red?.classLabel ?? '' }}</small>
          </div>
          <span class="ref-state" :class="`ref-state--${row.state}`">{{
            QUEUE_STATE_LABEL[row.state]
          }}</span>
          <div>
            <button
              v-if="row.state === 'loaded'"
              class="ref-btn"
              @click="emit('goto', 'kontrol-tanding')"
            >
              Buka Kontrol
            </button>
            <button
              v-else-if="row.state === 'waiting'"
              class="ref-btn ref-btn--primary"
              :disabled="!row.canLoad || c.busy"
              :title="
                row.canLoad
                  ? ''
                  : 'Selesaikan atau batalkan partai yang sedang di gelanggang terlebih dahulu'
              "
              @click="c.loadMatch(row.match.id)"
            >
              Muat ke Gelanggang
            </button>
            <button
              v-else-if="row.state === 'cancelled'"
              class="ref-btn"
              :disabled="c.busy"
              title="Kembalikan partai ke antrean dan ulang dari 0-0"
              @click="c.rescheduleMatch(row.match.id)"
            >
              Jadwalkan Ulang
            </button>
          </div>
        </div>
      </div>
    </section>

    <!-- SECTION 2: BAGAN PERTANDINGAN KELAS -->
    <section class="ref-card ref-section">
      <div class="ref-card__head font-mono">
        <span>BAGAN PERTANDINGAN RESMI</span>
        <select v-model="selectedKey" class="ref-select font-mono" :disabled="!categories.length">
          <option v-for="cat in categories" :key="cat.key" :value="cat.key">{{ cat.label }}</option>
        </select>
      </div>

      <div v-if="!categories.length" class="ref-card__sub font-mono">
        Belum ada peserta terverifikasi.
      </div>
      <div v-else-if="!bracketRounds.length" class="ref-card__sub font-mono">
        Bagan untuk kelas ini belum dibuat. Buat bagan dari menu Bagan Pertandingan pada panel
        operasional.
      </div>
      <div v-else class="ref-tree-wrapper">
        <TournamentTreeBracket
          :rounds="bracketRounds"
          :category-name="selected?.label || ''"
          @select-match="handleSelectBracketMatch"
        />
      </div>
    </section>
  </div>
</template>

<style scoped>
.ref-tree-wrapper {
  margin-top: 14px;
  width: 100%;
}
</style>
