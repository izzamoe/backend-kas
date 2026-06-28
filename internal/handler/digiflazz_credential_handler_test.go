package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"

	digiflazzclient "kas/internal/digiflazz"
	digiflazzdomain "kas/internal/domain/digiflazz"
	"kas/internal/handler"
	"kas/internal/middleware"
	"kas/internal/repository"
	_ "kas/migrations"
)

type fakeCredentialService struct {
	getCredential      func(ctx context.Context, familyID, userID string) (*digiflazzdomain.CredentialDTO, error)
	upsertCredential   func(ctx context.Context, familyID, userID string, req digiflazzdomain.UpsertCredentialRequest) (*digiflazzdomain.UpsertCredentialResult, error)
	deleteCredential   func(ctx context.Context, familyID, userID string) error
	rotateWebhookToken func(ctx context.Context, familyID, userID string) (*digiflazzdomain.RotateWebhookTokenResponse, error)
	testWebhook        func(ctx context.Context, familyID, userID string) (*digiflazzdomain.WebhookTestResponse, error)
	checkBalance       func(ctx context.Context, familyID, userID string) (*digiflazzdomain.BalanceResponse, error)
	deposit            func(ctx context.Context, familyID, userID string, amount float64, bank string) (*digiflazzclient.DepositResponse, error)
}

func (f *fakeCredentialService) GetCredential(ctx context.Context, familyID, userID string) (*digiflazzdomain.CredentialDTO, error) {
	if f.getCredential != nil {
		return f.getCredential(ctx, familyID, userID)
	}
	return nil, errors.New("digiflazz credential not found")
}

func (f *fakeCredentialService) UpsertCredential(ctx context.Context, familyID, userID string, req digiflazzdomain.UpsertCredentialRequest) (*digiflazzdomain.UpsertCredentialResult, error) {
	if f.upsertCredential != nil {
		return f.upsertCredential(ctx, familyID, userID, req)
	}
	return nil, errors.New("upsert failed")
}

func (f *fakeCredentialService) DeleteCredential(ctx context.Context, familyID, userID string) error {
	if f.deleteCredential != nil {
		return f.deleteCredential(ctx, familyID, userID)
	}
	return errors.New("delete failed")
}

func (f *fakeCredentialService) RotateWebhookToken(ctx context.Context, familyID, userID string) (*digiflazzdomain.RotateWebhookTokenResponse, error) {
	if f.rotateWebhookToken != nil {
		return f.rotateWebhookToken(ctx, familyID, userID)
	}
	return nil, errors.New("rotate failed")
}

func (f *fakeCredentialService) TestWebhook(ctx context.Context, familyID, userID string) (*digiflazzdomain.WebhookTestResponse, error) {
	if f.testWebhook != nil {
		return f.testWebhook(ctx, familyID, userID)
	}
	return nil, errors.New("test webhook failed")
}

func (f *fakeCredentialService) CheckBalance(ctx context.Context, familyID, userID string) (*digiflazzdomain.BalanceResponse, error) {
	if f.checkBalance != nil {
		return f.checkBalance(ctx, familyID, userID)
	}
	return nil, errors.New("check balance failed")
}

func (f *fakeCredentialService) Deposit(ctx context.Context, familyID, userID string, amount float64, bank string) (*digiflazzclient.DepositResponse, error) {
	if f.deposit != nil {
		return f.deposit(ctx, familyID, userID, amount, bank)
	}
	return nil, errors.New("deposit failed")
}

func seedDigiflazzCredentialTestData(t *testing.T, app *tests.TestApp) (userToken, familyID, userID string) {
	t.Helper()

	userCol, err := app.FindCollectionByNameOrId("users")
	if err != nil {
		t.Fatalf("find users: %v", err)
	}
	user := core.NewRecord(userCol)
	user.Set("name", "Cred Test Owner")
	user.Set("email", fmt.Sprintf("credtest+%d@example.com", time.Now().UnixNano()))
	user.SetPassword("password12345")
	if err := app.Save(user); err != nil {
		t.Fatalf("save user: %v", err)
	}

	familyCol, err := app.FindCollectionByNameOrId("families")
	if err != nil {
		t.Fatalf("find families: %v", err)
	}
	family := core.NewRecord(familyCol)
	family.Set("name", "Cred Test Family")
	family.Set("invite_code", fmt.Sprintf("CRED%d", time.Now().UnixNano()))
	if err := app.Save(family); err != nil {
		t.Fatalf("save family: %v", err)
	}

	memberCol, err := app.FindCollectionByNameOrId("family_members")
	if err != nil {
		t.Fatalf("find family_members: %v", err)
	}
	member := core.NewRecord(memberCol)
	member.Set("user_id", user.Id)
	member.Set("family_id", family.Id)
	member.Set("role", "owner")
	if err := app.Save(member); err != nil {
		t.Fatalf("save member: %v", err)
	}

	token, err := user.NewAuthToken()
	if err != nil {
		t.Fatalf("new auth token: %v", err)
	}

	return token, family.Id, user.Id
}

func seedDigiflazzCredentialMemberToken(t *testing.T, app *tests.TestApp, familyID string) (string, string) {
	t.Helper()

	userCol, err := app.FindCollectionByNameOrId("users")
	if err != nil {
		t.Fatalf("find users: %v", err)
	}
	user := core.NewRecord(userCol)
	user.Set("name", "Cred Test Member")
	user.Set("email", fmt.Sprintf("credtest-member+%d@example.com", time.Now().UnixNano()))
	user.SetPassword("password12345")
	if err := app.Save(user); err != nil {
		t.Fatalf("save member user: %v", err)
	}

	memberCol, err := app.FindCollectionByNameOrId("family_members")
	if err != nil {
		t.Fatalf("find family_members: %v", err)
	}
	member := core.NewRecord(memberCol)
	member.Set("user_id", user.Id)
	member.Set("family_id", familyID)
	member.Set("role", "member")
	if err := app.Save(member); err != nil {
		t.Fatalf("save member role: %v", err)
	}

	token, err := user.NewAuthToken()
	if err != nil {
		t.Fatalf("new auth token: %v", err)
	}

	return token, user.Id
}

func newDigiflazzCredentialTestApp(t *testing.T) *tests.TestApp {
	t.Helper()
	app, err := tests.NewTestApp()
	if err != nil {
		t.Fatalf("failed to create test app: %v", err)
	}
	t.Cleanup(app.Cleanup)
	return app
}

func bindDigiflazzCredentialRoutes(app *tests.TestApp, e *core.ServeEvent, svc *fakeCredentialService) {
	familyMemberRepo := repository.NewFamilyMemberRepository(app)
	requireFamily := middleware.RequireFamily(familyMemberRepo)
	requireFamilyOwner := middleware.RequireFamilyOwner()
	h := handler.NewDigiflazzCredentialHandler(svc, middleware.RequireAuth, requireFamily, requireFamilyOwner)
	h.RegisterRoutes(e)
}

func TestDigiflazzCredentialHandler_Get(t *testing.T) {
	app := newDigiflazzCredentialTestApp(t)
	token, familyID, userID := seedDigiflazzCredentialTestData(t, app)

	fakeDTO := &digiflazzdomain.CredentialDTO{
		ID:       "cred1",
		FamilyID: familyID,
		Username: "testuser",
		IsActive: true,
	}

	svc := &fakeCredentialService{
		getCredential: func(ctx context.Context, fID, uID string) (*digiflazzdomain.CredentialDTO, error) {
			if fID != familyID || uID != userID {
				return nil, errors.New("unauthorized: only family owner can manage digiflazz credentials")
			}
			return fakeDTO, nil
		},
	}

	scenarios := []tests.ApiScenario{
		{
			Name:   "get credential success",
			Method: http.MethodGet,
			URL:    "/api/digiflazz/credential",
			Headers: map[string]string{
				"Authorization": "Bearer " + token,
			},
			ExpectedStatus:  http.StatusOK,
			ExpectedContent: []string{`"id"`, `"family_id"`, `"username"`},
			TestAppFactory:  func(tb testing.TB) *tests.TestApp { return app },
			BeforeTestFunc: func(tb testing.TB, svrApp *tests.TestApp, e *core.ServeEvent) {
				bindDigiflazzCredentialRoutes(svrApp, e, svc)
			},
		},
	}

	for _, scenario := range scenarios {
		scenario.Test(t)
	}
}

func TestDigiflazzCredentialHandler_Delete(t *testing.T) {
	app := newDigiflazzCredentialTestApp(t)
	token, _, _ := seedDigiflazzCredentialTestData(t, app)

	svc := &fakeCredentialService{
		deleteCredential: func(ctx context.Context, fID, uID string) error {
			return nil
		},
	}

	scenarios := []tests.ApiScenario{
		{
			Name:   "delete credential success",
			Method: http.MethodDelete,
			URL:    "/api/digiflazz/credential",
			Headers: map[string]string{
				"Authorization": "Bearer " + token,
			},
			ExpectedStatus: http.StatusNoContent,
			TestAppFactory: func(tb testing.TB) *tests.TestApp { return app },
			BeforeTestFunc: func(tb testing.TB, svrApp *tests.TestApp, e *core.ServeEvent) {
				bindDigiflazzCredentialRoutes(svrApp, e, svc)
			},
		},
	}

	for _, scenario := range scenarios {
		scenario.Test(t)
	}
}

func TestDigiflazzCredentialHandler_RotateToken(t *testing.T) {
	app := newDigiflazzCredentialTestApp(t)
	token, familyID, _ := seedDigiflazzCredentialTestData(t, app)

	svc := &fakeCredentialService{
		rotateWebhookToken: func(ctx context.Context, fID, uID string) (*digiflazzdomain.RotateWebhookTokenResponse, error) {
			return &digiflazzdomain.RotateWebhookTokenResponse{
				Credential: &digiflazzdomain.CredentialDTO{ID: "cred1", FamilyID: familyID},
				Token:      "new-webhook-token-abc123",
			}, nil
		},
	}

	scenarios := []tests.ApiScenario{
		{
			Name:   "rotate webhook token success",
			Method: http.MethodPost,
			URL:    "/api/digiflazz/credential/rotate",
			Headers: map[string]string{
				"Authorization": "Bearer " + token,
			},
			ExpectedStatus:  http.StatusOK,
			ExpectedContent: []string{`"credential"`, `"token"`},
			TestAppFactory:  func(t testing.TB) *tests.TestApp { return app },
			BeforeTestFunc: func(t testing.TB, svrApp *tests.TestApp, e *core.ServeEvent) {
				bindDigiflazzCredentialRoutes(svrApp, e, svc)
			},
		},
	}

	for _, scenario := range scenarios {
		scenario.Test(t)
	}
}

func TestDigiflazzCredentialHandler_TestWebhook(t *testing.T) {
	app := newDigiflazzCredentialTestApp(t)
	token, familyID, userID := seedDigiflazzCredentialTestData(t, app)

	svc := &fakeCredentialService{
		testWebhook: func(ctx context.Context, fID, uID string) (*digiflazzdomain.WebhookTestResponse, error) {
			if fID != familyID || uID != userID {
				return nil, errors.New("unexpected auth context")
			}
			return &digiflazzdomain.WebhookTestResponse{
				Sed:    "ping-sed",
				HookID: "hook-123",
				Hook:   digiflazzdomain.WebhookTestHook{URL: "https://example.test/webhook", Type: "application/json", Status: 1},
			}, nil
		},
	}

	scenarios := []tests.ApiScenario{
		{
			Name:   "test webhook success",
			Method: http.MethodPost,
			URL:    "/api/digiflazz/credential/test-webhook",
			Headers: map[string]string{
				"Authorization": "Bearer " + token,
			},
			ExpectedStatus:     http.StatusOK,
			ExpectedContent:    []string{`"hook_id":"hook-123"`, `"status":1`, `"url":"https://example.test/webhook"`},
			NotExpectedContent: []string{`"secret"`, "must-not-leak"},
			TestAppFactory:     func(t testing.TB) *tests.TestApp { return app },
			BeforeTestFunc: func(t testing.TB, svrApp *tests.TestApp, e *core.ServeEvent) {
				bindDigiflazzCredentialRoutes(svrApp, e, svc)
			},
		},
	}

	for _, scenario := range scenarios {
		scenario.Test(t)
	}
}

func TestDigiflazzCredentialHandler_TestWebhookForbiddenForMember(t *testing.T) {
	app := newDigiflazzCredentialTestApp(t)
	_, familyID, _ := seedDigiflazzCredentialTestData(t, app)
	memberToken, _ := seedDigiflazzCredentialMemberToken(t, app, familyID)

	svc := &fakeCredentialService{}

	scenarios := []tests.ApiScenario{
		{
			Name:   "test webhook - non-owner member gets 403",
			Method: http.MethodPost,
			URL:    "/api/digiflazz/credential/test-webhook",
			Headers: map[string]string{
				"Authorization": "Bearer " + memberToken,
			},
			ExpectedStatus:  http.StatusForbidden,
			ExpectedContent: []string{`"status":403`},
			TestAppFactory:  func(t testing.TB) *tests.TestApp { return app },
			BeforeTestFunc: func(t testing.TB, svrApp *tests.TestApp, e *core.ServeEvent) {
				bindDigiflazzCredentialRoutes(svrApp, e, svc)
			},
		},
	}

	for _, scenario := range scenarios {
		scenario.Test(t)
	}
}

func TestDigiflazzCredentialHandler_CheckBalance(t *testing.T) {
	app := newDigiflazzCredentialTestApp(t)
	token, familyID, _ := seedDigiflazzCredentialTestData(t, app)

	svc := &fakeCredentialService{
		checkBalance: func(ctx context.Context, fID, uID string) (*digiflazzdomain.BalanceResponse, error) {
			return &digiflazzdomain.BalanceResponse{
				FamilyID: familyID,
				Balance:  500000,
			}, nil
		},
	}

	scenarios := []tests.ApiScenario{
		{
			Name:   "check balance success",
			Method: http.MethodGet,
			URL:    "/api/digiflazz/credential/balance",
			Headers: map[string]string{
				"Authorization": "Bearer " + token,
			},
			ExpectedStatus:  http.StatusOK,
			ExpectedContent: []string{`"family_id"`, `"balance"`},
			TestAppFactory:  func(t testing.TB) *tests.TestApp { return app },
			BeforeTestFunc: func(t testing.TB, svrApp *tests.TestApp, e *core.ServeEvent) {
				bindDigiflazzCredentialRoutes(svrApp, e, svc)
			},
		},
	}

	for _, scenario := range scenarios {
		scenario.Test(t)
	}
}

func TestDigiflazzDepositHandler(t *testing.T) {
	app := newDigiflazzCredentialTestApp(t)
	token, _, _ := seedDigiflazzCredentialTestData(t, app)

	svc := &fakeCredentialService{
		deposit: func(ctx context.Context, fID, uID string, amount float64, bank string) (*digiflazzclient.DepositResponse, error) {
			return &digiflazzclient.DepositResponse{
				Rc:            "00",
				Bank:          bank,
				PaymentMethod: "transfer",
				AccountNo:     "1234567890",
				Amount:        amount,
			}, nil
		},
	}

	body, _ := json.Marshal(map[string]any{"amount": 500000, "bank": "BCA"})

	scenarios := []tests.ApiScenario{
		{
			Name:   "deposit success",
			Method: http.MethodPost,
			URL:    "/api/digiflazz/deposit",
			Body:   bytes.NewReader(body),
			Headers: map[string]string{
				"Authorization": "Bearer " + token,
				"Content-Type":  "application/json",
			},
			ExpectedStatus:  http.StatusOK,
			ExpectedContent: []string{`"rc"`, `"bank"`, `"account_no"`},
			TestAppFactory:  func(t testing.TB) *tests.TestApp { return app },
			BeforeTestFunc: func(t testing.TB, svrApp *tests.TestApp, e *core.ServeEvent) {
				bindDigiflazzCredentialRoutes(svrApp, e, svc)
			},
		},
	}

	for _, scenario := range scenarios {
		scenario.Test(t)
	}
}

func TestDigiflazzDepositHandlerForbiddenForMember(t *testing.T) {
	app := newDigiflazzCredentialTestApp(t)
	_, familyID, _ := seedDigiflazzCredentialTestData(t, app)
	memberToken, _ := seedDigiflazzCredentialMemberToken(t, app, familyID)

	svc := &fakeCredentialService{
		deposit: func(ctx context.Context, fID, uID string, amount float64, bank string) (*digiflazzclient.DepositResponse, error) {
			return nil, errors.New("unauthorized: only family owner can manage digiflazz credentials")
		},
	}

	body, _ := json.Marshal(map[string]any{"amount": 100000, "bank": "BNI"})

	scenarios := []tests.ApiScenario{
		{
			Name:   "deposit forbidden for member",
			Method: http.MethodPost,
			URL:    "/api/digiflazz/deposit",
			Body:   bytes.NewReader(body),
			Headers: map[string]string{
				"Authorization": "Bearer " + memberToken,
				"Content-Type":  "application/json",
			},
			ExpectedStatus:  http.StatusForbidden,
			ExpectedContent: []string{`"status":403`},
			TestAppFactory:  func(t testing.TB) *tests.TestApp { return app },
			BeforeTestFunc: func(t testing.TB, svrApp *tests.TestApp, e *core.ServeEvent) {
				bindDigiflazzCredentialRoutes(svrApp, e, svc)
			},
		},
	}

	for _, scenario := range scenarios {
		scenario.Test(t)
	}
}

func TestDigiflazzCredentialHandler_UnauthorizedErrors(t *testing.T) {
	app := newDigiflazzCredentialTestApp(t)
	token, _, _ := seedDigiflazzCredentialTestData(t, app)

	svc := &fakeCredentialService{
		getCredential: func(ctx context.Context, fID, uID string) (*digiflazzdomain.CredentialDTO, error) {
			return nil, errors.New("unauthorized: only family owner can manage digiflazz credentials")
		},
	}

	scenarios := []tests.ApiScenario{
		{
			Name:   "get credential - forbidden for non-owner",
			Method: http.MethodGet,
			URL:    "/api/digiflazz/credential",
			Headers: map[string]string{
				"Authorization": "Bearer " + token,
			},
			ExpectedStatus:  http.StatusForbidden,
			ExpectedContent: []string{`"status":403`},
			TestAppFactory:  func(t testing.TB) *tests.TestApp { return app },
			BeforeTestFunc: func(t testing.TB, svrApp *tests.TestApp, e *core.ServeEvent) {
				bindDigiflazzCredentialRoutes(svrApp, e, svc)
			},
		},
	}

	for _, scenario := range scenarios {
		scenario.Test(t)
	}
}

func TestDigiflazzCredentialHandler_RequiresAuth(t *testing.T) {
	app := newDigiflazzCredentialTestApp(t)
	svc := &fakeCredentialService{}

	scenarios := []tests.ApiScenario{
		{
			Name:            "get credential - requires auth",
			Method:          http.MethodGet,
			URL:             "/api/digiflazz/credential",
			ExpectedStatus:  http.StatusUnauthorized,
			ExpectedContent: []string{`"status":401`},
			TestAppFactory:  func(t testing.TB) *tests.TestApp { return app },
			BeforeTestFunc: func(t testing.TB, svrApp *tests.TestApp, e *core.ServeEvent) {
				bindDigiflazzCredentialRoutes(svrApp, e, svc)
			},
		},
	}

	for _, scenario := range scenarios {
		scenario.Test(t)
	}
}

func TestCredential_Upsert_Create(t *testing.T) {
	app := newDigiflazzCredentialTestApp(t)
	token, _, _ := seedDigiflazzCredentialTestData(t, app)

	svc := &fakeCredentialService{
		upsertCredential: func(ctx context.Context, fID, uID string, req digiflazzdomain.UpsertCredentialRequest) (*digiflazzdomain.UpsertCredentialResult, error) {
			return &digiflazzdomain.UpsertCredentialResult{
				Credential:      &digiflazzdomain.CredentialDTO{ID: "cred-new", FamilyID: fID, Username: req.Username, IsActive: true},
				SyncInitiated:   true,
				RawWebhookToken: "raw-webhook-token-abc123",
			}, nil
		},
	}

	body, _ := json.Marshal(map[string]any{"username": "testuser", "api_key": "testapikey"})

	scenarios := []tests.ApiScenario{
		{
			Name:   "upsert create - 200 with sync_initiated and webhook_url",
			Method: http.MethodPost,
			URL:    "/api/digiflazz/credential",
			Body:   bytes.NewReader(body),
			Headers: map[string]string{
				"Authorization": "Bearer " + token,
				"Content-Type":  "application/json",
			},
			ExpectedStatus:  http.StatusOK,
			ExpectedContent: []string{`"sync_initiated":true`, `"webhook_url"`},
			TestAppFactory:  func(t testing.TB) *tests.TestApp { return app },
			BeforeTestFunc: func(t testing.TB, svrApp *tests.TestApp, e *core.ServeEvent) {
				bindDigiflazzCredentialRoutes(svrApp, e, svc)
			},
		},
	}

	for _, scenario := range scenarios {
		scenario.Test(t)
	}
}

func TestCredential_Upsert_Update(t *testing.T) {
	app := newDigiflazzCredentialTestApp(t)
	token, _, _ := seedDigiflazzCredentialTestData(t, app)

	svc := &fakeCredentialService{
		upsertCredential: func(ctx context.Context, fID, uID string, req digiflazzdomain.UpsertCredentialRequest) (*digiflazzdomain.UpsertCredentialResult, error) {
			return &digiflazzdomain.UpsertCredentialResult{
				Credential:      &digiflazzdomain.CredentialDTO{ID: "cred-existing", FamilyID: fID, Username: req.Username, IsActive: true},
				SyncInitiated:   true,
				RawWebhookToken: "",
			}, nil
		},
	}

	body, _ := json.Marshal(map[string]any{"username": "updateduser", "api_key": "updatedapikey"})

	scenarios := []tests.ApiScenario{
		{
			Name:   "upsert update - 200 sync_initiated true no webhook_url",
			Method: http.MethodPost,
			URL:    "/api/digiflazz/credential",
			Body:   bytes.NewReader(body),
			Headers: map[string]string{
				"Authorization": "Bearer " + token,
				"Content-Type":  "application/json",
			},
			ExpectedStatus:     http.StatusOK,
			ExpectedContent:    []string{`"sync_initiated":true`},
			NotExpectedContent: []string{`"webhook_url"`},
			TestAppFactory:     func(t testing.TB) *tests.TestApp { return app },
			BeforeTestFunc: func(t testing.TB, svrApp *tests.TestApp, e *core.ServeEvent) {
				bindDigiflazzCredentialRoutes(svrApp, e, svc)
			},
		},
	}

	for _, scenario := range scenarios {
		scenario.Test(t)
	}
}

func TestCredential_Upsert_WithWebhookSecret(t *testing.T) {
	app := newDigiflazzCredentialTestApp(t)
	token, _, _ := seedDigiflazzCredentialTestData(t, app)

	var capturedSecret string
	svc := &fakeCredentialService{
		upsertCredential: func(ctx context.Context, fID, uID string, req digiflazzdomain.UpsertCredentialRequest) (*digiflazzdomain.UpsertCredentialResult, error) {
			if req.WebhookSecret != nil {
				capturedSecret = *req.WebhookSecret
			}
			return &digiflazzdomain.UpsertCredentialResult{
				Credential:      &digiflazzdomain.CredentialDTO{ID: "cred-new", FamilyID: fID, Username: req.Username, IsActive: true},
				SyncInitiated:   true,
				RawWebhookToken: "raw-token-xyz",
			}, nil
		},
	}

	secret := "my-webhook-secret"
	body, _ := json.Marshal(map[string]any{
		"username":       "testuser",
		"api_key":        "testapikey",
		"webhook_secret": secret,
	})

	scenarios := []tests.ApiScenario{
		{
			Name:   "upsert with webhook_secret - 200 secret forwarded to service",
			Method: http.MethodPost,
			URL:    "/api/digiflazz/credential",
			Body:   bytes.NewReader(body),
			Headers: map[string]string{
				"Authorization": "Bearer " + token,
				"Content-Type":  "application/json",
			},
			ExpectedStatus:  http.StatusOK,
			ExpectedContent: []string{`"sync_initiated":true`, `"webhook_url"`},
			TestAppFactory:  func(t testing.TB) *tests.TestApp { return app },
			BeforeTestFunc: func(t testing.TB, svrApp *tests.TestApp, e *core.ServeEvent) {
				bindDigiflazzCredentialRoutes(svrApp, e, svc)
			},
		},
	}

	for _, scenario := range scenarios {
		scenario.Test(t)
	}

	if capturedSecret != secret {
		t.Errorf("webhook_secret not forwarded: got %q, want %q", capturedSecret, secret)
	}
}

func TestCredential_Delete_Cascade(t *testing.T) {
	app := newDigiflazzCredentialTestApp(t)
	token, familyID, _ := seedDigiflazzCredentialTestData(t, app)

	var deletedFamilyID string
	svc := &fakeCredentialService{
		deleteCredential: func(ctx context.Context, fID, uID string) error {
			deletedFamilyID = fID
			return nil
		},
	}

	scenarios := []tests.ApiScenario{
		{
			Name:   "delete cascade - 204 service called with correct familyID",
			Method: http.MethodDelete,
			URL:    "/api/digiflazz/credential",
			Headers: map[string]string{
				"Authorization": "Bearer " + token,
			},
			ExpectedStatus: http.StatusNoContent,
			TestAppFactory: func(t testing.TB) *tests.TestApp { return app },
			BeforeTestFunc: func(t testing.TB, svrApp *tests.TestApp, e *core.ServeEvent) {
				bindDigiflazzCredentialRoutes(svrApp, e, svc)
			},
		},
	}

	for _, scenario := range scenarios {
		scenario.Test(t)
	}

	if deletedFamilyID != familyID {
		t.Errorf("delete called with wrong familyID: got %q, want %q", deletedFamilyID, familyID)
	}
}

func TestCredential_Upsert_NonOwner_Forbidden(t *testing.T) {
	app := newDigiflazzCredentialTestApp(t)
	_, familyID, _ := seedDigiflazzCredentialTestData(t, app)
	memberToken, _ := seedDigiflazzCredentialMemberToken(t, app, familyID)

	svc := &fakeCredentialService{}

	body, _ := json.Marshal(map[string]any{"username": "testuser", "api_key": "testapikey"})

	scenarios := []tests.ApiScenario{
		{
			Name:   "upsert - non-owner member gets 403",
			Method: http.MethodPost,
			URL:    "/api/digiflazz/credential",
			Body:   bytes.NewReader(body),
			Headers: map[string]string{
				"Authorization": "Bearer " + memberToken,
				"Content-Type":  "application/json",
			},
			ExpectedStatus:  http.StatusForbidden,
			ExpectedContent: []string{`"status":403`},
			TestAppFactory:  func(t testing.TB) *tests.TestApp { return app },
			BeforeTestFunc: func(t testing.TB, svrApp *tests.TestApp, e *core.ServeEvent) {
				bindDigiflazzCredentialRoutes(svrApp, e, svc)
			},
		},
	}

	for _, scenario := range scenarios {
		scenario.Test(t)
	}
}

func TestCredential_Patch_NotFound(t *testing.T) {
	app := newDigiflazzCredentialTestApp(t)
	token, _, _ := seedDigiflazzCredentialTestData(t, app)

	svc := &fakeCredentialService{}
	body, _ := json.Marshal(map[string]any{"username": "test"})

	scenarios := []tests.ApiScenario{
		{
			Name:   "PATCH /credential - 404 route removed",
			Method: http.MethodPatch,
			URL:    "/api/digiflazz/credential",
			Body:   bytes.NewReader(body),
			Headers: map[string]string{
				"Authorization": "Bearer " + token,
				"Content-Type":  "application/json",
			},
			ExpectedStatus:  http.StatusNotFound,
			ExpectedContent: []string{`"status":404`},
			TestAppFactory:  func(t testing.TB) *tests.TestApp { return app },
			BeforeTestFunc: func(t testing.TB, svrApp *tests.TestApp, e *core.ServeEvent) {
				bindDigiflazzCredentialRoutes(svrApp, e, svc)
			},
		},
	}

	for _, scenario := range scenarios {
		scenario.Test(t)
	}
}
