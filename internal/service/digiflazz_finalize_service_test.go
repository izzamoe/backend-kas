package service

import (
	"testing"

	"github.com/pocketbase/pocketbase/core"

	"kas/internal/domain"
	digiflazzdomain "kas/internal/domain/digiflazz"
	"kas/internal/repository"
)

type digiflazzFinalizeFixture struct {
	familyID        string
	userID          string
	hiburanCategory *core.Record
	orderRepo       repository.DigiflazzOrderRepository
	transactionRepo repository.TransactionRepository
	svc             DigiflazzOrderService
}

func setupDigiflazzFinalizeFixture(t *testing.T) *digiflazzFinalizeFixture {
	t.Helper()
	app, familyID, userID := setupDigiflazzPriceServiceTestApp(t)

	hiburan := createServiceTestRecord(t, app, "categories", map[string]any{
		"family_id":  familyID,
		"name":       "Hiburan",
		"icon":       "🎉",
		"color":      "#5DCAA5",
		"is_default": true,
		"is_master":  false,
		"type":       "expense",
	})
	createServiceTestRecord(t, app, "categories", map[string]any{
		"family_id":  familyID,
		"name":       "Lainnya",
		"icon":       "📦",
		"color":      "#888780",
		"is_default": true,
		"is_master":  false,
		"type":       "expense",
	})

	orderRepo := repository.NewDigiflazzOrderRepository(app)
	transactionRepo := repository.NewTransactionRepository(app)
	categoryRepo := repository.NewCategoryRepository(app)
	svc := NewDigiflazzOrderService(orderRepo, DigiflazzOrderServiceDeps{
		TransactionRepo: transactionRepo,
		CategoryRepo:    categoryRepo,
	})

	return &digiflazzFinalizeFixture{
		familyID:        familyID,
		userID:          userID,
		hiburanCategory: hiburan,
		orderRepo:       orderRepo,
		transactionRepo: transactionRepo,
		svc:             svc,
	}
}

func (fx *digiflazzFinalizeFixture) createOrder(t *testing.T, status digiflazzdomain.OrderStatus) *digiflazzdomain.OrderDTO {
	t.Helper()
	order, err := fx.orderRepo.Create(repository.CreateDigiflazzOrderParams{
		FamilyID:        fx.familyID,
		UserID:          fx.userID,
		CredentialID:    "cred1",
		EventType:       digiflazzdomain.EventTypeTopup,
		RefID:           "DFZ-FINALIZE-1",
		ProductCode:     "TSEL10",
		ProductName:     "Telkomsel 10K",
		ProductBrand:    "Telkomsel",
		ProductCategory: "Pulsa",
		CustomerNo:      "08123456789",
		Price:           10000,
		Admin:           1000,
		Amount:          11000,
		Status:          status,
		IsPrepaid:       true,
	})
	if err != nil {
		t.Fatalf("create digiflazz order: %v", err)
	}
	return order
}

func TestDigiflazzFinalizeCreatesExpense(t *testing.T) {
	fx := setupDigiflazzFinalizeFixture(t)
	order := fx.createOrder(t, digiflazzdomain.OrderStatusProcessing)

	finalized, err := fx.svc.UpdateStatus(fx.familyID, order.ID, digiflazzdomain.OrderStatusSuccess, &digiflazzdomain.OrderResponseDTO{
		Message:      "Sukses",
		RC:           "00",
		SN:           "SN123",
		SellingPrice: 11000,
	})
	if err != nil {
		t.Fatalf("UpdateStatus success returned error: %v", err)
	}
	if finalized.TransactionID == "" {
		t.Fatalf("expected finalized order to be linked to a transaction: %+v", finalized)
	}

	transactions, err := fx.transactionRepo.GetByFamilyID(fx.familyID, 10, 0)
	if err != nil {
		t.Fatalf("GetByFamilyID returned error: %v", err)
	}
	if len(transactions) != 1 {
		t.Fatalf("expected exactly one transaction, got %d: %+v", len(transactions), transactions)
	}
	tx := transactions[0]
	if tx.ID != finalized.TransactionID {
		t.Fatalf("order transaction_id %q does not match created transaction %q", finalized.TransactionID, tx.ID)
	}
	if tx.Type != domain.TransactionTypeExpense || tx.Amount != 11000 || tx.CategoryID != fx.hiburanCategory.Id {
		t.Fatalf("unexpected transaction data: %+v", tx)
	}
	if tx.Note != "Pembelian Telkomsel 10K - 08123456789" || tx.CreatedBy != fx.userID || tx.FamilyID != fx.familyID {
		t.Fatalf("unexpected transaction identity/note: %+v", tx)
	}
}

func TestDigiflazzFinalizeIdempotent(t *testing.T) {
	fx := setupDigiflazzFinalizeFixture(t)
	order := fx.createOrder(t, digiflazzdomain.OrderStatusSuccess)

	first, err := fx.svc.FinalizeSuccessOrder(order.ID)
	if err != nil {
		t.Fatalf("first FinalizeSuccessOrder returned error: %v", err)
	}
	second, err := fx.svc.FinalizeSuccessOrder(order.ID)
	if err != nil {
		t.Fatalf("second FinalizeSuccessOrder returned error: %v", err)
	}
	if first.TransactionID == "" || second.TransactionID != first.TransactionID {
		t.Fatalf("expected stable transaction link, first=%+v second=%+v", first, second)
	}

	transactions, err := fx.transactionRepo.GetByFamilyID(fx.familyID, 10, 0)
	if err != nil {
		t.Fatalf("GetByFamilyID returned error: %v", err)
	}
	if len(transactions) != 1 {
		t.Fatalf("expected idempotent finalization to create one transaction, got %d: %+v", len(transactions), transactions)
	}
}
