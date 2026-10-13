<script setup lang="ts">
import { computed, ref } from 'vue'
import { getErrorMessage } from '@/api/error'
import { AppButton } from '@/components/ui'
import { useContingentsQuery } from '@/features/athletes/api/athleteQueries'
import type { Contingent } from '@/features/athletes/types/athlete.schema'
import { useResetContingentPasswordMutation } from '../api/officialQueries'
import { resetPasswordSchema } from '../types/official.schema'

const emit = defineEmits<{ (e: 'toast', title: string, message: string): void }>()

const { data, isLoading, error, refetch } = useContingentsQuery()
const resetMutation = useResetContingentPasswordMutation()

const search = ref('')
const contingents = computed(() => {
  const q = search.value.trim().toLowerCase()
  const list = data.value || []
  if (!q) return list
  return list.filter((c) =>
    [c.name, c.pic_name ?? '', c.email ?? ''].some((v) => v.toLowerCase().includes(q)),
  )
})

const target = ref<Contingent | null>(null)
const password = ref('')
const formError = ref('')

function openReset(c: Contingent) {
  target.value = c
  password.value = ''
  formError.value = ''
}

async function submit() {
  if (!target.value) return
  formError.value = ''
  const parsed = resetPasswordSchema.safeParse({ password: password.value })
  if (!parsed.success) {
    formError.value = parsed.error.issues[0]?.message || 'Password tidak valid'
    return
  }
  try {
    await resetMutation.mutateAsync({ id: target.value.id, password: parsed.data.password })
    emit(
      'toast',
      'Password Direset',
      `Password ${target.value.name} diganti. Sampaikan langsung kepada manajer kontingen.`,
    )
    target.value = null
  } catch (err) {
    formError.value = getErrorMessage(err, 'Gagal mereset password')
  }
}
</script>

<template>
  <section data-test="contingent-accounts">
    <div class="cap-toolbar">
      <input
        v-model="search"
        type="search"
        class="cap-search font-mono"
        placeholder="Cari kontingen, PIC, atau email…"
        aria-label="Cari kontingen"
        data-test="contingent-search"
      />
      <span class="cap-count font-mono" data-test="contingent-count"
        >{{ contingents.length }} KONTINGEN</span
      >
    </div>
    <p class="cap-hint">
      Manajer kontingen mendaftar sendiri. Bila lupa password, tetapkan password baru di sini dan
      sampaikan langsung kepadanya.
    </p>

    <div v-if="isLoading" class="cap-state font-mono">Memuat kontingen…</div>
    <div v-else-if="error" class="cap-state cap-state--error" role="alert">
      {{ getErrorMessage(error, 'Gagal memuat kontingen') }}
      <AppButton variant="secondary" size="sm" @click="() => refetch()">Coba Lagi</AppButton>
    </div>
    <div v-else-if="contingents.length === 0" class="cap-state font-mono">
      Belum ada kontingen yang cocok.
    </div>
    <div v-else class="cap-table-wrap">
      <table class="cap-table">
        <thead>
          <tr class="font-mono">
            <th>KONTINGEN</th>
            <th>PIC</th>
            <th>EMAIL</th>
            <th>TELEPON</th>
            <th class="cap-th-actions">AKSI</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="c in contingents" :key="c.id" data-test="contingent-row" :data-email="c.email">
            <td class="cap-name">{{ c.name }}</td>
            <td>{{ c.pic_name || '-' }}</td>
            <td class="font-mono">{{ c.email || '-' }}</td>
            <td class="font-mono">{{ c.phone || '-' }}</td>
            <td class="cap-actions">
              <AppButton
                variant="secondary"
                size="sm"
                :aria-label="`Reset password ${c.name}`"
                data-test="contingent-reset"
                @click="openReset(c)"
              >
                Reset Password
              </AppButton>
            </td>
          </tr>
        </tbody>
      </table>
    </div>

    <div
      v-if="target"
      class="cap-overlay"
      role="dialog"
      aria-modal="true"
      aria-labelledby="cap-reset-title"
      data-test="contingent-reset-dialog"
      @click.self="target = null"
    >
      <form class="cap-modal" @submit.prevent="submit">
        <header class="cap-modal__header">
          <h3 id="cap-reset-title" class="cap-modal__title">Reset Password Kontingen</h3>
          <button type="button" class="cap-modal__close" aria-label="Tutup" @click="target = null">
            ✕
          </button>
        </header>
        <div class="cap-modal__body">
          <p class="cap-hint">
            Tetapkan password baru untuk <strong>{{ target.name }}</strong> ({{ target.email }}).
          </p>
          <label class="cap-field">
            <span class="cap-label font-mono">PASSWORD BARU (MIN. 8 KARAKTER)</span>
            <input
              v-model="password"
              type="password"
              autocomplete="new-password"
              class="cap-input font-mono"
              data-test="contingent-reset-password"
            />
          </label>
          <p v-if="formError" class="cap-error" role="alert" data-test="contingent-reset-error">
            {{ formError }}
          </p>
          <footer class="cap-modal__footer font-mono">
            <AppButton type="button" variant="secondary" size="md" @click="target = null">
              Batal
            </AppButton>
            <AppButton
              type="submit"
              variant="emerald"
              size="md"
              :loading="resetMutation.isPending.value"
              data-test="contingent-reset-submit"
            >
              Reset Password
            </AppButton>
          </footer>
        </div>
      </form>
    </div>
  </section>
</template>

<style scoped>
.cap-toolbar {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 8px;
}
.cap-search {
  flex: 1;
  max-width: 420px;
  padding: 9px 14px;
  background: rgba(15, 23, 42, 0.8);
  border: 1px solid var(--border-card, #334155);
  border-radius: 6px;
  color: #fff;
  font-size: 0.8125rem;
  outline: none;
}
.cap-count {
  font-size: 0.6875rem;
  letter-spacing: 0.06em;
  color: var(--color-neutral-400, #94a3b8);
}
.cap-hint {
  font-size: 0.8125rem;
  color: var(--color-neutral-400, #94a3b8);
  margin-bottom: 14px;
}
.cap-state {
  padding: 28px;
  text-align: center;
  color: var(--color-neutral-400, #94a3b8);
  border: 1px dashed var(--border-card, #334155);
  border-radius: 10px;
  font-size: 0.8125rem;
}
.cap-state--error {
  color: var(--color-primary-500, #ef4444);
  display: flex;
  gap: 12px;
  justify-content: center;
  align-items: center;
}
.cap-table-wrap {
  overflow-x: auto;
  background: var(--bg-card, #131d33);
  border: 1px solid var(--border-card, #334155);
  border-radius: 10px;
}
.cap-table {
  width: 100%;
  border-collapse: collapse;
  font-size: 0.8125rem;
}
.cap-table th {
  text-align: left;
  padding: 12px 16px;
  font-size: 0.6875rem;
  letter-spacing: 0.06em;
  color: var(--color-neutral-400, #94a3b8);
  border-bottom: 1px solid var(--border-card, #334155);
  white-space: nowrap;
}
.cap-table td {
  padding: 12px 16px;
  border-bottom: 1px solid rgba(51, 65, 85, 0.5);
  vertical-align: middle;
}
.cap-table tbody tr:last-child td {
  border-bottom: none;
}
.cap-th-actions {
  text-align: right;
}
.cap-name {
  font-weight: 700;
  color: #fff;
}
.cap-actions {
  text-align: right;
}
.cap-overlay {
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
.cap-modal {
  width: 100%;
  max-width: 460px;
  background: var(--bg-card, #131d33);
  border: 1px solid var(--border-card, #334155);
  border-radius: 12px;
  box-shadow: 0 20px 40px rgba(0, 0, 0, 0.6);
  overflow: hidden;
}
.cap-modal__header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 18px 24px;
  border-bottom: 1px solid var(--border-card, #1e293b);
}
.cap-modal__title {
  font-size: 1.125rem;
  font-weight: 800;
  color: #fff;
}
.cap-modal__close {
  background: none;
  border: none;
  color: var(--color-neutral-400, #94a3b8);
  cursor: pointer;
  font-size: 1.25rem;
}
.cap-modal__body {
  padding: 20px 24px;
  display: flex;
  flex-direction: column;
  gap: 16px;
}
.cap-field {
  display: flex;
  flex-direction: column;
  gap: 6px;
}
.cap-label {
  font-size: 0.6875rem;
  font-weight: 700;
  letter-spacing: 0.06em;
  color: var(--color-neutral-300, #cbd5e1);
}
.cap-input {
  width: 100%;
  padding: 10px 14px;
  background: rgba(15, 23, 42, 0.8);
  border: 1px solid var(--border-card, #334155);
  border-radius: 6px;
  color: #fff;
  font-size: 0.8125rem;
  outline: none;
  box-sizing: border-box;
}
.cap-input:focus {
  border-color: var(--color-emerald-400, #34d399);
}
.cap-error {
  color: var(--color-primary-500, #ef4444);
  font-size: 0.8125rem;
}
.cap-modal__footer {
  display: flex;
  justify-content: flex-end;
  gap: 10px;
}
</style>
