package service

import (
	"errors"
	"testing"

	digiflazzdomain "kas/internal/domain/digiflazz"
	"kas/internal/repository"
	_ "kas/migrations"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
)

func setupDigiflazzPriceServiceTestApp(t *testing.T) (*tests.TestApp, string, string) {
	t.Helper()
	app, err := tests.NewTestApp()
	if err != nil {
		t.Fatalf("failed to create test app: %v", err)
	}
	t.Cleanup(app.Cleanup)

	familyCol, err := app.FindCollectionByNameOrId("families")
	if err != nil {
		t.Fatalf("find families collection: %v", err)
	}
	family := core.NewRecord(familyCol)
	family.Set("name", "Test Family")
	family.Set("invite_code", "SYNCTEST01")
	if err := app.Save(family); err != nil {
		t.Fatalf("save family: %v", err)
	}

	userCol, err := app.FindCollectionByNameOrId("users")
	if err != nil {
		t.Fatalf("find users collection: %v", err)
	}
	user := core.NewRecord(userCol)
	user.Set("email", "owner@synctest.com")
	user.Set("verified", true)
	user.Set("name", "Sync Owner")
	user.SetPassword("password123456")
	if err := app.Save(user); err != nil {
		t.Fatalf("save user: %v", err)
	}

	memberCol, err := app.FindCollectionByNameOrId("family_members")
	if err != nil {
		t.Fatalf("find family_members collection: %v", err)
	}
	member := core.NewRecord(memberCol)
	member.Set("family_id", family.Id)
	member.Set("user_id", user.Id)
	member.Set("role", "owner")
	if err := app.Save(member); err != nil {
		t.Fatalf("save family member: %v", err)
	}

	return app, family.Id, user.Id
}

func seedTestProducts(t *testing.T, repo repository.DigiflazzProductRepository, familyID string) {
	t.Helper()
	seeds := []*repository.UpsertProductInput{
		{FamilyID: familyID, CredentialID: "", ProductName: "Telkomsel 10K", Category: "Pulsa", Brand: "Telkomsel", Type: "Reguler", BuyerSKUCode: "tsel10", Price: 11000, BuyerProductStatus: "active", IsPrepaid: true},
		{FamilyID: familyID, CredentialID: "", ProductName: "Telkomsel 20K", Category: "Pulsa", Brand: "Telkomsel", Type: "Reguler", BuyerSKUCode: "tsel20", Price: 21000, BuyerProductStatus: "active", IsPrepaid: true},
		{FamilyID: familyID, CredentialID: "", ProductName: "PLN Pascabayar", Category: "PLN", Brand: "PLN", Type: "postpaid", BuyerSKUCode: "pln-pasca", Price: 1, Admin: 2500, BuyerProductStatus: "active", IsPrepaid: true},
	}
	for _, s := range seeds {
		if _, err := repo.Upsert(s); err != nil {
			t.Fatalf("seed product %s: %v", s.BuyerSKUCode, err)
		}
	}
}

func TestDigiflazzPriceService_SearchProducts(t *testing.T) {
	app, familyID, _ := setupDigiflazzPriceServiceTestApp(t)
	repo := repository.NewDigiflazzProductRepository(app)
	svc := NewDigiflazzProductService(app, repo, nil, nil)

	seedTestProducts(t, repo, familyID)

	all, err := svc.SearchProducts(familyID, nil)
	if err != nil {
		t.Fatalf("SearchProducts nil returned error: %v", err)
	}
	if len(all) != 3 {
		t.Fatalf("expected 3 products, got %d", len(all))
	}

	byBrand, err := svc.SearchProducts(familyID, &digiflazzdomain.ProductSearchRequest{Brand: "Telkomsel", Limit: 10})
	if err != nil {
		t.Fatalf("SearchProducts by brand returned error: %v", err)
	}
	if len(byBrand) != 2 {
		t.Fatalf("expected 2 Telkomsel products, got %d", len(byBrand))
	}

	byCategory, err := svc.SearchProducts(familyID, &digiflazzdomain.ProductSearchRequest{Category: "PLN", Limit: 10})
	if err != nil {
		t.Fatalf("SearchProducts by category returned error: %v", err)
	}
	if len(byCategory) != 1 || byCategory[0].Code != "pln-pasca" {
		t.Fatalf("expected 1 PLN product, got %+v", byCategory)
	}
}

func TestDigiflazzPriceService_GetProductBySKU(t *testing.T) {
	app, familyID, _ := setupDigiflazzPriceServiceTestApp(t)
	repo := repository.NewDigiflazzProductRepository(app)
	svc := NewDigiflazzProductService(app, repo, nil, nil)

	_, err := svc.GetProductBySKU(familyID, "")
	if err == nil {
		t.Fatal("expected error for empty SKU, got nil")
	}

	_, err = svc.GetProductBySKU(familyID, "missing-sku")
	if !errors.Is(err, digiflazzdomain.ErrProductNotFound) {
		t.Fatalf("expected ErrProductNotFound for missing SKU, got %v", err)
	}

	seedTestProducts(t, repo, familyID)

	product, err := svc.GetProductBySKU(familyID, "tsel20")
	if err != nil || product == nil {
		t.Fatalf("expected tsel20 in cache, got nil/err: %v", err)
	}
	if product.Code != "tsel20" || product.Price != 21000 {
		t.Fatalf("unexpected product: %+v", product)
	}
}
