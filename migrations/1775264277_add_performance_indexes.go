package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		// Index for: GetByFamilyID, GetTotalByFamily, general family filtering
		_, err := app.DB().NewQuery(`CREATE INDEX IF NOT EXISTS idx_transactions_family_id ON transactions (family_id)`).Execute()
		if err != nil {
			return err
		}

		// Composite index for: GetByFamilyAndMonth, GetMonthlyStats (family + date range)
		_, err = app.DB().NewQuery(`CREATE INDEX IF NOT EXISTS idx_transactions_family_date ON transactions (family_id, date)`).Execute()
		if err != nil {
			return err
		}

		// Composite index for: GetTotalByFamily (family + type aggregation)
		_, err = app.DB().NewQuery(`CREATE INDEX IF NOT EXISTS idx_transactions_family_type ON transactions (family_id, type)`).Execute()
		if err != nil {
			return err
		}

		// Index for categories by family
		_, err = app.DB().NewQuery(`CREATE INDEX IF NOT EXISTS idx_categories_family_id ON categories (family_id)`).Execute()
		if err != nil {
			return err
		}

		return nil
	}, func(app core.App) error {
		// Rollback
		app.DB().NewQuery(`DROP INDEX IF EXISTS idx_transactions_family_id`).Execute()
		app.DB().NewQuery(`DROP INDEX IF EXISTS idx_transactions_family_date`).Execute()
		app.DB().NewQuery(`DROP INDEX IF EXISTS idx_transactions_family_type`).Execute()
		app.DB().NewQuery(`DROP INDEX IF EXISTS idx_categories_family_id`).Execute()
		return nil
	})
}
