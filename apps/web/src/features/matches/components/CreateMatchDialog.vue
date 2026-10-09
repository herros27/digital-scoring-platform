<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { getErrorMessage } from '@/api/error'
import { AppButton } from '@/components/ui'
import type { Arena } from '@/features/arena/types/arena.schema'
import type { Participant } from '@/features/athletes/types/athlete.schema'
import { useCreateMatchMutation } from '../api/matchQueries'
import type { MatchDTO } from '../api/matchService'
import { matchupWarnings, selectablePlayers } from '../lib/matchDetail'
import { createMatchSchema } from '../types/match.schema'

const props = defineProps<{
  open: boolean
  competitionId: string
  participants: Participant[]
  arenas: Arena[]
}>()
const emit = defineEmits<{ (e: 'close'): void; (e: 'created', match: MatchDTO): void }>()

const compIdRef = computed(() => props.competitionId)
const createMutation = useCreateMatchMutation(compIdRef)

const arenaId = ref('')
const redId = ref('')
const blueId = ref('')
const error = ref('')

watch(
  () => props.open,
  (isOpen) => {
    if (!isOpen) return
    arenaId.value = props.arenas[0]?.id ?? ''
    redId.value = ''
    blueId.value = ''
    error.value = ''
  },
)

const players = computed(() => selectablePlayers(props.participants))
const byId = computed(() => new Map(players.value.map((p) => [p.id, p])))
const warnings = computed(() =>
  matchupWarnings(byId.value.get(redId.value), byId.value.get(blueId.value)),
)

const label = (p: Participant) =>
  `${p.name} — ${p.club || 'Mandiri'} · ${p.category_name || `Kelas ${p.weight_class}`}`

async function submit() {
  error.value = ''
  const parsed = createMatchSchema.safeParse({
    arena_id: arenaId.value,
    athlete_red_id: redId.value,
    athlete_blue_id: blueId.value,
    current_round: 1,
  })
  if (!parsed.success) {
    error.value = parsed.error.issues[0]?.message || 'Data tidak valid'
    return
  }
  if (redId.value === blueId.value) {
    error.value = 'Atlet merah dan biru tidak boleh sama'
    return
  }
  try {
    const created = await createMutation.mutateAsync(parsed.data)
    emit('created', created)
    emit('close')
  } catch (err) {
    error.value = getErrorMessage(err, 'Partai tidak dapat dibuat')
  }
}
</script>

<template>
  <div
    v-if="open"
    class="cmd-overlay"
    role="dialog"
    aria-modal="true"
    aria-labelledby="cmd-title"
    data-test="create-match-dialog"
    @click.self="emit('close')"
  >
    <form class="cmd-modal" @submit.prevent="submit">
      <header class="cmd-header">
        <div>
          <div class="cmd-tag font-mono">PARTAI MANUAL</div>
          <h3 id="cmd-title" class="cmd-title">Tambah Partai</h3>
        </div>
        <button type="button" class="cmd-close" aria-label="Tutup" @click="emit('close')">✕</button>
      </header>
      <div class="cmd-body">
        <p class="cmd-help">
          Untuk partai di luar bagan, mis. eksibisi atau partai tambahan. Hanya atlet yang sudah
          terverifikasi yang dapat dipertandingkan. Partai bagan dibuat otomatis dari menu Bagan.
        </p>

        <label class="cmd-field">
          <span class="cmd-label font-mono">GELANGGANG</span>
          <select v-model="arenaId" class="cmd-input font-mono" data-test="match-arena">
            <option value="" disabled>Pilih gelanggang</option>
            <option v-for="a in arenas" :key="a.id" :value="a.id">{{ a.name }}</option>
          </select>
        </label>

        <label class="cmd-field">
          <span class="cmd-label font-mono cmd-label--red">SUDUT MERAH</span>
          <select v-model="redId" class="cmd-input font-mono" data-test="match-red">
            <option value="" disabled>Pilih atlet</option>
            <option v-for="p in players" :key="p.id" :value="p.id" :disabled="p.id === blueId">
              {{ label(p) }}
            </option>
          </select>
        </label>

        <label class="cmd-field">
          <span class="cmd-label font-mono cmd-label--blue">SUDUT BIRU</span>
          <select v-model="blueId" class="cmd-input font-mono" data-test="match-blue">
            <option value="" disabled>Pilih atlet</option>
            <option v-for="p in players" :key="p.id" :value="p.id" :disabled="p.id === redId">
              {{ label(p) }}
            </option>
          </select>
        </label>

        <p v-if="players.length < 2" class="cmd-warn" data-test="match-few-players">
          Belum ada cukup atlet terverifikasi (minimal 2) di kejuaraan ini.
        </p>
        <ul v-if="warnings.length" class="cmd-warn" data-test="match-warnings">
          <li v-for="w in warnings" :key="w">{{ w }}</li>
        </ul>
        <p v-if="error" class="cmd-error" role="alert" data-test="match-error">{{ error }}</p>

        <footer class="cmd-footer font-mono">
          <AppButton type="button" variant="secondary" size="md" @click="emit('close')">
            Batal
          </AppButton>
          <AppButton
            type="submit"
            variant="emerald"
            size="md"
            :loading="createMutation.isPending.value"
            data-test="match-save"
          >
            Buat Partai
          </AppButton>
        </footer>
      </div>
    </form>
  </div>
</template>

<style scoped>
.cmd-overlay {
  position: fixed;
  inset: 0;
  z-index: 999;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 16px;
  background: rgba(0, 0, 0, 0.75);
  backdrop-filter: blur(4px);
}
.cmd-modal {
  width: 100%;
  max-width: 520px;
  background: var(--bg-card, #131d33);
  border: 1px solid var(--border-card, #334155);
  border-radius: 12px;
  box-shadow: 0 20px 40px rgba(0, 0, 0, 0.6);
  overflow: hidden;
}
.cmd-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  padding: 18px 24px;
  border-bottom: 1px solid var(--border-card, #1e293b);
}
.cmd-tag {
  font-size: 0.6875rem;
  font-weight: 700;
  letter-spacing: 0.08em;
  color: var(--color-emerald-400, #34d399);
  margin-bottom: 4px;
}
.cmd-title {
  font-size: 1.125rem;
  font-weight: 800;
  color: #fff;
}
.cmd-close {
  background: none;
  border: none;
  color: var(--color-neutral-400, #94a3b8);
  cursor: pointer;
  font-size: 1.25rem;
}
.cmd-body {
  padding: 20px 24px;
  display: flex;
  flex-direction: column;
  gap: 16px;
}
.cmd-help {
  font-size: 0.8125rem;
  color: var(--color-neutral-400, #94a3b8);
}
.cmd-field {
  display: flex;
  flex-direction: column;
  gap: 6px;
}
.cmd-label {
  font-size: 0.6875rem;
  font-weight: 700;
  letter-spacing: 0.06em;
  color: var(--color-neutral-300, #cbd5e1);
}
.cmd-label--red {
  color: #fca5a5;
}
.cmd-label--blue {
  color: #93c5fd;
}
.cmd-input {
  width: 100%;
  padding: 10px 14px;
  background: rgba(15, 23, 42, 0.9);
  border: 1px solid var(--border-card, #334155);
  border-radius: 6px;
  color: #fff;
  font-size: 0.8125rem;
  outline: none;
  box-sizing: border-box;
}
.cmd-input:focus {
  border-color: var(--color-emerald-400, #34d399);
}
.cmd-warn {
  font-size: 0.8125rem;
  color: #fbbf24;
  padding-left: 0;
  list-style: none;
  margin: 0;
}
.cmd-error {
  color: var(--color-primary-500, #ef4444);
  font-size: 0.8125rem;
}
.cmd-footer {
  display: flex;
  justify-content: flex-end;
  gap: 10px;
}
</style>
