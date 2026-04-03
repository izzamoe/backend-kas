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
│   │   └── transaction.go           # DTOs & request/response models
│   │
│   ├── repository/                  # Data access layer
│   │   └── transaction_repository.go # DB operations using generated proxies
│   │
│   ├── service/                     # Business logic layer
│   │   └── transaction_service.go   # Business rules & validation
│   │
│   ├── handler/                     # HTTP handlers
│   │   └── transaction_handler.go   # REST API endpoints
│   │
│   ├── middleware/                  # HTTP middleware
│   │   └── auth.go                  # Authentication middleware
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

### Repository Layer (Data Access)
- `internal/repository/transaction_repository.go`
  - `TransactionRepository` - Interface
  - `transactionRepo` - Implementation using generated proxies
  - CRUD operations
  - **Uses generated proxies for type-safety**

### Service Layer (Business Logic)
- `internal/service/transaction_service.go`
  - `TransactionService` - Interface
  - `transactionService` - Implementation
  - Business validation
  - Authorization logic

### Handler Layer (HTTP/API)
- `internal/handler/transaction_handler.go`
  - `TransactionHandler` - HTTP request handlers
  - REST API endpoints
  - Request/response handling

### Middleware
- `internal/middleware/auth.go`
  - Authentication middleware
  - Authorization checks

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
