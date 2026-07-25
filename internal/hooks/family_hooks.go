package hooks

import (
	"kas/generated"
	"kas/internal/repository"
)

func RegisterFamilyHooks(ph *generated.ProxyHooks, categoryRepo repository.CategoryRepository) {
	ph.OnFamiliesAfterCreateSuccess.BindFunc(func(e *generated.FamiliesEvent) error {
		familyID := e.PRecord.Id
		if err := categoryRepo.SeedMasterCategories(e.App, familyID); err != nil {
			e.App.Logger().Warn("failed to seed master categories", "family_id", familyID, "error", err)
		}
		_, err := e.App.DB().NewQuery(
			`INSERT INTO family_balances (id, family_id, balance, total_income, total_expense) VALUES ({:id}, {:familyID}, 0, 0, 0)`,
		).Bind(map[string]any{
			"id":       "bal_" + familyID,
			"familyID": familyID,
		}).Execute()
		if err != nil {
			e.App.Logger().Warn("failed to initialize family balance", "family_id", familyID, "error", err)
		}
		return e.Next()
	})
}
