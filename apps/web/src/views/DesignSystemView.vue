<script setup lang="ts">
import { ref } from 'vue'
import {
  AppButton,
  AppBadge,
  AppInput,
  AppProgressBar,
  AppIconBadge,
  AppCard,
  AppDock,
} from '@/components/ui'

const searchValue = ref('')
const activeDock = ref('home')
const copiedColor = ref<string | null>(null)

const primarySwatches = [
  '#000000', '#450a0a', '#7f1d1d', '#991b1b', '#b91c1c',
  '#dc2626', '#ef4444', '#f87171', '#fca5a5', '#fecaca', '#ffffff',
]

const secondarySwatches = [
  '#000000', '#172554', '#1e3a8a', '#1e40af', '#1d4ed8',
  '#2563eb', '#3b82f6', '#60a5fa', '#93c5fd', '#bfdbfe', '#ffffff',
]

const tertiarySwatches = [
  '#000000', '#022c22', '#064e3b', '#065f46', '#047857',
  '#10b981', '#34d399', '#6ee7b7', '#a7f3d0', '#d1fae5', '#ffffff',
]

const neutralSwatches = [
  '#000000', '#080d1a', '#0f172a', '#1e293b', '#334155',
  '#475569', '#64748b', '#94a3b8', '#cbd5e1', '#e2e8f0', '#ffffff',
]

function copyHex(hex: string) {
  navigator.clipboard.writeText(hex)
  copiedColor.value = hex
  setTimeout(() => {
    copiedColor.value = null
  }, 1500)
}
</script>

<template>
  <div class="ds-container">
    <!-- Header -->
    <header class="ds-header">
      <div>
        <div class="ds-header__tag">
          <AppBadge variant="primary" dot pulse>FIGMA NODE 2:1058</AppBadge>
          <span class="ds-header__sub">PENCAK SILAT DIGITAL SCORING</span>
        </div>
        <h1 class="ds-title">Design System & UI Tokens</h1>
      </div>
      <div class="ds-header__actions">
        <AppButton variant="secondary" size="sm" @click="$router.push('/')">
          ← Beranda
        </AppButton>
      </div>
    </header>

    <div v-if="copiedColor" class="ds-copy-toast">
      Hex copied: {{ copiedColor }}
    </div>

    <!-- Main Grid Layout (Mirroring 00_design_system.png) -->
    <div class="ds-grid">
      <!-- COLUMN 1: COLOR PALETTES -->
      <section class="ds-col">
        <h2 class="ds-col__title">Color Palette</h2>

        <!-- Primary Card (Red Corner) -->
        <div class="color-card" @click="copyHex('#DC2626')">
          <div class="color-card__preview color-card__preview--primary">
            <span class="color-card__name">Primary</span>
            <span class="color-card__hex">#DC2626</span>
          </div>
          <div class="color-swatches">
            <div
              v-for="color in primarySwatches"
              :key="color"
              class="color-swatch"
              :style="{ backgroundColor: color }"
              :title="color"
            />
          </div>
        </div>

        <!-- Secondary Card (Blue Corner) -->
        <div class="color-card" @click="copyHex('#2563EB')">
          <div class="color-card__preview color-card__preview--secondary">
            <span class="color-card__name">Secondary</span>
            <span class="color-card__hex">#2563EB</span>
          </div>
          <div class="color-swatches">
            <div
              v-for="color in secondarySwatches"
              :key="color"
              class="color-swatch"
              :style="{ backgroundColor: color }"
              :title="color"
            />
          </div>
        </div>

        <!-- Tertiary Card (Emerald Green Timer & Status) -->
        <div class="color-card" @click="copyHex('#10B981')">
          <div class="color-card__preview color-card__preview--tertiary">
            <span class="color-card__name">Tertiary</span>
            <span class="color-card__hex">#10B981</span>
          </div>
          <div class="color-swatches">
            <div
              v-for="color in tertiarySwatches"
              :key="color"
              class="color-swatch"
              :style="{ backgroundColor: color }"
              :title="color"
            />
          </div>
        </div>

        <!-- Neutral Card (Slate Dark Theme) -->
        <div class="color-card" @click="copyHex('#0F172A')">
          <div class="color-card__preview color-card__preview--neutral">
            <span class="color-card__name">Neutral</span>
            <span class="color-card__hex">#0F172A</span>
          </div>
          <div class="color-swatches">
            <div
              v-for="color in neutralSwatches"
              :key="color"
              class="color-swatch"
              :style="{ backgroundColor: color }"
              :title="color"
            />
          </div>
        </div>
      </section>

      <!-- COLUMN 2: TYPOGRAPHY -->
      <section class="ds-col">
        <h2 class="ds-col__title">Typography</h2>

        <!-- Headline: Space Grotesk -->
        <div class="typo-card">
          <div class="typo-card__header">
            <span class="typo-card__role">Headline</span>
            <span class="typo-card__font font-mono">Space Grotesk</span>
          </div>
          <div class="typo-card__preview font-sans">
            Aa
          </div>
        </div>

        <!-- Body: Space Grotesk -->
        <div class="typo-card">
          <div class="typo-card__header">
            <span class="typo-card__role">Body</span>
            <span class="typo-card__font font-mono">Space Grotesk</span>
          </div>
          <div class="typo-card__preview typo-card__preview--peach font-sans">
            Aa
          </div>
        </div>

        <!-- Label: JetBrains Mono -->
        <div class="typo-card">
          <div class="typo-card__header">
            <span class="typo-card__role">Label</span>
            <span class="typo-card__font font-mono">JetBrains Mono</span>
          </div>
          <div class="typo-card__preview typo-card__preview--peach font-mono">
            Aa
          </div>
        </div>
      </section>

      <!-- COLUMN 3: COMPONENTS & ATOMS -->
      <section class="ds-col ds-col--wide">
        <h2 class="ds-col__title">UI Components & Controls</h2>

        <!-- Buttons Grid (as in Figma) -->
        <AppCard class="mb-4">
          <div class="ds-component-grid">
            <div class="ds-buttons-row">
              <AppButton variant="primary">Primary</AppButton>
              <AppButton variant="secondary">Secondary</AppButton>
              <AppButton variant="inverted">Inverted</AppButton>
              <AppButton variant="outlined">Outlined</AppButton>
            </div>
            <div class="ds-search-row">
              <AppInput
                v-model="searchValue"
                icon
                placeholder="Search"
              />
            </div>
          </div>
        </AppCard>

        <!-- Progress Bars & Navigation Dock (as in Figma) -->
        <AppCard class="mb-4">
          <div class="ds-bars-dock-grid">
            <div class="ds-progress-list">
              <AppProgressBar :value="65" :max="100" variant="primary" :height="8" />
              <AppProgressBar :value="80" :max="100" variant="secondary" :height="8" />
              <AppProgressBar :value="45" :max="100" variant="tertiary" :height="8" />
            </div>
            <div class="ds-dock-wrapper">
              <AppDock :active-id="activeDock" @select="activeDock = $event" />
            </div>
          </div>
        </AppCard>

        <!-- Action Icons & Badges (as in Figma bottom row) -->
        <AppCard class="mb-4">
          <div class="ds-icons-row">
            <!-- Green check square -->
            <AppIconBadge variant="emerald" clickable>
              <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round">
                <polyline points="20 6 9 17 4 12" />
              </svg>
            </AppIconBadge>

            <!-- Red label button with pencil -->
            <button class="ds-label-btn">
              <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                <path d="M17 3a2.85 2.83 0 1 1 4 4L7.5 20.5 2 22l1.5-5.5Z" />
              </svg>
              <span>Label</span>
            </button>

            <!-- 4 action icons: magic, nodes, tag, trash -->
            <div class="ds-icon-group">
              <AppIconBadge variant="peach" clickable title="Magic Action">
                <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                  <path d="m12 3-1.9 5.8a2 2 0 0 1-1.3 1.3L3 12l5.8 1.9a2 2 0 0 1 1.3 1.3L12 21l1.9-5.8a2 2 0 0 1 1.3-1.3L21 12l-5.8-1.9a2 2 0 0 1-1.3-1.3Z" />
                </svg>
              </AppIconBadge>
              <AppIconBadge variant="blue" clickable title="Tournament Tree">
                <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                  <rect width="6" height="6" x="9" y="3" rx="1" />
                  <rect width="6" height="6" x="3" y="15" rx="1" />
                  <rect width="6" height="6" x="15" y="15" rx="1" />
                  <path d="M12 9v3a3 3 0 0 0-3 3v0" />
                  <path d="M12 12a3 3 0 0 1 3 3v0" />
                </svg>
              </AppIconBadge>
              <AppIconBadge variant="emerald" clickable title="Tag / Category">
                <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                  <path d="M12 2H2v10l9.29 9.29c.94.94 2.48.94 3.42 0l6.58-6.58c.94-.94.94-2.48 0-3.42L12 2Z" />
                  <path d="M7 7h.01" />
                </svg>
              </AppIconBadge>
              <AppIconBadge variant="red" clickable title="Delete / Trash">
                <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                  <path d="M3 6h18" />
                  <path d="M19 6v14c0 1-1 2-2 2H7c-1 0-2-1-2-2V6" />
                  <path d="M8 6V4c0-1 1-2 2-2h4c1 0 2 1 2 2v2" />
                </svg>
              </AppIconBadge>
            </div>
          </div>
        </AppCard>

        <!-- Pencak Silat Corner Scoring Tokens Preview -->
        <AppCard>
          <template #header>
            <div class="font-bold text-sm">Pratinjau Sudut Gelanggang (Pencak Silat Tokens)</div>
            <AppBadge variant="emerald" dot>Real-time Ready</AppBadge>
          </template>

          <div class="ds-corner-preview">
            <!-- Sudut Merah -->
            <div class="ds-corner-box ds-corner-box--red">
              <div class="ds-corner-box__header">
                <span class="ds-corner-box__name">Sudut Merah</span>
                <span class="ds-corner-box__athlete">Ahmad Fauzi</span>
              </div>
              <div class="ds-corner-box__score font-mono">14</div>
              <div class="ds-corner-box__actions">
                <AppButton variant="corner-red" size="sm">+1 Pukulan</AppButton>
                <AppButton variant="corner-red" size="sm">+2 Tendangan</AppButton>
                <AppButton variant="corner-red" size="sm">+3 Jatuhan</AppButton>
              </div>
            </div>

            <!-- Sudut Biru -->
            <div class="ds-corner-box ds-corner-box--blue">
              <div class="ds-corner-box__header">
                <span class="ds-corner-box__name">Sudut Biru</span>
                <span class="ds-corner-box__athlete">Bayu Pratama</span>
              </div>
              <div class="ds-corner-box__score font-mono">11</div>
              <div class="ds-corner-box__actions">
                <AppButton variant="corner-blue" size="sm">+1 Pukulan</AppButton>
                <AppButton variant="corner-blue" size="sm">+2 Tendangan</AppButton>
                <AppButton variant="corner-blue" size="sm">+3 Jatuhan</AppButton>
              </div>
            </div>
          </div>
        </AppCard>
      </section>
    </div>
  </div>
</template>

<style scoped>
.ds-container {
  max-width: 1400px;
  margin: 0 auto;
  padding: 32px 24px;
}

.ds-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 28px;
  padding-bottom: 20px;
  border-bottom: 1px solid var(--border-subtle);
}

.ds-header__tag {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 8px;
}

.ds-header__sub {
  font-family: var(--font-mono);
  font-size: 0.75rem;
  letter-spacing: 0.08em;
  color: var(--color-neutral-400);
}

.ds-title {
  font-size: 1.875rem;
  font-weight: 700;
  color: var(--text-main);
  letter-spacing: -0.03em;
}

.ds-copy-toast {
  position: fixed;
  bottom: 24px;
  right: 24px;
  background-color: var(--color-primary-peach);
  color: #0f172a;
  font-family: var(--font-mono);
  font-size: 0.8125rem;
  font-weight: 700;
  padding: 8px 16px;
  border-radius: var(--radius-sm);
  box-shadow: var(--shadow-lg);
  z-index: 1000;
}

/* Grid Layout */
.ds-grid {
  display: grid;
  grid-template-columns: 280px 240px 1fr;
  gap: 20px;
}

@media (max-width: 1024px) {
  .ds-grid {
    grid-template-columns: 1fr;
  }
}

.ds-col__title {
  font-size: 0.875rem;
  font-family: var(--font-mono);
  text-transform: uppercase;
  letter-spacing: 0.08em;
  color: var(--color-neutral-400);
  margin-bottom: 14px;
}

/* --- Color Cards --- */
.color-card {
  background-color: var(--color-neutral-850);
  border: 1px solid var(--color-neutral-700);
  border-radius: var(--radius-lg);
  overflow: hidden;
  margin-bottom: 14px;
  cursor: pointer;
  transition: all var(--transition-fast);
}

.color-card:hover {
  transform: translateY(-2px);
  border-color: rgba(255, 255, 255, 0.2);
}

.color-card__preview {
  height: 90px;
  padding: 12px 16px;
  display: flex;
  flex-direction: column;
  justify-content: space-between;
}

.color-card__preview--primary {
  background-color: var(--color-primary-600);
}

.color-card__preview--secondary {
  background-color: var(--color-secondary-600);
}

.color-card__preview--tertiary {
  background-color: var(--color-tertiary-500);
}

.color-card__preview--neutral {
  background-color: var(--color-neutral-900);
}

.color-card__name {
  font-family: var(--font-sans);
  font-weight: 700;
  font-size: 1rem;
  color: #ffffff;
}

.color-card__hex {
  font-family: var(--font-mono);
  font-weight: 600;
  font-size: 0.8125rem;
  color: rgba(255, 255, 255, 0.9);
}

.color-swatches {
  display: flex;
  height: 28px;
  background-color: #000000;
}

.color-swatch {
  flex: 1;
  height: 100%;
  transition: transform 0.1s ease;
}

.color-swatch:hover {
  transform: scaleY(1.3);
  z-index: 10;
}

/* --- Typo Cards --- */
.typo-card {
  background-color: var(--color-neutral-850);
  border: 1px solid var(--color-neutral-700);
  border-radius: var(--radius-lg);
  padding: 20px;
  margin-bottom: 14px;
}

.typo-card__header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 12px;
}

.typo-card__role {
  font-weight: 600;
  font-size: 0.875rem;
  color: var(--color-neutral-400);
}

.typo-card__font {
  font-size: 0.75rem;
  color: var(--color-neutral-500);
}

.typo-card__preview {
  font-size: 4.5rem;
  font-weight: 700;
  line-height: 1;
  color: #c9d8ea;
}

.typo-card__preview--peach {
  color: var(--color-primary-peach);
}

/* --- Component Section --- */
.ds-component-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 16px;
  align-items: center;
}

@media (max-width: 768px) {
  .ds-component-grid {
    grid-template-columns: 1fr;
  }
}

.ds-buttons-row {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.ds-bars-dock-grid {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 24px;
}

.ds-progress-list {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.ds-icons-row {
  display: flex;
  align-items: center;
  gap: 16px;
  flex-wrap: wrap;
}

.ds-label-btn {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  background-color: var(--color-primary-600);
  color: #ffffff;
  border: none;
  border-radius: var(--radius-xs);
  padding: 6px 14px;
  font-family: var(--font-mono);
  font-size: 0.8125rem;
  font-weight: 700;
  cursor: pointer;
}

.ds-icon-group {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-left: auto;
}

/* Corner Scoring Preview */
.ds-corner-preview {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 16px;
  margin-top: 8px;
}

@media (max-width: 640px) {
  .ds-corner-preview {
    grid-template-columns: 1fr;
  }
}

.ds-corner-box {
  padding: 18px;
  border-radius: var(--radius-md);
  border: 1px solid transparent;
}

.ds-corner-box--red {
  background: linear-gradient(180deg, rgba(220, 38, 38, 0.15) 0%, rgba(19, 29, 51, 0.95) 100%);
  border-color: rgba(220, 38, 38, 0.35);
}

.ds-corner-box--blue {
  background: linear-gradient(180deg, rgba(37, 99, 235, 0.15) 0%, rgba(19, 29, 51, 0.95) 100%);
  border-color: rgba(37, 99, 235, 0.35);
}

.ds-corner-box__header {
  display: flex;
  justify-content: space-between;
  align-items: baseline;
  margin-bottom: 8px;
}

.ds-corner-box--red .ds-corner-box__name {
  color: #f87171;
  font-weight: 700;
  font-size: 0.875rem;
  text-transform: uppercase;
}

.ds-corner-box--blue .ds-corner-box__name {
  color: #60a5fa;
  font-weight: 700;
  font-size: 0.875rem;
  text-transform: uppercase;
}

.ds-corner-box__athlete {
  font-size: 0.8125rem;
  color: var(--color-neutral-300);
}

.ds-corner-box__score {
  font-size: 3rem;
  font-weight: 800;
  line-height: 1;
  margin-bottom: 12px;
}

.ds-corner-box--red .ds-corner-box__score {
  color: #f87171;
}

.ds-corner-box--blue .ds-corner-box__score {
  color: #60a5fa;
}

.ds-corner-box__actions {
  display: flex;
  gap: 6px;
  flex-wrap: wrap;
}

.mb-4 {
  margin-bottom: 16px;
}
</style>
