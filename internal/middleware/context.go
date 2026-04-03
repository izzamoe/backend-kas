package middleware

import "context"

type contextKey string

const familyIDKey contextKey = "familyID"

func SetFamilyIDToContext(ctx context.Context, familyID string) context.Context {
	return context.WithValue(ctx, familyIDKey, familyID)
}

func GetFamilyIDFromContext(ctx context.Context) (string, bool) {
	val, ok := ctx.Value(familyIDKey).(string)
	if !ok || val == "" {
		return "", false
	}
	return val, true
}
