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

// FamilyRepository defines the data access contract for family records.
type FamilyRepository interface {
	// Create persists a new family with the given name and invite code, then returns the created family as a DTO.
	Create(app core.App, name string, inviteCode string) (*domain.FamilyDTO, error)
	// FindByInviteCode looks up a family by its unique invite code. Returns nil, nil if no family is found.
	FindByInviteCode(code string) (*domain.FamilyDTO, error)
}

// familyRepo is the concrete PocketBase implementation of FamilyRepository.
type familyRepo struct {
	app core.App
}

// NewFamilyRepository creates a new PocketBase-backed FamilyRepository.
func NewFamilyRepository(app core.App) FamilyRepository {
	return &familyRepo{app: app}
}

// Create persists a new family with the given name and invite code, then returns the created family as a DTO.
func (r *familyRepo) Create(app core.App, name string, inviteCode string) (*domain.FamilyDTO, error) {
	proxy, err := generated.NewProxy[generated.Families](app)
	if err != nil {
		return nil, fmt.Errorf("failed to create family proxy: %w", err)
	}

	proxy.SetName(name)
	proxy.SetInviteCode(inviteCode)

	if err := app.Save(proxy.Record); err != nil {
		return nil, fmt.Errorf("failed to save family: %w", err)
	}

	return &domain.FamilyDTO{
		ID:         proxy.Id,
		Name:       proxy.Name(),
		InviteCode: proxy.InviteCode(),
		CreatedAt:  proxy.Created().Time(),
		UpdatedAt:  proxy.Updated().Time(),
	}, nil
}

// FindByInviteCode looks up a family by its unique invite code. Returns nil, nil if no family is found.
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

// recordToDTO converts a raw PocketBase families record into a FamilyDTO using type-safe proxy access.
func (r *familyRepo) recordToDTO(record *core.Record) (*domain.FamilyDTO, error) {
	proxy, err := generated.WrapRecord[generated.Families](record)
	if err != nil {
		return nil, fmt.Errorf("failed to wrap family record: %w", err)
	}
	return &domain.FamilyDTO{
		ID:         proxy.Id,
		Name:       proxy.Name(),
		InviteCode: proxy.InviteCode(),
		CreatedAt:  proxy.Created().Time(),
		UpdatedAt:  proxy.Updated().Time(),
	}, nil
}
