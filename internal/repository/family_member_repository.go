package repository

import (
	"database/sql"
	"errors"
	"fmt"
	"kas/generated"
	"kas/internal/domain"

	"github.com/pocketbase/dbx"
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
	record, err := r.app.FindFirstRecordByFilter(
		"family_members",
		"user_id = {:userID}",
		dbx.Params{"userID": userID},
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	return r.recordToDTO(record)
}

func (r *familyMemberRepo) recordToDTO(record *core.Record) (*domain.FamilyMemberDTO, error) {
	proxy, err := generated.WrapRecord[generated.FamilyMembers](record)
	if err != nil {
		return nil, fmt.Errorf("failed to wrap family member record: %w", err)
	}

	return &domain.FamilyMemberDTO{
		ID:        proxy.Id,
		UserID:    record.GetString("user_id"),
		FamilyID:  record.GetString("family_id"),
		Role:      record.GetString("role"),
		CreatedAt: proxy.Created().Time(),
		UpdatedAt: proxy.Updated().Time(),
	}, nil
}

func (r *familyMemberRepo) GetFamilyName(familyID string) (string, error) {
	record, err := r.app.FindRecordById("families", familyID)
	if err != nil {
		return "", err
	}
	return record.GetString("name"), nil
}
