<script setup lang="ts">
import { ref, computed } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { useAuthStore } from '@/stores/auth'

const router = useRouter()
const route = useRoute()
const authStore = useAuthStore()

const currentPath = computed(() => route.path)
const showLogoutModal = ref(false)
const isLoggingOut = ref(false)

function handleLogout() {
  showLogoutModal.value = true
}

async function confirmLogout() {
  isLoggingOut.value = true
  try {
    await authStore.logout()
    showLogoutModal.value = false
    router.push('/login')
  } finally {
    isLoggingOut.value = false
  }
}
</script>

<template>
  <div class="admin-shell">
    <!-- Topbar -->
    <header class="admin-topbar">
      <div class="admin-topbar__brand">
        <div class="brand-badge">
          <svg
            width="18"
            height="18"
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            stroke-width="2.5"
          >
            <path d="M6 9H4.5a2.5 2.5 0 0 1 0-5H6" />
            <path d="M18 9h1.5a2.5 2.5 0 0 0 0-5H18" />
            <path d="M4 22h16" />
            <path d="M10 14.66V17c0 .55-.47.98-.97 1.21C7.85 18.75 7 20.24 7 22" />
            <path d="M14 14.66V17c0 .55.47.98.97 1.21C16.15 18.75 17 20.24 17 22" />
            <path d="M18 2H6v7a6 6 0 0 0 12 0V2Z" />
          </svg>
        </div>
        <div class="brand-text">
          <span class="brand-name font-heading">DIGITALSCORING</span>
          <span class="brand-tag font-mono">DIREKTORAT PERTANDINGAN</span>
        </div>
      </div>

      <nav class="admin-topbar__nav">
        <router-link
          to="/admin/competitions"
          class="topbar-link"
          :class="{ 'topbar-link--active': currentPath.includes('/competitions') }"
        >
          Kejuaraan
        </router-link>
        <router-link
          to="/ops"
          class="topbar-link"
          :class="{ 'topbar-link--active': currentPath.includes('/ops') }"
        >
          Bagan & Gelanggang
        </router-link>
        <router-link
          to="/admin/pins"
          class="topbar-link"
          :class="{ 'topbar-link--active': currentPath.includes('/pins') }"
        >
          PIN Perangkat
        </router-link>
        <router-link to="/scoreboard" class="topbar-link" target="_blank">
          Live Scoreboard ↗
        </router-link>
      </nav>

      <div class="admin-topbar__profile">
        <div class="profile-info">
          <span class="profile-name">{{ authStore.name || 'Petugas Pertandingan' }}</span>
          <span class="profile-role font-mono">{{ authStore.role.toUpperCase() }}</span>
        </div>
        <button class="logout-btn" title="Keluar sesi" @click="handleLogout">Keluar</button>
      </div>
    </header>

    <!-- Main Content Slot -->
    <main class="admin-main">
      <slot>
        <router-view />
      </slot>
    </main>

    <!-- Logout Confirmation Modal -->
    <div v-if="showLogoutModal" class="logout-modal-backdrop" @click.self="showLogoutModal = false">
      <div class="logout-modal">
        <div class="logout-modal-icon">
          <svg
            width="24"
            height="24"
            viewBox="0 0 24 24"
            fill="none"
            stroke="#EF4444"
            stroke-width="2.2"
          >
            <path d="M9 21H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h4" />
            <polyline points="16 17 21 12 16 7" />
            <line x1="21" x2="9" y1="12" y2="12" />
          </svg>
        </div>
        <h3 class="logout-modal-title font-heading">Konfirmasi Keluar Sesi Admin</h3>
        <p class="logout-modal-desc">
          Apakah Anda yakin ingin mengakhiri sesi Administrator dan keluar dari panel kontrol?
        </p>
        <div class="logout-modal-actions font-mono">
          <button class="logout-cancel-btn" @click="showLogoutModal = false">Batal</button>
          <button class="logout-confirm-btn" :disabled="isLoggingOut" @click="confirmLogout">
            {{ isLoggingOut ? 'Mengakhiri Sesi...' : 'Keluar Sekarang' }}
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.admin-shell {
  min-height: 100dvh;
  display: flex;
  flex-direction: column;
  background-color: var(--bg-app, #0a0e17);
  color: var(--text-primary, #f1f5f9);
}

.admin-topbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 12px 24px;
  background-color: var(--bg-card, #111827);
  border-bottom: 1px solid var(--border-card, #1f293d);
  gap: 16px;
  position: sticky;
  top: 0;
  z-index: 50;
}

.admin-topbar__brand {
  display: flex;
  align-items: center;
  gap: 12px;
}

.brand-badge {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 34px;
  height: 34px;
  border-radius: var(--radius-md, 8px);
  background-color: var(--color-primary-600, #2563eb);
  color: #fff;
}

.brand-text {
  display: flex;
  flex-direction: column;
}

.brand-name {
  font-size: 0.9375rem;
  font-weight: 800;
  letter-spacing: -0.01em;
}

.brand-tag {
  font-size: 0.625rem;
  font-weight: 700;
  color: var(--color-primary-400, #60a5fa);
  letter-spacing: 0.05em;
}

.admin-topbar__nav {
  display: flex;
  align-items: center;
  gap: 6px;
}

.topbar-link {
  padding: 6px 14px;
  border-radius: var(--radius-md, 6px);
  color: var(--text-secondary, #94a3b8);
  font-size: 0.8125rem;
  font-weight: 600;
  text-decoration: none;
  transition: all var(--transition-fast, 0.15s ease);
}

.topbar-link:hover {
  color: #fff;
  background-color: var(--bg-elevated, #1e293b);
}

.topbar-link--active {
  color: #fff;
  background-color: var(--bg-elevated, #1e293b);
  border: 1px solid var(--border-active, #3b82f6);
}

.admin-topbar__profile {
  display: flex;
  align-items: center;
  gap: 14px;
}

.profile-info {
  display: flex;
  flex-direction: column;
  align-items: flex-end;
}

.profile-name {
  font-size: 0.8125rem;
  font-weight: 700;
}

.profile-role {
  font-size: 0.625rem;
  color: var(--color-emerald-400, #34d399);
}

.logout-btn {
  background: transparent;
  border: 1px solid var(--border-card, #1f293d);
  padding: 5px 12px;
  border-radius: var(--radius-sm, 4px);
  color: var(--text-muted, #64748b);
  font-size: 0.75rem;
  font-weight: 600;
  cursor: pointer;
  transition: all var(--transition-fast, 0.15s ease);
}

.logout-btn:hover {
  color: var(--color-red-400, #f87171);
  border-color: rgba(239, 68, 68, 0.4);
}

.admin-main {
  flex: 1;
  display: flex;
  flex-direction: column;
}

/* Modal Confirmation */
.logout-modal-backdrop {
  position: fixed;
  inset: 0;
  background-color: rgba(0, 0, 0, 0.75);
  backdrop-filter: blur(4px);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 100;
}

.logout-modal {
  background-color: #0f172a;
  border: 1px solid rgba(255, 255, 255, 0.15);
  border-radius: 12px;
  padding: 24px;
  max-width: 440px;
  width: 90%;
  display: flex;
  flex-direction: column;
  align-items: center;
  text-align: center;
}

.logout-modal-icon {
  width: 48px;
  height: 48px;
  border-radius: 50%;
  background-color: rgba(239, 68, 68, 0.15);
  display: flex;
  align-items: center;
  justify-content: center;
  margin-bottom: 12px;
}

.logout-modal-title {
  font-size: 1.125rem;
  font-weight: 700;
  color: #ffffff;
  margin-bottom: 8px;
}

.logout-modal-desc {
  font-size: 0.8125rem;
  color: #94a3b8;
  line-height: 1.5;
  margin-bottom: 20px;
}

.logout-modal-actions {
  display: flex;
  gap: 12px;
  width: 100%;
  justify-content: center;
}

.logout-cancel-btn {
  background: transparent;
  border: 1px solid rgba(255, 255, 255, 0.15);
  color: #cbd5e1;
  padding: 8px 18px;
  border-radius: 6px;
  font-size: 0.8125rem;
  cursor: pointer;
  transition: all 0.15s ease;
}

.logout-cancel-btn:hover {
  background-color: rgba(255, 255, 255, 0.05);
}

.logout-confirm-btn {
  background-color: #dc2626;
  border: 1px solid #ef4444;
  color: #ffffff;
  padding: 8px 18px;
  border-radius: 6px;
  font-size: 0.8125rem;
  font-weight: 700;
  cursor: pointer;
  transition: all 0.15s ease;
}

.logout-confirm-btn:hover {
  background-color: #b91c1c;
}

.logout-confirm-btn:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}
</style>
