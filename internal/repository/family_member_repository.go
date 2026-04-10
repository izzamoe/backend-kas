package repository

import (
	"database/sql"
	"errors"
	"fmt"
	"kas/generated"
	"kas/internal/domain"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

type FamilyMemberRepository interface {
	GetByUserID(userID string) (*domain.FamilyMemberDTO, error)
	GetFamilyName(familyID string) (string, error)
	CreateMember(app core.App, familyID string, userID string, role string) error
	DeleteMember(userID string) error
}

type familyMemberRepo struct {
	app core.App
}

func NewFamilyMemberRepository(app core.App) FamilyMemberRepository {
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

func (r *familyMemberRepo) CreateMember(app core.App, familyID string, userID string, role string) error {
	collection, err := app.FindCachedCollectionByNameOrId("family_members")
	if err != nil {
		return fmt.Errorf("failed to find family_members collection: %w", err)
	}
	record := core.NewRecord(collection)
	record.Set("family_id", familyID)
	record.Set("user_id", userID)
	record.Set("role", role)
	return app.Save(record)
}

func (r *familyMemberRepo) DeleteMember(userID string) error {
	record, err := r.app.FindFirstRecordByFilter(
		"family_members",
		"user_id = {:userID}",
		dbx.Params{"userID": userID},
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("not a member of any family")
		}
		return fmt.Errorf("failed to find family member: %w", err)
	}
	if record == nil {
		return fmt.Errorf("not a member of any family")
	}
	return r.app.Delete(record)
}
