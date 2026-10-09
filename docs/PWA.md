# Aplikasi Juri sebagai PWA (dipasang di tablet)

Digital Scoring dapat dipasang ke layar utama tablet/ponsel seperti aplikasi biasa: ikon sendiri, layar penuh tanpa
bilah alamat, dan cangkang aplikasi (HTML/JS/CSS/ikon) tetap terbuka walau jaringan sedang putus.

## Yang WAJIB: HTTPS

Peramban hanya mendaftarkan service worker dan menawarkan pemasangan pada **konteks aman**: `https://…` atau
`http://localhost`. Alamat jaringan lokal seperti `http://192.168.1.10` **tidak** memenuhi syarat itu, jadi di tablet
juri aplikasi hanya terbuka sebagai situs biasa (tanpa pemasangan penuh dan tanpa mode offline). Pilihan yang umum:

| Cara | Cocok untuk | Catatan |
| --- | --- | --- |
| Domain + sertifikat Let's Encrypt (Caddy/nginx/Traefik) | server di internet/VPS atau LAN dengan domain | paling sederhana; tablet tidak perlu pengaturan |
| Caddy dengan sertifikat internal (`tls internal`) atau mkcert | LAN tertutup tanpa internet | root CA harus dipasang dan dipercaya di setiap tablet |
| Tunnel (Cloudflare Tunnel, Tailscale Funnel) | server di balik NAT | butuh internet saat acara |

Contoh Caddy di depan layanan `web` (nginx) pada docker compose:

```
silat.contoh.id {
    reverse_proxy web:80
}
```

WebSocket (`/ws/…`) ikut lewat reverse proxy yang sama dan otomatis menjadi `wss://`.

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
serta login PIN + koneksi realtime tetap berfungsi lewat service worker. **Belum teruji:** pemasangan di perangkat
Android/iOS sungguhan dan sertifikat HTTPS di jaringan acara.
