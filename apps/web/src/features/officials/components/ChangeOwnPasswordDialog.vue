<script setup lang="ts">
import { ref, watch } from 'vue'
import { getErrorMessage } from '@/api/error'
import { AppButton } from '@/components/ui'
import { useChangeOwnPasswordMutation } from '../api/officialQueries'
import { changeOwnPasswordSchema } from '../types/official.schema'

const props = defineProps<{ open: boolean }>()
const emit = defineEmits<{ (e: 'close'): void; (e: 'changed'): void }>()

const current = ref('')
const next = ref('')
const confirm = ref('')
const error = ref('')
const mutation = useChangeOwnPasswordMutation()

watch(
  () => props.open,
  (isOpen) => {
    if (isOpen) {
      current.value = ''
      next.value = ''
      confirm.value = ''
      error.value = ''
    }
  },
)

async function submit() {
  error.value = ''
  const parsed = changeOwnPasswordSchema.safeParse({
    current_password: current.value,
    new_password: next.value,
    confirm: confirm.value,
  })
  if (!parsed.success) {
    error.value = parsed.error.issues[0]?.message || 'Data tidak valid'
    return
  }
  try {
    await mutation.mutateAsync({ current: current.value, next: next.value })
    emit('changed')
    emit('close')
  } catch (err) {
    error.value = getErrorMessage(err, 'Gagal mengganti password')
  }
}
</script>

<template>
  <div
    v-if="open"
    class="ofc-overlay"
    role="dialog"
    aria-modal="true"
    aria-labelledby="own-pw-title"
    data-test="own-password-dialog"
    @click.self="emit('close')"
  >
    <form class="ofc-modal" @submit.prevent="submit">
      <header class="ofc-modal__header">
        <div>
          <div class="ofc-modal__tag font-mono">KEAMANAN AKUN</div>
          <h3 id="own-pw-title" class="ofc-modal__title">Ganti Password Saya</h3>
        </div>
        <button type="button" class="ofc-modal__close" aria-label="Tutup" @click="emit('close')">
          ✕
        </button>
      </header>
      <div class="ofc-modal__body">
        <label class="ofc-field">
          <span class="ofc-label font-mono">PASSWORD LAMA</span>
          <input
            v-model="current"
            data-test="own-password-current"
            type="password"
            autocomplete="current-password"
            class="ofc-input font-mono"
          />
        </label>
        <label class="ofc-field">
          <span class="ofc-label font-mono">PASSWORD BARU (MIN. 8 KARAKTER)</span>
          <input
            v-model="next"
            data-test="own-password-new"
            type="password"
            autocomplete="new-password"
            class="ofc-input font-mono"
          />
        </label>
        <label class="ofc-field">
          <span class="ofc-label font-mono">ULANGI PASSWORD BARU</span>
          <input
            v-model="confirm"
            data-test="own-password-confirm"
            type="password"
            autocomplete="new-password"
            class="ofc-input font-mono"
          />
        </label>
        <p v-if="error" class="ofc-error" role="alert" data-test="own-password-error">
          {{ error }}
        </p>
        <footer class="ofc-modal__footer font-mono">
          <AppButton type="button" variant="secondary" size="md" @click="emit('close')">
            Batal
          </AppButton>
          <AppButton
            type="submit"
            variant="emerald"
            size="md"
            :loading="mutation.isPending.value"
            data-test="own-password-submit"
          >
            Simpan Password
          </AppButton>
        </footer>
      </div>
    </form>
  </div>
</template>

<style scoped>
.ofc-overlay {
  position: fixed;
  inset: 0;
  z-index: 1100;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 16px;
  background: rgba(0, 0, 0, 0.75);
  backdrop-filter: blur(4px);
}
.ofc-modal {
  width: 100%;
  max-width: 460px;
  background: var(--bg-card, #131d33);
  border: 1px solid var(--border-card, #334155);
  border-radius: 12px;
  box-shadow: 0 20px 40px rgba(0, 0, 0, 0.6);
  overflow: hidden;
}
.ofc-modal__header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  padding: 20px 24px;
  border-bottom: 1px solid var(--border-card, #1e293b);
}
.ofc-modal__tag {
  font-size: 0.6875rem;
  font-weight: 700;
  letter-spacing: 0.08em;
  color: var(--color-emerald-400, #34d399);
  margin-bottom: 4px;
}
.ofc-modal__title {
  font-size: 1.25rem;
  font-weight: 800;
  color: #fff;
}
.ofc-modal__close {
  background: none;
  border: none;
  color: var(--color-neutral-400, #94a3b8);
  cursor: pointer;
  font-size: 1.25rem;
}
.ofc-modal__body {
  padding: 20px 24px;
  display: flex;
  flex-direction: column;
  gap: 16px;
}
.ofc-field {
  display: flex;
  flex-direction: column;
  gap: 6px;
}
.ofc-label {
  font-size: 0.6875rem;
  font-weight: 700;
  letter-spacing: 0.06em;
  color: var(--color-neutral-300, #cbd5e1);
}
.ofc-input {
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
.ofc-input:focus {
  border-color: var(--color-emerald-400, #34d399);
}
.ofc-error {
  color: var(--color-primary-500, #ef4444);
  font-size: 0.8125rem;
}
.ofc-modal__footer {
  display: flex;
  justify-content: flex-end;
  gap: 10px;
}
</style>
