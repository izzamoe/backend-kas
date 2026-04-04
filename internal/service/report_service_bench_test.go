package service_test

import (
	"os"
	"testing"

	"github.com/pocketbase/pocketbase"
	_ "kas/migrations"

	"kas/internal/domain"
	"kas/internal/repository"
	"kas/internal/service"
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

func BenchmarkGetMonthlyReport(b *testing.B) {
	app := setupTestApp(b)
	defer app.ResetBootstrapState()
	transactionRepo := repository.NewTransactionRepository(app)
	reportService := service.NewReportService(transactionRepo)

	req := &domain.MonthlyReportRequest{
		FamilyID: "test_family_001",
		Year:     2026,
		Month:    3,
	}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _ = reportService.GetMonthlyReport(req)
	}
}

func BenchmarkGetDashboardSummary(b *testing.B) {
	app := setupTestApp(b)
	defer app.ResetBootstrapState()
	transactionRepo := repository.NewTransactionRepository(app)
	reportService := service.NewReportService(transactionRepo)

	req := &domain.DashboardSummaryRequest{
		FamilyID: "test_family_001",
		Year:     2026,
		Month:    3,
	}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _ = reportService.GetDashboardSummary(req)
	}
}
