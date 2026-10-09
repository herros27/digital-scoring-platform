<script setup lang="ts">
import { computed, ref } from 'vue'
import { getErrorMessage } from '@/api/error'
import { AppButton } from '@/components/ui'
import { useAuthStore } from '@/stores/auth'
import {
  useCreateOfficialMutation,
  useDeleteOfficialMutation,
  useOfficialsQuery,
  useResetPasswordMutation,
  useUpdateOfficialMutation,
} from '../api/officialQueries'
import { assignableRoles, canManageRole, isSelf } from '../lib/permissions'
import ChangeOwnPasswordDialog from '../components/ChangeOwnPasswordDialog.vue'
import {
  createOfficialSchema,
  resetPasswordSchema,
  ROLE_LABELS,
  updateOfficialSchema,
  type Official,
} from '../types/official.schema'

const authStore = useAuthStore()
const { data, isLoading, isFetching, error, refetch } = useOfficialsQuery()
const createMutation = useCreateOfficialMutation()
const updateMutation = useUpdateOfficialMutation()
const deleteMutation = useDeleteOfficialMutation()
const resetMutation = useResetPasswordMutation()

const myRole = computed(() => authStore.role || '')
const me = computed(() => ({
  userId: authStore.session?.userId,
  email: authStore.session?.email,
}))
const roleOptions = computed(() => assignableRoles(myRole.value))

const search = ref('')
const officials = computed(() => {
  const q = search.value.trim().toLowerCase()
  const list = data.value || []
  if (!q) return list
  return list.filter((o) =>
    [o.name, o.email, ROLE_LABELS[o.role] ?? o.role].some((v) => v.toLowerCase().includes(q)),
  )
})

const toast = ref<{ title: string; message: string; type: 'success' | 'danger' } | null>(null)
let toastTimer: ReturnType<typeof setTimeout> | null = null
function showToast(title: string, message: string, type: 'success' | 'danger' = 'success') {
  if (toastTimer) clearTimeout(toastTimer)
  toast.value = { title, message, type }
  toastTimer = setTimeout(() => (toast.value = null), 4000)
}

// --- Tambah / ubah ---------------------------------------------------------------------------------------------
const formOpen = ref(false)
const editing = ref<Official | null>(null)
const fName = ref('')
const fEmail = ref('')
const fPassword = ref('')
const fRole = ref('operator')
const formError = ref('')

function openCreate() {
  editing.value = null
  fName.value = ''
  fEmail.value = ''
  fPassword.value = ''
  fRole.value = roleOptions.value.includes('operator') ? 'operator' : (roleOptions.value[0] ?? '')
  formError.value = ''
  formOpen.value = true
}

function openEdit(o: Official) {
  editing.value = o
  fName.value = o.name
  fEmail.value = o.email
  fRole.value = o.role
  formError.value = ''
  formOpen.value = true
}

async function submitForm() {
  formError.value = ''
  try {
    if (editing.value) {
      const parsed = updateOfficialSchema.safeParse({ name: fName.value, role: fRole.value })
      if (!parsed.success) {
        formError.value = parsed.error.issues[0]?.message || 'Data tidak valid'
        return
      }
      await updateMutation.mutateAsync({ id: editing.value.id, ...parsed.data })
      showToast('Akun Diperbarui', `${parsed.data.name} berhasil diperbarui.`)
    } else {
      const parsed = createOfficialSchema.safeParse({
        name: fName.value,
        email: fEmail.value,
        password: fPassword.value,
        role: fRole.value,
      })
      if (!parsed.success) {
        formError.value = parsed.error.issues[0]?.message || 'Data tidak valid'
        return
      }
      await createMutation.mutateAsync(parsed.data)
      showToast(
        'Akun Dibuat',
        `${parsed.data.name} (${ROLE_LABELS[parsed.data.role]}) dapat masuk.`,
      )
    }
    formOpen.value = false
  } catch (err) {
    formError.value = getErrorMessage(err, 'Gagal menyimpan akun')
  }
}

// --- Reset password --------------------------------------------------------------------------------------------
const resetTarget = ref<Official | null>(null)
const resetPassword = ref('')
const resetError = ref('')

function openReset(o: Official) {
  resetTarget.value = o
  resetPassword.value = ''
  resetError.value = ''
}

async function submitReset() {
  if (!resetTarget.value) return
  resetError.value = ''
  const parsed = resetPasswordSchema.safeParse({ password: resetPassword.value })
  if (!parsed.success) {
    resetError.value = parsed.error.issues[0]?.message || 'Password tidak valid'
    return
  }
  try {
    await resetMutation.mutateAsync({ id: resetTarget.value.id, password: parsed.data.password })
    showToast(
      'Password Direset',
      `Password ${resetTarget.value.name} diganti. Sesi yang sedang berjalan tetap aktif sampai habis.`,
    )
    resetTarget.value = null
  } catch (err) {
    resetError.value = getErrorMessage(err, 'Gagal mereset password')
  }
}

// --- Hapus -----------------------------------------------------------------------------------------------------
const deleteTarget = ref<Official | null>(null)
const deleteError = ref('')

function openDelete(o: Official) {
  deleteTarget.value = o
  deleteError.value = ''
}

async function confirmDelete() {
  if (!deleteTarget.value) return
  deleteError.value = ''
  try {
    await deleteMutation.mutateAsync(deleteTarget.value.id)
    showToast('Akun Dihapus', `${deleteTarget.value.name} tidak dapat masuk lagi.`)
    deleteTarget.value = null
  } catch (err) {
    deleteError.value = getErrorMessage(err, 'Gagal menghapus akun')
  }
}

// --- Ganti password sendiri ------------------------------------------------------------------------------------
const ownPasswordOpen = ref(false)

const dateFmt = new Intl.DateTimeFormat('id-ID', { dateStyle: 'medium' })
const formatDate = (iso: string) => (iso ? dateFmt.format(new Date(iso)) : '-')
</script>

<template>
  <div class="off-ops" data-test="officials-view">
    <Transition name="off-fade">
      <div
        v-if="toast"
        class="off-toast"
        :class="`off-toast--${toast.type}`"
        role="status"
        data-test="officials-toast"
      >
        <div>
          <div class="off-toast__title">{{ toast.title }}</div>
          <div class="off-toast__msg">{{ toast.message }}</div>
        </div>
        <button class="off-toast__close" aria-label="Tutup" @click="toast = null">✕</button>
      </div>
    </Transition>

    <header class="off-header">
      <div>
        <div class="off-header__sub font-mono">KEAMANAN & AKSES</div>
        <h2 class="off-header__title">AKUN PETUGAS</h2>
        <p class="off-header__desc">
          Admin, Dewan Juri, Wasit, dan Operator yang masuk dengan email dan password.
          <template v-if="myRole === 'dewan'">
            Dewan Juri hanya dapat mengelola akun Wasit dan Operator.
          </template>
        </p>
      </div>
      <div class="off-header__actions font-mono">
        <AppButton variant="secondary" size="sm" :loading="isFetching" @click="() => refetch()">
          Segarkan
        </AppButton>
        <AppButton
          variant="secondary"
          size="sm"
          data-test="own-password-open"
          @click="ownPasswordOpen = true"
        >
          Ganti Password Saya
        </AppButton>
        <AppButton variant="emerald" size="sm" data-test="official-add" @click="openCreate">
          Tambah Petugas
        </AppButton>
      </div>
    </header>

    <div class="off-toolbar">
      <input
        v-model="search"
        type="search"
        class="off-search font-mono"
        placeholder="Cari nama, email, atau peran…"
        aria-label="Cari petugas"
        data-test="official-search"
      />
      <span class="off-count font-mono" data-test="official-count">
        {{ officials.length }} AKUN
      </span>
    </div>

    <div v-if="isLoading" class="off-state font-mono">Memuat akun petugas…</div>
    <div v-else-if="error" class="off-state off-state--error" role="alert">
      {{ getErrorMessage(error, 'Gagal memuat akun petugas') }}
      <AppButton variant="secondary" size="sm" @click="() => refetch()">Coba Lagi</AppButton>
    </div>
    <div v-else-if="officials.length === 0" class="off-state font-mono">
      Tidak ada akun yang cocok.
    </div>

    <div v-else class="off-table-wrap">
      <table class="off-table">
        <thead>
          <tr class="font-mono">
            <th>NAMA</th>
            <th>EMAIL</th>
            <th>PERAN</th>
            <th>DIBUAT</th>
            <th class="off-th-actions">AKSI</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="o in officials" :key="o.id" data-test="official-row" :data-email="o.email">
            <td class="off-name">
              {{ o.name }}
              <span v-if="isSelf(me, o)" class="off-self font-mono">ANDA</span>
            </td>
            <td class="font-mono">{{ o.email }}</td>
            <td>
              <span class="off-role font-mono" :class="`off-role--${o.role}`">
                {{ ROLE_LABELS[o.role] ?? o.role }}
              </span>
            </td>
            <td class="font-mono">{{ formatDate(o.created_at) }}</td>
            <td class="off-actions">
              <template v-if="canManageRole(myRole, o.role)">
                <AppButton
                  variant="secondary"
                  size="sm"
                  :aria-label="`Ubah ${o.name}`"
                  data-test="official-edit"
                  @click="openEdit(o)"
                >
                  Ubah
                </AppButton>
                <AppButton
                  variant="secondary"
                  size="sm"
                  :aria-label="`Reset password ${o.name}`"
                  data-test="official-reset"
                  @click="openReset(o)"
                >
                  Reset Password
                </AppButton>
                <AppButton
                  variant="danger"
                  size="sm"
                  :disabled="isSelf(me, o)"
                  :title="isSelf(me, o) ? 'Akun sendiri tidak dapat dihapus' : ''"
                  :aria-label="`Hapus ${o.name}`"
                  data-test="official-delete"
                  @click="openDelete(o)"
                >
                  Hapus
                </AppButton>
              </template>
              <span v-else class="off-locked font-mono">Hanya Admin</span>
            </td>
          </tr>
        </tbody>
      </table>
    </div>

    <!-- Tambah / ubah -->
    <div
      v-if="formOpen"
      class="off-overlay"
      role="dialog"
      aria-modal="true"
      aria-labelledby="off-form-title"
      data-test="official-form"
      @click.self="formOpen = false"
    >
      <form class="off-modal" @submit.prevent="submitForm">
        <header class="off-modal__header">
          <h3 id="off-form-title" class="off-modal__title">
            {{ editing ? 'Ubah Akun Petugas' : 'Tambah Akun Petugas' }}
          </h3>
          <button
            type="button"
            class="off-modal__close"
            aria-label="Tutup"
            @click="formOpen = false"
          >
            ✕
          </button>
        </header>
        <div class="off-modal__body">
          <label class="off-field">
            <span class="off-label font-mono">NAMA LENGKAP</span>
            <input v-model="fName" class="off-input font-mono" data-test="official-name" />
          </label>
          <label class="off-field">
            <span class="off-label font-mono">EMAIL</span>
            <input
              v-model="fEmail"
              type="email"
              class="off-input font-mono"
              :disabled="!!editing"
              data-test="official-email"
            />
            <span v-if="editing" class="off-help font-mono">Email tidak dapat diubah.</span>
          </label>
          <label v-if="!editing" class="off-field">
            <span class="off-label font-mono">PASSWORD AWAL (MIN. 8 KARAKTER)</span>
            <input
              v-model="fPassword"
              type="password"
              autocomplete="new-password"
              class="off-input font-mono"
              data-test="official-password"
            />
          </label>
          <label class="off-field">
            <span class="off-label font-mono">PERAN</span>
            <select
              v-model="fRole"
              class="off-input font-mono"
              :disabled="!!editing && isSelf(me, editing)"
              data-test="official-role"
            >
              <option v-for="r in roleOptions" :key="r" :value="r">{{ ROLE_LABELS[r] }}</option>
            </select>
            <span v-if="editing && isSelf(me, editing)" class="off-help font-mono">
              Peran akun sendiri tidak dapat diubah.
            </span>
          </label>
          <p v-if="formError" class="off-error" role="alert" data-test="official-form-error">
            {{ formError }}
          </p>
          <footer class="off-modal__footer font-mono">
            <AppButton type="button" variant="secondary" size="md" @click="formOpen = false">
              Batal
            </AppButton>
            <AppButton
              type="submit"
              variant="emerald"
              size="md"
              :loading="createMutation.isPending.value || updateMutation.isPending.value"
              data-test="official-save"
            >
              Simpan
            </AppButton>
          </footer>
        </div>
      </form>
    </div>

    <!-- Reset password -->
    <div
      v-if="resetTarget"
      class="off-overlay"
      role="dialog"
      aria-modal="true"
      aria-labelledby="off-reset-title"
      data-test="official-reset-dialog"
      @click.self="resetTarget = null"
    >
      <form class="off-modal" @submit.prevent="submitReset">
        <header class="off-modal__header">
          <h3 id="off-reset-title" class="off-modal__title">Reset Password</h3>
          <button
            type="button"
            class="off-modal__close"
            aria-label="Tutup"
            @click="resetTarget = null"
          >
            ✕
          </button>
        </header>
        <div class="off-modal__body">
          <p class="off-help">
            Tetapkan password baru untuk <strong>{{ resetTarget.name }}</strong> ({{
              resetTarget.email
            }}). Sampaikan langsung kepada yang bersangkutan.
          </p>
          <label class="off-field">
            <span class="off-label font-mono">PASSWORD BARU (MIN. 8 KARAKTER)</span>
            <input
              v-model="resetPassword"
              type="password"
              autocomplete="new-password"
              class="off-input font-mono"
              data-test="official-reset-password"
            />
          </label>
          <p v-if="resetError" class="off-error" role="alert" data-test="official-reset-error">
            {{ resetError }}
          </p>
          <footer class="off-modal__footer font-mono">
            <AppButton type="button" variant="secondary" size="md" @click="resetTarget = null">
              Batal
            </AppButton>
            <AppButton
              type="submit"
              variant="emerald"
              size="md"
              :loading="resetMutation.isPending.value"
              data-test="official-reset-submit"
            >
              Reset Password
            </AppButton>
          </footer>
        </div>
      </form>
    </div>

    <!-- Hapus -->
    <div
      v-if="deleteTarget"
      class="off-overlay"
      role="alertdialog"
      aria-modal="true"
      aria-labelledby="off-delete-title"
      data-test="official-delete-dialog"
      @click.self="deleteTarget = null"
    >
      <div class="off-modal">
        <header class="off-modal__header">
          <h3 id="off-delete-title" class="off-modal__title">Hapus Akun?</h3>
        </header>
        <div class="off-modal__body">
          <p class="off-help">
            Akun <strong>{{ deleteTarget.name }}</strong> ({{ deleteTarget.email }}) akan dihapus
            dan tidak dapat masuk lagi. Tindakan ini tidak dapat dibatalkan.
          </p>
          <p v-if="deleteError" class="off-error" role="alert" data-test="official-delete-error">
            {{ deleteError }}
          </p>
          <footer class="off-modal__footer font-mono">
            <AppButton type="button" variant="secondary" size="md" @click="deleteTarget = null">
              Batal
            </AppButton>
            <AppButton
              type="button"
              variant="danger"
              size="md"
              :loading="deleteMutation.isPending.value"
              data-test="official-delete-confirm"
              @click="confirmDelete"
            >
              Hapus Akun
            </AppButton>
          </footer>
        </div>
      </div>
    </div>

    <ChangeOwnPasswordDialog
      :open="ownPasswordOpen"
      @close="ownPasswordOpen = false"
      @changed="showToast('Password Diganti', 'Gunakan password baru saat masuk berikutnya.')"
    />
  </div>
</template>

<style scoped>
.off-ops {
  display: flex;
  flex-direction: column;
  gap: 20px;
  width: 100%;
  color: var(--text-main, #f8fafc);
  box-sizing: border-box;
}
.off-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  gap: 16px;
  flex-wrap: wrap;
}
.off-header__sub {
  font-size: 0.6875rem;
  font-weight: 700;
  letter-spacing: 0.08em;
  color: var(--color-emerald-400, #34d399);
}
.off-header__title {
  font-size: 1.5rem;
  font-weight: 800;
  color: #fff;
}
.off-header__desc {
  margin-top: 4px;
  font-size: 0.8125rem;
  color: var(--color-neutral-400, #94a3b8);
  max-width: 60ch;
}
.off-header__actions {
  display: flex;
  gap: 8px;
  flex-wrap: wrap;
}
.off-toolbar {
  display: flex;
  align-items: center;
  gap: 12px;
}
.off-search {
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
.off-count {
  font-size: 0.6875rem;
  letter-spacing: 0.06em;
  color: var(--color-neutral-400, #94a3b8);
}
.off-state {
  padding: 28px;
  text-align: center;
  color: var(--color-neutral-400, #94a3b8);
  border: 1px dashed var(--border-card, #334155);
  border-radius: 10px;
  font-size: 0.8125rem;
}
.off-state--error {
  color: var(--color-primary-500, #ef4444);
  display: flex;
  gap: 12px;
  justify-content: center;
  align-items: center;
}
.off-table-wrap {
  overflow-x: auto;
  background: var(--bg-card, #131d33);
  border: 1px solid var(--border-card, #334155);
  border-radius: 10px;
}
.off-table {
  width: 100%;
  border-collapse: collapse;
  font-size: 0.8125rem;
}
.off-table th {
  text-align: left;
  padding: 12px 16px;
  font-size: 0.6875rem;
  letter-spacing: 0.06em;
  color: var(--color-neutral-400, #94a3b8);
  border-bottom: 1px solid var(--border-card, #334155);
  white-space: nowrap;
}
.off-table td {
  padding: 12px 16px;
  border-bottom: 1px solid rgba(51, 65, 85, 0.5);
  vertical-align: middle;
}
.off-table tbody tr:last-child td {
  border-bottom: none;
}
.off-th-actions {
  text-align: right;
}
.off-name {
  font-weight: 700;
  color: #fff;
}
.off-self {
  margin-left: 8px;
  padding: 2px 6px;
  font-size: 0.625rem;
  letter-spacing: 0.06em;
  border-radius: 4px;
  color: var(--color-emerald-400, #34d399);
  border: 1px solid rgba(52, 211, 153, 0.4);
}
.off-role {
  display: inline-block;
  padding: 3px 8px;
  font-size: 0.6875rem;
  font-weight: 700;
  border-radius: 999px;
  background: rgba(100, 116, 139, 0.2);
  color: var(--color-neutral-200, #e2e8f0);
}
.off-role--admin {
  background: rgba(220, 38, 38, 0.18);
  color: #fca5a5;
}
.off-role--dewan {
  background: rgba(245, 158, 11, 0.18);
  color: #fcd34d;
}
.off-role--wasit {
  background: rgba(37, 99, 235, 0.2);
  color: #93c5fd;
}
.off-role--operator {
  background: rgba(16, 185, 129, 0.18);
  color: #6ee7b7;
}
.off-actions {
  display: flex;
  gap: 6px;
  justify-content: flex-end;
  flex-wrap: wrap;
}
.off-locked {
  font-size: 0.6875rem;
  color: var(--color-neutral-500, #64748b);
}
.off-overlay {
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
.off-modal {
  width: 100%;
  max-width: 480px;
  background: var(--bg-card, #131d33);
  border: 1px solid var(--border-card, #334155);
  border-radius: 12px;
  box-shadow: 0 20px 40px rgba(0, 0, 0, 0.6);
  overflow: hidden;
}
.off-modal__header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 18px 24px;
  border-bottom: 1px solid var(--border-card, #1e293b);
}
.off-modal__title {
  font-size: 1.125rem;
  font-weight: 800;
  color: #fff;
}
.off-modal__close {
  background: none;
  border: none;
  color: var(--color-neutral-400, #94a3b8);
  cursor: pointer;
  font-size: 1.25rem;
}
.off-modal__body {
  padding: 20px 24px;
  display: flex;
  flex-direction: column;
  gap: 16px;
}
.off-field {
  display: flex;
  flex-direction: column;
  gap: 6px;
}
.off-label {
  font-size: 0.6875rem;
  font-weight: 700;
  letter-spacing: 0.06em;
  color: var(--color-neutral-300, #cbd5e1);
}
.off-input {
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
.off-input:focus {
  border-color: var(--color-emerald-400, #34d399);
}
.off-input:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}
.off-help {
  font-size: 0.75rem;
  color: var(--color-neutral-400, #94a3b8);
}
.off-error {
  color: var(--color-primary-500, #ef4444);
  font-size: 0.8125rem;
}
.off-modal__footer {
  display: flex;
  justify-content: flex-end;
  gap: 10px;
}
.off-toast {
  position: fixed;
  top: 24px;
  right: 24px;
  z-index: 1200;
  display: flex;
  align-items: center;
  gap: 14px;
  padding: 12px 18px;
  border-radius: 8px;
  background: var(--bg-card, #131d33);
  border: 1px solid var(--border-card, #334155);
  box-shadow: 0 10px 30px rgba(0, 0, 0, 0.45);
  max-width: 420px;
}
.off-toast--success {
  border-left: 4px solid var(--color-emerald-500, #10b981);
}
.off-toast--danger {
  border-left: 4px solid var(--color-primary-600, #dc2626);
}
.off-toast__title {
  font-size: 0.875rem;
  font-weight: 700;
}
.off-toast__msg {
  font-size: 0.8125rem;
  color: var(--color-neutral-400, #94a3b8);
}
.off-toast__close {
  background: none;
  border: none;
  color: var(--color-neutral-500, #64748b);
  cursor: pointer;
}
.off-fade-enter-active,
.off-fade-leave-active {
  transition:
    opacity 0.2s ease,
    transform 0.2s ease;
}
.off-fade-enter-from,
.off-fade-leave-to {
  opacity: 0;
  transform: translateY(-8px);
}
</style>
