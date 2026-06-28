package handler_test

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"

	digiflazzdomain "kas/internal/domain/digiflazz"
	"kas/internal/handler"
	"kas/internal/middleware"
	"kas/internal/repository"
	"kas/internal/service"
	_ "kas/migrations"
)

type fakeProductService struct {
	searchProducts              func(familyID string, req *digiflazzdomain.ProductSearchRequest) ([]*digiflazzdomain.ProductDTO, error)
	syncPricelistWithCredential func(ctx context.Context, credential *repository.DigiflazzCredentialRecord) (*service.SyncResult, error)
	syncForFamily               func(ctx context.Context, familyID string) (*service.SyncResult, error)
	getProductBySKU             func(familyID, sku string) (*digiflazzdomain.ProductDTO, error)
}

func (f *fakeProductService) SearchProducts(familyID string, req *digiflazzdomain.ProductSearchRequest) ([]*digiflazzdomain.ProductDTO, error) {
	if f.searchProducts != nil {
		return f.searchProducts(familyID, req)
	}
	return nil, nil
}

func (f *fakeProductService) SyncPricelistWithCredential(ctx context.Context, credential *repository.DigiflazzCredentialRecord) (*service.SyncResult, error) {
	if f.syncPricelistWithCredential != nil {
		return f.syncPricelistWithCredential(ctx, credential)
	}
	return nil, errors.New("not implemented in fake")
}

func (f *fakeProductService) SyncForFamily(ctx context.Context, familyID string) (*service.SyncResult, error) {
	if f.syncForFamily != nil {
		return f.syncForFamily(ctx, familyID)
	}
	return &service.SyncResult{}, nil
}

func (f *fakeProductService) GetProductBySKU(familyID, sku string) (*digiflazzdomain.ProductDTO, error) {
	if f.getProductBySKU != nil {
		return f.getProductBySKU(familyID, sku)
	}
	return nil, nil
}

func seedDigiflazzProductTestData(t *testing.T, app *tests.TestApp) (ownerToken, memberToken, familyID, ownerID string) {
	t.Helper()

	userCol, err := app.FindCollectionByNameOrId("users")
	if err != nil {
		t.Fatalf("find users: %v", err)
	}

	owner := core.NewRecord(userCol)
	owner.Set("name", "Product Owner")
	owner.Set("email", fmt.Sprintf("prodowner+%d@example.com", time.Now().UnixNano()))
	owner.SetPassword("password12345")
	if err := app.Save(owner); err != nil {
		t.Fatalf("save owner: %v", err)
	}

	member := core.NewRecord(userCol)
	member.Set("name", "Product Member")
	member.Set("email", fmt.Sprintf("prodmember+%d@example.com", time.Now().UnixNano()))
	member.SetPassword("password12345")
	if err := app.Save(member); err != nil {
		t.Fatalf("save member: %v", err)
	}

	familyCol, err := app.FindCollectionByNameOrId("families")
	if err != nil {
		t.Fatalf("find families: %v", err)
	}
	family := core.NewRecord(familyCol)
	family.Set("name", "Product Test Family")
	family.Set("invite_code", fmt.Sprintf("PROD%d", time.Now().UnixNano()))
	if err := app.Save(family); err != nil {
		t.Fatalf("save family: %v", err)
	}

	memberCol, err := app.FindCollectionByNameOrId("family_members")
	if err != nil {
		t.Fatalf("find family_members: %v", err)
	}

	ownerMember := core.NewRecord(memberCol)
	ownerMember.Set("user_id", owner.Id)
	ownerMember.Set("family_id", family.Id)
	ownerMember.Set("role", "owner")
	if err := app.Save(ownerMember); err != nil {
		t.Fatalf("save owner member: %v", err)
	}

	regularMember := core.NewRecord(memberCol)
	regularMember.Set("user_id", member.Id)
	regularMember.Set("family_id", family.Id)
	regularMember.Set("role", "member")
	if err := app.Save(regularMember); err != nil {
		t.Fatalf("save regular member: %v", err)
	}

	oToken, err := owner.NewAuthToken()
	if err != nil {
		t.Fatalf("owner auth token: %v", err)
	}
	mToken, err := member.NewAuthToken()
	if err != nil {
		t.Fatalf("member auth token: %v", err)
	}

	return oToken, mToken, family.Id, owner.Id
}

func newDigiflazzProductTestApp(t *testing.T) *tests.TestApp {
	t.Helper()
	app, err := tests.NewTestApp()
	if err != nil {
		t.Fatalf("failed to create test app: %v", err)
	}
	t.Cleanup(app.Cleanup)
	return app
}

func bindDigiflazzProductRoutes(app *tests.TestApp, e *core.ServeEvent, svc service.DigiflazzProductService) {
	familyMemberRepo := repository.NewFamilyMemberRepository(app)
	requireFamily := middleware.RequireFamily(familyMemberRepo)
	requireFamilyOwner := middleware.RequireFamilyOwner()
	h := handler.NewDigiflazzProductHandler(svc, middleware.RequireAuth, requireFamily, requireFamilyOwner)
	h.RegisterRoutes(e)
}

func bindDigiflazzProductRoutesReal(app *tests.TestApp, e *core.ServeEvent) {
	productRepo := repository.NewDigiflazzProductRepository(app)
	credentialRepo := repository.NewDigiflazzCredentialRepository(app)
	svc := service.NewDigiflazzProductService(app, productRepo, credentialRepo, nil)
	familyMemberRepo := repository.NewFamilyMemberRepository(app)
	requireFamily := middleware.RequireFamily(familyMemberRepo)
	requireFamilyOwner := middleware.RequireFamilyOwner()
	h := handler.NewDigiflazzProductHandler(svc, middleware.RequireAuth, requireFamily, requireFamilyOwner)
	h.RegisterRoutes(e)
}

func seedProduct(t *testing.T, app *tests.TestApp, familyID, sku, name string) {
	t.Helper()
	repo := repository.NewDigiflazzProductRepository(app)
	_, err := repo.Upsert(&repository.UpsertProductInput{
		FamilyID:            familyID,
		BuyerSKUCode:        sku,
		ProductName:         name,
		Category:            "Pulsa",
		Brand:               "Telkomsel",
		Type:                "prepaid",
		Price:               10000,
		BuyerProductStatus:  "active",
		SellerProductStatus: "active",
		IsPrepaid:           true,
	})
	if err != nil {
		t.Fatalf("seed product %s: %v", sku, err)
	}
}

func TestDigiflazzProductHandler_Search(t *testing.T) {
	svc := &fakeProductService{
		searchProducts: func(familyID string, req *digiflazzdomain.ProductSearchRequest) ([]*digiflazzdomain.ProductDTO, error) {
			return []*digiflazzdomain.ProductDTO{
				{Code: "tsel10", Name: "Telkomsel 10K", Category: "Pulsa"},
			}, nil
		},
	}

	t.Run("search products - owner can access", func(t *testing.T) {
		app := newDigiflazzProductTestApp(t)
		ownerToken, _, _, _ := seedDigiflazzProductTestData(t, app)

		(&tests.ApiScenario{
			Name:   "search products - owner can access",
			Method: "GET",
			URL:    "/api/digiflazz/products",
			Headers: map[string]string{
				"Authorization": "Bearer " + ownerToken,
			},
			ExpectedStatus:  http.StatusOK,
			ExpectedContent: []string{`"items"`, `"tsel10"`, `"Telkomsel 10K"`},
			TestAppFactory:  func(t testing.TB) *tests.TestApp { return app },
			BeforeTestFunc: func(t testing.TB, svrApp *tests.TestApp, e *core.ServeEvent) {
				bindDigiflazzProductRoutes(svrApp, e, svc)
			},
		}).Test(t)
	})

	t.Run("search products - member can access", func(t *testing.T) {
		app := newDigiflazzProductTestApp(t)
		_, memberToken, _, _ := seedDigiflazzProductTestData(t, app)

		(&tests.ApiScenario{
			Name:   "search products - member can access",
			Method: "GET",
			URL:    "/api/digiflazz/products",
			Headers: map[string]string{
				"Authorization": "Bearer " + memberToken,
			},
			ExpectedStatus:  http.StatusOK,
			ExpectedContent: []string{`"items"`, `"tsel10"`, `"Telkomsel 10K"`},
			TestAppFactory:  func(t testing.TB) *tests.TestApp { return app },
			BeforeTestFunc: func(t testing.TB, svrApp *tests.TestApp, e *core.ServeEvent) {
				bindDigiflazzProductRoutes(svrApp, e, svc)
			},
		}).Test(t)
	})
}

func TestDigiflazzProductHandler_Search_WithFilters(t *testing.T) {
	app := newDigiflazzProductTestApp(t)
	ownerToken, _, _, _ := seedDigiflazzProductTestData(t, app)

	var capturedReq *digiflazzdomain.ProductSearchRequest
	svc := &fakeProductService{
		searchProducts: func(familyID string, req *digiflazzdomain.ProductSearchRequest) ([]*digiflazzdomain.ProductDTO, error) {
			capturedReq = req
			return []*digiflazzdomain.ProductDTO{}, nil
		},
	}

	scenarios := []tests.ApiScenario{
		{
			Name:   "search products - query params parsed",
			Method: "GET",
			URL:    "/api/digiflazz/products?query=tsel&category=Pulsa&brand=Telkomsel&per_page=10",
			Headers: map[string]string{
				"Authorization": "Bearer " + ownerToken,
			},
			ExpectedStatus:  http.StatusOK,
			ExpectedContent: []string{`"items"`, `"limit":10`},
			TestAppFactory:  func(t testing.TB) *tests.TestApp { return app },
			BeforeTestFunc: func(t testing.TB, svrApp *tests.TestApp, e *core.ServeEvent) {
				bindDigiflazzProductRoutes(svrApp, e, svc)
			},
			AfterTestFunc: func(t testing.TB, app *tests.TestApp, res *http.Response) {
				if capturedReq == nil {
					t.Fatal("searchProducts was not called")
				}
				if capturedReq.Query != "tsel" {
					t.Errorf("expected query=tsel, got %s", capturedReq.Query)
				}
				if capturedReq.Category != "Pulsa" {
					t.Errorf("expected category=Pulsa, got %s", capturedReq.Category)
				}
				if capturedReq.Brand != "Telkomsel" {
					t.Errorf("expected brand=Telkomsel, got %s", capturedReq.Brand)
				}
				if capturedReq.Limit != 10 {
					t.Errorf("expected limit=10, got %d", capturedReq.Limit)
				}
			},
		},
	}

	for _, scenario := range scenarios {
		scenario.Test(t)
	}
}

func TestDigiflazzProductHandler_RequiresAuth(t *testing.T) {
	svc := &fakeProductService{}

	t.Run("search products - requires auth", func(t *testing.T) {
		app := newDigiflazzProductTestApp(t)
		(&tests.ApiScenario{
			Name:            "search products - requires auth",
			Method:          "GET",
			URL:             "/api/digiflazz/products",
			ExpectedStatus:  http.StatusUnauthorized,
			ExpectedContent: []string{`"Authentication required."`, `"status":401`},
			TestAppFactory:  func(t testing.TB) *tests.TestApp { return app },
			BeforeTestFunc: func(t testing.TB, svrApp *tests.TestApp, e *core.ServeEvent) {
				bindDigiflazzProductRoutes(svrApp, e, svc)
			},
		}).Test(t)
	})
}

func TestDigiflazzProduct_FamilyIsolation_SyncA_GetAsB(t *testing.T) {
	t.Setenv("DIGIFLAZZ_CREDENTIAL_ENCRYPTION_KEY", "test-encryption-key-for-tests")

	t.Run("family A sees own products", func(t *testing.T) {
		app := newDigiflazzProductTestApp(t)
		ownerTokenA, _, familyA, _ := seedDigiflazzProductTestData(t, app)
		seedProduct(t, app, familyA, "tsel-a-001", "Telkomsel A 10K")
		seedProduct(t, app, familyA, "tsel-a-002", "Telkomsel A 20K")

		(&tests.ApiScenario{
			Name:   "family A sees own products",
			Method: "GET",
			URL:    "/api/digiflazz/products",
			Headers: map[string]string{
				"Authorization": "Bearer " + ownerTokenA,
			},
			ExpectedStatus:  http.StatusOK,
			ExpectedContent: []string{`"tsel-a-001"`, `"tsel-a-002"`},
			TestAppFactory:  func(t testing.TB) *tests.TestApp { return app },
			BeforeTestFunc: func(t testing.TB, svrApp *tests.TestApp, e *core.ServeEvent) {
				bindDigiflazzProductRoutesReal(svrApp, e)
			},
		}).Test(t)
	})

	t.Run("family B sees 0 products from family A", func(t *testing.T) {
		app := newDigiflazzProductTestApp(t)
		_, _, familyA, _ := seedDigiflazzProductTestData(t, app)
		ownerTokenB, _, _, _ := seedDigiflazzProductTestData(t, app)
		seedProduct(t, app, familyA, "tsel-a-001", "Telkomsel A 10K")
		seedProduct(t, app, familyA, "tsel-a-002", "Telkomsel A 20K")

		(&tests.ApiScenario{
			Name:   "family B sees 0 products from family A",
			Method: "GET",
			URL:    "/api/digiflazz/products",
			Headers: map[string]string{
				"Authorization": "Bearer " + ownerTokenB,
			},
			ExpectedStatus:     http.StatusOK,
			ExpectedContent:    []string{`"items":[]`},
			NotExpectedContent: []string{`"tsel-a-001"`, `"tsel-a-002"`},
			TestAppFactory:     func(t testing.TB) *tests.TestApp { return app },
			BeforeTestFunc: func(t testing.TB, svrApp *tests.TestApp, e *core.ServeEvent) {
				bindDigiflazzProductRoutesReal(svrApp, e)
			},
		}).Test(t)
	})
}

func TestDigiflazzProduct_FamilyIsolation_BothFamilies(t *testing.T) {
	t.Setenv("DIGIFLAZZ_CREDENTIAL_ENCRYPTION_KEY", "test-encryption-key-for-tests")

	t.Run("family A sees only A products", func(t *testing.T) {
		app := newDigiflazzProductTestApp(t)
		ownerTokenA, _, familyA, _ := seedDigiflazzProductTestData(t, app)
		_, _, familyB, _ := seedDigiflazzProductTestData(t, app)
		seedProduct(t, app, familyA, "sku-a-only", "Product A Only")
		seedProduct(t, app, familyB, "sku-b-only", "Product B Only")

		(&tests.ApiScenario{
			Name:   "family A sees only A products",
			Method: "GET",
			URL:    "/api/digiflazz/products",
			Headers: map[string]string{
				"Authorization": "Bearer " + ownerTokenA,
			},
			ExpectedStatus:     http.StatusOK,
			ExpectedContent:    []string{`"sku-a-only"`},
			NotExpectedContent: []string{`"sku-b-only"`},
			TestAppFactory:     func(t testing.TB) *tests.TestApp { return app },
			BeforeTestFunc: func(t testing.TB, svrApp *tests.TestApp, e *core.ServeEvent) {
				bindDigiflazzProductRoutesReal(svrApp, e)
			},
		}).Test(t)
	})

	t.Run("family B sees only B products", func(t *testing.T) {
		app := newDigiflazzProductTestApp(t)
		_, _, familyA, _ := seedDigiflazzProductTestData(t, app)
		ownerTokenB, _, familyB, _ := seedDigiflazzProductTestData(t, app)
		seedProduct(t, app, familyA, "sku-a-only", "Product A Only")
		seedProduct(t, app, familyB, "sku-b-only", "Product B Only")

		(&tests.ApiScenario{
			Name:   "family B sees only B products",
			Method: "GET",
			URL:    "/api/digiflazz/products",
			Headers: map[string]string{
				"Authorization": "Bearer " + ownerTokenB,
			},
			ExpectedStatus:     http.StatusOK,
			ExpectedContent:    []string{`"sku-b-only"`},
			NotExpectedContent: []string{`"sku-a-only"`},
			TestAppFactory:     func(t testing.TB) *tests.TestApp { return app },
			BeforeTestFunc: func(t testing.TB, svrApp *tests.TestApp, e *core.ServeEvent) {
				bindDigiflazzProductRoutesReal(svrApp, e)
			},
		}).Test(t)
	})
}

func TestDigiflazzProduct_DeleteCredentialCascades(t *testing.T) {
	app := newDigiflazzProductTestApp(t)

	_, _, familyA, _ := seedDigiflazzProductTestData(t, app)
	_, _, familyB, _ := seedDigiflazzProductTestData(t, app)

	seedProduct(t, app, familyA, "sku-cascade-a", "Cascade A Product")
	seedProduct(t, app, familyB, "sku-cascade-b", "Cascade B Product")

	productRepo := repository.NewDigiflazzProductRepository(app)
	if err := productRepo.DeleteByFamilyID(familyA); err != nil {
		t.Fatalf("DeleteByFamilyID: %v", err)
	}

	resultsA, err := productRepo.Search(familyA, &digiflazzdomain.ProductSearchRequest{Limit: 50})
	if err != nil {
		t.Fatalf("search family A after delete: %v", err)
	}
	if len(resultsA) != 0 {
		t.Errorf("expected 0 products for family A after cascade delete, got %d", len(resultsA))
	}

	resultsB, err := productRepo.Search(familyB, &digiflazzdomain.ProductSearchRequest{Limit: 50})
	if err != nil {
		t.Fatalf("search family B after delete: %v", err)
	}
	if len(resultsB) != 1 {
		t.Errorf("expected 1 product for family B (sibling unaffected), got %d", len(resultsB))
	}
}

func TestDigiflazzProduct_SearchQueryFilter(t *testing.T) {
	t.Setenv("DIGIFLAZZ_CREDENTIAL_ENCRYPTION_KEY", "test-encryption-key-for-tests")
	app := newDigiflazzProductTestApp(t)

	ownerToken, _, familyID, _ := seedDigiflazzProductTestData(t, app)

	seedProduct(t, app, familyID, "tsel-filter-001", "Telkomsel Filter 10K")
	seedProduct(t, app, familyID, "xl-filter-001", "XL Filter 10K")

	(&tests.ApiScenario{
		Name:   "query filter returns only matching products",
		Method: "GET",
		URL:    "/api/digiflazz/products?query=Telkomsel+Filter",
		Headers: map[string]string{
			"Authorization": "Bearer " + ownerToken,
		},
		ExpectedStatus:     http.StatusOK,
		ExpectedContent:    []string{`"tsel-filter-001"`},
		NotExpectedContent: []string{`"xl-filter-001"`},
		TestAppFactory:     func(t testing.TB) *tests.TestApp { return app },
		BeforeTestFunc: func(t testing.TB, svrApp *tests.TestApp, e *core.ServeEvent) {
			bindDigiflazzProductRoutesReal(svrApp, e)
		},
	}).Test(t)
}

func TestDigiflazzProductHandler_Sync(t *testing.T) {
	t.Run("owner can trigger sync", func(t *testing.T) {
		app := newDigiflazzProductTestApp(t)
		ownerToken, _, _, _ := seedDigiflazzProductTestData(t, app)

		svc := &fakeProductService{
			syncForFamily: func(ctx context.Context, familyID string) (*service.SyncResult, error) {
				return &service.SyncResult{PrepaidUpserted: 5, PostpaidUpserted: 3, TotalUpserted: 8}, nil
			},
		}

		(&tests.ApiScenario{
			Name:   "owner can trigger sync",
			Method: "POST",
			URL:    "/api/digiflazz/products/sync",
			Headers: map[string]string{
				"Authorization": "Bearer " + ownerToken,
			},
			ExpectedStatus:  http.StatusOK,
			ExpectedContent: []string{`"total_upserted":8`, `"prepaid_upserted":5`, `"postpaid_upserted":3`},
			TestAppFactory:  func(t testing.TB) *tests.TestApp { return app },
			BeforeTestFunc: func(t testing.TB, svrApp *tests.TestApp, e *core.ServeEvent) {
				bindDigiflazzProductRoutes(svrApp, e, svc)
			},
		}).Test(t)
	})

	t.Run("member cannot trigger sync", func(t *testing.T) {
		app := newDigiflazzProductTestApp(t)
		_, memberToken, _, _ := seedDigiflazzProductTestData(t, app)

		svc := &fakeProductService{}

		(&tests.ApiScenario{
			Name:   "member cannot trigger sync",
			Method: "POST",
			URL:    "/api/digiflazz/products/sync",
			Headers: map[string]string{
				"Authorization": "Bearer " + memberToken,
			},
			ExpectedStatus:  http.StatusForbidden,
			ExpectedContent: []string{`"status":403`},
			TestAppFactory:  func(t testing.TB) *tests.TestApp { return app },
			BeforeTestFunc: func(t testing.TB, svrApp *tests.TestApp, e *core.ServeEvent) {
				bindDigiflazzProductRoutes(svrApp, e, svc)
			},
		}).Test(t)
	})

	t.Run("unauthenticated cannot trigger sync", func(t *testing.T) {
		app := newDigiflazzProductTestApp(t)

		svc := &fakeProductService{}

		(&tests.ApiScenario{
			Name:            "unauthenticated cannot trigger sync",
			Method:          "POST",
			URL:             "/api/digiflazz/products/sync",
			ExpectedStatus:  http.StatusUnauthorized,
			ExpectedContent: []string{`"status":401`},
			TestAppFactory:  func(t testing.TB) *tests.TestApp { return app },
			BeforeTestFunc: func(t testing.TB, svrApp *tests.TestApp, e *core.ServeEvent) {
				bindDigiflazzProductRoutes(svrApp, e, svc)
			},
		}).Test(t)
	})
}
