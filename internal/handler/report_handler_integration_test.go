package handler_test

import (
	"net/http"
	"testing"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
	"kas/internal/handler"
	"kas/internal/middleware"
	"kas/internal/repository"
	"kas/internal/service"
)

func bindReportRoutes(app *tests.TestApp, e *core.ServeEvent) {
	transactionRepo := repository.NewTransactionRepository(app)
	familyMemberRepo := repository.NewFamilyMemberRepository(app)
	reportService := service.NewReportService(transactionRepo)
	requireFamily := middleware.RequireFamily(familyMemberRepo)
	reportHandler := handler.NewReportHandler(reportService, familyMemberRepo, middleware.RequireAuth, requireFamily)
	reportHandler.RegisterRoutes(e)
}

func TestReportHandler(t *testing.T) {
	t.Run("GET /api/reports/monthly guest returns 401", func(t *testing.T) {
		(&tests.ApiScenario{
			Name:            "guest GET /api/reports/monthly returns 401",
			Method:          http.MethodGet,
			URL:             "/api/reports/monthly?year=2026&month=1",
			ExpectedStatus:  http.StatusUnauthorized,
			ExpectedContent: []string{`"message"`},
			TestAppFactory: func(t testing.TB) *tests.TestApp {
				return newTransactionTestApp(t)
			},
			BeforeTestFunc: func(t testing.TB, app *tests.TestApp, e *core.ServeEvent) {
				bindReportRoutes(app, e)
			},
		}).Test(t)
	})

	t.Run("GET /api/reports/monthly missing year returns 400", func(t *testing.T) {
		app := newTransactionTestApp(t)
		defer app.Cleanup()
		token, _, _, _ := seedTransactionTestData(t, app)

		(&tests.ApiScenario{
			Name:   "GET /api/reports/monthly missing year returns 400",
			Method: http.MethodGet,
			URL:    "/api/reports/monthly?month=3",
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
				bindReportRoutes(app, e)
			},
		}).Test(t)
	})

	t.Run("GET /api/reports/monthly missing month returns 400", func(t *testing.T) {
		app := newTransactionTestApp(t)
		defer app.Cleanup()
		token, _, _, _ := seedTransactionTestData(t, app)

		(&tests.ApiScenario{
			Name:   "GET /api/reports/monthly missing month returns 400",
			Method: http.MethodGet,
			URL:    "/api/reports/monthly?year=2026",
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
				bindReportRoutes(app, e)
			},
		}).Test(t)
	})

	t.Run("GET /api/reports/monthly invalid year returns 400", func(t *testing.T) {
		app := newTransactionTestApp(t)
		defer app.Cleanup()
		token, _, _, _ := seedTransactionTestData(t, app)

		(&tests.ApiScenario{
			Name:   "GET /api/reports/monthly invalid year returns 400",
			Method: http.MethodGet,
			URL:    "/api/reports/monthly?year=1800&month=3",
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
				bindReportRoutes(app, e)
			},
		}).Test(t)
	})

	t.Run("GET /api/reports/monthly invalid month > 12 returns 400", func(t *testing.T) {
		app := newTransactionTestApp(t)
		defer app.Cleanup()
		token, _, _, _ := seedTransactionTestData(t, app)

		(&tests.ApiScenario{
			Name:   "GET /api/reports/monthly invalid month > 12 returns 400",
			Method: http.MethodGet,
			URL:    "/api/reports/monthly?year=2026&month=13",
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
				bindReportRoutes(app, e)
			},
		}).Test(t)
	})

	t.Run("GET /api/reports/monthly valid returns 200", func(t *testing.T) {
		app := newTransactionTestApp(t)
		defer app.Cleanup()
		token, _, _, _ := seedTransactionTestData(t, app)

		(&tests.ApiScenario{
			Name:   "valid GET /api/reports/monthly returns 200",
			Method: http.MethodGet,
			URL:    "/api/reports/monthly?year=2026&month=1",
			Headers: map[string]string{
				"Authorization": "Bearer " + token,
			},
			ExpectedStatus:  http.StatusOK,
			ExpectedContent: []string{`"total_income"`, `"total_expense"`, `"balance"`, `"expense_breakdown"`, `"income_breakdown"`},
			TestAppFactory: func(t testing.TB) *tests.TestApp {
				return app
			},
			DisableTestAppCleanup: true,
			BeforeTestFunc: func(t testing.TB, app *tests.TestApp, e *core.ServeEvent) {
				bindReportRoutes(app, e)
			},
		}).Test(t)
	})

	t.Run("GET /api/reports/summary guest returns 401", func(t *testing.T) {
		(&tests.ApiScenario{
			Name:            "guest GET /api/reports/summary returns 401",
			Method:          http.MethodGet,
			URL:             "/api/reports/summary",
			ExpectedStatus:  http.StatusUnauthorized,
			ExpectedContent: []string{`"message"`},
			TestAppFactory: func(t testing.TB) *tests.TestApp {
				return newTransactionTestApp(t)
			},
			BeforeTestFunc: func(t testing.TB, app *tests.TestApp, e *core.ServeEvent) {
				bindReportRoutes(app, e)
			},
		}).Test(t)
	})

	t.Run("GET /api/reports/summary invalid year returns 400", func(t *testing.T) {
		app := newTransactionTestApp(t)
		defer app.Cleanup()
		token, _, _, _ := seedTransactionTestData(t, app)

		(&tests.ApiScenario{
			Name:   "GET /api/reports/summary invalid year returns 400",
			Method: http.MethodGet,
			URL:    "/api/reports/summary?year=bad&month=1",
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
				bindReportRoutes(app, e)
			},
		}).Test(t)
	})

	t.Run("GET /api/reports/summary invalid month returns 400", func(t *testing.T) {
		app := newTransactionTestApp(t)
		defer app.Cleanup()
		token, _, _, _ := seedTransactionTestData(t, app)

		(&tests.ApiScenario{
			Name:   "GET /api/reports/summary invalid month returns 400",
			Method: http.MethodGet,
			URL:    "/api/reports/summary?year=2026&month=0",
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
				bindReportRoutes(app, e)
			},
		}).Test(t)
	})

	t.Run("GET /api/reports/summary valid with explicit year/month returns 200", func(t *testing.T) {
		app := newTransactionTestApp(t)
		defer app.Cleanup()
		token, _, _, _ := seedTransactionTestData(t, app)

		(&tests.ApiScenario{
			Name:   "valid GET /api/reports/summary with year/month returns 200",
			Method: http.MethodGet,
			URL:    "/api/reports/summary?year=2026&month=1",
			Headers: map[string]string{
				"Authorization": "Bearer " + token,
			},
			ExpectedStatus:  http.StatusOK,
			ExpectedContent: []string{`"total_balance"`, `"monthly_income"`, `"monthly_expense"`, `"family_name"`, `"user_name"`},
			TestAppFactory: func(t testing.TB) *tests.TestApp {
				return app
			},
			DisableTestAppCleanup: true,
			BeforeTestFunc: func(t testing.TB, app *tests.TestApp, e *core.ServeEvent) {
				bindReportRoutes(app, e)
			},
		}).Test(t)
	})

	t.Run("GET /api/reports/summary valid with default year/month returns 200", func(t *testing.T) {
		app := newTransactionTestApp(t)
		defer app.Cleanup()
		token, _, _, _ := seedTransactionTestData(t, app)

		(&tests.ApiScenario{
			Name:   "valid GET /api/reports/summary default params returns 200",
			Method: http.MethodGet,
			URL:    "/api/reports/summary",
			Headers: map[string]string{
				"Authorization": "Bearer " + token,
			},
			ExpectedStatus:  http.StatusOK,
			ExpectedContent: []string{`"total_balance"`, `"monthly_income"`, `"monthly_expense"`, `"family_name"`, `"user_name"`},
			TestAppFactory: func(t testing.TB) *tests.TestApp {
				return app
			},
			DisableTestAppCleanup: true,
			BeforeTestFunc: func(t testing.TB, app *tests.TestApp, e *core.ServeEvent) {
				bindReportRoutes(app, e)
			},
		}).Test(t)
	})
}
