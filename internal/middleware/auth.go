package middleware

import (
	"kas/internal/repository"
	"sync"
	"time"

	"github.com/pocketbase/pocketbase/core"
)

// familyCache stores cached family membership lookups with TTL
type familyCache struct {
	mu      sync.RWMutex
	entries map[string]*familyCacheEntry
}

type familyCacheEntry struct {
	familyID  string
	expiresAt time.Time
}

const familyCacheTTL = 5 * time.Minute

var cache = &familyCache{
	entries: make(map[string]*familyCacheEntry, 64),
}

// get returns cached familyID if valid, empty string if expired/missing
func (c *familyCache) get(userID string) (string, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	entry, ok := c.entries[userID]
	if !ok || time.Now().After(entry.expiresAt) {
		return "", false
	}
	return entry.familyID, true
}

// set stores a familyID with TTL
func (c *familyCache) set(userID, familyID string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.entries[userID] = &familyCacheEntry{
		familyID:  familyID,
		expiresAt: time.Now().Add(familyCacheTTL),
	}
}

// RequireAuth is a middleware that requires authentication
func RequireAuth(e *core.RequestEvent) error {
	if e.Auth == nil {
		return e.UnauthorizedError("Authentication required", nil)
	}
	return e.Next()
}

// RequireFamily checks family membership and injects family_id into the request context.
// MUST be chained after RequireAuth middleware.
// Results are cached with a 5-minute TTL to avoid repeated DB queries.
func RequireFamily(repo repository.FamilyMemberRepository) func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		if e.Auth == nil {
			return e.InternalServerError("RequireFamily: authentication middleware not applied", nil)
		}

		userID := e.Auth.Id

		// Check cache first
		if familyID, ok := cache.get(userID); ok {
			ctx := SetFamilyIDToContext(e.Request.Context(), familyID)
			e.Request = e.Request.WithContext(ctx)
			return e.Next()
		}

		// Cache miss — query DB
		membership, err := repo.GetByUserID(userID)
		if err != nil {
			return e.InternalServerError("Failed to check family membership", err)
		}

		if membership == nil {
			return e.ForbiddenError("User is not a member of any family", nil)
		}

		// Store in cache
		cache.set(userID, membership.FamilyID)

		ctx := SetFamilyIDToContext(e.Request.Context(), membership.FamilyID)
		e.Request = e.Request.WithContext(ctx)

		return e.Next()
	}
}
