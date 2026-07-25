# AGENTS.md

Agent guide for **Uang Kas Keluarga** — Go + PocketBase family finance backend.

---

## Commands

```bash
make serve          # go run main.go serve  (dev, auto-migration enabled)
make build          # compile binary → ./kas
make run            # build + serve
make dev            # serve with air auto-reload (requires: go install github.com/air-verse/air@latest)
make test           # go test -v ./...
make generate       # regenerate type-safe proxies after schema changes (2-step, see below)
make admin          # interactive admin account creation
make backup         # copies pb_data/ to pb_data_backup/pb_data_YYYYMMDD_HHMMSS/

# Migrations (also exposed via make migrate-*)
go run . migrate up
go run . migrate down 1
go run . migrate create "name"
go run . migrate collections   # snapshot current schema to migration file
go run . migrate history-sync
```

CI checks (must pass): `go mod tidy && git diff --exit-code`, `go vet ./...`, `make test`, `make build`.

### Pre-commit Hook (install via `make install-hooks`)

Runs on staged `.go` files only — rejects commit if any step fails:
1. `go mod tidy && git diff --exit-code go.mod go.sum`
2. `go vet ./...`
3. `staticcheck` (skips `kas/generated`, `kas/pbschema`)
4. `go build -o /dev/null .`

### GitHub Actions CI (`.github/workflows/ci.yml`)

| Trigger | Steps |
|---|---|
| PR to any branch | `checkout` → `setup-go` → `go mod download` → `make test` → `make build` |
| Push to `main`/`master` | Same as above |

Concurrency: per-branch, cancels in-progress.

### Release Pipeline (`.github/workflows/release.yml`)

`release: [created]` or `workflow_dispatch` → **ci-checks** → **release** (cross-compile 5 targets: linux amd64/arm64, windows amd64, darwin amd64/arm64) → **deploy** (webhook POST).

### Dependabot (`.github/dependabot.yml`)

Daily gomod + GH Actions updates. Auto-squash-merged after CI passes (unless `semver-major`).

---

## Architecture

```
Handler → Service → Repository → generated proxies → PocketBase (SQLite)
```

### Layer Diagram

```
┌─────────────────────────────────────────────────────────────┐
│  Handler Layer  (internal/handler/)                         │
│  Parse HTTP → Call Service → Return JSON                    │
│  Swall: parsing body/path/query, middleware binding          │
└──────────────────────┬──────────────────────────────────────┘
                       │ service.Interface
┌──────────────────────▼──────────────────────────────────────┐
│  Service Layer  (internal/service/)                         │
│  Business logic → Validation → Authorization                │
│  Swall: orchestrating multiple repos, atomic tx, cache inv  │
└──────────────────────┬──────────────────────────────────────┘
                       │ repository.Interface
┌──────────────────────▼──────────────────────────────────────┐
│  Repository Layer  (internal/repository/)                   │
│  DB access via generated proxies → DTO conversion           │
│  Swall: all PocketBase queries, ExpandRecord, raw SQL       │
└──────────────────────┬──────────────────────────────────────┘
                       │ generated.WrapRecord / NewProxy
┌──────────────────────▼──────────────────────────────────────┐
│  generated/  (auto-generated, DO NOT EDIT)                  │
│  Type-safe proxy structs + getters/setters + events         │
└──────────────────────┬──────────────────────────────────────┘
                       │ core.Record
┌──────────────────────▼──────────────────────────────────────┐
│  PocketBase (SQLite)                                        │
│  WAL mode, 256 MB mmap, custom pragmas in main.go:42-55     │
└─────────────────────────────────────────────────────────────┘
```

### Directory Layout

```
main.go                         # Entry point & DI wiring
├── internal/
│   ├── domain/                 # Pure DTOs, request/response structs, sentinel errors
│   │   ├── digiflazz/          # Digiflazz-specific DTOs & errors
│   │   ├── errors.go           # Business logic sentinel errors
│   │   ├── family.go           # Family DTOs
│   │   ├── family_member.go    # FamilyMember DTO
│   │   ├── transaction.go      # Transaction DTOs + expanded relation structs
│   │   └── report.go           # Monthly report & dashboard DTOs
│   ├── repository/             # Data access: interfaces + impl + recordToDTO
│   ├── service/                # Business logic: interfaces + impl
│   ├── handler/                # HTTP handlers: struct + RegisterRoutes + methods
│   ├── middleware/             # RequireAuth, RequireFamily, RequireFamilyOwner
│   ├── digiflazz/              # Digiflazz API client, types, signature, webhook
│   └── utils/                  # token.go, redaction.go, encryption.go (AES-256-GCM)
├── generated/                  # DO NOT EDIT — auto-generated proxies, utils, events, hooks
├── pbschema/template.go        # Editable schema template (edit before re-gen)
├── migrations/                 # Auto-created; applied automatically on serve
├── pb_data/                    # gitignored — SQLite database & storage
└── pb_public/                  # Embedded static files (OpenAPI spec, etc.)
```

---

## Generated Code — Critical Rules

`generated/` is produced by `pocketbase-gogen` (github.com/snonky/pocketbase-gogen). **Never edit it manually.**

```bash
make generate
# Step 1: pocketbase-gogen template ./pb_data ./pbschema/template.go
# Step 2: pocketbase-gogen generate ./pbschema/template.go ./generated/proxies.go --utils --hooks
# Run this every time PocketBase schema changes.
```

### 4 Generated Files

| File | Lines | Content | Purpose |
|---|---|---|---|
| `generated/proxies.go` | 1208 | Proxy structs + getters/setters + select enums | Type-safe record access |
| `generated/utils.go` | 152 | `NewProxy`, `WrapRecord`, `CName`, `Relations` map | Generic utilities |
| `generated/proxy_hooks.go` | 719 | Type aliases for typed event handlers | Hook event typing |
| `generated/proxy_events.go` | 238 | Generic event struct definitions | Event system |

### Type-Safe Access — ALWAYS use proxies

```go
// ✅ type-safe — compile-time checked
proxy, _ := generated.WrapRecord[generated.Transactions](record)
amount := proxy.Amount()       // float64
note   := proxy.Note()         // string

// ❌ runtime panic risk
amount := record.Get("amount").(float64)
```

### NewProxy vs WrapRecord

| Function | When to use | Utils file:line |
|---|---|---|
| `NewProxy[P](app)` | Creating a **new** record | `utils.go:63-73` |
| `WrapRecord[P](record)` | Wrapping an **existing** record | `utils.go:78-87` |

### Proxy Type Constraints (from `generated/utils.go:10-51`)

```go
type Proxy interface {
    Users | Families | FamilyMembers | Categories | Transactions | File | Jenis |
    DigiflazzCredentials | DigiflazzProducts | DigiflazzOrders | DigiflazzEvents
}
type ProxyP[P Proxy] interface {
    *P
    core.RecordProxy
    CollectionName() string
}
```

### All 11 Proxy Types

`Users`, `Families`, `FamilyMembers`, `Categories`, `Transactions`, `File`, `Jenis`,
`DigiflazzCredentials`, `DigiflazzProducts`, `DigiflazzOrders`, `DigiflazzEvents`

### pbschema/template.go — Editable Schema Template

`pbschema/template.go` is the **intermediate** file — PocketBase schema → template → proxies.

**What you can edit:**
- Rename struct (changes proxy type name)
- Rename field (add `// schema-name: original_name` to preserve DB mapping)
- Add custom methods to a struct (they carry through to proxies)
- Remove structs/fields (they won't be generated)

**⚠️ Backup before `make generate`** — `pocketbase-gogen template` overwrites template.go from pb_data.

---

## Expand Relations — Non-Obvious Gotchas

**Call `ExpandRecord`/`ExpandRecords` AFTER the query, not as a query param.**

```go
// ✅ correct
record, _ := app.FindRecordById("transactions", id)
app.ExpandRecord(record, []string{"category_id", "family_id"}, nil)

// ❌ WRONG — dbx.HashExp{"expand": "..."} as param does nothing
record, _ := app.FindRecordById("transactions", id, dbx.HashExp{"expand": "category_id"})
```

**Always nil-check expanded fields — accessing an unexpanded relation panics:**

```go
// ✅ safe
if family := proxy.FamilyId(); family != nil {
    name := family.Name()
}

// ❌ PANIC if not expanded
name := proxy.FamilyId().Name()
```

Use `app.ExpandRecords(records, fields, nil)` for slices (more efficient than per-record calls).

---

## Middleware — Binding Order & Context Contract

### Available Middleware

| Function | Signature | File:Line |
|---|---|---|
| `RequireAuth` | `func(e *core.RequestEvent) error` | `auth.go:55-60` |
| `RequireFamily` | `func(repo FamilyMemberRepository) func(*core.RequestEvent) error` | `auth.go:65-98` |
| `RequireFamilyOwner` | `func() func(*core.RequestEvent) error` | `family_role.go:65-86` |
| `InvalidateFamily` | `func(userID string)` | `auth.go:102-106` |
| `IsFamilyOwner` | `func(app core.App, familyID, userID string) (bool, error)` | `family_role.go:13-34` |
| `GetFamilyRole` | `func(app core.App, familyID, userID string) (string, error)` | `family_role.go:37-62` |

### Non-Obvious Rules

- **Binding order matters**: `RequireFamily` returns 500 if `e.Auth == nil` — always bind after `RequireAuth`.
- **`RequireFamilyOwner`** must also be chained after `RequireAuth` + `RequireFamily`.
- `RequireFamily` caches membership per user for **5 minutes** (`const familyCacheTTL = 5 * time.Minute`, `auth.go:23`) and injects `familyID` into `e.Request.Context()`.
- Handlers do **not** receive `familyID` directly — they call `middleware.GetFamilyIDFromContext(e.Request.Context())`.
- `InvalidateFamily(userID)` **must** be called after family join/leave/create to flush the cache. It is injected into `FamilyService` in `main.go`.
- Role cache is separate (`family_role.go`). Use `middleware.InvalidateFamily(userID)` in tests that mutate membership.
- Middleware is passed as **constructor args** (not global). `main.go` creates `requireFamily := middleware.RequireFamily(familyMemberRepo)` and passes it to each handler constructor.
- `TransactionHandler.Update/Delete` only bind `RequireAuth`; ownership is enforced in the service layer.

### Middleware Chain Diagram

```
1. RequireAuth (auth.go:55)
   ├── e.Auth == nil? → 401 UNAUTHORIZED
   └── e.Next()

2. RequireFamily (auth.go:65)
   ├── Cache hit (5-min TTL)?
   │     └── inject familyID to context → e.Next()
   ├── Cache miss?
   │     ├── repo.GetByUserID(userID)
   │     ├── Error → 500
   │     ├── No membership → 403 FORBIDDEN
   │     └── Has membership → cache + inject → e.Next()
   └── e.Auth == nil? (misconfig) → 500

3. RequireFamilyOwner (family_role.go:65)
   ├── familyID from context?
   │     └── IsFamilyOwner(app, familyID, userID)
   │           ├── Error → 500
   │           └── Not owner → 403 FORBIDDEN
   └── e.Next()

4. Handler Method
   ├── middleware.GetFamilyIDFromContext(ctx)
   ├── Parse request body / params
   ├── Call service
   └── Return JSON response
```

### Per-route Middleware Summary

| Route Pattern | Auth | Family | Owner | Handler |
|---|---|---|---|---|
| `/api/transactions` (POST/GET) | ✅ | ✅ | ❌ | `TransactionHandler` |
| `/api/transactions/{id}` (PATCH/DELETE) | ✅ | ❌ | ❌ | `TransactionHandler` |
| `/api/transactions?start=...&end=...` | ✅ | ✅ | ❌ | `TransactionHandler` |
| `/api/families/{familyId}/*` | ✅ | ✅ | ❌ | `TransactionHandler` |
| `/api/reports/*` | ✅ | ✅ | ❌ | `ReportHandler` |
| `/api/families` (POST/join/leave) | ✅ | ❌ | ❌ | `FamilyHandler` |
| `/api/digiflazz/credential` (GET) | ✅ | ✅ | ❌ | `DigiflazzCredentialHandler` |
| `/api/digiflazz/credential` (POST/DELETE) | ✅ | ✅ | ✅ | `DigiflazzCredentialHandler` |
| `/api/digiflazz/orders/*` | ✅ | ✅ | ❌ | `DigiflazzOrderHandler` |
| `/api/digiflazz/products` (GET) | ✅ | ✅ | ❌ | `DigiflazzProductHandler` |
| `/api/digiflazz/products/sync` | ✅ | ✅ | ✅ | `DigiflazzProductHandler` |
| `/webhooks/digiflazz/{token}` | ❌ | ❌ | ❌ | `DigiflazzWebhookHandler` |

### Context Key Pattern

```go
// context.go:6-23
type contextKey string
const familyIDKey contextKey = "familyID"

func SetFamilyIDToContext(ctx context.Context, familyID string) context.Context
func GetFamilyIDFromContext(ctx context.Context) (string, bool)
```

Uses unexported `contextKey` type to prevent key collisions with other packages.

---

## Migrations

### 5 Timing Points — When Migrations Run

| # | Timing | What runs | File (PocketBase source) |
|---|---|---|---|
| 1 | `Bootstrap()` → `RunSystemMigrations()` | **System migrations only** (PocketBase internal) | `core/base.go:L391-L442` |
| 2 | `PocketBase.Execute()` → calls `Bootstrap()` | Auto-bootstrap (skipped for `--help`/`--version`) | `pocketbase.go:L179-L184` |
| 3 | `apis.Serve()` → `RunAllMigrations()` | **System + App migrations** (main trigger) | `apis/serve.go:L61-L70` |
| 4 | CLI `migrate up` | System + App (manual) | `plugins/migratecmd/migratecmd.go:L126-L136` |
| 5 | **Auto-migration** via Dashboard (dev only) | Creates new migration file + marks applied | `plugins/migratecmd/automigrate.go:L18-L96` |

### Dev vs Production

| Aspect | Dev (`go run`) | Production (built binary) |
|---|---|---|
| **Auto-migration** | ✅ Enabled — Dashboard changes auto-create migration files | ❌ Disabled |
| **Migration execution at startup** | ✅ Timing #3 (RunAllMigrations) | ✅ Same |
| **CLI `migrate up`** | ✅ Available | ✅ Available |
| **Detection** | `osutils.IsProbablyGoRun()` → true | `osutils.IsProbablyGoRun()` → false |

### Migration File Pattern

All 19 files in `migrations/` use identical structure:

```go
package migrations

import (
    "github.com/pocketbase/pocketbase/core"
    m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
    m.Register(func(app core.App) error {
        // UP migration logic
        return nil
    }, func(app core.App) error {
        // DOWN (revert) logic
        return nil
    })
}
```

3 types of migration:
- **Collection Snapshot** — `app.ImportCollectionsByMarshaledJSON([]byte(jsonData), false)` (extend mode)
- **Raw SQL/DML** — `app.DB().NewQuery(...)`
- **Field Add/Edit** — `collection.Fields.AddMarshaledJSONAt(...)` + `app.Save(collection)`

Migrations execute in **timestamp order** (numeric filename prefix).

---

## Repository Layer — Patterns

### Standard Structure (4 components)

Every repository follows this exact pattern:

```go
// 1. Interface (exported)
type XxxRepository interface {
    Create(...) (*domain.XxxDTO, error)
    GetByID(id string) (*domain.XxxDTO, error)
    // ...
}

// 2. Struct (unexported)
type xxxRepo struct {
    app core.App
}

// 3. Constructor (returns interface)
func NewXxxRepository(app core.App) XxxRepository {
    return &xxxRepo{app: app}
}

// 4. Private converter (core.Record → domain.DTO)
func (r *xxxRepo) recordToDTO(record *core.Record) (*domain.XxxDTO, error) {
    proxy, _ := generated.WrapRecord[generated.Xxx](record)
    return &domain.XxxDTO{...}, nil
}
```

### 3 Query Patterns

```go
// 1. FindRecordById — single record by PK
record, err := r.app.FindRecordById("transactions", id)

// 2. FindRecordsByFilter — filter + sort + pagination
records, err := r.app.FindRecordsByFilter(
    "transactions",
    "family_id = {:familyID} && date >= {:startDate}",
    "-created",           // sort: "-field" = DESC
    limit, offset,
    map[string]any{"familyID": familyID},
)

// 3. Raw SQL — for JOIN, aggregation, materialized tables
err := r.app.DB().NewQuery(`
    SELECT COALESCE(SUM(CASE WHEN type='income' THEN amount ELSE 0 END), 0)
    FROM transactions WHERE family_id = {:familyID}
`).Bind(map[string]any{"familyID": familyID}).Row(&result)
```

### recordToDTO — Proxy Nil-Check Pattern

```go
func (r *transactionRepo) recordToDTO(record *core.Record) (*domain.TransactionDTO, error) {
    proxy, err := generated.WrapRecord[generated.Transactions](record)

    // Direct fields (type-safe)
    dto := &domain.TransactionDTO{
        Amount: proxy.Amount(),
        Note:   proxy.Note(),
    }

    // Expanded relations — MUST nil-check
    if family := proxy.FamilyId(); family != nil {
        dto.FamilyID = family.Id
        dto.Family = &domain.FamilyExpand{
            ID:   family.Id,
            Name: family.Name(),
        }
    }
    return dto, nil
}
```

### Common Patterns

| Pattern | Example file | Lines |
|---|---|---|
| Create via `core.NewRecord` + `record.Set` | `transaction_repository.go` | 74-97 |
| Create via `NewProxy` + proxy setter | `family_repository.go` | 34-54 |
| Upsert (create or update) | `digiflazz_product_repository.go` | 50-103 |
| Partial update (optional fields) | `digiflazz_credential_repository.go` | 228-275 |
| Delete by finding first | `family_member_repository.go` | 97-113 |
| Bulk delete | `digiflazz_product_repository.go` | 194-215 |
| Transaction (`RunInTransaction`) | `category_repository.go` | 88-129 |
| Pagination with total count | `transaction_repository.go` | 168-214 |
| Dynamic filter construction | `digiflazz_product_repository.go` | 106-170 |
| Dual converter (DTO + full Record) | `digiflazz_credential_repository.go` | 328-361 |

---

## Service Layer — Patterns

### Standard Structure

```go
// 1. Interface (exported)
type XxxService interface {
    Create(req *domain.CreateXxxRequest, userID string) (*domain.XxxDTO, error)
}

// 2. Struct (unexported) with repository interfaces
type xxxService struct {
    xxxRepo repository.XxxRepository
    // core.App only when RunInTransaction is needed
    // func(string) only when cache invalidation is needed
}

// 3. Constructor (returns interface)
func NewXxxService(xxxRepo repository.XxxRepository) XxxService {
    return &xxxService{xxxRepo: xxxRepo}
}
```

### Business Logic Flow

```
1. Validate inputs (amount > 0, type is valid, date parsable)
2. Check authorization (is creator? is owner?)
3. Validate cross-entity constraints (category belongs to family?)
4. Call repository method(s)
5. Invalidate middleware cache if membership changed
6. Return DTO or error (wrapped with context)
```

### Authorization Patterns

| Scenario | Check location | Pattern |
|---|---|---|
| Creator-only (update/delete) | Service layer | `repo.GetCreatorID(id)` → compare with `userID` |
| Family membership | Middleware | `RequireFamily` (cached) |
| Owner-only (Digiflazz) | Middleware | `RequireFamilyOwner` |
| Owner-only (family ops) | Handler | Endpoint only accessible by auth path |
| Cross-family resource | Handler | Compare `dto.FamilyID` with `familyID` from context |

### Atomic Transaction

```go
err := s.app.RunInTransaction(func(txApp core.App) error {
    dto, err := s.familyRepo.Create(txApp, name, inviteCode)
    if err := s.familyMemberRepo.CreateMember(txApp, dto.ID, userID, "owner")
    return nil  // commit
})
```

---

## Handler Layer — Patterns

### Standard Structure

```go
type XxxHandler struct {
    service       service.XxxService
    requireAuth   *hook.Handler[*core.RequestEvent]
    requireFamily *hook.Handler[*core.RequestEvent]
}

func NewXxxHandler(
    service       service.XxxService,
    requireAuth   func(*core.RequestEvent) error,
    requireFamily func(*core.RequestEvent) error,
) *XxxHandler {
    return &XxxHandler{
        service:       service,
        requireAuth:   &hook.Handler[*core.RequestEvent]{Func: requireAuth},
        requireFamily: &hook.Handler[*core.RequestEvent]{Func: requireFamily},
    }
}
```

### RegisterRoutes

```go
func (h *XxxHandler) RegisterRoutes(e *core.ServeEvent) {
    e.Router.POST("/api/xxx", h.Create).Bind(h.requireAuth).Bind(h.requireFamily)
    e.Router.PATCH("/api/xxx/{id}", h.Update).Bind(h.requireAuth) // no family: owner check in service
}
```

### Handler Method Flow

```go
func (h *XxxHandler) Create(e *core.RequestEvent) error {
    // 1. Extract family from context
    familyID, ok := middleware.GetFamilyIDFromContext(e.Request.Context())
    if !ok { return e.InternalServerError("Family context not found", nil) }

    // 2. Parse request body
    var req domain.CreateXxxRequest
    if err := e.BindBody(&req); err != nil {
        return e.BadRequestError("Invalid request body", err)
    }

    // 3. Call service
    result, err := h.service.Create(&req, e.Auth.Id, familyID)
    if err != nil { return e.BadRequestError("Failed", err) }

    // 4. Return JSON
    return e.JSON(http.StatusCreated, result)
}
```

### Error Response Methods (on `*core.RequestEvent`)

| Method | Status | When |
|---|---|---|
| `e.BadRequestError(msg, err)` | 400 | Invalid input |
| `e.UnauthorizedError(msg, err)` | 401 | Missing/invalid auth |
| `e.ForbiddenError(msg, err)` | 403 | Not member/not owner |
| `e.NotFoundError(msg, err)` | 404 | Resource not found |
| `e.InternalServerError(msg, err)` | 500 | Unexpected/misconfig |

### Error Mapping for Complex Handlers

Group specific error → HTTP error mapping in a separate function:

```go
func mapXxxError(e *core.RequestEvent, err error) error {
    if errors.Is(err, domain.ErrXxx) {
        return e.NotFoundError("Xxx not found", err)
    }
    if strings.Contains(err.Error(), "unauthorized") {
        return e.ForbiddenError("Access denied", err)
    }
    return e.InternalServerError("Internal error", err)
}
```

### Pagination

```go
// pagination.go:17-35 — extracts page/page_size from query
func ParsePagination(query url.Values) (page, pageSize int)
// Accepts: page_size (canonical), pageSize, per_page (fallbacks)
// Defaults: page=1, pageSize=20, maxPageSize=100
```

---

## Business Logic — Core Features

### 1. Materialized Balance (`family_balances` table)

**Balance is NOT calculated on the fly.** It is stored in `family_balances` and updated via PocketBase hooks:

| Event | Action | Logic |
|---|---|---|
| **Family created** | INSERT (balance=0) | `main.go:126-136` |
| **Transaction created** | Delta UPDATE | Income: +amount; Expense: -amount `main.go:141-168` |
| **Transaction updated** | Full recalc | SUM from transactions table `main.go:171-201` |
| **Transaction deleted** | Reverse delta | Negate original delta `main.go:204-231` |
| **Read balance** | O(1) lookup from `family_balances` | Fallback to SUM query if table missing `transaction_repo.go:266-285` |

### 2. Transaction Types

- `income` — money coming in (salary, gift, etc.)
- `expense` — money going out (food, bills, etc.)

### 3. Family & Membership

- Each user can belong to exactly **one** family
- Two roles: `owner` (creator, can manage Digiflazz) | `member` (can view/create transactions)
- Join via 8-char **cryptographically random** invite code (`generateInviteCode()` in `family_service.go:43`)
- Family is created + membership in **single transaction**
- Cache invalidation (`InvalidateFamily`) called after create/join/leave

### 4. Master Categories

When a family is created, 13 master categories are **cloned** from templates (where `is_master=true && family_id=''`) into the new family. This happens atomically in `categoryRepo.SeedMasterCategories`.

### 5. Digiflazz Integration

| Feature | Flow |
|---|---|
| **Credential setup** | Owner sets username + API key → validated via `CekSaldo` → encrypted with AES-256-GCM → stored |
| **Webhook token** | Generated on credential create, stored as SHA-256 hash, returned plaintext only once |
| **Product sync** | Cron every 30 min OR manual. Fetches prepaid + postpaid pricelist, upserts per-family |
| **Prepaid order** | Validate product → check balance → `Topup` API → store order → poll/finalize |
| **Postpaid order** | Inquiry → Pay (separate step, validates amount hasn't changed) |
| **Webhook** | `POST /webhooks/digiflazz/{token}` → resolve credential by hash → verify HMAC-SHA1 signature → dedup by payload hash → apply status transition → finalize if success |
| **Finalize** | On success → auto-create expense transaction for the selling price → link to order |
| **Order status** | Inquiry → Pending → Processing → Success/Failed/Canceled (terminal states) |
| **RC code mapping** | `"00"`=success, `"03"`=pending, `"44"`=insufficient balance, `"51/52/54"`=invalid customer, etc. (`errors.go:57-107`) |

---

## Testing Patterns

### 3 Testing Levels

| Level | What | Test pattern | Example file |
|---|---|---|---|
| **Unit** | Service layer in isolation | Table-driven + mock structs with `Fn` fields | `transaction_service_test.go` |
| **Integration** | Repository/Service with real DB | `tests.NewTestApp()` + `_ "kas/migrations"` | `digiflazz_credential_service_test.go` |
| **API-level** | Full HTTP round-trip | `tests.ApiScenario{}` + `BeforeTestFunc` | `transaction_handler_integration_test.go` |

### Mock Struct Pattern

```go
type mockXxxRepo struct {
    createFn  func(...) (*domain.XxxDTO, error)
    getByIDFn func(id string) (*domain.XxxDTO, error)
}

func (m *mockXxxRepo) Create(..., userID, familyID string) (*domain.XxxDTO, error) {
    if m.createFn != nil { return m.createFn(...) }
    return nil, nil
}
```

### ApiScenario Pattern

```go
(&tests.ApiScenario{
    Name:   "description",
    Method: http.MethodGet,
    URL:    "/api/xxx",
    Headers: map[string]string{
        "Authorization": "Bearer " + token,
    },
    ExpectedStatus:  http.StatusOK,
    ExpectedContent: []string{`"field"`},
    TestAppFactory:  newTestApp,
    BeforeTestFunc: func(t testing.TB, app *tests.TestApp, e *core.ServeEvent) {
        bindRoutes(app, e)
    },
}).Test(t)
```

### Fixture Pattern (Complex Integration Tests)

```go
type xxxFixture struct {
    app      *tests.TestApp
    familyID string
    ownerID  string
    repo     repository.XxxRepository
    svc      XxxService
}
func setupFixture(t testing.TB) *xxxFixture { ... }
```

### Key Gotchas

- **No local `tests/` package** — `tests.NewTestApp()` and `tests.ApiScenario` come from `github.com/pocketbase/pocketbase/tests`
- **Always import `_ "kas/migrations"`** in integration test files
- **`t.Setenv("DIGIFLAZZ_CREDENTIAL_ENCRYPTION_KEY", "...")`** required for Digiflazz service tests
- **Clear cache in tests** that mutate membership: `middleware.InvalidateFamily(userID)` or private `resetGlobalFamilyCache(t)`
- **`t.Cleanup(app.Cleanup)`** is an alternative to `defer app.Cleanup()`
- **No assertion library** — use `t.Errorf`/`t.Fatalf` directly

---

## Data Flow — Complete Example (Create Transaction)

```
HTTP POST /api/transactions
Body: {"category_id":"cat1","type":"expense","amount":50000,"note":"belanja","date":"2026-06-29T10:00:00Z"}
──────────────────────────────────────────────────────────────────────

HANDLER (transaction_handler.go:109-126)
├── RequireAuth → cek e.Auth
├── RequireFamily → cek cache → DB → inject familyID ke context
├── GetFamilyIDFromContext(ctx)
├── e.BindBody(&req) → domain.CreateTransactionRequest
└── h.service.CreateTransaction(&req, e.Auth.Id, familyID)

SERVICE (transaction_service.go:37-67)
├── Validasi: amount > 0, type income|expense, date ISO 8601
├── Validasi: category exists & belongs to family
└── s.transactionRepo.Create(req, userID, familyID)

REPOSITORY (transaction_repository.go:74-97)
├── app.FindCachedCollectionByNameOrId("transactions")
├── core.NewRecord(collection)
├── record.Set("field", value) ×6
├── app.Save(record)
└── r.recordToDTO(record)
      ├── generated.WrapRecord[generated.Transactions](record)
      ├── proxy.Amount(), proxy.Note(), etc.
      └── nil-check expanded relations

BACK TO HANDLER
└── e.JSON(http.StatusCreated, transactionDTO)

POCKETBASE HOOK (main.go:141-168)
└── OnTransactionsAfterCreateSuccess
      └── Delta-update family_balances
```

---

## Adding a New Feature — Step by Step

### Prerequisites: Understanding the Codebase

Before starting, familiarize yourself with the layer architecture. Read the existing feature implementations:

| For reference | File |
|---|---|
| Simple CRUD (best template) | `internal/service/transaction_service.go` + `internal/handler/transaction_handler.go` |
| Simple repo (no expand needed) | `internal/repository/family_repository.go` |
| Atomic transaction | `internal/service/family_service.go:76-88` |
| Partial update with pointer fields | `internal/repository/digiflazz_credential_repository.go:228-275` |
| Full integration test pattern | `internal/handler/transaction_handler_integration_test.go` |
| Unit test pattern | `internal/service/transaction_service_test.go` |

### Steps

1. **`internal/domain/`** — Define DTO structs, request/response structs, sentinel errors
   - Swall: `TransactionDTO`, `CreateTransactionRequest`, `UpdateTransactionRequest`
   - Package: `domain`. No external dependencies.
   - Add sentinel errors to `errors.go` if needed.

2. **`internal/repository/`** — Define interface + struct + constructor + `recordToDTO`
   - Interface: exported method signatures returning domain DTOs
   - Struct: `type xxxRepo struct { app core.App }` (unexported)
   - Constructor: `func NewXxxRepository(app core.App) XxxRepository` (returns interface)
   - recordToDTO: `generated.WrapRecord[generated.Xxx](record)` + nil-check expanded relations
   - Query: `FindRecordById`/`FindRecordsByFilter`/`FindFirstRecordByFilter`/`Raw SQL`

3. **`internal/service/`** — Define interface + struct + constructor + business logic
   - Interface: exported methods returning domain DTOs
   - Struct: repository interfaces + optional `core.App` + optional `func(string)`
   - Constructor: `func NewXxxService(repo1, repo2, ...) XxxService`
   - Validate → authorize → call repository → return

4. **`internal/handler/`** — Define struct + constructor + `RegisterRoutes` + handler methods
   - Struct: service interface + `*hook.Handler[*core.RequestEvent]` for each middleware
   - Constructor: accepts service interface + middleware functions
   - `RegisterRoutes(e *core.ServeEvent)`: `e.Router.POST("/api/xxx", h.Method).Bind(h.mw)`
   - Handler methods: parse request → call service → return JSON

5. **`main.go`** — Wire dependencies
   ```go
   // Instantiate repository (butuh core.App)
   xxxRepo := repository.NewXxxRepository(app)

   // Instantiate service (butuh repo interfaces)
   xxxService := service.NewXxxService(xxxRepo)

   // Instantiate handler (butuh service + middleware functions)
   xxxHandler := handler.NewXxxHandler(xxxService, middleware.RequireAuth, requireFamily)

   // Register routes (dalam app.OnServe().BindFunc)
   xxxHandler.RegisterRoutes(se)
   ```

    **Wiring order**: `repositories (core.App)` → `middleware (repo)` → `services (repo interfaces)` → `handlers (service interfaces + middleware funcs)` → `RegisterRoutes (inside OnServe().BindFunc)`

---

## Key Gotchas

- **Module name is `kas`** — all internal imports are `kas/internal/...`, `kas/generated`, etc.
- `pb_data/` is gitignored. `generated/` is NOT gitignored (committed).
- `pb_public/` is embedded into the binary at build time (`//go:embed pb_public`).
- SQLite is tuned with custom pragmas in `main.go` (WAL, 256 MB mmap, etc.) — do not change without understanding the implications.
- On family creation, `categoryRepo.SeedMasterCategories` is called via `OnRecordAfterCreateSuccess("families")` hook — wired in `main.go`.
- Balance updates on transaction CRUD are done via hooks on `OnTransactionsAfterCreateSuccess/UpdateSuccess/DeleteSuccess` in `main.go`.
- Digiflazz webhook auth: resolves credential by `sha256(token)` hash, then validates `X-Digiflazz-Signature` as `sha1=<hex>` via HMAC-SHA1 over the raw body. Empty secret skips validation.
- Digiflazz product repo: free-text search uses the `~` operator on `product_name`/`buyer_sku_code`; default limit=50, hard cap 200.
- PocketBase Admin UI: http://localhost:8090/_/
- `godoc-mcp` is configured in `opencode.jsonc` — use it when looking up PocketBase or stdlib Go docs.
- **No local `tests/` package** — `tests.NewTestApp()` comes from `github.com/pocketbase/pocketbase/tests`.
- **Clear family cache after test mutations** — calls `middleware.InvalidateFamily(userID)` or private `resetGlobalFamilyCache()` in middleware tests.
- **No assertion library** — standard `t.Errorf`/`t.Fatalf` for test assertions.
- **`t.Setenv`** auto-restores environment variables after test (Go 1.17+).
