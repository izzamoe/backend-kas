# Digiflazz Integration — Frontend Flow & API

Dokumen ini adalah source of truth untuk FE ketika mengintegrasikan fitur Digiflazz pada backend `kas_be`. Endpoint di bawah mengikuti implementasi saat ini di `internal/handler/digiflazz_*_handler.go` dan wiring di `main.go`.

## 1. Apakah project sudah siap pakai?

Jawaban singkat: **backend sudah siap secara struktur dan fitur utama**, tetapi environment target tetap harus menyiapkan credential Digiflazz, encryption key, migrasi DB, dan menjalankan verifikasi.

| Komponen | Status | Catatan |
|---|---:|---|
| Clean architecture | ✅ | Flow `Handler → Service → Repository → generated proxies → PocketBase` sudah terpasang. |
| Route registration | ✅ | `main.go` mendaftarkan credential, product, order, webhook handler di `OnServe()`. |
| Middleware binding | ✅ | Endpoint `/api/digiflazz/*` memakai `RequireAuth` lalu `RequireFamily`. Urutan sudah benar. |
| Owner/member authorization | ✅ | Owner-only dicek di service untuk credential, balance, deposit, token rotation, dan product sync. Order/search dapat diakses member keluarga. |
| Database migrations | ✅ | Migrasi Digiflazz tersedia: `1778832234_digiflazz_collections.go`, `1780000001_add_digiflazz_inquiry_order_status.go`, `1780000002_add_digiflazz_webhook_fields.go`. |
| Generated proxies | ✅ | `generated/proxies.go` berisi `DigiflazzCredentials`, `DigiflazzProducts`, `DigiflazzOrders`, `DigiflazzEvents`. |
| Product catalog cache | ✅ | Sync menyimpan price list prepaid dan pascabayar ke collection `digiflazz_products`. |
| Credential encryption | ✅ | API key dienkripsi memakai `DIGIFLAZZ_CREDENTIAL_ENCRYPTION_KEY`. Jangan jalankan fitur credential/order tanpa env ini. |
| Webhook handling | ✅ | Endpoint tokenized `/webhooks/digiflazz/{token}` validasi `X-Digiflazz-Signature`, simpan event, update status order. |
| Auto expense transaction | ✅ | Order sukses otomatis dibuatkan expense transaction jika kategori keluarga tersedia. |
| Notification | ❌ | Belum ada fitur notifikasi FE/realtime khusus Digiflazz. FE perlu polling order atau pakai PocketBase realtime jika ditambahkan nanti. |
| `.env.example` | ❌ | Tidak ditemukan file `.env.example`; buat `.env` manual. |
| Build/test status saat dok dibuat | ⚠️ | `make build` berhasil. `make test` sempat dijalankan tetapi timeout setelah 120 detik; ulangi di CI/local dengan timeout lebih panjang sebelum rilis. |

## 2. Role dan akses user

| Role | Akses Digiflazz |
|---|---|
| Family owner | Mengelola credential, cek balance, deposit, rotate webhook token, sync product catalog, browse product, create/pay/check order. |
| Family member | Browse product, create prepaid order, create postpaid inquiry, pay inquiry, lihat/check order keluarga. Tidak bisa mengelola credential/sync/deposit. |
| System/Digiflazz | Mengirim webhook ke `/webhooks/digiflazz/{token}` tanpa user auth, tetapi wajib token valid dan signature valid jika secret diset. |

Catatan middleware:
- Semua endpoint `/api/digiflazz/*` membutuhkan `Authorization: Bearer <token>`.
- Semua endpoint `/api/digiflazz/*` membutuhkan user sudah punya family membership.
- `RequireFamily` memasukkan `familyID` ke request context; FE tidak perlu mengirim `family_id` untuk endpoint Digiflazz. Jika body mengirim `family_id`, backend tetap memakai context family.
- Role owner tidak dibinding sebagai middleware khusus pada route Digiflazz; enforcement dilakukan di service via `middleware.IsFamilyOwner`.

## 3. Frontend user flow

### 3.1 Owner: Setup Digiflazz Credential

```text
Login → Buka Family Settings → Klik "Connect Digiflazz"
→ Masukkan Username, API Key, Webhook Secret opsional, Testing mode opsional
→ Save → Backend validasi ke Digiflazz cek saldo
→ Backend encrypt API key & simpan metadata credential
→ FE tampilkan username, last4 API key, active/testing status, webhook configured
```

Endpoint utama:
- `POST /api/digiflazz/credential`
- `GET /api/digiflazz/credential`
- `PATCH /api/digiflazz/credential`
- `DELETE /api/digiflazz/credential`

Contoh body create:
```json
{
  "username": "digiflazz_username",
  "api_key": "digiflazz_api_key",
  "webhook_secret": "optional-shared-secret",
  "testing": true
}
```

FE handling:
- Jika `403`, tampilkan pesan hanya owner yang boleh mengelola credential.
- Jika `400` validasi gagal, tampilkan error dari backend; biasanya username/API key invalid atau credential sudah ada.
- API key tidak pernah dikembalikan utuh; hanya `api_key_last4`.

### 3.2 Owner: Setup Webhook URL

```text
Login sebagai owner → Buka Credential Detail → Klik "Rotate Webhook Token"
→ Backend mengembalikan token satu kali
→ FE tampilkan/copy URL webhook: https://<domain>/webhooks/digiflazz/<token>
→ Owner paste URL dan secret di dashboard Digiflazz
```

Endpoint:
- `POST /api/digiflazz/credential/rotate`

Response mengandung `token`; token hanya aman ditampilkan sekali. Backend menyimpan hash token, bukan raw token.

### 3.3 Owner: Sync Product Catalog

```text
Login → Buka Digiflazz Products → Klik "Sync Products"
→ Backend ambil prepaid + pascabayar dari Digiflazz memakai active credential
→ Backend upsert ke `digiflazz_products`
→ FE tampilkan jumlah produk tersync dan error parsial jika ada
```

Endpoint:
- `POST /api/digiflazz/products/sync`

Contoh response:
```json
{
  "prepaid_upserted": 100,
  "postpaid_upserted": 20,
  "total_upserted": 120,
  "errors": []
}
```

FE handling:
- Disable tombol sync saat request berjalan.
- Jika `403`, user bukan owner.
- Jika active credential belum ada, minta owner setup credential dulu.
- Product sync juga berjalan via cron `DIGIFLAZZ_PRICE_SYNC_INTERVAL` untuk semua active credential.

### 3.4 Member: Browse Products

```text
Login → Buka Digiflazz Products → Search/filter category/brand/type/status
→ Backend baca dari cache DB lokal
→ FE tampilkan harga, admin, status produk, stock/cutoff
```

Endpoint:
- `GET /api/digiflazz/products?query=&category=&brand=&type=&status=&per_page=`

Query params:
| Param | Deskripsi |
|---|---|
| `query` | Free-text search ke product name / SKU. |
| `category` | Filter kategori Digiflazz. |
| `brand` | Filter brand/provider. |
| `type` | Filter tipe; nilai `postpaid` dipakai untuk produk pascabayar. |
| `status` | Filter status produk. |
| `per_page` | Limit hasil; default 50, maksimum 200. |

Response:
```json
{
  "items": [
    {
      "code": "S10",
      "name": "Pulsa 10.000",
      "category": "Pulsa",
      "brand": "SMARTFREN",
      "type": "Umum",
      "price": 10500,
      "admin": 0,
      "status": "active"
    }
  ],
  "limit": 50
}
```

### 3.5 Member: Create Prepaid Order

```text
Login → Pilih prepaid product → Masukkan customer number
→ Klik "Buy" → Backend cek family member, active credential, product status/cutoff, saldo Digiflazz
→ Backend create order pending → Call topup Digiflazz
→ Backend update status processing/success/failed
→ Jika sukses, backend create expense transaction
→ FE tampilkan status dan detail SN/message
```

Endpoint:
- `POST /api/digiflazz/orders`

Body minimal:
```json
{
  "order_type": "prepaid",
  "product_code": "S10",
  "customer_no": "081234567890",
  "note": "Pulsa anak"
}
```

Body optional:
- `ref_id`: idempotency key dari FE. Jika kosong, backend generate `DFZ-...`.
- `amount`: override maksimum harga; normalnya FE tidak perlu kirim karena backend pakai harga cache.

FE handling:
- Tampilkan error jika product inactive/cutoff atau saldo Digiflazz kurang.
- Untuk status `processing`, FE dapat polling `GET /api/digiflazz/orders/{id}` atau `POST /api/digiflazz/orders/{id}/check-status`.

### 3.6 Member: Create Postpaid Inquiry dan Pay

```text
Login → Pilih postpaid product → Masukkan customer number
→ Klik "Check Bill" → Backend inquiry ke Digiflazz
→ FE tampilkan nama pelanggan, periode, admin, tagihan/total
→ User klik "Pay" → Backend bayar tagihan
→ Backend update status processing/success/failed
→ Jika sukses, backend create expense transaction
```

Endpoint inquiry:
- `POST /api/digiflazz/orders`

Body inquiry minimal:
```json
{
  "order_type": "postpaid",
  "product_code": "PLNPOSTPAID",
  "customer_no": "5353xxxxxxxx"
}
```

Body optional inquiry:
- `year`: untuk produk yang membutuhkan tahun.
- `amount`: untuk produk tertentu yang butuh amount saat inquiry.
- `customer_id2`: untuk produk yang butuh ID pelanggan kedua.
- `customer_name`: fallback nama pelanggan jika inquiry PLN tidak mengembalikan nama.
- `ref_id`: idempotency key.

Endpoint pay:
- `POST /api/digiflazz/orders/{id}/pay`

FE handling:
- Pay hanya valid untuk order status `inquiry`.
- Jika amount berubah antara inquiry dan pay, backend menolak dan FE harus meminta user membuat inquiry baru.
- Untuk status `processing`, gunakan status check/polling.

### 3.7 Member: Order List, Detail, dan Status Check

```text
Login → Buka Riwayat Digiflazz
→ FE load list order keluarga
→ User buka detail atau klik refresh status
```

Endpoint:
- `GET /api/digiflazz/orders?page=&page_size=`
- `GET /api/digiflazz/orders/{id}`
- `POST /api/digiflazz/orders/{id}/check-status`

Pagination:
- `page` default 1.
- `page_size` default 20, maksimum 100.

### 3.8 System: Webhook Handling

```text
Digiflazz kirim webhook ke /webhooks/digiflazz/{token}
→ Backend hash token dan cari credential keluarga
→ Backend validasi X-Digiflazz-Signature dengan webhook secret
→ Backend cari order via ref_id
→ Backend simpan event idempoten berdasar payload hash
→ Backend update status order
→ Jika status success, backend create expense transaction
→ FE belum menerima notifikasi khusus; gunakan polling/realtime tambahan
```

Endpoint webhook:
- `POST /webhooks/digiflazz/{token}`

Header penting:
- `X-Digiflazz-Signature: sha1=<hmac_sha1_hex>` jika webhook secret diisi.
- `X-Digiflazz-Event` opsional; backend menyimpan sebagai source event.

## 4. API endpoints dan auth requirements

| Method | Endpoint | Auth | Family | Role | Description |
|---|---|---|---|---|---|
| GET | `/api/digiflazz/credential` | Bearer user token | Wajib | Owner | Ambil metadata credential keluarga; API key tidak dikembalikan utuh. |
| POST | `/api/digiflazz/credential` | Bearer user token | Wajib | Owner | Buat credential baru; validasi ke Digiflazz lalu encrypt API key. |
| PATCH | `/api/digiflazz/credential` | Bearer user token | Wajib | Owner | Update username/API key/webhook secret/testing/is_active. |
| DELETE | `/api/digiflazz/credential` | Bearer user token | Wajib | Owner | Hapus credential keluarga. |
| POST | `/api/digiflazz/credential/rotate` | Bearer user token | Wajib | Owner | Generate token webhook baru; raw token dikembalikan sekali. |
| GET | `/api/digiflazz/credential/balance` | Bearer user token | Wajib | Owner | Cek saldo deposit Digiflazz dari active credential. |
| POST | `/api/digiflazz/deposit` | Bearer user token | Wajib | Owner | Buat request deposit Digiflazz. |
| GET | `/api/digiflazz/products` | Bearer user token | Wajib | Owner/member | Search/list product cache lokal. |
| POST | `/api/digiflazz/products/sync` | Bearer user token | Wajib | Owner | Manual sync product prepaid + pascabayar dari Digiflazz. |
| GET | `/api/digiflazz/orders` | Bearer user token | Wajib | Owner/member | List order Digiflazz milik keluarga. |
| GET | `/api/digiflazz/orders/{id}` | Bearer user token | Wajib | Owner/member | Detail order dalam keluarga user. |
| POST | `/api/digiflazz/orders` | Bearer user token | Wajib | Owner/member | Buat prepaid order atau postpaid inquiry; ditentukan dari `order_type` / `event_type`. |
| POST | `/api/digiflazz/orders/{id}/pay` | Bearer user token | Wajib | Owner/member | Bayar order pascabayar yang masih status `inquiry`. |
| POST | `/api/digiflazz/orders/{id}/check-status` | Bearer user token | Wajib | Owner/member | Check status order pending/processing ke Digiflazz. |
| POST | `/webhooks/digiflazz/{token}` | Token URL + signature | Dari credential token | System | Terima webhook Digiflazz, update order/event/transaction. |

> Catatan penting: README lama masih menuliskan beberapa endpoint plural seperti `/api/digiflazz/credentials` dan endpoint khusus `/prepaid`/`/inquiry`. Implementasi saat ini memakai endpoint singular `/api/digiflazz/credential` dan order generic `/api/digiflazz/orders`.

## 5. Status order yang perlu ditangani FE

| Status | Arti FE |
|---|---|
| `inquiry` | Inquiry pascabayar berhasil dan menunggu user klik Pay. |
| `pending` | Order tercatat dan menunggu diproses. |
| `processing` | Digiflazz sedang memproses; FE boleh polling/check status. |
| `success` | Order berhasil; backend akan link/create expense transaction. |
| `failed` | Order gagal; tampilkan message/RC dari backend. |
| `cancelled` | Order dibatalkan secara sistem/manual jika ada fitur lanjutan. |

## 6. FE checklist sebelum release

- [ ] Login PocketBase user dan simpan bearer token.
- [ ] Pastikan user sudah create/join family sebelum akses Digiflazz.
- [ ] Hide owner-only menu untuk member biasa: credential, balance, deposit, sync products, rotate webhook token.
- [ ] Form credential tidak pernah menyimpan API key di localStorage/sessionStorage.
- [ ] Gunakan `GET /api/digiflazz/credential` untuk menampilkan status connection.
- [ ] Tampilkan empty state product jika belum sync; arahkan owner untuk sync.
- [ ] Tampilkan status order dengan polling untuk `pending`/`processing`.
- [ ] Tampilkan fallback bahwa notifikasi realtime belum tersedia.
- [ ] Untuk webhook URL, tampilkan token sekali dan minta owner menyimpannya di dashboard Digiflazz.

Lihat juga: [`docs/setup-guide.md`](./setup-guide.md) untuk setup environment dan langkah menjalankan backend.
