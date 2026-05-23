package service

import (
	"context"
	"encoding/json"
	"errors"
	digiflazzclient "kas/internal/digiflazz"
	digiflazzdomain "kas/internal/domain/digiflazz"
	"kas/internal/repository"
	"kas/internal/utils"
	_ "kas/migrations"
	"strings"
	"testing"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
)

type digiflazzCredentialServiceFixture struct {
	app      *tests.TestApp
	familyID string
	ownerID  string
	memberID string
	repo     repository.DigiflazzCredentialRepository
	fake     *digiflazzclient.FakeClient
	svc      DigiflazzCredentialService
	calls    []digiflazzClientFactoryCall
}

type digiflazzClientFactoryCall struct {
	Username string
	APIKey   string
	Testing  bool
}

type fakeProductRepoForCredSvc struct{}

func (f *fakeProductRepoForCredSvc) Upsert(input *repository.UpsertProductInput) (*digiflazzdomain.ProductDTO, error) {
	return nil, nil
}
func (f *fakeProductRepoForCredSvc) Search(familyID string, req *digiflazzdomain.ProductSearchRequest) ([]*digiflazzdomain.ProductDTO, error) {
	return nil, nil
}
func (f *fakeProductRepoForCredSvc) GetBySKU(familyID, sku string) (*digiflazzdomain.ProductDTO, error) {
	return nil, nil
}
func (f *fakeProductRepoForCredSvc) DeleteByFamilyID(familyID string) error { return nil }

type fakeProductSvcForCredSvc struct{}

func (f *fakeProductSvcForCredSvc) SyncPricelistWithCredential(ctx context.Context, credential *repository.DigiflazzCredentialRecord) (*SyncResult, error) {
	return &SyncResult{}, nil
}
func (f *fakeProductSvcForCredSvc) SearchProducts(familyID string, req *digiflazzdomain.ProductSearchRequest) ([]*digiflazzdomain.ProductDTO, error) {
	return nil, nil
}
func (f *fakeProductSvcForCredSvc) GetProductBySKU(familyID, sku string) (*digiflazzdomain.ProductDTO, error) {
	return nil, nil
}

func setupDigiflazzCredentialServiceFixture(t *testing.T) *digiflazzCredentialServiceFixture {
	t.Helper()
	t.Setenv(digiflazzCredentialEncryptionKeyEnv, "test-encryption-key")

	app, err := tests.NewTestApp()
	if err != nil {
		t.Fatalf("failed to create test app: %v", err)
	}
	t.Cleanup(app.Cleanup)

	family := createServiceTestRecord(t, app, "families", map[string]any{
		"name":        "Keluarga Digiflazz",
		"invite_code": "DIGI1234",
	})
	owner := createServiceTestUser(t, app, "owner@example.com")
	member := createServiceTestUser(t, app, "member@example.com")
	createServiceTestRecord(t, app, "family_members", map[string]any{
		"family_id": family.Id,
		"user_id":   owner.Id,
		"role":      "owner",
	})
	createServiceTestRecord(t, app, "family_members", map[string]any{
		"family_id": family.Id,
		"user_id":   member.Id,
		"role":      "member",
	})

	repo := repository.NewDigiflazzCredentialRepository(app)
	fake := digiflazzclient.NewFakeClient()
	fixture := &digiflazzCredentialServiceFixture{
		app:      app,
		familyID: family.Id,
		ownerID:  owner.Id,
		memberID: member.Id,
		repo:     repo,
		fake:     fake,
	}
	fakeProductRepo := &fakeProductRepoForCredSvc{}
	fakeProductSvc := &fakeProductSvcForCredSvc{}
	fixture.svc = NewDigiflazzCredentialService(repo, app, func(username, apiKey string, testing bool) digiflazzclient.DigiflazzClient {
		fixture.calls = append(fixture.calls, digiflazzClientFactoryCall{Username: username, APIKey: apiKey, Testing: testing})
		return fake
	}, fakeProductRepo, fakeProductSvc)
	return fixture
}

func createServiceTestRecord(t *testing.T, app core.App, collectionName string, values map[string]any) *core.Record {
	t.Helper()
	collection, err := app.FindCollectionByNameOrId(collectionName)
	if err != nil {
		t.Fatalf("failed to find collection %s: %v", collectionName, err)
	}
	record := core.NewRecord(collection)
	for key, value := range values {
		record.Set(key, value)
	}
	if err := app.Save(record); err != nil {
		t.Fatalf("failed to save %s record: %v", collectionName, err)
	}
	return record
}

func createServiceTestUser(t *testing.T, app core.App, email string) *core.Record {
	t.Helper()
	collection, err := app.FindCollectionByNameOrId("users")
	if err != nil {
		t.Fatalf("failed to find users collection: %v", err)
	}
	user := core.NewRecord(collection)
	user.Set("email", email)
	user.Set("verified", true)
	user.Set("name", email)
	user.SetPassword("password123456")
	if err := app.Save(user); err != nil {
		t.Fatalf("failed to save user record: %v", err)
	}
	return user
}

func TestDigiflazzCredentialServiceCreateCredential(t *testing.T) {
	t.Run("owner creates encrypted credential after validation", func(t *testing.T) {
		fx := setupDigiflazzCredentialServiceFixture(t)
		fx.fake.SetResponse("CekSaldo", digiflazzclient.CekSaldoResponse{Deposit: 150000}, nil)

		testingTrue := true
		got, err := fx.svc.UpsertCredential(context.Background(), fx.familyID, fx.ownerID, digiflazzdomain.UpsertCredentialRequest{
			Username: " buyer ",
			APIKey:   "secret-api-key-1234",
			Testing:  &testingTrue,
		})
		if err != nil {
			t.Fatalf("UpsertCredential returned error: %v", err)
		}
		if got.Credential.ID == "" || got.Credential.Username != "buyer" || got.Credential.APIKeyLast4 != "1234" || !got.Credential.IsActive || !got.Credential.Testing {
			t.Fatalf("unexpected credential: %+v", got.Credential)
		}
		if fx.fake.CallCount("CekSaldo") != 1 {
			t.Fatalf("expected CekSaldo call count 1, got %d", fx.fake.CallCount("CekSaldo"))
		}
		if len(fx.calls) != 1 || fx.calls[0].Username != "buyer" || fx.calls[0].APIKey != "secret-api-key-1234" || !fx.calls[0].Testing {
			t.Fatalf("unexpected client factory calls: %+v", fx.calls)
		}

		secret, err := fx.repo.GetSecretByFamilyID(fx.familyID)
		if err != nil {
			t.Fatalf("GetSecretByFamilyID returned error: %v", err)
		}
		if secret.APIKeyCiphertext == "secret-api-key-1234" || secret.APIKeyHash == "" || secret.APIKeyLast4 != "1234" {
			t.Fatalf("credential was not stored safely: %+v", secret)
		}
		plaintext, err := utils.Decrypt(secret.APIKeyCiphertext, []byte("test-encryption-key"))
		if err != nil {
			t.Fatalf("failed to decrypt stored api key: %v", err)
		}
		if plaintext != "secret-api-key-1234" {
			t.Fatalf("unexpected decrypted api key: %q", plaintext)
		}
	})

	t.Run("validation failure does not persist", func(t *testing.T) {
		fx := setupDigiflazzCredentialServiceFixture(t)
		fx.fake.SetResponse("CekSaldo", nil, errors.New("invalid auth"))

		_, err := fx.svc.UpsertCredential(context.Background(), fx.familyID, fx.ownerID, digiflazzdomain.UpsertCredentialRequest{
			Username: "buyer",
			APIKey:   "bad-key",
		})
		if err == nil {
			t.Fatal("expected validation error")
		}
		count, err := fx.repo.CountByFamilyID(fx.familyID)
		if err != nil {
			t.Fatalf("CountByFamilyID returned error: %v", err)
		}
		if count != 0 {
			t.Fatalf("expected no persisted credential, got count %d", count)
		}
	})

	t.Run("non owner cannot create", func(t *testing.T) {
		fx := setupDigiflazzCredentialServiceFixture(t)
		fx.fake.SetResponse("CekSaldo", digiflazzclient.CekSaldoResponse{Deposit: 1}, nil)

		_, err := fx.svc.UpsertCredential(context.Background(), fx.familyID, fx.memberID, digiflazzdomain.UpsertCredentialRequest{
			Username: "buyer",
			APIKey:   "secret",
		})
		if err == nil {
			t.Fatal("expected unauthorized error")
		}
		if fx.fake.CallCount("CekSaldo") != 0 {
			t.Fatalf("expected no validation call for unauthorized user, got %d", fx.fake.CallCount("CekSaldo"))
		}
	})

	t.Run("upsert updates existing credential", func(t *testing.T) {
		fx := setupDigiflazzCredentialServiceFixture(t)
		fx.fake.SetResponse("CekSaldo", digiflazzclient.CekSaldoResponse{Deposit: 1}, nil)

		_, err := fx.svc.UpsertCredential(context.Background(), fx.familyID, fx.ownerID, digiflazzdomain.UpsertCredentialRequest{Username: "buyer", APIKey: "secret-one"})
		if err != nil {
			t.Fatalf("first UpsertCredential returned error: %v", err)
		}
		result2, err := fx.svc.UpsertCredential(context.Background(), fx.familyID, fx.ownerID, digiflazzdomain.UpsertCredentialRequest{Username: "buyer-updated", APIKey: "secret-two"})
		if err != nil {
			t.Fatalf("second UpsertCredential (update) returned error: %v", err)
		}
		if result2.Credential.Username != "buyer-updated" {
			t.Fatalf("expected updated username after second upsert, got %+v", result2.Credential)
		}
		count, err := fx.repo.CountByFamilyID(fx.familyID)
		if err != nil {
			t.Fatalf("CountByFamilyID returned error: %v", err)
		}
		if count != 1 {
			t.Fatalf("expected exactly 1 credential after two upserts, got %d", count)
		}
	})
}

func TestDigiflazzCredentialServiceTestWebhook(t *testing.T) {
	t.Run("errors when webhook id is missing", func(t *testing.T) {
		fx := setupDigiflazzCredentialServiceFixture(t)
		fx.fake.SetResponse("CekSaldo", digiflazzclient.CekSaldoResponse{Deposit: 1}, nil)
		_, err := fx.svc.UpsertCredential(context.Background(), fx.familyID, fx.ownerID, digiflazzdomain.UpsertCredentialRequest{Username: "buyer", APIKey: "secret-api-key-1234"})
		if err != nil {
			t.Fatalf("UpsertCredential returned error: %v", err)
		}

		_, err = fx.svc.TestWebhook(context.Background(), fx.familyID, fx.ownerID)
		if err == nil || !strings.Contains(err.Error(), "webhook id") || !strings.Contains(err.Error(), "required") {
			t.Fatalf("expected clear missing webhook id error, got %v", err)
		}
		if fx.fake.CallCount("TestWebhookPing") != 0 {
			t.Fatalf("expected no ping call when webhook id is missing, got %d", fx.fake.CallCount("TestWebhookPing"))
		}
	})

	t.Run("owner pings configured webhook", func(t *testing.T) {
		fx := setupDigiflazzCredentialServiceFixture(t)
		fx.fake.SetResponse("CekSaldo", digiflazzclient.CekSaldoResponse{Deposit: 1}, nil)
		fx.fake.SetResponse("TestWebhookPing", digiflazzclient.WebhookPingResponse{
			Sed:    "ping-sed",
			HookID: "hook-123",
			Hook:   digiflazzclient.WebhookPingHook{URL: "https://example.test/webhook", Secret: "must-not-leak", Type: "application/json", Status: 1},
		}, nil)
		webhookID := " hook-123 "
		_, err := fx.svc.UpsertCredential(context.Background(), fx.familyID, fx.ownerID, digiflazzdomain.UpsertCredentialRequest{Username: "buyer", APIKey: "secret-api-key-1234", WebhookID: &webhookID})
		if err != nil {
			t.Fatalf("UpsertCredential returned error: %v", err)
		}

		resp, err := fx.svc.TestWebhook(context.Background(), fx.familyID, fx.ownerID)
		if err != nil {
			t.Fatalf("TestWebhook returned error: %v", err)
		}
		if resp == nil || resp.HookID != "hook-123" {
			t.Fatalf("unexpected ping response: %+v", resp)
		}
		if resp.Sed != "ping-sed" || resp.Hook.URL != "https://example.test/webhook" || resp.Hook.Type != "application/json" || resp.Hook.Status != 1 {
			t.Fatalf("unexpected safe ping response mapping: %+v", resp)
		}
		encoded, err := json.Marshal(resp)
		if err != nil {
			t.Fatalf("marshal safe ping response: %v", err)
		}
		if body := string(encoded); strings.Contains(body, "secret") || strings.Contains(body, "must-not-leak") {
			t.Fatalf("safe ping response leaked upstream secret: %s", body)
		}
		if fx.fake.CallCount("TestWebhookPing") != 1 {
			t.Fatalf("expected one ping call, got %d", fx.fake.CallCount("TestWebhookPing"))
		}
		call := fx.fake.History()[1]
		if call.Method != "TestWebhookPing" || call.Request != "hook-123" {
			t.Fatalf("unexpected ping call: %+v", call)
		}
	})
}

func TestDigiflazzCredentialServiceCreateCredentialMissingEncryptionKey(t *testing.T) {
	fx := setupDigiflazzCredentialServiceFixture(t)
	t.Setenv(digiflazzCredentialEncryptionKeyEnv, "")
	fx.fake.SetResponse("CekSaldo", digiflazzclient.CekSaldoResponse{Deposit: 1}, nil)

	_, err := fx.svc.UpsertCredential(context.Background(), fx.familyID, fx.ownerID, digiflazzdomain.UpsertCredentialRequest{
		Username: "buyer",
		APIKey:   "secret-api-key-1234",
	})
	if err == nil {
		t.Fatal("expected missing encryption key error")
	}
	msg := err.Error()
	if !strings.Contains(msg, digiflazzCredentialEncryptionKeyEnv) || !strings.Contains(msg, "required") {
		t.Fatalf("unexpected error message: %q", msg)
	}
	if strings.Contains(msg, "secret-api-key-1234") || strings.Contains(msg, "DIGIFLAZZ") && strings.Contains(msg, "key") && strings.Contains(msg, "value") {
		t.Fatalf("error exposed secret details: %q", msg)
	}
	if fx.fake.CallCount("CekSaldo") != 1 {
		t.Fatalf("expected validation to reach CekSaldo once, got %d", fx.fake.CallCount("CekSaldo"))
	}
}

func TestDigiflazzCredentialServiceUpdateAndBalance(t *testing.T) {
	fx := setupDigiflazzCredentialServiceFixture(t)
	fx.fake.SetResponse("CekSaldo", digiflazzclient.CekSaldoResponse{Deposit: 100000}, nil)
	_, err := fx.svc.UpsertCredential(context.Background(), fx.familyID, fx.ownerID, digiflazzdomain.UpsertCredentialRequest{Username: "buyer", APIKey: "secret-api-key-1234"})
	if err != nil {
		t.Fatalf("UpsertCredential returned error: %v", err)
	}

	testingMode := true
	updated, err := fx.svc.UpsertCredential(context.Background(), fx.familyID, fx.ownerID, digiflazzdomain.UpsertCredentialRequest{
		Username: "buyer-updated",
		APIKey:   "updated-api-key-5678",
		Testing:  &testingMode,
	})
	if err != nil {
		t.Fatalf("UpsertCredential (update) returned error: %v", err)
	}
	if updated.Credential.Username != "buyer-updated" || updated.Credential.APIKeyLast4 != "5678" || !updated.Credential.Testing {
		t.Fatalf("unexpected updated credential: %+v", updated.Credential)
	}
	if fx.fake.CallCount("CekSaldo") != 2 {
		t.Fatalf("expected create+update validation calls, got %d", fx.fake.CallCount("CekSaldo"))
	}
	if len(fx.calls) != 2 || fx.calls[1].Username != "buyer-updated" || fx.calls[1].APIKey != "updated-api-key-5678" || !fx.calls[1].Testing {
		t.Fatalf("unexpected update factory calls: %+v", fx.calls)
	}

	fx.fake.SetResponse("CekSaldo", digiflazzclient.CekSaldoResponse{Deposit: 250000}, nil)
	balance, err := fx.svc.CheckBalance(context.Background(), fx.familyID, fx.ownerID)
	if err != nil {
		t.Fatalf("CheckBalance returned error: %v", err)
	}
	if balance.FamilyID != fx.familyID || balance.Balance != 250000 {
		t.Fatalf("unexpected balance: %+v", balance)
	}
	if len(fx.calls) != 3 || fx.calls[2].APIKey != "updated-api-key-5678" {
		t.Fatalf("expected CheckBalance to decrypt latest key, calls=%+v", fx.calls)
	}
}

func TestDigiflazzDepositService(t *testing.T) {
	t.Run("owner deposits via digiflazz client", func(t *testing.T) {
		fx := setupDigiflazzCredentialServiceFixture(t)
		fx.fake.SetResponse("CekSaldo", digiflazzclient.CekSaldoResponse{Deposit: 100000}, nil)
		fx.fake.SetResponse("Deposit", digiflazzclient.DepositResponse{Rc: "00", Bank: "BCA", PaymentMethod: "transfer", AccountNo: "1234567890", Amount: 500000}, nil)
		_, err := fx.svc.UpsertCredential(context.Background(), fx.familyID, fx.ownerID, digiflazzdomain.UpsertCredentialRequest{Username: "buyer", APIKey: "secret-api-key-1234"})
		if err != nil {
			t.Fatalf("UpsertCredential returned error: %v", err)
		}

		resp, err := fx.svc.Deposit(context.Background(), fx.familyID, fx.ownerID, 500000, "BCA")
		if err != nil {
			t.Fatalf("Deposit returned error: %v", err)
		}
		if resp == nil || resp.Bank != "BCA" || resp.Amount != 500000 || resp.Rc != "00" {
			t.Fatalf("unexpected deposit response: %+v", resp)
		}
		if fx.fake.CallCount("Deposit") != 1 {
			t.Fatalf("expected Deposit call count 1, got %d", fx.fake.CallCount("Deposit"))
		}
		if len(fx.calls) != 2 || fx.calls[1].Username != "buyer" || fx.calls[1].APIKey != "secret-api-key-1234" {
			t.Fatalf("unexpected client factory calls: %+v", fx.calls)
		}
		call := fx.fake.History()[1]
		req, ok := call.Request.(*digiflazzclient.DepositRequest)
		if !ok {
			t.Fatalf("unexpected deposit request type: %T", call.Request)
		}
		if req.Amount != 500000 || req.Bank != "BCA" || req.OwnerName == "" {
			t.Fatalf("unexpected deposit request: %+v", req)
		}
	})

	t.Run("member can deposit", func(t *testing.T) {
		fx := setupDigiflazzCredentialServiceFixture(t)
		fx.fake.SetResponse("CekSaldo", digiflazzclient.CekSaldoResponse{Deposit: 100000}, nil)
		fx.fake.SetResponse("Deposit", digiflazzclient.DepositResponse{Rc: "00"}, nil)
		_, err := fx.svc.UpsertCredential(context.Background(), fx.familyID, fx.ownerID, digiflazzdomain.UpsertCredentialRequest{Username: "buyer", APIKey: "secret-api-key-1234"})
		if err != nil {
			t.Fatalf("UpsertCredential returned error: %v", err)
		}

		resp, err := fx.svc.Deposit(context.Background(), fx.familyID, fx.memberID, 100000, "BCA")
		if err != nil {
			t.Fatalf("expected member to be able to deposit, got error: %v", err)
		}
		if resp == nil || resp.Rc != "00" {
			t.Fatalf("unexpected deposit response: %+v", resp)
		}
		if fx.fake.CallCount("Deposit") != 1 {
			t.Fatalf("expected 1 deposit call, got %d", fx.fake.CallCount("Deposit"))
		}
	})
}

func TestDigiflazzCredentialServiceUpdateValidationFailureDoesNotPersist(t *testing.T) {
	fx := setupDigiflazzCredentialServiceFixture(t)
	fx.fake.SetResponse("CekSaldo", digiflazzclient.CekSaldoResponse{Deposit: 100000}, nil)
	_, err := fx.svc.UpsertCredential(context.Background(), fx.familyID, fx.ownerID, digiflazzdomain.UpsertCredentialRequest{Username: "buyer", APIKey: "secret-api-key-1234"})
	if err != nil {
		t.Fatalf("UpsertCredential returned error: %v", err)
	}

	fx.fake.SetResponse("CekSaldo", nil, errors.New("invalid credential"))
	_, err = fx.svc.UpsertCredential(context.Background(), fx.familyID, fx.ownerID, digiflazzdomain.UpsertCredentialRequest{
		Username: "bad-buyer",
		APIKey:   "secret-api-key-1234",
	})
	if err == nil {
		t.Fatal("expected validation error")
	}

	credential, err := fx.repo.GetByFamilyID(fx.familyID)
	if err != nil {
		t.Fatalf("GetByFamilyID returned error: %v", err)
	}
	if credential.Username != "buyer" {
		t.Fatalf("expected username to remain buyer, got %+v", credential)
	}
}

func TestDigiflazzCredentialServiceRotateAndDelete(t *testing.T) {
	fx := setupDigiflazzCredentialServiceFixture(t)
	fx.fake.SetResponse("CekSaldo", digiflazzclient.CekSaldoResponse{Deposit: 100000}, nil)
	_, err := fx.svc.UpsertCredential(context.Background(), fx.familyID, fx.ownerID, digiflazzdomain.UpsertCredentialRequest{Username: "buyer", APIKey: "secret-api-key-1234"})
	if err != nil {
		t.Fatalf("UpsertCredential returned error: %v", err)
	}

	rotated, err := fx.svc.RotateWebhookToken(context.Background(), fx.familyID, fx.ownerID)
	if err != nil {
		t.Fatalf("RotateWebhookToken returned error: %v", err)
	}
	if rotated.Token == "" || rotated.Credential == nil || !rotated.Credential.WebhookConfigured {
		t.Fatalf("unexpected rotated response: %+v", rotated)
	}
	secret, err := fx.repo.GetSecretByFamilyID(fx.familyID)
	if err != nil {
		t.Fatalf("GetSecretByFamilyID returned error: %v", err)
	}
	if !utils.VerifyToken(rotated.Token, secret.WebhookTokenHash) {
		t.Fatal("stored webhook token hash does not verify returned token")
	}

	if err := fx.svc.DeleteCredential(context.Background(), fx.familyID, fx.ownerID); err != nil {
		t.Fatalf("DeleteCredential returned error: %v", err)
	}
	credential, err := fx.repo.GetByFamilyID(fx.familyID)
	if err != nil {
		t.Fatalf("GetByFamilyID returned error: %v", err)
	}
	if credential != nil {
		t.Fatalf("expected credential deleted, got %+v", credential)
	}
}
