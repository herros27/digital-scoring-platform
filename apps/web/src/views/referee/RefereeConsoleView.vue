<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { getErrorMessage } from '@/api/error'
import { useAuthStore } from '@/stores/auth'
import { arenaService } from '@/features/arena/api/arenaService'
import type { Arena } from '@/features/arena/types/arena.schema'
import { competitionService } from '@/features/competitions/api/competitionService'
import type { Competition } from '@/features/competitions/types/competition.schema'
import RefereeConsole from '@/features/referee/components/RefereeConsole.vue'
import '@/features/referee/referee.css'

// Meja wasit PIN sudah terikat ke turnamen + gelanggang dari token. Admin/wasit login email
// belum memilih keduanya, jadi diminta memilih di sini (atau lewat ?comp_id=&arena_id=).
const route = useRoute()
const router = useRouter()
const authStore = useAuthStore()

const compId = computed(() => (route.query.comp_id as string) || authStore.competitionId || '')
const arenaId = computed(
  () =>
    (route.params.arena_id as string) ||
    (route.query.arena_id as string) ||
    authStore.arenaId ||
    '',
)

const competitions = ref<Competition[]>([])
const arenas = ref<Arena[]>([])
const loadError = ref('')
const pickComp = ref('')
const pickArena = ref('')

async function loadCompetitions() {
  try {
    competitions.value = (
      await competitionService.getCompetitions({ page: 1, limit: 50, status: 'all' })
    ).data
    pickComp.value =
      competitions.value.find((x) => x.status === 'active')?.id ?? competitions.value[0]?.id ?? ''
  } catch (err) {
    loadError.value = getErrorMessage(err, 'Gagal memuat daftar kejuaraan.')
  }
}

async function loadArenas(id: string) {
  try {
    arenas.value = await arenaService.getArenas(id)
    pickArena.value = arenas.value[0]?.id ?? ''
  } catch (err) {
    loadError.value = getErrorMessage(err, 'Gagal memuat daftar gelanggang.')
  }
}

onMounted(async () => {
  if (!compId.value) await loadCompetitions()
  const id = compId.value || pickComp.value
  if (id && !arenaId.value) await loadArenas(id)
})

async function onCompChange() {
  pickArena.value = ''
  arenas.value = []
  if (pickComp.value) await loadArenas(pickComp.value)
}

function enter() {
  void router.replace({
    path: route.path,
    query: {
      ...route.query,
      comp_id: compId.value || pickComp.value,
      arena_id: arenaId.value || pickArena.value,
    },
  })
}
</script>

<template>
  <RefereeConsole
    v-if="compId && arenaId"
    :key="`${compId}:${arenaId}`"
    :comp-id="compId"
    :arena-id="arenaId"
  />

  <div v-else class="ref-app">
    <div class="ref-card ref-picker">
      <h1>Pilih Gelanggang</h1>
      <p>
        Akun ini belum terikat ke satu gelanggang. Pilih kejuaraan dan gelanggang yang akan
        dikendalikan.
      </p>
      <p v-if="loadError" class="ref-toast ref-toast--error">{{ loadError }}</p>

      <select v-if="!compId" v-model="pickComp" class="ref-select font-mono" @change="onCompChange">
        <option v-for="comp in competitions" :key="comp.id" :value="comp.id">
          {{ comp.name }}
        </option>
      </select>
      <select
        v-if="!arenaId"
        v-model="pickArena"
        class="ref-select font-mono"
        :disabled="!arenas.length"
      >
        <option v-for="a in arenas" :key="a.id" :value="a.id">{{ a.name }}</option>
      </select>
      <p v-if="!arenaId && !arenas.length && !loadError" class="ref-card__sub font-mono">
        Belum ada gelanggang pada kejuaraan ini.
      </p>

      <button
        class="ref-btn ref-btn--primary"
        :disabled="!(compId || pickComp) || !(arenaId || pickArena)"
        @click="enter"
      >
        Masuk Konsol Meja
      </button>
    </div>
  </div>
</template>
