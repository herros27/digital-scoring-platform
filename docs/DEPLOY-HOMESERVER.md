# Deploy ke Server Lokal (homeserver)

Panduan langkah demi langkah menjalankan Digital Scoring di server lokal (laptop Ubuntu) dengan HTTPS lewat
domain Cloudflare, sehingga tablet juri bisa memasang PWA tanpa root CA. Latar belakang HTTPS/PWA: [PWA.md](PWA.md).
Memeriksa kesehatan server dan menutup terminal yang menganggur: [CEK-SERVER.md](CEK-SERVER.md).
Rencana terbaru: lomba dijalankan di laptop pribadi dan demo online di homeserver (stack terpisah), lihat
[LOMBA-DAN-DEMO.md](LOMBA-DAN-DEMO.md). Panduan di bawah masih menjelaskan penyiapan server dan HTTPS-nya.

## Gambaran

```
tablet / PC ──https://skor.kemz.web.id──▶ Caddy :443 (scoring-https)
                                            └─▶ nginx :80 (scoring-web, 127.0.0.1:8081)
                                                  ├─▶ backend :8080 (scoring-backend, 127.0.0.1:8080)
                                                  └─────────▶ PostgreSQL :5432 (scoring-postgres, 127.0.0.1:5432)
```

- DNS `skor.kemz.web.id` di Cloudflare menunjuk ke **IP LAN server** (`192.168.1.11`). Hanya perangkat di jaringan
  yang sama yang bisa membukanya.
- Sertifikat Let's Encrypt didapat lewat tantangan DNS Cloudflare, jadi server tidak perlu terbuka ke internet.
- Hanya Caddy (80/443) yang terbuka ke jaringan. Backend, nginx, dan Postgres terikat ke `127.0.0.1`.

Spesifikasi server: Celeron N3060 (2 core), RAM 2 GB, HDD 5400 rpm. **Build image Go/Node di server ini sangat lambat
(20+ menit) dan bisa gagal bila SSH putus**, jadi image dibangun di PC lalu dikirim ke server (langkah 4).

---

## 1. Persiapan server (sekali saja)

```bash
# Docker + plugin compose
sudo apt update && sudo apt install -y docker.io docker-compose-v2 git tmux
sudo usermod -aG docker $USER   # logout-login setelah ini

# Laptop jangan tidur saat lid ditutup
sudo sed -i 's/^#\?HandleLidSwitch=.*/HandleLidSwitch=ignore/' /etc/systemd/logind.conf
sudo systemctl restart systemd-logind

# Ambil kode
git clone git@github.com:herros27/DigitalScoring.git ~/DigitalScoring
```

Di router, buat **reservasi DHCP** (IP tetap) untuk server, misalnya `192.168.1.11`. Bila IP berubah, record DNS
Cloudflare harus ikut diubah.

> **Tailscale Serve memakai port 443** dan bentrok dengan Caddy. Matikan bila aktif: `sudo tailscale serve reset`
> (SSH lewat Tailscale tetap jalan).

## 2. Cloudflare (sekali saja)

1. DNS → tambah record **A** `skor` → `192.168.1.11`, **Proxy status: DNS only** (awan abu-abu).
2. My Profile → API Tokens → Create Token → template **Edit zone DNS**, batasi ke zona `kemz.web.id`. Simpan tokennya.

## 3. Isi `.env.prod` di server (sekali saja)

```bash
cd ~/DigitalScoring
cp .env.example .env.prod
nano .env.prod
```

Nilai yang penting:

```ini
ENV=production
AUTO_MIGRATE=true
SEED_DATA=false                     # true = akun demo berpassword lemah, JANGAN untuk lomba nyata
JWT_SECRET=<hasil: openssl rand -base64 48>
ADMIN_EMAIL=admin@contoh.id         # admin pertama dibuat otomatis saat start
ADMIN_PASSWORD=<min 8 karakter>     # hapus setelah admin pertama berhasil login

POSTGRES_USER=scoring_user
POSTGRES_DB=digital_scoring
POSTGRES_PASSWORD=<hasil: openssl rand -hex 24>   # wajib; hanya dibaca saat volume database pertama dibuat

# Hanya Caddy yang terbuka ke jaringan. PORT dan POSTGRES_PORT berisi nomor port saja: compose sudah mengikatnya ke
# 127.0.0.1 (jangan tulis "127.0.0.1:8080"). WEB_PORT memakai bentuk host:port.
PORT=8080
WEB_PORT=127.0.0.1:8081
POSTGRES_PORT=5432

# HTTPS
SITE_ADDRESS=skor.kemz.web.id
TLS_DIRECTIVE=
ACME_DNS=acme_dns cloudflare {env.CLOUDFLARE_API_TOKEN}
CLOUDFLARE_API_TOKEN=<token dari langkah 2>
```

### Atau: isi di PC, lalu salin ke server

`.env.prod` diabaikan git, jadi `git pull` tidak membawanya. Bila sudah disiapkan di PC, salin dengan `scp` (jalankan di
PC, dari folder proyek) dan batasi izin bacanya:

```bash
scp .env.prod kemas@homeserver:~/DigitalScoring/.env.prod
ssh kemas@homeserver "chmod 600 ~/DigitalScoring/.env.prod"
```

Folder `~/DigitalScoring` harus sudah ada (langkah 1). Periksa bahwa semua variabel wajib terisi, tanpa menampilkan
isinya (tidak ada keluaran bila lolos; bila `POSTGRES_PASSWORD` kosong, compose menolak dengan pesannya):

```bash
ssh kemas@homeserver "cd ~/DigitalScoring && docker compose --env-file .env.prod config -q && echo OK"
```

Sebelum menyalin, pastikan nilai di PC cocok untuk server: `ENV=production`, `SEED_DATA=false`, `POSTGRES_PASSWORD` terisi,
`PORT=8080` dan `POSTGRES_PORT=5432` (nomor port saja), `WEB_PORT=127.0.0.1:8081`, serta `SITE_ADDRESS`, `ACME_DNS`, dan
`CLOUDFLARE_API_TOKEN` untuk HTTPS. Salin lagi setiap kali isinya berubah, lalu jalankan ulang `up -d` (langkah 5).

## 4. Build image di PC, kirim ke server

Jalankan di PC (Docker Desktop), dari folder proyek. Nama image mengikuti nama proyek compose `digitalscoring`
(nama folder di server `~/DigitalScoring`), jadi compose di server langsung memakainya.

```bash
docker build -t digitalscoring-app .
docker build -t digitalscoring-web apps/web
docker build -t digitalscoring-caddy deploy/caddy      # hanya perlu sekali / saat Caddy diperbarui

docker save -o scoring-images.tar digitalscoring-app digitalscoring-web digitalscoring-caddy
scp scoring-images.tar kemas@homeserver:~/
ssh kemas@homeserver "docker load -i ~/scoring-images.tar && rm ~/scoring-images.tar"
```

Bila nama image tidak cocok, lihat nama yang diharapkan compose di server:
`docker compose --env-file .env.prod --profile https config --images`.

<details>
<summary>Alternatif: build langsung di server (lambat)</summary>

Wajib di dalam `tmux` agar build tidak batal saat SSH putus:

```bash
tmux new -s build
cd ~/DigitalScoring
docker compose --env-file .env.prod --profile https up -d --build
# lepas: Ctrl+B lalu D   ·   kembali: tmux attach -t build
```
</details>

## 5. Jalankan

```bash
cd ~/DigitalScoring
docker compose --env-file .env.prod --profile https up -d --no-build
```

## 6. Periksa

```bash
docker ps --format "table {{.Names}}\t{{.Status}}\t{{.Ports}}"   # 4 container, 3 di antaranya (healthy)
docker logs scoring-https 2>&1 | grep "certificate obtained"      # sertifikat didapat
curl -sI https://skor.kemz.web.id | head -1                       # HTTP/2 200
```

Buka `https://skor.kemz.web.id` dari tablet/PC di jaringan yang sama:
tablet juri `/judge/login`, konsol wasit `/referee`, scoreboard `/scoreboard`, panel operasi `/ops`.

---

## Memperbarui aplikasi

1. Di PC: `git pull`, lalu ulangi **langkah 4** (cukup `digitalscoring-app` dan/atau `digitalscoring-web` yang berubah).
2. Di server:
   ```bash
   cd ~/DigitalScoring
   git pull                     # agar docker-compose.yml, Caddyfile, dan migrasi ikut terbaru
   docker compose --env-file .env.prod --profile https up -d --no-build
   docker image prune -f        # hapus image lama
   ```

Migrasi database berjalan otomatis saat backend start (`AUTO_MIGRATE=true`). Partai yang sedang berlangsung
dipulihkan setelah restart (lihat README, "Pemulihan Setelah Server Restart"); tetap usahakan update di luar jam lomba.

## Backup dan restore database

```bash
# backup
docker exec scoring-postgres pg_dump -U scoring_user digital_scoring > ~/backup-$(date +%F-%H%M).sql

# restore (ke database kosong)
cat ~/backup-XXXX.sql | docker exec -i scoring-postgres psql -U scoring_user digital_scoring
```

Lakukan backup sebelum update dan setelah setiap hari pertandingan. Salin berkasnya keluar dari server.

## Checklist hari pertandingan

- [ ] Server pakai **kabel LAN** ke router, bukan Wi-Fi.
- [ ] IP server sama dengan record DNS Cloudflare (`ip -4 addr`).
- [ ] Charger laptop terpasang (baterai = cadangan saat listrik padam).
- [ ] `SEED_DATA=false` dan akun demo tidak dipakai (backend menolak start bila `SEED_DATA=true` di luar development).
- [ ] `POSTGRES_PASSWORD` terisi dan tersimpan di tempat aman (tanpa itu database tidak bisa dibuka).
- [ ] Tidak ada proses berat lain di server: build, sesi agent/VS Code Remote (`ps aux --sort=-%mem | head`).
- [ ] `free -h`: kolom available > 500 MB.
- [ ] Backup database sebelum mulai.

---

## Masalah yang pernah terjadi

| Gejala | Penyebab | Solusi |
|---|---|---|
| `client_loop: send disconnect: Connection reset` saat build, image tidak jadi | SSH putus, build ikut batal | Build di PC (langkah 4) atau di dalam `tmux` |
| `failed to bind host port 0.0.0.0:443/tcp: address already in use` | Tailscale Serve memakai 443 | `sudo tailscale serve reset`. Cek pemakai port: `ss -ltn '( sport = :443 )'` |
| Log Caddy: `dial udp 8.8.8.8:53: connect: network is unreachable`, padahal internet server normal | Container Caddy tertinggal tanpa jaringan setelah start yang gagal | `docker compose --env-file .env.prod --profile https up -d --no-build --force-recreate caddy` |
| Server lambat, load tinggi, swap terpakai | RAM 2 GB habis oleh build atau proses lain | `ps aux --sort=-%mem \| head`, hentikan yang tidak perlu, `docker builder prune -f` |
| Disk penuh | Cache build Docker menumpuk | `docker system df`, lalu `docker builder prune -f` |
