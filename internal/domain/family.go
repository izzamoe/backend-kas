package domain

import "time"

// FamilyDTO represents a family group.
type FamilyDTO struct {
	ID         string    `json:"id"`
	Name       string    `json:"name"`
	InviteCode string    `json:"invite_code"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// CreateFamilyRequest is the payload for creating a new family.
type CreateFamilyRequest struct {
	Name string `json:"name"`
}

// JoinFamilyRequest is the payload for joining a family via invite code.
type JoinFamilyRequest struct {
	InviteCode string `json:"invite_code"`
}

// CreateFamilyResponse is the response after successfully creating a family.
type CreateFamilyResponse struct {
	Family FamilyDTO       `json:"family"`
	Member FamilyMemberDTO `json:"member"`
}
