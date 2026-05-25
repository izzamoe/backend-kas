package handler_test

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"net/http"
	"testing"

	digiflazzdomain "kas/internal/domain/digiflazz"
	"kas/internal/handler"
	"kas/internal/repository"
	"kas/internal/service"
	"kas/internal/utils"
	_ "kas/migrations"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
)

type fakeWebhookCredentialRepo struct {
	credential *repository.DigiflazzCredentialRecord
}

func (f *fakeWebhookCredentialRepo) GetSecretByWebhookTokenHash(tokenHash string) (*repository.DigiflazzCredentialRecord, error) {
	if f.credential == nil || f.credential.WebhookTokenHash != tokenHash {
		return nil, nil
	}
	return f.credential, nil
}

type fakeWebhookOrderRepo struct {
	order *digiflazzdomain.OrderDTO
}

func (f *fakeWebhookOrderRepo) GetByRefID(familyID, refID string) (*digiflazzdomain.OrderDTO, error) {
	if f.order == nil || f.order.RefID != refID {
		return nil, nil
	}
	return f.order, nil
}

type fakeWebhookEventRepo struct {
	exists  bool
	created []*repository.DigiflazzEventCreateData
}

func (f *fakeWebhookEventRepo) Create(data *repository.DigiflazzEventCreateData) (*repository.DigiflazzEventRecord, error) {
	f.created = append(f.created, data)
	return &repository.DigiflazzEventRecord{ID: fmt.Sprintf("event%d", len(f.created)), OrderID: data.OrderID, PayloadHash: data.PayloadHash}, nil
}

func (f *fakeWebhookEventRepo) ExistsByOrderAndPayloadHash(orderID, payloadHash string) (bool, error) {
	return f.exists, nil
}

type fakeWebhookOrderService struct {
	updates []digiflazzdomain.OrderStatus
}

var _ service.DigiflazzOrderService = (*fakeWebhookOrderService)(nil)

func (f *fakeWebhookOrderService) CreateOrder(_ context.Context, _, _ string, _ digiflazzdomain.CreateOrderRequest) (*digiflazzdomain.OrderDTO, error) {
	return nil, fmt.Errorf("not implemented")
}
func (f *fakeWebhookOrderService) CreatePrepaidOrder(context.Context, *digiflazzdomain.CreateOrderRequest, string, string) (*digiflazzdomain.OrderDTO, error) {
	return nil, fmt.Errorf("not implemented")
}
func (f *fakeWebhookOrderService) CreatePostpaidInquiry(context.Context, *digiflazzdomain.CreateOrderRequest, string, string) (*digiflazzdomain.OrderDTO, error) {
	return nil, fmt.Errorf("not implemented")
}
func (f *fakeWebhookOrderService) PayPostpaidOrder(context.Context, string, string, string) (*digiflazzdomain.OrderDTO, error) {
	return nil, fmt.Errorf("not implemented")
}
func (f *fakeWebhookOrderService) CheckPostpaidStatus(context.Context, string, string, string) (*digiflazzdomain.OrderDTO, error) {
	return nil, fmt.Errorf("not implemented")
}
func (f *fakeWebhookOrderService) GetOrder(string, string) (*digiflazzdomain.OrderDTO, error) {
	return nil, nil
}
func (f *fakeWebhookOrderService) ListFamilyOrders(string, int, int) ([]*digiflazzdomain.OrderDTO, error) {
	return nil, nil
}
func (f *fakeWebhookOrderService) UpdateStatus(familyID, id string, status digiflazzdomain.OrderStatus, response *digiflazzdomain.OrderResponseDTO) (*digiflazzdomain.OrderDTO, error) {
	f.updates = append(f.updates, status)
	return &digiflazzdomain.OrderDTO{ID: id, FamilyID: familyID, Status: status, CredentialID: "cred1", RefID: "REF-1"}, nil
}
func (f *fakeWebhookOrderService) FinalizeSuccessOrder(string) (*digiflazzdomain.OrderDTO, error) {
	return nil, nil
}
func (f *fakeWebhookOrderService) CheckAndUpdateStatus(context.Context, string) (*digiflazzdomain.OrderDTO, error) {
	return nil, nil
}
func (f *fakeWebhookOrderService) InquiryPLN(_ context.Context, _, _ string) (*digiflazzdomain.PLNInquiryResult, error) {
	return nil, nil
}

func newDigiflazzWebhookTestApp(t *testing.T) *tests.TestApp {
	t.Helper()
	app, err := tests.NewTestApp()
	if err != nil {
		t.Fatalf("failed to create test app: %v", err)
	}
	t.Cleanup(app.Cleanup)
	return app
}

func bindDigiflazzWebhookRoutes(e *core.ServeEvent, credentialRepo *fakeWebhookCredentialRepo, orderRepo *fakeWebhookOrderRepo, eventRepo *fakeWebhookEventRepo, orderService *fakeWebhookOrderService) {
	h := handler.NewDigiflazzWebhookHandler(credentialRepo, orderRepo, eventRepo, orderService)
	h.RegisterRoutes(e)
}

func TestDigiflazzWebhookHandler_ValidWebhookUpdatesOrderStatus(t *testing.T) {
	app := newDigiflazzWebhookTestApp(t)
	token := "webhook-token"
	secret := "shared-secret"
	body := []byte(`{"ref_id":"REF-1","customer_no":"08123456789","buyer_sku_code":"PLN20","message":"Sukses","status":"success","rc":"00","sn":"SN-SECRET","price":20000,"selling_price":21500}`)
	eventRepo := &fakeWebhookEventRepo{}
	orderService := &fakeWebhookOrderService{}

	(&tests.ApiScenario{
		Name:   "valid digiflazz webhook",
		Method: http.MethodPost,
		URL:    "/webhooks/digiflazz/" + token,
		Body:   bytes.NewReader(body),
		Headers: map[string]string{
			"Content-Type":          "application/json",
			"X-Digiflazz-Event":     "update",
			"X-Digiflazz-Signature": signDigiflazzWebhook(secret, body),
		},
		ExpectedStatus:  http.StatusOK,
		ExpectedContent: []string{`"status":"received"`},
		TestAppFactory:  func(t testing.TB) *tests.TestApp { return app },
		BeforeTestFunc: func(t testing.TB, svrApp *tests.TestApp, e *core.ServeEvent) {
			bindDigiflazzWebhookRoutes(e,
				&fakeWebhookCredentialRepo{credential: &repository.DigiflazzCredentialRecord{ID: "cred1", FamilyID: "fam1", WebhookTokenHash: utils.HashString(token), WebhookSecret: secret}},
				&fakeWebhookOrderRepo{order: &digiflazzdomain.OrderDTO{ID: "order1", FamilyID: "fam1", CredentialID: "cred1", RefID: "REF-1", Status: digiflazzdomain.OrderStatusProcessing}},
				eventRepo,
				orderService,
			)
		},
		AfterTestFunc: func(t testing.TB, app *tests.TestApp, res *http.Response) {
			if len(eventRepo.created) != 1 {
				t.Fatalf("expected one webhook event, got %d", len(eventRepo.created))
			}
			event := eventRepo.created[0]
			if event.EventType != digiflazzdomain.EventTypeWebhook || event.PayloadHash != utils.HashString(string(body)) {
				t.Fatalf("unexpected event: %+v", event)
			}
			if bytes.Contains([]byte(event.Payload), []byte("08123456789")) || bytes.Contains([]byte(event.Payload), []byte("SN-SECRET")) {
				t.Fatalf("webhook payload was not redacted: %s", event.Payload)
			}
			if len(orderService.updates) != 1 || orderService.updates[0] != digiflazzdomain.OrderStatusSuccess {
				t.Fatalf("unexpected status updates: %+v", orderService.updates)
			}
		},
	}).Test(t)
}

func TestDigiflazzWebhookHandler_InvalidTokenReturnsNotFound(t *testing.T) {
	app := newDigiflazzWebhookTestApp(t)
	body := []byte(`{"ref_id":"REF-1"}`)

	(&tests.ApiScenario{
		Name:            "invalid webhook token",
		Method:          http.MethodPost,
		URL:             "/webhooks/digiflazz/bad-token",
		Body:            bytes.NewReader(body),
		ExpectedStatus:  http.StatusNotFound,
		ExpectedContent: []string{`"status":404`},
		TestAppFactory:  func(t testing.TB) *tests.TestApp { return app },
		BeforeTestFunc: func(t testing.TB, svrApp *tests.TestApp, e *core.ServeEvent) {
			bindDigiflazzWebhookRoutes(e, &fakeWebhookCredentialRepo{}, &fakeWebhookOrderRepo{}, &fakeWebhookEventRepo{}, &fakeWebhookOrderService{})
		},
	}).Test(t)
}

func TestDigiflazzWebhookHandler_MismatchedFamilyReturnsForbidden(t *testing.T) {
	app := newDigiflazzWebhookTestApp(t)
	token := "webhook-token"
	secret := "shared-secret"
	body := []byte(`{"ref_id":"REF-1","status":"success"}`)

	(&tests.ApiScenario{
		Name:   "mismatched webhook order family",
		Method: http.MethodPost,
		URL:    "/webhooks/digiflazz/" + token,
		Body:   bytes.NewReader(body),
		Headers: map[string]string{
			"X-Digiflazz-Signature": signDigiflazzWebhook(secret, body),
		},
		ExpectedStatus:  http.StatusForbidden,
		ExpectedContent: []string{`"status":403`},
		TestAppFactory:  func(t testing.TB) *tests.TestApp { return app },
		BeforeTestFunc: func(t testing.TB, svrApp *tests.TestApp, e *core.ServeEvent) {
			bindDigiflazzWebhookRoutes(e,
				&fakeWebhookCredentialRepo{credential: &repository.DigiflazzCredentialRecord{ID: "cred1", FamilyID: "fam1", WebhookTokenHash: utils.HashString(token), WebhookSecret: secret}},
				&fakeWebhookOrderRepo{order: &digiflazzdomain.OrderDTO{ID: "order1", FamilyID: "fam2", CredentialID: "cred1", RefID: "REF-1", Status: digiflazzdomain.OrderStatusProcessing}},
				&fakeWebhookEventRepo{},
				&fakeWebhookOrderService{},
			)
		},
	}).Test(t)
}

func TestDigiflazzWebhookHandler_DuplicateWebhookReturnsOKWithoutReprocessing(t *testing.T) {
	app := newDigiflazzWebhookTestApp(t)
	token := "webhook-token"
	secret := "shared-secret"
	body := []byte(`{"ref_id":"REF-1","status":"success"}`)
	eventRepo := &fakeWebhookEventRepo{exists: true}
	orderService := &fakeWebhookOrderService{}

	(&tests.ApiScenario{
		Name:   "duplicate digiflazz webhook",
		Method: http.MethodPost,
		URL:    "/webhooks/digiflazz/" + token,
		Body:   bytes.NewReader(body),
		Headers: map[string]string{
			"X-Digiflazz-Signature": signDigiflazzWebhook(secret, body),
		},
		ExpectedStatus:  http.StatusOK,
		ExpectedContent: []string{`"status":"received"`},
		TestAppFactory:  func(t testing.TB) *tests.TestApp { return app },
		BeforeTestFunc: func(t testing.TB, svrApp *tests.TestApp, e *core.ServeEvent) {
			bindDigiflazzWebhookRoutes(e,
				&fakeWebhookCredentialRepo{credential: &repository.DigiflazzCredentialRecord{ID: "cred1", FamilyID: "fam1", WebhookTokenHash: utils.HashString(token), WebhookSecret: secret}},
				&fakeWebhookOrderRepo{order: &digiflazzdomain.OrderDTO{ID: "order1", FamilyID: "fam1", CredentialID: "cred1", RefID: "REF-1", Status: digiflazzdomain.OrderStatusProcessing}},
				eventRepo,
				orderService,
			)
		},
		AfterTestFunc: func(t testing.TB, app *tests.TestApp, res *http.Response) {
			if len(eventRepo.created) != 0 || len(orderService.updates) != 0 {
				t.Fatalf("duplicate webhook was reprocessed: events=%d updates=%d", len(eventRepo.created), len(orderService.updates))
			}
		},
	}).Test(t)
}

func TestDigiflazzWebhookHandler_InvalidSignatureReturnsUnauthorized(t *testing.T) {
	app := newDigiflazzWebhookTestApp(t)
	token := "webhook-token"
	body := []byte(`{"ref_id":"REF-1","status":"success"}`)

	(&tests.ApiScenario{
		Name:   "invalid digiflazz webhook signature",
		Method: http.MethodPost,
		URL:    "/webhooks/digiflazz/" + token,
		Body:   bytes.NewReader(body),
		Headers: map[string]string{
			"X-Digiflazz-Signature": "sha1=deadbeef",
		},
		ExpectedStatus:  http.StatusUnauthorized,
		ExpectedContent: []string{`"status":401`},
		TestAppFactory:  func(t testing.TB) *tests.TestApp { return app },
		BeforeTestFunc: func(t testing.TB, svrApp *tests.TestApp, e *core.ServeEvent) {
			bindDigiflazzWebhookRoutes(e,
				&fakeWebhookCredentialRepo{credential: &repository.DigiflazzCredentialRecord{ID: "cred1", FamilyID: "fam1", WebhookTokenHash: utils.HashString(token), WebhookSecret: "shared-secret"}},
				&fakeWebhookOrderRepo{},
				&fakeWebhookEventRepo{},
				&fakeWebhookOrderService{},
			)
		},
	}).Test(t)
}

func signDigiflazzWebhook(secret string, body []byte) string {
	mac := hmac.New(sha1.New, []byte(secret))
	_, _ = mac.Write(body)
	return "sha1=" + hex.EncodeToString(mac.Sum(nil))
}

func TestDigiflazzWebhookHandler_XHubSignatureValid(t *testing.T) {
	app := newDigiflazzWebhookTestApp(t)
	token := "webhook-token"
	secret := "shared-secret"
	body := []byte(`{"ref_id":"REF-1","status":"success"}`)

	(&tests.ApiScenario{
		Name:   "X-Hub-Signature valid not 401",
		Method: http.MethodPost,
		URL:    "/webhooks/digiflazz/" + token,
		Body:   bytes.NewReader(body),
		Headers: map[string]string{
			"X-Hub-Signature": signDigiflazzWebhook(secret, body),
		},
		ExpectedStatus:  http.StatusOK,
		ExpectedContent: []string{`"status":"received"`},
		TestAppFactory:  func(t testing.TB) *tests.TestApp { return app },
		BeforeTestFunc: func(t testing.TB, svrApp *tests.TestApp, e *core.ServeEvent) {
			bindDigiflazzWebhookRoutes(e,
				&fakeWebhookCredentialRepo{credential: &repository.DigiflazzCredentialRecord{ID: "cred1", FamilyID: "fam1", WebhookTokenHash: utils.HashString(token), WebhookSecret: secret}},
				&fakeWebhookOrderRepo{},
				&fakeWebhookEventRepo{},
				&fakeWebhookOrderService{},
			)
		},
	}).Test(t)
}

func TestDigiflazzWebhookHandler_XDigiflazzSignatureFallback(t *testing.T) {
	app := newDigiflazzWebhookTestApp(t)
	token := "webhook-token"
	secret := "shared-secret"
	body := []byte(`{"ref_id":"REF-1","status":"success"}`)

	(&tests.ApiScenario{
		Name:   "X-Digiflazz-Signature fallback not 401",
		Method: http.MethodPost,
		URL:    "/webhooks/digiflazz/" + token,
		Body:   bytes.NewReader(body),
		Headers: map[string]string{
			"X-Digiflazz-Signature": signDigiflazzWebhook(secret, body),
		},
		ExpectedStatus:  http.StatusOK,
		ExpectedContent: []string{`"status":"received"`},
		TestAppFactory:  func(t testing.TB) *tests.TestApp { return app },
		BeforeTestFunc: func(t testing.TB, svrApp *tests.TestApp, e *core.ServeEvent) {
			bindDigiflazzWebhookRoutes(e,
				&fakeWebhookCredentialRepo{credential: &repository.DigiflazzCredentialRecord{ID: "cred1", FamilyID: "fam1", WebhookTokenHash: utils.HashString(token), WebhookSecret: secret}},
				&fakeWebhookOrderRepo{},
				&fakeWebhookEventRepo{},
				&fakeWebhookOrderService{},
			)
		},
	}).Test(t)
}

func TestDigiflazzWebhookHandler_XHubTakesPriorityOverXDigiflazz(t *testing.T) {
	app := newDigiflazzWebhookTestApp(t)
	token := "webhook-token"
	secret := "shared-secret"
	body := []byte(`{"ref_id":"REF-1","status":"success"}`)

	(&tests.ApiScenario{
		Name:   "X-Hub-Signature takes priority over X-Digiflazz-Signature",
		Method: http.MethodPost,
		URL:    "/webhooks/digiflazz/" + token,
		Body:   bytes.NewReader(body),
		Headers: map[string]string{
			"X-Hub-Signature":       signDigiflazzWebhook(secret, body),
			"X-Digiflazz-Signature": "sha1=deadbeef",
		},
		ExpectedStatus:  http.StatusOK,
		ExpectedContent: []string{`"status":"received"`},
		TestAppFactory:  func(t testing.TB) *tests.TestApp { return app },
		BeforeTestFunc: func(t testing.TB, svrApp *tests.TestApp, e *core.ServeEvent) {
			bindDigiflazzWebhookRoutes(e,
				&fakeWebhookCredentialRepo{credential: &repository.DigiflazzCredentialRecord{ID: "cred1", FamilyID: "fam1", WebhookTokenHash: utils.HashString(token), WebhookSecret: secret}},
				&fakeWebhookOrderRepo{},
				&fakeWebhookEventRepo{},
				&fakeWebhookOrderService{},
			)
		},
	}).Test(t)
}

func TestDigiflazzWebhookHandler_BothSignaturesAbsentWithSecretReturnsUnauthorized(t *testing.T) {
	app := newDigiflazzWebhookTestApp(t)
	token := "webhook-token"
	body := []byte(`{"ref_id":"REF-1","status":"success"}`)

	(&tests.ApiScenario{
		Name:            "both signatures absent with non-empty secret returns 401",
		Method:          http.MethodPost,
		URL:             "/webhooks/digiflazz/" + token,
		Body:            bytes.NewReader(body),
		ExpectedStatus:  http.StatusUnauthorized,
		ExpectedContent: []string{`"status":401`},
		TestAppFactory:  func(t testing.TB) *tests.TestApp { return app },
		BeforeTestFunc: func(t testing.TB, svrApp *tests.TestApp, e *core.ServeEvent) {
			bindDigiflazzWebhookRoutes(e,
				&fakeWebhookCredentialRepo{credential: &repository.DigiflazzCredentialRecord{ID: "cred1", FamilyID: "fam1", WebhookTokenHash: utils.HashString(token), WebhookSecret: "shared-secret"}},
				&fakeWebhookOrderRepo{},
				&fakeWebhookEventRepo{},
				&fakeWebhookOrderService{},
			)
		},
	}).Test(t)
}

func TestDigiflazzWebhookHandler_MalformedXHubSignatureReturnsUnauthorized(t *testing.T) {
	app := newDigiflazzWebhookTestApp(t)
	token := "webhook-token"
	body := []byte(`{"ref_id":"REF-1","status":"success"}`)

	(&tests.ApiScenario{
		Name:   "malformed X-Hub-Signature returns 401",
		Method: http.MethodPost,
		URL:    "/webhooks/digiflazz/" + token,
		Body:   bytes.NewReader(body),
		Headers: map[string]string{
			"X-Hub-Signature": "notsha1format",
		},
		ExpectedStatus:  http.StatusUnauthorized,
		ExpectedContent: []string{`"status":401`},
		TestAppFactory:  func(t testing.TB) *tests.TestApp { return app },
		BeforeTestFunc: func(t testing.TB, svrApp *tests.TestApp, e *core.ServeEvent) {
			bindDigiflazzWebhookRoutes(e,
				&fakeWebhookCredentialRepo{credential: &repository.DigiflazzCredentialRecord{ID: "cred1", FamilyID: "fam1", WebhookTokenHash: utils.HashString(token), WebhookSecret: "shared-secret"}},
				&fakeWebhookOrderRepo{},
				&fakeWebhookEventRepo{},
				&fakeWebhookOrderService{},
			)
		},
	}).Test(t)
}

func TestDigiflazzWebhookHandler_EmptySecretSkipsValidation(t *testing.T) {
	app := newDigiflazzWebhookTestApp(t)
	token := "webhook-token"
	body := []byte(`{"ref_id":"REF-1","status":"success"}`)

	(&tests.ApiScenario{
		Name:            "empty secret skips signature validation not 401",
		Method:          http.MethodPost,
		URL:             "/webhooks/digiflazz/" + token,
		Body:            bytes.NewReader(body),
		ExpectedStatus:  http.StatusOK,
		ExpectedContent: []string{`"status":"received"`},
		TestAppFactory:  func(t testing.TB) *tests.TestApp { return app },
		BeforeTestFunc: func(t testing.TB, svrApp *tests.TestApp, e *core.ServeEvent) {
			bindDigiflazzWebhookRoutes(e,
				&fakeWebhookCredentialRepo{credential: &repository.DigiflazzCredentialRecord{ID: "cred1", FamilyID: "fam1", WebhookTokenHash: utils.HashString(token), WebhookSecret: ""}},
				&fakeWebhookOrderRepo{},
				&fakeWebhookEventRepo{},
				&fakeWebhookOrderService{},
			)
		},
	}).Test(t)
}

func TestDigiflazzWebhookHandler_PingPayloadReturnsReceived(t *testing.T) {
	app := newDigiflazzWebhookTestApp(t)
	token := "webhook-token"
	body := []byte(`{"hook_id":"123"}`)

	(&tests.ApiScenario{
		Name:   "ping payload returns 200 received before sig check",
		Method: http.MethodPost,
		URL:    "/webhooks/digiflazz/" + token,
		Body:   bytes.NewReader(body),
		Headers: map[string]string{
			"Content-Type": "application/json",
		},
		ExpectedStatus:  http.StatusOK,
		ExpectedContent: []string{`"status":"received"`},
		TestAppFactory:  func(t testing.TB) *tests.TestApp { return app },
		BeforeTestFunc: func(t testing.TB, svrApp *tests.TestApp, e *core.ServeEvent) {
			bindDigiflazzWebhookRoutes(e,
				&fakeWebhookCredentialRepo{credential: &repository.DigiflazzCredentialRecord{ID: "cred1", FamilyID: "fam1", WebhookTokenHash: utils.HashString(token), WebhookSecret: "shared-secret"}},
				&fakeWebhookOrderRepo{},
				&fakeWebhookEventRepo{},
				&fakeWebhookOrderService{},
			)
		},
	}).Test(t)
}
