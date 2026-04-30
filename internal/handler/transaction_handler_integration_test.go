package handler_test

import (
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
	"kas/internal/handler"
	"kas/internal/middleware"
	"kas/internal/repository"
	"kas/internal/service"
	_ "kas/migrations"
)

func newTransactionTestApp(t testing.TB) *tests.TestApp {
	app, err := tests.NewTestApp()
	if err != nil {
		t.Fatalf("failed to create test app: %v", err)
	}
	return app
}

func bindTransactionRoutes(app *tests.TestApp, e *core.ServeEvent) {
	transactionRepo := repository.NewTransactionRepository(app)
	familyMemberRepo := repository.NewFamilyMemberRepository(app)
	categoryRepo := repository.NewCategoryRepository(app)
	requireFamily := middleware.RequireFamily(familyMemberRepo)

	transactionService := service.NewTransactionService(transactionRepo, categoryRepo)
	transactionHandler := handler.NewTransactionHandler(transactionService, middleware.RequireAuth, requireFamily)
	transactionHandler.RegisterRoutes(e)
}

func seedTransactionTestData(t testing.TB, app *tests.TestApp) (string, string, string, string) {
	t.Helper()

	userCol, err := app.FindCollectionByNameOrId("users")
	if err != nil {
		t.Fatalf("failed to find users collection: %v", err)
	}
	user := core.NewRecord(userCol)
	user.Set("name", "Test User")
	user.Set("email", fmt.Sprintf("testuser+%d@example.com", time.Now().UnixNano()))
	user.SetPassword("password1234")
	if err := app.Save(user); err != nil {
		t.Fatalf("failed to save user: %v", err)
	}

	familyCol, err := app.FindCollectionByNameOrId("families")
	if err != nil {
		t.Fatalf("failed to find families collection: %v", err)
	}
	family := core.NewRecord(familyCol)
	family.Set("name", "Test Family")
	family.Set("invite_code", fmt.Sprintf("INV%d", time.Now().UnixNano()))
	if err := app.Save(family); err != nil {
		t.Fatalf("failed to save family: %v", err)
	}

	catCol, err := app.FindCollectionByNameOrId("categories")
	if err != nil {
		t.Fatalf("failed to find categories collection: %v", err)
	}
	category := core.NewRecord(catCol)
	category.Set("name", "Food")
	category.Set("family_id", family.Id)
	category.Set("is_default", false)
	if err := app.Save(category); err != nil {
		t.Fatalf("failed to save category: %v", err)
	}

	memberCol, err := app.FindCollectionByNameOrId("family_members")
	if err != nil {
		t.Fatalf("failed to find family_members collection: %v", err)
	}
	member := core.NewRecord(memberCol)
	member.Set("user_id", user.Id)
	member.Set("family_id", family.Id)
	member.Set("role", "owner")
	if err := app.Save(member); err != nil {
		t.Fatalf("failed to save family member: %v", err)
	}

	txCol, err := app.FindCollectionByNameOrId("transactions")
	if err != nil {
		t.Fatalf("failed to find transactions collection: %v", err)
	}
	tx := core.NewRecord(txCol)
	tx.Set("family_id", family.Id)
	tx.Set("created_by", user.Id)
	tx.Set("category_id", category.Id)
	tx.Set("type", "expense")
	tx.Set("amount", 50000)
	tx.Set("note", "test")
	tx.Set("date", "2026-01-15T00:00:00Z")
	if err := app.Save(tx); err != nil {
		t.Fatalf("failed to save transaction: %v", err)
	}

	token, err := user.NewAuthToken()
	if err != nil {
		t.Fatalf("failed to generate auth token: %v", err)
	}

	return token, family.Id, category.Id, tx.Id
}

func seedTransactionRecord(t testing.TB, app *tests.TestApp, familyID, userID, categoryID, note, date string) string {
	t.Helper()

	txCol, err := app.FindCollectionByNameOrId("transactions")
	if err != nil {
		t.Fatalf("failed to find transactions collection: %v", err)
	}
	tx := core.NewRecord(txCol)
	tx.Set("family_id", familyID)
	tx.Set("created_by", userID)
	tx.Set("category_id", categoryID)
	tx.Set("type", "expense")
	tx.Set("amount", 10000)
	tx.Set("note", note)
	tx.Set("date", date)
	if err := app.Save(tx); err != nil {
		t.Fatalf("failed to save transaction: %v", err)
	}

	return tx.Id
}

func seedOtherFamilyTransaction(t testing.TB, app *tests.TestApp, userID string) {
	t.Helper()

	family := createRecordForTransactionTest(t, app, "families", map[string]any{
		"name":        "Other Family",
		"invite_code": fmt.Sprintf("OTH%d", time.Now().UnixNano()),
	})
	category := createRecordForTransactionTest(t, app, "categories", map[string]any{
		"family_id":  family.Id,
		"name":       "Other Food",
		"is_default": false,
	})
	seedTransactionRecord(t, app, family.Id, userID, category.Id, "other family may", "2026-05-15T12:00:00Z")
}

func createRecordForTransactionTest(t testing.TB, app *tests.TestApp, collectionName string, values map[string]any) *core.Record {
	t.Helper()

	collection, err := app.FindCollectionByNameOrId(collectionName)
	if err != nil {
		t.Fatalf("failed to find %s collection: %v", collectionName, err)
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

func TestTransactionHandler(t *testing.T) {
	t.Run("POST guest returns 401", func(t *testing.T) {
		(&tests.ApiScenario{
			Name:            "guest POST /api/transactions returns 401",
			Method:          http.MethodPost,
			URL:             "/api/transactions",
			ExpectedStatus:  http.StatusUnauthorized,
			ExpectedContent: []string{`"message"`},
			TestAppFactory: func(t testing.TB) *tests.TestApp {
				return newTransactionTestApp(t)
			},
			BeforeTestFunc: func(t testing.TB, app *tests.TestApp, e *core.ServeEvent) {
				bindTransactionRoutes(app, e)
			},
		}).Test(t)
	})

	t.Run("POST auth but no family returns 403", func(t *testing.T) {
		app := newTransactionTestApp(t)
		defer app.Cleanup()

		userCol, err := app.FindCollectionByNameOrId("users")
		if err != nil {
			t.Fatalf("failed to find users collection: %v", err)
		}
		user := core.NewRecord(userCol)
		user.Set("name", "No Family User")
		user.Set("email", fmt.Sprintf("nofamily+%d@example.com", time.Now().UnixNano()))
		user.SetPassword("password1234")
		if err := app.Save(user); err != nil {
			t.Fatalf("failed to save user: %v", err)
		}
		token, err := user.NewAuthToken()
		if err != nil {
			t.Fatalf("failed to generate auth token: %v", err)
		}

		(&tests.ApiScenario{
			Name:   "auth no family POST /api/transactions returns 403",
			Method: http.MethodPost,
			URL:    "/api/transactions",
			Headers: map[string]string{
				"Authorization": "Bearer " + token,
				"Content-Type":  "application/json",
			},
			Body:            strings.NewReader(`{"type":"expense","amount":1000}`),
			ExpectedStatus:  http.StatusForbidden,
			ExpectedContent: []string{`"message"`},
			TestAppFactory: func(t testing.TB) *tests.TestApp {
				return app
			},
			DisableTestAppCleanup: true,
			BeforeTestFunc: func(t testing.TB, app *tests.TestApp, e *core.ServeEvent) {
				bindTransactionRoutes(app, e)
			},
		}).Test(t)
	})

	t.Run("POST valid create returns 201", func(t *testing.T) {
		app := newTransactionTestApp(t)
		defer app.Cleanup()

		token, famID, catID, _ := seedTransactionTestData(t, app)
		body := fmt.Sprintf(
			`{"family_id":%q,"category_id":%q,"type":"expense","amount":10000,"note":"test","date":"2026-01-01T00:00:00Z"}`,
			famID, catID,
		)

		(&tests.ApiScenario{
			Name:   "valid POST /api/transactions returns 201",
			Method: http.MethodPost,
			URL:    "/api/transactions",
			Headers: map[string]string{
				"Content-Type":  "application/json",
				"Authorization": "Bearer " + token,
			},
			Body:            strings.NewReader(body),
			ExpectedStatus:  http.StatusCreated,
			ExpectedContent: []string{`"id"`},
			TestAppFactory: func(t testing.TB) *tests.TestApp {
				return app
			},
			DisableTestAppCleanup: true,
			BeforeTestFunc: func(t testing.TB, app *tests.TestApp, e *core.ServeEvent) {
				bindTransactionRoutes(app, e)
			},
		}).Test(t)
	})

	t.Run("POST invalid JSON returns 400", func(t *testing.T) {
		app := newTransactionTestApp(t)
		defer app.Cleanup()
		token, _, _, _ := seedTransactionTestData(t, app)

		(&tests.ApiScenario{
			Name:   "POST /api/transactions invalid JSON returns 400",
			Method: http.MethodPost,
			URL:    "/api/transactions",
			Headers: map[string]string{
				"Content-Type":  "application/json",
				"Authorization": "Bearer " + token,
			},
			Body:            strings.NewReader(`{"type":`),
			ExpectedStatus:  http.StatusBadRequest,
			ExpectedContent: []string{`"message"`},
			TestAppFactory: func(t testing.TB) *tests.TestApp {
				return app
			},
			DisableTestAppCleanup: true,
			BeforeTestFunc: func(t testing.TB, app *tests.TestApp, e *core.ServeEvent) {
				bindTransactionRoutes(app, e)
			},
		}).Test(t)
	})

	t.Run("GET by ID guest returns 401", func(t *testing.T) {
		(&tests.ApiScenario{
			Name:            "guest GET /api/transactions/:id returns 401",
			Method:          http.MethodGet,
			URL:             "/api/transactions/nonexistentid",
			ExpectedStatus:  http.StatusUnauthorized,
			ExpectedContent: []string{`"message"`},
			TestAppFactory: func(t testing.TB) *tests.TestApp {
				return newTransactionTestApp(t)
			},
			BeforeTestFunc: func(t testing.TB, app *tests.TestApp, e *core.ServeEvent) {
				bindTransactionRoutes(app, e)
			},
		}).Test(t)
	})

	t.Run("GET by ID not found returns 404", func(t *testing.T) {
		app := newTransactionTestApp(t)
		defer app.Cleanup()

		token, _, _, _ := seedTransactionTestData(t, app)

		(&tests.ApiScenario{
			Name:   "GET /api/transactions/:id not found returns 404",
			Method: http.MethodGet,
			URL:    "/api/transactions/doesnotexist000",
			Headers: map[string]string{
				"Authorization": "Bearer " + token,
			},
			ExpectedStatus:  http.StatusNotFound,
			ExpectedContent: []string{`"message"`},
			TestAppFactory: func(t testing.TB) *tests.TestApp {
				return app
			},
			DisableTestAppCleanup: true,
			BeforeTestFunc: func(t testing.TB, app *tests.TestApp, e *core.ServeEvent) {
				bindTransactionRoutes(app, e)
			},
		}).Test(t)
	})

	t.Run("GET by ID valid returns 200", func(t *testing.T) {
		app := newTransactionTestApp(t)
		defer app.Cleanup()

		token, _, _, txID := seedTransactionTestData(t, app)

		(&tests.ApiScenario{
			Name:   "valid GET /api/transactions/:id returns 200",
			Method: http.MethodGet,
			URL:    "/api/transactions/" + txID,
			Headers: map[string]string{
				"Authorization": "Bearer " + token,
			},
			ExpectedStatus:  http.StatusOK,
			ExpectedContent: []string{`"id"`},
			TestAppFactory: func(t testing.TB) *tests.TestApp {
				return app
			},
			DisableTestAppCleanup: true,
			BeforeTestFunc: func(t testing.TB, app *tests.TestApp, e *core.ServeEvent) {
				bindTransactionRoutes(app, e)
			},
		}).Test(t)
	})

	t.Run("GET family transactions guest returns 401", func(t *testing.T) {
		(&tests.ApiScenario{
			Name:            "guest GET /api/families/:familyId/transactions returns 401",
			Method:          http.MethodGet,
			URL:             "/api/families/somefamilyid/transactions",
			ExpectedStatus:  http.StatusUnauthorized,
			ExpectedContent: []string{`"message"`},
			TestAppFactory: func(t testing.TB) *tests.TestApp {
				return newTransactionTestApp(t)
			},
			BeforeTestFunc: func(t testing.TB, app *tests.TestApp, e *core.ServeEvent) {
				bindTransactionRoutes(app, e)
			},
		}).Test(t)
	})

	t.Run("GET family transactions valid returns 200", func(t *testing.T) {
		app := newTransactionTestApp(t)
		defer app.Cleanup()

		token, famID, _, _ := seedTransactionTestData(t, app)

		(&tests.ApiScenario{
			Name:   "valid GET /api/families/:familyId/transactions returns 200",
			Method: http.MethodGet,
			URL:    "/api/families/" + famID + "/transactions",
			Headers: map[string]string{
				"Authorization": "Bearer " + token,
			},
			ExpectedStatus:  http.StatusOK,
			ExpectedContent: []string{`"items"`, `"page"`, `"pageSize"`},
			TestAppFactory: func(t testing.TB) *tests.TestApp {
				return app
			},
			DisableTestAppCleanup: true,
			BeforeTestFunc: func(t testing.TB, app *tests.TestApp, e *core.ServeEvent) {
				bindTransactionRoutes(app, e)
			},
		}).Test(t)
	})

	t.Run("GET transactions guest returns 401", func(t *testing.T) {
		(&tests.ApiScenario{
			Name:            "guest GET /api/transactions returns 401",
			Method:          http.MethodGet,
			URL:             "/api/transactions?start=2026-05-01&end=2026-05-31",
			ExpectedStatus:  http.StatusUnauthorized,
			ExpectedContent: []string{`"message"`},
			TestAppFactory: func(t testing.TB) *tests.TestApp {
				return newTransactionTestApp(t)
			},
			BeforeTestFunc: func(t testing.TB, app *tests.TestApp, e *core.ServeEvent) {
				bindTransactionRoutes(app, e)
			},
		}).Test(t)
	})

	t.Run("GET transactions missing start returns 400", func(t *testing.T) {
		app := newTransactionTestApp(t)
		defer app.Cleanup()

		token, _, _, _ := seedTransactionTestData(t, app)

		(&tests.ApiScenario{
			Name:   "GET /api/transactions missing start returns 400",
			Method: http.MethodGet,
			URL:    "/api/transactions?end=2026-05-31",
			Headers: map[string]string{
				"Authorization": "Bearer " + token,
			},
			ExpectedStatus:  http.StatusBadRequest,
			ExpectedContent: []string{`"message"`},
			TestAppFactory: func(t testing.TB) *tests.TestApp {
				return app
			},
			DisableTestAppCleanup: true,
			BeforeTestFunc: func(t testing.TB, app *tests.TestApp, e *core.ServeEvent) {
				bindTransactionRoutes(app, e)
			},
		}).Test(t)
	})

	t.Run("GET transactions uses auth family date range and pagination", func(t *testing.T) {
		app := newTransactionTestApp(t)
		defer app.Cleanup()

		token, famID, catID, txID := seedTransactionTestData(t, app)
		seededTx, err := app.FindRecordById("transactions", txID)
		if err != nil {
			t.Fatalf("failed to find seeded transaction: %v", err)
		}
		userID := seededTx.GetString("created_by")

		seedTransactionRecord(t, app, famID, userID, catID, "may first", "2026-05-01T08:00:00Z")
		seedTransactionRecord(t, app, famID, userID, catID, "may last", "2026-05-31T23:59:59Z")
		seedTransactionRecord(t, app, famID, userID, catID, "june boundary", "2026-06-01T00:00:00Z")
		seedOtherFamilyTransaction(t, app, userID)

		(&tests.ApiScenario{
			Name:   "GET /api/transactions date range returns paginated auth family results",
			Method: http.MethodGet,
			URL:    "/api/transactions?start=2026-05-01&end=2026-05-31&page=1&perPage=1&family_id=spoofed",
			Headers: map[string]string{
				"Authorization": "Bearer " + token,
			},
			ExpectedStatus: http.StatusOK,
			ExpectedContent: []string{
				`"items"`,
				`"page":1`,
				`"perPage":1`,
				`"totalItems":2`,
				`"totalPages":2`,
				`"category"`,
				`"creator"`,
			},
			TestAppFactory: func(t testing.TB) *tests.TestApp {
				return app
			},
			DisableTestAppCleanup: true,
			BeforeTestFunc: func(t testing.TB, app *tests.TestApp, e *core.ServeEvent) {
				bindTransactionRoutes(app, e)
			},
		}).Test(t)
	})

	t.Run("PATCH guest returns 401", func(t *testing.T) {
		(&tests.ApiScenario{
			Name:            "guest PATCH /api/transactions/:id returns 401",
			Method:          http.MethodPatch,
			URL:             "/api/transactions/someid",
			ExpectedStatus:  http.StatusUnauthorized,
			ExpectedContent: []string{`"message"`},
			TestAppFactory: func(t testing.TB) *tests.TestApp {
				return newTransactionTestApp(t)
			},
			BeforeTestFunc: func(t testing.TB, app *tests.TestApp, e *core.ServeEvent) {
				bindTransactionRoutes(app, e)
			},
		}).Test(t)
	})

	t.Run("PATCH wrong owner returns 400", func(t *testing.T) {
		app := newTransactionTestApp(t)
		defer app.Cleanup()

		_, _, _, txID := seedTransactionTestData(t, app)

		userCol, err := app.FindCollectionByNameOrId("users")
		if err != nil {
			t.Fatalf("failed to find users: %v", err)
		}
		otherUser := core.NewRecord(userCol)
		otherUser.Set("name", "Other User")
		otherUser.Set("email", fmt.Sprintf("other+%d@example.com", time.Now().UnixNano()))
		otherUser.SetPassword("password1234")
		if err := app.Save(otherUser); err != nil {
			t.Fatalf("failed to save other user: %v", err)
		}
		otherToken, err := otherUser.NewAuthToken()
		if err != nil {
			t.Fatalf("failed to generate token: %v", err)
		}

		(&tests.ApiScenario{
			Name:   "PATCH /api/transactions/:id wrong owner returns 400",
			Method: http.MethodPatch,
			URL:    "/api/transactions/" + txID,
			Headers: map[string]string{
				"Authorization": "Bearer " + otherToken,
				"Content-Type":  "application/json",
			},
			Body:            strings.NewReader(`{"note":"hacked"}`),
			ExpectedStatus:  http.StatusBadRequest,
			ExpectedContent: []string{`"message"`},
			TestAppFactory: func(t testing.TB) *tests.TestApp {
				return app
			},
			DisableTestAppCleanup: true,
			BeforeTestFunc: func(t testing.TB, app *tests.TestApp, e *core.ServeEvent) {
				bindTransactionRoutes(app, e)
			},
		}).Test(t)
	})

	t.Run("PATCH valid update returns 200", func(t *testing.T) {
		app := newTransactionTestApp(t)
		defer app.Cleanup()

		token, _, _, txID := seedTransactionTestData(t, app)

		(&tests.ApiScenario{
			Name:   "valid PATCH /api/transactions/:id returns 200",
			Method: http.MethodPatch,
			URL:    "/api/transactions/" + txID,
			Headers: map[string]string{
				"Authorization": "Bearer " + token,
				"Content-Type":  "application/json",
			},
			Body:            strings.NewReader(`{"note":"updated note"}`),
			ExpectedStatus:  http.StatusOK,
			ExpectedContent: []string{`"id"`},
			TestAppFactory: func(t testing.TB) *tests.TestApp {
				return app
			},
			DisableTestAppCleanup: true,
			BeforeTestFunc: func(t testing.TB, app *tests.TestApp, e *core.ServeEvent) {
				bindTransactionRoutes(app, e)
			},
		}).Test(t)
	})

	t.Run("PATCH invalid JSON returns 400", func(t *testing.T) {
		app := newTransactionTestApp(t)
		defer app.Cleanup()
		token, _, _, txID := seedTransactionTestData(t, app)

		(&tests.ApiScenario{
			Name:   "PATCH /api/transactions/:id invalid JSON returns 400",
			Method: http.MethodPatch,
			URL:    "/api/transactions/" + txID,
			Headers: map[string]string{
				"Authorization": "Bearer " + token,
				"Content-Type":  "application/json",
			},
			Body:            strings.NewReader(`{"note":`),
			ExpectedStatus:  http.StatusBadRequest,
			ExpectedContent: []string{`"message"`},
			TestAppFactory: func(t testing.TB) *tests.TestApp {
				return app
			},
			DisableTestAppCleanup: true,
			BeforeTestFunc: func(t testing.TB, app *tests.TestApp, e *core.ServeEvent) {
				bindTransactionRoutes(app, e)
			},
		}).Test(t)
	})

	t.Run("DELETE guest returns 401", func(t *testing.T) {
		(&tests.ApiScenario{
			Name:            "guest DELETE /api/transactions/:id returns 401",
			Method:          http.MethodDelete,
			URL:             "/api/transactions/someid",
			ExpectedStatus:  http.StatusUnauthorized,
			ExpectedContent: []string{`"message"`},
			TestAppFactory: func(t testing.TB) *tests.TestApp {
				return newTransactionTestApp(t)
			},
			BeforeTestFunc: func(t testing.TB, app *tests.TestApp, e *core.ServeEvent) {
				bindTransactionRoutes(app, e)
			},
		}).Test(t)
	})

	t.Run("DELETE valid returns 204", func(t *testing.T) {
		app := newTransactionTestApp(t)
		defer app.Cleanup()

		token, _, _, txID := seedTransactionTestData(t, app)

		(&tests.ApiScenario{
			Name:   "valid DELETE /api/transactions/:id returns 204",
			Method: http.MethodDelete,
			URL:    "/api/transactions/" + txID,
			Headers: map[string]string{
				"Authorization": "Bearer " + token,
			},
			ExpectedStatus: http.StatusNoContent,
			TestAppFactory: func(t testing.TB) *tests.TestApp {
				return app
			},
			DisableTestAppCleanup: true,
			BeforeTestFunc: func(t testing.TB, app *tests.TestApp, e *core.ServeEvent) {
				bindTransactionRoutes(app, e)
			},
		}).Test(t)
	})

	t.Run("GET balance guest returns 401", func(t *testing.T) {
		(&tests.ApiScenario{
			Name:            "guest GET /api/families/:familyId/balance returns 401",
			Method:          http.MethodGet,
			URL:             "/api/families/somefamilyid/balance",
			ExpectedStatus:  http.StatusUnauthorized,
			ExpectedContent: []string{`"message"`},
			TestAppFactory: func(t testing.TB) *tests.TestApp {
				return newTransactionTestApp(t)
			},
			BeforeTestFunc: func(t testing.TB, app *tests.TestApp, e *core.ServeEvent) {
				bindTransactionRoutes(app, e)
			},
		}).Test(t)
	})

	t.Run("GET balance valid returns 200", func(t *testing.T) {
		app := newTransactionTestApp(t)
		defer app.Cleanup()

		token, famID, _, _ := seedTransactionTestData(t, app)

		(&tests.ApiScenario{
			Name:   "valid GET /api/families/:familyId/balance returns 200",
			Method: http.MethodGet,
			URL:    "/api/families/" + famID + "/balance",
			Headers: map[string]string{
				"Authorization": "Bearer " + token,
			},
			ExpectedStatus:  http.StatusOK,
			ExpectedContent: []string{`"balance"`},
			TestAppFactory: func(t testing.TB) *tests.TestApp {
				return app
			},
			DisableTestAppCleanup: true,
			BeforeTestFunc: func(t testing.TB, app *tests.TestApp, e *core.ServeEvent) {
				bindTransactionRoutes(app, e)
			},
		}).Test(t)
	})
}
