package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("pbc_4011000003")
		if err != nil {
			return err
		}

		collection.Fields.RemoveById("select4011020009")
		if err := collection.Fields.AddMarshaledJSONAt(8, []byte(`{
			"hidden": false,
			"id": "select4011020009",
			"maxSelect": 1,
			"name": "status",
			"presentable": false,
			"required": true,
			"system": false,
			"type": "select",
			"values": ["inquiry", "pending", "processing", "success", "failed", "canceled"]
		}`)); err != nil {
			return err
		}

		if err := app.Save(collection); err != nil {
			return err
		}

		if _, err := app.DB().NewQuery(`UPDATE digiflazz_orders SET status = 'canceled' WHERE status = 'cancelled'`).Execute(); err != nil {
			return err
		}
		if _, err := app.DB().NewQuery(`UPDATE digiflazz_events SET status_before = 'canceled' WHERE status_before = 'cancelled'`).Execute(); err != nil {
			return err
		}
		if _, err := app.DB().NewQuery(`UPDATE digiflazz_events SET status_after = 'canceled' WHERE status_after = 'cancelled'`).Execute(); err != nil {
			return err
		}

		return nil
	}, func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("pbc_4011000003")
		if err != nil {
			return err
		}

		collection.Fields.RemoveById("select4011020009")
		if err := collection.Fields.AddMarshaledJSONAt(8, []byte(`{
			"hidden": false,
			"id": "select4011020009",
			"maxSelect": 1,
			"name": "status",
			"presentable": false,
			"required": true,
			"system": false,
			"type": "select",
			"values": ["inquiry", "pending", "processing", "success", "failed", "cancelled"]
		}`)); err != nil {
			return err
		}

		if err := app.Save(collection); err != nil {
			return err
		}

		if _, err := app.DB().NewQuery(`UPDATE digiflazz_orders SET status = 'cancelled' WHERE status = 'canceled'`).Execute(); err != nil {
			return err
		}
		if _, err := app.DB().NewQuery(`UPDATE digiflazz_events SET status_before = 'cancelled' WHERE status_before = 'canceled'`).Execute(); err != nil {
			return err
		}
		if _, err := app.DB().NewQuery(`UPDATE digiflazz_events SET status_after = 'cancelled' WHERE status_after = 'canceled'`).Execute(); err != nil {
			return err
		}

		return nil
	})
}
