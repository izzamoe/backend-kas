# Guide: Expand Relations di PocketBase dengan Generated Proxies

## Overview

Expand adalah fitur untuk memuat relasi (relations) antar collection dalam satu query.
Generated proxies dari `pocketbase-gogen` sudah support expand secara type-safe.

## Cara Kerja Expand

### 1. Query Record + Expand

Ada **2 cara** untuk expand relations:

#### Cara 1: Gunakan `ExpandRecord()` / `ExpandRecords()` (RECOMMENDED)

```go
// Single record
record, err := app.FindRecordById("transactions", id)
if err != nil {
    return nil, err
}

// Expand relations
expandFields := []string{"category_id", "created_by", "family_id"}
app.ExpandRecord(record, expandFields, nil)

// Multiple records
records, err := app.FindRecordsByFilter("transactions", filter, sort, limit, offset, params)
if err != nil {
    return nil, err
}

// Expand relations untuk semua records
expandFields := []string{"category_id", "created_by", "family_id"}
app.ExpandRecords(records, expandFields, nil)
```

#### Cara 2: Gunakan `optFilters` callback (Advanced)

```go
// Gunakan optFilters parameter untuk modify query
record, err := app.FindRecordById("transactions", id, func(q *dbx.SelectQuery) error {
    // Custom query modification
    return nil
})
```

**PENTING**: 
- `FindRecordById`, `FindRecordsByFilter`, dll **TIDAK** accept `dbx.HashExp{"expand": "..."}` sebagai parameter!
- Parameter ke-3 adalah **variadic function** `func(q *dbx.SelectQuery) error`, bukan map!
- Untuk expand, gunakan method `ExpandRecord()` atau `ExpandRecords()` setelah query.

### 2. Akses Expanded Data dengan Type-Safe Proxies

Setelah expand, gunakan generated proxies dengan **nil check**:

```go
proxy, _ := generated.WrapRecord[generated.Transactions](record)

// ❌ SALAH: Tanpa nil check akan PANIC!
familyID := proxy.FamilyId().Id  // PANIC jika tidak di-expand!

// ✅ BENAR: Dengan nil check
if family := proxy.FamilyId(); family != nil {
    familyID := family.Id
    familyName := family.Name()
    // ... akses field lainnya
}
```

## Contoh Implementasi Lengkap

### Domain Model (DTO)

```go
// internal/domain/transaction.go
type TransactionDTO struct {
    ID         string          `json:"id"`
    FamilyID   string          `json:"family_id"`
    CreatedBy  string          `json:"created_by"`
    CategoryID string          `json:"category_id"`
    Type       TransactionType `json:"type"`
    Amount     float64         `json:"amount"`
    Note       string          `json:"note"`
    Date       time.Time       `json:"date"`
    CreatedAt  time.Time       `json:"created_at"`
    UpdatedAt  time.Time       `json:"updated_at"`
    
    // Expanded fields (optional)
    Family   *FamilyExpand   `json:"family,omitempty"`
    Category *CategoryExpand `json:"category,omitempty"`
    Creator  *UserExpand     `json:"creator,omitempty"`
}

type FamilyExpand struct {
    ID         string `json:"id"`
    Name       string `json:"name"`
    InviteCode string `json:"invite_code,omitempty"`
}

type CategoryExpand struct {
    ID        string `json:"id"`
    Name      string `json:"name"`
    Icon      string `json:"icon"`
    Color     string `json:"color"`
    IsDefault bool   `json:"is_default"`
}

type UserExpand struct {
    ID     string `json:"id"`
    Name   string `json:"name"`
    Avatar string `json:"avatar,omitempty"`
}
```

### Repository Implementation

```go
// internal/repository/transaction_repository.go

// GetByID dengan expand
func (r *transactionRepo) GetByID(id string) (*domain.TransactionDTO, error) {
    // 1. Query record
    record, err := r.app.FindRecordById("transactions", id)
    if err != nil {
        return nil, err
    }

    // 2. Expand relations
    expandFields := []string{"category_id", "created_by", "family_id"}
    r.app.ExpandRecord(record, expandFields, nil)

    // 3. Convert to DTO
    return r.recordToDTO(record), nil
}

// GetByFamilyID dengan expand (multiple records)
func (r *transactionRepo) GetByFamilyID(familyID string, limit, offset int) ([]*domain.TransactionDTO, error) {
    // 1. Query records
    records, err := r.app.FindRecordsByFilter(
        "transactions",
        "family_id = {:familyID}",
        "-created",
        limit, offset,
        map[string]any{"familyID": familyID},
    )
    if err != nil {
        return nil, err
    }

    // 2. Expand relations untuk semua records
    expandFields := []string{"category_id", "created_by", "family_id"}
    r.app.ExpandRecords(records, expandFields, nil)

    // 3. Convert to DTOs
    dtos := make([]*domain.TransactionDTO, len(records))
    for i, record := range records {
        dtos[i] = r.recordToDTO(record)
    }

    return dtos, nil
}

// recordToDTO dengan nil check untuk expanded fields
func (r *transactionRepo) recordToDTO(record *core.Record) *domain.TransactionDTO {
    proxy, _ := generated.WrapRecord[generated.Transactions](record)

    typeStr := "income"
    if proxy.Type() == generated.Expense {
        typeStr = "expense"
    }

    dto := &domain.TransactionDTO{
        ID:        proxy.Id,
        Type:      domain.TransactionType(typeStr),
        Amount:    proxy.Amount(),
        Note:      proxy.Note(),
        Date:      proxy.Date().Time(),
        CreatedAt: proxy.Created().Time(),
        UpdatedAt: proxy.Updated().Time(),
    }

    // Nil check untuk expanded relations
    if family := proxy.FamilyId(); family != nil {
        dto.FamilyID = family.Id
        dto.Family = &domain.FamilyExpand{
            ID:         family.Id,
            Name:       family.Name(),
            InviteCode: family.InviteCode(),
        }
    }

    if creator := proxy.CreatedBy(); creator != nil {
        dto.CreatedBy = creator.Id
        dto.Creator = &domain.UserExpand{
            ID:     creator.Id,
            Name:   creator.Name(),
            Avatar: creator.Avatar(),
        }
    }

    if category := proxy.CategoryId(); category != nil {
        dto.CategoryID = category.Id
        dto.Category = &domain.CategoryExpand{
            ID:        category.Id,
            Name:      category.Name(),
            Icon:      category.Icon(),
            Color:     category.Color(),
            IsDefault: category.IsDefault(),
        }
    }

    return dto
}
```

## API Response Examples

### Tanpa Expand
```json
{
  "id": "abc123",
  "family_id": "fam456",
  "created_by": "user789",
  "category_id": "cat012",
  "type": "expense",
  "amount": 50000,
  "note": "Belanja bulanan",
  "date": "2024-01-15T10:00:00Z",
  "created_at": "2024-01-15T10:00:00Z",
  "updated_at": "2024-01-15T10:00:00Z"
}
```

### Dengan Expand
```json
{
  "id": "abc123",
  "family_id": "fam456",
  "created_by": "user789",
  "category_id": "cat012",
  "type": "expense",
  "amount": 50000,
  "note": "Belanja bulanan",
  "date": "2024-01-15T10:00:00Z",
  "created_at": "2024-01-15T10:00:00Z",
  "updated_at": "2024-01-15T10:00:00Z",
  "family": {
    "id": "fam456",
    "name": "Keluarga Besar",
    "invite_code": "ABC123"
  },
  "creator": {
    "id": "user789",
    "name": "John Doe",
    "avatar": "avatar.jpg"
  },
  "category": {
    "id": "cat012",
    "name": "Groceries",
    "icon": "🛒",
    "color": "#FF5733",
    "is_default": true
  }
}
```

## Best Practices

### ✅ DO

1. **Selalu gunakan nil check** saat akses expanded relations:
   ```go
   if family := proxy.FamilyId(); family != nil {
       // safe to access
   }
   ```

2. **Expand hanya field yang dibutuhkan**:
   ```go
   // Hanya expand category jika perlu
   expandFields := []string{"category_id"}
   ```

3. **Gunakan `ExpandRecords()` untuk multiple records**:
   ```go
   app.ExpandRecords(records, expandFields, nil)
   ```

4. **Tambahkan field expanded di DTO dengan `omitempty`**:
   ```go
   Family *FamilyExpand `json:"family,omitempty"`
   ```

### ❌ DON'T

1. **Jangan akses expanded field tanpa nil check**:
   ```go
   // PANIC jika tidak di-expand!
   name := proxy.FamilyId().Name()
   ```

2. **Jangan gunakan `dbx.HashExp{"expand": "..."}` di params**:
   ```go
   // ❌ SALAH - ini tidak bekerja!
   record, _ := app.FindRecordById("transactions", id, dbx.HashExp{
       "expand": "category_id",
   })
   ```

3. **Jangan expand semua field jika tidak perlu**:
   ```go
   // Waste of resources jika tidak digunakan
   expandFields := []string{"field1", "field2", "field3", "field4"}
   ```

## Nested Expand

Untuk expand relation di dalam relation (nested):

```go
// Expand category dan family dari category
expandFields := []string{
    "category_id",
    "category_id.family_id",  // nested expand
}
app.ExpandRecord(record, expandFields, nil)

// Akses nested
proxy, _ := generated.WrapRecord[generated.Transactions](record)
if category := proxy.CategoryId(); category != nil {
    if family := category.FamilyId(); family != nil {
        // akses family dari category
        familyName := family.Name()
    }
}
```

## Generated Relations Map

File `generated/utils.go` berisi map relasi:

```go
var Relations = map[string]map[string][]RelationField{
    "transactions": {
        "users": {
            {"created_by", false},
        },
        "families": {
            {"family_id", false},
        },
        "categories": {
            {"category_id", false},
        },
    },
    // ...
}
```

Map ini berguna untuk:
- Melihat relasi antar collection
- Validasi field expand
- Auto-completion di IDE

## Performance Tips

1. **Lazy expand**: Expand hanya saat dibutuhkan
2. **Batch expand**: Gunakan `ExpandRecords()` untuk multiple records (lebih efisien)
3. **Cache**: Cache expanded data jika sering diakses
4. **Pagination**: Limit records sebelum expand

## Troubleshooting

### Problem: PANIC saat akses expanded field

**Solusi**: Tambahkan nil check
```go
if family := proxy.FamilyId(); family != nil {
    // safe
}
```

### Problem: Expanded field tidak terisi

**Solusi**: 
1. Pastikan `ExpandRecord()` dipanggil setelah query
2. Cek nama field expand sesuai dengan schema
3. Cek relasi exist di database

### Problem: Field ID nil tapi expand berhasil

**Solusi**: Field relasi bisa null di database, cek schema PocketBase

---

**Last Updated**: 2024  
**PocketBase Version**: v0.36.8  
**pocketbase-gogen**: Latest
