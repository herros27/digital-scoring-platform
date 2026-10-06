<script setup lang="ts">
import { computed } from 'vue'
import { MATCH_STATUS_LABEL } from '@/composables/matchStatus'
import type { WsTarget } from '@/composables/useWebSocket'
import type { RefereeConsole } from '../composables/useRefereeConsole'
import { formatClock, TARGET_LABEL } from '../lib/matchLog'
import { penaltiesOf, penaltySummary, penaltyTotal } from '../lib/penalties'
import { finishRequest, type default as ConfirmRequest } from './confirmRequest'

const props = defineProps<{ c: RefereeConsole }>()
const emit = defineEmits<{
  confirm: [request: ConfirmRequest]
  goto: [tab: string]
}>()

const statusLabel = computed(() => MATCH_STATUS_LABEL[props.c.status])
const matchTitle = computed(() => {
  const cls = props.c.redCompetitor?.classLabel
  return cls ? `${cls.toUpperCase()} • TANDING` : 'PERTANDINGAN TANDING'
})

function kindPoints(target: WsTarget, prefix: string): number {
  return penaltiesOf(props.c.penalties, target)
    .filter((p) => p.type.startsWith(prefix))
    .reduce((sum, p) => sum + p.points, 0)
}

/** "2 (-3)" bila ada potongan, "1" bila hanya peringatan tanpa potongan, "0" bila kosong. */
function penaltyText(count: number, points: number): string {
  return points > 0 ? `${count} (-${points})` : String(count)
}

function scoreNote(target: WsTarget): string {
  if (props.c.lastConfirmed?.target === target)
    return `KONSENSUS JURI (+${props.c.lastConfirmed.points} TERAKHIR)`
  const cut = penaltyTotal(props.c.penalties, target)
  return cut > 0 ? `TERPOTONG SANKSI -${cut}` : 'BELUM ADA POIN SAH'
}

const fighterOf = (target: WsTarget) => ({
  target,
  label: TARGET_LABEL[target],
  competitor: target === 'red' ? props.c.redCompetitor : props.c.blueCompetitor,
  score: target === 'red' ? props.c.redTotal : props.c.blueTotal,
  summary: penaltySummary(props.c.penalties, target),
  teguranPts: kindPoints(target, 'teguran'),
  peringatanPts: kindPoints(target, 'peringatan'),
  note: scoreNote(target),
  tech: props.c.techniques[target],
})
const red = computed(() => fighterOf('red'))
const blue = computed(() => fighterOf('blue'))

const winnerTarget = computed<WsTarget | null>(() => {
  const corner = props.c.result?.corner
  return corner === 'red' || corner === 'blue' ? corner : null
})

const recent = computed(() => [...props.c.logEntries].reverse().slice(0, 6))

function askFinish() {
  emit('confirm', finishRequest(props.c))
}

function askCancel() {
  emit('confirm', {
    title: 'Batalkan Partai?',
    message:
      'Partai dibatalkan tanpa pemenang dan tidak dapat dilanjutkan. Gunakan Diskualifikasi bila ada sudut yang harus kalah.',
    confirmLabel: 'Batalkan Partai',
    action: () => props.c.cancelMatch(),
  })
}
</script>

<template>
  <div>
    <div v-if="!c.hasMatch" class="ref-empty ref-section">
      <strong>Belum ada partai di gelanggang ini</strong>
      <span
        >Pilih partai dari antrean lalu muat ke gelanggang untuk mulai mengendalikan
        pertandingan.</span
      >
      <button class="ref-btn ref-btn--primary" @click="emit('goto', 'bagan-antrean')">
        Buka Bagan &amp; Antrean
      </button>
    </div>

    <template v-else>
      <section class="ref-banner">
        <div class="ref-banner__left">
          <span class="ref-badge-arena font-mono">{{
            (c.arenaName || 'GELANGGANG').toUpperCase()
          }}</span>
          <div class="ref-banner-titles">
            <span class="ref-tourney-title font-mono">{{ c.competitionName.toUpperCase() }}</span>
            <h2 class="ref-match-title">{{ matchTitle }}</h2>
          </div>
        </div>
        <div class="ref-banner__right font-mono">
          <span v-if="c.rulesetName" class="ref-chip-green">{{ c.rulesetName.toUpperCase() }}</span>
          <span class="ref-chip-round">BABAK: {{ c.round }}/{{ c.totalRounds }}</span>
        </div>
      </section>

      <section v-if="c.tiePending" class="ref-result-banner ref-result-banner--muted font-mono">
        <strong>WAKTU HABIS — SKOR SERI</strong>
        <span>Tentukan pemenang sebagai keputusan wasit.</span>
        <button class="ref-btn ref-btn--primary" @click="askFinish">Tetapkan Pemenang</button>
      </section>

      <section
        v-if="c.isFinished"
        class="ref-result-banner font-mono"
        :class="winnerTarget ? `ref-result-banner--${winnerTarget}` : 'ref-result-banner--muted'"
      >
        <strong v-if="winnerTarget"
          >PERTANDINGAN SELESAI — PEMENANG SUDUT {{ TARGET_LABEL[winnerTarget] }}</strong
        >
        <strong v-else>PERTANDINGAN BERAKHIR</strong>
        <span v-if="c.result?.result_type">{{ c.result.result_type }}</span>
        <button class="ref-btn" @click="emit('goto', 'bagan-antrean')">
          Pilih Partai Berikutnya
        </button>
      </section>

      <section class="ref-summary-grid">
        <div v-if="red" class="ref-fighter-card ref-fighter-card--red">
          <div class="ref-fighter-card__top font-mono">
            <span class="ref-corner-tag ref-corner-tag--red">SUDUT MERAH</span>
            <span class="ref-fighter-team">{{ red.competitor?.team }}</span>
          </div>
          <h3 class="ref-fighter-name">{{ red.competitor?.name ?? 'Sudut Merah' }}</h3>
          <span class="ref-fighter-class font-mono">{{ red.competitor?.classLabel }}</span>
          <div class="ref-score-row">
            <div class="ref-score-display ref-score-display--red">
              <span class="ref-score-sub font-mono">SKOR SAH</span>
              <div class="ref-score-num font-mono">{{ red.score }}</div>
              <span class="ref-score-note font-mono">{{ red.note }}</span>
            </div>
            <div class="ref-penalties-mini font-mono">
              <div class="ref-pen-item">
                <span>TEGURAN</span
                ><span class="ref-pen-num ref-pen-num--red">{{
                  penaltyText(red.summary.teguran, red.teguranPts)
                }}</span>
              </div>
              <div class="ref-pen-item">
                <span>PERINGATAN</span
                ><span class="ref-pen-num ref-pen-num--red">{{
                  penaltyText(red.summary.peringatan, red.peringatanPts)
                }}</span>
              </div>
              <div class="ref-pen-item">
                <span>BINAAN</span
                ><span class="ref-pen-num ref-pen-num--green">{{ red.summary.binaan }}</span>
              </div>
            </div>
          </div>
          <div class="ref-tech-summary font-mono">
            Pukulan: <strong>{{ red.tech.pukulan }}</strong> &bull; Tendangan:
            <strong>{{ red.tech.tendangan }}</strong> &bull; Jatuhan:
            <strong>{{ red.tech.jatuhan }}</strong>
          </div>
        </div>

        <div class="ref-chrono-card">
          <div class="ref-chrono-header font-mono">
            <span class="ref-chrono-label"><span class="ref-pulse-dot" /> CHRONOMETER RESMI</span>
            <span class="ref-chrono-round">BABAK {{ c.round }} &mdash; {{ statusLabel }}</span>
            <span class="ref-chrono-max">{{ formatClock(c.roundDurationMs) }} MAX</span>
          </div>
          <div class="ref-chrono-time font-mono">{{ c.timer.formattedTime }}</div>
          <div class="ref-chrono-controls font-mono">
            <button
              class="ref-time-adj-btn"
              :disabled="!c.canAdjustTime"
              title="Kurangi 10 detik"
              @click="c.adjustTime(-10)"
            >
              -10 DETIK
            </button>
            <button
              class="ref-time-adj-btn"
              :disabled="!c.canAdjustTime"
              title="Tambah 10 detik"
              @click="c.adjustTime(10)"
            >
              +10 DETIK
            </button>
            <button
              class="ref-time-adj-btn"
              :disabled="!c.canResetRound"
              title="Kembalikan waktu babak ke durasi penuh (hanya saat dijeda)"
              @click="c.resetRound()"
            >
              RESET BABAK
            </button>
          </div>
        </div>

        <div v-if="blue" class="ref-fighter-card ref-fighter-card--blue">
          <div class="ref-fighter-card__top font-mono">
            <span class="ref-fighter-team">{{ blue.competitor?.team }}</span>
            <span class="ref-corner-tag ref-corner-tag--blue">SUDUT BIRU</span>
          </div>
          <h3 class="ref-fighter-name">{{ blue.competitor?.name ?? 'Sudut Biru' }}</h3>
          <span class="ref-fighter-class font-mono">{{ blue.competitor?.classLabel }}</span>
          <div class="ref-score-row">
            <div class="ref-penalties-mini font-mono">
              <div class="ref-pen-item">
                <span>TEGURAN</span
                ><span class="ref-pen-num ref-pen-num--red">{{
                  penaltyText(blue.summary.teguran, blue.teguranPts)
                }}</span>
              </div>
              <div class="ref-pen-item">
                <span>PERINGATAN</span
                ><span class="ref-pen-num ref-pen-num--red">{{
                  penaltyText(blue.summary.peringatan, blue.peringatanPts)
                }}</span>
              </div>
              <div class="ref-pen-item">
                <span>BINAAN</span
                ><span class="ref-pen-num ref-pen-num--green">{{ blue.summary.binaan }}</span>
              </div>
            </div>
            <div class="ref-score-display ref-score-display--blue">
              <span class="ref-score-sub font-mono">SKOR SAH</span>
              <div class="ref-score-num font-mono">{{ blue.score }}</div>
              <span class="ref-score-note font-mono">{{ blue.note }}</span>
            </div>
          </div>
          <div class="ref-tech-summary font-mono">
            Pukulan: <strong>{{ blue.tech.pukulan }}</strong> &bull; Tendangan:
            <strong>{{ blue.tech.tendangan }}</strong> &bull; Jatuhan:
            <strong>{{ blue.tech.jatuhan }}</strong>
          </div>
        </div>
      </section>

      <section class="ref-command-section">
        <div class="ref-command-header font-mono">
          <div class="ref-command-title">
            <span>KONTROL UTAMA PERTANDINGAN (COMMAND STRIP)</span>
          </div>
          <div class="ref-operator-mode">
            <button
              class="ref-btn ref-btn--danger"
              :disabled="!c.canFinish || c.busy"
              @click="askCancel"
            >
              BATALKAN PARTAI
            </button>
          </div>
        </div>

        <div class="ref-main-buttons-grid">
          <button
            class="ref-cmd-btn ref-cmd-btn--start"
            :disabled="!c.canStart || c.busy"
            @click="c.start()"
          >
            <div class="ref-cmd-text">
              <span class="ref-cmd-label">MULAI RONDE</span>
              <span class="ref-cmd-shortcut font-mono">SHORTCUT [SPACEBAR]</span>
            </div>
          </button>
          <button
            class="ref-cmd-btn ref-cmd-btn--pause"
            :disabled="!c.canPause || c.busy"
            @click="c.pause()"
          >
            <div class="ref-cmd-text">
              <span class="ref-cmd-label">JEDA / PAUSE</span>
              <span class="ref-cmd-shortcut font-mono">SHORTCUT [P]</span>
            </div>
          </button>
          <button
            class="ref-cmd-btn ref-cmd-btn--resume"
            :disabled="!c.canResume || c.busy"
            @click="c.resume()"
          >
            <div class="ref-cmd-text">
              <span class="ref-cmd-label">LANJUTKAN</span>
              <span class="ref-cmd-shortcut font-mono">SHORTCUT [L]</span>
            </div>
          </button>
          <button
            class="ref-cmd-btn ref-cmd-btn--finish"
            :disabled="!c.canFinish || c.busy"
            @click="askFinish"
          >
            <div class="ref-cmd-text">
              <span class="ref-cmd-label">SELESAIKAN</span>
              <span class="ref-cmd-shortcut font-mono">SHORTCUT [ESC]</span>
            </div>
          </button>
        </div>

        <div class="ref-secondary-strip">
          <button class="ref-sec-btn" :disabled="!c.canSignal" @click="c.ringGong()">
            <span>BUNYIKAN GONG ARENA</span><span class="ref-sec-shortcut font-mono">[G]</span>
          </button>
          <button
            class="ref-sec-btn"
            :class="{ 'ref-sec-btn--active': c.medicActive }"
            :disabled="!c.canSignal"
            @click="c.toggleMedic()"
          >
            <span>{{
              c.medicActive ? 'DOKTER DIPANGGIL — SELESAIKAN' : 'DOKTER PERTANDINGAN'
            }}</span
            ><span class="ref-sec-shortcut font-mono">[MED]</span>
          </button>
          <button
            class="ref-sec-btn"
            :class="{ 'ref-sec-btn--active': c.varActive }"
            :disabled="!c.canSignal"
            @click="c.toggleVar()"
          >
            <span>{{
              c.varActive ? 'TINJAUAN VIDEO BERJALAN — SELESAIKAN' : 'TINJAUAN VIDEO (VAR)'
            }}</span
            ><span class="ref-sec-shortcut font-mono">[VAR]</span>
          </button>
        </div>
      </section>

      <section class="ref-card ref-section">
        <div class="ref-card__head font-mono">
          <span>KEJADIAN TERAKHIR</span>
          <button class="ref-chip-btn" @click="emit('goto', 'log-poin')">Lihat semua log</button>
        </div>
        <div v-if="!recent.length" class="ref-card__sub font-mono">Belum ada kejadian.</div>
        <div v-else class="ref-log-list font-mono">
          <div v-for="e in recent" :key="e.id" class="ref-log-card">
            <span class="ref-log-time">{{ formatClock(e.clockMs) }}</span>
            <div class="ref-log-content">
              <strong class="ref-log-title" :class="`ref-log-title--${e.category}`">{{
                e.title
              }}</strong>
              <span class="ref-log-detail">{{ e.detail }}</span>
            </div>
          </div>
        </div>
      </section>
    </template>
  </div>
</template>
