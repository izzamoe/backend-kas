package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		jsonData := `[
			{
				"createRule": "@request.auth.id != ''",
				"deleteRule": "@collection.family_members.user_id ?= @request.auth.id && @collection.family_members.family_id ?= family_id && @collection.family_members.role ?= \"owner\"",
				"fields": [
					{
						"autogeneratePattern": "[a-z0-9]{15}",
						"hidden": false,
						"id": "text4011000001",
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
						"id": "relation4011000002",
						"maxSelect": 1,
						"minSelect": 0,
						"name": "family_id",
						"presentable": false,
						"required": true,
						"system": false,
						"type": "relation"
					},
					{
						"autogeneratePattern": "",
						"hidden": false,
						"id": "text4011000003",
						"max": 0,
						"min": 0,
						"name": "username",
						"pattern": "",
						"presentable": false,
						"primaryKey": false,
						"required": true,
						"system": false,
						"type": "text"
					},
					{
						"autogeneratePattern": "",
						"hidden": false,
						"id": "text4011000004",
						"max": 0,
						"min": 0,
						"name": "api_key",
						"pattern": "",
						"presentable": false,
						"primaryKey": false,
						"required": true,
						"system": false,
						"type": "text"
					},
					{
						"default": false,
						"hidden": false,
						"id": "bool4011000005",
						"name": "testing",
						"presentable": false,
						"required": false,
						"system": false,
						"type": "bool"
					},
					{
						"default": true,
						"hidden": false,
						"id": "bool4011000006",
						"name": "is_active",
						"presentable": false,
						"required": false,
						"system": false,
						"type": "bool"
					},
					{
						"hidden": false,
						"id": "autodate4011000007",
						"name": "created",
						"onCreate": true,
						"onUpdate": false,
						"presentable": false,
						"system": false,
						"type": "autodate"
					},
					{
						"hidden": false,
						"id": "autodate4011000008",
						"name": "updated",
						"onCreate": true,
						"onUpdate": true,
						"presentable": false,
						"system": false,
						"type": "autodate"
					}
				],
				"id": "pbc_4011000001",
				"indexes": [],
				"listRule": "@collection.family_members.user_id ?= @request.auth.id && @collection.family_members.family_id ?= family_id && @collection.family_members.role ?= \"owner\"",
				"name": "digiflazz_credentials",
				"system": false,
				"type": "base",
				"updateRule": "@collection.family_members.user_id ?= @request.auth.id && @collection.family_members.family_id ?= family_id && @collection.family_members.role ?= \"owner\"",
				"viewRule": "@collection.family_members.user_id ?= @request.auth.id && @collection.family_members.family_id ?= family_id && @collection.family_members.role ?= \"owner\""
			},
			{
				"createRule": "@request.auth.id != ''",
				"deleteRule": null,
				"fields": [
					{
						"autogeneratePattern": "[a-z0-9]{15}",
						"hidden": false,
						"id": "text4011010001",
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
						"autogeneratePattern": "",
						"hidden": false,
						"id": "text4011010002",
						"max": 0,
						"min": 0,
						"name": "product_name",
						"pattern": "",
						"presentable": false,
						"primaryKey": false,
						"required": true,
						"system": false,
						"type": "text"
					},
					{
						"autogeneratePattern": "",
						"hidden": false,
						"id": "text4011010003",
						"max": 0,
						"min": 0,
						"name": "category",
						"pattern": "",
						"presentable": false,
						"primaryKey": false,
						"required": true,
						"system": false,
						"type": "text"
					},
					{
						"autogeneratePattern": "",
						"hidden": false,
						"id": "text4011010004",
						"max": 0,
						"min": 0,
						"name": "brand",
						"pattern": "",
						"presentable": false,
						"primaryKey": false,
						"required": false,
						"system": false,
						"type": "text"
					},
					{
						"autogeneratePattern": "",
						"hidden": false,
						"id": "text4011010005",
						"max": 0,
						"min": 0,
						"name": "type",
						"pattern": "",
						"presentable": false,
						"primaryKey": false,
						"required": false,
						"system": false,
						"type": "text"
					},
					{
						"autogeneratePattern": "",
						"hidden": false,
						"id": "text4011010006",
						"max": 0,
						"min": 0,
						"name": "buyer_sku_code",
						"pattern": "",
						"presentable": false,
						"primaryKey": false,
						"required": true,
						"system": false,
						"type": "text"
					},
					{
						"hidden": false,
						"id": "number4011010007",
						"max": null,
						"min": null,
						"name": "price",
						"onlyInt": false,
						"presentable": false,
						"required": true,
						"system": false,
						"type": "number"
					},
					{
						"hidden": false,
						"id": "number4011010008",
						"max": null,
						"min": null,
						"name": "admin",
						"onlyInt": false,
						"presentable": false,
						"required": false,
						"system": false,
						"type": "number"
					},
					{
						"autogeneratePattern": "",
						"hidden": false,
						"id": "text4011010009",
						"max": 0,
						"min": 0,
						"name": "buyer_product_status",
						"pattern": "",
						"presentable": false,
						"primaryKey": false,
						"required": false,
						"system": false,
						"type": "text"
					},
					{
						"autogeneratePattern": "",
						"hidden": false,
						"id": "text4011010010",
						"max": 0,
						"min": 0,
						"name": "seller_product_status",
						"pattern": "",
						"presentable": false,
						"primaryKey": false,
						"required": false,
						"system": false,
						"type": "text"
					},
					{
						"hidden": false,
						"id": "number4011010011",
						"max": null,
						"min": null,
						"name": "stock",
						"onlyInt": false,
						"presentable": false,
						"required": false,
						"system": false,
						"type": "number"
					},
					{
						"hidden": false,
						"id": "bool4011010012",
						"name": "multi",
						"presentable": false,
						"required": false,
						"system": false,
						"type": "bool"
					},
					{
						"autogeneratePattern": "",
						"hidden": false,
						"id": "text4011010013",
						"max": 0,
						"min": 0,
						"name": "desc",
						"pattern": "",
						"presentable": false,
						"primaryKey": false,
						"required": false,
						"system": false,
						"type": "text"
					},
					{
						"autogeneratePattern": "",
						"hidden": false,
						"id": "text4011010014",
						"max": 0,
						"min": 0,
						"name": "provider",
						"pattern": "",
						"presentable": false,
						"primaryKey": false,
						"required": false,
						"system": false,
						"type": "text"
					},
					{
						"hidden": false,
						"id": "bool4011010015",
						"name": "is_prepaid",
						"presentable": false,
						"required": true,
						"system": false,
						"type": "bool"
					},
					{
						"hidden": false,
						"id": "autodate4011010016",
						"name": "created",
						"onCreate": true,
						"onUpdate": false,
						"presentable": false,
						"system": false,
						"type": "autodate"
					},
					{
						"hidden": false,
						"id": "autodate4011010017",
						"name": "updated",
						"onCreate": true,
						"onUpdate": true,
						"presentable": false,
						"system": false,
						"type": "autodate"
					}
				],
				"id": "pbc_4011000002",
				"indexes": [],
				"listRule": "@request.auth.id != ''",
				"name": "digiflazz_products",
				"system": false,
				"type": "base",
				"updateRule": null,
				"viewRule": "@request.auth.id != ''"
			},
			{
				"createRule": "@request.auth.id != ''",
				"deleteRule": "created_by = @request.auth.id",
				"fields": [
					{
						"autogeneratePattern": "[a-z0-9]{15}",
						"hidden": false,
						"id": "text4011020001",
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
						"id": "relation4011020002",
						"maxSelect": 1,
						"minSelect": 0,
						"name": "family_id",
						"presentable": false,
						"required": true,
						"system": false,
						"type": "relation"
					},
					{
						"cascadeDelete": true,
						"collectionId": "_pb_users_auth_",
						"hidden": false,
						"id": "relation4011020003",
						"maxSelect": 1,
						"minSelect": 0,
						"name": "created_by",
						"presentable": false,
						"required": true,
						"system": false,
						"type": "relation"
					},
					{
						"autogeneratePattern": "",
						"hidden": false,
						"id": "text4011020004",
						"max": 0,
						"min": 0,
						"name": "ref_id",
						"pattern": "",
						"presentable": false,
						"primaryKey": false,
						"required": true,
						"system": false,
						"type": "text"
					},
					{
						"autogeneratePattern": "",
						"hidden": false,
						"id": "text4011020005",
						"max": 0,
						"min": 0,
						"name": "buyer_sku_code",
						"pattern": "",
						"presentable": false,
						"primaryKey": false,
						"required": true,
						"system": false,
						"type": "text"
					},
					{
						"autogeneratePattern": "",
						"hidden": false,
						"id": "text4011020006",
						"max": 0,
						"min": 0,
						"name": "customer_no",
						"pattern": "",
						"presentable": false,
						"primaryKey": false,
						"required": true,
						"system": false,
						"type": "text"
					},
					{
						"autogeneratePattern": "",
						"hidden": false,
						"id": "text4011020007",
						"max": 0,
						"min": 0,
						"name": "product_name",
						"pattern": "",
						"presentable": false,
						"primaryKey": false,
						"required": false,
						"system": false,
						"type": "text"
					},
					{
						"autogeneratePattern": "",
						"hidden": false,
						"id": "text4011020008",
						"max": 0,
						"min": 0,
						"name": "category",
						"pattern": "",
						"presentable": false,
						"primaryKey": false,
						"required": false,
						"system": false,
						"type": "text"
					},
					{
						"hidden": false,
						"id": "select4011020009",
						"maxSelect": 1,
						"name": "status",
						"presentable": false,
						"required": true,
						"system": false,
						"type": "select",
						"values": [
							"pending",
							"processing",
							"success",
							"failed",
							"cancelled"
						]
					},
					{
						"hidden": false,
						"id": "number4011020010",
						"max": null,
						"min": null,
						"name": "price",
						"onlyInt": false,
						"presentable": false,
						"required": false,
						"system": false,
						"type": "number"
					},
					{
						"hidden": false,
						"id": "number4011020011",
						"max": null,
						"min": null,
						"name": "admin",
						"onlyInt": false,
						"presentable": false,
						"required": false,
						"system": false,
						"type": "number"
					},
					{
						"hidden": false,
						"id": "number4011020012",
						"max": null,
						"min": null,
						"name": "total",
						"onlyInt": false,
						"presentable": false,
						"required": false,
						"system": false,
						"type": "number"
					},
					{
						"autogeneratePattern": "",
						"hidden": false,
						"id": "text4011020013",
						"max": 0,
						"min": 0,
						"name": "sn",
						"pattern": "",
						"presentable": false,
						"primaryKey": false,
						"required": false,
						"system": false,
						"type": "text"
					},
					{
						"autogeneratePattern": "",
						"hidden": false,
						"id": "text4011020014",
						"max": 0,
						"min": 0,
						"name": "message",
						"pattern": "",
						"presentable": false,
						"primaryKey": false,
						"required": false,
						"system": false,
						"type": "text"
					},
					{
						"autogeneratePattern": "",
						"hidden": false,
						"id": "text4011020015",
						"max": 0,
						"min": 0,
						"name": "rc",
						"pattern": "",
						"presentable": false,
						"primaryKey": false,
						"required": false,
						"system": false,
						"type": "text"
					},
					{
						"hidden": false,
						"id": "json4011020016",
						"maxSize": 0,
						"name": "payload",
						"presentable": false,
						"required": false,
						"system": false,
						"type": "json"
					},
					{
						"hidden": false,
						"id": "json4011020017",
						"maxSize": 0,
						"name": "response",
						"presentable": false,
						"required": false,
						"system": false,
						"type": "json"
					},
					{
						"autogeneratePattern": "",
						"hidden": false,
						"id": "text4011020018",
						"max": 0,
						"min": 0,
						"name": "transaction_id",
						"pattern": "",
						"presentable": false,
						"primaryKey": false,
						"required": false,
						"system": false,
						"type": "text"
					},
					{
						"hidden": false,
						"id": "bool4011020019",
						"name": "is_prepaid",
						"presentable": false,
						"required": true,
						"system": false,
						"type": "bool"
					},
					{
						"hidden": false,
						"id": "autodate4011020020",
						"name": "created",
						"onCreate": true,
						"onUpdate": false,
						"presentable": false,
						"system": false,
						"type": "autodate"
					},
					{
						"hidden": false,
						"id": "autodate4011020021",
						"name": "updated",
						"onCreate": true,
						"onUpdate": true,
						"presentable": false,
						"system": false,
						"type": "autodate"
					}
				],
				"id": "pbc_4011000003",
				"indexes": [
					"CREATE UNIQUE INDEX idx_digiflazz_orders_ref_id ON digiflazz_orders (ref_id)"
				],
				"listRule": "@collection.family_members.user_id ?= @request.auth.id && @collection.family_members.family_id ?= family_id",
				"name": "digiflazz_orders",
				"system": false,
				"type": "base",
				"updateRule": "created_by = @request.auth.id",
				"viewRule": "@collection.family_members.user_id ?= @request.auth.id && @collection.family_members.family_id ?= family_id"
			},
			{
				"createRule": null,
				"deleteRule": null,
				"fields": [
					{
						"autogeneratePattern": "[a-z0-9]{15}",
						"hidden": false,
						"id": "text4011030001",
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
						"collectionId": "pbc_4011000003",
						"hidden": false,
						"id": "relation4011030002",
						"maxSelect": 1,
						"minSelect": 0,
						"name": "order_id",
						"presentable": false,
						"required": true,
						"system": false,
						"type": "relation"
					},
					{
						"hidden": false,
						"id": "select4011030003",
						"maxSelect": 1,
						"name": "event_type",
						"presentable": false,
						"required": true,
						"system": false,
						"type": "select",
						"values": [
							"topup",
							"inquiry",
							"pay",
							"status",
							"deposit",
							"webhook",
							"error"
						]
					},
					{
						"autogeneratePattern": "",
						"hidden": false,
						"id": "text4011030004",
						"max": 0,
						"min": 0,
						"name": "status_before",
						"pattern": "",
						"presentable": false,
						"primaryKey": false,
						"required": false,
						"system": false,
						"type": "text"
					},
					{
						"autogeneratePattern": "",
						"hidden": false,
						"id": "text4011030005",
						"max": 0,
						"min": 0,
						"name": "status_after",
						"pattern": "",
						"presentable": false,
						"primaryKey": false,
						"required": false,
						"system": false,
						"type": "text"
					},
					{
						"hidden": false,
						"id": "json4011030006",
						"maxSize": 0,
						"name": "payload",
						"presentable": false,
						"required": false,
						"system": false,
						"type": "json"
					},
					{
						"hidden": false,
						"id": "json4011030007",
						"maxSize": 0,
						"name": "response",
						"presentable": false,
						"required": false,
						"system": false,
						"type": "json"
					},
					{
						"autogeneratePattern": "",
						"hidden": false,
						"id": "text4011030008",
						"max": 0,
						"min": 0,
						"name": "source",
						"pattern": "",
						"presentable": false,
						"primaryKey": false,
						"required": false,
						"system": false,
						"type": "text"
					},
					{
						"hidden": false,
						"id": "autodate4011030009",
						"name": "created",
						"onCreate": true,
						"onUpdate": false,
						"presentable": false,
						"system": false,
						"type": "autodate"
					},
					{
						"hidden": false,
						"id": "autodate4011030010",
						"name": "updated",
						"onCreate": true,
						"onUpdate": true,
						"presentable": false,
						"system": false,
						"type": "autodate"
					}
				],
				"id": "pbc_4011000004",
				"indexes": [],
				"listRule": "@collection.family_members.user_id ?= @request.auth.id && @collection.family_members.family_id ?= order_id.family_id && @collection.family_members.role ?= \"owner\"",
				"name": "digiflazz_events",
				"system": false,
				"type": "base",
				"updateRule": null,
				"viewRule": "@collection.family_members.user_id ?= @request.auth.id && @collection.family_members.family_id ?= order_id.family_id && @collection.family_members.role ?= \"owner\""
			}
		]`

		return app.ImportCollectionsByMarshaledJSON([]byte(jsonData), false)
	}, func(app core.App) error {
		for _, name := range []string{
			"digiflazz_events",
			"digiflazz_orders",
			"digiflazz_products",
			"digiflazz_credentials",
		} {
			collection, err := app.FindCollectionByNameOrId(name)
			if err != nil {
				return err
			}

			if err := app.Delete(collection); err != nil {
				return err
			}
		}

		return nil
	})
}
