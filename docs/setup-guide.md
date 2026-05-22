# Setup Guide — Uang Kas Keluarga + Digiflazz

Panduan ini menjelaskan setup backend dari nol sampai siap dipakai FE untuk fitur Digiflazz integration.

## Prasyarat

- Go sesuai `go.mod`: `1.26.1`.
- Akses internet untuk `go mod download` dan API Digiflazz.
- Akun buyer Digiflazz dengan username dan API key.
- IP server sudah di-whitelist pada dashboard Digiflazz.
- Untuk generate proxy: install `pocketbase-gogen` jika schema berubah.

## 1. Install dependencies

```bash
make install
```

Target ini menjalankan:
```bash
go mod download
go mod tidy
```

## 2. Buat file environment

Project belum menyediakan `.env.example`, jadi buat `.env` manual di root project jika deployment/runtime membacanya.

```bash
PB_PORT=8090
PB_HOST=0.0.0.0

# Initial migration superuser; ganti untuk production
ADMIN_EMAIL=admin@example.com
ADMIN_PASSWORD=change-me-strong-password

# Wajib untuk fitur Digiflazz credential/order
DIGIFLAZZ_CREDENTIAL_ENCRYPTION_KEY=change-me-long-random-secret

# Cron format PocketBase; default jika kosong:
DIGIFLAZZ_PRICE_SYNC_INTERVAL=*/30 * * * *
DIGIFLAZZ_ORDER_POLL_INTERVAL=*/5 * * * *
```

Catatan:
- `DIGIFLAZZ_CREDENTIAL_ENCRYPTION_KEY` wajib non-empty karena API key Digiflazz disimpan terenkripsi.
- Jangan mengganti encryption key setelah credential dibuat kecuali credential akan dibuat ulang; key berbeda membuat API key lama tidak bisa didekripsi.
- `ADMIN_EMAIL` dan `ADMIN_PASSWORD` dipakai migration initial superuser. Jangan pakai default di production.

## 3. Jalankan migrations

```bash
go run . migrate up
```

Migrasi yang relevan untuk Digiflazz sudah ada:
- `1778832234_digiflazz_collections.go` — membuat collections Digiflazz.
- `1780000001_add_digiflazz_inquiry_order_status.go` — menambah status inquiry order.
- `1780000002_add_digiflazz_webhook_fields.go` — menambah field webhook/token.

Mode development (`go run`) mengaktifkan automigrate untuk perubahan dari PocketBase Admin UI. Production binary tidak memakai automigrate; jalankan migration command eksplisit.

## 4. Generate proxies jika schema berubah

Jika hanya menjalankan project existing, proxies sudah committed dan siap pakai. Jika mengubah schema PocketBase, jalankan:

```bash
make generate
```

Generated proxy Digiflazz yang harus ada:
- `DigiflazzCredentials`
- `DigiflazzProducts`
- `DigiflazzOrders`
- `DigiflazzEvents`

Jangan edit file di `generated/` secara manual.

## 5. Build dan run

```bash
make build
make serve
```

Alternatif development auto-reload:
```bash
make dev
```

PocketBase Admin UI:
```text
http://localhost:8090/_/
```

## 6. Verifikasi dasar

```bash
go mod tidy && git diff --exit-code
go vet ./...
make test
make build
```

Status saat dokumentasi ini dibuat:
- `make build` ✅ berhasil.
- `make test` ⚠️ timeout setelah 120 detik pada environment penulis; jalankan ulang dengan timeout lebih panjang sebelum release.

## 7. Setup data awal untuk FE

1. Buat user via PocketBase auth API atau Admin UI.
2. Login dari FE dan simpan token user.
3. Buat family atau join family melalui endpoint family existing.
4. Pastikan role owner untuk user yang akan setup Digiflazz.
5. Owner membuat credential Digiflazz:
   ```http
   POST /api/digiflazz/credential
   Authorization: Bearer <user-token>
   Content-Type: application/json
   ```
   ```json
   {
     "username": "digiflazz_username",
     "api_key": "digiflazz_api_key",
     "webhook_secret": "optional-shared-secret",
     "testing": true
   }
   ```
6. Owner rotate webhook token:
   ```http
   POST /api/digiflazz/credential/rotate
   Authorization: Bearer <user-token>
   ```
7. Daftarkan URL webhook di dashboard Digiflazz:
   ```text
   https://<backend-domain>/webhooks/digiflazz/<token>
   ```
8. Owner sync product catalog:
   ```http
   POST /api/digiflazz/products/sync
   Authorization: Bearer <user-token>
   ```
9. Member/owner browse product dan mulai membuat order.

## 8. Middleware dan auth summary

Semua endpoint `/api/digiflazz/*` dibinding dalam urutan:

```text
RequireAuth → RequireFamily → Handler → Service authorization
```

Konsekuensi:
- Request tanpa bearer token mendapat `401 Authentication required`.
- User tanpa family mendapat `403 User is not a member of any family`.
- User member biasa yang mengakses owner-only action mendapat `403 Access denied`.
- Handler mengambil `familyID` dari context, bukan dari body FE.

Webhook Digiflazz **tidak** memakai bearer token:

```text
POST /webhooks/digiflazz/{token}
```

Backend mencocokkan hash token ke credential, lalu memvalidasi signature HMAC-SHA1 `X-Digiflazz-Signature` jika webhook secret dikonfigurasi.

## 9. Operational notes

- Cron `digiflazz-price-sync` berjalan default setiap 30 menit untuk active credential.
- Cron `digiflazz-order-poll` berjalan default setiap 5 menit untuk pending/processing order 24 jam terakhir.
- Order sukses otomatis membuat expense transaction dengan kategori keluarga yang sesuai; jika kategori tidak ada, finalization bisa gagal.
- Notifikasi realtime belum tersedia; FE harus polling order atau menambahkan integrasi PocketBase realtime pada fase berikutnya.
- Jangan commit `pb_data/`, `.env`, atau credential rahasia.

Untuk detail flow FE dan tabel endpoint, lihat [`docs/frontend-flow.md`](./frontend-flow.md).
