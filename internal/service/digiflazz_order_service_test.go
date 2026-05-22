package service

import (
	"context"
	"errors"
	digiflazzclient "kas/internal/digiflazz"
	digiflazzdomain "kas/internal/domain/digiflazz"
	"kas/internal/repository"
	"kas/internal/utils"
	"regexp"
	"strings"
	"testing"
	"time"
)

type mockDigiflazzOrderRepo struct {
	createFn       func(params repository.CreateDigiflazzOrderParams) (*digiflazzdomain.OrderDTO, error)
	getByIDFn      func(familyID, id string) (*digiflazzdomain.OrderDTO, error)
	getByRefIDFn   func(familyID, refID string) (*digiflazzdomain.OrderDTO, error)
	updateStatusFn func(familyID, id string, params repository.UpdateDigiflazzOrderStatusParams) (*digiflazzdomain.OrderDTO, error)
	listByFamilyFn func(familyID string, limit, offset int) ([]*digiflazzdomain.OrderDTO, error)
}

func (m *mockDigiflazzOrderRepo) Create(params repository.CreateDigiflazzOrderParams) (*digiflazzdomain.OrderDTO, error) {
	if m.createFn != nil {
		return m.createFn(params)
	}
	return nil, nil
}

func (m *mockDigiflazzOrderRepo) GetByID(familyID, id string) (*digiflazzdomain.OrderDTO, error) {
	if m.getByIDFn != nil {
		return m.getByIDFn(familyID, id)
	}
	return nil, nil
}

func (m *mockDigiflazzOrderRepo) GetByRefID(familyID, refID string) (*digiflazzdomain.OrderDTO, error) {
	if m.getByRefIDFn != nil {
		return m.getByRefIDFn(familyID, refID)
	}
	return nil, nil
}

func (m *mockDigiflazzOrderRepo) UpdateStatus(familyID, id string, params repository.UpdateDigiflazzOrderStatusParams) (*digiflazzdomain.OrderDTO, error) {
	if m.updateStatusFn != nil {
		return m.updateStatusFn(familyID, id, params)
	}
	return nil, nil
}

func (m *mockDigiflazzOrderRepo) ListByFamily(familyID string, limit, offset int) ([]*digiflazzdomain.OrderDTO, error) {
	if m.listByFamilyFn != nil {
		return m.listByFamilyFn(familyID, limit, offset)
	}
	return nil, nil
}

func (m *mockDigiflazzOrderRepo) ListPendingForPoll(createdAfter time.Time, limit int) ([]*digiflazzdomain.OrderDTO, error) {
	return nil, nil
}

type fakeOrderTestProductSvc struct {
	getProductBySKUFn func(familyID, sku string) (*digiflazzdomain.ProductDTO, error)
}

func (f *fakeOrderTestProductSvc) SyncPricelistWithCredential(ctx context.Context, credential *repository.DigiflazzCredentialRecord) (*SyncResult, error) {
	return nil, nil
}
func (f *fakeOrderTestProductSvc) SearchProducts(familyID string, req *digiflazzdomain.ProductSearchRequest) ([]*digiflazzdomain.ProductDTO, error) {
	return nil, nil
}
func (f *fakeOrderTestProductSvc) GetProductBySKU(familyID, sku string) (*digiflazzdomain.ProductDTO, error) {
	if f.getProductBySKUFn != nil {
		return f.getProductBySKUFn(familyID, sku)
	}
	return nil, nil
}

type fakeOrderTestCredRepo struct {
	getActiveSecretFn func(familyID string) (*repository.DigiflazzCredentialRecord, error)
}

func (f *fakeOrderTestCredRepo) Create(data *repository.DigiflazzCredentialCreateData) (*digiflazzdomain.CredentialDTO, error) {
	return nil, nil
}
func (f *fakeOrderTestCredRepo) Update(id string, data *repository.DigiflazzCredentialUpdateData) (*digiflazzdomain.CredentialDTO, error) {
	return nil, nil
}
func (f *fakeOrderTestCredRepo) GetByFamilyID(familyID string) (*digiflazzdomain.CredentialDTO, error) {
	return nil, nil
}
func (f *fakeOrderTestCredRepo) GetSecretByFamilyID(familyID string) (*repository.DigiflazzCredentialRecord, error) {
	return nil, nil
}
func (f *fakeOrderTestCredRepo) GetActiveSecretByFamilyID(familyID string) (*repository.DigiflazzCredentialRecord, error) {
	if f.getActiveSecretFn != nil {
		return f.getActiveSecretFn(familyID)
	}
	return nil, nil
}
func (f *fakeOrderTestCredRepo) GetSecretByWebhookTokenHash(hash string) (*repository.DigiflazzCredentialRecord, error) {
	return nil, nil
}
func (f *fakeOrderTestCredRepo) GetActiveByFamilyID(familyID string) (*digiflazzdomain.CredentialDTO, error) {
	return nil, nil
}
func (f *fakeOrderTestCredRepo) CountByFamilyID(familyID string) (int, error) { return 0, nil }
func (f *fakeOrderTestCredRepo) Delete(id string) error                        { return nil }
func (f *fakeOrderTestCredRepo) GetByID(id string) (*digiflazzdomain.CredentialDTO, error) {
	return nil, nil
}
func (f *fakeOrderTestCredRepo) GetSecretByID(id string) (*repository.DigiflazzCredentialRecord, error) {
	return nil, nil
}
func (f *fakeOrderTestCredRepo) ListByFamilyID(familyID string, limit, offset int) ([]*digiflazzdomain.CredentialDTO, error) {
	return nil, nil
}
func (f *fakeOrderTestCredRepo) Disable(id string) (*digiflazzdomain.CredentialDTO, error) {
	return nil, nil
}
func (f *fakeOrderTestCredRepo) ListAllActive() ([]*repository.DigiflazzCredentialRecord, error) {
	return nil, nil
}

func TestDigiflazzOrderServiceCreateOrderGeneratesRefAndStoresSnapshot(t *testing.T) {
	t.Setenv(digiflazzCredentialEncryptionKeyEnv, "test-key")
	ciphertext, encErr := utils.Encrypt("test-api-key", []byte("test-key"))
	if encErr != nil {
		t.Fatalf("failed to encrypt test api key: %v", encErr)
	}
	var captured repository.CreateDigiflazzOrderParams
	repo := &mockDigiflazzOrderRepo{
		createFn: func(params repository.CreateDigiflazzOrderParams) (*digiflazzdomain.OrderDTO, error) {
			captured = params
			return &digiflazzdomain.OrderDTO{ID: "order1", FamilyID: params.FamilyID, RefID: params.RefID, Status: params.Status}, nil
		},
	}
	productSvc := &fakeOrderTestProductSvc{
		getProductBySKUFn: func(familyID, sku string) (*digiflazzdomain.ProductDTO, error) {
			return &digiflazzdomain.ProductDTO{
				Code: "PLN20", Name: "PLN 20K", Brand: "PLN", Category: "Listrik",
				Price: 20000, Admin: 1500, IsPrepaid: true, Status: "active",
			}, nil
		},
	}
	credRepo := &fakeOrderTestCredRepo{
		getActiveSecretFn: func(familyID string) (*repository.DigiflazzCredentialRecord, error) {
			return &repository.DigiflazzCredentialRecord{
				ID: "cred1", FamilyID: familyID, Username: "buyer",
				APIKeyCiphertext: ciphertext, IsActive: true,
			}, nil
		},
	}
	fake := digiflazzclient.NewFakeClient()
	fake.SetResponse("Topup", digiflazzclient.TransactionResponse{Rc: "03", Price: 20000, Admin: 1500, SellingPrice: 22500}, nil)
	service := NewDigiflazzOrderService(repo, DigiflazzOrderServiceDeps{
		ProductService: productSvc,
		CredentialRepo: credRepo,
		ClientFactory:  func(username, apiKey string, testing bool) digiflazzclient.DigiflazzClient { return fake },
	})
	order, err := service.CreateOrder(context.Background(), "fam123456789", "user1", digiflazzdomain.CreateOrderRequest{
		BuyerSKUCode: "PLN20",
		CustomerNo:   " 08123456789 ",
	})
	if err != nil {
		t.Fatalf("CreateOrder returned error: %v", err)
	}
	if order.ID != "order1" || order.Status != digiflazzdomain.OrderStatusPending {
		t.Fatalf("unexpected created order: %+v", order)
	}
	if matched := regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`).MatchString(captured.RefID); !matched {
		t.Fatalf("ref id %q does not match expected UUID format", captured.RefID)
	}
	if captured.FamilyID != "fam123456789" || captured.UserID != "user1" || captured.CredentialID != "cred1" {
		t.Fatalf("unexpected identity params: %+v", captured)
	}
	if captured.ProductName != "PLN 20K" || captured.ProductBrand != "PLN" || captured.ProductCategory != "Listrik" || captured.ProductCode != "PLN20" {
		t.Fatalf("unexpected product snapshot params: %+v", captured)
	}
	if captured.CustomerNo != "08123456789" || captured.Price != 20000 || captured.Admin != 1500 || captured.Amount != 22500 {
		t.Fatalf("unexpected customer/amount params: %+v", captured)
	}
}

func TestDigiflazzOrderServiceUpdateStatusTransitions(t *testing.T) {
	tests := []struct {
		name             string
		current          digiflazzdomain.OrderStatus
		next             digiflazzdomain.OrderStatus
		wantErrSubstring string
	}{
		{name: "pending to processing allowed", current: digiflazzdomain.OrderStatusPending, next: digiflazzdomain.OrderStatusProcessing},
		{name: "pending to cancelled allowed", current: digiflazzdomain.OrderStatusPending, next: digiflazzdomain.OrderStatusCancelled},
		{name: "processing to success allowed", current: digiflazzdomain.OrderStatusProcessing, next: digiflazzdomain.OrderStatusSuccess},
		{name: "processing to failed allowed", current: digiflazzdomain.OrderStatusProcessing, next: digiflazzdomain.OrderStatusFailed},
		{name: "pending to success denied", current: digiflazzdomain.OrderStatusPending, next: digiflazzdomain.OrderStatusSuccess, wantErrSubstring: "invalid digiflazz order status transition"},
		{name: "terminal success cannot be overwritten", current: digiflazzdomain.OrderStatusSuccess, next: digiflazzdomain.OrderStatusFailed, wantErrSubstring: "invalid digiflazz order status transition"},
		{name: "terminal failed cannot be overwritten", current: digiflazzdomain.OrderStatusFailed, next: digiflazzdomain.OrderStatusProcessing, wantErrSubstring: "invalid digiflazz order status transition"},
		{name: "terminal cancelled cannot be overwritten", current: digiflazzdomain.OrderStatusCancelled, next: digiflazzdomain.OrderStatusProcessing, wantErrSubstring: "invalid digiflazz order status transition"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			updateCalled := false
			repo := &mockDigiflazzOrderRepo{
				getByIDFn: func(familyID, id string) (*digiflazzdomain.OrderDTO, error) {
					if familyID != "fam1" || id != "order1" {
						t.Fatalf("unexpected GetByID args family=%s id=%s", familyID, id)
					}
					return &digiflazzdomain.OrderDTO{ID: id, FamilyID: familyID, Status: tt.current}, nil
				},
				updateStatusFn: func(familyID, id string, params repository.UpdateDigiflazzOrderStatusParams) (*digiflazzdomain.OrderDTO, error) {
					updateCalled = true
					if params.Status != tt.next || params.Message != "ok" || params.RC != "00" || params.SN != "SN1" {
						t.Fatalf("unexpected update params: %+v", params)
					}
					return &digiflazzdomain.OrderDTO{ID: id, FamilyID: familyID, Status: params.Status}, nil
				},
			}
			service := NewDigiflazzOrderService(repo, DigiflazzOrderServiceDeps{})

			updated, err := service.UpdateStatus("fam1", "order1", tt.next, &digiflazzdomain.OrderResponseDTO{Message: "ok", RC: "00", SN: "SN1"})
			if tt.wantErrSubstring != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErrSubstring) {
					t.Fatalf("expected error containing %q, got %v", tt.wantErrSubstring, err)
				}
				if updateCalled {
					t.Fatal("forbidden transition should not call repository UpdateStatus")
				}
				return
			}

			if err != nil {
				t.Fatalf("UpdateStatus returned error: %v", err)
			}
			if !updateCalled || updated.Status != tt.next {
				t.Fatalf("expected update to status %s, got called=%v order=%+v", tt.next, updateCalled, updated)
			}
		})
	}
}

func TestDigiflazzOrderServiceUpdateStatusMissingOrder(t *testing.T) {
	repo := &mockDigiflazzOrderRepo{
		getByIDFn: func(familyID, id string) (*digiflazzdomain.OrderDTO, error) {
			return nil, nil
		},
	}
	service := NewDigiflazzOrderService(repo, DigiflazzOrderServiceDeps{})

	_, err := service.UpdateStatus("fam1", "missing", digiflazzdomain.OrderStatusProcessing, nil)
	if err == nil || !strings.Contains(err.Error(), "digiflazz order not found") {
		t.Fatalf("expected missing order error, got %v", err)
	}
}

func TestDigiflazzOrderServiceListFamilyOrdersNormalizesPagination(t *testing.T) {
	repoErr := errors.New("repo failure")
	repo := &mockDigiflazzOrderRepo{
		listByFamilyFn: func(familyID string, limit, offset int) ([]*digiflazzdomain.OrderDTO, error) {
			if familyID != "fam1" || limit != 20 || offset != 0 {
				t.Fatalf("unexpected list args family=%s limit=%d offset=%d", familyID, limit, offset)
			}
			return nil, repoErr
		},
	}
	service := NewDigiflazzOrderService(repo, DigiflazzOrderServiceDeps{})

	_, err := service.ListFamilyOrders("fam1", 0, 999)
	if !errors.Is(err, repoErr) {
		t.Fatalf("expected repo error, got %v", err)
	}
}
