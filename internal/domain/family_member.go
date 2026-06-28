package domain

import "time"

// FamilyMemberDTO represents a user's membership in a family.
type FamilyMemberDTO struct {
	ID       string `json:"id"`
	UserID   string `json:"user_id"`
	FamilyID string `json:"family_id"`
	// @Enums owner, member
	Role      string    `json:"role" enums:"owner,member"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
