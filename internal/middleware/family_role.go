package middleware

import (
	"database/sql"
	"errors"
	"fmt"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

// IsFamilyOwner checks whether the given user is the owner of the specified family.
func IsFamilyOwner(app core.App, familyID, userID string) (bool, error) {
	if familyID == "" || userID == "" {
		return false, nil
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
			return false, nil
		}
		return false, fmt.Errorf("failed to check family ownership: %w", err)
	}

	return record != nil, nil
}

// GetFamilyRole returns the role of the given user in the specified family.
func GetFamilyRole(app core.App, familyID, userID string) (string, error) {
	if familyID == "" || userID == "" {
		return "", nil
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
			return "", nil
		}
		return "", fmt.Errorf("failed to get family role: %w", err)
	}

	if record == nil {
		return "", nil
	}

	return record.GetString("role"), nil
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
