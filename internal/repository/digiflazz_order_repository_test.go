package repository

import (
	"errors"
	"testing"

	"github.com/pocketbase/pocketbase/core"

	digiflazzdomain "kas/internal/domain/digiflazz"
	_ "kas/migrations"
)

func TestDigiflazzOrderRepositoryCreateGetListAndUpdate(t *testing.T) {
	app := setupRepositoryTestApp(t)
	familyID, userID, _ := createRepositoryFixtures(t, app)
	otherFamily := createTestRecord(t, app, "families", map[string]any{
		"name":        "Other Family",
		"invite_code": "OTHER001",
	})
	repo := NewDigiflazzOrderRepository(app)

	created, err := repo.Create(CreateDigiflazzOrderParams{
		FamilyID:        familyID,
		UserID:          userID,
		CredentialID:    "cred123",
		EventType:       digiflazzdomain.EventTypeTopup,
		RefID:           "DFZ-ABC123-1700000000-XYZ789",
		ProductCode:     "PLN20",
		ProductName:     "PLN 20K",
		ProductBrand:    "PLN",
		ProductCategory: "PLN",
		CustomerNo:      "08123456789",
		CustomerName:    "Budi",
		Price:           20000,
		Admin:           1500,
		Amount:          21500,
		Status:          digiflazzdomain.OrderStatusPending,
		IsPrepaid:       true,
	})
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}
	if created.ID == "" || created.FamilyID != familyID || created.CredentialID != "cred123" || created.RefID != "DFZ-ABC123-1700000000-XYZ789" {
		t.Fatalf("unexpected created order identity: %+v", created)
	}
	if created.ProductCode != "PLN20" || created.CustomerNo != "08123456789" || created.CustomerName != "Budi" {
		t.Fatalf("unexpected created order snapshot: %+v", created)
	}
	if created.Price != 20000 || created.Admin != 1500 || created.SellingPrice != 21500 || created.Status != digiflazzdomain.OrderStatusPending {
		t.Fatalf("unexpected created order totals/status: %+v", created)
	}

	duplicate, err := repo.Create(CreateDigiflazzOrderParams{
		FamilyID: familyID,
		UserID:   userID,
		RefID:    created.RefID,
		Status:   digiflazzdomain.OrderStatusPending,
	})
	if err != nil {
		t.Fatalf("duplicate Create returned error: %v", err)
	}
	if duplicate.ID != created.ID {
		t.Fatalf("expected duplicate create to return existing order %s, got %s", created.ID, duplicate.ID)
	}

	found, err := repo.GetByID(familyID, created.ID)
	if err != nil {
		t.Fatalf("GetByID returned error: %v", err)
	}
	if found == nil || found.ID != created.ID || found.FamilyID != familyID {
		t.Fatalf("unexpected found order: %+v", found)
	}

	wrongFamilyFound, err := repo.GetByID(otherFamily.Id, created.ID)
	if !errors.Is(err, digiflazzdomain.ErrOrderNotFound) {
		t.Fatalf("expected cross-family GetByID to return ErrOrderNotFound, got err=%v result=%+v", err, wrongFamilyFound)
	}

	byRef, err := repo.GetByRefID(familyID, created.RefID)
	if err != nil {
		t.Fatalf("GetByRefID returned error: %v", err)
	}
	if byRef == nil || byRef.ID != created.ID {
		t.Fatalf("unexpected order by ref: %+v", byRef)
	}

	crossFamilyByRef, err := repo.GetByRefID(otherFamily.Id, created.RefID)
	if err != nil {
		t.Fatalf("cross-family GetByRefID returned error: %v", err)
	}
	if crossFamilyByRef != nil {
		t.Fatalf("expected cross-family GetByRefID nil, got %+v", crossFamilyByRef)
	}

	otherOrder := createDigiflazzOrderRecord(t, app, otherFamily.Id, userID, "DFZ-OTHER1-1700000001-ABC123")
	orders, err := repo.ListByFamily(familyID, 20, 0)
	if err != nil {
		t.Fatalf("ListByFamily returned error: %v", err)
	}
	if len(orders) != 1 || orders[0].ID != created.ID || orders[0].ID == otherOrder.Id {
		t.Fatalf("expected list scoped to one family, got %+v", orders)
	}

	updated, err := repo.UpdateStatus(familyID, created.ID, UpdateDigiflazzOrderStatusParams{
		Status:  digiflazzdomain.OrderStatusProcessing,
		Message: "diproses",
		RC:      "01",
		SN:      "SN123",
	})
	if err != nil {
		t.Fatalf("UpdateStatus returned error: %v", err)
	}
	if updated.Status != digiflazzdomain.OrderStatusProcessing || updated.Message != "diproses" || updated.RC != "01" || updated.SN != "SN123" {
		t.Fatalf("unexpected updated status order: %+v", updated)
	}

	_, err = repo.UpdateStatus(otherFamily.Id, created.ID, UpdateDigiflazzOrderStatusParams{Status: digiflazzdomain.OrderStatusSuccess})
	if !errors.Is(err, digiflazzdomain.ErrOrderNotFound) {
		t.Fatalf("expected cross-family UpdateStatus to return ErrOrderNotFound, got %v", err)
	}
}

func createDigiflazzOrderRecord(t *testing.T, app core.App, familyID, userID, refID string) *core.Record {
	t.Helper()

	return createTestRecord(t, app, "digiflazz_orders", map[string]any{
		"family_id":      familyID,
		"created_by":     userID,
		"ref_id":         refID,
		"buyer_sku_code": "PULSA10",
		"customer_no":    "0811111111",
		"product_name":   "Pulsa 10K",
		"category":       "Pulsa",
		"status":         "pending",
		"price":          10000,
		"admin":          0,
		"total":          10000,
		"is_prepaid":     true,
	})
}
