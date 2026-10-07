<script setup lang="ts">
import { ref, computed, onMounted, onBeforeUnmount } from 'vue'

export interface AthleteEntry {
  name: string
  contingent: string
  score?: number | string
  isWinner?: boolean
}

export interface MatchCard {
  id: string
  partaiNumber: number
  arenaLabel: string
  status: 'selesai' | 'live' | 'ready' | 'waiting'
  statusBadge: string
  statusVariant: 'neutral' | 'emerald' | 'amber' | 'blue' | 'red'
  red: AthleteEntry
  blue: AthleteEntry
  metaLeft?: string
  metaRight?: string
  isLive?: boolean
  canLoad?: boolean
}

export interface BracketRoundDisplay {
  roundNumber: number
  title: string
  badgeVariant: 'neutral' | 'blue' | 'red' | 'emerald'
  matches: MatchCard[]
}

const props = withDefaults(
  defineProps<{
    rounds: BracketRoundDisplay[]
    selectedMatchId?: string | null
    readOnly?: boolean
    categoryName?: string
  }>(),
  {
    selectedMatchId: null,
    readOnly: false,
    categoryName: '',
  },
)

const emit = defineEmits<{
  (e: 'select-match', match: MatchCard, round: BracketRoundDisplay): void
  (e: 'load-arena', match: MatchCard): void
  (e: 'generate-draw'): void
}>()

// Base Constants for Mathematical Tree Alignment
const BASE_SLOT_HEIGHT = 144 // Height of a match slot in the first round

function getSlotHeight(roundIndex: number): number {
  return BASE_SLOT_HEIGHT * Math.pow(2, roundIndex)
}

// Maximize (Fullscreen) State
const isMaximized = ref<boolean>(false)

function toggleMaximize() {
  isMaximized.value = !isMaximized.value
}

function handleKeyDown(e: KeyboardEvent) {
  if (e.key === 'Escape' && isMaximized.value) {
    isMaximized.value = false
  }
}

// Pan & Zoom State (Figma/Miro Style Free Pan via Transform)
const panX = ref<number>(0)
const panY = ref<number>(0)
const zoomLevel = ref<number>(1)
const isDragging = ref<boolean>(false)
const hasDragged = ref<boolean>(false)
const dragStartX = ref<number>(0)
const dragStartY = ref<number>(0)
const startPanX = ref<number>(0)
const startPanY = ref<number>(0)

const bracketViewport = ref<HTMLElement | null>(null)

function zoomIn(step = 0.08) {
  zoomLevel.value = Math.min(1.8, Number((zoomLevel.value + step).toFixed(2)))
}

function zoomOut(step = 0.08) {
  zoomLevel.value = Math.max(0.4, Number((zoomLevel.value - step).toFixed(2)))
}

function resetZoom() {
  zoomLevel.value = 1
  panX.value = 0
  panY.value = 0
}

// Global Drag-to-Pan (Works everywhere in viewport)
function handleMouseDown(e: MouseEvent) {
  if (e.button !== 0 || !bracketViewport.value) return
  const target = e.target as HTMLElement
  if (target.closest('button') || target.closest('input') || target.closest('select')) return

  isDragging.value = true
  hasDragged.value = false
  dragStartX.value = e.clientX
  dragStartY.value = e.clientY
  startPanX.value = panX.value
  startPanY.value = panY.value

  window.addEventListener('mousemove', onGlobalMouseMove, { passive: false })
  window.addEventListener('mouseup', onGlobalMouseUp)
}

function onGlobalMouseMove(e: MouseEvent) {
  if (!isDragging.value) return
  const dx = e.clientX - dragStartX.value
  const dy = e.clientY - dragStartY.value

  if (Math.abs(dx) > 3 || Math.abs(dy) > 3) {
    hasDragged.value = true
    e.preventDefault()
  }

  panX.value = startPanX.value + dx
  panY.value = startPanY.value + dy
}

function onGlobalMouseUp() {
  isDragging.value = false
  window.removeEventListener('mousemove', onGlobalMouseMove)
  window.removeEventListener('mouseup', onGlobalMouseUp)
}

// Ctrl + Scroll Zoom Handler
function handleWheel(e: WheelEvent) {
  if (e.ctrlKey || e.metaKey) {
    e.preventDefault()
    if (e.deltaY < 0) {
      zoomIn(0.06)
    } else if (e.deltaY > 0) {
      zoomOut(0.06)
    }
  }
}

onMounted(() => {
  if (bracketViewport.value) {
    bracketViewport.value.addEventListener('wheel', handleWheel, { passive: false })
  }
  window.addEventListener('keydown', handleKeyDown)
})

onBeforeUnmount(() => {
  if (bracketViewport.value) {
    bracketViewport.value.removeEventListener('wheel', handleWheel)
  }
  window.removeEventListener('mousemove', onGlobalMouseMove)
  window.removeEventListener('mouseup', onGlobalMouseUp)
  window.removeEventListener('keydown', handleKeyDown)
})

// Final champion lookup
const champion = computed(() => {
  if (!props.rounds.length) return null
  const finalRound = props.rounds[props.rounds.length - 1]
  const finalMatch = finalRound.matches[0]
  if (!finalMatch || finalMatch.status !== 'selesai') return null

  if (finalMatch.red.isWinner) {
    return {
      corner: 'red',
      name: finalMatch.red.name,
      contingent: finalMatch.red.contingent,
      partai: finalMatch.partaiNumber,
    }
  }
  if (finalMatch.blue.isWinner) {
    return {
      corner: 'blue',
      name: finalMatch.blue.name,
      contingent: finalMatch.blue.contingent,
      partai: finalMatch.partaiNumber,
    }
  }
  return null
})

function onMatchClick(m: MatchCard, r: BracketRoundDisplay) {
  if (hasDragged.value) return
  emit('select-match', m, r)
}

// Helpers to get feeder matches in previous round for connector lines
function getTopFeeder(rIdx: number, targetIdx: number): MatchCard | null {
  if (rIdx === 0) return null
  return props.rounds[rIdx - 1]?.matches[targetIdx * 2] || null
}

function getBottomFeeder(rIdx: number, targetIdx: number): MatchCard | null {
  if (rIdx === 0) return null
  return props.rounds[rIdx - 1]?.matches[targetIdx * 2 + 1] || null
}
</script>

<template>
  <div class="tournament-bracket-root" :class="{ 'is-maximized': isMaximized }">
    <!-- TOOLBAR / CONTROLS -->
    <div class="t-bracket-toolbar font-mono">
      <div class="t-bracket-toolbar__meta">
        <span class="t-bracket-live-dot" />
        <span class="t-bracket-toolbar__title">
          {{ categoryName ? `BAGAN RESMI — ${categoryName}` : 'POHON BAGAN PERTANDINGAN' }}
        </span>
        <span class="t-bracket-badge">{{ rounds.length }} BABAK</span>
      </div>

      <div class="t-bracket-toolbar__actions">
        <div class="t-zoom-controls">
          <span class="t-shortcut-hint" title="Tahan Ctrl lalu putar scroll mouse untuk zoom bebas">
            <kbd>Ctrl</kbd> + Scroll
          </span>
          <div class="t-zoom-divider" />
          <button
            type="button"
            class="t-btn-icon"
            title="Perkecil Bagan"
            :disabled="zoomLevel <= 0.4"
            @click="zoomOut()"
          >
            <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <circle cx="11" cy="11" r="8" />
              <line x1="21" y1="21" x2="16.65" y2="16.65" />
              <line x1="8" y1="11" x2="14" y2="11" />
            </svg>
          </button>
          <span class="t-zoom-label">{{ Math.round(zoomLevel * 100) }}%</span>
          <button
            type="button"
            class="t-btn-icon"
            title="Perbesar Bagan"
            :disabled="zoomLevel >= 1.8"
            @click="zoomIn()"
          >
            <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <circle cx="11" cy="11" r="8" />
              <line x1="21" y1="21" x2="16.65" y2="16.65" />
              <line x1="11" y1="8" x2="11" y2="14" />
              <line x1="8" y1="11" x2="14" y2="11" />
            </svg>
          </button>
          <button
            type="button"
            class="t-btn-text"
            title="Kembalikan Posisi & Zoom Awal"
            @click="resetZoom"
          >
            Reset
          </button>
          <div class="t-zoom-divider" />
          <!-- MAXIMIZE BUTTON -->
          <button
            type="button"
            class="t-btn-icon t-btn-icon--maximize"
            :class="{ 'is-active': isMaximized }"
            :title="isMaximized ? 'Keluar dari Layar Penuh (Esc)' : 'Tampilkan Bagan Layar Penuh (Maximize)'"
            @click="toggleMaximize"
          >
            <svg v-if="!isMaximized" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path d="M8 3H5a2 2 0 0 0-2 2v3m18 0V5a2 2 0 0 0-2-2h-3m0 18h3a2 2 0 0 0 2-2v-3M3 16v3a2 2 0 0 0 2 2h3" />
            </svg>
            <svg v-else width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path d="M4 14h6m0 0v6m0-6-7 7m17-11h-6m0 0V4m0 6 7-7m-7 17v-6m0 0h6m-6 0 7 7M10 4v6m0 0H4m6 0L3 3" />
            </svg>
          </button>
        </div>
      </div>
    </div>

    <!-- BRACKET VIEWPORT (FREE DRAG-TO-PAN CANVASES) -->
    <div
      ref="bracketViewport"
      class="t-bracket-viewport"
      :class="{ 'is-dragging': isDragging }"
      @mousedown="handleMouseDown"
    >
      <div
        class="t-bracket-canvas"
        :style="{
          transform: `translate3d(${panX}px, ${panY}px, 0) scale(${zoomLevel})`,
          transformOrigin: '0 0',
        }"
      >
        <!-- TOURNAMENT TREE ROUNDS AND CONNECTOR COLUMNS -->
        <template v-for="(round, rIdx) in rounds" :key="round.roundNumber">
          <!-- CONNECTOR COLUMN (BRIDGING ROUND rIdx-1 TO ROUND rIdx) -->
          <div v-if="rIdx > 0" class="t-connector-column">
            <div class="t-connector-column__spacer" />
            <div class="t-connector-slots">
              <div
                v-for="(_, targetIdx) in round.matches"
                :key="targetIdx"
                class="t-connector-slot"
                :style="{ height: `${getSlotHeight(rIdx)}px` }"
              >
                <!-- Mathematical Vector Fork connecting Feeder 1 & Feeder 2 to Target -->
                <svg
                  class="t-fork-svg"
                  :viewBox="`0 0 48 ${getSlotHeight(rIdx)}`"
                  preserveAspectRatio="none"
                >
                  <!-- Top Feeder Line (enters at 25% height, merges at 50% height) -->
                  <path
                    :d="`M 0 ${getSlotHeight(rIdx) * 0.25} H 24 V ${getSlotHeight(rIdx) * 0.5} H 48`"
                    class="t-fork-path"
                    :class="{
                      't-fork-path--live': getTopFeeder(rIdx, targetIdx)?.isLive,
                      't-fork-path--winner': getTopFeeder(rIdx, targetIdx)?.red?.isWinner || getTopFeeder(rIdx, targetIdx)?.blue?.isWinner,
                    }"
                  />
                  <!-- Bottom Feeder Line (enters at 75% height, merges at 50% height) -->
                  <path
                    v-if="getBottomFeeder(rIdx, targetIdx)"
                    :d="`M 0 ${getSlotHeight(rIdx) * 0.75} H 24 V ${getSlotHeight(rIdx) * 0.5} H 48`"
                    class="t-fork-path"
                    :class="{
                      't-fork-path--live': getBottomFeeder(rIdx, targetIdx)?.isLive,
                      't-fork-path--winner': getBottomFeeder(rIdx, targetIdx)?.red?.isWinner || getBottomFeeder(rIdx, targetIdx)?.blue?.isWinner,
                    }"
                  />
                </svg>
              </div>
            </div>
          </div>

          <!-- ROUND MATCHES COLUMN -->
          <div class="t-round-column">
            <!-- ROUND HEADER -->
            <div class="t-round-header font-mono">
              <div class="t-round-header__left">
                <span class="t-round-dot" :class="round.badgeVariant" />
                <span class="t-round-title">{{ round.title }}</span>
              </div>
              <span class="t-round-badge" :class="round.badgeVariant">
                {{ round.matches.length }} PARTAI
              </span>
            </div>

            <!-- MATCH SLOTS LIST (HEIGHT DOUBLES MATHEMATICALLY EACH ROUND) -->
            <div class="t-slots-list">
              <div
                v-for="match in round.matches"
                :key="match.id"
                class="t-match-slot"
                :style="{ height: `${getSlotHeight(rIdx)}px` }"
              >
                <!-- CARD WRAPPER (CENTERED VERTICALLY IN ITS EXACT SLOT) -->
                <div class="t-card-wrapper">
                  <div
                    class="t-match-card"
                    :class="{
                      't-match-card--selected': selectedMatchId === match.id,
                      't-match-card--live': match.isLive,
                      't-match-card--clickable': !readOnly,
                    }"
                    @click="onMatchClick(match, round)"
                  >
                    <!-- CARD HEADER -->
                    <div class="t-card-header font-mono">
                      <div class="t-card-header__partai">
                        <span v-if="match.isLive" class="t-live-pulse" />
                        <span>PARTAI {{ String(match.partaiNumber).padStart(2, '0') }}</span>
                        <span class="t-arena-tag">{{ match.arenaLabel }}</span>
                      </div>
                      <span
                        class="t-card-badge"
                        :class="match.isLive ? 'emerald' : match.status === 'selesai' ? 'neutral' : 'ready'"
                      >
                        {{ match.statusBadge }}
                      </span>
                    </div>

                    <!-- SUDUT MERAH (RED CORNER) -->
                    <div
                      class="t-fighter-row t-fighter-row--red"
                      :class="{ 't-fighter-row--winner': match.red.isWinner }"
                    >
                      <div class="t-fighter-corner-indicator red">
                        <span>M</span>
                      </div>
                      <div class="t-fighter-info">
                        <span class="t-fighter-name" :title="match.red.name">
                          {{ match.red.name }}
                        </span>
                        <span class="t-fighter-club font-mono" :title="match.red.contingent">
                          {{ match.red.contingent }}
                        </span>
                      </div>
                      <div class="t-fighter-score font-mono">
                        <span v-if="match.red.isWinner" class="t-win-badge">WIN</span>
                        <span class="t-score-num">{{ match.red.score }}</span>
                      </div>
                    </div>

                    <!-- SUDUT BIRU (BLUE CORNER) -->
                    <div
                      class="t-fighter-row t-fighter-row--blue"
                      :class="{ 't-fighter-row--winner': match.blue.isWinner }"
                    >
                      <div class="t-fighter-corner-indicator blue">
                        <span>B</span>
                      </div>
                      <div class="t-fighter-info">
                        <span class="t-fighter-name" :title="match.blue.name">
                          {{ match.blue.name }}
                        </span>
                        <span class="t-fighter-club font-mono" :title="match.blue.contingent">
                          {{ match.blue.contingent }}
                        </span>
                      </div>
                      <div class="t-fighter-score font-mono">
                        <span v-if="match.blue.isWinner" class="t-win-badge">WIN</span>
                        <span class="t-score-num">{{ match.blue.score }}</span>
                      </div>
                    </div>
                  </div>
                </div>
              </div>
            </div>
          </div>
        </template>

        <!-- CHAMPION PODIUM COLUMN (APPEARS ONCE FINAL IS COMPLETE) -->
        <template v-if="champion">
          <!-- Connector line to Champion -->
          <div class="t-connector-column">
            <div class="t-connector-column__spacer" />
            <div class="t-connector-slots">
              <div class="t-connector-slot" :style="{ height: `${getSlotHeight(rounds.length - 1)}px` }">
                <svg
                  class="t-fork-svg"
                  :viewBox="`0 0 48 ${getSlotHeight(rounds.length - 1)}`"
                  preserveAspectRatio="none"
                >
                  <path
                    :d="`M 0 ${getSlotHeight(rounds.length - 1) * 0.5} H 48`"
                    class="t-fork-path t-fork-path--champion"
                  />
                </svg>
              </div>
            </div>
          </div>

          <!-- Podium Card -->
          <div class="t-round-column t-round-column--champion">
            <div class="t-round-header font-mono">
              <div class="t-round-header__left">
                <span class="t-round-dot emerald" />
                <span class="t-round-title">JUARA 1</span>
              </div>
              <span class="t-round-badge emerald">CHAMPION</span>
            </div>

            <div class="t-slots-list">
              <div class="t-match-slot" :style="{ height: `${getSlotHeight(rounds.length - 1)}px` }">
                <div class="t-card-wrapper">
                  <div class="t-champion-card font-mono" :class="champion.corner">
                    <div class="t-champion-trophy">
                      <svg width="28" height="28" viewBox="0 0 24 24" fill="none" stroke="#f59e0b" stroke-width="2">
                        <path d="M6 9H4.5a2.5 2.5 0 0 1 0-5H6" />
                        <path d="M18 9h1.5a2.5 2.5 0 0 0 0-5H18" />
                        <path d="M4 22h16" />
                        <path d="M10 14.66V17c0 .55-.45 1-1 1H7c-.55 0-1 .45-1 1v1c0 .55.45 1 1 1h10c.55 0 1-.45 1-1v-1c0-.55-.45-1-1-1h-2c-.55 0-1-.45-1-1v-2.34" />
                        <path d="M6 4h12v5a6 6 0 0 1-12 0V4Z" />
                      </svg>
                    </div>
                    <div class="t-champion-label">MEDALI EMAS</div>
                    <div class="t-champion-name">{{ champion.name }}</div>
                    <div class="t-champion-club">{{ champion.contingent }}</div>
                    <div class="t-champion-meta">PEMENANG PARTAI {{ String(champion.partai).padStart(2, '0') }}</div>
                  </div>
                </div>
              </div>
            </div>
          </div>
        </template>
      </div>
    </div>
  </div>
</template>

<style scoped>
/* ==========================================================================
   TOURNAMENT TREE BRACKET - MATHEMATICAL VECTOR FORK (ZERO MISALIGNMENT)
   ========================================================================== */

.tournament-bracket-root {
  display: flex;
  flex-direction: column;
  background-color: #080e1a;
  border: 1px solid rgba(255, 255, 255, 0.08);
  border-radius: 12px;
  overflow: hidden;
  box-shadow: 0 10px 30px rgba(0, 0, 0, 0.4);
  width: 100%;
  transition: all 0.25s cubic-bezier(0.16, 1, 0.3, 1);
}

/* Maximized Mode (Fullscreen overlay) */
.tournament-bracket-root.is-maximized {
  position: fixed !important;
  inset: 0 !important;
  width: 100vw !important;
  height: 100vh !important;
  z-index: 9999 !important;
  border-radius: 0 !important;
  box-shadow: none !important;
}

.tournament-bracket-root.is-maximized .t-bracket-viewport {
  height: calc(100vh - 56px) !important;
  min-height: auto !important;
}

/* Toolbar */
.t-bracket-toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 10px 18px;
  background-color: #0b1424;
  border-bottom: 1px solid rgba(255, 255, 255, 0.07);
  gap: 12px;
  flex-wrap: wrap;
  height: 52px;
  box-sizing: border-box;
}

.t-bracket-toolbar__meta {
  display: flex;
  align-items: center;
  gap: 10px;
}

.t-bracket-live-dot {
  width: 8px;
  height: 8px;
  background-color: #10b981;
  border-radius: 50%;
  box-shadow: 0 0 8px #10b981;
}

.t-bracket-toolbar__title {
  font-size: 0.8125rem;
  font-weight: 700;
  letter-spacing: 0.06em;
  color: #f1f5f9;
}

.t-bracket-badge {
  font-size: 0.6875rem;
  font-weight: 700;
  background-color: rgba(255, 255, 255, 0.06);
  border: 1px solid rgba(255, 255, 255, 0.12);
  color: #94a3b8;
  padding: 2px 8px;
  border-radius: 4px;
}

/* Zoom Controls */
.t-zoom-controls {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  background-color: #111c30;
  border: 1px solid rgba(255, 255, 255, 0.1);
  padding: 3px 8px;
  border-radius: 6px;
}

.t-shortcut-hint {
  font-size: 0.6875rem;
  color: #64748b;
  display: flex;
  align-items: center;
  gap: 4px;
  padding-right: 4px;
}

.t-shortcut-hint kbd {
  background-color: rgba(255, 255, 255, 0.08);
  border: 1px solid rgba(255, 255, 255, 0.15);
  border-radius: 3px;
  padding: 1px 4px;
  font-size: 0.625rem;
  color: #94a3b8;
}

.t-zoom-divider {
  width: 1px;
  height: 14px;
  background-color: rgba(255, 255, 255, 0.1);
  margin: 0 2px;
}

.t-btn-icon {
  background: none;
  border: none;
  color: #94a3b8;
  padding: 4px;
  border-radius: 4px;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: all 0.15s ease;
}

.t-btn-icon:hover:not(:disabled) {
  color: #ffffff;
  background-color: rgba(255, 255, 255, 0.1);
}

.t-btn-icon:disabled {
  opacity: 0.35;
  cursor: not-allowed;
}

.t-btn-icon--maximize.is-active {
  color: #38bdf8;
  background-color: rgba(56, 189, 248, 0.15);
}

.t-zoom-label {
  font-size: 0.75rem;
  font-weight: 700;
  color: #cbd5e1;
  min-width: 42px;
  text-align: center;
}

.t-btn-text {
  background: none;
  border: none;
  color: #64748b;
  font-size: 0.6875rem;
  padding: 2px 6px;
  border-radius: 4px;
  cursor: pointer;
  transition: color 0.15s ease;
}

.t-btn-text:hover {
  color: #38bdf8;
}

/* Viewport & Canvas */
.t-bracket-viewport {
  overflow: hidden;
  padding: 28px 36px 40px 36px;
  user-select: none;
  cursor: grab;
  position: relative;
  min-height: 560px;
  background-color: #060b14;
  background-image: radial-gradient(rgba(255, 255, 255, 0.04) 1px, transparent 1px);
  background-size: 24px 24px;
}

.t-bracket-viewport.is-dragging {
  cursor: grabbing;
}

.t-bracket-canvas {
  display: flex;
  align-items: flex-start;
  width: max-content;
  will-change: transform;
}

/* Round Columns */
.t-round-column {
  display: flex;
  flex-direction: column;
  width: 290px;
  flex-shrink: 0;
  position: relative;
}

.t-round-column--champion {
  width: 240px;
}

.t-round-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding-bottom: 12px;
  margin-bottom: 12px;
  border-bottom: 1px solid rgba(255, 255, 255, 0.08);
  height: 40px;
  box-sizing: border-box;
}

.t-round-header__left {
  display: flex;
  align-items: center;
  gap: 8px;
}

.t-round-title {
  font-size: 0.75rem;
  font-weight: 800;
  letter-spacing: 0.08em;
  color: #e2e8f0;
}

.t-round-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
}

.t-round-dot.neutral {
  background-color: #64748b;
}
.t-round-dot.blue {
  background-color: #38bdf8;
  box-shadow: 0 0 6px rgba(56, 189, 248, 0.8);
}
.t-round-dot.red {
  background-color: #ef4444;
  box-shadow: 0 0 6px rgba(239, 68, 68, 0.8);
}
.t-round-dot.emerald {
  background-color: #10b981;
  box-shadow: 0 0 6px rgba(16, 185, 129, 0.8);
}

.t-round-badge {
  font-size: 0.625rem;
  font-weight: 800;
  padding: 2px 6px;
  border-radius: 4px;
}

.t-round-badge.neutral {
  background-color: rgba(100, 116, 139, 0.2);
  color: #94a3b8;
}
.t-round-badge.blue {
  background-color: rgba(56, 189, 248, 0.15);
  color: #38bdf8;
  border: 1px solid rgba(56, 189, 248, 0.3);
}
.t-round-badge.red {
  background-color: rgba(239, 68, 68, 0.15);
  color: #f87171;
  border: 1px solid rgba(239, 68, 68, 0.3);
}
.t-round-badge.emerald {
  background-color: rgba(16, 185, 129, 0.15);
  color: #34d399;
  border: 1px solid rgba(16, 185, 129, 0.3);
}

/* Slots List */
.t-slots-list {
  display: flex;
  flex-direction: column;
  position: relative;
}

/* A Single Match Slot (Height doubles each round) */
.t-match-slot {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 100%;
  position: relative;
}

/* Card Wrapper (Always Centered in Slot) */
.t-card-wrapper {
  width: 100%;
  display: flex;
  align-items: center;
  justify-content: center;
}

/* ==========================================================================
   CONNECTOR COLUMNS (MATHEMATICALLY ACCURATE SVG VECTORS)
   ========================================================================== */

.t-connector-column {
  width: 48px;
  flex-shrink: 0;
  display: flex;
  flex-direction: column;
}

.t-connector-column__spacer {
  height: 52px; /* Aligns with header height + margin */
}

.t-connector-slots {
  display: flex;
  flex-direction: column;
  width: 100%;
}

.t-connector-slot {
  width: 48px;
  display: flex;
  align-items: center;
  justify-content: center;
}

.t-fork-svg {
  width: 48px;
  height: 100%;
  display: block;
  overflow: visible;
}

.t-fork-path {
  fill: none;
  stroke: rgba(255, 255, 255, 0.22);
  stroke-width: 2px;
  stroke-linecap: round;
  stroke-linejoin: round;
  transition: stroke 0.2s ease, stroke-width 0.2s ease;
}

.t-fork-path--winner {
  stroke: rgba(56, 189, 248, 0.55);
}

.t-fork-path--live {
  stroke: #10b981 !important;
  stroke-width: 2.5px;
  filter: drop-shadow(0 0 6px rgba(16, 185, 129, 0.7));
}

.t-fork-path--champion {
  stroke: rgba(245, 158, 11, 0.6);
  stroke-width: 2.5px;
}

/* ==========================================================================
   THE MATCH CARD
   ========================================================================== */

.t-match-card {
  width: 100%;
  background-color: #0d1527;
  border: 1px solid rgba(255, 255, 255, 0.08);
  border-radius: 8px;
  overflow: hidden;
  box-shadow: 0 4px 14px rgba(0, 0, 0, 0.35);
  transition: all 0.2s cubic-bezier(0.16, 1, 0.3, 1);
  position: relative;
  z-index: 2;
}

.t-match-card--clickable {
  cursor: pointer;
}

.t-match-card--clickable:hover {
  border-color: rgba(255, 255, 255, 0.22);
  transform: translateY(-2px);
  box-shadow: 0 8px 20px rgba(0, 0, 0, 0.5);
}

.t-match-card--selected {
  border-color: #38bdf8 !important;
  background-color: #0f1d38 !important;
  box-shadow: 0 0 20px rgba(56, 189, 248, 0.25) !important;
}

.t-match-card--live {
  border-color: rgba(16, 185, 129, 0.7) !important;
  background-color: #0a1c22 !important;
  box-shadow: 0 0 18px rgba(16, 185, 129, 0.2) !important;
}

/* Card Header */
.t-card-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 6px 10px;
  background-color: rgba(255, 255, 255, 0.02);
  border-bottom: 1px solid rgba(255, 255, 255, 0.05);
  font-size: 0.6875rem;
  color: #94a3b8;
}

.t-card-header__partai {
  display: flex;
  align-items: center;
  gap: 6px;
  font-weight: 700;
  color: #e2e8f0;
}

.t-arena-tag {
  color: #64748b;
  font-size: 0.625rem;
}

.t-live-pulse {
  width: 6px;
  height: 6px;
  background-color: #10b981;
  border-radius: 50%;
  box-shadow: 0 0 6px #10b981;
  animation: t-pulse 1.4s infinite;
}

@keyframes t-pulse {
  0% { transform: scale(0.95); opacity: 0.7; }
  50% { transform: scale(1.3); opacity: 1; }
  100% { transform: scale(0.95); opacity: 0.7; }
}

.t-card-badge {
  font-size: 0.625rem;
  font-weight: 800;
  padding: 1px 6px;
  border-radius: 4px;
}

.t-card-badge.neutral {
  background-color: rgba(100, 116, 139, 0.2);
  color: #94a3b8;
}
.t-card-badge.emerald {
  background-color: rgba(16, 185, 129, 0.2);
  color: #34d399;
  border: 1px solid rgba(16, 185, 129, 0.4);
}
.t-card-badge.ready {
  background-color: rgba(56, 189, 248, 0.1);
  color: #38bdf8;
}

/* Fighter Row */
.t-fighter-row {
  display: flex;
  align-items: center;
  padding: 7px 10px;
  gap: 8px;
  transition: background-color 0.15s ease;
  border-bottom: 1px solid rgba(255, 255, 255, 0.03);
}

.t-fighter-row:last-child {
  border-bottom: none;
}

.t-fighter-row--red {
  background-color: rgba(220, 38, 38, 0.05);
}

.t-fighter-row--blue {
  background-color: rgba(37, 99, 235, 0.05);
}

.t-fighter-row--winner {
  background-color: rgba(16, 185, 129, 0.12) !important;
}

.t-fighter-corner-indicator {
  width: 18px;
  height: 18px;
  border-radius: 4px;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 0.625rem;
  font-weight: 900;
  flex-shrink: 0;
}

.t-fighter-corner-indicator.red {
  background-color: rgba(220, 38, 38, 0.25);
  color: #ef4444;
  border: 1px solid rgba(220, 38, 38, 0.5);
}

.t-fighter-corner-indicator.blue {
  background-color: rgba(37, 99, 235, 0.25);
  color: #38bdf8;
  border: 1px solid rgba(37, 99, 235, 0.5);
}

.t-fighter-info {
  flex: 1;
  display: flex;
  flex-direction: column;
  min-width: 0;
}

.t-fighter-name {
  font-size: 0.8125rem;
  font-weight: 600;
  color: #cbd5e1;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.t-fighter-row--winner .t-fighter-name {
  color: #ffffff;
  font-weight: 700;
}

.t-fighter-club {
  font-size: 0.625rem;
  color: #64748b;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.t-fighter-score {
  display: flex;
  align-items: center;
  gap: 6px;
  flex-shrink: 0;
}

.t-score-num {
  font-size: 0.875rem;
  font-weight: 700;
  color: #94a3b8;
  min-width: 18px;
  text-align: right;
}

.t-fighter-row--winner .t-score-num {
  color: #10b981;
  font-weight: 800;
}

.t-win-badge {
  font-size: 0.5625rem;
  font-weight: 800;
  background-color: rgba(16, 185, 129, 0.2);
  color: #10b981;
  border: 1px solid rgba(16, 185, 129, 0.4);
  padding: 1px 4px;
  border-radius: 3px;
}

/* Champion Podium Card */
.t-champion-card {
  background: linear-gradient(180deg, #132238 0%, #0d1726 100%);
  border: 1px solid rgba(245, 158, 11, 0.35);
  box-shadow: 0 0 25px rgba(245, 158, 11, 0.15);
  border-radius: 12px;
  padding: 20px 16px;
  text-align: center;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 8px;
  position: relative;
  z-index: 2;
  width: 100%;
}

.t-champion-card.red {
  border-color: rgba(239, 68, 68, 0.5);
  box-shadow: 0 0 25px rgba(239, 68, 68, 0.2);
}

.t-champion-card.blue {
  border-color: rgba(56, 189, 248, 0.5);
  box-shadow: 0 0 25px rgba(56, 189, 248, 0.2);
}

.t-champion-trophy {
  width: 48px;
  height: 48px;
  border-radius: 50%;
  background-color: rgba(245, 158, 11, 0.15);
  border: 1px solid rgba(245, 158, 11, 0.3);
  display: flex;
  align-items: center;
  justify-content: center;
  margin-bottom: 4px;
}

.t-champion-label {
  font-size: 0.6875rem;
  font-weight: 800;
  letter-spacing: 0.1em;
  color: #f59e0b;
}

.t-champion-name {
  font-size: 1rem;
  font-weight: 800;
  color: #ffffff;
}

.t-champion-club {
  font-size: 0.75rem;
  color: #94a3b8;
}

.t-champion-meta {
  font-size: 0.625rem;
  color: #64748b;
  margin-top: 6px;
  border-top: 1px solid rgba(255, 255, 255, 0.08);
  padding-top: 6px;
  width: 100%;
}
</style>
