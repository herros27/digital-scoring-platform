<script setup lang="ts">
import { computed, reactive, ref } from 'vue'
import {
  METHOD_LABEL,
  PHASE_LABEL,
  IPSI_YOUTH_CODES,
  IPSI_YOUTH_SUGGESTION,
  POINT_GAP_MODE_LABEL,
  TIEBREAK_METHODS,
  normalizedPriority,
  payloadFromDraft,
  validateDraft,
  type RulesetDraft,
} from '../lib/rulesetDraft'
import type { RulesetPayload } from '../types'

const props = defineProps<{ draft: RulesetDraft; saving?: boolean; serverError?: string }>()
const emit = defineEmits<{ save: [payload: RulesetPayload, id?: string]; cancel: [] }>()

// salinan kerja: perubahan baru berlaku setelah "Simpan"
const d = reactive<RulesetDraft>(JSON.parse(JSON.stringify(props.draft)))
const submitted = ref(false)
const errors = computed(() => validateDraft(d))
const hasErrors = computed(() => Object.keys(errors.value).length > 0)
const showError = (key: string) => (submitted.value ? errors.value[key] : undefined)

const isEdit = computed(() => !!d.id)

// Mengaktifkan pengecualian: mulai dari waktu utama agar panitia tinggal mengubah yang berbeda.
function toggleAge(i: number) {
  const a = d.ageTimings[i]
  if (a.enabled) {
    a.roundDuration = d.roundDuration
    a.breakDuration = d.breakDuration
    a.totalRounds = d.totalRounds
  }
}

function applyYouthSuggestion() {
  for (const a of d.ageTimings) {
    if (IPSI_YOUTH_CODES.includes(a.code))
      Object.assign(a, { enabled: true }, IPSI_YOUTH_SUGGESTION)
  }
}
const priority = computed(() => normalizedPriority(d))
const techOf = (code: string) => d.techniques.find((t) => t.type === code)

function addTechnique() {
  d.techniques.push({ type: '', label: '', points: 1 })
}
function removeTechnique(i: number) {
  d.techniques.splice(i, 1)
}
function addPenalty() {
  d.penalties.push({ type: '', label: '', points: 0, disqualify: false })
}
function removePenalty(i: number) {
  d.penalties.splice(i, 1)
}
function movePriority(i: number, delta: -1 | 1) {
  const list = [...priority.value]
  const j = i + delta
  if (j < 0 || j >= list.length) return
  ;[list[i], list[j]] = [list[j], list[i]]
  d.tiebreak.priority = list
}
function toggleManual(method: string) {
  const at = d.tiebreak.manual.indexOf(method)
  if (at >= 0) d.tiebreak.manual.splice(at, 1)
  else d.tiebreak.manual.push(method)
}

function submit() {
  submitted.value = true
  if (hasErrors.value) return
  d.tiebreak.priority = priority.value
  emit('save', payloadFromDraft(d), d.id)
}
</script>

<template>
  <div class="rs-backdrop" role="dialog" aria-modal="true">
    <form class="rs-modal" novalidate @submit.prevent="submit">
      <header class="rs-modal__head">
        <div>
          <h2 class="rs-modal__title">{{ isEdit ? 'Ubah Ruleset' : 'Ruleset Baru' }}</h2>
          <p class="rs-hint">
            Perubahan berlaku untuk partai yang dimuat ke gelanggang setelah disimpan. Partai yang
            sudah berjalan tidak berubah.
          </p>
        </div>
        <button type="button" class="rs-icon-btn" aria-label="Tutup" @click="emit('cancel')">
          ×
        </button>
      </header>

      <div class="rs-modal__body">
        <!-- Umum -->
        <section class="rs-section">
          <h3>Umum</h3>
          <div class="rs-grid">
            <label class="rs-field rs-field--wide">
              <span>Nama</span>
              <input
                v-model="d.name"
                class="rs-input"
                data-test="name"
                placeholder="mis. Kejurda Remaja 2026"
              />
              <small v-if="showError('name')" class="rs-error">{{ errors.name }}</small>
            </label>
            <label class="rs-field">
              <span>Versi</span>
              <input v-model="d.version" class="rs-input" />
              <small v-if="showError('version')" class="rs-error">{{ errors.version }}</small>
            </label>
            <label class="rs-field">
              <span>Status</span>
              <select v-model="d.status" class="rs-input">
                <option value="active">Aktif</option>
                <option value="draft">Draf</option>
                <option value="archived">Arsip</option>
              </select>
            </label>
          </div>
        </section>

        <!-- Waktu -->
        <section class="rs-section">
          <h3>Waktu &amp; Babak</h3>
          <div class="rs-grid">
            <label class="rs-field">
              <span>Durasi babak (detik)</span>
              <input v-model.number="d.roundDuration" type="number" min="1" class="rs-input" />
              <small v-if="showError('roundDuration')" class="rs-error">{{
                errors.roundDuration
              }}</small>
            </label>
            <label class="rs-field">
              <span>Istirahat (detik)</span>
              <input v-model.number="d.breakDuration" type="number" min="0" class="rs-input" />
              <small v-if="showError('breakDuration')" class="rs-error">{{
                errors.breakDuration
              }}</small>
            </label>
            <label class="rs-field">
              <span>Jumlah babak</span>
              <input
                v-model.number="d.totalRounds"
                type="number"
                min="1"
                max="3"
                class="rs-input"
              />
              <small v-if="showError('totalRounds')" class="rs-error">{{
                errors.totalRounds
              }}</small>
            </label>
          </div>
        </section>

        <!-- Waktu per golongan umur -->
        <section class="rs-section" data-test="age-timings">
          <h3>Waktu per Golongan Umur</h3>
          <p class="rs-hint">
            Opsional. Golongan yang dicentang memakai waktunya sendiri (bebas ditentukan panitia,
            mis. 2 babak × 2 menit untuk Usia Dini); golongan lain ikut waktu di atas. Berlaku
            otomatis menurut golongan umur kategori partai.
          </p>
          <button
            type="button"
            class="rs-btn"
            data-test="age-suggest"
            @click="applyYouthSuggestion"
          >
            Isi usulan IPSI untuk Usia Dini (2 babak × 1,5 menit)
          </button>
          <div
            v-for="(a, i) in d.ageTimings"
            :key="a.code"
            class="rs-age-row"
            :data-test="`age-${a.code}`"
          >
            <label class="rs-check">
              <input v-model="a.enabled" type="checkbox" @change="toggleAge(i)" />
              <span>{{ a.label }}</span>
            </label>
            <template v-if="a.enabled">
              <label class="rs-field">
                <span>Durasi babak (detik)</span>
                <input v-model.number="a.roundDuration" type="number" min="1" class="rs-input" />
                <small v-if="showError(`ageTimings.${i}.roundDuration`)" class="rs-error">{{
                  errors[`ageTimings.${i}.roundDuration`]
                }}</small>
              </label>
              <label class="rs-field">
                <span>Istirahat (detik)</span>
                <input v-model.number="a.breakDuration" type="number" min="0" class="rs-input" />
                <small v-if="showError(`ageTimings.${i}.breakDuration`)" class="rs-error">{{
                  errors[`ageTimings.${i}.breakDuration`]
                }}</small>
              </label>
              <label class="rs-field">
                <span>Jumlah babak</span>
                <input
                  v-model.number="a.totalRounds"
                  type="number"
                  min="1"
                  max="3"
                  class="rs-input"
                />
                <small v-if="showError(`ageTimings.${i}.totalRounds`)" class="rs-error">{{
                  errors[`ageTimings.${i}.totalRounds`]
                }}</small>
              </label>
            </template>
          </div>
        </section>

        <!-- Teknik -->
        <section class="rs-section">
          <h3>Teknik &amp; Nilai</h3>
          <p class="rs-hint">
            Menjadi tombol di tablet juri. Kode dipakai sistem dan tidak boleh diubah di tengah
            event.
          </p>
          <small v-if="showError('techniques')" class="rs-error">{{ errors.techniques }}</small>
          <div v-for="(t, i) in d.techniques" :key="i" class="rs-row" data-test="technique-row">
            <label class="rs-field">
              <span v-if="i === 0">Kode</span>
              <input v-model="t.type" class="rs-input font-mono" placeholder="catch_fall" />
              <small v-if="showError(`techniques.${i}.type`)" class="rs-error">{{
                errors[`techniques.${i}.type`]
              }}</small>
            </label>
            <label class="rs-field rs-field--grow">
              <span v-if="i === 0">Label tombol</span>
              <input v-model="t.label" class="rs-input" />
              <small v-if="showError(`techniques.${i}.label`)" class="rs-error">{{
                errors[`techniques.${i}.label`]
              }}</small>
            </label>
            <label class="rs-field rs-field--narrow">
              <span v-if="i === 0">Nilai</span>
              <input v-model.number="t.points" type="number" min="1" class="rs-input" />
              <small v-if="showError(`techniques.${i}.points`)" class="rs-error">{{
                errors[`techniques.${i}.points`]
              }}</small>
            </label>
            <button
              type="button"
              class="rs-icon-btn"
              aria-label="Hapus teknik"
              @click="removeTechnique(i)"
            >
              ×
            </button>
          </div>
          <button type="button" class="rs-btn" @click="addTechnique">+ Tambah teknik</button>
        </section>

        <!-- Sanksi -->
        <section class="rs-section">
          <h3>Sanksi</h3>
          <p class="rs-hint">
            Potongan = nilai yang dikurangkan dari sudut yang dikenai. Centang "Diskualifikasi" agar
            sanksi itu langsung mengakhiri partai.
          </p>
          <div v-for="(p, i) in d.penalties" :key="i" class="rs-row" data-test="penalty-row">
            <label class="rs-field">
              <span v-if="i === 0">Kode</span>
              <input v-model="p.type" class="rs-input font-mono" />
              <small v-if="showError(`penalties.${i}.type`)" class="rs-error">{{
                errors[`penalties.${i}.type`]
              }}</small>
            </label>
            <label class="rs-field rs-field--grow">
              <span v-if="i === 0">Label</span>
              <input v-model="p.label" class="rs-input" />
              <small v-if="showError(`penalties.${i}.label`)" class="rs-error">{{
                errors[`penalties.${i}.label`]
              }}</small>
            </label>
            <label class="rs-field rs-field--narrow">
              <span v-if="i === 0">Potongan</span>
              <input
                v-model.number="p.points"
                type="number"
                min="0"
                class="rs-input"
                :disabled="p.disqualify"
              />
              <small v-if="showError(`penalties.${i}.points`)" class="rs-error">{{
                errors[`penalties.${i}.points`]
              }}</small>
            </label>
            <label class="rs-check">
              <input v-model="p.disqualify" type="checkbox" />
              <span>Diskualifikasi</span>
            </label>
            <button
              type="button"
              class="rs-icon-btn"
              aria-label="Hapus sanksi"
              @click="removePenalty(i)"
            >
              ×
            </button>
          </div>
          <button type="button" class="rs-btn" @click="addPenalty">+ Tambah sanksi</button>
        </section>

        <!-- Konsensus -->
        <section class="rs-section">
          <h3>Konsensus Juri</h3>
          <div class="rs-grid">
            <label class="rs-field">
              <span>Jumlah juri (1-5)</span>
              <input
                v-model.number="d.consensus.judges"
                type="number"
                min="1"
                max="5"
                class="rs-input"
                data-test="judges"
              />
              <small v-if="showError('consensus.judges')" class="rs-error">{{
                errors['consensus.judges']
              }}</small>
            </label>
            <label class="rs-field">
              <span>Juri yang harus sepakat</span>
              <input
                v-model.number="d.consensus.majority"
                type="number"
                min="1"
                class="rs-input"
                data-test="majority"
              />
              <small v-if="showError('consensus.majority')" class="rs-error">{{
                errors['consensus.majority']
              }}</small>
            </label>
            <label class="rs-field">
              <span>Window waktu (ms)</span>
              <input
                v-model.number="d.consensus.windowMs"
                type="number"
                min="500"
                max="10000"
                step="100"
                class="rs-input"
              />
              <small v-if="showError('consensus.windowMs')" class="rs-error">{{
                errors['consensus.windowMs']
              }}</small>
            </label>
          </div>
          <p class="rs-hint">
            Contoh: {{ d.consensus.majority }} dari {{ d.consensus.judges }} juri harus menekan
            teknik dan sudut yang sama dalam {{ (d.consensus.windowMs / 1000).toFixed(1) }} detik.
          </p>
        </section>

        <!-- WMP -->
        <section class="rs-section">
          <h3>Wasit Menghentikan Pertandingan (WMP)</h3>
          <p class="rs-hint">
            Bila selisih total poin kedua sudut (setelah sanksi) mencapai batas pada babak yang
            ditentukan, pertandingan langsung dihentikan dan sudut yang unggul menang.
          </p>
          <div class="rs-grid">
            <label class="rs-field">
              <span>Mode</span>
              <select v-model="d.pointGap.mode" class="rs-input" data-test="gap-mode">
                <option v-for="(label, mode) in POINT_GAP_MODE_LABEL" :key="mode" :value="mode">
                  {{ label }}
                </option>
              </select>
            </label>
            <template v-if="d.pointGap.mode === 'custom'">
              <label class="rs-field">
                <span>Selisih poin</span>
                <input
                  v-model.number="d.pointGap.margin"
                  type="number"
                  min="1"
                  class="rs-input"
                  data-test="gap-margin"
                />
                <small v-if="showError('pointGap.margin')" class="rs-error">{{
                  errors['pointGap.margin']
                }}</small>
              </label>
              <label class="rs-field">
                <span>Berlaku mulai babak</span>
                <input
                  v-model.number="d.pointGap.fromRound"
                  type="number"
                  min="1"
                  :max="d.totalRounds"
                  class="rs-input"
                  data-test="gap-from-round"
                />
                <small v-if="showError('pointGap.fromRound')" class="rs-error">{{
                  errors['pointGap.fromRound']
                }}</small>
              </label>
            </template>
          </div>
          <p class="rs-hint">
            <template v-if="d.pointGap.mode === 'auto'">
              Remaja, Dewasa, dan golongan lain: 30 poin. Usia Dini dan Pra-Remaja: 20 poin. Berlaku
              mulai babak 2.
            </template>
            <template v-else-if="d.pointGap.mode === 'custom'">
              Pertandingan dihentikan bila selisih mencapai {{ d.pointGap.margin }} poin sejak babak
              {{ d.pointGap.fromRound }}.
            </template>
            <template v-else
              >Pertandingan hanya berakhir lewat waktu atau keputusan operator meja.</template
            >
          </p>
        </section>

        <!-- Seri -->
        <section class="rs-section">
          <h3>Aturan Seri</h3>
          <p class="rs-hint">
            Dijalankan berurutan bila skor bersih seri; bila masih seri, operator meja memutuskan manual.
          </p>
          <label class="rs-check">
            <input v-model="d.tiebreak.fewestPenalties" type="checkbox" />
            <span>1. Potongan sanksi paling sedikit menang</span>
          </label>
          <label class="rs-check">
            <input v-model="d.tiebreak.techniquePriority" type="checkbox" />
            <span>2. Teknik terbanyak menurut urutan prioritas</span>
          </label>
          <div v-if="d.tiebreak.techniquePriority" class="rs-priority">
            <span class="rs-hint">Urutan prioritas teknik (paling atas = paling bernilai):</span>
            <ol>
              <li v-for="(code, i) in priority" :key="code" data-test="priority-row">
                <span
                  >{{ techOf(code)?.label || code }}
                  <small class="font-mono">({{ code }}, {{ techOf(code)?.points }})</small></span
                >
                <span class="rs-priority__btns">
                  <button
                    type="button"
                    class="rs-icon-btn"
                    :disabled="i === 0"
                    aria-label="Naikkan"
                    @click="movePriority(i, -1)"
                  >
                    ↑
                  </button>
                  <button
                    type="button"
                    class="rs-icon-btn"
                    :disabled="i === priority.length - 1"
                    aria-label="Turunkan"
                    @click="movePriority(i, 1)"
                  >
                    ↓
                  </button>
                </span>
              </li>
            </ol>
          </div>
          <div class="rs-methods">
            <span class="rs-hint">Dasar keputusan manual yang boleh dipilih operator meja:</span>
            <label v-for="m in TIEBREAK_METHODS" :key="m" class="rs-check">
              <input
                type="checkbox"
                :checked="d.tiebreak.manual.includes(m)"
                @change="toggleManual(m)"
              />
              <span>{{ METHOD_LABEL[m] }}</span>
            </label>
          </div>
        </section>

        <!-- Hasil khusus -->
        <section class="rs-section">
          <h3>Hasil Khusus</h3>
          <p class="rs-hint">
            Tombol di konsol meja di luar skor. Nonaktifkan jenis yang tidak dipakai pada regulasi
            ini.
          </p>
          <div v-for="(o, i) in d.outcomes" :key="o.type" class="rs-row" data-test="outcome-row">
            <label class="rs-check rs-check--fixed">
              <input v-model="o.enabled" type="checkbox" />
              <span class="font-mono">{{ o.type }}</span>
            </label>
            <label class="rs-field rs-field--grow">
              <input v-model="o.label" class="rs-input" :disabled="!o.enabled" />
              <small v-if="showError(`outcomes.${i}.label`)" class="rs-error">{{
                errors[`outcomes.${i}.label`]
              }}</small>
            </label>
            <label class="rs-field">
              <select v-model="o.phase" class="rs-input" :disabled="!o.enabled">
                <option v-for="(label, phase) in PHASE_LABEL" :key="phase" :value="phase">
                  {{ label }}
                </option>
              </select>
            </label>
          </div>
        </section>
      </div>

      <footer class="rs-modal__foot">
        <p v-if="serverError" class="rs-error rs-error--block" role="alert">{{ serverError }}</p>
        <p v-else-if="submitted && hasErrors" class="rs-error rs-error--block" role="alert">
          Periksa isian yang bertanda merah.
        </p>
        <div class="rs-actions">
          <button type="button" class="rs-btn" @click="emit('cancel')">Batal</button>
          <button type="submit" class="rs-btn rs-btn--primary" :disabled="saving" data-test="save">
            {{ saving ? 'Menyimpan…' : isEdit ? 'Simpan Perubahan' : 'Buat Ruleset' }}
          </button>
        </div>
      </footer>
    </form>
  </div>
</template>

<style scoped>
.rs-backdrop {
  position: fixed;
  inset: 0;
  z-index: 900;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 16px;
  background: rgba(2, 6, 23, 0.75);
}
.rs-modal {
  display: flex;
  flex-direction: column;
  width: min(880px, 100%);
  max-height: 100%;
  background: var(--bg-card, #131d33);
  border: 1px solid var(--border-card, #334155);
  border-radius: var(--radius-lg, 12px);
  color: var(--text-main, #f8fafc);
}
.rs-modal__head,
.rs-modal__foot {
  padding: 16px 20px;
}
.rs-modal__head {
  display: flex;
  justify-content: space-between;
  gap: 12px;
  border-bottom: 1px solid var(--border-card, #1e293b);
}
.rs-modal__title {
  margin: 0;
  font-size: 1.125rem;
}
.rs-modal__body {
  padding: 8px 20px 16px;
  overflow-y: auto;
}
.rs-modal__foot {
  border-top: 1px solid var(--border-card, #1e293b);
}
.rs-section {
  padding: 14px 0;
  border-bottom: 1px dashed rgba(148, 163, 184, 0.18);
}
.rs-section:last-child {
  border-bottom: 0;
}
.rs-section h3 {
  margin: 0 0 8px;
  font-size: 0.9375rem;
  letter-spacing: 0.04em;
}
.rs-hint {
  margin: 2px 0 8px;
  font-size: 0.75rem;
  color: var(--color-neutral-400, #94a3b8);
}
.rs-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(160px, 1fr));
  gap: 12px;
}
.rs-field {
  display: flex;
  flex-direction: column;
  gap: 4px;
  font-size: 0.75rem;
  color: var(--color-neutral-400, #94a3b8);
  min-width: 0;
}
.rs-field--wide {
  grid-column: span 2;
}
.rs-field--grow {
  flex: 1;
}
.rs-field--narrow {
  width: 96px;
}
.rs-input {
  width: 100%;
  box-sizing: border-box;
  padding: 8px 10px;
  border-radius: var(--radius-xs, 6px);
  border: 1px solid var(--border-card, #334155);
  background: rgba(15, 23, 42, 0.9);
  color: var(--text-main, #f8fafc);
  font: inherit;
  font-size: 0.8125rem;
}
.rs-input:disabled {
  opacity: 0.5;
}
.rs-row {
  display: flex;
  align-items: flex-start;
  gap: 8px;
  margin-bottom: 8px;
  flex-wrap: wrap;
}
.rs-row .rs-field:first-child:not(.rs-field--grow) {
  width: 160px;
}
.rs-check {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 0.8125rem;
  margin: 4px 0;
  padding-top: 6px;
}
.rs-check--fixed {
  width: 120px;
}
.rs-priority {
  margin: 8px 0 8px 22px;
}
.rs-priority ol {
  margin: 4px 0 0;
  padding-left: 20px;
  font-size: 0.8125rem;
}
.rs-priority li {
  display: flex;
  justify-content: space-between;
  align-items: center;
  max-width: 420px;
  padding: 2px 0;
}
.rs-priority__btns {
  display: inline-flex;
  gap: 4px;
}
.rs-methods {
  margin: 10px 0 0 22px;
}
.rs-btn {
  padding: 8px 14px;
  border-radius: var(--radius-xs, 6px);
  border: 1px solid var(--border-card, #334155);
  background: rgba(30, 41, 59, 0.8);
  color: var(--text-main, #f8fafc);
  font-size: 0.8125rem;
  font-weight: 600;
  cursor: pointer;
}
.rs-btn--primary {
  background: var(--color-emerald-600, #059669);
  border-color: var(--color-emerald-500, #10b981);
}
.rs-btn:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}
.rs-icon-btn {
  width: 28px;
  height: 28px;
  margin-top: 2px;
  border-radius: 6px;
  border: 1px solid var(--border-card, #334155);
  background: transparent;
  color: var(--color-neutral-300, #cbd5e1);
  cursor: pointer;
}
.rs-icon-btn:disabled {
  opacity: 0.35;
  cursor: not-allowed;
}
.rs-error {
  color: #fca5a5;
  font-size: 0.7188rem;
}
.rs-error--block {
  margin: 0 0 10px;
  font-size: 0.8125rem;
}
.rs-actions {
  display: flex;
  justify-content: flex-end;
  gap: 10px;
}
.rs-age-row {
  display: grid;
  grid-template-columns: minmax(130px, 1fr) repeat(3, minmax(90px, 1fr));
  gap: 10px 12px;
  align-items: end;
  padding: 8px 0;
  border-top: 1px dashed rgba(255, 255, 255, 0.08);
}
.rs-age-row > .rs-check {
  grid-column: 1 / -1;
}
@media (min-width: 720px) {
  .rs-age-row > .rs-check {
    grid-column: auto;
  }
}
</style>
