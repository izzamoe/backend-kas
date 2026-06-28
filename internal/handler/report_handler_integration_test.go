package handler_test

import (
	"errors"
	"net/http"
	"testing"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"

	"kas/internal/domain"
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

type fakeReportService struct {
	monthlyErr error
	summaryErr error
}

func (f *fakeReportService) GetMonthlyReport(req *domain.MonthlyReportRequest) (*domain.MonthlyReportDTO, error) {
	if f.monthlyErr != nil {
		return nil, f.monthlyErr
	}
	return &domain.MonthlyReportDTO{FamilyID: req.FamilyID, Year: req.Year, Month: req.Month}, nil
}

func (f *fakeReportService) GetDashboardSummary(req *domain.DashboardSummaryRequest) (*domain.DashboardSummaryDTO, error) {
	if f.summaryErr != nil {
		return nil, f.summaryErr
	}
	return &domain.DashboardSummaryDTO{TotalBalance: 1000, MonthlyIncome: 100, MonthlyExpense: 50}, nil
}

type fakeReportFamilyMemberRepo struct {
	member        *domain.FamilyMemberDTO
	familyNameErr error
}

func (f *fakeReportFamilyMemberRepo) GetByUserID(userID string) (*domain.FamilyMemberDTO, error) {
	if f.member != nil {
		return f.member, nil
	}
	return &domain.FamilyMemberDTO{UserID: userID, FamilyID: "fam1", Role: "owner"}, nil
}

func (f *fakeReportFamilyMemberRepo) GetFamilyName(familyID string) (string, error) {
	if f.familyNameErr != nil {
		return "", f.familyNameErr
	}
	return "Keluarga Test", nil
}

func (f *fakeReportFamilyMemberRepo) CreateMember(app core.App, familyID string, userID string, role string) error {
	return nil
}

func (f *fakeReportFamilyMemberRepo) DeleteMember(userID string) error {
	return nil
}

func bindReportRoutesWithFakes(e *core.ServeEvent, reportService service.ReportService, familyMemberRepo repository.FamilyMemberRepository) {
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
			TestAppFactory:  newTransactionTestApp,
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

	t.Run("GET /api/reports/monthly service error returns 500", func(t *testing.T) {
		app := newTransactionTestApp(t)
		defer app.Cleanup()
		token, userID := seedFamilyTestUser(t, app)
		familyMemberRepo := &fakeReportFamilyMemberRepo{member: &domain.FamilyMemberDTO{UserID: userID, FamilyID: "fam1", Role: "owner"}}

		(&tests.ApiScenario{
			Name:   "GET /api/reports/monthly service error returns 500",
			Method: http.MethodGet,
			URL:    "/api/reports/monthly?year=2026&month=1",
			Headers: map[string]string{
				"Authorization": "Bearer " + token,
			},
			ExpectedStatus:  http.StatusInternalServerError,
			ExpectedContent: []string{`"message"`},
			TestAppFactory: func(t testing.TB) *tests.TestApp {
				return app
			},
			DisableTestAppCleanup: true,
			BeforeTestFunc: func(t testing.TB, app *tests.TestApp, e *core.ServeEvent) {
				bindReportRoutesWithFakes(e, &fakeReportService{monthlyErr: errors.New("monthly failed")}, familyMemberRepo)
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
			TestAppFactory:  newTransactionTestApp,
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

	t.Run("GET /api/reports/summary service error returns 500", func(t *testing.T) {
		app := newTransactionTestApp(t)
		defer app.Cleanup()
		token, userID := seedFamilyTestUser(t, app)
		familyMemberRepo := &fakeReportFamilyMemberRepo{member: &domain.FamilyMemberDTO{UserID: userID, FamilyID: "fam1", Role: "owner"}}

		(&tests.ApiScenario{
			Name:   "GET /api/reports/summary service error returns 500",
			Method: http.MethodGet,
			URL:    "/api/reports/summary?year=2026&month=1",
			Headers: map[string]string{
				"Authorization": "Bearer " + token,
			},
			ExpectedStatus:  http.StatusInternalServerError,
			ExpectedContent: []string{`"message"`},
			TestAppFactory: func(t testing.TB) *tests.TestApp {
				return app
			},
			DisableTestAppCleanup: true,
			BeforeTestFunc: func(t testing.TB, app *tests.TestApp, e *core.ServeEvent) {
				bindReportRoutesWithFakes(e, &fakeReportService{summaryErr: errors.New("summary failed")}, familyMemberRepo)
			},
		}).Test(t)
	})

	t.Run("GET /api/reports/summary family name error returns 500", func(t *testing.T) {
		app := newTransactionTestApp(t)
		defer app.Cleanup()
		token, userID := seedFamilyTestUser(t, app)
		familyMemberRepo := &fakeReportFamilyMemberRepo{
			member:        &domain.FamilyMemberDTO{UserID: userID, FamilyID: "fam1", Role: "owner"},
			familyNameErr: errors.New("family name failed"),
		}

		(&tests.ApiScenario{
			Name:   "GET /api/reports/summary family name error returns 500",
			Method: http.MethodGet,
			URL:    "/api/reports/summary?year=2026&month=1",
			Headers: map[string]string{
				"Authorization": "Bearer " + token,
			},
			ExpectedStatus:  http.StatusInternalServerError,
			ExpectedContent: []string{`"message"`},
			TestAppFactory: func(t testing.TB) *tests.TestApp {
				return app
			},
			DisableTestAppCleanup: true,
			BeforeTestFunc: func(t testing.TB, app *tests.TestApp, e *core.ServeEvent) {
				bindReportRoutesWithFakes(e, &fakeReportService{}, familyMemberRepo)
			},
		}).Test(t)
	})
}
