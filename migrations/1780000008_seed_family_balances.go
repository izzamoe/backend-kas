package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		// Seed family_balances from existing transaction data
		// Uses INSERT OR IGNORE to handle families with no transactions
		_, err := app.DB().NewQuery(`
			INSERT INTO family_balances (id, family_id, balance, total_income, total_expense)
			SELECT 
				'seed_' || f.id,
				f.id,
				COALESCE((
					SELECT SUM(CASE WHEN type = 'income' THEN amount ELSE 0 END) -
					     SUM(CASE WHEN type = 'expense' THEN amount ELSE 0 END)
					FROM transactions WHERE family_id = f.id
				), 0),
				COALESCE((
					SELECT SUM(CASE WHEN type = 'income' THEN amount ELSE 0 END)
					FROM transactions WHERE family_id = f.id
				), 0),
				COALESCE((
					SELECT SUM(CASE WHEN type = 'expense' THEN amount ELSE 0 END)
					FROM transactions WHERE family_id = f.id
				), 0)
			FROM families f
		`).Execute()
		return err
	}, func(app core.App) error {
		// Rollback: delete all rows seeded with 'seed_' prefix
		_, err := app.DB().NewQuery(`DELETE FROM family_balances WHERE id LIKE 'seed_%'`).Execute()
		return err
	})
}
