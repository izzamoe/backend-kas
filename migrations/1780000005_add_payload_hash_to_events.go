package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("pbc_4011000004")
		if err != nil {
			return err
		}

		if err := collection.Fields.AddMarshaledJSONAt(len(collection.Fields), []byte(`{
			"autogeneratePattern": "",
			"hidden": false,
			"id": "text4011000030",
			"max": 0,
			"min": 0,
			"name": "payload_hash",
			"pattern": "",
			"presentable": false,
			"primaryKey": false,
			"required": false,
			"system": false,
			"type": "text"
		}`)); err != nil {
			return err
		}

		if err := app.Save(collection); err != nil {
			return err
		}

		_, err = app.DB().NewQuery(
			`CREATE INDEX IF NOT EXISTS idx_dfe_order_payload_hash ON digiflazz_events (order_id, payload_hash)`,
		).Execute()
		return err
	}, func(app core.App) error {
		if _, err := app.DB().NewQuery(
			`DROP INDEX IF EXISTS idx_dfe_order_payload_hash`,
		).Execute(); err != nil {
			return err
		}

		collection, err := app.FindCollectionByNameOrId("pbc_4011000004")
		if err != nil {
			return err
		}

		collection.Fields.RemoveById("text4011000030")
		return app.Save(collection)
	})
}
