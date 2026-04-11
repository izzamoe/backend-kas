package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		if _, err := app.DB().NewQuery("UPDATE categories SET is_master = FALSE WHERE family_id != ''").Execute(); err != nil {
			return err
		}

		return app.RunInTransaction(func(txApp core.App) error {
			collection, err := txApp.FindCollectionByNameOrId("categories")
			if err != nil {
				return err
			}

			categories := []struct {
				name  string
				icon  string
				color string
				type_ string
			}{
				{name: "Anak & pendidikan", icon: "👶", color: "#D4537E", type_: "expense"},
				{name: "Belanja", icon: "🛒", color: "#378ADD", type_: "expense"},
				{name: "Hiburan", icon: "🎉", color: "#5DCAA5", type_: "expense"},
				{name: "Kesehatan", icon: "💊", color: "#E24B4A", type_: "expense"},
				{name: "Lainnya", icon: "📦", color: "#888780", type_: "expense"},
				{name: "Makan & minum", icon: "🍽️", color: "#1D9E75", type_: "expense"},
				{name: "Pakaian", icon: "👗", color: "#D85A30", type_: "expense"},
				{name: "Rumah & utilitas", icon: "🏠", color: "#7F77DD", type_: "expense"},
				{name: "Transportasi", icon: "🚗", color: "#EF9F27", type_: "expense"},
				{name: "Gaji", icon: "💰", color: "#2ECC71", type_: "income"},
				{name: "Bonus", icon: "🎁", color: "#F39C12", type_: "income"},
				{name: "Investasi", icon: "📈", color: "#3498DB", type_: "income"},
				{name: "Lainnya", icon: "💵", color: "#95A5A6", type_: "income"},
			}

			for _, category := range categories {
				record := core.NewRecord(collection)
				record.Set("family_id", "")
				record.Set("is_master", true)
				record.Set("is_default", true)
				record.Set("name", category.name)
				record.Set("icon", category.icon)
				record.Set("color", category.color)
				record.Set("type", category.type_)

				if err := txApp.Save(record); err != nil {
					return err
				}
			}

			return nil
		})
	}, func(app core.App) error {
		_, err := app.DB().NewQuery("DELETE FROM categories WHERE is_master = 1 AND family_id = ''").Execute()
		return err
	})
}
