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

type FamilyRepository interface {
	Create(app core.App, name string, inviteCode string) (*core.Record, error)
	FindByInviteCode(code string) (*domain.FamilyDTO, error)
}

type familyRepo struct {
	app core.App
}

func NewFamilyRepository(app core.App) FamilyRepository {
	return &familyRepo{app: app}
}

func (r *familyRepo) Create(app core.App, name string, inviteCode string) (*core.Record, error) {
	collection, err := app.FindCachedCollectionByNameOrId("families")
	if err != nil {
		return nil, fmt.Errorf("failed to find families collection: %w", err)
	}
	record := core.NewRecord(collection)
	record.Set("name", name)
	record.Set("invite_code", inviteCode)
	if err := app.Save(record); err != nil {
		return nil, fmt.Errorf("failed to save family: %w", err)
	}
	return record, nil
}

func (r *familyRepo) FindByInviteCode(code string) (*domain.FamilyDTO, error) {
	record, err := r.app.FindFirstRecordByFilter(
		"families",
		"invite_code = {:code}",
		dbx.Params{"code": code},
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to find family by invite code: %w", err)
	}
	return r.recordToDTO(record)
}

func (r *familyRepo) recordToDTO(record *core.Record) (*domain.FamilyDTO, error) {
	proxy, err := generated.WrapRecord[generated.Families](record)
	if err != nil {
		return nil, fmt.Errorf("failed to wrap family record: %w", err)
	}
	return &domain.FamilyDTO{
		ID:         proxy.Id,
		Name:       proxy.Name(),
		InviteCode: record.GetString("invite_code"),
		CreatedAt:  proxy.Created().Time(),
		UpdatedAt:  proxy.Updated().Time(),
	}, nil
}
