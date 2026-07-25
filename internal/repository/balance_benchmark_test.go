package repository_test

import (
	"database/sql"
	"fmt"
	"testing"

	_ "modernc.org/sqlite"
)

// BenchmarkBalanceQuery compares O(1) materialized read vs O(n) SUM aggregation.
//
// Setup:
//   - Creates a temp SQLite database (no PocketBase overhead)
//   - Seeds N transactions for one family
//   - Measures both query paths:
//     OLD: COALESCE(SUM(CASE WHEN type='income' THEN amount ELSE 0 END), 0) - ...
//     NEW: SELECT balance FROM family_balances WHERE family_id = ?
type benchData struct {
	db       *sql.DB
	familyID string
}

func setupBenchData(b *testing.B, txCount int) *benchData {
	b.Helper()

	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		b.Fatalf("failed to open db: %v", err)
	}
	b.Cleanup(func() { db.Close() })

	// Enable WAL + performance pragmas (same as production)
	pragmas := []string{
		"PRAGMA journal_mode=WAL",
		"PRAGMA synchronous=NORMAL",
		"PRAGMA temp_store=MEMORY",
		"PRAGMA cache_size=-32000",
		"PRAGMA mmap_size=268435456",
	}
	for _, p := range pragmas {
		if _, err := db.Exec(p); err != nil {
			b.Fatalf("pragma %s: %v", p, err)
		}
	}

	// Create transactions table (schema-agnostic, just what we need)
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS transactions (
			id TEXT PRIMARY KEY,
			family_id TEXT NOT NULL,
			type TEXT NOT NULL CHECK(type IN ('income','expense')),
			amount REAL NOT NULL,
			date TEXT NOT NULL
		)
	`)
	if err != nil {
		b.Fatalf("create transactions: %v", err)
	}

	// Create family_balances table (materialized)
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS family_balances (
			family_id TEXT PRIMARY KEY,
			balance REAL NOT NULL DEFAULT 0,
			total_income REAL NOT NULL DEFAULT 0,
			total_expense REAL NOT NULL DEFAULT 0
		)
	`)
	if err != nil {
		b.Fatalf("create family_balances: %v", err)
	}

	familyID := "bench_family_001"

	// Seed transactions
	incomeAmt := 1000000.0
	expenseAmt := 50000.0

	tx, err := db.Begin()
	if err != nil {
		b.Fatalf("begin tx: %v", err)
	}

	stmt, err := tx.Prepare("INSERT INTO transactions (id, family_id, type, amount, date) VALUES (?, ?, ?, ?, ?)")
	if err != nil {
		b.Fatalf("prepare stmt: %v", err)
	}
	defer stmt.Close()

	for i := range txCount {
		typ := "income"
		amt := incomeAmt
		if i%3 == 0 { // every 3rd is an expense
			typ = "expense"
			amt = expenseAmt
		}
		id := fmt.Sprintf("tx_%s_%06d", familyID, i)
		date := fmt.Sprintf("2026-%02d-%02d", (i%12)+1, (i%28)+1)
		if _, err := stmt.Exec(id, familyID, typ, amt, date); err != nil {
			b.Fatalf("insert tx %d: %v", i, err)
		}
	}

	if err := tx.Commit(); err != nil {
		b.Fatalf("commit: %v", err)
	}

	// Populate family_balances with correct aggregate
	var totalBalance, totalIncome, totalExpense float64
	row := db.QueryRow(`
		SELECT
			COALESCE(SUM(CASE WHEN type='income' THEN amount ELSE 0 END), 0) -
			COALESCE(SUM(CASE WHEN type='expense' THEN amount ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN type='income' THEN amount ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN type='expense' THEN amount ELSE 0 END), 0)
		FROM transactions WHERE family_id = ?
	`, familyID)
	if err := row.Scan(&totalBalance, &totalIncome, &totalExpense); err != nil {
		b.Fatalf("compute balance: %v", err)
	}
	if _, err := db.Exec(
		"INSERT INTO family_balances (family_id, balance, total_income, total_expense) VALUES (?, ?, ?, ?)",
		familyID, totalBalance, totalIncome, totalExpense,
	); err != nil {
		b.Fatalf("insert family_balances: %v", err)
	}

	return &benchData{db: db, familyID: familyID}
}

// --- OLD PATH: Full SUM aggregation on transactions ---

func BenchmarkOLD_GetTotalByFamily_0tx(b *testing.B) {
	d := setupBenchData(b, 0)
	b.ResetTimer()
	b.ReportAllocs()
	for range b.N {
		d.db.QueryRow(`
			SELECT
				COALESCE(SUM(CASE WHEN type='income' THEN amount ELSE 0 END), 0) -
				COALESCE(SUM(CASE WHEN type='expense' THEN amount ELSE 0 END), 0)
			FROM transactions WHERE family_id = ?
		`, d.familyID)
	}
}

func BenchmarkOLD_GetTotalByFamily_100tx(b *testing.B) {
	d := setupBenchData(b, 100)
	b.ResetTimer()
	b.ReportAllocs()
	for range b.N {
		d.db.QueryRow(`
			SELECT
				COALESCE(SUM(CASE WHEN type='income' THEN amount ELSE 0 END), 0) -
				COALESCE(SUM(CASE WHEN type='expense' THEN amount ELSE 0 END), 0)
			FROM transactions WHERE family_id = ?
		`, d.familyID)
	}
}

func BenchmarkOLD_GetTotalByFamily_1000tx(b *testing.B) {
	d := setupBenchData(b, 1000)
	b.ResetTimer()
	b.ReportAllocs()
	for range b.N {
		d.db.QueryRow(`
			SELECT
				COALESCE(SUM(CASE WHEN type='income' THEN amount ELSE 0 END), 0) -
				COALESCE(SUM(CASE WHEN type='expense' THEN amount ELSE 0 END), 0)
			FROM transactions WHERE family_id = ?
		`, d.familyID)
	}
}

func BenchmarkOLD_GetTotalByFamily_5000tx(b *testing.B) {
	d := setupBenchData(b, 5000)
	b.ResetTimer()
	b.ReportAllocs()
	for range b.N {
		d.db.QueryRow(`
			SELECT
				COALESCE(SUM(CASE WHEN type='income' THEN amount ELSE 0 END), 0) -
				COALESCE(SUM(CASE WHEN type='expense' THEN amount ELSE 0 END), 0)
			FROM transactions WHERE family_id = ?
		`, d.familyID)
	}
}

// --- NEW PATH: O(1) read from materialized family_balances table ---

func BenchmarkNEW_GetTotalByFamily_0tx(b *testing.B) {
	d := setupBenchData(b, 0)
	b.ResetTimer()
	b.ReportAllocs()
	for range b.N {
		d.db.QueryRow("SELECT balance FROM family_balances WHERE family_id = ?", d.familyID)
	}
}

func BenchmarkNEW_GetTotalByFamily_100tx(b *testing.B) {
	d := setupBenchData(b, 100)
	b.ResetTimer()
	b.ReportAllocs()
	for range b.N {
		d.db.QueryRow("SELECT balance FROM family_balances WHERE family_id = ?", d.familyID)
	}
}

func BenchmarkNEW_GetTotalByFamily_1000tx(b *testing.B) {
	d := setupBenchData(b, 1000)
	b.ResetTimer()
	b.ReportAllocs()
	for range b.N {
		d.db.QueryRow("SELECT balance FROM family_balances WHERE family_id = ?", d.familyID)
	}
}

func BenchmarkNEW_GetTotalByFamily_5000tx(b *testing.B) {
	d := setupBenchData(b, 5000)
	b.ResetTimer()
	b.ReportAllocs()
	for range b.N {
		d.db.QueryRow("SELECT balance FROM family_balances WHERE family_id = ?", d.familyID)
	}
}

// --- DASHBOARD QUERY: monthly stats with covering index ---

func BenchmarkOLD_GetDashboardData_100tx(b *testing.B) {
	d := setupBenchData(b, 100)
	b.ResetTimer()
	b.ReportAllocs()
	for range b.N {
		// Old dashboard query: reads balance + monthly in one full table scan
		d.db.QueryRow(`
			SELECT
				COALESCE(SUM(CASE WHEN type='income' THEN amount ELSE 0 END), 0) -
				COALESCE(SUM(CASE WHEN type='expense' THEN amount ELSE 0 END), 0),
				COALESCE(SUM(CASE WHEN type='income' AND date>='2026-06-01' AND date<'2026-07-01' THEN amount ELSE 0 END), 0),
				COALESCE(SUM(CASE WHEN type='expense' AND date>='2026-06-01' AND date<'2026-07-01' THEN amount ELSE 0 END), 0),
				COALESCE(SUM(CASE WHEN type='income' AND date>='2026-05-01' AND date<'2026-06-01' THEN amount ELSE 0 END), 0),
				COALESCE(SUM(CASE WHEN type='expense' AND date>='2026-05-01' AND date<'2026-06-01' THEN amount ELSE 0 END), 0)
			FROM transactions WHERE family_id = ?
		`, d.familyID)
	}
}

func BenchmarkOLD_GetDashboardData_1000tx(b *testing.B) {
	d := setupBenchData(b, 1000)
	b.ResetTimer()
	b.ReportAllocs()
	for range b.N {
		d.db.QueryRow(`
			SELECT
				COALESCE(SUM(CASE WHEN type='income' THEN amount ELSE 0 END), 0) -
				COALESCE(SUM(CASE WHEN type='expense' THEN amount ELSE 0 END), 0),
				COALESCE(SUM(CASE WHEN type='income' AND date>='2026-06-01' AND date<'2026-07-01' THEN amount ELSE 0 END), 0),
				COALESCE(SUM(CASE WHEN type='expense' AND date>='2026-06-01' AND date<'2026-07-01' THEN amount ELSE 0 END), 0),
				COALESCE(SUM(CASE WHEN type='income' AND date>='2026-05-01' AND date<'2026-06-01' THEN amount ELSE 0 END), 0),
				COALESCE(SUM(CASE WHEN type='expense' AND date>='2026-05-01' AND date<'2026-06-01' THEN amount ELSE 0 END), 0)
			FROM transactions WHERE family_id = ?
		`, d.familyID)
	}
}

func BenchmarkNEW_GetDashboardData_100tx(b *testing.B) {
	d := setupBenchData(b, 100)
	b.ResetTimer()
	b.ReportAllocs()
	for range b.N {
		// New dashboard query: balance from family_balances via CTE (single round-trip)
		d.db.QueryRow(`
			WITH bal AS (
				SELECT COALESCE(balance, 0) AS v FROM family_balances WHERE family_id = ?
			)
			SELECT
				(SELECT v FROM bal),
				COALESCE(SUM(CASE WHEN type='income' AND date>='2026-06-01' AND date<'2026-07-01' THEN amount ELSE 0 END), 0),
				COALESCE(SUM(CASE WHEN type='expense' AND date>='2026-06-01' AND date<'2026-07-01' THEN amount ELSE 0 END), 0),
				COALESCE(SUM(CASE WHEN type='income' AND date>='2026-05-01' AND date<'2026-06-01' THEN amount ELSE 0 END), 0),
				COALESCE(SUM(CASE WHEN type='expense' AND date>='2026-05-01' AND date<'2026-06-01' THEN amount ELSE 0 END), 0)
			FROM transactions WHERE family_id = ? AND date >= '2026-05-01'
		`, d.familyID, d.familyID)
	}
}

func BenchmarkNEW_GetDashboardData_1000tx(b *testing.B) {
	d := setupBenchData(b, 1000)
	b.ResetTimer()
	b.ReportAllocs()
	for range b.N {
		d.db.QueryRow(`
			WITH bal AS (
				SELECT COALESCE(balance, 0) AS v FROM family_balances WHERE family_id = ?
			)
			SELECT
				(SELECT v FROM bal),
				COALESCE(SUM(CASE WHEN type='income' AND date>='2026-06-01' AND date<'2026-07-01' THEN amount ELSE 0 END), 0),
				COALESCE(SUM(CASE WHEN type='expense' AND date>='2026-06-01' AND date<'2026-07-01' THEN amount ELSE 0 END), 0),
				COALESCE(SUM(CASE WHEN type='income' AND date>='2026-05-01' AND date<'2026-06-01' THEN amount ELSE 0 END), 0),
				COALESCE(SUM(CASE WHEN type='expense' AND date>='2026-05-01' AND date<'2026-06-01' THEN amount ELSE 0 END), 0)
			FROM transactions WHERE family_id = ? AND date >= '2026-05-01'
		`, d.familyID, d.familyID)
	}
}
