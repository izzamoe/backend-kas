package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		jsonData := `[{
			"createRule": "@request.auth.id != '' && @collection.family_members.user_id ?= @request.auth.id && @collection.family_members.family_id ?= family_id",
			"deleteRule": null,
			"fields": [
				{
					"autogeneratePattern": "[a-z0-9]{15}",
					"hidden": false,
					"id": "text5000000001",
					"max": 15,
					"min": 15,
					"name": "id",
					"pattern": "^[a-z0-9]+$",
					"presentable": false,
					"primaryKey": true,
					"required": true,
					"system": true,
					"type": "text"
				},
				{
					"cascadeDelete": true,
					"collectionId": "pbc_3641796565",
					"hidden": false,
					"id": "relation5000000002",
					"maxSelect": 1,
					"minSelect": 0,
					"name": "family_id",
					"presentable": false,
					"required": true,
					"system": false,
					"type": "relation"
				},
				{
					"hidden": false,
					"id": "number5000000003",
					"max": null,
					"min": null,
					"name": "balance",
					"onlyInt": false,
					"presentable": false,
					"required": false,
					"system": false,
					"type": "number"
				},
				{
					"hidden": false,
					"id": "number5000000004",
					"max": null,
					"min": null,
					"name": "total_income",
					"onlyInt": false,
					"presentable": false,
					"required": false,
					"system": false,
					"type": "number"
				},
				{
					"hidden": false,
					"id": "number5000000005",
					"max": null,
					"min": null,
					"name": "total_expense",
					"onlyInt": false,
					"presentable": false,
					"required": false,
					"system": false,
					"type": "number"
				},
				{
					"hidden": false,
					"id": "autodate5000000006",
					"name": "created",
					"onCreate": true,
					"onUpdate": false,
					"presentable": false,
					"system": false,
					"type": "autodate"
				},
				{
					"hidden": false,
					"id": "autodate5000000007",
					"name": "updated",
					"onCreate": true,
					"onUpdate": true,
					"presentable": false,
					"system": false,
					"type": "autodate"
				}
			],
			"id": "pbc_5000000001",
			"indexes": [
				"CREATE UNIQUE INDEX idx_family_balances_family_id ON family_balances (family_id)"
			],
			"listRule": "@request.auth.id != '' && @collection.family_members.user_id ?= @request.auth.id && @collection.family_members.family_id ?= family_id",
			"name": "family_balances",
			"system": false,
			"type": "base",
			"updateRule": "@request.auth.id != '' && @collection.family_members.user_id ?= @request.auth.id && @collection.family_members.family_id ?= family_id && @collection.family_members.role ?= \"owner\"",
			"viewRule": "@request.auth.id != '' && @collection.family_members.user_id ?= @request.auth.id && @collection.family_members.family_id ?= family_id"
		}]`

		return app.ImportCollectionsByMarshaledJSON([]byte(jsonData), false)
	}, func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("family_balances")
		if err != nil {
			return nil
		}
		return app.Delete(collection)
	})
}
