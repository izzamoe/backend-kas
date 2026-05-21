package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"testing"

	digiflazzdomain "kas/internal/domain/digiflazz"
	"kas/internal/handler"
	"kas/internal/middleware"
	"kas/internal/repository"
	"kas/internal/service"
	_ "kas/migrations"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
)

type fakeOrderService struct {
	createOrder           func(ctx context.Context, familyID, createdBy string, req digiflazzdomain.CreateOrderRequest) (*digiflazzdomain.OrderDTO, error)
	createPrepaidOrder    func(ctx context.Context, req *digiflazzdomain.CreateOrderRequest, userID, familyID string) (*digiflazzdomain.OrderDTO, error)
	createPostpaidInquiry func(ctx context.Context, req *digiflazzdomain.CreateOrderRequest, userID, familyID string) (*digiflazzdomain.OrderDTO, error)
	payPostpaidOrder      func(ctx context.Context, familyID, userID, orderID string) (*digiflazzdomain.OrderDTO, error)
	checkPostpaidStatus   func(ctx context.Context, familyID, userID, orderID string) (*digiflazzdomain.OrderDTO, error)
	getOrder              func(familyID, id string) (*digiflazzdomain.OrderDTO, error)
	listFamilyOrders      func(familyID string, page, pageSize int) ([]*digiflazzdomain.OrderDTO, error)
}

var _ service.DigiflazzOrderService = (*fakeOrderService)(nil)

func (f *fakeOrderService) CreateOrder(ctx context.Context, familyID, createdBy string, req digiflazzdomain.CreateOrderRequest) (*digiflazzdomain.OrderDTO, error) {
	if f.createOrder != nil {
		return f.createOrder(ctx, familyID, createdBy, req)
	}
	return nil, fmt.Errorf("not implemented")
}
func (f *fakeOrderService) CreatePrepaidOrder(ctx context.Context, req *digiflazzdomain.CreateOrderRequest, userID, familyID string) (*digiflazzdomain.OrderDTO, error) {
	if f.createPrepaidOrder != nil {
		return f.createPrepaidOrder(ctx, req, userID, familyID)
	}
	return nil, fmt.Errorf("not implemented")
}
func (f *fakeOrderService) CreatePostpaidInquiry(ctx context.Context, req *digiflazzdomain.CreateOrderRequest, userID, familyID string) (*digiflazzdomain.OrderDTO, error) {
	if f.createPostpaidInquiry != nil {
		return f.createPostpaidInquiry(ctx, req, userID, familyID)
	}
	return nil, fmt.Errorf("create inquiry failed")
}
func (f *fakeOrderService) PayPostpaidOrder(ctx context.Context, familyID, userID, orderID string) (*digiflazzdomain.OrderDTO, error) {
	if f.payPostpaidOrder != nil {
		return f.payPostpaidOrder(ctx, familyID, userID, orderID)
	}
	return nil, fmt.Errorf("pay failed")
}
func (f *fakeOrderService) CheckPostpaidStatus(ctx context.Context, familyID, userID, orderID string) (*digiflazzdomain.OrderDTO, error) {
	if f.checkPostpaidStatus != nil {
		return f.checkPostpaidStatus(ctx, familyID, userID, orderID)
	}
	return nil, fmt.Errorf("status failed")
}
func (f *fakeOrderService) GetOrder(familyID, id string) (*digiflazzdomain.OrderDTO, error) {
	if f.getOrder != nil {
		return f.getOrder(familyID, id)
	}
	return nil, nil
}
func (f *fakeOrderService) ListFamilyOrders(familyID string, page, pageSize int) ([]*digiflazzdomain.OrderDTO, error) {
	if f.listFamilyOrders != nil {
		return f.listFamilyOrders(familyID, page, pageSize)
	}
	return nil, nil
}
func (f *fakeOrderService) UpdateStatus(string, string, digiflazzdomain.OrderStatus, *digiflazzdomain.OrderResponseDTO) (*digiflazzdomain.OrderDTO, error) {
	return nil, nil
}
func (f *fakeOrderService) FinalizeSuccessOrder(string) (*digiflazzdomain.OrderDTO, error) {
	return nil, nil
}
func (f *fakeOrderService) CheckAndUpdateStatus(ctx context.Context, orderID string) (*digiflazzdomain.OrderDTO, error) {
	return nil, nil
}

func bindDigiflazzOrderRoutes(app *tests.TestApp, e *core.ServeEvent, svc service.DigiflazzOrderService) {
	familyMemberRepo := repository.NewFamilyMemberRepository(app)
	requireFamily := middleware.RequireFamily(familyMemberRepo)
	h := handler.NewDigiflazzOrderHandler(svc, middleware.RequireAuth, requireFamily)
	h.RegisterRoutes(e)
}

func TestDigiflazzPrepaidOrderHandler_CreateAllowsFamilyMember(t *testing.T) {
	app := newDigiflazzProductTestApp(t)
	_, memberToken, familyID, _ := seedDigiflazzProductTestData(t, app)
	body, _ := json.Marshal(map[string]any{
		"buyer_sku_code": "PLN20",
		"customer_no":    "08123456789",
	})

	var capturedFamilyID, capturedCreatedBy string
	var capturedReq digiflazzdomain.CreateOrderRequest
	svc := &fakeOrderService{
		createOrder: func(ctx context.Context, fID, createdBy string, req digiflazzdomain.CreateOrderRequest) (*digiflazzdomain.OrderDTO, error) {
			capturedFamilyID = fID
			capturedCreatedBy = createdBy
			capturedReq = req
			if fID != familyID {
				return nil, fmt.Errorf("unexpected family context")
			}
			return &digiflazzdomain.OrderDTO{
				ID:          "order1",
				FamilyID:    fID,
				Status:      digiflazzdomain.OrderStatusProcessing,
				ProductCode: req.BuyerSKUCode,
				CustomerNo:  req.CustomerNo,
			}, nil
		},
	}

	(&tests.ApiScenario{
		Name:   "create order - member succeeds",
		Method: http.MethodPost,
		URL:    "/api/digiflazz/orders",
		Body:   bytes.NewReader(body),
		Headers: map[string]string{
			"Authorization": "Bearer " + memberToken,
			"Content-Type":  "application/json",
		},
		ExpectedStatus:  http.StatusCreated,
		ExpectedContent: []string{`"id":"order1"`, `"status":"processing"`, `"product_code":"PLN20"`},
		TestAppFactory:  func(t testing.TB) *tests.TestApp { return app },
		BeforeTestFunc: func(t testing.TB, svrApp *tests.TestApp, e *core.ServeEvent) {
			bindDigiflazzOrderRoutes(svrApp, e, svc)
		},
		AfterTestFunc: func(t testing.TB, app *tests.TestApp, res *http.Response) {
			if capturedFamilyID != familyID {
				t.Errorf("expected familyID=%s, got %s", familyID, capturedFamilyID)
			}
			if capturedCreatedBy == "" {
				t.Error("expected createdBy to be set")
			}
			if capturedReq.CustomerNo != "08123456789" {
				t.Errorf("unexpected customer_no: %s", capturedReq.CustomerNo)
			}
		},
	}).Test(t)
}

func TestDigiflazzPrepaidOrderHandler_RequiresAuth(t *testing.T) {
	app := newDigiflazzProductTestApp(t)
	svc := &fakeOrderService{}

	(&tests.ApiScenario{
		Name:            "create order - requires auth",
		Method:          http.MethodPost,
		URL:             "/api/digiflazz/orders",
		Body:            bytes.NewReader([]byte(`{"buyer_sku_code":"PLN20","customer_no":"0812"}`)),
		ExpectedStatus:  http.StatusUnauthorized,
		ExpectedContent: []string{`"Authentication required."`, `"status":401`},
		TestAppFactory:  func(t testing.TB) *tests.TestApp { return app },
		BeforeTestFunc: func(t testing.TB, svrApp *tests.TestApp, e *core.ServeEvent) {
			bindDigiflazzOrderRoutes(svrApp, e, svc)
		},
	}).Test(t)
}

func TestDigiflazzPrepaidOrderHandler_InsufficientBalanceMapsBadRequest(t *testing.T) {
	app := newDigiflazzProductTestApp(t)
	ownerToken, _, _, _ := seedDigiflazzProductTestData(t, app)
	svc := &fakeOrderService{
		createOrder: func(ctx context.Context, familyID, createdBy string, req digiflazzdomain.CreateOrderRequest) (*digiflazzdomain.OrderDTO, error) {
			return nil, errors.New("Digiflazz balance is insufficient")
		},
	}

	(&tests.ApiScenario{
		Name:   "create order - insufficient balance",
		Method: http.MethodPost,
		URL:    "/api/digiflazz/orders",
		Body:   bytes.NewReader([]byte(`{"buyer_sku_code":"PLN20","customer_no":"0812"}`)),
		Headers: map[string]string{
			"Authorization": "Bearer " + ownerToken,
			"Content-Type":  "application/json",
		},
		ExpectedStatus:  http.StatusBadRequest,
		ExpectedContent: []string{`"status":400`, `insufficient`},
		TestAppFactory:  func(t testing.TB) *tests.TestApp { return app },
		BeforeTestFunc: func(t testing.TB, svrApp *tests.TestApp, e *core.ServeEvent) {
			bindDigiflazzOrderRoutes(svrApp, e, svc)
		},
	}).Test(t)
}

func TestDigiflazzPascaOrderHandler_CreatePostpaidInquiry(t *testing.T) {
	app := newDigiflazzProductTestApp(t)
	token, _, familyID, userID := seedDigiflazzProductTestData(t, app)

	var capturedFamilyID, capturedCreatedBy string
	var capturedReq digiflazzdomain.CreateOrderRequest
	svc := &fakeOrderService{
		createOrder: func(ctx context.Context, fID, createdBy string, req digiflazzdomain.CreateOrderRequest) (*digiflazzdomain.OrderDTO, error) {
			capturedFamilyID = fID
			capturedCreatedBy = createdBy
			capturedReq = req
			if fID != familyID || createdBy != userID {
				return nil, fmt.Errorf("unexpected identity")
			}
			return &digiflazzdomain.OrderDTO{
				ID:          "order1",
				FamilyID:    fID,
				Status:      digiflazzdomain.OrderStatusInquiry,
				ProductCode: req.BuyerSKUCode,
			}, nil
		},
	}
	body, _ := json.Marshal(map[string]any{"buyer_sku_code": "PLNPOST", "customer_no": "12345"})

	(&tests.ApiScenario{
		Name:   "create postpaid inquiry",
		Method: http.MethodPost,
		URL:    "/api/digiflazz/orders",
		Body:   bytes.NewReader(body),
		Headers: map[string]string{
			"Authorization": "Bearer " + token,
			"Content-Type":  "application/json",
		},
		ExpectedStatus:  http.StatusCreated,
		ExpectedContent: []string{`"id":"order1"`, `"status":"inquiry"`, `"product_code":"PLNPOST"`},
		TestAppFactory:  func(t testing.TB) *tests.TestApp { return app },
		BeforeTestFunc: func(t testing.TB, svrApp *tests.TestApp, e *core.ServeEvent) {
			bindDigiflazzOrderRoutes(svrApp, e, svc)
		},
		AfterTestFunc: func(t testing.TB, app *tests.TestApp, res *http.Response) {
			if capturedFamilyID != familyID {
				t.Errorf("expected familyID=%s, got %s", familyID, capturedFamilyID)
			}
			if capturedCreatedBy != userID {
				t.Errorf("expected createdBy=%s, got %s", userID, capturedCreatedBy)
			}
			if capturedReq.BuyerSKUCode != "PLNPOST" {
				t.Errorf("unexpected buyer_sku_code: %s", capturedReq.BuyerSKUCode)
			}
		},
	}).Test(t)
}

func TestDigiflazzPascaOrderHandler_PayAndCheckStatus(t *testing.T) {
	t.Run("pay postpaid order", func(t *testing.T) {
		app := newDigiflazzProductTestApp(t)
		token, _, familyID, userID := seedDigiflazzProductTestData(t, app)
		svc := &fakeOrderService{
			payPostpaidOrder: func(ctx context.Context, fID, uID, orderID string) (*digiflazzdomain.OrderDTO, error) {
				if fID != familyID || uID != userID || orderID != "order1" {
					return nil, fmt.Errorf("unexpected pay args")
				}
				return &digiflazzdomain.OrderDTO{ID: orderID, FamilyID: fID, Status: digiflazzdomain.OrderStatusSuccess}, nil
			},
		}
		(&tests.ApiScenario{
			Name:            "pay postpaid order",
			Method:          http.MethodPost,
			URL:             "/api/digiflazz/orders/order1/pay",
			Headers:         map[string]string{"Authorization": "Bearer " + token},
			ExpectedStatus:  http.StatusOK,
			ExpectedContent: []string{`"id":"order1"`, `"status":"success"`},
			TestAppFactory:  func(t testing.TB) *tests.TestApp { return app },
			BeforeTestFunc: func(t testing.TB, svrApp *tests.TestApp, e *core.ServeEvent) {
				bindDigiflazzOrderRoutes(svrApp, e, svc)
			},
		}).Test(t)
	})

	t.Run("check-status endpoint removed", func(t *testing.T) {
		app := newDigiflazzProductTestApp(t)
		token, _, _, _ := seedDigiflazzProductTestData(t, app)
		svc := &fakeOrderService{}
		(&tests.ApiScenario{
			Name:           "check-status route removed - 404",
			Method:         http.MethodPost,
			URL:            "/api/digiflazz/orders/order1/check-status",
			Headers:        map[string]string{"Authorization": "Bearer " + token},
			ExpectedStatus: http.StatusNotFound,
			TestAppFactory: func(t testing.TB) *tests.TestApp { return app },
			BeforeTestFunc: func(t testing.TB, svrApp *tests.TestApp, e *core.ServeEvent) {
				bindDigiflazzOrderRoutes(svrApp, e, svc)
			},
		}).Test(t)
	})
}

func TestDigiflazzOrderHandler_ListFamilyOrders(t *testing.T) {
	app := newDigiflazzProductTestApp(t)
	_, memberToken, familyID, _ := seedDigiflazzProductTestData(t, app)

	svc := &fakeOrderService{
		listFamilyOrders: func(fID string, page, pageSize int) ([]*digiflazzdomain.OrderDTO, error) {
			if fID != familyID {
				return nil, fmt.Errorf("unexpected family id: %s", fID)
			}
			return []*digiflazzdomain.OrderDTO{
				{ID: "order1", FamilyID: fID, Status: digiflazzdomain.OrderStatusSuccess, ProductCode: "PLN20"},
				{ID: "order2", FamilyID: fID, Status: digiflazzdomain.OrderStatusProcessing, ProductCode: "TSEL10"},
			}, nil
		},
	}

	(&tests.ApiScenario{
		Name:            "list family orders - member succeeds",
		Method:          http.MethodGet,
		URL:             "/api/digiflazz/orders?page=1&page_size=10",
		Headers:         map[string]string{"Authorization": "Bearer " + memberToken},
		ExpectedStatus:  http.StatusOK,
		ExpectedContent: []string{`"items"`, `"order1"`, `"order2"`, `"page":1`, `"page_size":10`},
		TestAppFactory:  func(t testing.TB) *tests.TestApp { return app },
		BeforeTestFunc: func(t testing.TB, svrApp *tests.TestApp, e *core.ServeEvent) {
			bindDigiflazzOrderRoutes(svrApp, e, svc)
		},
	}).Test(t)
}

func TestDigiflazzOrderHandler_GetOrder(t *testing.T) {
	app := newDigiflazzProductTestApp(t)
	_, memberToken, familyID, _ := seedDigiflazzProductTestData(t, app)

	svc := &fakeOrderService{
		getOrder: func(fID, id string) (*digiflazzdomain.OrderDTO, error) {
			if fID != familyID || id != "order1" {
				return nil, nil
			}
			return &digiflazzdomain.OrderDTO{
				ID: "order1", FamilyID: fID, Status: digiflazzdomain.OrderStatusSuccess,
				ProductCode: "PLN20", CustomerNo: "12345",
			}, nil
		},
	}

	(&tests.ApiScenario{
		Name:            "get single order - member succeeds",
		Method:          http.MethodGet,
		URL:             "/api/digiflazz/orders/order1",
		Headers:         map[string]string{"Authorization": "Bearer " + memberToken},
		ExpectedStatus:  http.StatusOK,
		ExpectedContent: []string{`"id":"order1"`, `"status":"success"`, `"product_code":"PLN20"`, `"customer_no":"12345"`},
		TestAppFactory:  func(t testing.TB) *tests.TestApp { return app },
		BeforeTestFunc: func(t testing.TB, svrApp *tests.TestApp, e *core.ServeEvent) {
			bindDigiflazzOrderRoutes(svrApp, e, svc)
		},
	}).Test(t)
}

func TestDigiflazzOrderHandler_CrossFamilyDenied(t *testing.T) {
	app := newDigiflazzProductTestApp(t)
	_, _, family1ID, _ := seedDigiflazzProductTestData(t, app)
	ownerToken2, _, _, _ := seedDigiflazzProductTestData(t, app)

	svc := &fakeOrderService{
		getOrder: func(fID, id string) (*digiflazzdomain.OrderDTO, error) {
			if fID == family1ID {
				return &digiflazzdomain.OrderDTO{ID: id, FamilyID: family1ID, Status: digiflazzdomain.OrderStatusSuccess}, nil
			}
			return nil, nil
		},
	}

	(&tests.ApiScenario{
		Name:            "cross-family order access denied",
		Method:          http.MethodGet,
		URL:             "/api/digiflazz/orders/order1",
		Headers:         map[string]string{"Authorization": "Bearer " + ownerToken2},
		ExpectedStatus:  http.StatusNotFound,
		ExpectedContent: []string{`"status":404`},
		TestAppFactory:  func(t testing.TB) *tests.TestApp { return app },
		BeforeTestFunc: func(t testing.TB, svrApp *tests.TestApp, e *core.ServeEvent) {
			bindDigiflazzOrderRoutes(svrApp, e, svc)
		},
	}).Test(t)
}

func TestDigiflazzPascaOrderHandler_AmountChangeMapsToBadRequest(t *testing.T) {
	app := newDigiflazzProductTestApp(t)
	token, _, _, _ := seedDigiflazzProductTestData(t, app)
	svc := &fakeOrderService{
		payPostpaidOrder: func(ctx context.Context, familyID, userID, orderID string) (*digiflazzdomain.OrderDTO, error) {
			return nil, fmt.Errorf("postpaid amount changed since inquiry; please create a fresh inquiry")
		},
	}

	(&tests.ApiScenario{
		Name:            "amount changed maps to bad request",
		Method:          http.MethodPost,
		URL:             "/api/digiflazz/orders/order1/pay",
		Headers:         map[string]string{"Authorization": "Bearer " + token},
		ExpectedStatus:  http.StatusBadRequest,
		ExpectedContent: []string{`fresh inquiry`, `"status":400`},
		TestAppFactory:  func(t testing.TB) *tests.TestApp { return app },
		BeforeTestFunc: func(t testing.TB, svrApp *tests.TestApp, e *core.ServeEvent) {
			bindDigiflazzOrderRoutes(svrApp, e, svc)
		},
	}).Test(t)
}
