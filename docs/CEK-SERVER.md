# Cek Kondisi Server

Perintah untuk memeriksa kesehatan homeserver dan membersihkan terminal (sesi SSH) yang menganggur.
Semua perintah dijalankan di server (`ssh kemas@homeserver`). Panduan deploy: [DEPLOY-HOMESERVER.md](DEPLOY-HOMESERVER.md).

## 1. Cek cepat (salin semua sekaligus)

```bash
echo "=== LOAD ===";      uptime
echo "=== RAM ===";       free -h
echo "=== DISK ===";      df -h /
echo "=== TOP RAM ===";   ps aux --sort=-%mem | head -8
echo "=== TOP CPU ===";   ps aux --sort=-%cpu | head -8
echo "=== CONTAINER ==="; docker ps --format "table {{.Names}}\t{{.Status}}"
echo "=== DOCKER ===";    docker stats --no-stream --format "table {{.Name}}\t{{.CPUPerc}}\t{{.MemUsage}}"
echo "=== SESI ===";      w
```

## 2. Cara membaca hasilnya

| Bagian | Normal | Perlu dicek |
|---|---|---|
| `uptime` → *load average* (3 angka: 1, 5, 15 menit) | di bawah **2** (server punya 2 core) | di atas 2 terus-menerus: ada proses berat, lihat TOP CPU |
| `free -h` → kolom **available** | di atas **500 Mi** | di bawah 300 Mi: lihat TOP RAM |
| `free -h` → baris **Swap used** | kecil / tidak naik | terus naik: RAM kurang, server melambat |
| `df -h /` → **Use%** | di bawah **80%** | di atas 80%: `docker builder prune -f` |
| `docker ps` | 4 container `Up`, 3 di antaranya `(healthy)` | `Restarting` / `Exited` / `unhealthy`: lihat log, langkah 5 |

Kolom `buff/cache` di `free -h` adalah cache disk. Kernel melepasnya sendiri saat RAM dibutuhkan, jadi angka besar di
situ tidak masalah.

Proses yang **wajar** muncul di TOP RAM: `dockerd`, `containerd`, `tailscaled`, `postgres`, `caddy`, `nginx`, `server`
(backend). Yang **patut dicurigai** bila tidak sedang dipakai: `go build`/`link`/`node` (build Docker), `agy`,
`node ... vscode-server` (VS Code Remote), `npm`.

## 3. Melihat terminal yang sedang terbuka

```bash
tty      # terminal KAMU sendiri, mis. /dev/pts/9  → JANGAN ditutup
w        # semua sesi yang login
```

Contoh hasil `w`:

```
USER   TTY    FROM             LOGIN@  IDLE    JCPU   PCPU  WHAT
kemas  pts/0  100.65.149.101   Wed12   2days   0.05s  0.05s -bash
kemas  pts/3  100.122.238.90   00:56   11:00m  35:02  35:01 agy
kemas  pts/8  100.65.149.101   11:44   3:00    0.30s  0.10s docker compose ... up -d --build
kemas  pts/9  100.65.149.101   11:56   0.00s   0.02s  0.00s w
```

- **TTY**: nama terminal (`pts/N` = sesi SSH, `tty1`/`seat0` = layar laptop, abaikan).
- **IDLE**: sudah berapa lama tidak ada ketikan.
- **WHAT**: perintah yang sedang jalan di terminal itu.

Terminal **aman ditutup** bila: bukan terminal kamu (`tty`), IDLE lama (jam/hari), dan WHAT hanya `-bash`.
Pada contoh di atas: **pts/0**.

Terminal **jangan langsung ditutup** bila WHAT berisi pekerjaan yang masih jalan (build, `docker compose`, `git`,
editor). Contoh: **pts/8** (build sedang jalan). Tutup setelah selesai, atau biarkan.

Terminal yang menjalankan program lama tanpa dipakai (contoh **pts/3**, `agy` sudah 11 jam) boleh ditutup bila kamu yakin
tidak ada pekerjaan yang belum disimpan di dalamnya.

Untuk melihat detail proses di satu terminal:

```bash
ps -t pts/3 -o pid,etime,%cpu,%mem,args     # etime = sudah jalan berapa lama
```

## 4. Menutup terminal

```bash
# satu terminal
pkill -HUP -t pts/0

# beberapa sekaligus (pisahkan dengan koma, tanpa spasi)
pkill -HUP -t pts/0,pts/1,pts/3

# cek lagi
w
```

Bila terminal masih muncul setelah beberapa detik, paksa:

```bash
pkill -KILL -t pts/3
```

Menghentikan satu proses saja (tanpa menutup terminalnya), pakai PID dari `ps`:

```bash
kill 1177017        # berhenti baik-baik
kill -9 1177017     # paksa, bila yang atas tidak mempan
```

> **Selalu cek `tty` dulu** dan jangan masukkan terminalmu sendiri ke perintah di atas, nanti koneksimu ikut terputus.
> Container aplikasi (Docker) **tidak** ikut mati saat terminal ditutup.

## 5. Bila container bermasalah

```bash
cd ~/DigitalScoring
docker compose --env-file .env.prod logs --tail 50 app      # backend
docker compose --env-file .env.prod logs --tail 50 web      # nginx
docker compose --env-file .env.prod logs --tail 50 caddy    # HTTPS
docker compose --env-file .env.prod logs --tail 50 postgres

# nyalakan ulang satu layanan
docker compose --env-file .env.prod --profile https up -d --no-build --force-recreate caddy
```

Masalah umum dan solusinya ada di tabel akhir [DEPLOY-HOMESERVER.md](DEPLOY-HOMESERVER.md#masalah-yang-pernah-terjadi).

## 6. Bersih-bersih rutin

```bash
docker system df            # berapa banyak yang bisa dihapus (kolom RECLAIMABLE)
docker builder prune -f     # hapus cache build (aman, hanya membuat build berikutnya lebih lama)
docker image prune -f       # hapus image lama yang tidak dipakai
```

Jangan pakai `docker volume prune` atau `docker system prune --volumes`: **data database ada di volume** dan bisa
ikut terhapus.
