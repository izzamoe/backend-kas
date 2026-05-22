package service

import (
	"context"
	"strings"
	"testing"

	digiflazzclient "kas/internal/digiflazz"
	digiflazzdomain "kas/internal/domain/digiflazz"
	"kas/internal/repository"
	"kas/internal/utils"
)

type mockDigiflazzPascaProductService struct {
	product *digiflazzdomain.ProductDTO
	err     error
}

func (m *mockDigiflazzPascaProductService) SyncPricelistWithCredential(_ context.Context, _ *repository.DigiflazzCredentialRecord) (*SyncResult, error) {
	return nil, nil
}
func (m *mockDigiflazzPascaProductService) SearchProducts(_ string, _ *digiflazzdomain.ProductSearchRequest) ([]*digiflazzdomain.ProductDTO, error) {
	return nil, nil
}
func (m *mockDigiflazzPascaProductService) GetProductBySKU(_, sku string) (*digiflazzdomain.ProductDTO, error) {
	return m.product, m.err
}

type mockDigiflazzPascaCredentialRepo struct {
	secret *repository.DigiflazzCredentialRecord
}

func (m *mockDigiflazzPascaCredentialRepo) Create(*repository.DigiflazzCredentialCreateData) (*digiflazzdomain.CredentialDTO, error) {
	return nil, nil
}
func (m *mockDigiflazzPascaCredentialRepo) GetByID(string) (*digiflazzdomain.CredentialDTO, error) {
	return nil, nil
}
func (m *mockDigiflazzPascaCredentialRepo) GetSecretByID(string) (*repository.DigiflazzCredentialRecord, error) {
	return nil, nil
}
func (m *mockDigiflazzPascaCredentialRepo) GetByFamilyID(string) (*digiflazzdomain.CredentialDTO, error) {
	return nil, nil
}
func (m *mockDigiflazzPascaCredentialRepo) GetSecretByFamilyID(string) (*repository.DigiflazzCredentialRecord, error) {
	return nil, nil
}
func (m *mockDigiflazzPascaCredentialRepo) GetActiveByFamilyID(string) (*digiflazzdomain.CredentialDTO, error) {
	return nil, nil
}
func (m *mockDigiflazzPascaCredentialRepo) GetActiveSecretByFamilyID(string) (*repository.DigiflazzCredentialRecord, error) {
	return m.secret, nil
}
func (m *mockDigiflazzPascaCredentialRepo) GetSecretByWebhookTokenHash(string) (*repository.DigiflazzCredentialRecord, error) {
	return nil, nil
}
func (m *mockDigiflazzPascaCredentialRepo) ListByFamilyID(string, int, int) ([]*digiflazzdomain.CredentialDTO, error) {
	return nil, nil
}
func (m *mockDigiflazzPascaCredentialRepo) CountByFamilyID(string) (int, error) { return 0, nil }
func (m *mockDigiflazzPascaCredentialRepo) Update(string, *repository.DigiflazzCredentialUpdateData) (*digiflazzdomain.CredentialDTO, error) {
	return nil, nil
}
func (m *mockDigiflazzPascaCredentialRepo) Disable(string) (*digiflazzdomain.CredentialDTO, error) {
	return nil, nil
}
func (m *mockDigiflazzPascaCredentialRepo) Delete(string) error { return nil }
func (m *mockDigiflazzPascaCredentialRepo) ListAllActive() ([]*repository.DigiflazzCredentialRecord, error) {
	return nil, nil
}

type mockDigiflazzPascaEventRepo struct {
	created []*repository.DigiflazzEventCreateData
}

func (m *mockDigiflazzPascaEventRepo) Create(data *repository.DigiflazzEventCreateData) (*repository.DigiflazzEventRecord, error) {
	m.created = append(m.created, data)
	return &repository.DigiflazzEventRecord{ID: "event1", OrderID: data.OrderID, EventType: data.EventType}, nil
}
func (m *mockDigiflazzPascaEventRepo) GetByID(string) (*repository.DigiflazzEventRecord, error) {
	return nil, nil
}
func (m *mockDigiflazzPascaEventRepo) GetByFamilyAndID(string, string) (*repository.DigiflazzEventRecord, error) {
	return nil, nil
}
func (m *mockDigiflazzPascaEventRepo) ListByFamilyID(string, int, int) ([]*repository.DigiflazzEventRecord, error) {
	return nil, nil
}
func (m *mockDigiflazzPascaEventRepo) ExistsByOrderAndPayloadHash(string, string) (bool, error) {
	return false, nil
}

func newDigiflazzPascaCredential(t *testing.T) *repository.DigiflazzCredentialRecord {
	t.Helper()
	key := "12345678901234567890123456789012"
	t.Setenv(digiflazzCredentialEncryptionKeyEnv, key)
	ciphertext, err := utils.Encrypt("api-key", []byte(key))
	if err != nil {
		t.Fatalf("encrypt credential: %v", err)
	}
	return &repository.DigiflazzCredentialRecord{ID: "cred1", FamilyID: "fam1", Username: "user", APIKeyCiphertext: ciphertext, Testing: true, IsActive: true}
}

func TestDigiflazzPascaInquiryCreatesInquiryOrderAndPLNEvent(t *testing.T) {
	fake := digiflazzclient.NewFakeClient()
	fake.SetResponse("InqPasca", &digiflazzclient.TransactionResponse{
		RefID:        "REF-PASCA-1",
		CustomerNo:   "12345",
		CustomerName: "Budi PLN",
		BuyerSKUCode: "PLNPOST",
		Message:      "Tagihan tersedia",
		Status:       digiflazzclient.StatusPending,
		Rc:           "00",
		Price:        100000,
		Admin:        2500,
		SellingPrice: 102500,
	}, nil)

	var created repository.CreateDigiflazzOrderParams
	orderRepo := &mockDigiflazzOrderRepo{
		createFn: func(params repository.CreateDigiflazzOrderParams) (*digiflazzdomain.OrderDTO, error) {
			created = params
			return &digiflazzdomain.OrderDTO{ID: "order1", FamilyID: params.FamilyID, RefID: params.RefID, ProductCode: params.ProductCode, CustomerNo: params.CustomerNo, CustomerName: params.CustomerName, Status: params.Status, SellingPrice: params.Amount, RC: params.RC, Message: params.Message}, nil
		},
	}
	eventRepo := &mockDigiflazzPascaEventRepo{}
	svc := NewDigiflazzOrderService(orderRepo, DigiflazzOrderServiceDeps{
		CredentialRepo: &mockDigiflazzPascaCredentialRepo{secret: newDigiflazzPascaCredential(t)},
		ProductService: &mockDigiflazzPascaProductService{product: &digiflazzdomain.ProductDTO{Code: "PLNPOST", Name: "PLN Pascabayar", Brand: "PLN", Category: "Listrik", Price: 2500, Admin: 2500, Status: "active"}},
		EventRepo:      eventRepo,
		ClientFactory: func(string, string, bool) digiflazzclient.DigiflazzClient {
			return fake
		},
	})

	order, err := svc.CreatePostpaidInquiry(context.Background(), &digiflazzdomain.CreateOrderRequest{BuyerSKUCode: "PLNPOST", CustomerNo: " 12345 "}, "user1", "fam1")
	if err != nil {
		t.Fatalf("CreatePostpaidInquiry returned error: %v", err)
	}
	if order.Status != digiflazzdomain.OrderStatusInquiry || created.Status != digiflazzdomain.OrderStatusInquiry {
		t.Fatalf("expected inquiry status, got order=%+v created=%+v", order, created)
	}
	if created.Amount != 102500 || created.CustomerName != "Budi PLN" || created.IsPrepaid {
		t.Fatalf("unexpected inquiry snapshot: %+v", created)
	}
	if fake.CallCount("InqPasca") != 1 {
		t.Fatalf("expected pasca inquiry call, history=%+v", fake.History())
	}
	if len(eventRepo.created) != 1 || eventRepo.created[0].EventType != digiflazzdomain.EventTypeInquiry || eventRepo.created[0].StatusAfter != "inquiry" {
		t.Fatalf("expected inquiry audit event, got %+v", eventRepo.created)
	}
}

func TestDigiflazzPascaPayDetectsAmountChange(t *testing.T) {
	fake := digiflazzclient.NewFakeClient()
	fake.SetResponse("PayPasca", &digiflazzclient.TransactionResponse{RefID: "REF1", CustomerNo: "12345", BuyerSKUCode: "PDAM", Status: digiflazzclient.StatusSukses, Rc: "00", Price: 120000, Admin: 2500, SellingPrice: 122500}, nil)
	updated := false
	orderRepo := &mockDigiflazzOrderRepo{
		getByIDFn: func(string, string) (*digiflazzdomain.OrderDTO, error) {
			return &digiflazzdomain.OrderDTO{ID: "order1", FamilyID: "fam1", ProductCode: "PDAM", CustomerNo: "12345", RefID: "REF1", Status: digiflazzdomain.OrderStatusInquiry, SellingPrice: 100000}, nil
		},
		updateStatusFn: func(string, string, repository.UpdateDigiflazzOrderStatusParams) (*digiflazzdomain.OrderDTO, error) {
			updated = true
			return nil, nil
		},
	}
	eventRepo := &mockDigiflazzPascaEventRepo{}
	svc := NewDigiflazzOrderService(orderRepo, DigiflazzOrderServiceDeps{
		CredentialRepo: &mockDigiflazzPascaCredentialRepo{secret: newDigiflazzPascaCredential(t)},
		EventRepo:      eventRepo,
		ClientFactory: func(string, string, bool) digiflazzclient.DigiflazzClient {
			return fake
		},
	})

	_, err := svc.PayPostpaidOrder(context.Background(), "fam1", "user1", "order1")
	if err == nil || !strings.Contains(err.Error(), "fresh inquiry") {
		t.Fatalf("expected fresh inquiry error, got %v", err)
	}
	if updated {
		t.Fatal("amount mismatch must not update order status")
	}
	if len(eventRepo.created) != 1 || eventRepo.created[0].StatusBefore != "inquiry" || eventRepo.created[0].StatusAfter != "inquiry" {
		t.Fatalf("expected mismatch audit event preserving status, got %+v", eventRepo.created)
	}
}

func TestDigiflazzPascaStatusUpdatesPendingOrder(t *testing.T) {
	fake := digiflazzclient.NewFakeClient()
	fake.SetResponse("StatusPasca", &digiflazzclient.TransactionResponse{RefID: "REF1", CustomerNo: "12345", BuyerSKUCode: "PDAM", Status: digiflazzclient.StatusSukses, Rc: "00", Message: "Sukses", Sn: "SN123", Price: 100000}, nil)
	orderRepo := &mockDigiflazzOrderRepo{
		getByIDFn: func(string, string) (*digiflazzdomain.OrderDTO, error) {
			return &digiflazzdomain.OrderDTO{ID: "order1", FamilyID: "fam1", ProductCode: "PDAM", CustomerNo: "12345", RefID: "REF1", Status: digiflazzdomain.OrderStatusPending}, nil
		},
		updateStatusFn: func(familyID, id string, params repository.UpdateDigiflazzOrderStatusParams) (*digiflazzdomain.OrderDTO, error) {
			if params.Status != digiflazzdomain.OrderStatusSuccess || params.SN != "SN123" {
				t.Fatalf("unexpected status update params: %+v", params)
			}
			return &digiflazzdomain.OrderDTO{ID: id, FamilyID: familyID, Status: params.Status, SN: params.SN}, nil
		},
	}
	svc := NewDigiflazzOrderService(orderRepo, DigiflazzOrderServiceDeps{
		CredentialRepo: &mockDigiflazzPascaCredentialRepo{secret: newDigiflazzPascaCredential(t)},
		EventRepo:      &mockDigiflazzPascaEventRepo{},
		ClientFactory: func(string, string, bool) digiflazzclient.DigiflazzClient {
			return fake
		},
	})

	updated, err := svc.CheckPostpaidStatus(context.Background(), "fam1", "user1", "order1")
	if err != nil {
		t.Fatalf("CheckPostpaidStatus returned error: %v", err)
	}
	if updated.Status != digiflazzdomain.OrderStatusSuccess || fake.CallCount("StatusPasca") != 1 {
		t.Fatalf("unexpected status result order=%+v calls=%d", updated, fake.CallCount("StatusPasca"))
	}
}
