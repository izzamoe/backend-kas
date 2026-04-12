# Project Structure

## Directory Tree

```
uang-kas-keluarga/
├── main.go                          # Entry point & DI container
├── go.mod                           # Go modules
├── Makefile                         # Build commands
├── README.md                        # Documentation
│
├── pbschema/                        # Editable schema templates
│   └── template.go                  # PocketBase schema as code
│
├── generated/                       # Auto-generated (DO NOT EDIT)
│   ├── proxies.go                   # Type-safe proxy structs
│   ├── proxy_events.go              # Event definitions
│   ├── proxy_hooks.go               # PocketBase lifecycle hooks
│   └── utils.go                     # Helper functions (WrapRecord, etc)
│
├── internal/                        # Application code
│   ├── domain/                      # Business models
│   │   ├── transaction.go           # DTOs & request/response models
│   │   ├── report.go                # MonthlyReportDTO, CategoryBreakdownDTO, DashboardSummaryDTO
│   │   ├── family.go                # Family DTOs
│   │   └── family_member.go         # FamilyMember DTOs
│   │
│   ├── repository/                  # Data access layer
│   │   ├── transaction_repository.go # DB operations using generated proxies
│   │   ├── category_repository.go   # Category DB operations
│   │   ├── family_repository.go     # Family DB operations
│   │   └── family_member_repository.go # FamilyMember DB operations
│   │
│   ├── service/                     # Business logic layer
│   │   ├── transaction_service.go   # Business rules & validation
│   │   ├── report_service.go        # Monthly report & dashboard summary logic
│   │   └── family_service.go        # Family business logic
│   │
│   ├── handler/                     # HTTP handlers
│   │   ├── transaction_handler.go   # Transaction REST API endpoints
│   │   ├── report_handler.go        # Report REST API endpoints
│   │   └── family_handler.go        # Family REST API endpoints
│   │
│   ├── middleware/                  # HTTP middleware
│   │   ├── auth.go                  # Authentication middleware
│   │   └── context.go               # Context helpers (family_id, etc)
│   │
│   └── utils/                       # Internal utilities
│
└── pb_data/                         # PocketBase database (ignored in git)
    ├── data.db                      # SQLite database
    └── ...
```

## Files Created

### Core Application
- `main.go` - Application entry point with dependency injection
- `Makefile` - Build automation and shortcuts

### Domain Layer (Business Models)
- `internal/domain/transaction.go`
  - `TransactionDTO` - Data transfer object
  - `CreateTransactionRequest` - Request model
  - `UpdateTransactionRequest` - Update model
- `internal/domain/report.go`
  - `MonthlyReportDTO` - Monthly report with `expense_breakdown` & `income_breakdown`
  - `CategoryBreakdownDTO` - Per-category aggregation (amount + count)
  - `DashboardSummaryDTO` - Balance + monthly stats with % change
  - `MonthlyReportRequest` / `DashboardSummaryRequest` - Request models
- `internal/domain/family.go` - Family DTOs
- `internal/domain/family_member.go` - FamilyMember DTOs

### Repository Layer (Data Access)
- `internal/repository/transaction_repository.go`
  - `TransactionRepository` - Interface
  - `transactionRepo` - Implementation using generated proxies
  - CRUD operations + `GetMonthlyReportData` (expense & income breakdown by category) + `GetDashboardData`
  - **Uses generated proxies for type-safety**
- `internal/repository/category_repository.go` - Category data access
- `internal/repository/family_repository.go` - Family data access
- `internal/repository/family_member_repository.go` - FamilyMember data access

### Service Layer (Business Logic)
- `internal/service/transaction_service.go`
  - `TransactionService` - Interface
  - `transactionService` - Implementation
  - Business validation + authorization logic
- `internal/service/report_service.go`
  - `ReportService` - Interface
  - `reportService` - Implementation
  - `GetMonthlyReport` - Aggregates income & expense totals + per-category breakdowns
  - `GetDashboardSummary` - Overall balance + monthly stats with % change vs prev month
- `internal/service/family_service.go` - Family business logic

### Handler Layer (HTTP/API)
- `internal/handler/transaction_handler.go`
  - `TransactionHandler` - HTTP request handlers
  - REST API endpoints for transactions
- `internal/handler/report_handler.go`
  - `ReportHandler` - HTTP request handlers
  - `GET /api/reports/monthly` - Monthly report (income & expense per category)
  - `GET /api/reports/summary` - Dashboard summary
- `internal/handler/family_handler.go` - Family HTTP handlers

### Middleware
- `internal/middleware/auth.go` - Authentication & authorization middleware
- `internal/middleware/context.go` - Context helpers (e.g. `GetFamilyIDFromContext`)

### Generated Code (by pocketbase-gogen)
- `pbschema/template.go` - Editable schema representation
- `generated/proxies.go` - Type-safe proxy structs
- `generated/utils.go` - `WrapRecord[T]`, `NewProxy[T]` helpers
- `generated/proxy_events.go` - Event definitions
- `generated/proxy_hooks.go` - PocketBase hooks

## Data Flow

```
HTTP Request
    ↓
Handler (parse & validate request)
    ↓
Service (business logic)
    ↓
Repository (data access with generated proxies)
    ↓
PocketBase → SQLite
```

## Example: Create Transaction Flow

```
POST /api/transactions
    ↓
TransactionHandler.Create()
    ├─ Parse JSON body → CreateTransactionRequest
    ├─ Get user from auth context
    └─ Call service
        ↓
    TransactionService.CreateTransaction()
        ├─ Validate amount > 0
        ├─ Validate date format
        └─ Call repository
            ↓
        TransactionRepository.Create()
            ├─ Create new Record
            ├─ Set fields
            ├─ Save to PocketBase
            └─ Convert to DTO using WrapRecord[Transactions]
                ↓
            Return TransactionDTO
```

## Type-Safe Proxies Usage

### Before (Unsafe)
```go
// ❌ Runtime error if field doesn't exist
amount := record.Get("amount").(float64)
```

### After (Type-safe)
```go
// ✅ Compile-time error if field doesn't exist
proxy, _ := generated.WrapRecord[generated.Transactions](record)
amount := proxy.Amount() // float64
```

## Regenerating Code

When PocketBase schema changes:

```bash
make generate
```

This will:
1. Read schema from `pb_data/data.db`
2. Generate `pbschema/template.go`
3. Generate type-safe proxies in `generated/`

## Adding New Features

### Example: Add Category Management

1. **Create domain model** (`internal/domain/category.go`)
2. **Create repository** (`internal/repository/category_repository.go`)
3. **Create service** (`internal/service/category_service.go`)
4. **Create handler** (`internal/handler/category_handler.go`)
5. **Wire in main.go**

Each layer only depends on the layer below it:
- Handler → Service
- Service → Repository
- Repository → Generated Proxies → PocketBase
