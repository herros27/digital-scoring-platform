<script setup lang="ts">
import { getErrorMessage } from '@/api/error'
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { AppButton, AppInput } from '@/components/ui'
import { authService } from '@/features/auth/api/authService'

const router = useRouter()

const form = ref({
  name: '',
  pic_name: '',
  email: '',
  phone: '',
  password: '',
  confirmPassword: '',
})

const isLoading = ref(false)
const errorMessage = ref('')
const successMessage = ref('')

const showPassword = ref(false)
const showConfirmPassword = ref(false)

async function handleRegister() {
  errorMessage.value = ''
  successMessage.value = ''

  if (
    !form.value.name ||
    !form.value.pic_name ||
    !form.value.email ||
    !form.value.phone ||
    !form.value.password
  ) {
    errorMessage.value = 'Semua data formulir pendaftaran wajib diisi.'
    return
  }

  if (form.value.password.length < 6) {
    errorMessage.value = 'Kata sandi minimal 6 karakter.'
    return
  }

  if (form.value.password !== form.value.confirmPassword) {
    errorMessage.value = 'Konfirmasi kata sandi tidak cocok.'
    return
  }

  isLoading.value = true

  try {
    const res = await authService.contingentRegister({
      name: form.value.name.trim(),
      pic_name: form.value.pic_name.trim(),
      email: form.value.email.trim(),
      phone: form.value.phone.trim(),
      password: form.value.password,
    })

    successMessage.value = `Akun kontingen "${res.name}" berhasil didaftarkan. Mengalihkan ke login...`

    setTimeout(() => {
      router.push('/login')
    }, 1600)
  } catch (err) {
    errorMessage.value = getErrorMessage(
      err,
      'Pendaftaran gagal. Pastikan email belum pernah terdaftar.',
    )
  } finally {
    isLoading.value = false
  }
}
</script>

<template>
  <div class="auth-page">
    <div class="auth-card">
      <!-- Header -->
      <header class="brand-header">
        <div class="brand-logo" aria-hidden="true">
          <svg
            width="26"
            height="26"
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            stroke-width="2.2"
            stroke-linecap="round"
            stroke-linejoin="round"
          >
            <path d="M16 21v-2a4 4 0 0 0-4-4H6a4 4 0 0 0-4 4v2" />
            <circle cx="9" cy="7" r="4" />
            <line x1="19" y1="8" x2="19" y2="14" />
            <line x1="22" y1="11" x2="16" y2="11" />
          </svg>
        </div>
        <h1 class="brand-title">Daftar Kontingen Baru</h1>
        <p class="brand-subtitle">Registrasi akun manajer kontingen & official perguruan silat</p>
      </header>

      <!-- Feedback Banners -->
      <div v-if="successMessage" class="success-banner" role="status" aria-live="polite">
        <svg
          width="18"
          height="18"
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          stroke-width="2"
          aria-hidden="true"
        >
          <path d="M22 11.08V12a10 10 0 1 1-5.93-9.14" />
          <polyline points="22 4 12 14.01 9 11.01" />
        </svg>
        <span>{{ successMessage }}</span>
      </div>

      <div v-if="errorMessage" class="error-banner" role="alert" aria-live="polite">
        <svg
          width="18"
          height="18"
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          stroke-width="2"
          aria-hidden="true"
        >
          <circle cx="12" cy="12" r="10" />
          <line x1="12" y1="8" x2="12" y2="12" />
          <line x1="12" y1="16" x2="12.01" y2="16" />
        </svg>
        <span>{{ errorMessage }}</span>
      </div>

      <!-- Registration Form -->
      <form class="auth-form" @submit.prevent="handleRegister" novalidate>
        <div class="form-group">
          <label class="form-label" for="reg-name">Nama kontingen / perguruan</label>
          <AppInput
            id="reg-name"
            v-model="form.name"
            placeholder="Contoh: Tapak Suci Sleman / PSHT Solo"
            :disabled="isLoading"
            required
          />
        </div>

        <div class="form-group">
          <label class="form-label" for="reg-pic">Nama official (PIC penanggung jawab)</label>
          <AppInput
            id="reg-pic"
            v-model="form.pic_name"
            placeholder="Contoh: Budi Santoso, S.Pd"
            :disabled="isLoading"
            required
          />
        </div>

        <div class="form-row">
          <div class="form-group flex-1">
            <label class="form-label" for="reg-email">Alamat email resmi</label>
            <AppInput
              id="reg-email"
              v-model="form.email"
              type="email"
              placeholder="official@kontingen.id"
              :disabled="isLoading"
              required
            />
          </div>

          <div class="form-group flex-1">
            <label class="form-label" for="reg-phone">No. WhatsApp / HP</label>
            <AppInput
              id="reg-phone"
              v-model="form.phone"
              placeholder="081234567890"
              :disabled="isLoading"
              required
            />
          </div>
        </div>

        <div class="form-row">
          <div class="form-group flex-1">
            <label class="form-label" for="reg-password">Kata sandi</label>
            <AppInput
              id="reg-password"
              v-model="form.password"
              :type="showPassword ? 'text' : 'password'"
              placeholder="Min. 6 karakter"
              :disabled="isLoading"
              autocomplete="new-password"
              required
            >
              <template #suffix>
                <button
                  type="button"
                  class="toggle-pwd-btn"
                  :aria-label="showPassword ? 'Sembunyikan kata sandi' : 'Tampilkan kata sandi'"
                  @click="showPassword = !showPassword"
                  tabindex="-1"
                >
                  {{ showPassword ? 'Sembunyikan' : 'Lihat' }}
                </button>
              </template>
            </AppInput>
          </div>

          <div class="form-group flex-1">
            <label class="form-label" for="reg-confirm">Konfirmasi sandi</label>
            <AppInput
              id="reg-confirm"
              v-model="form.confirmPassword"
              :type="showConfirmPassword ? 'text' : 'password'"
              placeholder="Ulangi kata sandi"
              :disabled="isLoading"
              autocomplete="new-password"
              required
            >
              <template #suffix>
                <button
                  type="button"
                  class="toggle-pwd-btn"
                  :aria-label="showConfirmPassword ? 'Sembunyikan kata sandi' : 'Tampilkan kata sandi'"
                  @click="showConfirmPassword = !showConfirmPassword"
                  tabindex="-1"
                >
                  {{ showConfirmPassword ? 'Sembunyikan' : 'Lihat' }}
                </button>
              </template>
            </AppInput>
          </div>
        </div>

        <AppButton type="submit" variant="primary" size="lg" full-width :loading="isLoading">
          Daftarkan Kontingen
        </AppButton>
      </form>

      <!-- Footer Link -->
      <footer class="auth-footer">
        <p class="login-prompt">
          Sudah memiliki akun kontingen?
          <router-link to="/login" class="link-primary">Masuk sekarang</router-link>
        </p>
      </footer>
    </div>
  </div>
</template>

<style scoped>
.auth-page {
  min-height: 100dvh;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: clamp(16px, 4vw, 32px);
  padding-top: max(clamp(16px, 4vw, 32px), env(safe-area-inset-top));
  padding-bottom: max(clamp(16px, 4vw, 32px), env(safe-area-inset-bottom));
  padding-left: max(clamp(16px, 4vw, 32px), env(safe-area-inset-left));
  padding-right: max(clamp(16px, 4vw, 32px), env(safe-area-inset-right));
  background-color: var(--color-neutral-950, #090d16);
  box-sizing: border-box;
}

.auth-card {
  width: 100%;
  max-width: 500px;
  background: var(--color-neutral-900, #0f172a);
  border: 1px solid rgba(148, 163, 184, 0.14);
  border-radius: var(--radius-lg, 16px);
  padding: clamp(20px, 5vw, 32px) clamp(16px, 4.5vw, 28px);
  box-shadow:
    0 10px 25px -5px rgba(0, 0, 0, 0.5),
    0 8px 10px -6px rgba(0, 0, 0, 0.5),
    0 0 0 1px rgba(255, 255, 255, 0.05);
  box-sizing: border-box;
  margin: auto;
}

.brand-header {
  text-align: center;
  margin-bottom: clamp(18px, 4vw, 24px);
}

.brand-logo {
  width: clamp(42px, 8vw, 48px);
  height: clamp(42px, 8vw, 48px);
  margin: 0 auto 12px auto;
  border-radius: var(--radius-md, 12px);
  background: #059669;
  color: #ffffff;
  display: flex;
  align-items: center;
  justify-content: center;
  box-shadow: 0 4px 12px rgba(5, 150, 105, 0.35);
}

.brand-title {
  font-size: clamp(1.125rem, 4vw, 1.35rem);
  font-weight: 700;
  color: #ffffff;
  margin-bottom: 4px;
  text-wrap: balance;
  line-height: 1.3;
}

.brand-subtitle {
  font-size: clamp(0.75rem, 2.8vw, 0.8125rem);
  color: #94a3b8;
  text-wrap: pretty;
  line-height: 1.45;
}

.success-banner {
  background: rgba(16, 185, 129, 0.12);
  border: 1px solid rgba(16, 185, 129, 0.35);
  color: #a7f3d0;
  padding: 10px 14px;
  border-radius: 8px;
  font-size: clamp(0.75rem, 2.5vw, 0.8125rem);
  margin-bottom: 16px;
  display: flex;
  align-items: center;
  gap: 10px;
  line-height: 1.4;
  word-break: break-word;
}

.error-banner {
  background: rgba(220, 38, 38, 0.12);
  border: 1px solid rgba(239, 68, 68, 0.3);
  color: #fca5a5;
  padding: 10px 14px;
  border-radius: 8px;
  font-size: clamp(0.75rem, 2.5vw, 0.8125rem);
  margin-bottom: 16px;
  display: flex;
  align-items: flex-start;
  gap: 10px;
  line-height: 1.4;
  word-break: break-word;
}

.error-banner svg {
  flex-shrink: 0;
  margin-top: 2px;
}

.auth-form {
  display: flex;
  flex-direction: column;
  gap: clamp(12px, 3vw, 16px);
}

.form-group {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.form-row {
  display: flex;
  gap: 12px;
}
@media (max-width: 560px) {
  .form-row {
    flex-direction: column;
    gap: 12px;
  }
}

.flex-1 {
  flex: 1;
  min-width: 0;
}

.form-label {
  font-size: clamp(0.75rem, 2.5vw, 0.8125rem);
  font-weight: 500;
  color: #cbd5e1;
}

.toggle-pwd-btn {
  background: none;
  border: none;
  color: #94a3b8;
  font-size: clamp(0.6875rem, 2.4vw, 0.75rem);
  font-weight: 600;
  cursor: pointer;
  padding: 4px 8px;
  min-height: 32px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border-radius: 4px;
  touch-action: manipulation;
  transition: color 0.15s ease;
  white-space: nowrap;
}
.toggle-pwd-btn:hover {
  color: #f1f5f9;
}

.auth-footer {
  margin-top: clamp(16px, 3.5vw, 22px);
  text-align: center;
}

.login-prompt {
  font-size: clamp(0.75rem, 2.5vw, 0.8125rem);
  color: #94a3b8;
  line-height: 1.4;
}

.link-primary {
  color: #38bdf8;
  font-weight: 600;
  text-decoration: none;
  margin-left: 4px;
  display: inline-block;
  padding: 2px 4px;
}
.link-primary:hover {
  text-decoration: underline;
}

/* Compact screens / Landscape mobile orientation */
@media (max-height: 640px) {
  .auth-page {
    padding-top: 16px;
    padding-bottom: 16px;
  }
  .brand-header {
    margin-bottom: 14px;
  }
  .brand-logo {
    width: 36px;
    height: 36px;
    margin-bottom: 8px;
  }
  .brand-logo svg {
    width: 20px;
    height: 20px;
  }
  .auth-card {
    padding: 18px 16px;
  }
  .auth-form {
    gap: 10px;
  }
  .auth-footer {
    margin-top: 14px;
  }
}
</style>
