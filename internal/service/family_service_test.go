package service

import (
	"errors"
	"kas/internal/domain"
	"strings"
	"testing"

	"github.com/pocketbase/pocketbase/core"
)

// mockFamilyRepo is a mock implementation of repository.FamilyRepository.
type mockFamilyRepo struct {
	createFn           func(app core.App, name, inviteCode string) (*core.Record, error)
	findByInviteCodeFn func(code string) (*domain.FamilyDTO, error)
}

func (m *mockFamilyRepo) Create(app core.App, name, inviteCode string) (*core.Record, error) {
	if m.createFn != nil {
		return m.createFn(app, name, inviteCode)
	}
	return nil, nil
}

func (m *mockFamilyRepo) FindByInviteCode(code string) (*domain.FamilyDTO, error) {
	if m.findByInviteCodeFn != nil {
		return m.findByInviteCodeFn(code)
	}
	return nil, nil
}

// mockFamilyMemberRepo is a mock implementation of repository.FamilyMemberRepository.
type mockFamilyMemberRepo struct {
	getByUserIDFn   func(userID string) (*domain.FamilyMemberDTO, error)
	getFamilyNameFn func(familyID string) (string, error)
	createMemberFn  func(app core.App, familyID, userID, role string) error
	deleteMemberFn  func(userID string) error
}

func (m *mockFamilyMemberRepo) GetByUserID(userID string) (*domain.FamilyMemberDTO, error) {
	if m.getByUserIDFn != nil {
		return m.getByUserIDFn(userID)
	}
	return nil, nil
}

func (m *mockFamilyMemberRepo) GetFamilyName(familyID string) (string, error) {
	if m.getFamilyNameFn != nil {
		return m.getFamilyNameFn(familyID)
	}
	return "", nil
}

func (m *mockFamilyMemberRepo) CreateMember(app core.App, familyID, userID, role string) error {
	if m.createMemberFn != nil {
		return m.createMemberFn(app, familyID, userID, role)
	}
	return nil
}

func (m *mockFamilyMemberRepo) DeleteMember(userID string) error {
	if m.deleteMemberFn != nil {
		return m.deleteMemberFn(userID)
	}
	return nil
}

// ---- CreateFamily tests ----

func TestCreateFamily(t *testing.T) {
	tests := []struct {
		name          string
		req           *domain.CreateFamilyRequest
		userID        string
		getByUserIDFn func(userID string) (*domain.FamilyMemberDTO, error)
		wantErr       bool
		errMsg        string
		exactMatch    bool
	}{
		{
			name:       "empty name returns error",
			req:        &domain.CreateFamilyRequest{Name: ""},
			userID:     "user1",
			wantErr:    true,
			errMsg:     "family name cannot be empty",
			exactMatch: true,
		},
		{
			name:   "user already in a family returns error",
			req:    &domain.CreateFamilyRequest{Name: "My Family"},
			userID: "user1",
			getByUserIDFn: func(userID string) (*domain.FamilyMemberDTO, error) {
				return &domain.FamilyMemberDTO{UserID: userID, FamilyID: "fam1", Role: "member"}, nil
			},
			wantErr:    true,
			errMsg:     "user already in a family",
			exactMatch: true,
		},
		{
			name:   "GetByUserID repo error propagates",
			req:    &domain.CreateFamilyRequest{Name: "My Family"},
			userID: "user1",
			getByUserIDFn: func(userID string) (*domain.FamilyMemberDTO, error) {
				return nil, errors.New("db error")
			},
			wantErr:    true,
			errMsg:     "failed to check family membership",
			exactMatch: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			memberRepo := &mockFamilyMemberRepo{}
			if tt.getByUserIDFn != nil {
				memberRepo.getByUserIDFn = tt.getByUserIDFn
			}
			svc := NewFamilyService(&mockFamilyRepo{}, memberRepo, nil, func(string) {})

			_, err := svc.CreateFamily(tt.req, tt.userID)

			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error but got nil")
				}
				if tt.errMsg != "" {
					if tt.exactMatch {
						if err.Error() != tt.errMsg {
							t.Errorf("expected error %q, got %q", tt.errMsg, err.Error())
						}
					} else {
						if !strings.Contains(err.Error(), tt.errMsg) {
							t.Errorf("expected error containing %q, got %q", tt.errMsg, err.Error())
						}
					}
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

// ---- JoinFamily tests ----

func TestJoinFamily(t *testing.T) {
	tests := []struct {
		name               string
		req                *domain.JoinFamilyRequest
		userID             string
		getByUserIDFn      func(userID string) (*domain.FamilyMemberDTO, error)
		findByInviteCodeFn func(code string) (*domain.FamilyDTO, error)
		wantErr            bool
		errMsg             string
		exactMatch         bool
	}{
		{
			name:   "already a member returns error",
			req:    &domain.JoinFamilyRequest{InviteCode: "ABCD1234"},
			userID: "user1",
			getByUserIDFn: func(userID string) (*domain.FamilyMemberDTO, error) {
				return &domain.FamilyMemberDTO{UserID: userID, FamilyID: "fam1", Role: "member"}, nil
			},
			wantErr:    true,
			errMsg:     "already a member of a family",
			exactMatch: true,
		},
		{
			name:   "invalid invite code returns error",
			req:    &domain.JoinFamilyRequest{InviteCode: "BADCODE"},
			userID: "user1",
			getByUserIDFn: func(userID string) (*domain.FamilyMemberDTO, error) {
				return nil, nil
			},
			findByInviteCodeFn: func(code string) (*domain.FamilyDTO, error) {
				return nil, nil
			},
			wantErr:    true,
			errMsg:     "invalid invite code",
			exactMatch: true,
		},
		{
			name:   "FindByInviteCode repo error propagates",
			req:    &domain.JoinFamilyRequest{InviteCode: "BADCODE"},
			userID: "user1",
			getByUserIDFn: func(userID string) (*domain.FamilyMemberDTO, error) {
				return nil, nil
			},
			findByInviteCodeFn: func(code string) (*domain.FamilyDTO, error) {
				return nil, errors.New("db error")
			},
			wantErr:    true,
			errMsg:     "failed to find family",
			exactMatch: false,
		},
		{
			name:   "GetByUserID error propagates in join",
			req:    &domain.JoinFamilyRequest{InviteCode: "ABCD1234"},
			userID: "user1",
			getByUserIDFn: func(userID string) (*domain.FamilyMemberDTO, error) {
				return nil, errors.New("db error")
			},
			wantErr:    true,
			errMsg:     "failed to check family membership",
			exactMatch: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			memberRepo := &mockFamilyMemberRepo{}
			if tt.getByUserIDFn != nil {
				memberRepo.getByUserIDFn = tt.getByUserIDFn
			}
			familyRepo := &mockFamilyRepo{}
			if tt.findByInviteCodeFn != nil {
				familyRepo.findByInviteCodeFn = tt.findByInviteCodeFn
			}
			svc := NewFamilyService(familyRepo, memberRepo, nil, func(string) {})

			_, err := svc.JoinFamily(tt.req, tt.userID)

			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error but got nil")
				}
				if tt.errMsg != "" {
					if tt.exactMatch {
						if err.Error() != tt.errMsg {
							t.Errorf("expected error %q, got %q", tt.errMsg, err.Error())
						}
					} else {
						if !strings.Contains(err.Error(), tt.errMsg) {
							t.Errorf("expected error containing %q, got %q", tt.errMsg, err.Error())
						}
					}
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

// ---- LeaveFamily tests ----

func TestLeaveFamily(t *testing.T) {
	t.Run("DeleteMember error propagates", func(t *testing.T) {
		memberRepo := &mockFamilyMemberRepo{
			deleteMemberFn: func(userID string) error {
				return errors.New("db delete error")
			},
		}
		svc := NewFamilyService(&mockFamilyRepo{}, memberRepo, nil, func(string) {})

		err := svc.LeaveFamily("user1")

		if err == nil {
			t.Fatalf("expected error but got nil")
		}
		if !strings.Contains(err.Error(), "failed to leave family") {
			t.Errorf("expected error containing %q, got %q", "failed to leave family", err.Error())
		}
	})

	t.Run("successful leave calls invalidateCache", func(t *testing.T) {
		memberRepo := &mockFamilyMemberRepo{
			deleteMemberFn: func(userID string) error {
				return nil
			},
		}
		called := false
		svc := NewFamilyService(&mockFamilyRepo{}, memberRepo, nil, func(string) { called = true })

		err := svc.LeaveFamily("user1")

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !called {
			t.Error("expected invalidateCache to be called")
		}
	})
}
