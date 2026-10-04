# 🔐 Daftar Akun & Kredensial Sistem (DigitalScoring Pencak Silat)

> **Catatan Penggunaan:**  
> Seluruh kredensial, akun, PIN perangkat, dan data turnamen pada dokumen ini digenerate secara otomatis melalui database seeder backend (`cmd/seed/main.go`).  
> Dokumen ini dirancang sebagai panduan referensi utama bagi tim pengembang Frontend, QA/Tester, dan Operator untuk pengujian integrasi antarmuka dan autentikasi multi-peran.

---

## 🚀 Cara Menjalankan & Mengisi Ulang Data (Seeding)

Jika database Anda baru dibuat atau ingin direset kembali ke status default:

```bash
# Jalankan seeder melalui CLI (idempoten & aman dijalankan berkali-kali)
go run ./cmd/seed/main.go
```

Atau set environment variable pada file `.env`:
```env
SEED_DATA=true
```

---

## 1. Official & Admin (`POST /api/v1/auth/login`)

Digunakan untuk masuk ke **Admin Dashboard**, **Dewan Juri Pad**, dan konsol manajemen turnamen.

| Role | Email | Password | Hak Akses & Deskripsi |
| :--- | :--- | :--- | :--- |
| `admin` | `admin@silat.id` | `DemoPass_2026` | **Full Administrator**: Manajemen kompetisi, arena, kategori, pendaftaran, generate PIN juri, dan bagan turnamen. |
| `dewan` | `dewan@silat.id` | `DemoPass_2026` | **Ketua Dewan Pertandingan**: Pengawasan pertandingan multi-arena, verifikasi hasil pertandingan, dan penanganan protes. |
| `wasit` | `wasit@silat.id` | `DemoPass_2026` | **Wasit Resmi**: Akses status pertandingan dan pengawasan arena. |
| `operator` | `operator@silat.id` | `DemoPass_2026` | **Operator Gelanggang**: Pengendali display layar pertandingan dan input operasional lapangan. |

### Contoh Request Login Official:
```bash
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "email": "admin@silat.id",
    "password": "DemoPass_2026"
  }'
```

### Format Response:
```json
{
  "code": "OK",
  "data": {
    "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
    "user": {
      "id": "7f8b91c2-3e4a-4d5b-8f1a-9c0d1e2f3a4b",
      "name": "Administrator Utama",
      "email": "admin@silat.id",
      "role": "admin"
    }
  }
}
```

---

## 2. Kontingen / Manajer Tim (`POST /api/v1/auth/contingent/login`)

Digunakan untuk masuk ke **Portal Kontingen** guna mendaftarkan atlet, melihat jadwal, dan memantau status verifikasi atlet.

| Kontingen | Email | Password | PIC / Kontak |
| :--- | :--- | :--- | :--- |
| **Tapak Suci Putera Muhammadiyah** | `tapaksuci@contingent.id` | `DemoPass_2026` | Ahmad Dahlan (`081234567801`) |
| **Persaudaraan Setia Hati Terate (PSHT)** | `psht@contingent.id` | `DemoPass_2026` | Bambang Haryo (`081234567802`) |
| **PPS Betako Merpati Putih** | `merpatiputih@contingent.id` | `DemoPass_2026` | Cecep Rahman (`081234567803`) |
| **KSN Perisai Diri** | `perisaidiri@contingent.id` | `DemoPass_2026` | Danang Wijaya (`081234567804`) |

### Contoh Request Login Kontingen:
```bash
curl -X POST http://localhost:8080/api/v1/auth/contingent/login \
  -H "Content-Type: application/json" \
  -d '{
    "email": "tapaksuci@contingent.id",
    "password": "DemoPass_2026"
  }'
```

---

## 3. Wasit & Juri Sudut Gelanggang (`POST /api/v1/auth/judge-login`)

Digunakan untuk login cepat perangkat touch-screen (**Tablet Juri PWA** dan **Referee Control Pad**) menggunakan kombinasi **Arena ID + PIN**.

### Parameter Turnamen & Arena yang Tersedia:
- **Competition ID**: `b26d7145-d341-4795-ac58-bff86d13cbf8` *(Piala Menpora 2026)*
- **Gelanggang A ID**: `c7397556-eb39-406f-aa89-6b3053a2fd08`
- **Gelanggang B ID**: `32d25db4-5636-46ad-bb2c-a43d052a405a`

### A. Gelanggang A (`c7397556-eb39-406f-aa89-6b3053a2fd08`)

| Peran | Kursi (Seat) | PIN | Perangkat / Antarmuka |
| :--- | :---: | :---: | :--- |
| `wasit` | - | `111111` | **Referee Pad**: Kendali Timer (*Start*, *Pause*, *Resume*, *Finish*), Sanksi Teguran/Peringatan. |
| `juri` | Kursi 1 | `123001` | **Tablet Juri 1**: Input Pukulan, Tendangan, Bantingan. |
| `juri` | Kursi 2 | `123002` | **Tablet Juri 2**: Input Pukulan, Tendangan, Bantingan. |
| `juri` | Kursi 3 | `123003` | **Tablet Juri 3**: Input Pukulan, Tendangan, Bantingan. |

### B. Gelanggang B (`32d25db4-5636-46ad-bb2c-a43d052a405a`)

| Peran | Kursi (Seat) | PIN | Perangkat / Antarmuka |
| :--- | :---: | :---: | :--- |
| `wasit` | - | `222222` | **Referee Pad**: Kendali Timer (*Start*, *Pause*, *Resume*, *Finish*), Sanksi Teguran/Peringatan. |
| `juri` | Kursi 1 | `234001` | **Tablet Juri 1**: Input Pukulan, Tendangan, Bantingan. |
| `juri` | Kursi 2 | `234002` | **Tablet Juri 2**: Input Pukulan, Tendangan, Bantingan. |
| `juri` | Kursi 3 | `234003` | **Tablet Juri 3**: Input Pukulan, Tendangan, Bantingan. |

### Contoh Request Login PIN Juri (Gelanggang A - Juri 1):
```bash
curl -X POST http://localhost:8080/api/v1/auth/judge-login \
  -H "Content-Type: application/json" \
  -d '{
    "competition_id": "b26d7145-d341-4795-ac58-bff86d13cbf8",
    "arena_id": "c7397556-eb39-406f-aa89-6b3053a2fd08",
    "pin": "123001"
  }'
```

### Format Response:
```json
{
  "code": "OK",
  "data": {
    "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
    "role": "judge",
    "seat_number": 1,
    "arena_id": "c7397556-eb39-406f-aa89-6b3053a2fd08",
    "competition_id": "b26d7145-d341-4795-ac58-bff86d13cbf8"
  }
}
```

---

## 4. Logout Sesi (`POST /api/v1/auth/logout`)

Digunakan untuk konfirmasi logout dari aplikasi.

> **Mekanisme Stateless JWT:**  
> Pada sistem berbasis token JWT murni, penghancuran sesi utama dilakukan di sisi **Frontend / Client** dengan menghapus token dari `localStorage` atau Pinia store (`authStore.logout()`).  
> Endpoint ini disediakan agar klien dapat melakukan *handshake* logout formal ke server.

### Contoh Request Logout:
```bash
curl -X POST http://localhost:8080/api/v1/auth/logout \
  -H "Authorization: Bearer <TOKEN>"
```

### Format Response:
```json
{
  "code": "OK",
  "data": {
    "message": "successfully logged out"
  }
}
```

---

## 5. Endpoint Publik & Display Scoreboard (Tanpa Autentikasi)

Endpoint berikut dapat dibuka langsung di browser atau perangkat scoreboard penonton tanpa header Authorization:

- **Scoreboard Gelanggang A**:  
  `GET /api/v1/live/scoreboard/c7397556-eb39-406f-aa89-6b3053a2fd08`
- **Scoreboard Gelanggang B**:  
  `GET /api/v1/live/scoreboard/32d25db4-5636-46ad-bb2c-a43d052a405a`
- **Bagan Turnamen (Tree Visualizer)**:  
  `GET /api/v1/competitions/b26d7145-d341-4795-ac58-bff86d13cbf8/brackets`
- **Rekapitulasi Medali Turnamen**:  
  `GET /api/v1/competitions/b26d7145-d341-4795-ac58-bff86d13cbf8/report/medals`
- **Ringkasan Turnamen (Summary)**:  
  `GET /api/v1/competitions/b26d7145-d341-4795-ac58-bff86d13cbf8/report/summary`
- **Export Laporan Turnamen (CSV)**:  
  `GET /api/v1/competitions/b26d7145-d341-4795-ac58-bff86d13cbf8/report/export`

---

## 6. Koneksi Real-Time WebSocket

Untuk menghubungkan klien tablet juri, wasit, atau scoreboard ke real-time engine:

- **URL WebSocket**:  
  `ws://localhost:8080/ws/{arena_id}?token={JWT_TOKEN}`  
  *(Atau tanpa token untuk mode penonton/scoreboard publik: `ws://localhost:8080/ws/{arena_id}`)*

### Contoh Koneksi JavaScript/TypeScript:
```typescript
const arenaId = "c7397556-eb39-406f-aa89-6b3053a2fd08";
const token = "YOUR_JWT_TOKEN"; // Didapatkan dari respon login PIN di atas

const socket = new WebSocket(`ws://localhost:8080/ws/${arenaId}?token=${token}`);

socket.onopen = () => {
  console.log("Terhubung ke WebSocket Gelanggang A!");
  // Kirim sinkronisasi awal status pertandingan
  socket.send(JSON.stringify({ type: "sync_request" }));
};

socket.onmessage = (event) => {
  const msg = JSON.parse(event.data);
  console.log("Pesan diterima:", msg.type, msg.payload);
};
```
