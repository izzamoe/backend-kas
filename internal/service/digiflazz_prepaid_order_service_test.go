package service

import (
	"context"
	"errors"
	"strings"
	"testing"

	digiflazzclient "kas/internal/digiflazz"
	digiflazzdomain "kas/internal/domain/digiflazz"
	"kas/internal/repository"
	"kas/internal/utils"
	_ "kas/migrations"
)

type digiflazzPrepaidOrderFixture struct {
	familyID string
	ownerID  string
	memberID string
	fake     *digiflazzclient.FakeClient
	svc      DigiflazzOrderService
	orders   repository.DigiflazzOrderRepository
	events   repository.DigiflazzEventRepository
	calls    []digiflazzClientFactoryCall
}

func setupDigiflazzPrepaidOrderFixture(t *testing.T) *digiflazzPrepaidOrderFixture {
	t.Helper()
	t.Setenv(digiflazzCredentialEncryptionKeyEnv, "test-encryption-key")

	app, familyID, ownerID := setupDigiflazzPriceServiceTestApp(t)
	member := createServiceTestUser(t, app, "prepaid-member@example.com")
	createServiceTestRecord(t, app, "family_members", map[string]any{
		"family_id": familyID,
		"user_id":   member.Id,
		"role":      "member",
	})

	credentialRepo := repository.NewDigiflazzCredentialRepository(app)
	productRepo := repository.NewDigiflazzProductRepository(app)
	orderRepo := repository.NewDigiflazzOrderRepository(app)
	eventRepo := repository.NewDigiflazzEventRepository(app)

	apiKeyCiphertext, err := utils.Encrypt("secret-api-key-1234", []byte("test-encryption-key"))
	if err != nil {
		t.Fatalf("encrypt api key: %v", err)
	}
	if _, err := credentialRepo.Create(&repository.DigiflazzCredentialCreateData{
		FamilyID:         familyID,
		Username:         "buyer",
		APIKeyCiphertext: apiKeyCiphertext,
		APIKeyLast4:      "1234",
		APIKeyHash:       "hash",
		Testing:          true,
		IsActive:         true,
	}); err != nil {
		t.Fatalf("create credential: %v", err)
	}
	if _, err := productRepo.Upsert(&repository.UpsertProductInput{
		FamilyID:           familyID,
		CredentialID:       "",
		ProductName:        "PLN 20K",
		Category:           "PLN",
		Brand:              "PLN",
		Type:               "prepaid",
		BuyerSKUCode:       "PLN20",
		Price:              20000,
		Admin:              1500,
		BuyerProductStatus: "active",
		Provider:           "PLN",
		IsPrepaid:          true,
	}); err != nil {
		t.Fatalf("upsert product: %v", err)
	}

	fake := digiflazzclient.NewFakeClient()
	fx := &digiflazzPrepaidOrderFixture{
		familyID: familyID,
		ownerID:  ownerID,
		memberID: member.Id,
		fake:     fake,
		orders:   orderRepo,
		events:   eventRepo,
	}
	productSvc := NewDigiflazzProductService(app, productRepo, credentialRepo, nil)
	fx.svc = NewDigiflazzOrderService(orderRepo, DigiflazzOrderServiceDeps{
		App:            app,
		CredentialRepo: credentialRepo,
		ProductService: productSvc,
		EventRepo:      eventRepo,
		ClientFactory: func(username, apiKey string, testing bool) digiflazzclient.DigiflazzClient {
			fx.calls = append(fx.calls, digiflazzClientFactoryCall{Username: username, APIKey: apiKey, Testing: testing})
			return fake
		},
	})

	return fx
}

func TestDigiflazzPrepaidOrderServiceCreateSuccessForFamilyMember(t *testing.T) {
	fx := setupDigiflazzPrepaidOrderFixture(t)
	fx.fake.SetResponse("CekSaldo", digiflazzclient.CekSaldoResponse{Deposit: 100000}, nil)
	fx.fake.SetResponse("Topup", digiflazzclient.TransactionResponse{
		RefID:          "ignored-ref-from-provider",
		CustomerNo:     "08123456789",
		BuyerSKUCode:   "PLN20",
		Message:        "Sukses",
		Status:         digiflazzclient.StatusSukses,
		Rc:             "00",
		Sn:             "SN123",
		BuyerLastSaldo: 78500,
		Price:          20000,
		Admin:          1500,
		SellingPrice:   21500,
	}, nil)

	order, err := fx.svc.CreatePrepaidOrder(context.Background(), &digiflazzdomain.CreateOrderRequest{
		BuyerSKUCode: "PLN20",
		CustomerNo:   " 08123456789 ",
	}, fx.memberID, fx.familyID)
	if err != nil {
		t.Fatalf("CreatePrepaidOrder returned error: %v", err)
	}
	if order.ID == "" || order.Status != digiflazzdomain.OrderStatusSuccess || order.RC != "00" || order.SN != "SN123" {
		t.Fatalf("unexpected successful order: %+v", order)
	}
	if order.CredentialID == "" || order.EventType != digiflazzdomain.EventTypeTopup || order.SellingPrice != 21500 || order.BuyerLastSaldo != 78500 {
		t.Fatalf("order snapshot/response not persisted: %+v", order)
	}
	if fx.fake.CallCount("Topup") != 1 {
		t.Fatalf("expected topup once, got %d", fx.fake.CallCount("Topup"))
	}
	if len(fx.calls) != 1 || fx.calls[0].Username != "buyer" || fx.calls[0].APIKey != "secret-api-key-1234" || !fx.calls[0].Testing {
		t.Fatalf("unexpected client factory calls: %+v", fx.calls)
	}
	history := fx.fake.History()
	topupReq, ok := history[1].Request.(*digiflazzclient.TopupRequest)
	if !ok {
		t.Fatalf("expected TopupRequest at history[1], got %+v", history)
	}
	if topupReq.BuyerSKUCode != "PLN20" || topupReq.CustomerNo != "08123456789" || topupReq.RefID != order.RefID || topupReq.MaxPrice != 21500 {
		t.Fatalf("unexpected topup request: %+v", topupReq)
	}

	events, err := fx.events.ListByFamilyID(fx.familyID, 10, 0)
	if err != nil {
		t.Fatalf("ListByFamilyID events returned error: %v", err)
	}
	if len(events) != 1 || events[0].OrderID != order.ID || events[0].StatusAfter != "success" {
		t.Fatalf("unexpected audit event: %+v", events)
	}
	if strings.Contains(events[0].RedactedPayload, "secret-api-key") {
		t.Fatalf("event payload leaked api key: %s", events[0].RedactedPayload)
	}
}

func TestDigiflazzPrepaidOrderServiceTopupProceedsWithSufficientBalance(t *testing.T) {
	fx := setupDigiflazzPrepaidOrderFixture(t)
	fx.fake.SetResponse("CekSaldo", digiflazzclient.CekSaldoResponse{Deposit: 100000}, nil)
	fx.fake.SetResponse("Topup", digiflazzclient.TransactionResponse{
		Rc: "03", Status: digiflazzclient.StatusPending,
	}, nil)

	order, err := fx.svc.CreatePrepaidOrder(context.Background(), &digiflazzdomain.CreateOrderRequest{
		BuyerSKUCode: "PLN20",
		CustomerNo:   "08123456789",
	}, fx.ownerID, fx.familyID)
	if err != nil {
		t.Fatalf("CreatePrepaidOrder returned error: %v", err)
	}
	if order == nil {
		t.Fatal("expected order to be created")
	}
	if fx.fake.CallCount("Topup") != 1 {
		t.Fatalf("expected Topup called once, got %d", fx.fake.CallCount("Topup"))
	}
	if fx.fake.CallCount("CekSaldo") != 1 {
		t.Fatalf("expected CekSaldo called once, got %d", fx.fake.CallCount("CekSaldo"))
	}
}

func TestDigiflazzPrepaidOrderServiceTimeoutReturnsProcessing(t *testing.T) {
	fx := setupDigiflazzPrepaidOrderFixture(t)
	fx.fake.SetResponse("CekSaldo", digiflazzclient.CekSaldoResponse{Deposit: 100000}, nil)
	fx.fake.SetResponse("Topup", nil, context.DeadlineExceeded)

	order, err := fx.svc.CreatePrepaidOrder(context.Background(), &digiflazzdomain.CreateOrderRequest{
		BuyerSKUCode: "PLN20",
		CustomerNo:   "08123456789",
	}, fx.ownerID, fx.familyID)
	if err != nil {
		t.Fatalf("CreatePrepaidOrder timeout returned error: %v", err)
	}
	if order.Status != digiflazzdomain.OrderStatusProcessing || !strings.Contains(order.Message, "processing") {
		t.Fatalf("expected processing order on timeout, got %+v", order)
	}
}

func TestDigiflazzPrepaidOrderServiceRejectsUnavailableProductBeforeBalance(t *testing.T) {
	fx := setupDigiflazzPrepaidOrderFixture(t)
	productSvc := &fakeDigiflazzPrepaidProductService{product: &digiflazzdomain.ProductDTO{
		Code:      "INACTIVE",
		Name:      "Inactive",
		Price:     1000,
		Status:    "inactive",
		IsPrepaid: true,
	}}
	validCiphertext, _ := utils.Encrypt("test-api-key", []byte("test-encryption-key"))
	fx.svc = NewDigiflazzOrderService(fx.orders, DigiflazzOrderServiceDeps{
		App:            nil,
		CredentialRepo: &fakeDigiflazzPrepaidCredentialRepo{credential: &repository.DigiflazzCredentialRecord{ID: "cred", FamilyID: fx.familyID, Username: "buyer", APIKeyCiphertext: validCiphertext, IsActive: true}},
		ProductService: productSvc,
		EventRepo:      fx.events,
		ClientFactory:  func(username, apiKey string, testing bool) digiflazzclient.DigiflazzClient { return fx.fake },
	})

	_, err := fx.svc.CreatePrepaidOrder(context.Background(), &digiflazzdomain.CreateOrderRequest{BuyerSKUCode: "INACTIVE", CustomerNo: "0812"}, fx.ownerID, fx.familyID)
	if err == nil || !errors.Is(err, digiflazzdomain.ErrDigiflazzProductUnavailable) {
		t.Fatalf("expected unavailable product error, got %v", err)
	}
	if fx.fake.CallCount("CekSaldo") != 0 || fx.fake.CallCount("Topup") != 0 {
		t.Fatalf("expected no Digiflazz calls, got balance=%d topup=%d", fx.fake.CallCount("CekSaldo"), fx.fake.CallCount("Topup"))
	}
}

type fakeDigiflazzPrepaidProductService struct {
	product *digiflazzdomain.ProductDTO
}

func (f *fakeDigiflazzPrepaidProductService) SyncPricelistWithCredential(_ context.Context, _ *repository.DigiflazzCredentialRecord) (*SyncResult, error) {
	return nil, nil
}

func (f *fakeDigiflazzPrepaidProductService) SearchProducts(_ string, _ *digiflazzdomain.ProductSearchRequest) ([]*digiflazzdomain.ProductDTO, error) {
	return nil, nil
}

func (f *fakeDigiflazzPrepaidProductService) GetProductBySKU(_, sku string) (*digiflazzdomain.ProductDTO, error) {
	return f.product, nil
}

type fakeDigiflazzPrepaidCredentialRepo struct {
	repository.DigiflazzCredentialRepository
	credential *repository.DigiflazzCredentialRecord
}

func (f *fakeDigiflazzPrepaidCredentialRepo) GetActiveSecretByFamilyID(string) (*repository.DigiflazzCredentialRecord, error) {
	return f.credential, nil
}
