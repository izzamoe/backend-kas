# Migrations

Direktori ini berisi semua migration files untuk PocketBase database.

## Migration Files

### 1774949033_collections_snapshot.go
Snapshot dari semua collections yang ada di database. File ini di-generate otomatis dan berisi struktur lengkap dari:
- users
- families
- family_members
- categories
- transactions
- file
- System collections (_mfas, _otps, _externalAuths, _authOrigins, _superusers)

### 1774949034_initial_superuser.go
Migration untuk membuat initial superuser/admin account.

**Default Credentials:**
- Email: `admin@example.com`
- Password: `admin123456`

**Untuk Production:**
Set environment variables sebelum menjalankan migrasi:
```bash
export ADMIN_EMAIL="your-email@example.com"
export ADMIN_PASSWORD="your-secure-password"
go run . migrate up
```

## Commands

### Generate new migration
```bash
go run . migrate create "migration_name"
```

### Generate collections snapshot
```bash
go run . migrate collections
```

### Apply migrations manually
```bash
go run . migrate up
```

### Revert last migration
```bash
go run . migrate down
```

### Sync migration history
```bash
go run . migrate history-sync
```

## Auto-migration

Automigrate diaktifkan saat development (`go run`). Setiap perubahan collection di Admin Dashboard akan otomatis membuat migration file baru.

Untuk production, set `Automigrate: false` di main.go.

## Notes

- Migration files akan di-apply otomatis saat `serve` command dijalankan
- Migration history disimpan di table `_migrations`
- Setiap migration file harus memiliki fungsi `up` dan `down` (revert)
- Migration dijalankan dalam urutan timestamp filename
