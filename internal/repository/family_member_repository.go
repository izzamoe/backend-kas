package repository

import (
	"database/sql"
	"errors"
	"fmt"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"

	"kas/generated"
	"kas/internal/domain"
)

// FamilyMemberRepository defines the data access contract for family membership records.
type FamilyMemberRepository interface {
	// GetByUserID retrieves the family membership for a user. Returns nil, nil if the user is not a member of any family.
	GetByUserID(userID string) (*domain.FamilyMemberDTO, error)
	// GetFamilyName retrieves the display name of a family by its ID.
	GetFamilyName(familyID string) (string, error)
	// CreateMember adds a user to a family with the specified role.
	CreateMember(app core.App, familyID string, userID string, role string) error
	// DeleteMember removes a user from their current family. Returns an error if the user is not a member.
	DeleteMember(userID string) error
}

// familyMemberRepo is the concrete PocketBase implementation of FamilyMemberRepository.
type familyMemberRepo struct {
	app core.App
}

// NewFamilyMemberRepository creates a new PocketBase-backed FamilyMemberRepository.
func NewFamilyMemberRepository(app core.App) FamilyMemberRepository {
	return &familyMemberRepo{app: app}
}

// GetByUserID retrieves the family membership for a user. Returns nil, nil if the user is not a member of any family.
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

// recordToDTO converts a raw PocketBase family_members record into a FamilyMemberDTO using proxy access.
func (r *familyMemberRepo) recordToDTO(record *core.Record) (*domain.FamilyMemberDTO, error) {
	proxy, err := generated.WrapRecord[generated.FamilyMembers](record)
	if err != nil {
		return nil, fmt.Errorf("failed to wrap family member record: %w", err)
	}

	return &domain.FamilyMemberDTO{
		ID:        proxy.Id,
		UserID:    proxy.GetString("user_id"),
		FamilyID:  proxy.GetString("family_id"),
		Role:      proxy.GetString("role"),
		CreatedAt: proxy.Created().Time(),
		UpdatedAt: proxy.Updated().Time(),
	}, nil
}

// GetFamilyName retrieves the display name of a family by its ID.
func (r *familyMemberRepo) GetFamilyName(familyID string) (string, error) {
	record, err := r.app.FindRecordById("families", familyID)
	if err != nil {
		return "", err
	}
	proxy, err := generated.WrapRecord[generated.Families](record)
	if err != nil {
		return "", fmt.Errorf("failed to wrap family record: %w", err)
	}
	return proxy.Name(), nil
}

// CreateMember adds a user to a family with the specified role.
func (r *familyMemberRepo) CreateMember(app core.App, familyID string, userID string, role string) error {
	proxy, err := generated.NewProxy[generated.FamilyMembers](app)
	if err != nil {
		return fmt.Errorf("failed to create family member record: %w", err)
	}
	proxy.Set("family_id", familyID)
	proxy.Set("user_id", userID)
	proxy.Set("role", role)
	return app.Save(proxy.Record)
}

// DeleteMember removes a user from their current family. Returns an error if the user is not a member.
func (r *familyMemberRepo) DeleteMember(userID string) error {
	record, err := r.app.FindFirstRecordByFilter(
		"family_members",
		"user_id = {:userID}",
		dbx.Params{"userID": userID},
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return errors.New("not a member of any family")
		}
		return fmt.Errorf("failed to find family member: %w", err)
	}
	if record == nil {
		return errors.New("not a member of any family")
	}
	return r.app.Delete(record)
}
