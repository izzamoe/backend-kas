package repository

import (
	"errors"
	"testing"

	digiflazzdomain "kas/internal/domain/digiflazz"
)

func TestDigiflazzPriceRepository_Upsert(t *testing.T) {
	app := setupRepositoryTestApp(t)
	family := createTestRecord(t, app, "families", map[string]any{"name": "Test Family", "invite_code": "UPSERT01"})
	repo := NewDigiflazzProductRepository(app)

	input := &UpsertProductInput{
		FamilyID:           family.Id,
		CredentialID:       "",
		ProductName:        "Telkomsel 10K",
		Category:           "Pulsa",
		Brand:              "Telkomsel",
		Type:               "Reguler",
		BuyerSKUCode:       "tsel10",
		Price:              11000,
		BuyerProductStatus: "active",
		Stock:              100,
		IsPrepaid:          true,
	}

	dto, err := repo.Upsert(input)
	if err != nil {
		t.Fatalf("Upsert returned error: %v", err)
	}
	if dto == nil || dto.Code != "tsel10" || dto.Name != "Telkomsel 10K" || dto.Price != 11000 {
		t.Fatalf("unexpected upsert result: %+v", dto)
	}

	input.Price = 12000
	input.ProductName = "Telkomsel 10K Updated"

	dto2, err := repo.Upsert(input)
	if err != nil {
		t.Fatalf("second Upsert returned error: %v", err)
	}
	if dto2.Price != 12000 || dto2.Name != "Telkomsel 10K Updated" {
		t.Fatalf("expected updated product after second upsert, got: %+v", dto2)
	}

	results, err := repo.Search(family.Id, &digiflazzdomain.ProductSearchRequest{Query: "tsel10", Limit: 10})
	if err != nil {
		t.Fatalf("Search returned error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 record after upsert idempotency, got %d", len(results))
	}
}

func TestDigiflazzPriceRepository_GetBySKU(t *testing.T) {
	app := setupRepositoryTestApp(t)
	family := createTestRecord(t, app, "families", map[string]any{"name": "Test Family", "invite_code": "GETSKU01"})
	repo := NewDigiflazzProductRepository(app)

	_, err := repo.GetBySKU(family.Id, "nonexistent-sku")
	if !errors.Is(err, digiflazzdomain.ErrProductNotFound) {
		t.Fatalf("expected ErrProductNotFound for missing SKU, got %v", err)
	}

	if _, upsertErr := repo.Upsert(&UpsertProductInput{
		FamilyID:     family.Id,
		CredentialID: "",
		ProductName:  "XL 5K",
		Category:     "Pulsa",
		Brand:        "XL",
		Type:         "Reguler",
		BuyerSKUCode: "xl5",
		Price:        6000,
		IsPrepaid:    true,
	}); upsertErr != nil {
		t.Fatalf("Upsert returned error: %v", upsertErr)
	}

	found, err := repo.GetBySKU(family.Id, "xl5")
	if err != nil {
		t.Fatalf("GetBySKU returned error: %v", err)
	}
	if found == nil || found.Code != "xl5" || found.Price != 6000 || found.Brand != "XL" {
		t.Fatalf("unexpected GetBySKU result: %+v", found)
	}
}

func TestDigiflazzPriceRepository_Search(t *testing.T) {
	app := setupRepositoryTestApp(t)
	family := createTestRecord(t, app, "families", map[string]any{"name": "Test Family", "invite_code": "SEARCH01"})
	repo := NewDigiflazzProductRepository(app)

	seeds := []*UpsertProductInput{
		{FamilyID: family.Id, CredentialID: "", ProductName: "Telkomsel 10K", Category: "Pulsa", Brand: "Telkomsel", Type: "Reguler", BuyerSKUCode: "tsel10", Price: 11000, BuyerProductStatus: "active", IsPrepaid: true},
		{FamilyID: family.Id, CredentialID: "", ProductName: "Telkomsel 20K", Category: "Pulsa", Brand: "Telkomsel", Type: "Reguler", BuyerSKUCode: "tsel20", Price: 21000, BuyerProductStatus: "active", IsPrepaid: true},
		{FamilyID: family.Id, CredentialID: "", ProductName: "XL 10K", Category: "Pulsa", Brand: "XL", Type: "Reguler", BuyerSKUCode: "xl10", Price: 11500, BuyerProductStatus: "active", IsPrepaid: true},
		{FamilyID: family.Id, CredentialID: "", ProductName: "PLN Pasca", Category: "PLN", Brand: "PLN", Type: "postpaid", BuyerSKUCode: "pln1", Price: 2500, Admin: 2500, BuyerProductStatus: "active", IsPrepaid: true},
	}
	for _, p := range seeds {
		if _, err := repo.Upsert(p); err != nil {
			t.Fatalf("Upsert seed %s returned error: %v", p.BuyerSKUCode, err)
		}
	}

	all, err := repo.Search(family.Id, &digiflazzdomain.ProductSearchRequest{Limit: 20})
	if err != nil {
		t.Fatalf("Search all returned error: %v", err)
	}
	if len(all) != 4 {
		t.Fatalf("expected 4 total products, got %d", len(all))
	}

	byBrand, err := repo.Search(family.Id, &digiflazzdomain.ProductSearchRequest{Brand: "Telkomsel", Limit: 10})
	if err != nil {
		t.Fatalf("Search by brand returned error: %v", err)
	}
	if len(byBrand) != 2 {
		t.Fatalf("expected 2 Telkomsel products, got %d", len(byBrand))
	}

	byCategory, err := repo.Search(family.Id, &digiflazzdomain.ProductSearchRequest{Category: "PLN", Limit: 10})
	if err != nil {
		t.Fatalf("Search by category returned error: %v", err)
	}
	if len(byCategory) != 1 || byCategory[0].Code != "pln1" {
		t.Fatalf("expected 1 PLN product, got %+v", byCategory)
	}

	byType, err := repo.Search(family.Id, &digiflazzdomain.ProductSearchRequest{Type: "postpaid", Limit: 10})
	if err != nil {
		t.Fatalf("Search by type returned error: %v", err)
	}
	if len(byType) != 1 || byType[0].Code != "pln1" {
		t.Fatalf("expected 1 postpaid product, got %+v", byType)
	}

	byQuery, err := repo.Search(family.Id, &digiflazzdomain.ProductSearchRequest{Query: "Telkomsel", Limit: 10})
	if err != nil {
		t.Fatalf("Search by query returned error: %v", err)
	}
	if len(byQuery) != 2 {
		t.Fatalf("expected 2 Telkomsel query results, got %d", len(byQuery))
	}

	byStatus, err := repo.Search(family.Id, &digiflazzdomain.ProductSearchRequest{Status: "active", Limit: 20})
	if err != nil {
		t.Fatalf("Search by status returned error: %v", err)
	}
	if len(byStatus) != 4 {
		t.Fatalf("expected 4 active products, got %d", len(byStatus))
	}

	empty, err := repo.Search(family.Id, &digiflazzdomain.ProductSearchRequest{Brand: "nonexistent-brand", Limit: 10})
	if err != nil {
		t.Fatalf("Search nonexistent brand returned error: %v", err)
	}
	if len(empty) != 0 {
		t.Fatalf("expected 0 results for nonexistent brand, got %d", len(empty))
	}

	nilReq, err := repo.Search(family.Id, nil)
	if err != nil {
		t.Fatalf("Search nil req returned error: %v", err)
	}
	if len(nilReq) != 4 {
		t.Fatalf("expected 4 results for nil req, got %d", len(nilReq))
	}
}
