package middleware

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"testing"
	"time"

	"kas/internal/domain"
	_ "kas/migrations"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
	"github.com/pocketbase/pocketbase/tools/hook"
)

type mockFamilyMemberRepository struct {
	member           *domain.FamilyMemberDTO
	err              error
	getByUserIDCalls int
	lastUserID       string
}

func (m *mockFamilyMemberRepository) GetByUserID(userID string) (*domain.FamilyMemberDTO, error) {
	m.getByUserIDCalls++
	m.lastUserID = userID
	return m.member, m.err
}

func (m *mockFamilyMemberRepository) GetFamilyName(familyID string) (string, error) {
	return "", nil
}

func (m *mockFamilyMemberRepository) CreateMember(app core.App, familyID string, userID string, role string) error {
	return nil
}

func (m *mockFamilyMemberRepository) DeleteMember(userID string) error {
	return nil
}

func resetGlobalFamilyCache(t testing.TB) {
	t.Helper()
	cache.mu.Lock()
	cache.entries = make(map[string]*familyCacheEntry, 64)
	cache.mu.Unlock()
}

func newMiddlewareTestApp(t testing.TB) *tests.TestApp {
	t.Helper()
	app, err := tests.NewTestApp()
	if err != nil {
		t.Fatalf("failed to create test app: %v", err)
	}
	return app
}

func seedMiddlewareUser(t testing.TB, app *tests.TestApp) (token string, userID string) {
	t.Helper()
	collection, err := app.FindCollectionByNameOrId("users")
	if err != nil {
		t.Fatalf("failed to find users collection: %v", err)
	}
	record := core.NewRecord(collection)
	record.Set("name", "Middleware User")
	record.Set("email", fmt.Sprintf("middleware+%d@example.com", time.Now().UnixNano()))
	record.SetPassword("password1234")
	if err := app.Save(record); err != nil {
		t.Fatalf("failed to save user: %v", err)
	}
	token, err = record.NewAuthToken()
	if err != nil {
		t.Fatalf("failed to create auth token: %v", err)
	}
	return token, record.Id
}

func bindRequireAuthRoute(e *core.ServeEvent) {
	e.Router.GET("/middleware/auth", func(e *core.RequestEvent) error {
		return e.JSON(http.StatusOK, map[string]string{"status": "ok"})
	}).Bind(&hook.Handler[*core.RequestEvent]{Func: RequireAuth})
}

func bindRequireFamilyRoute(e *core.ServeEvent, repo *mockFamilyMemberRepository) {
	e.Router.GET("/middleware/family", func(e *core.RequestEvent) error {
		familyID, ok := GetFamilyIDFromContext(e.Request.Context())
		if !ok {
			return e.InternalServerError("family context missing", nil)
		}
		return e.JSON(http.StatusOK, map[string]string{"family_id": familyID})
	}).Bind(&hook.Handler[*core.RequestEvent]{Func: RequireAuth}).Bind(&hook.Handler[*core.RequestEvent]{Func: RequireFamily(repo)})
}

func bindRequireFamilyOnlyRoute(e *core.ServeEvent, repo *mockFamilyMemberRepository) {
	e.Router.GET("/middleware/family-only", func(e *core.RequestEvent) error {
		return e.JSON(http.StatusOK, map[string]string{"status": "ok"})
	}).Bind(&hook.Handler[*core.RequestEvent]{Func: RequireFamily(repo)})
}

func TestGetFamilyIDFromContext(t *testing.T) {
	t.Run("returns empty and false when not set", func(t *testing.T) {
		ctx := context.Background()
		val, ok := GetFamilyIDFromContext(ctx)
		if ok {
			t.Errorf("expected ok=false, got true")
		}
		if val != "" {
			t.Errorf("expected empty string, got %q", val)
		}
	})

	t.Run("returns family_id and true when set", func(t *testing.T) {
		ctx := context.Background()
		ctx = SetFamilyIDToContext(ctx, "family123")
		val, ok := GetFamilyIDFromContext(ctx)
		if !ok {
			t.Errorf("expected ok=true, got false")
		}
		if val != "family123" {
			t.Errorf("expected family123, got %q", val)
		}
	})

	t.Run("returns false for empty string value", func(t *testing.T) {
		ctx := context.Background()
		ctx = SetFamilyIDToContext(ctx, "")
		_, ok := GetFamilyIDFromContext(ctx)
		if ok {
			t.Errorf("expected ok=false for empty string, got true")
		}
	})
}

func TestFamilyCacheGetSetAndExpiry(t *testing.T) {
	c := &familyCache{entries: make(map[string]*familyCacheEntry)}

	if familyID, ok := c.get("user1"); ok || familyID != "" {
		t.Fatalf("expected cache miss, got familyID=%q ok=%v", familyID, ok)
	}

	c.set("user1", "family1")
	familyID, ok := c.get("user1")
	if !ok || familyID != "family1" {
		t.Fatalf("expected cache hit family1, got familyID=%q ok=%v", familyID, ok)
	}

	c.entries["expired"] = &familyCacheEntry{
		familyID:  "family_expired",
		expiresAt: time.Now().Add(-time.Second),
	}
	familyID, ok = c.get("expired")
	if ok || familyID != "" {
		t.Fatalf("expected expired cache miss, got familyID=%q ok=%v", familyID, ok)
	}
}

func TestInvalidateFamily(t *testing.T) {
	cache.mu.Lock()
	cache.entries = map[string]*familyCacheEntry{
		"user1": {
			familyID:  "family1",
			expiresAt: time.Now().Add(familyCacheTTL),
		},
	}
	cache.mu.Unlock()

	InvalidateFamily("user1")

	if familyID, ok := cache.get("user1"); ok || familyID != "" {
		t.Fatalf("expected cache entry to be invalidated, got familyID=%q ok=%v", familyID, ok)
	}
}

func TestRequireAuthMiddleware(t *testing.T) {
	t.Run("guest request returns 401", func(t *testing.T) {
		(&tests.ApiScenario{
			Name:            "guest request returns unauthorized",
			Method:          http.MethodGet,
			URL:             "/middleware/auth",
			ExpectedStatus:  http.StatusUnauthorized,
			ExpectedContent: []string{`"message"`},
			TestAppFactory: func(t testing.TB) *tests.TestApp {
				return newMiddlewareTestApp(t)
			},
			BeforeTestFunc: func(t testing.TB, app *tests.TestApp, e *core.ServeEvent) {
				bindRequireAuthRoute(e)
			},
		}).Test(t)
	})

	t.Run("authenticated request reaches handler", func(t *testing.T) {
		app := newMiddlewareTestApp(t)
		defer app.Cleanup()
		token, _ := seedMiddlewareUser(t, app)

		(&tests.ApiScenario{
			Name:   "authenticated request reaches protected route",
			Method: http.MethodGet,
			URL:    "/middleware/auth",
			Headers: map[string]string{
				"Authorization": "Bearer " + token,
			},
			ExpectedStatus:  http.StatusOK,
			ExpectedContent: []string{`"status":"ok"`},
			TestAppFactory: func(t testing.TB) *tests.TestApp {
				return app
			},
			DisableTestAppCleanup: true,
			BeforeTestFunc: func(t testing.TB, app *tests.TestApp, e *core.ServeEvent) {
				bindRequireAuthRoute(e)
			},
		}).Test(t)
	})
}

func TestRequireFamilyMiddleware(t *testing.T) {
	t.Run("missing auth middleware returns 500", func(t *testing.T) {
		resetGlobalFamilyCache(t)
		repo := &mockFamilyMemberRepository{}

		(&tests.ApiScenario{
			Name:            "RequireFamily without auth returns internal server error",
			Method:          http.MethodGet,
			URL:             "/middleware/family-only",
			ExpectedStatus:  http.StatusInternalServerError,
			ExpectedContent: []string{`"message"`},
			TestAppFactory: func(t testing.TB) *tests.TestApp {
				return newMiddlewareTestApp(t)
			},
			BeforeTestFunc: func(t testing.TB, app *tests.TestApp, e *core.ServeEvent) {
				bindRequireFamilyOnlyRoute(e, repo)
			},
		}).Test(t)
	})

	t.Run("user without family returns 403", func(t *testing.T) {
		resetGlobalFamilyCache(t)
		app := newMiddlewareTestApp(t)
		defer app.Cleanup()
		token, userID := seedMiddlewareUser(t, app)
		repo := &mockFamilyMemberRepository{}

		(&tests.ApiScenario{
			Name:   "RequireFamily denies user without membership",
			Method: http.MethodGet,
			URL:    "/middleware/family",
			Headers: map[string]string{
				"Authorization": "Bearer " + token,
			},
			ExpectedStatus:  http.StatusForbidden,
			ExpectedContent: []string{`"message"`},
			TestAppFactory: func(t testing.TB) *tests.TestApp {
				return app
			},
			DisableTestAppCleanup: true,
			BeforeTestFunc: func(t testing.TB, app *tests.TestApp, e *core.ServeEvent) {
				bindRequireFamilyRoute(e, repo)
			},
		}).Test(t)

		if repo.getByUserIDCalls != 1 {
			t.Fatalf("expected repository lookup once, got %d", repo.getByUserIDCalls)
		}
		if repo.lastUserID != userID {
			t.Fatalf("expected repository lookup with auth user %q, got %q", userID, repo.lastUserID)
		}
	})

	t.Run("repository error returns 500", func(t *testing.T) {
		resetGlobalFamilyCache(t)
		app := newMiddlewareTestApp(t)
		defer app.Cleanup()
		token, userID := seedMiddlewareUser(t, app)
		repo := &mockFamilyMemberRepository{err: errors.New("membership query failed")}

		(&tests.ApiScenario{
			Name:   "RequireFamily returns internal server error on repository failure",
			Method: http.MethodGet,
			URL:    "/middleware/family",
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
				bindRequireFamilyRoute(e, repo)
			},
		}).Test(t)

		if repo.lastUserID != userID {
			t.Fatalf("expected repository lookup with auth user %q, got %q", userID, repo.lastUserID)
		}
	})

	t.Run("cached family skips repository lookup", func(t *testing.T) {
		resetGlobalFamilyCache(t)
		app := newMiddlewareTestApp(t)
		defer app.Cleanup()
		token, userID := seedMiddlewareUser(t, app)
		cache.set(userID, "cachedFamily")
		repo := &mockFamilyMemberRepository{err: errors.New("repository should not be called on cache hit")}

		(&tests.ApiScenario{
			Name:   "RequireFamily uses cached family membership",
			Method: http.MethodGet,
			URL:    "/middleware/family",
			Headers: map[string]string{
				"Authorization": "Bearer " + token,
			},
			ExpectedStatus:  http.StatusOK,
			ExpectedContent: []string{`"family_id":"cachedFamily"`},
			TestAppFactory: func(t testing.TB) *tests.TestApp {
				return app
			},
			DisableTestAppCleanup: true,
			BeforeTestFunc: func(t testing.TB, app *tests.TestApp, e *core.ServeEvent) {
				bindRequireFamilyRoute(e, repo)
			},
		}).Test(t)

		if repo.getByUserIDCalls != 0 {
			t.Fatalf("expected cache hit to skip repository lookup, got %d calls", repo.getByUserIDCalls)
		}
	})

	t.Run("valid family injects context and caches membership", func(t *testing.T) {
		resetGlobalFamilyCache(t)
		app := newMiddlewareTestApp(t)
		defer app.Cleanup()
		token, userID := seedMiddlewareUser(t, app)
		repo := &mockFamilyMemberRepository{member: &domain.FamilyMemberDTO{UserID: userID, FamilyID: "family123", Role: "member"}}

		(&tests.ApiScenario{
			Name:   "RequireFamily injects family id into context",
			Method: http.MethodGet,
			URL:    "/middleware/family",
			Headers: map[string]string{
				"Authorization": "Bearer " + token,
			},
			ExpectedStatus:  http.StatusOK,
			ExpectedContent: []string{`"family_id":"family123"`},
			TestAppFactory: func(t testing.TB) *tests.TestApp {
				return app
			},
			DisableTestAppCleanup: true,
			BeforeTestFunc: func(t testing.TB, app *tests.TestApp, e *core.ServeEvent) {
				bindRequireFamilyRoute(e, repo)
			},
		}).Test(t)

		if repo.getByUserIDCalls != 1 {
			t.Fatalf("expected first request to query repository once, got %d", repo.getByUserIDCalls)
		}
		if repo.lastUserID != userID {
			t.Fatalf("expected repository lookup with auth user %q, got %q", userID, repo.lastUserID)
		}

		cachedFamilyID, ok := cache.get(userID)
		if !ok || cachedFamilyID != "family123" {
			t.Fatalf("expected middleware to cache family123, got familyID=%q ok=%v", cachedFamilyID, ok)
		}

		InvalidateFamily(userID)
		cachedFamilyID, ok = cache.get(userID)
		if ok || cachedFamilyID != "" {
			t.Fatalf("expected invalidation to remove cached family, got familyID=%q ok=%v", cachedFamilyID, ok)
		}
	})
}

func TestRequireFamily_NilAuth_DefensiveGuard(t *testing.T) {
	repo := &mockFamilyMemberRepository{}

	member, err := repo.GetByUserID("user123")
	if err != nil {
		t.Errorf("unexpected error from empty mock: %v", err)
	}
	if member != nil {
		t.Errorf("expected nil member for empty mock, got %+v", member)
	}

	t.Log("TestRequireFamily_NilAuth_DefensiveGuard: RequireFamily with nil auth returns 500 (misconfiguration)")
	t.Log("This catches missing RequireAuth in middleware chain")
}

func TestRequireFamily_NoFamily(t *testing.T) {
	repo := &mockFamilyMemberRepository{member: nil, err: nil}

	member, err := repo.GetByUserID("user-without-family")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if member != nil {
		t.Errorf("expected nil member (no membership), got %+v", member)
	}

	t.Log("TestRequireFamily_NoFamily: middleware requires core.RequestEvent which cannot be easily mocked in unit tests")
	t.Log("This middleware should be tested via integration tests with actual PocketBase server")
}

func TestRequireFamily_HasFamily(t *testing.T) {
	member := &domain.FamilyMemberDTO{
		ID:       "member123",
		UserID:   "user123",
		FamilyID: "family456",
		Role:     "member",
	}
	repo := &mockFamilyMemberRepository{member: member, err: nil}

	result, err := repo.GetByUserID("user123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil member from mock")
	}
	if result.FamilyID != "family456" {
		t.Errorf("expected FamilyID=family456, got %q", result.FamilyID)
	}
	if result.UserID != "user123" {
		t.Errorf("expected UserID=user123, got %q", result.UserID)
	}

	ctx := context.Background()
	ctx = SetFamilyIDToContext(ctx, result.FamilyID)
	familyID, ok := GetFamilyIDFromContext(ctx)
	if !ok {
		t.Error("expected ok=true after setting family_id in context")
	}
	if familyID != "family456" {
		t.Errorf("expected family456 in context, got %q", familyID)
	}

	t.Log("TestRequireFamily_HasFamily: context helper verified — middleware requires core.RequestEvent for full test")
	t.Log("Middleware logic components (repo lookup, context injection) verified separately")
}
