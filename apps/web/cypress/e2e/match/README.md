# E2E: Alur Pertandingan (Wasit & Juri)

Menguji pertandingan yang **dimulai** dengan backend, Vite (`/api`, `/ws`), WebSocket, dan browser sungguhan.
Tidak ada mock: data dibuat lewat API publik (kejuaraan, gelanggang, ruleset cepat, atlet, partai, PIN wasit
dan 3 juri), lalu perangkat dimainkan lewat WebSocket nyata atau layar aplikasi.

## Arsitektur tes

| Lapisan             | Berkas                                                       | Fungsi                                                                         |
| ------------------- | ------------------------------------------------------------ | ------------------------------------------------------------------------------ |
| Driver (tanpa `cy`) | `support/match/http.ts`, `socket.ts`, `kit.ts`, `helpers.ts` | fetch + WebSocket + penyiapan data; sama persis di Cypress dan Node            |
| Spec alur API/WS    | `01`–`04-*.cy.ts`                                            | memakai driver lewat `runAsync` (satu pembungkus per tes)                      |
| Spec UI             | `ui-05`, `ui-06`                                             | layar juri & konsol wasit dengan perintah Cypress asli                         |
| Perintah            | `visitWithSession` (`support/commands.ts`)                   | membuka halaman sebagai perangkat PIN tanpa login ulang (batas login 10/menit) |

Spec alur (`01`–`04`) tidak memakai `cy.*`, sehingga **dapat dijalankan juga tanpa browser** lewat runner Node
(`npm run test:match-flow`). Spec `ui-*` hanya berjalan di Cypress.

## Menjalankan

```bash
# prasyarat: Postgres + backend + seed, dan Vite dev server
go run ./cmd/seed/main.go && go run ./cmd/server     # terminal 1 (root repo)
npm run dev                                           # terminal 2 (apps/web)

npx cypress run --spec "cypress/e2e/match/**"         # terminal 3: semua spec (butuh binary Cypress)
npm run test:match-flow                               # alternatif headless tanpa browser (spec 01–04)
npm run test:match-flow -- 03                         # satu spec
```

Ruleset uji: ronde 8–20 dtk, istirahat 3–4 dtk, 2 babak, konsensus 2-dari-3 dalam 3 dtk. Seluruh suite ±3 menit.
Setiap `describe` membuat kejuaraan sendiri dan membersihkannya (`teardown`).

## Cakupan

| Spec                            | Isi                                                                                                                                                                                                       |
| ------------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `01-wasit-kontrol-pertandingan` | load-arena, start, status partai di DB, timer (jalan/beku/lanjut/koreksi/`timer_sync`), pergantian ronde otomatis, selesai otomatis, seri → keputusan wasit, finish/cancel/outcome, sanksi, sinyal, audit |
| `02-juri-sinkronisasi-realtime` | tiap kursi 1–3: snapshot identik, urutan `state_change` identik, hitung mundur konsisten, konsensus (2 dari 3), jendela 3 dtk, suara ganda, input di luar waktu, hak akses WS juri, audit                 |
| `03-otorisasi-role`             | matriks REST (wasit/dewan/admin vs juri/operator/kontingen/anon/gelanggang lain), skor REST hanya juri atas identitas token, handshake WS, perintah WS per role                                           |
| `04-ketahanan-dan-kasus-tepi`   | gagal memulai, ARENA_BUSY, reschedule, balapan (5× start, 3× finish), aksi berulang, putus/sambung ulang (juri, wasit, semua perangkat, tablet kembar), pesan rusak, konsistensi dengan endpoint publik   |
| `ui-05-layar-juri`              | per kursi: layar online, atlet, status/timer mengikuti wasit tanpa reload, konsensus tampil, layar terkunci saat istirahat, SELESAI                                                                       |
| `ui-06-konsol-wasit`            | tombol MULAI/JEDA/LANJUTKAN/SELESAIKAN memanggil server dan terbukti dari soket juri; skor konsensus tampil; akses antar role/gelanggang                                                                  |

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

## Perilaku yang dicatat (belum diubah)

- Skor juri tetap diterima saat partai **dijeda** (state tetap `round_N`); hanya istirahat/selesai yang menolak.
- Tombol **BATAL** pada layar juri mengirim `score_type: "undo"` yang ditolak server (`BAD_PAYLOAD`) tanpa umpan balik di layar.
- Tidak ada ping/pong WebSocket: koneksi mati bertahan sampai TCP menyerah.
- Pemulihan setelah **restart server** diuji di tes Go (`internal/orchestrator/runtime_test.go`), bukan di E2E ini.
