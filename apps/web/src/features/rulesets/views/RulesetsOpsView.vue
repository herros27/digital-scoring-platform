<script setup lang="ts">
import { computed, ref } from 'vue'
import { getErrorMessage } from '@/api/error'
import {
  useDeleteRulesetMutation,
  useRulesetsQuery,
  useSaveRulesetMutation,
} from '../api/rulesetQueries'
import RulesetEditor from '../components/RulesetEditor.vue'
import {
  copyDraft,
  draftFromRuleset,
  newDraft,
  summarize,
  type RulesetDraft,
} from '../lib/rulesetDraft'
import type { RulesetDTO, RulesetPayload } from '../types'

const { data, isLoading, error, refetch } = useRulesetsQuery()
const saveMutation = useSaveRulesetMutation()
const deleteMutation = useDeleteRulesetMutation()

const rulesets = computed(() => data.value ?? [])
const editing = ref<RulesetDraft | null>(null)
const serverError = ref('')
const confirmDeleteId = ref<string | null>(null)
const toast = ref<{ kind: 'success' | 'danger'; text: string } | null>(null)

function notify(kind: 'success' | 'danger', text: string) {
  toast.value = { kind, text }
  setTimeout(() => {
    if (toast.value?.text === text) toast.value = null
  }, 4000)
}

function openNew() {
  serverError.value = ''
  editing.value = newDraft()
}
function openEdit(r: RulesetDTO) {
  serverError.value = ''
  editing.value = draftFromRuleset(r)
}
function openCopy(r: RulesetDTO) {
  serverError.value = ''
  editing.value = copyDraft(draftFromRuleset(r))
}

async function save(payload: RulesetPayload, id?: string) {
  serverError.value = ''
  try {
    await saveMutation.mutateAsync({ id, payload })
    editing.value = null
    notify('success', id ? 'Ruleset diperbarui.' : 'Ruleset dibuat.')
  } catch (err) {
    serverError.value = getErrorMessage(err, 'Gagal menyimpan ruleset.')
  }
}

async function remove(r: RulesetDTO) {
  if (confirmDeleteId.value !== r.id) {
    confirmDeleteId.value = r.id
    return
  }
  confirmDeleteId.value = null
  try {
    await deleteMutation.mutateAsync(r.id)
    notify('success', `Ruleset "${r.name}" dihapus.`)
  } catch (err) {
    notify('danger', getErrorMessage(err, 'Gagal menghapus ruleset.'))
  }
}

const STATUS_LABEL: Record<string, string> = { active: 'AKTIF', draft: 'DRAF', archived: 'ARSIP' }
</script>

<template>
  <section class="rs-ops">
    <header class="rs-ops__head">
      <div>
        <h2>Regulasi &amp; Ruleset</h2>
        <p>
          Atur nilai teknik, sanksi, konsensus juri, aturan seri, dan hasil khusus per regulasi.
          Pilih ruleset saat membuat atau mengubah kejuaraan.
        </p>
      </div>
      <button class="rs-btn rs-btn--primary" data-test="new-ruleset" @click="openNew">
        + Ruleset Baru
      </button>
    </header>

    <p v-if="isLoading" class="rs-state">Memuat ruleset…</p>
    <p v-else-if="error" class="rs-state rs-state--error">
      Gagal memuat ruleset. <button class="rs-link" @click="refetch()">Coba lagi</button>
    </p>
    <p v-else-if="!rulesets.length" class="rs-state">Belum ada ruleset.</p>

    <ul class="rs-list">
      <li v-for="r in rulesets" :key="r.id" class="rs-card" data-test="ruleset-card">
        <div class="rs-card__top">
          <strong class="rs-card__name">{{ r.name }}</strong>
          <span class="rs-card__ver font-mono">v{{ r.version }}</span>
          <span class="rs-badge" :class="`rs-badge--${r.status}`">{{
            STATUS_LABEL[r.status] ?? r.status
          }}</span>
        </div>
        <p class="rs-card__sum font-mono">{{ summarize(r) }}</p>
        <div class="rs-card__actions">
          <button class="rs-btn" data-test="edit" @click="openEdit(r)">Ubah</button>
          <button class="rs-btn" data-test="copy" @click="openCopy(r)">Duplikat</button>
          <button class="rs-btn rs-btn--danger" data-test="delete" @click="remove(r)">
            {{ confirmDeleteId === r.id ? 'Yakin hapus?' : 'Hapus' }}
          </button>
        </div>
      </li>
    </ul>

    <RulesetEditor
      v-if="editing"
      :draft="editing"
      :saving="saveMutation.isPending.value"
      :server-error="serverError"
      @save="save"
      @cancel="editing = null"
    />

    <div v-if="toast" class="rs-toast" :class="`rs-toast--${toast.kind}`" role="status">
      {{ toast.text }}
    </div>
  </section>
</template>

<style scoped>
.rs-ops {
  display: flex;
  flex-direction: column;
  gap: 16px;
  color: var(--text-main, #f8fafc);
}
.rs-ops__head {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  gap: 16px;
  padding: 18px 24px;
  background: var(--bg-card, #131d33);
  border: 1px solid var(--border-card, #1e293b);
  border-radius: var(--radius-lg, 12px);
}
.rs-ops__head h2 {
  margin: 0 0 4px;
  font-size: 1.125rem;
}
.rs-ops__head p {
  margin: 0;
  font-size: 0.8125rem;
  color: var(--color-neutral-400, #94a3b8);
  max-width: 62ch;
}
.rs-list {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(320px, 1fr));
  gap: 14px;
  margin: 0;
  padding: 0;
  list-style: none;
}
.rs-card {
  display: flex;
  flex-direction: column;
  gap: 8px;
  padding: 16px;
  background: var(--bg-card, #131d33);
  border: 1px solid var(--border-card, #1e293b);
  border-radius: var(--radius-lg, 12px);
}
.rs-card__top {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
}
.rs-card__name {
  font-size: 1rem;
}
.rs-card__ver {
  font-size: 0.75rem;
  color: var(--color-neutral-400, #94a3b8);
}
.rs-card__sum {
  margin: 0;
  font-size: 0.75rem;
  color: var(--color-neutral-300, #cbd5e1);
}
.rs-card__actions {
  display: flex;
  gap: 8px;
  margin-top: 4px;
}
.rs-badge {
  margin-left: auto;
  padding: 2px 8px;
  border-radius: 999px;
  font-size: 0.6875rem;
  font-weight: 700;
  background: rgba(148, 163, 184, 0.15);
}
.rs-badge--active {
  background: rgba(16, 185, 129, 0.18);
  color: #6ee7b7;
}
.rs-btn {
  padding: 8px 14px;
  border-radius: var(--radius-xs, 6px);
  border: 1px solid var(--border-card, #334155);
  background: rgba(30, 41, 59, 0.8);
  color: var(--text-main, #f8fafc);
  font-size: 0.8125rem;
  font-weight: 600;
  cursor: pointer;
}
.rs-btn--primary {
  background: var(--color-emerald-600, #059669);
  border-color: var(--color-emerald-500, #10b981);
}
.rs-btn--danger {
  color: #fca5a5;
  border-color: rgba(220, 38, 38, 0.5);
}
.rs-state {
  padding: 20px;
  text-align: center;
  color: var(--color-neutral-400, #94a3b8);
  border: 1px dashed var(--border-card, #334155);
  border-radius: var(--radius-lg, 12px);
}
.rs-state--error {
  color: #fca5a5;
}
.rs-link {
  background: none;
  border: 0;
  color: inherit;
  text-decoration: underline;
  cursor: pointer;
}
.rs-toast {
  position: fixed;
  right: 24px;
  bottom: 24px;
  z-index: 1000;
  padding: 12px 18px;
  border-radius: var(--radius-md, 8px);
  background: var(--bg-card, #131d33);
  border: 1px solid var(--border-card, #334155);
  border-left-width: 4px;
  box-shadow: 0 10px 30px rgba(0, 0, 0, 0.45);
}
.rs-toast--success {
  border-left-color: var(--color-emerald-500, #10b981);
}
.rs-toast--danger {
  border-left-color: var(--color-primary-600, #dc2626);
}
</style>
