# Aplikasi Juri sebagai PWA (dipasang di tablet)

Digital Scoring dapat dipasang ke layar utama tablet/ponsel seperti aplikasi biasa: ikon sendiri, layar penuh tanpa
bilah alamat, dan cangkang aplikasi (HTML/JS/CSS/ikon) tetap terbuka walau jaringan sedang putus.

## Rekomendasi untuk acara: server di laptop + router Wi-Fi (internet tidak pasti)

Rancang untuk **skenario terburuk: tanpa internet**. Aplikasi sendiri tidak membutuhkan internet (font sudah di-host
sendiri, tidak ada permintaan ke domain luar; diuji di `playwright/pwa.spec.ts`). Satu-satunya hal yang bergantung pada
internet adalah sertifikat Let's Encrypt, jadi pakai **CA internal Caddy** dengan **IP tetap**:

1. **IP tetap untuk laptop server.** Di router, buat *DHCP reservation* (atau set IP statis di laptop), mis. `192.168.1.10`.
   Tablet mengetik alamat ini; bila IP berubah, alamat dan sertifikat tablet tidak cocok lagi.
2. `.env.prod`:
   ```
   WEB_PORT=127.0.0.1:8081
   SITE_ADDRESS=192.168.1.10
   TLS_DIRECTIVE=tls internal
   ```
   lalu `docker compose --env-file .env.prod --profile https up -d --build`.
3. **Pasang root CA di setiap tablet (sekali, sebelum hari-H):** buka `http://192.168.1.10/root.crt` di tablet (HTTP biasa,
   hanya berkas ini yang dilayani tanpa HTTPS; sisanya dialihkan ke HTTPS).
   - Android: berkas terunduh → Setelan → Keamanan → Enkripsi & kredensial → Instal sertifikat → **Sertifikat CA**
     (pilih berkas `silat-root-ca.crt`; nama menu bervariasi menurut merek).
   - iPad/iPhone: ketuk **Izinkan** untuk mengunduh profil → Setelan → **Profil Diunduh** → Pasang, lalu Setelan → Umum → Info →
     **Pengaturan Kepercayaan Sertifikat** → aktifkan untuk "Caddy Local Authority".
4. Buka `https://192.168.1.10/` di tablet: tidak boleh ada peringatan sertifikat. Lalu pasang PWA (lihat bawah).
5. **Jangan menghapus volume `caddy_data`.** Di dalamnya ada root CA; bila hilang, CA baru terbentuk dan semua tablet
   harus memasang ulang. Cadangkan berkas `silat-root-ca.crt` dan folder volume itu.

Bila ternyata ada internet **dan** domain, `TLS_DIRECTIVE` dikosongkan memakai Let's Encrypt dan langkah 3 tidak perlu; tetapi
tablet harus bisa menjangkau server lewat domain itu (DNS publik yang menunjuk ke IP server tidak cukup untuk jaringan
tertutup: tantangan HTTP Let's Encrypt harus dapat menjangkau server dari internet). Karena itu **jalur CA internal tetap
yang disarankan** untuk acara di jaringan lokal.

### Pengaturan jaringan yang sering terlewat

- **Matikan "AP/Client Isolation"** di router Wi-Fi. Bila aktif, tablet tidak bisa menjangkau laptop sama sekali.
- **Firewall laptop** harus mengizinkan masuk TCP **80 dan 443** (Windows: izinkan Docker Desktop di jaringan *Private*).
- **Jangan biarkan laptop tidur/hibernasi** (Setelan daya: tidak pernah sleep saat dicolok) dan colokkan ke listrik.
- Sinyal Wi-Fi: satu AP kelas 5 GHz dekat gelanggang; uji dengan semua tablet menyala sekaligus.

### Risiko: semua tablet terlihat satu IP (Docker Desktop di Windows/macOS)

Pembatas login/PIN backend membatasi **10 percobaan per menit per IP** (burst 10) dan semua pembatas bekerja per IP klien.
Di Docker Desktop (Windows/macOS), sambungan masuk sering terlihat datang dari gateway Docker sehingga **seluruh tablet
dianggap satu IP**; beberapa tablet yang login berbarengan atau satu juri yang salah PIN berulang dapat membuat yang lain
terkena "too many requests". Di Linux (Docker Engine) IP klien asli diteruskan. **Uji sebelum acara:** login PIN dari
6–8 tablet dalam waktu satu menit; bila ada yang ditolak dengan "too many requests", jalankan server di Linux, atau
backend langsung di host (tanpa Docker NAT) atau hubungi pengembang untuk menyesuaikan pembatas.

## Detail HTTPS

Peramban hanya mendaftarkan service worker dan menawarkan pemasangan pada **konteks aman**: `https://…` atau
`http://localhost`. Alamat jaringan lokal seperti `http://192.168.1.10` **tidak** memenuhi syarat itu, jadi di tablet
juri aplikasi hanya terbuka sebagai situs biasa (tanpa pemasangan penuh dan tanpa mode offline).

### Mengaktifkan HTTPS dengan docker compose (Caddy)

Repo menyertakan layanan `caddy` (profil `https`, konfigurasi di `deploy/Caddyfile`) yang menerima HTTPS dan
meneruskan semuanya, termasuk WebSocket (`wss://`), ke layanan `web`.

1. Di `.env.prod`, pindahkan `web` ke loopback agar port 80/443 dipakai Caddy dan `web` tidak terbuka ke LAN:
   ```
   WEB_PORT=127.0.0.1:8081
   SITE_ADDRESS=silat.contoh.id        # domain atau IP yang diketik di tablet
   ```
2. Pilih jenis sertifikat:

   | Situasi | Pengaturan | Catatan |
   | --- | --- | --- |
   | Domain publik, server terjangkau internet (port 80/443 terbuka) | `TLS_DIRECTIVE=` (kosong) | Let's Encrypt otomatis dan diperbarui sendiri; tablet tidak perlu pengaturan |
   | LAN tertutup tanpa domain, atau memakai IP | `TLS_DIRECTIVE=tls internal` | CA internal Caddy; **root CA wajib dipasang dan dipercaya di setiap tablet** (lihat di bawah) |
   | Server di balik NAT | tunnel (Cloudflare Tunnel / Tailscale Funnel) di depan layanan `web` | tanpa Caddy |

3. Jalankan: `docker compose --env-file .env.prod --profile https up -d --build`
4. Buka `https://<SITE_ADDRESS>/` di satu tablet; harus tanpa peringatan sertifikat.

Tanpa `--profile https` perilaku lama tidak berubah (HTTP di `WEB_PORT`).

**Cara lain mengambil root CA** (selain `http://<SITE_ADDRESS>/root.crt`), bila perlu mengirimnya lewat AirDrop/USB:
```bash
docker compose --env-file .env.prod cp caddy:/data/caddy/pki/authorities/local/root.crt ./silat-root.crt
```
- Android: Setelan → Keamanan → Enkripsi & kredensial → Instal sertifikat → **Sertifikat CA**.
- iPad/iPhone: buka berkas → pasang profil, lalu Setelan → Umum → Info → **Pengaturan Kepercayaan Sertifikat** → aktifkan.
- Simpan volume `caddy_data` (sudah persisten di compose); bila terhapus, root CA berubah dan semua tablet harus memasang ulang.

**Mengapa `web` harus di loopback:** di balik Caddy, nginx memulihkan IP klien dari `X-Forwarded-For` hanya untuk
sambungan dari jaringan Docker (`set_real_ip_from 172.16.0.0/12`), supaya pembatas login/PIN per IP membedakan setiap
tablet (tanpa ini semua tablet terlihat satu IP dan satu juri salah PIN bisa mengunci semuanya). Port `web` yang
terbuka ke LAN dapat dipakai untuk memalsukan header itu pada Docker Desktop (alamat sumber disamarkan jadi 172.x).

## Cara memasang

- **Android (Chrome):** buka alamat HTTPS aplikasi, menu ⋮ → **Instal aplikasi** (atau **Tambahkan ke layar utama**).
- **iPad/iPhone (Safari):** buka alamat HTTPS, tombol Bagikan → **Tambahkan ke Layar Utama**.
- Aplikasi terbuka di `/judge` (masuk dengan PIN). Pintasan lama-tekan ikon: Juri/Wasit (PIN), Konsol Wasit, Scoreboard.
- Orientasi dikunci **landscape** (layar juri dirancang lebar). Ubah `orientation` di `vite.config.ts` bila perlu.

## Perilaku offline dan pembaruan

- **Offline:** hanya cangkang aplikasi. Skor, status pertandingan, login PIN, dan WebSocket **selalu** butuh server;
  tidak ada data pertandingan yang di-cache (`/api` dan `/ws` dikecualikan dari service worker), jadi layar tidak pernah
  menampilkan data usang. Tanpa jaringan, layar juri menampilkan status terputus dan tombol skor tidak mengirim apa pun.
- **Pembaruan aplikasi:** versi baru diunduh di latar belakang tetapi baru dipakai setelah aplikasi **ditutup dan dibuka
  lagi** (`registerType: 'prompt'`). Tidak ada muat ulang mendadak di tengah pertandingan. Sebelum acara, buka aplikasi di
  setiap tablet dengan internet/LAN agar versi terbaru sudah terunduh, lalu tutup dan buka sekali.
- Font Google di-cache setelah pernah dimuat; bila belum pernah, aplikasi memakai font cadangan.

## Pengujian

```bash
npm run test:pw:pwa     # di apps/web; membangun, menyajikan lewat vite preview, lalu menguji PWA (backend harus jalan)
```

Mencakup: manifest dan ikon, service worker aktif, kriteria installable Chrome, cangkang offline, API tidak di-cache,
serta login PIN + koneksi realtime tetap berfungsi lewat service worker. Rantai HTTPS (Caddy `tls internal` → nginx →
backend, termasuk `wss://`) sudah diuji dengan suite yang sama, dan konfigurasi nginx diuji perilaku
`set_real_ip_from`-nya (XFF palsu dari klien biasa diabaikan, dari jaringan Docker dipercaya). **Belum teruji:**
`docker compose --profile https` itu sendiri (tidak ada daemon Docker di lingkungan pengembangan), pemasangan di
perangkat Android/iOS sungguhan, dan sertifikat Let's Encrypt/root CA di jaringan acara.
