package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		_, err := app.DB().NewQuery(
			`CREATE UNIQUE INDEX IF NOT EXISTS idx_dfc_webhook_token_hash ON digiflazz_credentials (webhook_token_hash)`,
		).Execute()
		return err
	}, func(app core.App) error {
		_, err := app.DB().NewQuery(
			`DROP INDEX IF EXISTS idx_dfc_webhook_token_hash`,
		).Execute()
		return err
	})
}
