package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		// Covering index for dashboard monthly queries (GetDashboardData, GetMonthlyStats, GetMonthlyReportData)
		// Covers: family_id (filter), type (CASE WHEN discriminator), date (date range), amount (SUM aggregation)
		// SQLite can answer queries entirely from this index without touching the table.
		_, err := app.DB().NewQuery(`
			CREATE INDEX IF NOT EXISTS idx_transactions_covering
			ON transactions (family_id, type, date, amount)
		`).Execute()
		return err
	}, func(app core.App) error {
		_, err := app.DB().NewQuery(`DROP INDEX IF EXISTS idx_transactions_covering`).Execute()
		return err
	})
}
