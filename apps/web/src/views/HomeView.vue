<script setup lang="ts">
import { AppButton, AppBadge, AppCard } from '@/components/ui'
import { useAuthStore } from '@/stores/auth'

const authStore = useAuthStore()

async function handleLogout() {
  await authStore.logout()
}
</script>

<template>
  <main class="home-container">
    <div class="home-hero">
      <div class="home-badge-row">
        <AppBadge variant="emerald" dot pulse>SISTEM PENCAK SILAT DIGITAL SCORING</AppBadge>
        <span class="home-sub font-mono">NEXT-GEN PERSILAT / IPSI 2024</span>
      </div>
      <h1 class="home-title">Komando Digital Scoring Silat</h1>
      <p class="home-desc">
        Arsitektur Frontend Real-Time terintegrasi penuh dengan Go Backend WebSocket & Konsensus
        Juri.
      </p>

      <div v-if="authStore.isAuthenticated" class="home-user-session font-mono">
        <div class="user-pill">
          <span class="user-dot" />
          <span
            >Sesi Aktif: <strong>{{ authStore.name }}</strong> ({{
              (authStore.role || 'user').toUpperCase()
            }})</span
          >
        </div>
        <div class="session-btns">
          <AppButton
            variant="emerald"
            size="lg"
            @click="
              $router.push(
                authStore.role === 'admin' || authStore.role === 'dewan'
                  ? '/ops?tab=kompetisi'
                  : authStore.role === 'contingent'
                    ? '/ops?tab=atlet'
                    : '/ops',
              )
            "
          >
            Buka Panel Operasi →
          </AppButton>
          <button class="home-logout-btn font-mono" @click="handleLogout">Keluar (Logout)</button>
        </div>
      </div>
      <div v-else class="home-actions">
        <AppButton variant="primary" size="lg" @click="$router.push('/login')">
          🔐 Masuk ke Akun
        </AppButton>
        <AppButton variant="secondary" size="lg" @click="$router.push('/register')">
          📝 Daftar Kontingen
        </AppButton>
        <AppButton variant="emerald" size="lg" @click="$router.push('/judge/login')">
          📱 Tablet Juri PIN
        </AppButton>
      </div>
    </div>

    <!-- Roadmap Eksekusi Layar -->
    <div class="home-modules">
      <h2 class="home-modules__title font-mono">MODUL & ROADMAP EKSEKUSI DESAIN</h2>

      <div class="home-cards-grid">
        <!-- 0. Design System (Selesai) -->
        <AppCard variant="emerald" glow>
          <template #header>
            <span class="font-bold font-mono text-emerald-400">00. DESIGN SYSTEM</span>
            <AppBadge variant="emerald" dot>SELESAI</AppBadge>
          </template>
          <p class="card-desc">
            Tokens warna (#DC2626 Merah, #2563EB Biru, #10B981 Hijau), font Space Grotesk &
            JetBrains Mono, tombol, input, progress bar, dan dock.
          </p>
          <template #footer>
            <AppButton variant="outlined" size="sm" @click="$router.push('/design-system')">
              Lihat Design System →
            </AppButton>
          </template>
        </AppCard>

        <!-- 1. Scoreboard Display (Selesai) -->
        <AppCard variant="emerald" glow>
          <template #header>
            <span class="font-bold font-mono text-emerald-400">01. ARENA SCOREBOARD</span>
            <AppBadge variant="emerald" dot>SELESAI</AppBadge>
          </template>
          <p class="card-desc">
            Layar LED TV stadium 16:9 fullscreen resolusi tinggi dengan timer digital raksasa,
            breakdown nilai per babak, dan status konsensus juri.
          </p>
          <template #footer>
            <AppButton variant="outlined" size="sm" @click="$router.push('/scoreboard')">
              Buka Live Scoreboard →
            </AppButton>
          </template>
        </AppCard>

        <!-- 2. Judge Tablet PWA (Selesai) -->
        <AppCard variant="emerald" glow>
          <template #header>
            <span class="font-bold font-mono text-emerald-400">02. TABLET JURI PWA</span>
            <AppBadge variant="emerald" dot>SELESAI</AppBadge>
          </template>
          <p class="card-desc">
            Layar sentuh ergonomis landscape tablet juri gelanggang dengan tombol pukulan (+1),
            tendangan (+2), jatuhan (+3), dan anti-bias Pasal 28.
          </p>
          <template #footer>
            <AppButton variant="outlined" size="sm" @click="$router.push('/judge')">
              Buka Tablet Juri →
            </AppButton>
          </template>
        </AppCard>

        <!-- 3. Referee Console (Selesai) -->
        <AppCard variant="emerald" glow>
          <template #header>
            <span class="font-bold font-mono text-emerald-400">03. KONSOL WASIT</span>
            <AppBadge variant="emerald" dot>SELESAI</AppBadge>
          </template>
          <p class="card-desc">
            Meja pengawas pertandingan: tombol kontrol timer, gong, dokter pertandingan, video VAR,
            dan panel sanksi teguran s/d diskualifikasi.
          </p>
          <template #footer>
            <AppButton variant="outlined" size="sm" @click="$router.push('/referee')">
              Buka Konsol Wasit →
            </AppButton>
          </template>
        </AppCard>

        <!-- 4. Tournament Bracket Ops (Selesai) -->
        <AppCard variant="emerald" glow>
          <template #header>
            <span class="font-bold font-mono text-emerald-400">04. BAGAN TURNAMEN OPS</span>
            <AppBadge variant="emerald" dot>SELESAI</AppBadge>
          </template>
          <p class="card-desc">
            Dashboard admin komando operasi, pohon bagan single-elimination, integrasi timbang badan
            sah, dan tombol load arena (Kirim ke Gelanggang).
          </p>
          <template #footer>
            <AppButton variant="outlined" size="sm" @click="$router.push('/ops')">
              Buka Bagan Turnamen →
            </AppButton>
          </template>
        </AppCard>

        <!-- 5. Admin & Perangkat PIN (Selesai) -->
        <AppCard variant="emerald" glow>
          <template #header>
            <span class="font-bold font-mono text-emerald-400">05. MEJA KOMANDO & PIN JURI</span>
            <AppBadge variant="emerald" dot>SELESAI</AppBadge>
          </template>
          <p class="card-desc">
            Panel Admin Turnamen & Dewan Hakim: generate PIN juri 6-digit per kursi gelanggang,
            monitor baterai & latency perangkat, serta cetak slip meja juri.
          </p>
          <template #footer>
            <AppButton variant="outlined" size="sm" @click="$router.push('/admin')">
              Buka Manajemen PIN →
            </AppButton>
          </template>
        </AppCard>

        <!-- 6. Akun & Autentikasi -->
        <AppCard variant="emerald" glow>
          <template #header>
            <span class="font-bold font-mono text-emerald-400">06. AKUN & AUTENTIKASI</span>
            <AppBadge variant="emerald" dot>RESMI PERSILAT</AppBadge>
          </template>
          <p class="card-desc">
            Halaman login terpadu (Admin, Operator, Dewan Hakim, dan Official Kontingen),
            pendaftaran kontingen baru, serta login tablet juri via PIN gelanggang.
          </p>
          <template #footer>
            <AppButton variant="outlined" size="sm" @click="$router.push('/login')">
              Buka Halaman Login →
            </AppButton>
          </template>
        </AppCard>
      </div>
    </div>
  </main>
</template>

<style scoped>
.home-container {
  max-width: 1200px;
  margin: 0 auto;
  padding: 60px 24px;
}

.home-hero {
  text-align: center;
  max-width: 800px;
  margin: 0 auto 60px auto;
}

.home-badge-row {
  display: inline-flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 20px;
}

.home-sub {
  font-size: 0.8125rem;
  letter-spacing: 0.08em;
  color: var(--color-neutral-400);
}

.home-title {
  font-size: 3rem;
  font-weight: 800;
  letter-spacing: -0.04em;
  color: var(--text-main);
  margin-bottom: 16px;
  line-height: 1.15;
}

.home-desc {
  font-size: 1.125rem;
  color: var(--color-neutral-400);
  line-height: 1.6;
  margin-bottom: 32px;
}

.home-actions {
  display: flex;
  justify-content: center;
  gap: 16px;
}

.home-modules__title {
  font-size: 0.875rem;
  text-transform: uppercase;
  letter-spacing: 0.08em;
  color: var(--color-neutral-400);
  margin-bottom: 20px;
}

.home-cards-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(340px, 1fr));
  gap: 20px;
}

.card-desc {
  font-size: 0.875rem;
  color: var(--color-neutral-300);
  line-height: 1.5;
}

/* Home User Session */
.home-user-session {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 16px;
  background: rgba(16, 185, 129, 0.06);
  border: 1px solid rgba(16, 185, 129, 0.25);
  border-radius: 12px;
  padding: 20px 24px;
  max-width: 520px;
  margin: 0 auto;
}

.user-pill {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 0.875rem;
  color: #e2e8f0;
}

.user-pill strong {
  color: #34d399;
}

.user-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background-color: #10b981;
  box-shadow: 0 0 8px #10b981;
}

.session-btns {
  display: flex;
  align-items: center;
  gap: 12px;
}

.home-logout-btn {
  background: transparent;
  border: 1px solid rgba(239, 68, 68, 0.35);
  color: #f87171;
  padding: 10px 18px;
  border-radius: 6px;
  font-size: 0.8125rem;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.15s ease;
}

.home-logout-btn:hover {
  background: rgba(239, 68, 68, 0.15);
  border-color: #ef4444;
}
</style>
