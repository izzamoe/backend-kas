package domain

import "time"

type FamilyMemberDTO struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	FamilyID  string    `json:"family_id"`
	Role      string    `json:"role"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
