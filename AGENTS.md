# AGENTS.md

Agent guide for **Uang Kas Keluarga** — Go + PocketBase family finance backend.

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

## Architecture

```
Handler → Service → Repository → generated proxies → PocketBase (SQLite)
```

- `main.go` — only place for dependency injection; wire repo→service→handler→`RegisterRoutes(se)`
- `internal/domain/` — pure DTOs and request/response models; no external deps
- `internal/repository/` — all DB access; use generated proxies here only
- `internal/service/` — business logic and authorization
- `internal/handler/` — thin HTTP layer; parse, call service, return JSON
- `internal/middleware/` — `RequireAuth`, `RequireFamily`, `RequireFamilyOwner`, `InvalidateFamily`
- `internal/digiflazz/` — external Digiflazz API client, types, webhook, HMAC-SHA1 signature
- `internal/utils/` — `token.go`, `redaction.go`, `encryption.go` (AES-256-GCM helpers)
- `generated/` — **DO NOT EDIT** (auto-generated; IS committed to git)
- `pbschema/template.go` — editable schema template (edit this to customise codegen)
- `migrations/` — auto-created during dev (`go run`); applied automatically on serve

## Generated Code — Critical Rules

`generated/` is produced by `pocketbase-gogen`. **Never edit it manually.**

```bash
make generate
# Step 1: pocketbase-gogen template ./pb_data ./pbschema/template.go
# Step 2: pocketbase-gogen generate ./pbschema/template.go ./generated/proxies.go --utils --hooks
# Run this every time PocketBase schema changes.
```

**Always use proxies in repository layer:**
```go
// ✅ type-safe
proxy, _ := generated.WrapRecord[generated.Transactions](record)
amount := proxy.Amount()  // float64

// ❌ runtime panic risk
amount := record.Get("amount").(float64)
```

Utility signatures from `generated/utils.go`:
```go
func NewProxy[P Proxy, PP ProxyP[P]](app core.App) (PP, error)
func WrapRecord[P Proxy, PP ProxyP[P]](record *core.Record) (PP, error)
```

**All 11 proxy types** (from `generated/proxies.go`):
`Users`, `Families`, `FamilyMembers`, `Categories`, `Transactions`, `File`, `Jenis`,
`DigiflazzCredentials`, `DigiflazzProducts`, `DigiflazzOrders`, `DigiflazzEvents`

`pbschema/template.go` currently has no custom methods — safe to regenerate without losing handwritten logic.

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

## Middleware — Binding Order & Context Contract

Available middleware:

| Function | Signature |
|---|---|
| `RequireAuth` | `func(e *core.RequestEvent) error` |
| `RequireFamily` | `func(repo FamilyMemberRepository) func(*core.RequestEvent) error` |
| `RequireFamilyOwner` | `func() func(*core.RequestEvent) error` |
| `InvalidateFamily` | `func(userID string)` |
| `IsFamilyOwner` | `func(app core.App, familyID, userID string) (bool, error)` |
| `GetFamilyRole` | `func(app core.App, familyID, userID string) (string, error)` |

Non-obvious rules:

- **Binding order matters**: `RequireFamily` returns 500 if `e.Auth == nil` — always bind after `RequireAuth`.
- **`RequireFamilyOwner`** must also be chained after `RequireAuth` + `RequireFamily`.
- `RequireFamily` caches membership per user for **5 minutes** and injects `familyID` into `e.Request.Context()`.
- Handlers do **not** receive `familyID` directly — they call `middleware.GetFamilyIDFromContext(e.Request.Context())`.
- `InvalidateFamily(userID)` **must** be called after family join/leave/create to flush the cache. It is injected into `FamilyService` in `main.go`.
- Role cache is separate (`family_role.go`). Use `middleware.ClearRoleCache()` in tests that mutate membership.
- Middleware is passed as constructor args (not global). `main.go` creates `requireFamily := middleware.RequireFamily(familyMemberRepo)` and passes it to each handler constructor.
- `TransactionHandler.Update/Delete` only bind `RequireAuth`; ownership is enforced in the service layer.

## Migrations

- **Auto-migration** is on when running via `go run` (`osutils.IsProbablyGoRun()`). Admin Dashboard collection changes auto-create migration files in `migrations/`.
- For production, run `go run . migrate up` — automigrate is disabled in built binary.
- Initial superuser migration reads `ADMIN_EMAIL` / `ADMIN_PASSWORD` env vars. Defaults: `admin@example.com` / `admin123456` — **change in production**.

## Environment Variables

`.env` is optional (gitignored). Key vars:

```bash
PB_PORT=8090
PB_HOST=0.0.0.0

# Initial migration superuser
ADMIN_EMAIL=your@email.com
ADMIN_PASSWORD=yourpassword

# Digiflazz integration
DIGIFLAZZ_CREDENTIAL_ENCRYPTION_KEY=<key>   # any non-empty string; derived key is always 32 bytes
DIGIFLAZZ_PRICE_SYNC_INTERVAL=*/30 * * * *  # default: every 30 min
DIGIFLAZZ_ORDER_POLL_INTERVAL=*/5 * * * *   # default: every 5 min
```

## Testing

Test styles in this repo:
- **Unit**: table-driven, file-local mocks/fakes, no DB — `internal/utils/`, `internal/digiflazz/` tests
- **Integration**: `tests.NewTestApp()` + `_ "kas/migrations"` import for PocketBase-backed tests
- **API-level**: `tests.ApiScenario` in `internal/handler/`

Gotchas:
- Digiflazz service integration tests require `t.Setenv("DIGIFLAZZ_CREDENTIAL_ENCRYPTION_KEY", "...")`.
- `internal/service/digiflazz_smartfren_test.go` bootstraps against the real `pb_data/` directory — slow and brittle; do not rely on it in CI.
- `RequireFamily` middleware cannot be fully unit-mocked; use integration tests with a real PocketBase request event for coverage.
- Tests that mutate family membership must call `middleware.ClearRoleCache()` to avoid cache poisoning between cases.
- No shared `TestMain` or test-helper package; helpers are file-local.

## Adding a New Feature

1. `internal/domain/` — add DTO + request/response structs
2. `internal/repository/` — add interface + implementation using generated proxies
3. `internal/service/` — add interface + implementation with business logic
4. `internal/handler/` — add handler struct, `RegisterRoutes(se *core.ServeEvent)`
5. `main.go` — wire: `repo → service → handler`, call `handler.RegisterRoutes(se)` inside `OnServe().BindFunc`

## Key Gotchas

- **Module name is `kas`** — all internal imports are `kas/internal/...`, `kas/generated`, etc.
- `pb_data/` is gitignored. `generated/` is NOT gitignored (committed).
- `pb_public/` is embedded into the binary at build time (`//go:embed pb_public`).
- SQLite is tuned with custom pragmas in `main.go` (WAL, 256 MB mmap, etc.) — do not change without understanding the implications.
- On family creation, `categoryRepo.SeedMasterCategories` is called via `OnRecordAfterCreateSuccess("families")` hook — wired in `main.go`.
- Digiflazz webhook auth: resolves credential by `sha256(token)` hash, then validates `X-Digiflazz-Signature` as `sha1=<hex>` via HMAC-SHA1 over the raw body. Empty secret skips validation.
- Digiflazz product repo: free-text search uses the `~` operator on `product_name`/`buyer_sku_code`; default limit=50, hard cap 200.
- PocketBase Admin UI: http://localhost:8090/_/
- `godoc-mcp` is configured in `opencode.jsonc` — use it when looking up PocketBase or stdlib Go docs.
