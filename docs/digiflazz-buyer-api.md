# Digiflazz API — Dokumentasi Buyer (API Transaksi)

> Sumber: https://developer.digiflazz.com/api/
> Diperbarui: 2026-05-13

---

## Daftar Isi

1. [Overview](#overview)
2. [Persiapan](#persiapan)
3. [Cek Saldo](#cek-saldo)
4. [Daftar Harga](#daftar-harga)
5. [Deposit](#deposit)
6. [Topup (Prabayar)](#topup-prabayar)
7. [Cek Tagihan (Pascabayar)](#cek-tagihan-pascabayar)
8. [Bayar Tagihan (Pascabayar)](#bayar-tagihan-pascabayar)
9. [Cek Status](#cek-status)
10. [Inquiry PLN](#inquiry-pln)
11. [Test Case](#test-case)
12. [Response Code](#response-code)
13. [Webhooks](#webhooks)

---

## Overview

- Semua HTTP Request dibungkus dalam format **JSON**
- Seluruh HTTP Request Method dikirim sebagai **POST** Request
- Whitelist IP Digiflazz: `52.74.250.133`

---

## Persiapan

- Kunjungi [Pengaturan Koneksi API](https://member.digiflazz.com/buyer-area/connection/api)
- Dapatkan **username** dan **API Key** di halaman tersebut
- Tambahkan Whitelist IP untuk development dan production
- Set `Content-Type: application/json` pada header request
- Seluruh transaksi via API menggunakan method `POST`

---

## Cek Saldo

Cek sisa deposit yang dimiliki.

### Endpoint

```
POST https://api.digiflazz.com/v1/cek-saldo
```

### Request

```json
{
  "cmd": "deposit",
  "username": "username",
  "sign": "740b00a1b8784e028cc8078edf66d12b"
}
```

| Parameter | Deskripsi | Tipe | Wajib |
|-----------|-----------|------|-------|
| `cmd` | Value: `"deposit"` | String | Ya |
| `username` | Username dari pengaturan koneksi API | String | Ya |
| `sign` | Signature: `md5(username + apiKey + "depo")` | String | Ya |

### Response

```json
{
  "data": {
    "deposit": 500000000000
  }
}
```

| Parameter | Deskripsi | Tipe | Wajib |
|-----------|-----------|------|-------|
| `deposit` | Sisa deposit Anda | Float | Ya |

> **Perhatian:** Response JSON dibungkus oleh variable `data`.

---

## Daftar Harga

### Endpoint

```
POST https://api.digiflazz.com/v1/price-list
```

> **Perhatian:** Ada limitasi pengecekan daftar harga. Disarankan simpan di database sendiri dan update berkala. Data per category/brand/type tidak real time (delay 10–15 menit).

---

### Price List Prabayar (Prepaid)

#### Request

```json
{
  "cmd": "prepaid",
  "username": "username",
  "sign": "740b00a1b8784e028cc8078edf66d12b"
}
```

| Parameter | Deskripsi | Tipe | Wajib |
|-----------|-----------|------|-------|
| `cmd` | `prepaid` atau `pasca` | String | Ya |
| `username` | Username dari pengaturan koneksi API | String | Ya |
| `code` | Kode produk Buyer | String | Tidak |
| `category` | Kategori produk | String | Tidak |
| `brand` | Merek produk | String | Tidak |
| `type` | Tipe produk | String | Tidak |
| `sign` | Signature: `md5(username + apiKey + "pricelist")` | String | Ya |

#### Response

```json
{
  "data": [
    {
      "product_name": "Xl 100.000",
      "category": "Pulsa",
      "brand": "XL",
      "type": "Umum",
      "seller_name": "PT. ABC",
      "price": 98000,
      "buyer_sku_code": "X100",
      "buyer_product_status": true,
      "seller_product_status": true,
      "unlimited_stock": true,
      "stock": 0,
      "multi": true,
      "start_cut_off": "23:45",
      "end_cut_off": "00:15",
      "desc": "Pulsa Xl Rp 100.000"
    }
  ]
}
```

| Parameter | Deskripsi | Tipe | Wajib |
|-----------|-----------|------|-------|
| `product_name` | Nama produk | String | Ya |
| `category` | Nama kategori | String | Ya |
| `brand` | Nama brand | String | Ya |
| `type` | Nama tipe | String | Ya |
| `seller_name` | Nama seller | String | Ya |
| `price` | Harga produk | String | Ya |
| `buyer_sku_code` | Kode produk Buyer | String | Ya |
| `buyer_product_status` | Status produk sebagai buyer | Boolean | Ya |
| `seller_product_status` | Status produk seller | Boolean | Ya |
| `unlimited_stock` | Apakah stok tidak terbatas | Boolean | Ya |
| `stock` | Sisa stok seller (abaikan jika `unlimited_stock=true`) | String | Ya |
| `multi` | Bisa transaksi lebih dari 1x ke denom & nomor tujuan yang sama dalam sehari | Bool | Ya |
| `start_cut_off` | Jam mulai cut off (format: `hh:mm`) | String | Ya |
| `end_cut_off` | Jam akhir cut off (format: `hh:mm`) | String | Ya |
| `desc` | Deskripsi produk | String | Ya |

---

### Price List Pascabayar

#### Request

```json
{
  "cmd": "pasca",
  "username": "username",
  "sign": "740b00a1b8784e028cc8078edf66d12b"
}
```

| Parameter | Deskripsi | Tipe | Wajib |
|-----------|-----------|------|-------|
| `cmd` | `prepaid` atau `pasca` | String | Ya |
| `username` | Username dari pengaturan koneksi API | String | Ya |
| `code` | Kode produk Buyer | String | Tidak |
| `brand` | Merek produk | String | Tidak |
| `sign` | Signature: `md5(username + apiKey + "pricelist")` | String | Ya |

#### Response

```json
{
  "data": [
    {
      "product_name": "Pln Postpaid",
      "category": "Pascabayar",
      "brand": "PLN",
      "seller_name": "PT. ABC",
      "admin": 2750,
      "commission": 1800,
      "buyer_sku_code": "pln",
      "buyer_product_status": true,
      "seller_product_status": true,
      "desc": "-"
    }
  ]
}
```

| Parameter | Deskripsi | Tipe | Wajib |
|-----------|-----------|------|-------|
| `product_name` | Nama produk | String | Ya |
| `category` | Nama kategori | String | Ya |
| `brand` | Nama brand | String | Ya |
| `seller_name` | Nama seller | String | Ya |
| `admin` | Biaya admin | Int | Ya |
| `commission` | Komisi yang didapatkan Buyer | Int | Ya |
| `buyer_sku_code` | Kode produk Buyer | String | Ya |
| `buyer_product_status` | Status produk sebagai buyer | Boolean | Ya |
| `seller_product_status` | Status produk seller | Boolean | Ya |
| `desc` | Deskripsi produk | String | Ya |

---

## Deposit

Fitur untuk melakukan penarikan tiket deposit.

### Endpoint

```
POST https://api.digiflazz.com/v1/deposit
```

### Request

```json
{
  "username": "your-username",
  "amount": 10000000,
  "Bank": "BCA",
  "owner_name": "John Doe",
  "sign": "740b00a1b8784e028cc8078edf66d12b"
}
```

| Parameter | Deskripsi | Tipe | Wajib |
|-----------|-----------|------|-------|
| `username` | Username dari pengaturan koneksi API | String | Ya |
| `amount` | Jumlah deposit yang diinginkan | Int | Ya |
| `bank` | Nama bank tujuan transfer. **Perorangan**: `Flip` / `ShopeePay`. **Perusahaan**: `BCA` / `MANDIRI` / `BRI` / `BNI` | String | Ya |
| `owner_name` | Nama pemilik rekening yang melakukan transfer | String | Ya |
| `sign` | Signature: `md5(username + apiKey + "deposit")` | String | Ya |

### Response

```json
{
  "data": {
    "rc": "00",
    "bank": "BCA",
    "payment_method": "Bank Transfer",
    "account_no": "0123 4567 89",
    "notes": "A6R5UPV",
    "amount": 10000001
  }
}
```

| Parameter | Deskripsi | Tipe | Wajib |
|-----------|-----------|------|-------|
| `rc` | Response Code | String | Ya |
| `bank` | Bank tujuan | String | Ya |
| `payment_method` | Metode pembayaran: `"Bank Transfer"` atau `"Virtual Account"` | String | Ya |
| `account_no` | Nomor rekening bank tujuan | String | Ya |
| `amount` | Jumlah akhir yang harus ditransfer | Int | Ya |
| `notes` | Berita yang harus dimasukkan saat transfer | String | Ya |

---

## Topup (Prabayar)

Seluruh transaksi diproses secara **sinkron** — request langsung mendapat respons sukses/gagal/pending.

> **Cek Status:** Respons pending dapat dicek ulang dengan melakukan topup menggunakan `ref_id` yang sama.

### Endpoint

```
POST https://api.digiflazz.com/v1/transaction
```

### Request

| Parameter | Deskripsi | Tipe | Wajib |
|-----------|-----------|------|-------|
| `username` | Username dari pengaturan koneksi API | String | Ya |
| `buyer_sku_code` | Kode produk Anda | String | Ya |
| `customer_no` | Nomor Pelanggan | String | Ya |
| `ref_id` | Ref ID unik Anda | String | Ya |
| `sign` | Signature: `md5(username + apiKey + ref_id)` | String | Ya |
| `testing` | `true` untuk mode development | Boolean | Tidak |
| `max_price` | Limit harga maksimum | Int | Tidak |
| `cb_url` | Callback URL (jika punya lebih dari 1 webhook) | String | Tidak |
| `allow_dot` | `true` jika `customer_no` boleh berisi titik | Boolean | Tidak |

```json
{
  "username": "username",
  "buyer_sku_code": "xld25",
  "customer_no": "087800001233",
  "ref_id": "some1d",
  "sign": "740b00a1b8784e028cc8078edf66d12b"
}
```

### Response

| Parameter | Deskripsi | Tipe | Wajib |
|-----------|-----------|------|-------|
| `ref_id` | Ref ID unik Anda | String | Ya |
| `customer_no` | Nomor pelanggan | String | Ya |
| `buyer_sku_code` | Kode produk Anda | String | Ya |
| `message` | Deskripsi status transaksi | String | Ya |
| `status` | Status: `Sukses`, `Pending`, atau `Gagal` | String | Ya |
| `rc` | Response Code | String | Ya |
| `sn` | Serial Number | String | Tidak |
| `buyer_last_saldo` | Saldo terakhir setelah transaksi | Float | Tidak |
| `price` | Harga produk | Integer | Ya |
| `tele` | Telegram Seller | String | Tidak |
| `wa` | WhatsApp Seller | String | Tidak |

```json
{
  "data": {
    "ref_id": "some1d",
    "customer_no": "087800001233",
    "buyer_sku_code": "xld25",
    "message": "Transaksi Pending",
    "status": "Pending",
    "rc": "03",
    "sn": "",
    "buyer_last_saldo": 100000,
    "price": 25000,
    "tele": "@telegram",
    "wa": "081234512345"
  }
}
```

> **Perhatian:** Response JSON dibungkus oleh variable `data`.

---

## Cek Tagihan (Pascabayar)

Seluruh transaksi diproses secara **sinkron**.

### Endpoint

```
POST https://api.digiflazz.com/v1/transaction
```

### Request Standar

Produk dengan format tambahan request: **PBB**, **E-Money**, **SAMSAT** — lihat sub-seksi masing-masing.

| Parameter | Deskripsi | Tipe | Wajib |
|-----------|-----------|------|-------|
| `commands` | `"inq-pasca"` | String | Ya |
| `username` | Username dari pengaturan koneksi API | String | Ya |
| `buyer_sku_code` | Kode produk Anda | String | Ya |
| `customer_no` | Nomor Pelanggan | String | Ya |
| `ref_id` | Ref ID unik Anda | String | Ya |
| `sign` | Signature: `md5(username + apiKey + ref_id)` | String | Ya |
| `testing` | `true` untuk mode development | Boolean | Tidak |

```json
{
  "commands": "inq-pasca",
  "username": "username",
  "buyer_sku_code": "pln",
  "customer_no": "530000000003",
  "ref_id": "some1d",
  "sign": "740b00a1b8784e028cc8078edf66d12b"
}
```

---

### Response PLN

```json
{
  "data": {
    "ref_id": "some1d",
    "customer_no": "530000000001",
    "customer_name": "Nama Pelanggan Pertama",
    "buyer_sku_code": "i5",
    "admin": 2500,
    "message": "Transaksi Sukses",
    "status": "Sukses",
    "rc": "00",
    "periode": "201901",
    "buyer_last_saldo": 100000,
    "price": 10000,
    "selling_price": 11000,
    "desc": {
      "tarif": "R1",
      "daya": 1300,
      "lembar_tagihan": 1,
      "detail": [
        {
          "periode": "201901",
          "nilai_tagihan": "8000",
          "admin": "2500",
          "denda": "500"
        }
      ]
    }
  }
}
```

| Parameter | Deskripsi | Tipe | Wajib |
|-----------|-----------|------|-------|
| `ref_id` | Ref ID unik | String | Ya |
| `customer_no` | Nomor pelanggan | String | Ya |
| `customer_name` | Nama pelanggan | String | Ya |
| `buyer_sku_code` | Kode produk | String | Ya |
| `admin` | Total biaya admin | Int | Ya |
| `message` | Deskripsi status | String | Ya |
| `status` | `Sukses` atau `Gagal` | String | Ya |
| `rc` | Response Code | String | Ya |
| `periode` | Periode tagihan | String | Tidak |
| `buyer_last_saldo` | Saldo terakhir | Float | Ya |
| `price` | Harga dipotong dari deposit | Int | Ya |
| `selling_price` | Harga dipotong dari client | Int | Ya |
| `desc.tarif` | Tarif PLN | String | Tidak |
| `desc.daya` | Daya PLN | Int | Tidak |
| `desc.lembar_tagihan` | Jumlah lembar tagihan | Int | Tidak |
| `desc.detail[].periode` | Periode per tagihan | String | Tidak |
| `desc.detail[].nilai_tagihan` | Nilai tagihan | String | Tidak |
| `desc.detail[].admin` | Biaya admin per tagihan | String | Tidak |
| `desc.detail[].denda` | Biaya denda per tagihan | String | Tidak |

---

### Response PDAM

```json
{
  "data": {
    "ref_id": "353688162",
    "customer_no": "1013226",
    "customer_name": "Nama Pelanggan Pertama",
    "buyer_sku_code": "pdam",
    "admin": 2500,
    "message": "Transaksi Sukses",
    "status": "Sukses",
    "rc": "00",
    "periode": "201901",
    "buyer_last_saldo": 100000,
    "price": 11500,
    "selling_price": 12500,
    "desc": {
      "tarif": "3A",
      "lembar_tagihan": 1,
      "alamat": "WONOKROMO S.S BARU 2 8",
      "jatuh_tempo": "1-15 DES 2014",
      "detail": [
        {
          "periode": "201901",
          "nilai_tagihan": "8000",
          "denda": "500",
          "meter_awal": "00080000",
          "meter_akhir": "00090000",
          "biaya_lain": "1500"
        }
      ]
    }
  }
}
```

| Parameter | Deskripsi | Tipe | Wajib |
|-----------|-----------|------|-------|
| `desc.tarif` | Tarif PDAM | String | Tidak |
| `desc.lembar_tagihan` | Jumlah lembar tagihan | Int | Tidak |
| `desc.alamat` | Alamat tagihan | String | Tidak |
| `desc.jatuh_tempo` | Tanggal jatuh tempo | String | Tidak |
| `desc.detail[].nilai_tagihan` | Nilai tagihan per periode | String | Tidak |
| `desc.detail[].denda` | Biaya denda | String | Tidak |
| `desc.detail[].meter_awal` | Meter awal | String | Tidak |
| `desc.detail[].meter_akhir` | Meter akhir | String | Tidak |
| `desc.detail[].biaya_lain` | Biaya lainnya | String | Tidak |

---

### Response INTERNET

```json
{
  "data": {
    "ref_id": "4536881875",
    "customer_no": "6391601001",
    "customer_name": "Nama Pelanggan",
    "buyer_sku_code": "internet",
    "admin": 5000,
    "message": "Transaksi Sukses",
    "status": "Sukses",
    "rc": "00",
    "periode": "MEI 2019,JUN 2019",
    "buyer_last_saldo": 100000,
    "price": 22500,
    "selling_price": 24500,
    "desc": {
      "lembar_tagihan": 2,
      "detail": [
        { "periode": "MEI 2019", "nilai_tagihan": "8000", "admin": "2500" },
        { "periode": "JUN 2019", "nilai_tagihan": "11500", "admin": "2500" }
      ]
    }
  }
}
```

---

### Response BPJS Kesehatan

```json
{
  "data": {
    "ref_id": "4536881875",
    "customer_no": "8801234560001",
    "customer_name": "Nama Pelanggan",
    "buyer_sku_code": "bpjs",
    "admin": 2500,
    "message": "Transaksi Sukses",
    "status": "Sukses",
    "rc": "00",
    "periode": "01",
    "buyer_last_saldo": 100000,
    "price": 24700,
    "selling_price": 25000,
    "desc": {
      "jumlah_peserta": "2",
      "lembar_tagihan": 1,
      "alamat": "JAKARTA PUSAT",
      "detail": [{ "periode": "01" }]
    }
  }
}
```

| Parameter | Deskripsi | Tipe |
|-----------|-----------|------|
| `desc.jumlah_peserta` | Jumlah peserta BPJS | String |
| `desc.alamat` | Alamat peserta | String |

---

### Response Multifinance

```json
{
  "data": {
    "ref_id": "ref-1",
    "customer_no": "6391601201",
    "customer_name": "Nama Pelanggan Pertama",
    "buyer_sku_code": "multifinance",
    "admin": 2500,
    "message": "Transaksi Sukses",
    "status": "Sukses",
    "rc": "00",
    "periode": "002",
    "buyer_last_saldo": 100000,
    "price": 24700,
    "selling_price": 25000,
    "desc": {
      "lembar_tagihan": 1,
      "item_name": "HONDA VARIO TECHNO 125 PGM FI NON CBS",
      "no_rangka": "MH1JFB111CK196426",
      "no_pol": "B6213UWX",
      "tenor": "030",
      "detail": [{ "periode": "002", "denda": "0", "biaya_lain": "0" }]
    }
  }
}
```

---

### PBB — Request Khusus

Format `customer_no` PBB: `Kode Pembayaran, Nomor Identitas`

| Parameter | Deskripsi | Tipe | Wajib |
|-----------|-----------|------|-------|
| `commands` | `"inq-pasca"` | String | Ya |
| `username` | Username | String | Ya |
| `buyer_sku_code` | Kode produk | String | Ya |
| `customer_no` | Format: Kode Pembayaran, Nomor Identitas | String | Ya |
| `ref_id` | Ref ID unik | String | Ya |
| `sign` | `md5(username + apiKey + ref_id)` | String | Ya |
| `year` | Tahun pajak (contoh: `2025`). Default: tahun berjalan | Int | Tidak |
| `testing` | `true` untuk development | Boolean | Tidak |

### Response PBB

```json
{
  "data": {
    "ref_id": "ref-4",
    "customer_no": "329801092375999991",
    "customer_name": "Nama Pelanggan Pertama",
    "buyer_sku_code": "cimahi",
    "admin": 2500,
    "message": "Transaksi Sukses",
    "status": "Sukses",
    "rc": "00",
    "periode": "2019",
    "buyer_last_saldo": 100000,
    "price": 99500,
    "selling_price": 100000,
    "desc": {
      "lembar_tagihan": 1,
      "alamat": "KO. GRIYA ASRI CIPAGERAN",
      "tahun_pajak": "2019",
      "kelurahan": "CIPAGERAN",
      "kecamatan": "CIPAGERAN",
      "kode_kab_kota": "0023",
      "kab_kota": "PEMKOT CIMAHI",
      "luas_tanah": "113 M2",
      "luas_gedung": "47 M2"
    }
  }
}
```

---

### Response Pajak Daerah Lainnya

Sama dengan PBB, ditambah field `desc.provinsi` (String, opsional).

---

### Response GAS NEGARA / PERTAGAS

```json
{
  "data": {
    "ref_id": "ref-9",
    "customer_no": "0110014601",
    "customer_name": "Nama Pelanggan",
    "buyer_sku_code": "pgas",
    "admin": 2500,
    "message": "Transaksi Sukses",
    "status": "Sukses",
    "rc": "00",
    "buyer_last_saldo": 500,
    "price": 99500,
    "selling_price": 100000,
    "desc": {
      "lembar_tagihan": 1,
      "alamat": "KO. GRIYA ASRI CIPAGERAN",
      "detail": [
        {
          "periode": "0320",
          "meter_awal": "006538",
          "meter_akhir": "006573",
          "usage": "35"
        }
      ]
    }
  }
}
```

---

### Response TV

```json
{
  "data": {
    "ref_id": "ref-367",
    "customer_no": "127246500101",
    "customer_name": "BAITUS MONGJENG",
    "buyer_sku_code": "tv",
    "admin": 2500,
    "message": "Transaksi Sukses",
    "status": "Sukses",
    "rc": "00",
    "buyer_last_saldo": 976793000,
    "price": 100500,
    "selling_price": 101500,
    "desc": {
      "lembar_tagihan": 1,
      "detail": [
        { "periode": "MEI 22", "nilai_tagihan": "99000", "no_ref": "205A" }
      ]
    }
  }
}
```

---

### Response BPJSTK

Struktur response mengikuti pola umum (ref_id, customer_no, customer_name, buyer_sku_code, admin, message, status, rc, buyer_last_saldo, price, selling_price, desc).

---

### Response BPJSTKPU (Penerima Upah)

Sama dengan BPJSTK, dengan detail peserta upah.

---

### Response PLN Nontaglis

Sama dengan PLN reguler, dengan detail nontaglis.

---

### E-Money — Request Khusus

| Parameter | Deskripsi | Tipe | Wajib |
|-----------|-----------|------|-------|
| `commands` | `"inq-pasca"` | String | Ya |
| `username` | Username | String | Ya |
| `buyer_sku_code` | Kode produk | String | Ya |
| `customer_no` | Nomor pelanggan | String | Ya |
| `ref_id` | Ref ID unik | String | Ya |
| `sign` | `md5(username + apiKey + ref_id)` | String | Ya |
| `amount` | Nominal top up (kelipatan Rp 1.000) | Int | Ya |

### Response E-Money

Sama dengan pola response umum (ref_id, customer_no, customer_name, buyer_sku_code, admin, message, status, rc, buyer_last_saldo, price, selling_price).

---

### SAMSAT — Request Khusus

| Parameter | Deskripsi | Tipe | Wajib |
|-----------|-----------|------|-------|
| `commands` | `"inq-pasca"` | String | Ya |
| `username` | Username | String | Ya |
| `buyer_sku_code` | Kode produk | String | Ya |
| `customer_no` | Nomor plat kendaraan | String | Ya |
| `ref_id` | Ref ID unik | String | Ya |
| `sign` | `md5(username + apiKey + ref_id)` | String | Ya |
| `id_pelanggan2` | NIK pemilik kendaraan | String | Ya |

### Response SAMSAT

Sama dengan pola response umum, dengan tambahan:

| Field desc | Deskripsi |
|-----------|-----------|
| `nama` | Nama pemilik kendaraan |
| `alamat` | Alamat pemilik |
| `merk` | Merk kendaraan |
| `model` | Model kendaraan |
| `tahun` | Tahun kendaraan |
| `warna` | Warna kendaraan |
| `no_rangka` | Nomor rangka |
| `no_mesin` | Nomor mesin |
| `jatuh_tempo` | Tanggal jatuh tempo pajak |

---

### Response HP / Lainnya

Response umum tanpa `desc` khusus:

```json
{
  "data": {
    "ref_id": "...",
    "customer_no": "...",
    "customer_name": "...",
    "buyer_sku_code": "...",
    "admin": 2500,
    "message": "Transaksi Sukses",
    "status": "Sukses",
    "rc": "00",
    "buyer_last_saldo": 100000,
    "price": 50000,
    "selling_price": 52000,
    "desc": {}
  }
}
```

---

## Bayar Tagihan (Pascabayar)

Diproses secara **sinkron**. Gunakan `ref_id` yang sama dengan saat inquiry.

> **Perhatian:** Pembayaran hanya bisa dilakukan pada **tanggal yang sama** dengan tanggal pengecekan tagihan.

### Endpoint

```
POST https://api.digiflazz.com/v1/transaction
```

### Request

| Parameter | Deskripsi | Tipe | Wajib |
|-----------|-----------|------|-------|
| `commands` | `"pay-pasca"` | String | Ya |
| `username` | Username dari pengaturan koneksi API | String | Ya |
| `buyer_sku_code` | Kode produk Anda | String | Ya |
| `customer_no` | Nomor Pelanggan | String | Ya |
| `ref_id` | Ref ID unik **sama dengan saat inquiry** | String | Ya |
| `sign` | Signature: `md5(username + apiKey + ref_id)` | String | Ya |
| `testing` | `true` untuk mode development | Boolean | Tidak |

```json
{
  "commands": "pay-pasca",
  "username": "username",
  "buyer_sku_code": "pln",
  "customer_no": "530000000003",
  "ref_id": "some1d",
  "sign": "740b00a1b8784e028cc8078edf66d12b"
}
```

### Response

Response bayar tagihan memiliki struktur yang **sama dengan Cek Tagihan**, dengan tambahan field:

| Parameter | Deskripsi | Tipe | Wajib |
|-----------|-----------|------|-------|
| `sn` | Serial Number / Reference Number | String | Ya |

Contoh response bayar PLN:

```json
{
  "data": {
    "ref_id": "some1d",
    "customer_no": "530000000001",
    "customer_name": "Nama Pelanggan Pertama",
    "buyer_sku_code": "pln",
    "admin": 2500,
    "message": "Transaksi Sukses",
    "status": "Sukses",
    "rc": "00",
    "periode": "201901",
    "sn": "S1234554321N",
    "buyer_last_saldo": 90000,
    "price": 10000,
    "selling_price": 11000,
    "desc": {
      "tarif": "R1",
      "daya": 1300,
      "lembar_tagihan": 1,
      "detail": [
        {
          "periode": "201901",
          "nilai_tagihan": "8000",
          "admin": "2500",
          "denda": "500",
          "meter_awal": "00080000",
          "meter_akhir": "00090000"
        }
      ]
    }
  }
}
```

> Jika status **Pending**, tunggu notifikasi via [Webhook](#webhooks) atau lakukan [Cek Status](#cek-status).

---

## Cek Status

> **Perhatian:** Jangan memanggil API untuk transaksi/data yang sama dalam interval kurang dari **1 menit** untuk menghindari race condition.

### Prabayar

Cek status dilakukan dengan **topup ulang menggunakan `ref_id` yang sama**. Lihat [Topup](#topup-prabayar).

> **Peringatan:** Jangan cek status transaksi yang sudah lewat **90 hari** — akan membuat transaksi **BARU**.

### Pascabayar

#### Endpoint

```
POST https://api.digiflazz.com/v1/transaction
```

#### Request

| Parameter | Deskripsi | Tipe | Wajib |
|-----------|-----------|------|-------|
| `commands` | `"status-pasca"` | String | Ya |
| `username` | Username dari pengaturan koneksi API | String | Ya |
| `buyer_sku_code` | Kode produk Anda | String | Ya |
| `customer_no` | Nomor Pelanggan | String | Ya |
| `ref_id` | Ref ID unik | String | Ya |
| `sign` | Signature: `md5(username + apiKey + ref_id)` | String | Ya |

```json
{
  "commands": "status-pasca",
  "username": "username",
  "buyer_sku_code": "pln",
  "customer_no": "530000000003",
  "ref_id": "some1d",
  "sign": "740b00a1b8784e028cc8078edf66d12b"
}
```

> Cek status pascabayar untuk transaksi lewat **90 hari** akan mendapat pesan `"Data belum ada"`.

---

## Inquiry PLN

Validasi ID PLN.

### Endpoint

```
POST https://api.digiflazz.com/v1/inquiry-pln
```

### Request

```json
{
  "username": "username",
  "customer_no": "1234554321",
  "sign": "740b00a1b8784e028cc8078edf66d12b"
}
```

| Parameter | Deskripsi | Tipe | Wajib |
|-----------|-----------|------|-------|
| `username` | Username dari pengaturan koneksi API | String | Ya |
| `customer_no` | ID PLN Customer | String | Ya |
| `sign` | Signature: `md5(username + apiKey + customer_no)` | String | Ya |

### Response

```json
{
  "data": {
    "message": "Transaksi Sukses",
    "status": "Sukses",
    "rc": "00",
    "customer_no": "1234554321",
    "meter_no": "1234554321",
    "subscriber_id": "523300817840",
    "name": "DAVID",
    "segment_power": "R1 /000001300"
  }
}
```

| Parameter | Deskripsi | Tipe | Wajib |
|-----------|-----------|------|-------|
| `message` | Deskripsi status | String | Ya |
| `status` | `Sukses` atau `Gagal` | String | Ya |
| `rc` | Response Code | String | Ya |
| `customer_no` | ID PLN | String | Ya |
| `meter_no` | Nomor meteran | String | Tidak |
| `subscriber_id` | ID customer | String | Tidak |
| `name` | Nama customer | String | Tidak |
| `segment_power` | Daya | String | Tidak |

---

## Test Case

### Prabayar

| `buyer_sku_code` | `customer_no` | Status |
|-----------------|---------------|--------|
| `xld10` | `087800001230` | Sukses |
| `xld10` | `087800001232` | Gagal |
| `xld10` | `087800001233` | Pending → Callback Sukses |
| `xld10` | `087800001234` | Pending → Callback Gagal |

#### Contoh Request Sukses

```json
{
  "username": "username",
  "buyer_sku_code": "xld10",
  "customer_no": "087800001230",
  "ref_id": "test1",
  "testing": true,
  "sign": "740b00a1b8784e028cc8078edf66d12b"
}
```

#### Contoh Response Sukses

```json
{
  "data": {
    "ref_id": "test1",
    "customer_no": "087800001230",
    "buyer_sku_code": "xld10",
    "message": "Transaksi Sukses",
    "status": "Sukses",
    "rc": "00",
    "sn": "1234567890",
    "buyer_last_saldo": 990000,
    "price": 10000
  }
}
```

---

### Pascabayar — Cara Request

#### Request Inquiry

```json
{
  "commands": "inq-pasca",
  "username": "username",
  "buyer_sku_code": "pln",
  "customer_no": "530000000001",
  "ref_id": "some1d",
  "testing": true,
  "sign": "740b00a1b8784e028cc8078edf66d12b"
}
```

#### Request Payment

```json
{
  "commands": "pay-pasca",
  "username": "username",
  "buyer_sku_code": "pln",
  "customer_no": "530000000001",
  "ref_id": "some1d",
  "sign": "740b00a1b8784e028cc8078edf66d12b"
}
```

#### Response Gagal

```json
{
  "data": {
    "ref_id": "some1d",
    "customer_no": "530000000003",
    "buyer_sku_code": "pln",
    "message": "Transaksi Gagal",
    "status": "Gagal",
    "rc": "02"
  }
}
```

#### Response Pending

```json
{
  "data": {
    "ref_id": "some1d",
    "customer_no": "530000000005",
    "buyer_sku_code": "pln",
    "message": "Transaksi Pending",
    "status": "Pending",
    "rc": "03"
  }
}
```

---

### Test Cases Pascabayar per Produk

#### PLN

| `buyer_sku_code` | `customer_no` | Status |
|-----------------|---------------|--------|
| `pln` | `530000000001` | Sukses (1 Tagihan) |
| `pln` | `530000000002` | Sukses (2 Tagihan) |
| `pln` | `530000000003` | Inquiry Gagal |
| `pln` | `530000000006` | Pembayaran Gagal |
| `pln` | `630000000001` | Pending → Callback Sukses (1 Tagihan) |
| `pln` | `630000000002` | Pending → Callback Sukses (2 Tagihan) |
| `pln` | `630000000006` | Pending → Callback Gagal |

#### PDAM

| `buyer_sku_code` | `customer_no` | Status |
|-----------------|---------------|--------|
| `pdam` | `1013226` | Sukses |
| `pdam` | `1013227` | Inquiry Gagal |
| `pdam` | `1013230` | Pembayaran Gagal |
| `pdam` | `2013226` | Pending → Callback Sukses |
| `pdam` | `2013230` | Pending → Callback Gagal |

#### INTERNET

| `buyer_sku_code` | `customer_no` | Status |
|-----------------|---------------|--------|
| `internet` | `6391601001` | Sukses |
| `internet` | `6391601002` | Inquiry Gagal |
| `internet` | `6391601005` | Pembayaran Gagal |
| `internet` | `7391601001` | Pending → Callback Sukses |
| `internet` | `7391601005` | Pending → Callback Gagal |

#### BPJS Kesehatan

| `buyer_sku_code` | `customer_no` | Status |
|-----------------|---------------|--------|
| `bpjs` | `8801234560001` | Sukses |
| `bpjs` | `8801234560002` | Inquiry Gagal |
| `bpjs` | `8801234560005` | Pembayaran Gagal |
| `bpjs` | `9801234560001` | Pending → Callback Sukses |
| `bpjs` | `9801234560005` | Pending → Callback Gagal |

#### Multifinance

| `buyer_sku_code` | `customer_no` | Status |
|-----------------|---------------|--------|
| `multifinance` | `6391601201` | Sukses |
| `multifinance` | `6391601202` | Inquiry Gagal |
| `multifinance` | `6391601205` | Pembayaran Gagal |
| `multifinance` | `7391601201` | Pending → Callback Sukses |
| `multifinance` | `7391601205` | Pending → Callback Gagal |

#### PBB

| `buyer_sku_code` | `customer_no` | Status |
|-----------------|---------------|--------|
| `cimahi` | `329801092375999991` | Sukses |
| `cimahi` | `329801092375999992` | Inquiry Gagal |
| `cimahi` | `329801092375999995` | Pembayaran Gagal |
| `cimahi` | `429801092375999991` | Pending → Callback Sukses |
| `cimahi` | `429801092375999995` | Pending → Callback Gagal |

#### Pajak Daerah Lainnya

| `buyer_sku_code` | `customer_no` | Status |
|-----------------|---------------|--------|
| `pdl` | `3298010921` | Sukses |
| `pdl` | `3298010922` | Inquiry Gagal |
| `pdl` | `3298010923` | Pembayaran Gagal |
| `pdl` | `4298010921` | Pending → Callback Sukses |
| `pdl` | `4298010923` | Pending → Callback Gagal |

#### GAS NEGARA

| `buyer_sku_code` | `customer_no` | Status |
|-----------------|---------------|--------|
| `pgas` | `0110014601` | Sukses (1 Tagihan) |
| `pgas` | `0110014602` | Sukses (2 Tagihan) |
| `pgas` | `0110014603` | Inquiry Gagal |
| `pgas` | `0110014605` | Pembayaran Gagal |
| `pgas` | `1110014601` | Pending → Callback Sukses (1 Tagihan) |
| `pgas` | `1110014602` | Pending → Callback Sukses (2 Tagihan) |
| `pgas` | `1110014605` | Pending → Callback Gagal |

#### TV

| `buyer_sku_code` | `customer_no` | Status |
|-----------------|---------------|--------|
| `tv` | `127246500101` | Sukses |
| `tv` | `127246500102` | Inquiry Gagal |
| `tv` | `127246500105` | Pembayaran Gagal |
| `tv` | `227246500101` | Pending → Callback Sukses |
| `tv` | `227246500105` | Pending → Callback Gagal |

#### BPJSTK

| `buyer_sku_code` | `customer_no` | Status |
|-----------------|---------------|--------|
| `bpjstk` | `8102051011270001` | Sukses |
| `bpjstk` | `8102051011270002` | Inquiry Gagal |
| `bpjstk` | `8102051011270003` | Pembayaran Gagal |
| `bpjstk` | `9102051011270001` | Pending → Callback Sukses |
| `bpjstk` | `9102051011270003` | Pending → Callback Gagal |

#### BPJSTKPU (Penerima Upah)

| `buyer_sku_code` | `customer_no` | Status |
|-----------------|---------------|--------|
| `bpjstkpu` | `8102051011280001` | Sukses |
| `bpjstkpu` | `8102051011280002` | Inquiry Gagal |
| `bpjstkpu` | `8102051011280003` | Pembayaran Gagal |
| `bpjstkpu` | `9102051011280001` | Pending → Callback Sukses |
| `bpjstkpu` | `9102051011280003` | Pending → Callback Gagal |

#### PLN Nontaglis

| `buyer_sku_code` | `customer_no` | Status |
|-----------------|---------------|--------|
| `plnnontaglis` | `8102051011290001` | Sukses |
| `plnnontaglis` | `8102051011290002` | Inquiry Gagal |
| `plnnontaglis` | `8102051011290003` | Pembayaran Gagal |
| `plnnontaglis` | `9102051011290001` | Pending → Callback Sukses |
| `plnnontaglis` | `9102051011290003` | Pending → Callback Gagal |

#### E-Money

| `buyer_sku_code` | `customer_no` | Status |
|-----------------|---------------|--------|
| `emoney` | `8102051011300001` | Sukses |
| `emoney` | `8102051011300002` | Inquiry Gagal |
| `emoney` | `8102051011300003` | Pembayaran Gagal |
| `emoney` | `9102051011300001` | Pending → Callback Sukses |
| `emoney` | `9102051011300003` | Pending → Callback Gagal |

#### SAMSAT

| `buyer_sku_code` | `customer_no` | Status |
|-----------------|---------------|--------|
| `samsat` | `8102051011310001` | Sukses |
| `samsat` | `8102051011310002` | Inquiry Gagal |
| `samsat` | `8102051011310003` | Pembayaran Gagal |
| `samsat` | `9102051011310001` | Pending → Callback Sukses |
| `samsat` | `9102051011310003` | Pending → Callback Gagal |

#### HP dan Tagihan Lainnya

| `buyer_sku_code` | `customer_no` | Status |
|-----------------|---------------|--------|
| `hp` | `8102051011320001` | Sukses |
| `hp` | `8102051011320002` | Inquiry Gagal |
| `hp` | `8102051011320003` | Pembayaran Gagal |
| `hp` | `9102051011320001` | Pending → Callback Sukses |
| `hp` | `9102051011320003` | Pending → Callback Gagal |

---

## Response Code

| RC | Message | Status | Transaksi Terbentuk | Deskripsi |
|----|---------|--------|---------------------|-----------|
| `00` | Transaksi Sukses | Sukses | Ya | — |
| `01` | Timeout | Gagal | Ya | — |
| `02` | Transaksi Gagal | Gagal | Ya | — |
| `03` | Transaksi Pending | Pending | Ya | — |
| `40` | Payload Error | Gagal | Tidak | Tipe data atau parameter tidak sesuai |
| `41` | Signature tidak valid | Gagal | Tidak | Periksa formula signature dan apiKey (Development/Production) |
| `42` | Gagal memproses API Buyer | Gagal | Tidak | Username belum sesuai |
| `43` | SKU tidak ditemukan atau Non-Aktif | Gagal | Tidak | — |
| `44` | Saldo tidak cukup | Gagal | Tidak | — |
| `45` | IP tidak dikenali | Gagal | Tidak | Whitelist IP di pengaturan koneksi |
| `47` | Transaksi sudah terjadi di buyer lain | Gagal | Tidak | — |
| `49` | Ref ID tidak unik | Gagal | Tidak | — |
| `50` | Transaksi Tidak Ditemukan | Gagal | Ya | — |
| `51` | Nomor Tujuan Diblokir | Gagal | Ya | — |
| `52` | Prefix Tidak Sesuai Dengan Operator | Gagal | Ya | — |
| `53` | Produk Seller Sedang Tidak Tersedia | Gagal | Ya | — |
| `54` | Nomor Tujuan Salah | Gagal | Ya | — |
| `55` | Produk Sedang Gangguan | Gagal | Ya | — |
| `56` | Limit saldo seller | Gagal | Tidak | *Deprecated* |
| `57` | Jumlah Digit Kurang Atau Lebih | Gagal | Ya | — |
| `58` | Sedang Cut Off | Gagal | Ya | — |
| `59` | Tujuan di Luar Wilayah/Cluster | Gagal | Ya | — |
| `60` | Tagihan belum tersedia | Gagal | Ya | — |
| `61` | Belum pernah melakukan deposit | Gagal | Tidak | — |
| `62` | Seller sedang mengalami gangguan | Gagal | Tidak | — |
| `63` | Tidak support transaksi multi | Gagal | Tidak | — |
| `64` | Tarik tiket gagal, coba nominal lain | Gagal | Tidak | — |
| `65` | Limit transaksi multi | Gagal | Tidak | *Deprecated* |
| `66` | Cut Off (Perbaikan Sistem Seller) | Gagal | Tidak | — |
| `67` | Seller belum ter-verifikasi | Gagal | Tidak | — |
| `68` | Stok habis | Gagal | Tidak | — |
| `69` | Harga seller lebih besar dari ketentuan | Gagal | Tidak | — |
| `70` | Timeout Dari Biller | Gagal | Ya | — |
| `71` | Produk Sedang Tidak Stabil | Gagal | Ya | — |
| `72` | Lakukan Unreg Paket Dahulu | Gagal | Ya | — |
| `73` | Kwh Melebihi Batas | Gagal | Ya | — |
| `74` | Transaksi Refund | Gagal | Ya | — |
| `80` | Akun Anda telah diblokir oleh Seller | Gagal | Tidak | — |
| `81` | Seller ini telah diblokir oleh Anda | Gagal | Tidak | — |
| `82` | Akun Anda belum ter-verifikasi | Gagal | Tidak | — |
| `83` | Limitasi pengecekan pricelist | Gagal | Tidak | Max 1x per 5 menit untuk semua produk; 1x per detik per kode |
| `84` | Nominal tidak valid | Gagal | Ya | — |
| `85` | Limitasi transaksi, coba 1 menit lagi | Gagal | Ya | — |
| `86` | Limitasi pengecekan nomor PLN | Gagal | Ya | — |
| `87` | Transaksi E-money wajib kelipatan Rp 1.000 | Gagal | Tidak | — |
| `88` | Akun tidak dapat melakukan aksi ini | Gagal | Tidak | — |
| `99` | DF Router Issue | Pending | Ya | — |

---

## Webhooks

Webhooks mengirimkan **HTTP POST** ke URL yang dikonfigurasi saat ada event transaksi (tambah/update status).

Konfigurasi di: **Atur Koneksi > API > Webhook**

### Headers

| Header | Deskripsi |
|--------|-----------|
| `X-Digiflazz-Event` | Nama event: `create` atau `update` |
| `X-Hub-Signature` | HMAC hex sha1 dari response body (hanya jika pakai `secret`) |
| `User-Agent` | `Digiflazz-Hookshot` (prepaid) atau `Digiflazz-Pasca-Hookshot` (postpaid) |

### Events

| Event | Deskripsi |
|-------|-----------|
| `create` | Transaksi baru terjadi |
| `update` | Transaksi mengalami perubahan status |

### Contoh Payload — Prabayar

```http
POST /webhook HTTP/1.1
Host: localhost:4567
X-Hub-Signature: sha1=7d6f016c23d03b696e76dada91c07f178cc0af4d
User-Agent: Digiflazz-Hookshot
Content-Type: application/json
X-Digiflazz-Event: create

{
  "data": {
    "ref_id": "30467470",
    "customer_no": "081280556115",
    "buyer_sku_code": "ovo100",
    "message": "Sukses",
    "status": "Sukses",
    "rc": "00",
    "buyer_last_saldo": 326719460,
    "sn": "SEPTIAPAR/20190401214753214742",
    "price": 199800,
    "tele": "@telegram",
    "wa": "081234512345"
  }
}
```

### Contoh Payload — Pascabayar (PLN)

```http
POST /webhook HTTP/1.1
Host: localhost:4567
X-Hub-Signature: sha1=debdf6dfb3b62dfd3e98cd39e600027080938f52
User-Agent: Digiflazz-Pasca-Hookshot
Content-Type: application/json
X-Digiflazz-Event: update

{
  "data": {
    "ref_id": "1763103975",
    "customer_no": "530000000000",
    "customer_name": "SUBCRIBER NAME",
    "buyer_sku_code": "plnpsaca",
    "admin": 2750,
    "message": "Transaksi Sukses",
    "status": "Sukses",
    "rc": "00",
    "sn": "004212C9245F1BA43A77CEBD5CD5DA39",
    "periode": "201608",
    "buyer_last_saldo": 326719460,
    "price": 300950,
    "selling_price": 302750,
    "desc": {
      "tarif": "R1",
      "daya": 1300,
      "lembar_tagihan": 1800,
      "detail": [
        {
          "periode": "201608",
          "nilai_tagihan": "300000",
          "admin": "2750",
          "denda": "0",
          "meter_awal": "00080000",
          "meter_akhir": "00080000"
        }
      ]
    }
  }
}
```

### Contoh Handle Event (PHP/Laravel)

```php
Route::post('/webhook', function(Request $request) {
    $secret = 'somesecretvalue';
    $post_data = file_get_contents('php://input');
    $signature = hash_hmac('sha1', $post_data, $secret);

    if ($request->header('X-Hub-Signature') == 'sha1='.$signature) {
        \Log::info(json_decode($request->getContent(), true));
    }
});
```

### Ping Event

Saat webhook dikonfigurasi, Digiflazz mengirim event `ping` untuk memverifikasi URL.

#### Ping Endpoint

```
POST https://api.digiflazz.com/v1/report/hooks/[YOUR-WEBHOOK-ID]/pings
```

#### Contoh Ping Response

```json
{
  "sed": "AgXXtVAHp",
  "hook_id": "11aaabbb",
  "hook": {
    "url": "https://awesomesite.com/webhooks",
    "secret": "somesecretkeywords",
    "type": "application/json",
    "status": 1
  }
}
```

---

## Catatan Penting

1. **Signature Formula:**
   - Cek Saldo: `md5(username + apiKey + "depo")`
   - Price List: `md5(username + apiKey + "pricelist")`
   - Deposit: `md5(username + apiKey + "deposit")`
   - Topup / Cek Tagihan / Bayar Tagihan / Cek Status: `md5(username + apiKey + ref_id)`
   - Inquiry PLN: `md5(username + apiKey + customer_no)`

2. **Mode Development vs Production:** Pastikan apiKey sesuai mode yang digunakan.

3. **Whitelist IP:** Whitelist IP server Anda di pengaturan koneksi API, dan whitelist `52.74.250.133` (IP Digiflazz) di sistem Anda.

4. **ref_id harus unik** per transaksi. Gunakan ref_id yang sama hanya untuk cek status.

5. **Response selalu dibungkus** dalam variable `data`.
