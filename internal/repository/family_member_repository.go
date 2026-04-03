package repository

import (
	"kas/generated"
	"kas/internal/domain"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

type FamilyMemberRepository interface {
	GetByUserID(userID string) (*domain.FamilyMemberDTO, error)
	GetFamilyName(familyID string) (string, error)
}

type familyMemberRepo struct {
	app *pocketbase.PocketBase
}

func NewFamilyMemberRepository(app *pocketbase.PocketBase) FamilyMemberRepository {
	return &familyMemberRepo{app: app}
}

func (r *familyMemberRepo) GetByUserID(userID string) (*domain.FamilyMemberDTO, error) {
	records, err := r.app.FindRecordsByFilter(
		"family_members",
		"user_id = {:userID}",
		"",
		1,
		0,
		map[string]any{"userID": userID},
	)
	if err != nil {
		return nil, err
	}

	if len(records) == 0 {
		return nil, nil
	}

	return r.recordToDTO(records[0]), nil
}

func (r *familyMemberRepo) recordToDTO(record *core.Record) *domain.FamilyMemberDTO {
	proxy, _ := generated.WrapRecord[generated.FamilyMembers](record)

	return &domain.FamilyMemberDTO{
		ID:        proxy.Id,
		UserID:    record.GetString("user_id"),
		FamilyID:  record.GetString("family_id"),
		Role:      record.GetString("role"),
		CreatedAt: proxy.Created().Time(),
		UpdatedAt: proxy.Updated().Time(),
	}
}

func (r *familyMemberRepo) GetFamilyName(familyID string) (string, error) {
	record, err := r.app.FindRecordById("families", familyID)
	if err != nil {
		return "", err
	}
	return record.GetString("name"), nil
}
