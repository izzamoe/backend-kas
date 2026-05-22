package middleware

import (
	"database/sql"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

type familyRoleCache struct {
	mu      sync.RWMutex
	entries map[string]*familyRoleCacheEntry
}

type familyRoleCacheEntry struct {
	role      string
	expiresAt time.Time
}

const familyRoleCacheTTL = 5 * time.Minute

var roleCache = &familyRoleCache{
	entries: make(map[string]*familyRoleCacheEntry, 64),
}

func (c *familyRoleCache) get(key string) (string, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	entry, ok := c.entries[key]
	if !ok || time.Now().After(entry.expiresAt) {
		return "", false
	}

	return entry.role, true
}

func (c *familyRoleCache) set(key, role string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.entries[key] = &familyRoleCacheEntry{
		role:      role,
		expiresAt: time.Now().Add(familyRoleCacheTTL),
	}
}

func familyRoleCacheKey(familyID, userID string) string {
	return familyID + ":" + userID
}

func (c *familyRoleCache) clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.entries = make(map[string]*familyRoleCacheEntry, 64)
}

// ClearRoleCache removes all entries from the role cache.
// This is useful for testing to avoid stale cache entries across test runs.
func ClearRoleCache() {
	roleCache.clear()
}

// IsFamilyOwner checks whether the given user is the owner of the specified family.
func IsFamilyOwner(app core.App, familyID, userID string) (bool, error) {
	if familyID == "" || userID == "" {
		return false, nil
	}

	key := familyRoleCacheKey(familyID, userID)
	if role, ok := roleCache.get(key); ok {
		return role == "owner", nil
	}

	record, err := app.FindFirstRecordByFilter(
		"family_members",
		"family_id = {:familyId} && user_id = {:userId} && role = \"owner\"",
		dbx.Params{
			"familyId": familyID,
			"userId":   userID,
		},
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			roleCache.set(key, "")
			return false, nil
		}
		return false, fmt.Errorf("failed to check family ownership: %w", err)
	}

	if record == nil {
		roleCache.set(key, "")
		return false, nil
	}

	roleCache.set(key, "owner")
	return true, nil
}

// GetFamilyRole returns the role of the given user in the specified family.
func GetFamilyRole(app core.App, familyID, userID string) (string, error) {
	if familyID == "" || userID == "" {
		return "", nil
	}

	key := familyRoleCacheKey(familyID, userID)
	if role, ok := roleCache.get(key); ok {
		return role, nil
	}

	record, err := app.FindFirstRecordByFilter(
		"family_members",
		"family_id = {:familyId} && user_id = {:userId}",
		dbx.Params{
			"familyId": familyID,
			"userId":   userID,
		},
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			roleCache.set(key, "")
			return "", nil
		}
		return "", fmt.Errorf("failed to get family role: %w", err)
	}

	if record == nil {
		roleCache.set(key, "")
		return "", nil
	}

	role := record.GetString("role")
	roleCache.set(key, role)
	return role, nil
}

// RequireFamilyOwner ensures the authenticated user is the owner of the current family.
func RequireFamilyOwner() func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		if e.Auth == nil {
			return e.InternalServerError("RequireFamilyOwner: authentication middleware not applied", nil)
		}

		familyID, ok := GetFamilyIDFromContext(e.Request.Context())
		if !ok {
			return e.InternalServerError("RequireFamilyOwner: family context not found", nil)
		}

		isOwner, err := IsFamilyOwner(e.App, familyID, e.Auth.Id)
		if err != nil {
			return e.InternalServerError("Failed to check family ownership", err)
		}
		if !isOwner {
			return e.ForbiddenError("User is not the family owner", nil)
		}

		return e.Next()
	}
}
