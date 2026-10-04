<script setup lang="ts">
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { AppButton, AppInput } from '@/components/ui'
import { useAuthStore } from '@/stores/auth'
import { authApi } from '@/lib/authApi'
import { useHaptics } from '@/composables/useHaptics'

const router = useRouter()
const authStore = useAuthStore()
const haptics = useHaptics()

const arenaId = ref('gelanggang-1')
const competitionId = ref('comp-default')
const pin = ref('')
const isLoading = ref(false)
const errorMessage = ref('')

function handleKeypad(val: number | string) {
  haptics.light()
  if (pin.value.length < 6) {
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

function fillSamplePin(samplePin: string) {
  haptics.medium()
  pin.value = samplePin
  errorMessage.value = ''
}

async function handleLogin() {
  if (pin.value.length < 4) {
    errorMessage.value = 'PIN minimal 4 sampai 6 digit.'
    return
  }

  isLoading.value = true
  errorMessage.value = ''

  try {
    const res = await authApi.judgeLogin({
      competition_id: competitionId.value.trim(),
      arena_id: arenaId.value,
      pin: pin.value.trim(),
    })

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
  } catch (err: any) {
    haptics.heavy()
    errorMessage.value = err.message || 'PIN tidak valid atau sudah kadaluarsa untuk gelanggang ini.'
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
          <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.2">
            <rect width="18" height="12" x="3" y="6" rx="2" />
            <path d="M7 6v-2a2 2 0 0 1 2-2h6a2 2 0 0 1 2 2v2" />
          </svg>
        </div>
        <h1 class="card-title">Login Tablet Gelanggang</h1>
        <p class="card-subtitle">Pilih gelanggang dan masukkan PIN resmi dari Meja Komando</p>
      </header>

      <!-- Active Session Notice -->
      <div v-if="authStore.isAuthenticated" class="active-judge-session font-mono">
        <div class="active-judge-info">
          <span class="active-judge-dot" />
          <span>Sesi Aktif: <strong>{{ authStore.name }}</strong> ({{ authStore.arenaId || 'Gelanggang' }})</span>
        </div>
        <div class="active-judge-actions">
          <button type="button" class="btn-judge-dash font-mono" @click="router.push(authStore.role === 'wasit' ? '/referee' : '/judge')">
            Lanjut ke Layar →
          </button>
          <button type="button" class="btn-judge-logout font-mono" @click="handleLogoutSession">
            Keluar (Ganti PIN)
          </button>
        </div>
      </div>

      <!-- Error Banner -->
      <div v-if="errorMessage" class="error-banner" role="alert" aria-live="polite">
        <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" aria-hidden="true">
          <circle cx="12" cy="12" r="10" />
          <line x1="12" y1="8" x2="12" y2="12" />
          <line x1="12" y1="16" x2="12.01" y2="16" />
        </svg>
        <span>{{ errorMessage }}</span>
      </div>

      <!-- Arena Selector -->
      <div class="arena-group">
        <label class="form-label">Pilih gelanggang</label>
        <div class="arena-grid" role="group" aria-label="Pilihan Gelanggang">
          <button
            v-for="i in 4"
            :key="i"
            type="button"
            class="arena-btn font-mono"
            :class="{ 'arena-btn--active': arenaId === `gelanggang-${i}` }"
            @click="arenaId = `gelanggang-${i}`"
          >
            Arena {{ i }}
          </button>
        </div>
      </div>

      <!-- Competition ID Field -->
      <div class="comp-id-group">
        <label class="form-label" for="comp-id">ID Turnamen</label>
        <AppInput
          id="comp-id"
          v-model="competitionId"
          placeholder="comp-default"
          :disabled="isLoading"
        />
      </div>

      <!-- PIN Slots Display -->
      <div class="pin-display-wrapper" aria-label="Input PIN 6 Digit">
        <div class="pin-slots font-mono tabular-nums">
          <div
            v-for="idx in 6"
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
        <button
          type="button"
          class="keypad-btn"
          :disabled="isLoading"
          @click="handleKeypad(0)"
        >
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
        <AppButton
          variant="emerald"
          size="lg"
          full-width
          :loading="isLoading"
          @click="handleLogin"
        >
          Masuk ke Gelanggang
        </AppButton>
      </div>

      <!-- Sample PIN Shortcuts for Development/Testing -->
      <div class="samples-box">
        <span class="sample-label font-mono">Sampel PIN:</span>
        <div class="sample-tags">
          <button type="button" class="sample-tag font-mono" @click="fillSamplePin('839102')">
            Juri 1: 839102
          </button>
          <button type="button" class="sample-tag font-mono" @click="fillSamplePin('491205')">
            Juri 2: 491205
          </button>
          <button type="button" class="sample-tag font-mono" @click="fillSamplePin('662901')">
            Wasit: 662901
          </button>
        </div>
      </div>

      <!-- Back to main login -->
      <footer class="card-footer">
        <router-link to="/login" class="back-link">
          ← Kembali ke Login Email Utama
        </router-link>
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
  padding: 24px;
  background-color: var(--color-neutral-950, #090d16);
}

.judge-login-card {
  width: 100%;
  max-width: 400px;
  background: var(--color-neutral-900, #0f172a);
  border: 1px solid rgba(148, 163, 184, 0.14);
  border-radius: 12px;
  padding: 28px 24px;
  box-shadow: 0 4px 6px -1px rgba(0, 0, 0, 0.4), 0 2px 4px -2px rgba(0, 0, 0, 0.4), 0 0 0 1px rgba(255, 255, 255, 0.05);
}

.card-header {
  text-align: center;
  margin-bottom: 18px;
}

.header-icon {
  width: 44px;
  height: 44px;
  margin: 0 auto 10px auto;
  border-radius: 10px;
  background: #059669;
  color: #ffffff;
  display: flex;
  align-items: center;
  justify-content: center;
}

.card-title {
  font-size: 1.125rem;
  font-weight: 700;
  color: #ffffff;
  margin-bottom: 4px;
  text-wrap: balance;
}

.card-subtitle {
  font-size: 0.8125rem;
  color: #94a3b8;
  text-wrap: pretty;
}

.error-banner {
  background: rgba(220, 38, 38, 0.12);
  border: 1px solid rgba(239, 68, 68, 0.3);
  color: #fca5a5;
  padding: 10px 12px;
  border-radius: 6px;
  font-size: 0.8125rem;
  margin-bottom: 14px;
  display: flex;
  align-items: center;
  gap: 8px;
}

.arena-group {
  margin-bottom: 12px;
}

.comp-id-group {
  margin-bottom: 14px;
}

.form-label {
  display: block;
  font-size: 0.75rem;
  font-weight: 500;
  color: #cbd5e1;
  margin-bottom: 6px;
}

.arena-grid {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 6px;
}

.arena-btn {
  background: rgba(30, 41, 59, 0.7);
  border: 1px solid rgba(148, 163, 184, 0.18);
  color: #94a3b8;
  padding: 8px 4px;
  border-radius: 6px;
  font-size: 0.75rem;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.15s ease;
  touch-action: manipulation;
}
.arena-btn:hover {
  background: rgba(51, 65, 85, 0.8);
  color: #f1f5f9;
}
.arena-btn--active {
  background: rgba(16, 185, 129, 0.2);
  border-color: #10b981;
  color: #a7f3d0;
}

.pin-display-wrapper {
  display: flex;
  justify-content: center;
  margin: 14px 0;
}

.pin-slots {
  display: flex;
  gap: 8px;
}

.pin-slot {
  width: 40px;
  height: 48px;
  border: 1px solid rgba(148, 163, 184, 0.25);
  border-radius: 6px;
  background: rgba(15, 23, 42, 0.9);
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 1.375rem;
  font-weight: 700;
  color: #10b981;
}
.pin-slot--filled {
  border-color: #10b981;
  background: rgba(6, 78, 59, 0.25);
}

.keypad-grid {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 8px;
  margin-bottom: 16px;
}

.keypad-btn {
  background: rgba(30, 41, 59, 0.7);
  border: 1px solid rgba(148, 163, 184, 0.15);
  color: #f8fafc;
  padding: 12px;
  border-radius: 6px;
  font-size: 1.125rem;
  font-weight: 600;
  cursor: pointer;
  transition: transform 0.1s ease, background-color 0.15s ease;
  touch-action: manipulation;
  user-select: none;
}
.keypad-btn:hover {
  background: rgba(51, 65, 85, 0.85);
}
.keypad-btn:active {
  transform: scale(0.96);
}
.keypad-btn--clear {
  color: #f87171;
  font-size: 0.9375rem;
}
.keypad-btn--del {
  color: #fbbf24;
}

.actions-wrapper {
  margin-bottom: 14px;
}

.samples-box {
  background: rgba(15, 23, 42, 0.5);
  border: 1px dashed rgba(148, 163, 184, 0.15);
  border-radius: 6px;
  padding: 8px 10px;
  display: flex;
  align-items: center;
  gap: 6px;
  flex-wrap: wrap;
  margin-bottom: 14px;
}

.sample-label {
  font-size: 0.6875rem;
  color: #64748b;
}

.sample-tags {
  display: flex;
  gap: 4px;
  flex-wrap: wrap;
}

.sample-tag {
  background: rgba(16, 185, 129, 0.08);
  border: 1px solid rgba(16, 185, 129, 0.2);
  color: #6ee7b7;
  font-size: 0.6875rem;
  padding: 2px 6px;
  border-radius: 4px;
  cursor: pointer;
  transition: all 0.12s ease;
  touch-action: manipulation;
}
.sample-tag:hover {
  background: rgba(16, 185, 129, 0.18);
}

.card-footer {
  text-align: center;
  border-top: 1px solid rgba(148, 163, 184, 0.1);
  padding-top: 12px;
}

.back-link {
  color: #94a3b8;
  font-size: 0.75rem;
  text-decoration: none;
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
  padding: 12px 14px;
  margin-bottom: 20px;
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.active-judge-info {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 0.75rem;
  color: #CBD5E1;
}

.active-judge-info strong {
  color: #34D399;
}

.active-judge-dot {
  width: 7px;
  height: 7px;
  border-radius: 50%;
  background-color: #10B981;
  box-shadow: 0 0 6px #10B981;
}

.active-judge-actions {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
}

.btn-judge-dash {
  background-color: #10B981;
  border: 1px solid #059669;
  color: #0F172A;
  font-weight: 700;
  padding: 6px 12px;
  border-radius: 6px;
  font-size: 0.6875rem;
  cursor: pointer;
  transition: all 0.15s ease;
}

.btn-judge-dash:hover {
  background-color: #34D399;
}

.btn-judge-logout {
  background: transparent;
  border: 1px solid rgba(239, 68, 68, 0.3);
  color: #F87171;
  padding: 5px 10px;
  border-radius: 6px;
  font-size: 0.6875rem;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.15s ease;
}

.btn-judge-logout:hover {
  background-color: rgba(239, 68, 68, 0.15);
  border-color: #EF4444;
}
</style>
