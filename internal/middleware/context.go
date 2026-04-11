package middleware

import "context"

// contextKey is an unexported type for context keys in this package, preventing key collisions.
type contextKey string

// familyIDKey is the context key for storing the family ID in request context.
const familyIDKey contextKey = "familyID"

// SetFamilyIDToContext stores the given familyID in the request context and returns the updated context.
func SetFamilyIDToContext(ctx context.Context, familyID string) context.Context {
	return context.WithValue(ctx, familyIDKey, familyID)
}

// GetFamilyIDFromContext retrieves the familyID from the given context. Returns empty string and false if not set.
func GetFamilyIDFromContext(ctx context.Context) (string, bool) {
	val, ok := ctx.Value(familyIDKey).(string)
	if !ok || val == "" {
		return "", false
	}
	return val, true
}
