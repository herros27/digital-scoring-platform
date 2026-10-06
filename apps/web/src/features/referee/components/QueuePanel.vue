<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import type { RefereeConsole } from '../composables/useRefereeConsole'
import { bracketCategories, groupBracket, QUEUE_STATE_LABEL, roundLabel } from '../lib/queue'

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

const rounds = computed(() => groupBracket(props.c.bracketSlots, props.c.competitors))
const lastRound = computed(() => Math.max(1, ...rounds.value.map((r) => r.round)))
</script>

<template>
  <div>
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

    <section class="ref-card ref-section">
      <div class="ref-card__head font-mono">
        <span>BAGAN PERTANDINGAN</span>
        <select v-model="selectedKey" class="ref-select font-mono" :disabled="!categories.length">
          <option v-for="cat in categories" :key="cat.key" :value="cat.key">{{ cat.label }}</option>
        </select>
      </div>

      <div v-if="!categories.length" class="ref-card__sub font-mono">
        Belum ada peserta terverifikasi.
      </div>
      <div v-else-if="!rounds.length" class="ref-card__sub font-mono">
        Bagan untuk kelas ini belum dibuat. Buat bagan dari menu Bagan Pertandingan pada panel
        operasional.
      </div>
      <div v-else class="ref-bracket font-mono">
        <div v-for="r in rounds" :key="r.round" class="ref-bracket-col">
          <div class="ref-bracket-title">{{ roundLabel(r.round, lastRound) }}</div>
          <div
            v-for="s in r.slots"
            :key="s.id"
            class="ref-bracket-slot"
            :class="{
              'ref-bracket-slot--empty': !s.competitor,
              'ref-bracket-slot--winner': !!s.winner_id && s.winner_id === s.participant_id,
            }"
          >
            <template v-if="s.competitor">
              {{ s.competitor.name }}
              <small>{{ s.competitor.team }}</small>
            </template>
            <template v-else>{{ s.is_bye ? 'BYE' : 'Menunggu pemenang' }}</template>
          </div>
        </div>
      </div>
    </section>
  </div>
</template>
