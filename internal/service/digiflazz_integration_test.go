package service

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	digiflazzclient "kas/internal/digiflazz"
	digiflazzdomain "kas/internal/domain/digiflazz"
	"kas/internal/repository"
	_ "kas/migrations"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
)

// ---------------------------------------------------------------------------
// Shared E2E fixture helpers
// ---------------------------------------------------------------------------

type digiflazzE2EFixture struct {
	app                *tests.TestApp
	familyID           string
	ownerID            string
	ownerToken         string
	memberID           string
	memberToken        string
	credentialRepo     repository.DigiflazzCredentialRepository
	productRepo        repository.DigiflazzProductRepository
	orderRepo          repository.DigiflazzOrderRepository
	eventRepo          repository.DigiflazzEventRepository
	transactionRepo    repository.TransactionRepository
	categoryRepo       repository.CategoryRepository
	fake               *digiflazzclient.FakeClient
	clientFactoryCalls []digiflazzClientFactoryCall
}

func setupDigiflazzE2EFixture(t *testing.T) *digiflazzE2EFixture {
	t.Helper()
	t.Setenv(digiflazzCredentialEncryptionKeyEnv, "test-encryption-key-32bytes!")

	app, err := tests.NewTestApp()
	if err != nil {
		t.Fatalf("failed to create test app: %v", err)
	}
	t.Cleanup(app.Cleanup)

	// Create family
	familyCol, _ := app.FindCollectionByNameOrId("families")
	family := core.NewRecord(familyCol)
	family.Set("name", "E2E Test Family")
	family.Set("invite_code", fmt.Sprintf("E2E%d", time.Now().UnixNano()))
	app.Save(family)

	// Create owner user
	userCol, _ := app.FindCollectionByNameOrId("users")
	owner := core.NewRecord(userCol)
	owner.Set("email", fmt.Sprintf("e2e-owner+%d@example.com", time.Now().UnixNano()))
	owner.Set("name", "E2E Owner")
	owner.Set("verified", true)
	owner.SetPassword("password123456")
	app.Save(owner)
	ownerToken, _ := owner.NewAuthToken()

	// Create member user
	member := core.NewRecord(userCol)
	member.Set("email", fmt.Sprintf("e2e-member+%d@example.com", time.Now().UnixNano()))
	member.Set("name", "E2E Member")
	member.Set("verified", true)
	member.SetPassword("password123456")
	app.Save(member)
	memberToken, _ := member.NewAuthToken()

	// Create family_members
	memberCol, _ := app.FindCollectionByNameOrId("family_members")
	ownerMember := core.NewRecord(memberCol)
	ownerMember.Set("family_id", family.Id)
	ownerMember.Set("user_id", owner.Id)
	ownerMember.Set("role", "owner")
	app.Save(ownerMember)

	memberRec := core.NewRecord(memberCol)
	memberRec.Set("family_id", family.Id)
	memberRec.Set("user_id", member.Id)
	memberRec.Set("role", "member")
	app.Save(memberRec)

	// Create default expense categories
	catCol, _ := app.FindCollectionByNameOrId("categories")
	for _, name := range []string{"Hiburan", "Rumah & utilitas", "Lainnya"} {
		cat := core.NewRecord(catCol)
		cat.Set("family_id", family.Id)
		cat.Set("name", name)
		cat.Set("icon", "📦")
		cat.Set("color", "#888780")
		cat.Set("is_default", true)
		cat.Set("is_master", false)
		cat.Set("type", "expense")
		app.Save(cat)
	}

	// Repos
	credentialRepo := repository.NewDigiflazzCredentialRepository(app)
	productRepo := repository.NewDigiflazzProductRepository(app)
	orderRepo := repository.NewDigiflazzOrderRepository(app)
	eventRepo := repository.NewDigiflazzEventRepository(app)
	transactionRepo := repository.NewTransactionRepository(app)
	categoryRepo := repository.NewCategoryRepository(app)

	fake := digiflazzclient.NewFakeClient()
	fx := &digiflazzE2EFixture{
		app:             app,
		familyID:        family.Id,
		ownerID:         owner.Id,
		ownerToken:      ownerToken,
		memberID:        member.Id,
		memberToken:     memberToken,
		credentialRepo:  credentialRepo,
		productRepo:     productRepo,
		orderRepo:       orderRepo,
		eventRepo:       eventRepo,
		transactionRepo: transactionRepo,
		categoryRepo:    categoryRepo,
		fake:            fake,
	}

	return fx
}

func (fx *digiflazzE2EFixture) createCredential(t *testing.T) {
	t.Helper()
	fx.fake.SetResponse("CekSaldo", digiflazzclient.CekSaldoResponse{Deposit: 500000}, nil)
	svc := NewDigiflazzCredentialService(fx.credentialRepo, fx.app, func(username, apiKey string, testing bool) digiflazzclient.DigiflazzClient {
		return fx.fake
	}, fx.productRepo, fx.productService())
	testingTrue := true
	_, err := svc.UpsertCredential(context.Background(), fx.familyID, fx.ownerID, digiflazzdomain.UpsertCredentialRequest{
		Username: "e2e-buyer",
		APIKey:   "test-api-key-1234",
		Testing:  &testingTrue,
	})
	if err != nil {
		t.Fatalf("create credential: %v", err)
	}
}

func (fx *digiflazzE2EFixture) syncProducts(t *testing.T) {
	t.Helper()
	fx.fake.SetResponse("DaftarHargaPrabayar", []digiflazzclient.PriceListPrepaidItem{
		{
			ProductName: "PLN 20K", Category: "Listrik", Brand: "PLN", Type: "Reguler",
			BuyerSKUCode: "PLN20", Price: 20000, BuyerProductStatus: "active",
		},
		{
			ProductName: "Telkomsel 10K", Category: "Pulsa", Brand: "Telkomsel", Type: "Reguler",
			BuyerSKUCode: "TSEL10", Price: 10000, BuyerProductStatus: "active",
		},
	}, nil)
	fx.fake.SetResponse("DaftarHargaPascabayar", []digiflazzclient.PriceListPascaItem{
		{
			ProductName: "PLN Pascabayar", Category: "Listrik", Brand: "PLN",
			BuyerSKUCode: "PLNPOST", Admin: 2500, BuyerProductStatus: "active",
		},
	}, nil)

	productSvc := NewDigiflazzProductService(fx.app, fx.productRepo, fx.credentialRepo, func(username, apiKey string, testing bool) digiflazzclient.DigiflazzClient {
		return fx.fake
	})
	credRecord, err := fx.credentialRepo.GetSecretByFamilyID(fx.familyID)
	if err != nil {
		t.Fatalf("get credential for sync: %v", err)
	}
	result, err := productSvc.SyncPricelistWithCredential(context.Background(), credRecord)
	if err != nil {
		t.Fatalf("sync products: %v", err)
	}
	if result.TotalUpserted < 1 {
		t.Fatalf("expected at least 1 product upserted, got %d", result.TotalUpserted)
	}
}

func (fx *digiflazzE2EFixture) orderService() DigiflazzOrderService {
	return NewDigiflazzOrderService(fx.orderRepo, DigiflazzOrderServiceDeps{
		App:             fx.app,
		CredentialRepo:  fx.credentialRepo,
		ProductService:  NewDigiflazzProductService(fx.app, fx.productRepo, fx.credentialRepo, func(username, apiKey string, testing bool) digiflazzclient.DigiflazzClient { return fx.fake }),
		EventRepo:       fx.eventRepo,
		TransactionRepo: fx.transactionRepo,
		CategoryRepo:    fx.categoryRepo,
		ClientFactory: func(username, apiKey string, testing bool) digiflazzclient.DigiflazzClient {
			fx.clientFactoryCalls = append(fx.clientFactoryCalls, digiflazzClientFactoryCall{Username: username, APIKey: apiKey, Testing: testing})
			return fx.fake
		},
	})
}

func (fx *digiflazzE2EFixture) productService() DigiflazzProductService {
	return NewDigiflazzProductService(fx.app, fx.productRepo, fx.credentialRepo, func(username, apiKey string, testing bool) digiflazzclient.DigiflazzClient {
		return fx.fake
	})
}

// ---------------------------------------------------------------------------
// 1. Full Member Prepaid Order E2E
// ---------------------------------------------------------------------------

func TestDigiflazzIntegration_FullPrepaidOrderE2E(t *testing.T) {
	fx := setupDigiflazzE2EFixture(t)

	// Step 1: Owner creates credential
	fx.createCredential(t)

	// Step 2: Owner syncs product catalog
	fx.syncProducts(t)

	// Step 3: Member searches products
	productSvc := fx.productService()
	products, err := productSvc.SearchProducts(fx.familyID, &digiflazzdomain.ProductSearchRequest{Query: "PLN"})
	if err != nil {
		t.Fatalf("search products: %v", err)
	}
	if len(products) == 0 {
		t.Fatal("expected at least one product from search")
	}

	// Step 4: Member creates prepaid order
	fx.fake.Reset()
	fx.fake.SetResponse("CekSaldo", digiflazzclient.CekSaldoResponse{Deposit: 500000}, nil)
	fx.fake.SetResponse("Topup", digiflazzclient.TransactionResponse{
		RefID:        "DFZ-E2E-1",
		CustomerNo:   "08123456789",
		BuyerSKUCode: "PLN20",
		Message:      "Sukses",
		Status:       digiflazzclient.StatusSukses,
		Rc:           "00",
		Sn:           "SN-E2E-12345",
		Price:        20000,
		Admin:        1500,
		SellingPrice: 21500,
	}, nil)

	orderSvc := fx.orderService()
	_, err = orderSvc.CreatePrepaidOrder(context.Background(), &digiflazzdomain.CreateOrderRequest{
		BuyerSKUCode: "PLN20",
		CustomerNo:   "08123456789",
	}, fx.ownerID, fx.familyID)
	if err != nil {
		t.Fatalf("CreatePrepaidOrder: %v", err)
	}

	events, err := fx.eventRepo.ListByFamilyID(fx.familyID, 10, 0)
	if err != nil {
		t.Fatalf("ListByFamilyID: %v", err)
	}
	if len(events) == 0 {
		t.Fatal("expected at least one audit event")
	}
	for _, evt := range events {
		if strings.Contains(evt.RedactedPayload, "test-api-key") {
			t.Fatalf("event payload MUST NOT contain api key, got: %s", evt.RedactedPayload)
		}
		if strings.Contains(evt.RedactedPayload, "ciphertext") {
			t.Fatalf("event payload MUST NOT contain ciphertext, got: %s", evt.RedactedPayload)
		}
	}
}
