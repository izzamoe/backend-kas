package service

import (
	"crypto/rand"
	"errors"
	"fmt"
	"kas/internal/domain"
	"kas/internal/repository"
	"math/big"

	"github.com/pocketbase/pocketbase/core"
)

// FamilyService interface - business logic layer for family management
type FamilyService interface {
	CreateFamily(req *domain.CreateFamilyRequest, userID string) (*domain.CreateFamilyResponse, error)
	JoinFamily(req *domain.JoinFamilyRequest, userID string) (*domain.FamilyDTO, error)
	LeaveFamily(userID string) error
}

// familyService is the concrete implementation of FamilyService.
type familyService struct {
	familyRepo       repository.FamilyRepository
	familyMemberRepo repository.FamilyMemberRepository
	app              core.App
	invalidateCache  func(string)
}

// NewFamilyService creates new family service
func NewFamilyService(familyRepo repository.FamilyRepository, familyMemberRepo repository.FamilyMemberRepository, app core.App, invalidateCache func(string)) FamilyService {
	return &familyService{
		familyRepo:       familyRepo,
		familyMemberRepo: familyMemberRepo,
		app:              app,
		invalidateCache:  invalidateCache,
	}
}

const inviteCodeChars = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789"

// generateInviteCode generates a cryptographically secure 8-character alphanumeric invite code.
func generateInviteCode() (string, error) {
	result := make([]byte, 8)
	for i := range result {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(inviteCodeChars))))
		if err != nil {
			return "", err
		}
		result[i] = inviteCodeChars[n.Int64()]
	}
	return string(result), nil
}

// CreateFamily creates a new family and adds user as owner atomically
func (s *familyService) CreateFamily(req *domain.CreateFamilyRequest, userID string) (*domain.CreateFamilyResponse, error) {
	if req.Name == "" {
		return nil, errors.New("family name cannot be empty")
	}

	existing, err := s.familyMemberRepo.GetByUserID(userID)
	if err != nil {
		return nil, fmt.Errorf("failed to check family membership: %w", err)
	}
	if existing != nil {
		return nil, errors.New("user already in a family")
	}

	inviteCode, err := generateInviteCode()
	if err != nil {
		return nil, fmt.Errorf("failed to generate invite code: %w", err)
	}

	var familyDTO *domain.FamilyDTO

	err = s.app.RunInTransaction(func(txApp core.App) error {
		dto, err := s.familyRepo.Create(txApp, req.Name, inviteCode)
		if err != nil {
			return fmt.Errorf("failed to create family: %w", err)
		}
		familyDTO = dto

		if err := s.familyMemberRepo.CreateMember(txApp, familyDTO.ID, userID, "owner"); err != nil {
			return fmt.Errorf("failed to create family member: %w", err)
		}

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create family: %w", err)
	}

	s.invalidateCache(userID)

	response := &domain.CreateFamilyResponse{
		Family: domain.FamilyDTO{
			ID:         familyDTO.ID,
			Name:       familyDTO.Name,
			InviteCode: familyDTO.InviteCode,
			CreatedAt:  familyDTO.CreatedAt,
			UpdatedAt:  familyDTO.UpdatedAt,
		},
		Member: domain.FamilyMemberDTO{
			FamilyID: familyDTO.ID,
			UserID:   userID,
			Role:     "owner",
		},
	}

	return response, nil
}

// JoinFamily adds user to a family via invite code
func (s *familyService) JoinFamily(req *domain.JoinFamilyRequest, userID string) (*domain.FamilyDTO, error) {
	existing, err := s.familyMemberRepo.GetByUserID(userID)
	if err != nil {
		return nil, fmt.Errorf("failed to check family membership: %w", err)
	}
	if existing != nil {
		return nil, errors.New("already a member of a family")
	}

	familyDTO, err := s.familyRepo.FindByInviteCode(req.InviteCode)
	if err != nil {
		return nil, fmt.Errorf("failed to find family: %w", err)
	}
	if familyDTO == nil {
		return nil, errors.New("invalid invite code")
	}

	if err := s.familyMemberRepo.CreateMember(s.app, familyDTO.ID, userID, "member"); err != nil {
		return nil, fmt.Errorf("failed to join family: %w", err)
	}

	s.invalidateCache(userID)

	return familyDTO, nil
}

// LeaveFamily removes user from their family
func (s *familyService) LeaveFamily(userID string) error {
	if err := s.familyMemberRepo.DeleteMember(userID); err != nil {
		return fmt.Errorf("failed to leave family: %w", err)
	}

	s.invalidateCache(userID)

	return nil
}
