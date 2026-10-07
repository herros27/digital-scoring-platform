<script setup lang="ts">
import { computed, ref, onMounted, onUnmounted } from 'vue'
import { useRouter } from 'vue-router'
import { AppButton } from '@/components/ui'
import { useAuthStore } from '@/stores/auth'
import { authService } from '@/features/auth/api/authService'
import { useHaptics } from '@/composables/useHaptics'
import { MAX_PIN_LENGTH, isCompletePin, pinLoginErrorMessage } from '@/features/auth/lib/pin'

const router = useRouter()
const authStore = useAuthStore()
const haptics = useHaptics()

const pin = ref('')
const isLoading = ref(false)
const errorMessage = ref('')

function handleKeypad(val: number | string) {
  haptics.light()
  if (pin.value.length < MAX_PIN_LENGTH) {
    pin.value += String(val)
  }
}

function handleBackspace() {
  haptics.light()
  pin.value = pin.value.slice(0, -1)
}

function handleClear() {
  haptics.light()
  pin.value = ''
  errorMessage.value = ''
}

// Kotak PIN: 6 slot (juri), bertambah menjadi 8 saat PIN wasit diketik
const pinSlots = computed(() => (pin.value.length > 6 ? 8 : 6))

function handleKeyDown(e: KeyboardEvent) {
  if (isLoading.value) return
  if (e.key >= '0' && e.key <= '9') {
    handleKeypad(e.key)
  } else if (e.key === 'Backspace') {
    handleBackspace()
  } else if (e.key === 'Escape' || e.key === 'Delete') {
    handleClear()
  } else if (e.key === 'Enter') {
    if (isCompletePin(pin.value)) {
      handleLogin()
    }
  }
}

onMounted(() => {
  window.addEventListener('keydown', handleKeyDown)
})

onUnmounted(() => {
  window.removeEventListener('keydown', handleKeyDown)
})

async function handleLogin() {
  if (!isCompletePin(pin.value)) {
    errorMessage.value = 'PIN harus 6 digit (juri) atau 8 digit (wasit).'
    return
  }

  isLoading.value = true
  errorMessage.value = ''

  try {
    const res = await authService.judgeLogin({ pin: pin.value })

    haptics.success()
    const role = res.role || 'judge'
    authStore.setSession({
      token: res.token,
      role: role,
      seatNumber: res.seat_number,
      arenaId: res.arena_id,
      competitionId: res.competition_id,
      name: res.seat_number ? `Juri ${res.seat_number}` : 'Wasit Gelanggang',
    })

    if (role === 'wasit') {
      router.push({ path: '/referee', query: { arena_id: res.arena_id } })
    } else {
      router.push({ path: '/judge', query: { arena_id: res.arena_id, seat: res.seat_number } })
    }
  } catch (err) {
    haptics.heavy()
    errorMessage.value = pinLoginErrorMessage(err)
  } finally {
    isLoading.value = false
  }
}

async function handleLogoutSession() {
  await authStore.logout()
  pin.value = ''
  errorMessage.value = ''
}
</script>

<template>
  <div class="judge-login-page">
    <div class="judge-login-card">
      <!-- Header -->
      <header class="card-header">
        <div class="header-icon" aria-hidden="true">
          <svg
            width="24"
            height="24"
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            stroke-width="2.2"
          >
            <rect width="18" height="12" x="3" y="6" rx="2" />
            <path d="M7 6v-2a2 2 0 0 1 2-2h6a2 2 0 0 1 2 2v2" />
          </svg>
        </div>
        <h1 class="card-title">Login Tablet Gelanggang</h1>
        <p class="card-subtitle">Masukkan PIN resmi dari Meja Komando</p>
      </header>

      <!-- Active Session Notice -->
      <div v-if="authStore.isAuthenticated" class="active-judge-session font-mono">
        <div class="active-judge-info">
          <span class="active-judge-dot" />
          <span
            >Sesi Aktif: <strong>{{ authStore.name }}</strong> ({{
              authStore.arenaId || 'Gelanggang'
            }})</span
          >
        </div>
        <div class="active-judge-actions">
          <button
            type="button"
            class="btn-judge-dash font-mono"
            @click="router.push(authStore.role === 'wasit' ? '/referee' : '/judge')"
          >
            Lanjut ke Layar →
          </button>
          <button type="button" class="btn-judge-logout font-mono" @click="handleLogoutSession">
            Keluar (Ganti PIN)
          </button>
        </div>
      </div>

      <!-- Error Banner -->
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

      <!-- PIN Slots Display -->
      <div class="pin-display-wrapper" aria-label="Input PIN">
        <div class="pin-slots font-mono tabular-nums">
          <div
            v-for="idx in pinSlots"
            :key="idx"
            class="pin-slot"
            :class="{ 'pin-slot--filled': pin.length >= idx }"
          >
            {{ pin[idx - 1] ? '•' : '' }}
          </div>
        </div>
      </div>

      <!-- Onscreen Numeric Keypad -->
      <div class="keypad-grid font-mono tabular-nums" role="group" aria-label="Keypad Angka">
        <button
          v-for="num in [1, 2, 3, 4, 5, 6, 7, 8, 9]"
          :key="num"
          type="button"
          class="keypad-btn"
          :disabled="isLoading"
          @click="handleKeypad(num)"
        >
          {{ num }}
        </button>
        <button
          type="button"
          class="keypad-btn keypad-btn--clear"
          :disabled="isLoading"
          aria-label="Hapus semua digit"
          @click="handleClear"
        >
          C
        </button>
        <button type="button" class="keypad-btn" :disabled="isLoading" @click="handleKeypad(0)">
          0
        </button>
        <button
          type="button"
          class="keypad-btn keypad-btn--del"
          :disabled="isLoading"
          aria-label="Hapus satu digit"
          @click="handleBackspace"
        >
          ⌫
        </button>
      </div>

      <!-- Submit Action -->
      <div class="actions-wrapper">
        <AppButton variant="emerald" size="lg" full-width :loading="isLoading" @click="handleLogin">
          Masuk
        </AppButton>
      </div>

      <!-- Back to main login -->
      <footer class="card-footer">
        <router-link to="/login" class="back-link"> ← Kembali ke Login Email Utama </router-link>
      </footer>
    </div>
  </div>
</template>

<style scoped>
.judge-login-page {
  min-height: 100dvh;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: clamp(14px, 3.5vw, 24px);
  padding-top: max(clamp(14px, 3.5vw, 24px), env(safe-area-inset-top));
  padding-bottom: max(clamp(14px, 3.5vw, 24px), env(safe-area-inset-bottom));
  padding-left: max(clamp(14px, 3.5vw, 24px), env(safe-area-inset-left));
  padding-right: max(clamp(14px, 3.5vw, 24px), env(safe-area-inset-right));
  background-color: var(--color-neutral-950, #090d16);
  box-sizing: border-box;
}

.judge-login-card {
  width: 100%;
  max-width: 420px;
  background: var(--color-neutral-900, #0f172a);
  border: 1px solid rgba(148, 163, 184, 0.14);
  border-radius: var(--radius-lg, 16px);
  padding: clamp(18px, 4.5vw, 28px) clamp(14px, 4vw, 24px);
  box-shadow:
    0 10px 25px -5px rgba(0, 0, 0, 0.5),
    0 8px 10px -6px rgba(0, 0, 0, 0.5),
    0 0 0 1px rgba(255, 255, 255, 0.05);
  box-sizing: border-box;
  margin: auto;
}

.card-header {
  text-align: center;
  margin-bottom: clamp(12px, 3vw, 18px);
}

.header-icon {
  width: clamp(38px, 7vw, 44px);
  height: clamp(38px, 7vw, 44px);
  margin: 0 auto 10px auto;
  border-radius: var(--radius-md, 10px);
  background: #059669;
  color: #ffffff;
  display: flex;
  align-items: center;
  justify-content: center;
  box-shadow: 0 4px 12px rgba(5, 150, 105, 0.35);
}

.card-title {
  font-size: clamp(1rem, 3.5vw, 1.25rem);
  font-weight: 700;
  color: #ffffff;
  margin-bottom: 4px;
  text-wrap: balance;
  line-height: 1.3;
}

.card-subtitle {
  font-size: clamp(0.75rem, 2.5vw, 0.8125rem);
  color: #94a3b8;
  text-wrap: pretty;
  line-height: 1.4;
}

.error-banner {
  background: rgba(220, 38, 38, 0.12);
  border: 1px solid rgba(239, 68, 68, 0.3);
  color: #fca5a5;
  padding: 10px 12px;
  border-radius: 8px;
  font-size: clamp(0.75rem, 2.5vw, 0.8125rem);
  margin-bottom: 14px;
  display: flex;
  align-items: flex-start;
  gap: 8px;
  line-height: 1.4;
  word-break: break-word;
}

.error-banner svg {
  flex-shrink: 0;
  margin-top: 2px;
}

.pin-display-wrapper {
  display: flex;
  justify-content: center;
  margin: clamp(10px, 2.5vh, 18px) 0;
  width: 100%;
}

.pin-slots {
  display: flex;
  justify-content: center;
  gap: clamp(4px, 1.8vw, 8px);
  width: 100%;
  max-width: 380px;
}

.pin-slot {
  flex: 1 1 0;
  max-width: 44px;
  min-width: 0;
  height: clamp(42px, 8vw, 52px);
  border: 1px solid rgba(148, 163, 184, 0.25);
  border-radius: var(--radius-sm, 8px);
  background: rgba(15, 23, 42, 0.9);
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: clamp(1.2rem, 3.5vw, 1.6rem);
  font-weight: 700;
  color: #10b981;
  box-sizing: border-box;
  transition: border-color 0.15s ease, background-color 0.15s ease, box-shadow 0.15s ease;
}

.pin-slot--filled {
  border-color: #10b981;
  background: rgba(6, 78, 59, 0.3);
  box-shadow: 0 0 10px rgba(16, 185, 129, 0.25);
}

.keypad-grid {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: clamp(6px, 1.8vw, 10px);
  margin-bottom: clamp(12px, 2.5vh, 18px);
  width: 100%;
}

.keypad-btn {
  background: rgba(30, 41, 59, 0.7);
  border: 1px solid rgba(148, 163, 184, 0.15);
  color: #f8fafc;
  padding: 0;
  height: clamp(46px, 7vh, 58px);
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: var(--radius-sm, 8px);
  font-size: clamp(1.1rem, 3.5vw, 1.35rem);
  font-weight: 700;
  cursor: pointer;
  transition:
    transform 0.1s ease,
    background-color 0.15s ease,
    border-color 0.15s ease;
  touch-action: manipulation;
  user-select: none;
  box-sizing: border-box;
}

.keypad-btn:hover {
  background: rgba(51, 65, 85, 0.85);
  border-color: rgba(148, 163, 184, 0.3);
}

.keypad-btn:active {
  transform: scale(0.95);
  background: rgba(71, 85, 105, 0.9);
}

.keypad-btn--clear {
  color: #f87171;
  font-size: clamp(0.95rem, 3vw, 1.1rem);
}

.keypad-btn--del {
  color: #fbbf24;
  font-size: clamp(1.1rem, 3.5vw, 1.35rem);
}

.actions-wrapper {
  margin-bottom: clamp(10px, 2vh, 14px);
}

.card-footer {
  text-align: center;
  border-top: 1px solid rgba(148, 163, 184, 0.1);
  padding-top: clamp(10px, 2vh, 14px);
}

.back-link {
  color: #94a3b8;
  font-size: clamp(0.6875rem, 2.4vw, 0.75rem);
  text-decoration: none;
  display: inline-block;
  padding: 4px 6px;
  touch-action: manipulation;
  transition: color 0.15s ease;
}

.back-link:hover {
  color: #38bdf8;
}

/* Active Judge Session */
.active-judge-session {
  background-color: rgba(16, 185, 129, 0.08);
  border: 1px solid rgba(16, 185, 129, 0.3);
  border-radius: 8px;
  padding: 10px 14px;
  margin-bottom: 14px;
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.active-judge-info {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: clamp(0.6875rem, 2.4vw, 0.75rem);
  color: #cbd5e1;
  word-break: break-word;
}

.active-judge-info strong {
  color: #34d399;
}

.active-judge-dot {
  width: 7px;
  height: 7px;
  border-radius: 50%;
  background-color: #10b981;
  box-shadow: 0 0 6px #10b981;
  flex-shrink: 0;
}

.active-judge-actions {
  display: flex;
  align-items: stretch;
  gap: 8px;
  flex-wrap: wrap;
}

.btn-judge-dash {
  flex: 1 1 130px;
  background-color: #10b981;
  border: 1px solid #059669;
  color: #0f172a;
  font-weight: 700;
  padding: 8px 12px;
  min-height: 36px;
  border-radius: 6px;
  font-size: 0.6875rem;
  cursor: pointer;
  transition: all 0.15s ease;
  touch-action: manipulation;
  text-align: center;
}

.btn-judge-dash:hover {
  background-color: #34d399;
}

.btn-judge-logout {
  flex: 0 1 auto;
  background: transparent;
  border: 1px solid rgba(239, 68, 68, 0.3);
  color: #f87171;
  padding: 8px 12px;
  min-height: 36px;
  border-radius: 6px;
  font-size: 0.6875rem;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.15s ease;
  touch-action: manipulation;
}

.btn-judge-logout:hover {
  background-color: rgba(239, 68, 68, 0.15);
  border-color: #ef4444;
}

/* Compact screens / Landscape mobile orientation */
@media (max-height: 600px) {
  .judge-login-page {
    padding-top: 10px;
    padding-bottom: 10px;
  }
  .card-header {
    margin-bottom: 8px;
  }
  .header-icon {
    width: 32px;
    height: 32px;
    margin-bottom: 4px;
  }
  .header-icon svg {
    width: 18px;
    height: 18px;
  }
  .card-title {
    font-size: 1rem;
  }
  .card-subtitle {
    font-size: 0.75rem;
  }
  .pin-display-wrapper {
    margin: 6px 0;
  }
  .pin-slot {
    height: 36px;
  }
  .keypad-grid {
    gap: 4px;
    margin-bottom: 8px;
  }
  .keypad-btn {
    height: 38px;
    font-size: 1rem;
  }
  .judge-login-card {
    padding: 14px 16px;
  }
  .actions-wrapper {
    margin-bottom: 8px;
  }
  .card-footer {
    padding-top: 6px;
  }
}
</style>
