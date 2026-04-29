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

func bindFamilyRoutes(app *tests.TestApp, e *core.ServeEvent) {
	familyRepo := repository.NewFamilyRepository(app)
	familyMemberRepo := repository.NewFamilyMemberRepository(app)
	familyService := service.NewFamilyService(familyRepo, familyMemberRepo, app, func(string) {})
	familyHandler := handler.NewFamilyHandler(familyService, middleware.RequireAuth)
	familyHandler.RegisterRoutes(e)
}

func seedFamilyTestUser(t testing.TB, app *tests.TestApp) (token string, userID string) {
	t.Helper()
	userCol, err := app.FindCollectionByNameOrId("users")
	if err != nil {
		t.Fatalf("failed to find users collection: %v", err)
	}
	user := core.NewRecord(userCol)
	user.Set("name", "Family Test User")
	user.Set("email", fmt.Sprintf("familytest+%d@example.com", time.Now().UnixNano()))
	user.SetPassword("password1234")
	if err := app.Save(user); err != nil {
		t.Fatalf("failed to save user: %v", err)
	}
	token, err = user.NewAuthToken()
	if err != nil {
		t.Fatalf("failed to generate auth token: %v", err)
	}
	return token, user.Id
}

func seedUserWithFamily(t testing.TB, app *tests.TestApp) (token string, inviteCode string) {
	t.Helper()
	token, userID := seedFamilyTestUser(t, app)
	familyCol, err := app.FindCollectionByNameOrId("families")
	if err != nil {
		t.Fatalf("failed to find families collection: %v", err)
	}
	family := core.NewRecord(familyCol)
	inviteCode = fmt.Sprintf("T%07d", time.Now().UnixNano()%10000000)
	family.Set("name", "Test Family")
	family.Set("invite_code", inviteCode)
	if err := app.Save(family); err != nil {
		t.Fatalf("failed to save family: %v", err)
	}
	memberCol, err := app.FindCollectionByNameOrId("family_members")
	if err != nil {
		t.Fatalf("failed to find family_members collection: %v", err)
	}
	member := core.NewRecord(memberCol)
	member.Set("user_id", userID)
	member.Set("family_id", family.Id)
	member.Set("role", "owner")
	if err := app.Save(member); err != nil {
		t.Fatalf("failed to save family member: %v", err)
	}
	return token, inviteCode
}

func TestFamilyHandler(t *testing.T) {
	t.Run("POST guest returns 401", func(t *testing.T) {
		(&tests.ApiScenario{
			Name:            "guest POST /api/families returns 401",
			Method:          http.MethodPost,
			URL:             "/api/families",
			ExpectedStatus:  http.StatusUnauthorized,
			ExpectedContent: []string{`"message"`},
			TestAppFactory: func(t testing.TB) *tests.TestApp {
				return newTransactionTestApp(t)
			},
			BeforeTestFunc: func(t testing.TB, app *tests.TestApp, e *core.ServeEvent) {
				bindFamilyRoutes(app, e)
			},
		}).Test(t)
	})

	t.Run("POST valid create returns 201", func(t *testing.T) {
		app := newTransactionTestApp(t)
		defer app.Cleanup()

		token, _ := seedFamilyTestUser(t, app)

		(&tests.ApiScenario{
			Name:   "valid POST /api/families returns 201",
			Method: http.MethodPost,
			URL:    "/api/families",
			Headers: map[string]string{
				"Content-Type":  "application/json",
				"Authorization": "Bearer " + token,
			},
			Body:            strings.NewReader(`{"name":"My Family"}`),
			ExpectedStatus:  http.StatusCreated,
			ExpectedContent: []string{`"invite_code"`, `"family"`, `"member"`},
			TestAppFactory: func(t testing.TB) *tests.TestApp {
				return app
			},
			DisableTestAppCleanup: true,
			BeforeTestFunc: func(t testing.TB, app *tests.TestApp, e *core.ServeEvent) {
				bindFamilyRoutes(app, e)
			},
		}).Test(t)
	})

	t.Run("POST invalid JSON returns 400", func(t *testing.T) {
		app := newTransactionTestApp(t)
		defer app.Cleanup()
		token, _ := seedFamilyTestUser(t, app)

		(&tests.ApiScenario{
			Name:   "POST /api/families invalid JSON returns 400",
			Method: http.MethodPost,
			URL:    "/api/families",
			Headers: map[string]string{
				"Content-Type":  "application/json",
				"Authorization": "Bearer " + token,
			},
			Body:            strings.NewReader(`{"name":`),
			ExpectedStatus:  http.StatusBadRequest,
			ExpectedContent: []string{`"message"`},
			TestAppFactory: func(t testing.TB) *tests.TestApp {
				return app
			},
			DisableTestAppCleanup: true,
			BeforeTestFunc: func(t testing.TB, app *tests.TestApp, e *core.ServeEvent) {
				bindFamilyRoutes(app, e)
			},
		}).Test(t)
	})

	t.Run("POST already in family returns 400", func(t *testing.T) {
		app := newTransactionTestApp(t)
		defer app.Cleanup()

		token, _ := seedUserWithFamily(t, app)

		(&tests.ApiScenario{
			Name:   "POST /api/families already in family returns 400",
			Method: http.MethodPost,
			URL:    "/api/families",
			Headers: map[string]string{
				"Content-Type":  "application/json",
				"Authorization": "Bearer " + token,
			},
			Body:            strings.NewReader(`{"name":"Another Family"}`),
			ExpectedStatus:  http.StatusBadRequest,
			ExpectedContent: []string{`"message"`},
			TestAppFactory: func(t testing.TB) *tests.TestApp {
				return app
			},
			DisableTestAppCleanup: true,
			BeforeTestFunc: func(t testing.TB, app *tests.TestApp, e *core.ServeEvent) {
				bindFamilyRoutes(app, e)
			},
		}).Test(t)
	})

	t.Run("JOIN guest returns 401", func(t *testing.T) {
		(&tests.ApiScenario{
			Name:            "guest POST /api/families/join returns 401",
			Method:          http.MethodPost,
			URL:             "/api/families/join",
			ExpectedStatus:  http.StatusUnauthorized,
			ExpectedContent: []string{`"message"`},
			TestAppFactory: func(t testing.TB) *tests.TestApp {
				return newTransactionTestApp(t)
			},
			BeforeTestFunc: func(t testing.TB, app *tests.TestApp, e *core.ServeEvent) {
				bindFamilyRoutes(app, e)
			},
		}).Test(t)
	})

	t.Run("JOIN valid returns 200", func(t *testing.T) {
		app := newTransactionTestApp(t)
		defer app.Cleanup()

		_, inviteCode := seedUserWithFamily(t, app)
		secondToken, _ := seedFamilyTestUser(t, app)

		(&tests.ApiScenario{
			Name:   "valid POST /api/families/join returns 200",
			Method: http.MethodPost,
			URL:    "/api/families/join",
			Headers: map[string]string{
				"Content-Type":  "application/json",
				"Authorization": "Bearer " + secondToken,
			},
			Body:            strings.NewReader(fmt.Sprintf(`{"invite_code":%q}`, inviteCode)),
			ExpectedStatus:  http.StatusOK,
			ExpectedContent: []string{`"id"`, `"name"`, `"invite_code"`},
			TestAppFactory: func(t testing.TB) *tests.TestApp {
				return app
			},
			DisableTestAppCleanup: true,
			BeforeTestFunc: func(t testing.TB, app *tests.TestApp, e *core.ServeEvent) {
				bindFamilyRoutes(app, e)
			},
		}).Test(t)
	})

	t.Run("JOIN invalid JSON returns 400", func(t *testing.T) {
		app := newTransactionTestApp(t)
		defer app.Cleanup()
		token, _ := seedFamilyTestUser(t, app)

		(&tests.ApiScenario{
			Name:   "POST /api/families/join invalid JSON returns 400",
			Method: http.MethodPost,
			URL:    "/api/families/join",
			Headers: map[string]string{
				"Content-Type":  "application/json",
				"Authorization": "Bearer " + token,
			},
			Body:            strings.NewReader(`{"invite_code":`),
			ExpectedStatus:  http.StatusBadRequest,
			ExpectedContent: []string{`"message"`},
			TestAppFactory: func(t testing.TB) *tests.TestApp {
				return app
			},
			DisableTestAppCleanup: true,
			BeforeTestFunc: func(t testing.TB, app *tests.TestApp, e *core.ServeEvent) {
				bindFamilyRoutes(app, e)
			},
		}).Test(t)
	})

	t.Run("JOIN invalid invite code returns 404", func(t *testing.T) {
		app := newTransactionTestApp(t)
		defer app.Cleanup()

		token, _ := seedFamilyTestUser(t, app)

		(&tests.ApiScenario{
			Name:   "POST /api/families/join invalid invite code returns 404",
			Method: http.MethodPost,
			URL:    "/api/families/join",
			Headers: map[string]string{
				"Content-Type":  "application/json",
				"Authorization": "Bearer " + token,
			},
			Body:            strings.NewReader(`{"invite_code":"XXXXXXXX"}`),
			ExpectedStatus:  http.StatusNotFound,
			ExpectedContent: []string{`"message"`},
			TestAppFactory: func(t testing.TB) *tests.TestApp {
				return app
			},
			DisableTestAppCleanup: true,
			BeforeTestFunc: func(t testing.TB, app *tests.TestApp, e *core.ServeEvent) {
				bindFamilyRoutes(app, e)
			},
		}).Test(t)
	})

	t.Run("JOIN already member returns 400", func(t *testing.T) {
		app := newTransactionTestApp(t)
		defer app.Cleanup()

		token, inviteCode := seedUserWithFamily(t, app)

		(&tests.ApiScenario{
			Name:   "POST /api/families/join already member returns 400",
			Method: http.MethodPost,
			URL:    "/api/families/join",
			Headers: map[string]string{
				"Content-Type":  "application/json",
				"Authorization": "Bearer " + token,
			},
			Body:            strings.NewReader(fmt.Sprintf(`{"invite_code":%q}`, inviteCode)),
			ExpectedStatus:  http.StatusBadRequest,
			ExpectedContent: []string{`"message"`},
			TestAppFactory: func(t testing.TB) *tests.TestApp {
				return app
			},
			DisableTestAppCleanup: true,
			BeforeTestFunc: func(t testing.TB, app *tests.TestApp, e *core.ServeEvent) {
				bindFamilyRoutes(app, e)
			},
		}).Test(t)
	})

	t.Run("LEAVE guest returns 401", func(t *testing.T) {
		(&tests.ApiScenario{
			Name:            "guest POST /api/families/leave returns 401",
			Method:          http.MethodPost,
			URL:             "/api/families/leave",
			ExpectedStatus:  http.StatusUnauthorized,
			ExpectedContent: []string{`"message"`},
			TestAppFactory: func(t testing.TB) *tests.TestApp {
				return newTransactionTestApp(t)
			},
			BeforeTestFunc: func(t testing.TB, app *tests.TestApp, e *core.ServeEvent) {
				bindFamilyRoutes(app, e)
			},
		}).Test(t)
	})

	t.Run("LEAVE valid returns 204", func(t *testing.T) {
		app := newTransactionTestApp(t)
		defer app.Cleanup()

		token, _ := seedUserWithFamily(t, app)

		(&tests.ApiScenario{
			Name:   "valid POST /api/families/leave returns 204",
			Method: http.MethodPost,
			URL:    "/api/families/leave",
			Headers: map[string]string{
				"Authorization": "Bearer " + token,
			},
			ExpectedStatus: http.StatusNoContent,
			TestAppFactory: func(t testing.TB) *tests.TestApp {
				return app
			},
			DisableTestAppCleanup: true,
			BeforeTestFunc: func(t testing.TB, app *tests.TestApp, e *core.ServeEvent) {
				bindFamilyRoutes(app, e)
			},
		}).Test(t)
	})

	t.Run("LEAVE not a member returns 400", func(t *testing.T) {
		app := newTransactionTestApp(t)
		defer app.Cleanup()

		token, _ := seedFamilyTestUser(t, app)

		(&tests.ApiScenario{
			Name:   "POST /api/families/leave not a member returns 400",
			Method: http.MethodPost,
			URL:    "/api/families/leave",
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
				bindFamilyRoutes(app, e)
			},
		}).Test(t)
	})
}
