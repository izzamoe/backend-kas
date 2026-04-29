package service

import (
	"errors"
	"kas/internal/domain"
	_ "kas/migrations"
	"os"
	"strings"
	"testing"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

func setupFamilyServiceTestApp(t *testing.T) *pocketbase.PocketBase {
	t.Helper()

	dir, err := os.MkdirTemp("", "family_service_test_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	t.Cleanup(func() { os.RemoveAll(dir) })

	app := pocketbase.NewWithConfig(pocketbase.Config{DefaultDataDir: dir})
	if err := app.Bootstrap(); err != nil {
		t.Fatalf("failed to bootstrap app: %v", err)
	}
	t.Cleanup(func() { _ = app.ResetBootstrapState() })

	return app
}

// mockFamilyRepo is a mock implementation of repository.FamilyRepository.
type mockFamilyRepo struct {
	createFn           func(app core.App, name, inviteCode string) (*domain.FamilyDTO, error)
	findByInviteCodeFn func(code string) (*domain.FamilyDTO, error)
}

func (m *mockFamilyRepo) Create(app core.App, name, inviteCode string) (*domain.FamilyDTO, error) {
	if m.createFn != nil {
		return m.createFn(app, name, inviteCode)
	}
	return &domain.FamilyDTO{}, nil
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

func TestCreateFamilySuccessAndTransactionErrors(t *testing.T) {
	t.Run("successful create returns family and owner member", func(t *testing.T) {
		app := setupFamilyServiceTestApp(t)
		familyRepo := &mockFamilyRepo{
			createFn: func(app core.App, name, inviteCode string) (*domain.FamilyDTO, error) {
				if name != "Keluarga Test" {
					t.Fatalf("expected family name Keluarga Test, got %s", name)
				}
				if len(inviteCode) != 8 {
					t.Fatalf("expected 8 character invite code, got %q", inviteCode)
				}
				return &domain.FamilyDTO{ID: "fam1", Name: name, InviteCode: inviteCode}, nil
			},
		}
		memberCreated := false
		memberRepo := &mockFamilyMemberRepo{
			createMemberFn: func(app core.App, familyID, userID, role string) error {
				memberCreated = true
				if familyID != "fam1" || userID != "user1" || role != "owner" {
					t.Fatalf("unexpected member args: family=%s user=%s role=%s", familyID, userID, role)
				}
				return nil
			},
		}
		invalidatedUserID := ""
		svc := NewFamilyService(familyRepo, memberRepo, app, func(userID string) { invalidatedUserID = userID })

		got, err := svc.CreateFamily(&domain.CreateFamilyRequest{Name: "Keluarga Test"}, "user1")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !memberCreated {
			t.Fatal("expected owner membership to be created")
		}
		if invalidatedUserID != "user1" {
			t.Fatalf("expected cache invalidation for user1, got %s", invalidatedUserID)
		}
		if got.Family.ID != "fam1" || got.Family.Name != "Keluarga Test" || got.Member.Role != "owner" {
			t.Fatalf("unexpected response: %+v", got)
		}
	})

	t.Run("family create error is wrapped", func(t *testing.T) {
		app := setupFamilyServiceTestApp(t)
		svc := NewFamilyService(&mockFamilyRepo{
			createFn: func(app core.App, name, inviteCode string) (*domain.FamilyDTO, error) {
				return nil, errors.New("insert family failed")
			},
		}, &mockFamilyMemberRepo{}, app, func(string) {})

		_, err := svc.CreateFamily(&domain.CreateFamilyRequest{Name: "Keluarga Test"}, "user1")
		if err == nil || !strings.Contains(err.Error(), "failed to create family") {
			t.Fatalf("expected wrapped create family error, got %v", err)
		}
	})

	t.Run("member create error is wrapped", func(t *testing.T) {
		app := setupFamilyServiceTestApp(t)
		svc := NewFamilyService(&mockFamilyRepo{
			createFn: func(app core.App, name, inviteCode string) (*domain.FamilyDTO, error) {
				return &domain.FamilyDTO{ID: "fam1", Name: name, InviteCode: inviteCode}, nil
			},
		}, &mockFamilyMemberRepo{
			createMemberFn: func(app core.App, familyID, userID, role string) error {
				return errors.New("insert member failed")
			},
		}, app, func(string) {})

		_, err := svc.CreateFamily(&domain.CreateFamilyRequest{Name: "Keluarga Test"}, "user1")
		if err == nil || !strings.Contains(err.Error(), "failed to create family") {
			t.Fatalf("expected wrapped transaction error, got %v", err)
		}
	})
}

func TestGenerateInviteCode(t *testing.T) {
	for i := 0; i < 20; i++ {
		code, err := generateInviteCode()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(code) != 8 {
			t.Fatalf("expected 8 characters, got %q", code)
		}
		for _, char := range code {
			if !strings.ContainsRune(inviteCodeChars, char) {
				t.Fatalf("invite code %q contains invalid char %q", code, char)
			}
		}
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

func TestJoinFamilySuccessAndCreateMemberError(t *testing.T) {
	t.Run("successful join creates member and invalidates cache", func(t *testing.T) {
		familyRepo := &mockFamilyRepo{
			findByInviteCodeFn: func(code string) (*domain.FamilyDTO, error) {
				if code != "ABCD1234" {
					t.Fatalf("expected invite code ABCD1234, got %s", code)
				}
				return &domain.FamilyDTO{ID: "fam1", Name: "Keluarga Test", InviteCode: code}, nil
			},
		}
		memberCreated := false
		memberRepo := &mockFamilyMemberRepo{
			createMemberFn: func(app core.App, familyID, userID, role string) error {
				memberCreated = true
				if familyID != "fam1" || userID != "user1" || role != "member" {
					t.Fatalf("unexpected member args: family=%s user=%s role=%s", familyID, userID, role)
				}
				return nil
			},
		}
		invalidatedUserID := ""
		svc := NewFamilyService(familyRepo, memberRepo, nil, func(userID string) { invalidatedUserID = userID })

		got, err := svc.JoinFamily(&domain.JoinFamilyRequest{InviteCode: "ABCD1234"}, "user1")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got.ID != "fam1" || !memberCreated || invalidatedUserID != "user1" {
			t.Fatalf("unexpected join result: family=%+v memberCreated=%v invalidated=%s", got, memberCreated, invalidatedUserID)
		}
	})

	t.Run("member create error is wrapped", func(t *testing.T) {
		familyRepo := &mockFamilyRepo{
			findByInviteCodeFn: func(code string) (*domain.FamilyDTO, error) {
				return &domain.FamilyDTO{ID: "fam1", Name: "Keluarga Test", InviteCode: code}, nil
			},
		}
		memberRepo := &mockFamilyMemberRepo{
			createMemberFn: func(app core.App, familyID, userID, role string) error {
				return errors.New("insert member failed")
			},
		}
		svc := NewFamilyService(familyRepo, memberRepo, nil, func(string) {})

		_, err := svc.JoinFamily(&domain.JoinFamilyRequest{InviteCode: "ABCD1234"}, "user1")
		if err == nil || !strings.Contains(err.Error(), "failed to join family") {
			t.Fatalf("expected failed to join family error, got %v", err)
		}
	})
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
