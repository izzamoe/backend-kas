# Uang Kas Keluarga

Aplikasi backend untuk manajemen kas keluarga menggunakan PocketBase dengan Go.

## Arsitektur

Project ini menggunakan **Clean Architecture** dengan layer separation yang jelas:

```
.
├── main.go                    # Entry point & dependency injection
├── pbschema/
│   └── template.go            # Schema as code (editable)
├── generated/                 # Generated code (DO NOT EDIT)
│   ├── proxies.go             # Type-safe proxies
│   ├── proxy_events.go        # Event handlers
│   ├── proxy_hooks.go         # PocketBase hooks
│   └── utils.go               # Utility functions
└── internal/
    ├── domain/                # Domain models & DTOs
    ├── repository/            # Data access layer (menggunakan generated proxies)
    ├── service/               # Business logic layer
    ├── handler/               # HTTP handlers/controllers
    ├── middleware/            # Custom middleware
    └── utils/                 # Utilities
```

## Flow Architecture

```
┌─────────────┐
│   Client    │
└──────┬──────┘
       │ HTTP Request
       ▼
┌─────────────────────────────────────┐
│         Handler Layer               │  ← HTTP handling, request/response
│  (internal/handler)                 │
└──────┬──────────────────────────────┘
       │ Call service
       ▼
┌─────────────────────────────────────┐
│         Service Layer               │  ← Business logic & validation
│  (internal/service)                 │
└──────┬──────────────────────────────┘
       │ Call repository
       ▼
┌─────────────────────────────────────┐
│       Repository Layer              │  ← Data access
│  (internal/repository)              │  ← Menggunakan generated proxies
│  Uses: kas/generated proxies        │
└──────┬──────────────────────────────┘
       │ PocketBase API
       ▼
┌─────────────────────────────────────┐
│         PocketBase DB               │
│         (SQLite)                    │
└─────────────────────────────────────┘
```

## Layer Explanation

### 1. Domain Layer (`internal/domain/`)
- Berisi **pure business models** dan DTOs
- Tidak ada dependency ke layer lain
- Define contracts untuk request/response

**Example:**
```go
type TransactionDTO struct {
    ID         string
    FamilyID   string
    Amount     float64
    Note       string
}
```

### 2. Repository Layer (`internal/repository/`)
- **Data access layer** - interact dengan PocketBase
- Menggunakan **generated type-safe proxies** dari `kas/generated`
- Convert antara `core.Record` ↔ `domain.DTO`
- Semua query database ada di sini

**Example:**
```go
// Repository menggunakan generated proxy untuk type-safety
func (r *transactionRepo) recordToDTO(record *core.Record) *domain.TransactionDTO {
    // Gunakan generated WrapRecord untuk create proxy
    proxy, _ := generated.WrapRecord[generated.Transactions](record)
    
    return &domain.TransactionDTO{
        ID:     proxy.Id,
        Amount: proxy.Amount(),
        Note:   proxy.Note(),
        // Type-safe! Compiler akan error jika field tidak ada
    }
}
```

### 3. Service Layer (`internal/service/`)
- **Business logic** dan validation
- Authorization checks
- Orchestrate multiple repositories jika perlu
- Transaction handling

**Example:**
```go
func (s *transactionService) CreateTransaction(req *domain.CreateTransactionRequest, userID string) (*domain.TransactionDTO, error) {
    // Business validation
    if req.Amount <= 0 {
        return nil, errors.New("amount must be greater than 0")
    }
    
    // Call repository
    return s.transactionRepo.Create(req, userID)
}
```

### 4. Handler Layer (`internal/handler/`)
- Handle HTTP requests/responses
- Parse request body & query params
- Call services
- Return JSON responses

**Example:**
```go
func (h *TransactionHandler) Create(e *core.RequestEvent) error {
    var req domain.CreateTransactionRequest
    if err := e.BindBody(&req); err != nil {
        return e.BadRequestError("Invalid request body", err)
    }
    
    transaction, err := h.service.CreateTransaction(&req, e.Auth.Id)
    if err != nil {
        return e.BadRequestError("Failed to create", err)
    }
    
    return e.JSON(http.StatusCreated, transaction)
}
```

## Generated Code (Type-Safe Proxies)

### Apa itu Generated Proxies?

Generated proxies adalah **type-safe wrappers** untuk `core.Record` dari PocketBase. Daripada akses field dengan string:

```go
// ❌ Tidak type-safe, error di runtime
amount := record.Get("amount").(float64)
```

Kita bisa pakai generated getter yang **type-safe**:

```go
// ✅ Type-safe, error di compile time
proxy, _ := generated.WrapRecord[generated.Transactions](record)
amount := proxy.Amount() // float64
```

### Cara Kerja Generated Code:

1. **Schema → Template**
   ```bash
   pocketbase-gogen template ./pb_data ./pbschema/template.go
   ```
   - Membaca schema dari `pb_data/data.db`
   - Generate file `template.go` yang editable
   - Bisa custom: rename fields, add methods, etc

2. **Template → Proxies**
   ```bash
   pocketbase-gogen generate ./pbschema/template.go ./generated/proxies.go --utils --hooks
   ```
   - Generate type-safe proxy structs
   - Generate getters/setters untuk setiap field
   - Generate utils dan hooks jika pakai flag `--utils --hooks`

### Regenerate Proxies:

Setiap kali schema PocketBase berubah:

```bash
make generate
```

Ini akan:
1. Generate ulang template dari `pb_data`
2. Generate ulang proxies dari template

## Workflow Development

### 1. Setup Awal
```bash
# Install dependencies
make install

# Run server
make serve
```

### 2. Ubah Schema di PocketBase Admin
- Buka http://localhost:8090/_/
- Edit collections (tambah/hapus field)

### 3. Regenerate Code
```bash
# Generate ulang proxies setelah schema berubah
make generate
```

### 4. Update Business Logic
- Update di `internal/repository/` jika perlu akses field baru
- Update di `internal/service/` untuk business logic
- Update di `internal/handler/` untuk API endpoints

### 5. Testing
```bash
make test
```

## API Endpoints

> Semua endpoint memerlukan header `Authorization: Bearer YOUR_TOKEN` kecuali disebutkan lain.

### Transactions

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/api/transactions` | Create new transaction |
| GET | `/api/transactions/:id` | Get transaction by ID |
| GET | `/api/families/:familyId/transactions` | Get family transactions (paginated) |
| PATCH | `/api/transactions/:id` | Update transaction |
| DELETE | `/api/transactions/:id` | Delete transaction |
| GET | `/api/families/:familyId/balance` | Get family balance |

**Example Request:**
```bash
# Create transaction
curl -X POST http://localhost:8090/api/transactions \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -d '{
    "category_id": "cat123",
    "type": "expense",
    "amount": 50000,
    "note": "Belanja bulanan",
    "date": "2026-03-31T10:00:00Z"
  }'
```

### Reports

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/reports/monthly?year=&month=` | Monthly financial report with income & expense breakdown per category |
| GET | `/api/reports/summary?year=&month=` | Dashboard summary (balance, monthly stats, % change vs prev month) |

#### GET `/api/reports/monthly`

Query parameters:

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `year` | integer | Yes | Year (1900–2100) |
| `month` | integer | Yes | Month (1–12) |

**Response:**
```json
{
  "family_id": "abc123",
  "year": 2026,
  "month": 4,
  "total_income": 5000000,
  "total_expense": 1800000,
  "balance": 3200000,
  "expense_breakdown": [
    {
      "category_id": "cat1",
      "category_name": "Makanan",
      "icon": "🍔",
      "color": "#FF5733",
      "total_amount": 800000,
      "count": 12
    }
  ],
  "income_breakdown": [
    {
      "category_id": "cat5",
      "category_name": "Gaji",
      "icon": "💼",
      "color": "#33FF57",
      "total_amount": 5000000,
      "count": 1
    }
  ]
}
```

**Example Request:**
```bash
curl http://localhost:8090/api/reports/monthly?year=2026&month=4 \
  -H "Authorization: Bearer YOUR_TOKEN"
```

#### GET `/api/reports/summary`

Query parameters:

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `year` | integer | No | Year — default: current year |
| `month` | integer | No | Month (1–12) — default: current month |

**Response:**
```json
{
  "family_name": "Keluarga Bahagia",
  "user_name": "Budi",
  "total_balance": 12000000,
  "monthly_income": 5000000,
  "monthly_income_change": 10.5,
  "monthly_expense": 1800000,
  "monthly_expense_change": -5.2
}
```

**Example Request:**
```bash
curl "http://localhost:8090/api/reports/summary?year=2026&month=4" \
  -H "Authorization: Bearer YOUR_TOKEN"
```

## Commands

```bash
make help              # Show all commands
make install           # Install dependencies
make build             # Build the app
make run               # Build and run
make serve             # Run PocketBase server
make dev               # Run with auto-reload (needs air)
make generate          # Regenerate type-safe proxies
make test              # Run tests
make clean             # Clean build artifacts
make admin             # Create admin account
make backup            # Backup pb_data
make update            # Update dependencies
```

## Menambah Feature Baru

### Contoh: Menambah Category Management

1. **Buat domain model** (`internal/domain/category.go`):
```go
type CategoryDTO struct {
    ID       string
    Name     string
    Icon     string
    Color    string
}
```

2. **Buat repository** (`internal/repository/category_repository.go`):
```go
type CategoryRepository interface {
    Create(req *CreateCategoryRequest) (*CategoryDTO, error)
    GetByID(id string) (*CategoryDTO, error)
}

// Implementation menggunakan generated.Categories proxy
```

3. **Buat service** (`internal/service/category_service.go`):
```go
type CategoryService interface {
    CreateCategory(req *CreateCategoryRequest) (*CategoryDTO, error)
}
```

4. **Buat handler** (`internal/handler/category_handler.go`):
```go
func (h *CategoryHandler) Create(e *core.RequestEvent) error {
    // Handle HTTP request
}
```

5. **Wire di main.go**:
```go
categoryRepo := repository.NewCategoryRepository(app)
categoryService := service.NewCategoryService(categoryRepo)
categoryHandler := handler.NewCategoryHandler(categoryService)
categoryHandler.RegisterRoutes(se)
```

## Best Practices

### ✅ DO
- Gunakan generated proxies di **repository layer** untuk type-safety
- Put business logic di **service layer**
- Keep handlers thin - hanya HTTP handling
- Use dependency injection (DI) di `main.go`
- Regenerate proxies setiap schema berubah

### ❌ DON'T
- Jangan edit file di `kas/generated/` - akan ter-overwrite
- Jangan akses database langsung dari handler/service
- Jangan put business logic di handler
- Jangan hardcode connection strings
- Jangan commit `pb_data/` ke git

## Environment Variables

Buat file `.env` (optional):
```bash
PB_PORT=8090
PB_HOST=0.0.0.0

# Digiflazz Integration
DIGIFLAZZ_CREDENTIAL_ENCRYPTION_KEY=your-32-char-encryption-key
DIGIFLAZZ_PRICE_SYNC_INTERVAL=24h
DIGIFLAZZ_ORDER_POLL_INTERVAL=1m
```

### Digiflazz API Endpoints

Aplikasi ini mendukung integrasi dengan Digiflazz untuk pembelian pulsa, paket data, dan tagihan PPOB.

#### Credentials

| Method | Endpoint | Description | Auth |
|--------|----------|-------------|------|
| GET | `/api/digiflazz/credential` | Get credential metadata for family | Member+ |
| POST | `/api/digiflazz/credential` | Set/update Digiflazz credentials | Owner |
| DELETE | `/api/digiflazz/credential` | Remove Digiflazz credentials | Owner |
| POST | `/api/digiflazz/credential/rotate` | Rotate webhook token | Owner |
| POST | `/api/digiflazz/credential/test-webhook` | Trigger Digiflazz webhook ping test | Owner |
| GET | `/api/digiflazz/credential/balance` | Check deposit balance from Digiflazz | Member+ |
| POST | `/api/digiflazz/deposit` | Create deposit ticket (bank transfer) | Owner |

#### Products

| Method | Endpoint | Description | Auth |
|--------|----------|-------------|------|
| GET | `/api/digiflazz/products` | Search/list available products | Member+ |
| POST | `/api/digiflazz/products/sync` | Manually trigger pricelist sync from Digiflazz | Owner |

#### Orders

| Method | Endpoint | Description | Auth |
|--------|----------|-------------|------|
| GET | `/api/digiflazz/orders` | List orders (paginated) | Member+ |
| GET | `/api/digiflazz/orders/{id}` | Get order detail | Member+ |
| POST | `/api/digiflazz/orders` | Create prepaid order or postpaid inquiry (set `order_type`: `prepaid`/`postpaid`) | Member+ |
| POST | `/api/digiflazz/orders/{id}/pay` | Pay for an existing postpaid inquiry | Member+ |

#### Webhook

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/webhooks/digiflazz/{token}` | Receive Digiflazz callbacks (no auth — token in path) |


## Tech Stack

- **Backend Framework**: PocketBase
- **Language**: Go 1.26.1
- **Database**: SQLite (via PocketBase)
- **Code Generation**: pocketbase-gogen
- **Architecture**: Clean Architecture / Layered

## License

MIT
