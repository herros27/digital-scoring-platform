<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import type { RefereeConsole } from '../composables/useRefereeConsole'
import { QUEUE_STATE_LABEL } from '../lib/queue'
import { buildRefereeRounds } from '../lib/bracketView'
import TournamentTreeBracket, {
  type BracketRoundDisplay,
  type MatchCard,
} from '@/features/tournament/components/TournamentTreeBracket.vue'

const props = defineProps<{ c: RefereeConsole }>()
const emit = defineEmits<{ goto: [tab: string] }>()

// Bagan milik kategori (bukan gelanggang): partainya bisa tersebar di beberapa gelanggang. Kategori yang punya partai
// di gelanggang ini diurutkan lebih dulu dan menjadi pilihan awal; kategori lain tetap bisa dilihat.
const ownCategoryIds = computed(() => {
  const ids = new Set<string>()
  for (const row of props.c.queue) {
    for (const pid of [row.match.athlete_red_id, row.match.athlete_blue_id]) {
      const cat = props.c.participants.find((p) => p.id === pid)?.category_id
      if (cat) ids.add(cat)
    }
  }
  return ids
})

const categoryOptions = computed(() =>
  props.c.categories
    .map((cat) => ({ id: cat.id, label: cat.name, own: ownCategoryIds.value.has(cat.id) }))
    .sort(
      (a, b) =>
        Number(b.own) - Number(a.own) ||
        a.label.localeCompare(b.label, undefined, { numeric: true }),
    ),
)

const selectedId = ref('')
const selected = computed(
  () => categoryOptions.value.find((x) => x.id === selectedId.value) ?? null,
)

watch(
  categoryOptions,
  (list) => {
    if (!list.some((x) => x.id === selectedId.value)) selectedId.value = list[0]?.id ?? ''
  },
  { immediate: true },
)

watch(
  selectedId,
  (id) => {
    if (id) void props.c.loadCategoryBracket(id)
  },
  { immediate: true },
)

const bracketRounds = computed<BracketRoundDisplay[]>(() =>
  buildRefereeRounds({
    slots: props.c.bracketSlots,
    competitors: props.c.competitors,
    matches: props.c.allMatches,
    arenas: props.c.arenas,
    arenaId: props.c.arenaId,
    queueNumbers: new Map(props.c.queue.map((q) => [q.match.id, q.number])),
    loadedMatchId: props.c.matchId,
  }),
)

function handleSelectBracketMatch(m: MatchCard) {
  if (!m.isOwnArena) return // partai gelanggang lain hanya untuk dilihat
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
        <select
          v-model="selectedId"
          class="ref-select font-mono"
          :disabled="!categoryOptions.length"
        >
          <option v-for="cat in categoryOptions" :key="cat.id" :value="cat.id">
            {{ cat.own ? '● ' : '' }}{{ cat.label }}
          </option>
        </select>
      </div>

      <div v-if="!categoryOptions.length" class="ref-card__sub font-mono">
        Belum ada kategori pada kejuaraan ini.
      </div>
      <div v-else-if="!bracketRounds.length" class="ref-card__sub font-mono">
        Bagan untuk kategori ini belum dibuat. Buat bagan dari menu Bagan Pertandingan pada panel
        operasional.
      </div>
      <div v-else class="ref-tree-wrapper">
        <p class="ref-card__sub font-mono" data-testid="ref-bracket-note">
          Bagan kategori mencakup semua gelanggang. Kartu bertepi biru adalah partai di
          {{ c.arenaName || 'gelanggang ini' }}.
        </p>
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
