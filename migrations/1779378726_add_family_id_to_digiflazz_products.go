package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func stringPtr(s string) *string { return &s }

func init() {
	m.Register(func(app core.App) error {
		if _, err := app.DB().NewQuery(`DELETE FROM digiflazz_products`).Execute(); err != nil {
			return err
		}

		collection, err := app.FindCollectionByNameOrId("pbc_4011000002")
		if err != nil {
			return err
		}

		if err := collection.Fields.AddMarshaledJSONAt(1, []byte(`{
			"cascadeDelete": true,
			"collectionId": "pbc_3641796565",
			"hidden": false,
			"id": "relation4011010002",
			"maxSelect": 1,
			"minSelect": 0,
			"name": "family_id",
			"presentable": false,
			"required": true,
			"system": false,
			"type": "relation"
		}`)); err != nil {
			return err
		}

		if err := collection.Fields.AddMarshaledJSONAt(2, []byte(`{
			"cascadeDelete": false,
			"collectionId": "pbc_4011000001",
			"hidden": false,
			"id": "relation4011010003",
			"maxSelect": 1,
			"minSelect": 0,
			"name": "credential_id",
			"presentable": false,
			"required": false,
			"system": false,
			"type": "relation"
		}`)); err != nil {
			return err
		}

		collection.ListRule = stringPtr(`@collection.family_members.user_id ?= @request.auth.id && @collection.family_members.family_id ?= family_id`)
		collection.ViewRule = stringPtr(`@collection.family_members.user_id ?= @request.auth.id && @collection.family_members.family_id ?= family_id`)
		collection.CreateRule = nil
		collection.UpdateRule = nil
		collection.DeleteRule = nil

		if err := app.Save(collection); err != nil {
			return err
		}

		if _, err := app.DB().NewQuery(`CREATE UNIQUE INDEX IF NOT EXISTS idx_digiflazz_products_family_sku ON digiflazz_products (family_id, buyer_sku_code)`).Execute(); err != nil {
			return err
		}

		rows, err := app.DB().NewQuery(`SELECT name FROM sqlite_master WHERE type='index' AND name='idx_digiflazz_credentials_family_unique'`).Rows()
		if err != nil {
			return err
		}
		defer rows.Close()
		idxExists := rows.Next()
		if !idxExists {
			if _, err := app.DB().NewQuery(`DELETE FROM digiflazz_credentials WHERE id NOT IN (SELECT id FROM (SELECT id, ROW_NUMBER() OVER (PARTITION BY family_id ORDER BY created DESC, id DESC) AS rn FROM digiflazz_credentials) WHERE rn = 1)`).Execute(); err != nil {
				return err
			}
			if _, err := app.DB().NewQuery(`CREATE UNIQUE INDEX idx_digiflazz_credentials_family_unique ON digiflazz_credentials (family_id)`).Execute(); err != nil {
				return err
			}
		}

		cred, err := app.FindCollectionByNameOrId("pbc_4011000001")
		if err != nil {
			return err
		}
		cred.ListRule = nil
		cred.ViewRule = nil
		cred.CreateRule = nil
		cred.UpdateRule = nil
		cred.DeleteRule = nil
		return app.Save(cred)

	}, func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("pbc_4011000002")
		if err != nil {
			return err
		}
		collection.Fields.RemoveById("relation4011010003")
		collection.Fields.RemoveById("relation4011010002")
		collection.ListRule = stringPtr(`@request.auth.id != ''`)
		collection.ViewRule = stringPtr(`@request.auth.id != ''`)
		collection.CreateRule = stringPtr(`@request.auth.id != ''`)
		collection.UpdateRule = nil
		collection.DeleteRule = nil
		if err := app.Save(collection); err != nil {
			return err
		}

		if _, err := app.DB().NewQuery(`DROP INDEX IF EXISTS idx_digiflazz_products_family_sku`).Execute(); err != nil {
			return err
		}
		if _, err := app.DB().NewQuery(`DROP INDEX IF EXISTS idx_digiflazz_credentials_family_unique`).Execute(); err != nil {
			return err
		}

		cred, err := app.FindCollectionByNameOrId("pbc_4011000001")
		if err != nil {
			return err
		}
		cred.ListRule = stringPtr(`@collection.family_members.user_id ?= @request.auth.id && @collection.family_members.family_id ?= family_id && @collection.family_members.role ?= "owner"`)
		cred.ViewRule = stringPtr(`@collection.family_members.user_id ?= @request.auth.id && @collection.family_members.family_id ?= family_id && @collection.family_members.role ?= "owner"`)
		cred.CreateRule = stringPtr(`@request.auth.id != ''`)
		cred.UpdateRule = stringPtr(`@collection.family_members.user_id ?= @request.auth.id && @collection.family_members.family_id ?= family_id && @collection.family_members.role ?= "owner"`)
		cred.DeleteRule = stringPtr(`@collection.family_members.user_id ?= @request.auth.id && @collection.family_members.family_id ?= family_id && @collection.family_members.role ?= "owner"`)
		return app.Save(cred)

	})
}
