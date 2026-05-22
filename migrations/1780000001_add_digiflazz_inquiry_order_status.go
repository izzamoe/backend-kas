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
			"values": ["inquiry", "pending", "processing", "success", "failed", "cancelled"]
		}`)); err != nil {
			return err
		}

		return app.Save(collection)
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
			"values": ["pending", "processing", "success", "failed", "cancelled"]
		}`)); err != nil {
			return err
		}

		return app.Save(collection)
	})
}
