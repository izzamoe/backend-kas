package middleware

import (
	"kas/internal/repository"

	"github.com/pocketbase/pocketbase/core"
)

// RequireAuth is a middleware that requires authentication
func RequireAuth(e *core.RequestEvent) error {
	if e.Auth == nil {
		return e.UnauthorizedError("Authentication required", nil)
	}
	return e.Next()
}

// RequireFamily is a middleware that requires authentication and family membership.
// It injects the family_id into the request context for downstream handlers.
func RequireFamily(repo repository.FamilyMemberRepository) func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		if e.Auth == nil {
			return e.UnauthorizedError("Authentication required", nil)
		}

		membership, err := repo.GetByUserID(e.Auth.Id)
		if err != nil {
			return e.InternalServerError("Failed to check family membership", err)
		}

		if membership == nil {
			return e.ForbiddenError("User is not a member of any family", nil)
		}

		ctx := SetFamilyIDToContext(e.Request.Context(), membership.FamilyID)
		e.Request = e.Request.WithContext(ctx)

		return e.Next()
	}
}
