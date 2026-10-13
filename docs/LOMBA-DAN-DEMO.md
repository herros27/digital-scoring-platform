# Menjalankan Versi Lomba dan Versi Demo

Dua stack terpisah penuh di satu repo, dijalankan di **dua mesin berbeda**. Keduanya tidak berbagi database, jaringan,
maupun berkas env.

| | **Lomba** | **Demo** |
|---|---|---|
| Untuk | Perlombaan nyata di venue | Menunjukkan tampilan ke client lewat internet |
| Mesin | **Laptop pribadi** (Windows, Docker Desktop) | **Homeserver** (Ubuntu, RAM 2 GB) |
| Compose | `docker-compose.yml` | `docker-compose.demo.yml` |
| Env | `.env.prod` | `.env.demo` |
| Database | `postgres_data` (data asli) | `demo_postgres_data` (data contoh dari seed) |
| Akses | LAN venue, HTTPS lewat Caddy | Hanya lewat Cloudflare Tunnel; port host cuma `127.0.0.1:8090` |
| Ke internet | **Tidak ada jalan masuk** (tanpa port-forward dan tunnel; laptop hanya butuh internet keluar untuk DNS dan sertifikat) | Ya, via tunnel |
| Alamat | `https://skor.kemz.web.id` (record A → IP LAN laptop) | `https://skor-demo.kemz.web.id` (tunnel) |
| Akun | Dibuat sendiri (`ADMIN_EMAIL`/`ADMIN_PASSWORD`) | Akun demo publik ([CREDENTIALS.md](CREDENTIALS.md)) |

```
LAPTOP (lomba)  : tablet juri ──LAN──▶ Caddy ─▶ nginx ─▶ app ─▶ postgres        (tanpa jalan ke internet)
HOMESERVER (demo): client ──▶ Cloudflare ─▶ cloudflared ─▶ nginx ─▶ app ─▶ postgres (data contoh)
```

Karena beda mesin, tidak ada bentrok nama kontainer atau port, dan demo tidak membebani laptop saat lomba.

---

## A. Versi lomba (laptop pribadi, Windows)

Semua perintah di PowerShell, dari folder proyek.

### Sekali saja

1. Docker Desktop terpasang dan berjalan.
2. `Copy-Item .env.example .env.prod`, lalu isi seperti di [DEPLOY-HOMESERVER.md](DEPLOY-HOMESERVER.md) langkah 3:
   `ENV=production`, `SEED_DATA=false`, `POSTGRES_PASSWORD`, `JWT_SECRET`, `ADMIN_EMAIL`/`ADMIN_PASSWORD`,
   `WEB_PORT=127.0.0.1:8081`, dan pengaturan HTTPS (`SITE_ADDRESS`, `TLS_DIRECTIVE`/`ACME_DNS`).
   Acak tanpa openssl:
   ```powershell
   -join ((1..24) | % { '{0:x2}' -f (Get-Random -Maximum 256) })                      # POSTGRES_PASSWORD (hex)
   [Convert]::ToBase64String((1..48 | % { Get-Random -Maximum 256 }))                  # JWT_SECRET
   ```
3. **Domain lomba `skor.kemz.web.id` di Cloudflare** (tanpa memasang CA di tablet). Sertifikat Let's Encrypt asli
   didapat Caddy lewat tantangan DNS, jadi semua tablet langsung percaya.
   1. Cari IP LAN laptop: `ipconfig`, lihat **IPv4 Address** pada adapter yang dipakai (Ethernet/Wi-Fi), mis.
      `192.168.1.20`.
   2. Cloudflare → pilih zona `kemz.web.id` → **DNS → Records → Add record**: Type **A**, Name `skor`,
      IPv4 address = IP dari langkah 1, **Proxy status: DNS only** (awan abu-abu, bukan oranye). Save.
   3. My Profile → **API Tokens → Create Token** → template **Edit zone DNS**. Pada *Zone Resources* pilih
      **Include → Specific zone → `kemz.web.id`**. Create Token, lalu simpan tokennya (hanya tampil sekali).
   4. Isi di `.env.prod`:
      ```
      SITE_ADDRESS=skor.kemz.web.id
      ACME_DNS=acme_dns cloudflare {env.CLOUDFLARE_API_TOKEN}
      CLOUDFLARE_API_TOKEN=<token dari langkah 3>
      ```

   Record ini hanya bisa dibuka dari jaringan yang sama dengan laptop karena menunjuk ke IP LAN. Demo memakai nama
   lain, `skor-demo.kemz.web.id`, lewat tunnel (bagian B); keduanya tidak bentrok.
4. **IP laptop harus tetap dan sama dengan record A.** Pakai DHCP reservation di router atau IP statis. Di venue
   dengan router lain, IP laptop berbeda: ubah record A ke IP baru di dashboard Cloudflare (satu menit), atau bawa
   **travel router sendiri** supaya subnet dan IP laptop selalu sama.
5. **Tablet harus bisa me-resolve `skor.kemz.web.id`.** Itu perlu router venue yang terhubung ke internet. Bila
   internet venue mati dan tablet belum menyimpan cache DNS, nama tidak terbuka. Cadangan: hotspot HP yang ada
   internetnya, atau travel router dengan override DNS `skor.kemz.web.id` → IP laptop.
6. **Minta sertifikat di rumah, jauh sebelum hari lomba.** Laptop butuh internet saat pertama meminta sertifikat dan
   saat perpanjangan (sekitar tiap 60 hari). Sertifikat tersimpan di volume `caddy_data`, jadi sesudahnya lomba tetap
   jalan walau offline. Cek: `docker logs scoring-https 2>&1 | Select-String "certificate obtained"`.
7. **Windows Defender Firewall**: izinkan koneksi masuk TCP 80 dan 443 untuk Docker Desktop pada jaringan Private,
   atau tablet tidak bisa menjangkau laptop.
8. **Jangan sampai laptop tidur**: Settings → Power → Sleep = Never saat tersambung charger, dan "When I close the
   lid" = Do nothing. Pakai kabel LAN bila ada, bukan Wi-Fi.

### Menjalankan

```powershell
docker compose --env-file .env.prod --profile https up -d --build
```

Build pertama di laptop memakan waktu beberapa menit; sesudahnya cepat. Jangan lupa build sebelum hari lomba, bukan
di venue.

### Periksa

```powershell
docker ps --format "table {{.Names}}\t{{.Status}}\t{{.Ports}}"
curl.exe -sI https://skor.kemz.web.id | Select-Object -First 1      # HTTP/2 200
```

Dari tablet di Wi-Fi/LAN yang sama: `https://skor.kemz.web.id`; juri `/judge/login`, wasit `/referee`,
scoreboard `/scoreboard`, panel operasi `/ops`.

### Matikan dan backup

```powershell
docker compose --env-file .env.prod --profile https down        # data aman di volume
.\scripts\backup-db.ps1                                          # backup database (taruh salinannya di luar laptop)
```

Jangan memakai `down -v` pada stack lomba: itu menghapus database asli.

### Memperbarui

`git pull`, lalu jalankan lagi perintah `up -d --build`. Lakukan di luar jam lomba.

---

## B. Versi demo (homeserver)

### Sekali saja: buat tunnel `skor-demo.kemz.web.id` di Cloudflare

Syarat: domain `kemz.web.id` sudah dikelola di akun Cloudflare yang sama (sudah, dipakai untuk `skor`). Nama menu
Cloudflare kadang berubah; cari "Tunnels" bila letaknya berbeda.

1. Buka [one.dash.cloudflare.com](https://one.dash.cloudflare.com) (Zero Trust) → **Networks → Tunnels** →
   **Create a tunnel**.
2. Pilih tipe **Cloudflared**, beri nama `skor-demo`, **Save tunnel**.
3. Pilih lingkungan **Docker**. Dashboard menampilkan perintah `docker run cloudflare/cloudflared ... --token eyJ...`.
   **Salin hanya nilai setelah `--token`** (itu `CLOUDFLARE_TUNNEL_TOKEN`). Jangan jalankan perintahnya: tunnel akan
   dijalankan oleh compose demo. **Next**.
4. Tab **Public Hostname** → **Add a public hostname**:

   | Isian | Nilai |
   |---|---|
   | Subdomain | `skor-demo` |
   | Domain | `kemz.web.id` |
   | Path | kosongkan |
   | Type | `HTTP` |
   | URL | `web:80` |

   Simpan. Cloudflare membuat record CNAME `skor-demo` otomatis (jangan buat sendiri). `web:80` dikenali karena
   `cloudflared` satu jaringan compose dengan layanan `web`. WebSocket jalan tanpa pengaturan tambahan.
5. Setelah stack demo menyala (bagian Menjalankan di bawah), status tunnel di dashboard berubah menjadi **Healthy**.
   Bila tetap *Down*, cek `docker logs` kontainer `cloudflared` (token salah atau server tanpa internet).

Opsional, disarankan: batasi akses ke client saja. **Access → Applications → Add an application → Self-hosted**,
domain `skor-demo.kemz.web.id`, buat policy *Allow* untuk email client (kode masuk dikirim ke email). Gratis sampai
50 pengguna. Tanpa ini, siapa pun yang tahu URL bisa membuka demo (lihat "Yang perlu diketahui").

### Sekali saja: env demo di server

1. Di server, ambil kode dan buat `.env.demo` dari template, lalu isi tiga nilai:
   ```bash
   cd ~/DigitalScoring && git pull
   cp .env.demo.example .env.demo
   openssl rand -hex 24       # → POSTGRES_PASSWORD
   openssl rand -base64 48    # → JWT_SECRET
   # CLOUDFLARE_TUNNEL_TOKEN = token dari langkah 3 di atas
   ```

### Menjalankan

Build di laptop (server 2 GB sangat lambat), lalu kirim image ke homeserver. Di laptop (PowerShell):

```powershell
docker build -t scoring-demo-app .
docker build -t scoring-demo-web apps/web
docker save -o demo-images.tar scoring-demo-app scoring-demo-web
scp demo-images.tar kemas@homeserver:~/
ssh kemas@homeserver "docker load -i ~/demo-images.tar && rm ~/demo-images.tar"
Remove-Item demo-images.tar
```

Di homeserver:

```bash
cd ~/DigitalScoring
docker compose -f docker-compose.demo.yml --env-file .env.demo up -d --no-build
```

### Periksa

```bash
docker compose -f docker-compose.demo.yml --env-file .env.demo ps
curl -sI http://127.0.0.1:8090 | head -1              # dari server: HTTP/1.1 200
curl -sI https://skor-demo.kemz.web.id | head -1              # dari luar: lewat tunnel
free -h                                               # available > 500 MB
```

Data contoh terisi otomatis saat start (seed). Akun demo ada di [CREDENTIALS.md](CREDENTIALS.md); admin:
`admin@silat.id` / `DemoPass_2026`.

### Mematikan dan mereset

```bash
# matikan (data demo tetap)
docker compose -f docker-compose.demo.yml --env-file .env.demo down

# reset ke data contoh bersih (hanya menghapus data DEMO)
docker compose -f docker-compose.demo.yml --env-file .env.demo down -v
docker compose -f docker-compose.demo.yml --env-file .env.demo up -d --no-build
```

### Memperbarui demo

`git pull` di laptop dan server, ulangi build + kirim image, lalu `up -d --no-build` di server.

### Yang perlu diketahui

- Siapa pun yang tahu URL demo bisa login sebagai admin dengan kredensial publik. Data lomba tidak terpengaruh
  (ada di laptop, tanpa jalan ke internet), dan demo pulih dengan restart (seed mengembalikan password akun demo).
  Untuk membatasi ke client saja, pasang Cloudflare Access (login email) di hostname demo.
- Jika homeserver masih menjalankan stack lomba lama (`scoring-*`), hentikan agar RAM 2 GB cukup:
  `docker compose --env-file .env.prod --profile https down`.

---

## Checklist sebelum hari lomba (laptop)

- [ ] Lomba dijalankan dari laptop; `docker ps` hanya berisi kontainer `scoring-*` lomba.
- [ ] Tidak ada `cloudflared` di laptop, dan `.env.prod` tidak berisi `CLOUDFLARE_TUNNEL_TOKEN`.
- [ ] `.env.prod`: `SEED_DATA=false`.
- [ ] IP laptop sama dengan record A `skor.kemz.web.id`; tablet bisa membuka `https://skor.kemz.web.id`.
- [ ] Router venue punya internet (untuk DNS tablet), atau hotspot/travel router cadangan siap.
- [ ] Sertifikat sudah didapat dan masih berlaku (`certificate obtained` di log `scoring-https`).
- [ ] Firewall Windows mengizinkan 80/443; sleep dimatikan; charger terpasang.
- [ ] Router tidak punya port-forward ke laptop.
- [ ] Backup database sebelum mulai, dan salin ke luar laptop sesudah tiap hari.
