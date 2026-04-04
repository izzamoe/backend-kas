package repository_test

import (
	"os"
	"testing"

	"github.com/pocketbase/pocketbase"
	_ "kas/migrations"

	"kas/internal/repository"
)

// setupTestApp creates a minimal PocketBase instance for benchmarking.
// It uses a temp directory so it doesn't pollute the real pb_data.
func setupTestApp(tb testing.TB) *pocketbase.PocketBase {
	dir, err := os.MkdirTemp("", "pb_bench_*")
	if err != nil {
		tb.Fatalf("failed to create temp dir: %v", err)
	}
	tb.Cleanup(func() { os.RemoveAll(dir) })

	app := pocketbase.NewWithConfig(pocketbase.Config{
		DefaultDataDir: dir,
	})

	if err := app.Bootstrap(); err != nil {
		tb.Fatalf("failed to bootstrap app: %v", err)
	}

	return app
}

func BenchmarkGetTotalByFamily(b *testing.B) {
	app := setupTestApp(b)
	defer app.ResetBootstrapState()
	repo := repository.NewTransactionRepository(app)

	const familyID = "test_family_001"

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _ = repo.GetTotalByFamily(familyID)
	}
}

func BenchmarkGetMonthlyStats(b *testing.B) {
	app := setupTestApp(b)
	defer app.ResetBootstrapState()
	repo := repository.NewTransactionRepository(app)

	const familyID = "test_family_001"

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _, _ = repo.GetMonthlyStats(familyID, 2026, 3)
	}
}

func BenchmarkGetByFamilyAndMonth(b *testing.B) {
	app := setupTestApp(b)
	defer app.ResetBootstrapState()
	repo := repository.NewTransactionRepository(app)

	const familyID = "test_family_001"

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _ = repo.GetByFamilyAndMonth(familyID, 2026, 3)
	}
}

func BenchmarkGetByFamilyID(b *testing.B) {
	app := setupTestApp(b)
	defer app.ResetBootstrapState()
	repo := repository.NewTransactionRepository(app)

	const familyID = "test_family_001"

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _ = repo.GetByFamilyID(familyID, 20, 0)
	}
}
