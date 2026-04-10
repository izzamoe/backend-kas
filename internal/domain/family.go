package domain

import "time"

type FamilyDTO struct {
	ID         string    `json:"id"`
	Name       string    `json:"name"`
	InviteCode string    `json:"invite_code"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

type CreateFamilyRequest struct {
	Name string `json:"name"`
}

type JoinFamilyRequest struct {
	InviteCode string `json:"invite_code"`
}

type CreateFamilyResponse struct {
	Family FamilyDTO       `json:"family"`
	Member FamilyMemberDTO `json:"member"`
}
