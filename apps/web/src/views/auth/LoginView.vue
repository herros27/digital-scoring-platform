<script setup lang="ts">
import { ref } from "vue";
import { useRouter } from "vue-router";
import { AppButton, AppInput } from "@/components/ui";
import { useAuthStore } from "@/stores/auth";
import { authApi } from "@/lib/authApi";

const router = useRouter();
const authStore = useAuthStore();

const email = ref("");
const password = ref("");
const rememberMe = ref(true);
const isLoading = ref(false);
const errorMessage = ref("");
const showPassword = ref(false);

async function handleLogin() {
  if (!email.value || !password.value) {
    errorMessage.value = "Harap isi alamat email dan kata sandi Anda.";
    return;
  }

  isLoading.value = true;
  errorMessage.value = "";

  try {
    // 1. Coba login sebagai Petugas Pertandingan (Admin, Dewan, Operator, Wasit)
    try {
      const res = await authApi.login({
        email: email.value.trim(),
        password: password.value,
      });

      const role = res.user?.role || "admin";
      authStore.setSession({
        token: res.token,
        role: role,
        name: res.user?.name || "Ofisial Pertandingan",
        email: res.user?.email,
      });

      // Redirect otomatis sesuai wewenang
      if (role === "admin" || role === "dewan") {
        router.push("/ops?tab=kompetisi");
      } else if (role === "wasit") {
        router.push("/referee");
      } else {
        router.push("/ops?tab=kompetisi");
      }
      return;
    } catch {
      // Jika bukan akun ofisial, coba login sebagai Manajer Kontingen
    }

    // 2. Coba login sebagai Akun Manajer Kontingen
    const contRes = await authApi.contingentLogin({
      email: email.value.trim(),
      password: password.value,
    });

    authStore.setSession({
      token: contRes.token,
      role: "contingent",
      name: contRes.contingent?.name,
      email: contRes.contingent?.email,
      contingentId: contRes.contingent?.id,
    });

    router.push("/ops?tab=atlet");
  } catch (err: any) {
    errorMessage.value =
      err.message || "Alamat email atau kata sandi tidak sesuai.";
  } finally {
    isLoading.value = false;
  }
}

// Preset demo akun untuk kemudahan demonstrasi
function fillDemo(type: "admin" | "contingent") {
  if (type === "admin") {
    email.value = "admin@silat.id";
    password.value = "AdminSecret123!";
  } else {
    email.value = "tapaksuci@example.com";
    password.value = "Secret123!";
  }
  errorMessage.value = "";
}

async function handleLogoutSession() {
  isLoading.value = true;
  try {
    await authStore.logout();
    email.value = "";
    password.value = "";
  } finally {
    isLoading.value = false;
  }
}

function handleContinueSession() {
  if (authStore.role === "admin" || authStore.role === "dewan") {
    router.push("/ops?tab=kompetisi");
  } else if (authStore.role === "wasit") {
    router.push("/referee");
  } else if (authStore.role === "contingent") {
    router.push("/ops?tab=atlet");
  } else {
    router.push("/ops?tab=kompetisi");
  }
}
</script>

<template>
  <div class="auth-page">
    <div class="auth-card">
      <!-- Brand Header -->
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
            stroke-linejoin="round">
            <path d="M12 2L2 7l10 5 10-5-10-5z" />
            <path d="M2 17l10 5 10-5" />
            <path d="M2 12l10 5 10-5" />
          </svg>
        </div>
        <h1 class="brand-title">Digital Scoring Silat</h1>
        <p class="brand-subtitle">
          Masuk ke sistem pertandingan & pendaftaran kontingen
        </p>
      </header>

      <!-- Active Session Notice -->
      <div
        v-if="authStore.isAuthenticated"
        class="active-session-banner font-mono">
        <div class="active-session-info">
          <div class="active-session-dot" />
          <span
            >Sesi Aktif: <strong>{{ authStore.name }}</strong> ({{
              authStore.role.toUpperCase()
            }})</span
          >
        </div>
        <div class="active-session-actions">
          <button
            type="button"
            class="btn-session-dash font-mono"
            @click="handleContinueSession">
            Lanjut ke Dashboard →
          </button>
          <button
            type="button"
            class="btn-session-logout font-mono"
            :disabled="isLoading"
            @click="handleLogoutSession">
            Keluar (Logout)
          </button>
        </div>
      </div>

      <!-- Error Notification -->
      <div
        v-if="errorMessage"
        class="error-banner"
        role="alert"
        aria-live="polite">
        <svg
          width="18"
          height="18"
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          stroke-width="2"
          aria-hidden="true">
          <circle cx="12" cy="12" r="10" />
          <line x1="12" y1="8" x2="12" y2="12" />
          <line x1="12" y1="16" x2="12.01" y2="16" />
        </svg>
        <span>{{ errorMessage }}</span>
      </div>

      <!-- Main Form -->
      <form class="auth-form" @submit.prevent="handleLogin" novalidate>
        <div class="form-group">
          <label class="form-label" for="login-email">Alamat email</label>
          <AppInput
            id="login-email"
            v-model="email"
            type="email"
            placeholder="nama@email.com"
            :disabled="isLoading"
            autocomplete="email"
            required />
        </div>

        <div class="form-group">
          <div class="label-row">
            <label class="form-label" for="login-password">Kata sandi</label>
            <span class="forgot-link">Lupa sandi?</span>
          </div>
          <div class="password-wrapper">
            <AppInput
              id="login-password"
              v-model="password"
              :type="showPassword ? 'text' : 'password'"
              placeholder="Masukkan kata sandi"
              :disabled="isLoading"
              autocomplete="current-password"
              required />
            <button
              type="button"
              class="toggle-pwd-btn"
              :aria-label="
                showPassword ? 'Sembunyikan kata sandi' : 'Tampilkan kata sandi'
              "
              @click="showPassword = !showPassword"
              tabindex="-1">
              {{ showPassword ? "Sembunyikan" : "Lihat" }}
            </button>
          </div>
        </div>

        <div class="remember-row">
          <label class="checkbox-label">
            <input
              type="checkbox"
              v-model="rememberMe"
              class="custom-checkbox" />
            <span>Ingat saya di perangkat ini</span>
          </label>
        </div>

        <AppButton
          type="submit"
          variant="primary"
          size="lg"
          full-width
          :loading="isLoading">
          Masuk ke Akun
        </AppButton>
      </form>

      <!-- Quick Demo Access -->
      <div class="demo-helpers">
        <span class="demo-caption">Akun Demo:</span>
        <div class="demo-buttons">
          <button type="button" class="demo-chip" @click="fillDemo('admin')">
            Admin Pertandingan
          </button>
          <button
            type="button"
            class="demo-chip"
            @click="fillDemo('contingent')">
            Official Kontingen
          </button>
        </div>
      </div>

      <!-- Footer Actions -->
      <footer class="auth-footer">
        <p class="register-prompt">
          Belum memiliki akun kontingen?
          <router-link to="/register" class="link-primary"
            >Daftar Kontingen</router-link
          >
        </p>

        <div class="divider" aria-hidden="true">
          <span>atau</span>
        </div>

        <router-link to="/judge/login" class="judge-login-btn">
          <svg
            width="18"
            height="18"
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            stroke-width="2"
            aria-hidden="true">
            <rect width="18" height="12" x="3" y="6" rx="2" />
            <path d="M7 6v-2a2 2 0 0 1 2-2h6a2 2 0 0 1 2 2v2" />
          </svg>
          <span>Masuk sebagai Tablet Juri / Wasit dengan PIN</span>
        </router-link>
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
  padding: 24px;
  background-color: var(--color-neutral-950, #090d16);
}

.auth-card {
  width: 100%;
  max-width: 420px;
  background: var(--color-neutral-900, #0f172a);
  border: 1px solid rgba(148, 163, 184, 0.14);
  border-radius: 12px;
  padding: 32px 28px;
  box-shadow:
    0 4px 6px -1px rgba(0, 0, 0, 0.4),
    0 2px 4px -2px rgba(0, 0, 0, 0.4),
    0 0 0 1px rgba(255, 255, 255, 0.05);
}

.brand-header {
  text-align: center;
  margin-bottom: 24px;
}

.brand-logo {
  width: 48px;
  height: 48px;
  margin: 0 auto 12px auto;
  border-radius: 10px;
  background: #2563eb;
  color: #ffffff;
  display: flex;
  align-items: center;
  justify-content: center;
}

.brand-title {
  font-size: 1.25rem;
  font-weight: 700;
  color: #ffffff;
  margin-bottom: 4px;
  text-wrap: balance;
}

.brand-subtitle {
  font-size: 0.8125rem;
  color: #94a3b8;
  text-wrap: pretty;
}

.error-banner {
  background: rgba(220, 38, 38, 0.12);
  border: 1px solid rgba(239, 68, 68, 0.3);
  color: #fca5a5;
  padding: 10px 14px;
  border-radius: 6px;
  font-size: 0.8125rem;
  margin-bottom: 18px;
  display: flex;
  align-items: center;
  gap: 10px;
}

.auth-form {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.form-group {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.form-label {
  font-size: 0.8125rem;
  font-weight: 500;
  color: #cbd5e1;
}

.label-row {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.forgot-link {
  font-size: 0.75rem;
  color: #60a5fa;
  cursor: pointer;
}
.forgot-link:hover {
  text-decoration: underline;
}

.password-wrapper {
  position: relative;
}

.toggle-pwd-btn {
  position: absolute;
  right: 12px;
  top: 50%;
  transform: translateY(-50%);
  background: none;
  border: none;
  color: #94a3b8;
  font-size: 0.75rem;
  cursor: pointer;
  padding: 4px 6px;
}
.toggle-pwd-btn:hover {
  color: #f1f5f9;
}

.remember-row {
  display: flex;
  align-items: center;
  font-size: 0.8125rem;
  color: #94a3b8;
}

.checkbox-label {
  display: flex;
  align-items: center;
  gap: 8px;
  cursor: pointer;
  user-select: none;
}

.custom-checkbox {
  accent-color: #2563eb;
  width: 15px;
  height: 15px;
}

.demo-helpers {
  margin-top: 16px;
  padding-top: 14px;
  border-top: 1px solid rgba(148, 163, 184, 0.1);
  display: flex;
  align-items: center;
  gap: 8px;
}

.demo-caption {
  font-size: 0.6875rem;
  color: #64748b;
}

.demo-buttons {
  display: flex;
  gap: 6px;
}

.demo-chip {
  background: rgba(30, 41, 59, 0.7);
  border: 1px solid rgba(148, 163, 184, 0.2);
  color: #93c5fd;
  font-size: 0.6875rem;
  padding: 4px 8px;
  border-radius: 4px;
  cursor: pointer;
  transition: all 0.15s ease;
  touch-action: manipulation;
}
.demo-chip:hover {
  background: rgba(37, 99, 235, 0.2);
  border-color: #3b82f6;
}

.auth-footer {
  margin-top: 20px;
  text-align: center;
}

.register-prompt {
  font-size: 0.8125rem;
  color: #94a3b8;
}

.link-primary {
  color: #60a5fa;
  font-weight: 600;
  text-decoration: none;
  margin-left: 4px;
}
.link-primary:hover {
  text-decoration: underline;
}

.divider {
  display: flex;
  align-items: center;
  margin: 16px 0;
  color: #64748b;
  font-size: 0.75rem;
}
.divider::before,
.divider::after {
  content: "";
  flex: 1;
  border-bottom: 1px solid rgba(148, 163, 184, 0.12);
}
.divider span {
  padding: 0 10px;
}

.judge-login-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  width: 100%;
  padding: 10px 14px;
  border: 1px solid rgba(16, 185, 129, 0.25);
  border-radius: 6px;
  background: rgba(16, 185, 129, 0.06);
  color: #6ee7b7;
  font-size: 0.8125rem;
  font-weight: 500;
  text-decoration: none;
  transition: all 0.15s ease;
  touch-action: manipulation;
}
.judge-login-btn:hover {
  background: rgba(16, 185, 129, 0.14);
  border-color: #10b981;
}

/* Active Session Banner */
.active-session-banner {
  background-color: rgba(16, 185, 129, 0.08);
  border: 1px solid rgba(16, 185, 129, 0.3);
  border-radius: 8px;
  padding: 12px 14px;
  margin-bottom: 20px;
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.active-session-info {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 0.75rem;
  color: #cbd5e1;
}

.active-session-info strong {
  color: #34d399;
}

.active-session-dot {
  width: 7px;
  height: 7px;
  border-radius: 50%;
  background-color: #10b981;
  box-shadow: 0 0 6px #10b981;
}

.active-session-actions {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
}

.btn-session-dash {
  background-color: #10b981;
  border: 1px solid #059669;
  color: #0f172a;
  font-weight: 700;
  padding: 6px 12px;
  border-radius: 6px;
  font-size: 0.6875rem;
  cursor: pointer;
  transition: all 0.15s ease;
}

.btn-session-dash:hover {
  background-color: #34d399;
}

.btn-session-logout {
  background: transparent;
  border: 1px solid rgba(239, 68, 68, 0.3);
  color: #f87171;
  padding: 5px 10px;
  border-radius: 6px;
  font-size: 0.6875rem;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.15s ease;
}

.btn-session-logout:hover {
  background-color: rgba(239, 68, 68, 0.15);
  border-color: #ef4444;
}
</style>
