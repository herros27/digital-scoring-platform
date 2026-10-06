<script setup lang="ts">
import { computed } from 'vue'
import type { WsTarget } from '@/composables/useWebSocket'
import type { RefereeConsole } from '../composables/useRefereeConsole'
import { formatClock, TARGET_LABEL } from '../lib/matchLog'
import { hasPenalty, lastPenalty, penaltyTotal } from '../lib/penalties'
import type ConfirmRequest from './confirmRequest'

const props = defineProps<{ c: RefereeConsole }>()
const emit = defineEmits<{ confirm: [request: ConfirmRequest]; goto: [tab: string] }>()

const sides = computed(() =>
  (['red', 'blue'] as const).map((target) => ({
    target,
    label: TARGET_LABEL[target],
    name:
      (target === 'red' ? props.c.redCompetitor : props.c.blueCompetitor)?.name ??
      `Sudut ${TARGET_LABEL[target]}`,
    total: penaltyTotal(props.c.penalties, target),
    last: lastPenalty(props.c.penalties, target),
  })),
)

const history = computed(() => [...props.c.penalties].reverse())

function askPenalty(
  target: WsTarget,
  name: string,
  kind: { type: string; label: string; action?: string },
) {
  if (kind.action !== 'disqualify') {
    props.c.applyPenalty(target, kind.type)
    return
  }
  emit('confirm', {
    title: `${kind.label} — Diskualifikasi Sudut ${TARGET_LABEL[target]}?`,
    message: `${name} langsung didiskualifikasi dan pertandingan berakhir dengan kemenangan sudut lawan. Tindakan ini tidak dapat dibatalkan.`,
    confirmLabel: 'Jatuhkan Sanksi',
    danger: true,
    action: () => props.c.applyPenalty(target, kind.type),
  })
}

function askDisqualify(target: WsTarget, name: string) {
  emit('confirm', {
    title: `Diskualifikasi Sudut ${TARGET_LABEL[target]}?`,
    message: `${name} didiskualifikasi dan pertandingan langsung berakhir dengan kemenangan sudut lawan. Tindakan ini tidak dapat dibatalkan.`,
    confirmLabel: 'Diskualifikasi',
    danger: true,
    action: () => props.c.disqualify(target),
  })
}
</script>

<template>
  <div>
    <div v-if="!c.hasMatch" class="ref-empty ref-section">
      <strong>Belum ada partai di gelanggang ini</strong>
      <span>Sanksi dapat dijatuhkan setelah partai dimuat dan dimulai.</span>
      <button class="ref-btn ref-btn--primary" @click="emit('goto', 'bagan-antrean')">
        Buka Bagan &amp; Antrean
      </button>
    </div>

    <template v-else>
      <p v-if="!c.canPenalize" class="ref-card__sub font-mono ref-section">
        {{
          c.isFinished
            ? 'Pertandingan sudah selesai; sanksi tidak dapat diubah.'
            : 'Sanksi dapat dijatuhkan setelah pertandingan dimulai.'
        }}
      </p>

      <section class="ref-two-col ref-section">
        <div v-for="s in sides" :key="s.target" class="ref-sanction-col">
          <div class="ref-sanction-header font-mono">
            <span :class="s.target === 'red' ? 'ref-badge-mini-red' : 'ref-badge-mini-blue'">{{
              s.label
            }}</span>
            <span class="ref-sanction-name">{{ s.name }}</span>
            <span class="ref-sanction-total"
              >TOTAL SANKSI: {{ s.total > 0 ? `-${s.total}` : 0 }}</span
            >
          </div>

          <div class="ref-sanction-list font-mono">
            <button
              v-for="k in c.penaltyCatalog"
              :key="k.type"
              class="ref-sanc-btn"
              :class="{ 'ref-sanc-btn--active': hasPenalty(c.penalties, s.target, k.type) }"
              :disabled="!c.canPenalize || hasPenalty(c.penalties, s.target, k.type)"
              @click="askPenalty(s.target, s.name, k)"
            >
              <span>{{ hasPenalty(c.penalties, s.target, k.type) ? '✓' : '•' }} {{ k.label }}</span>
              <strong
                class="ref-sanc-pts"
                :class="{ 'ref-sanc-pts--warn': k.points >= 5 || k.action === 'disqualify' }"
              >
                {{
                  k.action === 'disqualify'
                    ? 'DISKUALIFIKASI'
                    : k.points
                      ? `-${k.points} POIN`
                      : 'TANPA POTONGAN'
                }}
              </strong>
            </button>

            <button
              class="ref-disqualify-btn font-mono"
              :disabled="!c.canPenalize"
              @click="askDisqualify(s.target, s.name)"
            >
              <span>DISKUALIFIKASI {{ s.label }}</span>
            </button>
          </div>

          <div class="ref-sanc-footer font-mono">
            <button
              class="ref-undo-sanc-btn"
              :disabled="!s.last || c.isFinished"
              @click="c.revokePenalty(s.target)"
            >
              {{ s.last ? `BATALKAN: ${s.last.label.toUpperCase()}` : 'BATALKAN SANKSI' }}
            </button>
            <span class="ref-card__sub">Membatalkan sanksi terakhir sudut ini</span>
          </div>
        </div>
      </section>

      <section class="ref-card ref-section">
        <div class="ref-card__head font-mono">
          <span>RIWAYAT SANKSI</span>
          <span class="ref-card__sub">{{ history.length }} sanksi</span>
        </div>
        <div v-if="!history.length" class="ref-card__sub font-mono">
          Belum ada sanksi pada pertandingan ini.
        </div>
        <table v-else class="ref-table font-mono">
          <thead>
            <tr>
              <th>Babak</th>
              <th>Waktu</th>
              <th>Sudut</th>
              <th>Sanksi</th>
              <th>Potongan</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="p in history" :key="p.id">
              <td>B{{ p.round }}</td>
              <td>{{ formatClock(p.clock_ms) }}</td>
              <td :class="p.target === 'red' ? 'ref-red' : 'ref-blue'">
                {{ TARGET_LABEL[p.target] }}
              </td>
              <td>{{ p.label }}</td>
              <td>{{ p.points ? `-${p.points}` : '0' }}</td>
            </tr>
          </tbody>
        </table>
      </section>
    </template>
  </div>
</template>
