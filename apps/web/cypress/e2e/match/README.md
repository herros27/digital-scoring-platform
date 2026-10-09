# E2E: Alur Pertandingan (Wasit & Juri)

Menguji pertandingan yang **dimulai** dengan backend, Vite (`/api`, `/ws`), WebSocket, dan browser sungguhan.
Tidak ada mock: data dibuat lewat API publik (kejuaraan, gelanggang, ruleset cepat, atlet, partai, PIN wasit
dan 3 juri), lalu perangkat dimainkan lewat WebSocket nyata atau layar aplikasi.

## Arsitektur tes

| Lapisan             | Berkas                                                       | Fungsi                                                                         |
| ------------------- | ------------------------------------------------------------ | ------------------------------------------------------------------------------ |
| Driver (tanpa `cy`) | `support/match/http.ts`, `socket.ts`, `kit.ts`, `helpers.ts` | fetch + WebSocket + penyiapan data; sama persis di Cypress dan Node            |
| Spec alur API/WS    | `01`–`06-*.cy.ts`                                            | memakai driver lewat `runAsync` (satu pembungkus per tes)                      |
| Spec UI             | `ui-05`, `ui-06`                                             | layar juri & konsol wasit dengan perintah Cypress asli                         |
| Perintah            | `visitWithSession` (`support/commands.ts`)                   | membuka halaman sebagai perangkat PIN tanpa login ulang (batas login 10/menit) |

Spec alur (`01`–`06`) tidak memakai `cy.*`, sehingga **dapat dijalankan juga tanpa browser** lewat runner Node
(`npm run test:match-flow`). Spec `ui-*` hanya berjalan di Cypress.

## Menjalankan

```bash
# prasyarat: Postgres + backend + seed, dan Vite dev server
go run ./cmd/seed/main.go && go run ./cmd/server     # terminal 1 (root repo)
npm run dev                                           # terminal 2 (apps/web)

npx cypress run --spec "cypress/e2e/match/**"         # terminal 3: semua spec (butuh binary Cypress)
npm run test:match-flow                               # alternatif headless tanpa browser (spec 01–06)
npm run test:match-flow -- 03                         # satu spec
```

Ruleset uji: ronde 8–20 dtk, istirahat 3–4 dtk, 2 babak, konsensus 2-dari-3 dalam 3 dtk. Seluruh suite ±3 menit.
Setiap `describe` membuat kejuaraan sendiri dan membersihkannya (`teardown`).

## Cakupan

| Spec                            | Isi                                                                                                                                                                                                                               |
| ------------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `01-wasit-kontrol-pertandingan` | load-arena, start, status partai di DB, timer (jalan/beku/lanjut/koreksi/`timer_sync`), pergantian ronde otomatis, selesai otomatis, seri → keputusan wasit, finish/cancel/outcome, sanksi, sinyal, audit                         |
| `02-juri-sinkronisasi-realtime` | tiap kursi 1–3: snapshot identik, urutan `state_change` identik, hitung mundur konsisten, konsensus (2 dari 3), jendela 3 dtk, suara ganda, input di luar waktu, hak akses WS juri, audit                                         |
| `03-otorisasi-role`             | matriks REST (wasit/dewan/admin vs juri/operator/kontingen/anon/gelanggang lain), skor REST hanya juri atas identitas token, handshake WS, perintah WS per role                                                                   |
| `04-ketahanan-dan-kasus-tepi`   | gagal memulai, ARENA_BUSY, reschedule, balapan (5× start, 3× finish), aksi berulang, putus/sambung ulang (juri, wasit, semua perangkat, tablet kembar), pesan rusak, konsistensi dengan endpoint publik                           |
| `05-format-5-juri`              | konsensus 3-dari-5: kelima juri online, progres 1→2→3, kombinasi juri mana pun, sudut berbeda tidak digabung, di luar window tidak digabung                                                                                       |
| `06-pemulihan-restart-server`   | restart backend di tengah partai: skor/babak bertahan, jam dijeda, skor ditolak sampai wasit melanjutkan, partai selesai normal (perlu `E2E_RESTART_CMD`)                                                                         |
| `07-kompetisi-penuh`            | satu kompetisi dari pendaftaran sampai juara: 4 atlet/3 kontingen → verifikasi → bagan → bagan publik (tersembunyi sebelum aktif) → 2 semifinal → final otomatis → final dua babak habis waktu → juara di bagan → klasemen medali |
| `08-kompetisi-8-peserta`        | bagan 8 peserta (perempat → semi → final) dimainkan penuh; partai berikutnya terbentuk otomatis; medali hanya final dan semifinal                                                                                                 |
| `ui-05-layar-juri`              | per kursi: layar online, atlet, status/timer mengikuti wasit tanpa reload, konsensus tampil, terkunci saat istirahat/dijeda, BATAL + pesan penolakan, SELESAI                                                                     |
| `ui-06-konsol-wasit`            | tombol MULAI/JEDA/LANJUTKAN/SELESAIKAN memanggil server dan terbukti dari soket juri; skor konsensus tampil; akses antar role/gelanggang                                                                                          |

## Cacat yang ditemukan tes ini dan sudah diperbaiki

1. **`finish` pada partai yang belum dimulai** menetapkan pemenang "red", memajukan bagan, dan menyiarkan hasil
   sementara state machine tetap `idle`. Kini ditolak (`INVALID_STATE_TRANSITION`).
2. **`POST …/scores` (REST) tanpa pembatasan role**: siapa pun yang login bisa mengirim suara atas nama juri lain
   (`judge_id` dari body). Kini hanya token PIN juri, identitas dari token, dan harus gelanggang yang sama.
3. **REST kontrol partai tidak membatasi gelanggang**: wasit PIN gelanggang B dapat menjeda/menyelesaikan/membatalkan/
   WO partai di gelanggang A (WebSocket sudah menolak). Kini token PIN hanya mengontrol gelanggangnya.
4. **Status `running` tidak pernah disimpan**: partai berjalan tetap `idle` di DB sehingga badge LIVE, penghitung
   "sedang tanding" dan `GET matches?status=running` selalu kosong.
5. **`state_change break_N` membawa timer ronde yang sudah habis** (0, tidak berjalan): layar juri menampilkan 00:00
   sampai `timer_sync` berikutnya. Timer istirahat kini dibuat sebelum disiarkan.
6. **Partai selesai tidak menyiarkan `state_change finished`**: tablet juri tetap "BERJALAN" dengan tombol aktif.
7. **Balasan `sync_response`/`error` dikirim ke koneksi pertama berID sama**: tablet juri yang tersambung ulang
   (atau tablet kembar) bisa tidak menerima snapshot. Kini membalas ke koneksi yang meminta.

8. **Skor juri diterima saat partai dijeda**: input (WebSocket dan REST) kini ditolak `SCORING_NOT_ALLOWED`
   selama jam berhenti, termasuk partai yang baru dipulihkan setelah restart; tombol di layar juri terkunci
   dengan keterangan "dijeda".
9. **Tombol BATAL tidak berfungsi**: `score_type: "undo"` diproses seperti suara biasa lalu tak berefek, dan
   layar juri tidak menampilkan penolakan. Kini BATAL menarik input juri itu yang masih menunggu juri lain
   (lampu konsensus turun/padam di semua tablet), menarik sudut yang **terakhir ditekan** (bukan selalu merah),
   dan bila tidak ada yang bisa ditarik server membalas `UNDO_NOT_POSSIBLE` yang tampil di layar. Poin yang
   sudah terkonfirmasi tidak dapat dibatalkan dari tablet (itu wewenang wasit).
10. **Klasemen medali salah** (ditemukan spec 07): setiap pemenang partai dihitung emas dan setiap yang kalah perak
    (4 peserta menghasilkan 3 emas + 3 perak), tanpa perunggu, dan semua masuk tim "Umum" karena partai menyimpan ID
    peserta sedangkan penghitungan mencari ID atlet. Kini medali diturunkan dari struktur bagan: juara emas, finalis perak,
    dua yang kalah di semifinal perunggu, perempat final tanpa medali, dan tim diambil dari atlet milik peserta.
11. **Tidak ada ping/pong WebSocket**: koneksi mati (jaringan putus tanpa menutup soket) tetap tercatat online.
    Server kini mengirim ping tiap 54 dtk dan memutus klien yang tidak membalas dalam 60 dtk
    (`WS_PONG_WAIT_MS` mengubahnya; diuji di `internal/hub/hub_test.go`).

## Playwright Test: 3 layar juri + 1 konsol wasit

`playwright/tiga-juri-satu-wasit.spec.ts` membuka **4 browser context** (3 juri + wasit) dengan UI sungguhan, hal
yang tidak mungkin dalam satu tes Cypress. Tiap tes berdiri sendiri (partai baru), sehingga bisa dipilih satu per satu.
Mencakup: semua online, MULAI di konsol wasit, tiga juri menekan tombol sama bersamaan (tepat 1 poin), satu suara
menunggu lalu padam, BATAL (dengan dan tanpa tekanan menunggu), JEDA/LANJUT mengunci semua layar, dua juri
bersamaan, dan SELESAIKAN. Tes ini menemukan tombol skor juri masih aktif setelah SELESAI (kini terkunci).

```bash
npm run test:pw          # headless
npm run test:pw:ui       # UI Mode: daftar tes, pilih/jalankan, time-travel, watch
npm run test:pw:headed   # 4 jendela browser tersusun grid 2x2 (skala PW_ZOOM, bawaan 0.5)
npx playwright test -g "BATAL"     # pilih tes lewat nama
npx playwright show-report         # laporan HTML (trace tersimpan bila tes gagal)
```

Variabel opsional: `E2E_BASE_URL`, `PW_CHROMIUM` (path Chromium bila bukan bawaan Playwright), `PW_ZOOM`, `PW_SLOWMO`.

## Spesifik lingkungan

- `05-format-5-juri`: ruleset 5 juri, 3 harus sepakat (kit: `judges: 5, majority: 3`).
- `06-pemulihan-restart-server`: mematikan dan menyalakan ulang backend sungguhan, sehingga **hanya berjalan bila
  `E2E_RESTART_CMD` diset** (perintah shell yang me-restart backend dan baru kembali saat server siap, dengan
  env/DB yang sama). Tanpa itu spec dilewati. Cypress menjalankannya lewat `cy.task('server:restart')`.

  ```bash
  E2E_RESTART_CMD="./scripts/restart-backend.sh" npx cypress run --spec "cypress/e2e/match/06-*"
  E2E_RESTART_CMD="./scripts/restart-backend.sh" npm run test:match-flow -- 06
  ```

## Perilaku yang dicatat

- Poin yang sudah terkonfirmasi hanya dikoreksi wasit (tidak ada pembatalan oleh juri).
- Sisa waktu setelah restart dipulihkan dari simpanan terakhir, jadi bisa sedikit lebih besar daripada saat
  server mati; jam tidak berjalan selama server mati (partai dipulihkan dalam keadaan dijeda).
