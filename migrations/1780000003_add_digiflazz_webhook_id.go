package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("pbc_4011000001")
		if err != nil {
			return err
		}

		if err := collection.Fields.AddMarshaledJSONAt(5, []byte(`{
                "autogeneratePattern": "",
                "hidden": false,
                "id": "text4011000011",
                "max": 0,
                "min": 0,
                "name": "webhook_id",
                "pattern": "",
                "presentable": false,
                "primaryKey": false,
                "required": false,
                "system": false,
                "type": "text"
            }`)); err != nil {
			return err
		}

		return app.Save(collection)
	}, func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("pbc_4011000001")
		if err != nil {
			return err
		}

		collection.Fields.RemoveById("text4011000011")
		return app.Save(collection)
	})
}
