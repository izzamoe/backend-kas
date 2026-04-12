package service

import (
	"errors"
	"kas/internal/domain"
	"kas/internal/repository"
	"testing"

	"github.com/pocketbase/pocketbase/core"
)

// mockTransactionRepo is a mock implementation of repository.TransactionRepository.
// Each method delegates to the corresponding Fn field if non-nil, else returns zero values.
type mockTransactionRepo struct {
	createFn               func(req *domain.CreateTransactionRequest, userID, familyID string) (*domain.TransactionDTO, error)
	getByIDFn              func(id string) (*domain.TransactionDTO, error)
	getCreatorIDFn         func(id string) (string, error)
	getByFamilyIDFn        func(familyID string, limit, offset int) ([]*domain.TransactionDTO, error)
	getByFamilyAndMonthFn  func(familyID string, year, month int) ([]*domain.TransactionDTO, error)
	updateFn               func(id string, req *domain.UpdateTransactionRequest) (*domain.TransactionDTO, error)
	deleteFn               func(id string) error
	getTotalByFamilyFn     func(familyID string) (float64, error)
	getMonthlyStatsFn      func(familyID string, year, month int) (income, expense float64, err error)
	getMonthlyReportDataFn func(familyID string, year, month int) (*repository.MonthlyReportData, error)
	getDashboardDataFn     func(familyID string, year, month int) (float64, float64, float64, float64, float64, error)
}

func (m *mockTransactionRepo) Create(req *domain.CreateTransactionRequest, userID, familyID string) (*domain.TransactionDTO, error) {
	if m.createFn != nil {
		return m.createFn(req, userID, familyID)
	}
	return nil, nil
}

func (m *mockTransactionRepo) GetByID(id string) (*domain.TransactionDTO, error) {
	if m.getByIDFn != nil {
		return m.getByIDFn(id)
	}
	return nil, nil
}

func (m *mockTransactionRepo) GetCreatorID(id string) (string, error) {
	if m.getCreatorIDFn != nil {
		return m.getCreatorIDFn(id)
	}
	return "", nil
}

func (m *mockTransactionRepo) GetByFamilyID(familyID string, limit, offset int) ([]*domain.TransactionDTO, error) {
	if m.getByFamilyIDFn != nil {
		return m.getByFamilyIDFn(familyID, limit, offset)
	}
	return nil, nil
}

func (m *mockTransactionRepo) GetByFamilyAndMonth(familyID string, year, month int) ([]*domain.TransactionDTO, error) {
	if m.getByFamilyAndMonthFn != nil {
		return m.getByFamilyAndMonthFn(familyID, year, month)
	}
	return nil, nil
}

func (m *mockTransactionRepo) Update(id string, req *domain.UpdateTransactionRequest) (*domain.TransactionDTO, error) {
	if m.updateFn != nil {
		return m.updateFn(id, req)
	}
	return nil, nil
}

func (m *mockTransactionRepo) Delete(id string) error {
	if m.deleteFn != nil {
		return m.deleteFn(id)
	}
	return nil
}

func (m *mockTransactionRepo) GetTotalByFamily(familyID string) (float64, error) {
	if m.getTotalByFamilyFn != nil {
		return m.getTotalByFamilyFn(familyID)
	}
	return 0, nil
}

func (m *mockTransactionRepo) GetMonthlyStats(familyID string, year, month int) (income, expense float64, err error) {
	if m.getMonthlyStatsFn != nil {
		return m.getMonthlyStatsFn(familyID, year, month)
	}
	return 0, 0, nil
}

func (m *mockTransactionRepo) GetMonthlyReportData(familyID string, year, month int) (*repository.MonthlyReportData, error) {
	if m.getMonthlyReportDataFn != nil {
		return m.getMonthlyReportDataFn(familyID, year, month)
	}
	return &repository.MonthlyReportData{ExpenseCategories: []repository.CategoryBreakdownData{}, IncomeCategories: []repository.CategoryBreakdownData{}}, nil
}

func (m *mockTransactionRepo) GetDashboardData(familyID string, year, month int) (float64, float64, float64, float64, float64, error) {
	if m.getDashboardDataFn != nil {
		return m.getDashboardDataFn(familyID, year, month)
	}
	return 0, 0, 0, 0, 0, nil
}

// mockCategoryRepo is a mock implementation of repository.CategoryRepository.
type mockCategoryRepo struct {
	getByIDFn func(id string) (*repository.CategoryInfo, error)
}

func (m *mockCategoryRepo) GetByID(id string) (*repository.CategoryInfo, error) {
	if m.getByIDFn != nil {
		return m.getByIDFn(id)
	}
	return nil, nil
}

func (m *mockCategoryRepo) SeedMasterCategories(app core.App, familyID string) error {
	return nil
}

// ---- CreateTransaction tests ----

func TestCreateTransaction(t *testing.T) {
	tests := []struct {
		name                string
		req                 *domain.CreateTransactionRequest
		userID              string
		familyID            string
		mockCreate          func(req *domain.CreateTransactionRequest, userID, familyID string) (*domain.TransactionDTO, error)
		mockGetCategoryByID func(id string) (*repository.CategoryInfo, error)
		wantErr             bool
		errMsg              string
	}{
		{
			name: "amount zero returns error",
			req: &domain.CreateTransactionRequest{
				Amount: 0,
				Type:   domain.TransactionTypeIncome,
				Date:   "2026-01-01T00:00:00Z",
			},
			familyID: "",
			wantErr:  true,
			errMsg:   "amount must be greater than 0",
		},
		{
			name: "negative amount returns error",
			req: &domain.CreateTransactionRequest{
				Amount: -100,
				Type:   domain.TransactionTypeIncome,
				Date:   "2026-01-01T00:00:00Z",
			},
			familyID: "",
			wantErr:  true,
			errMsg:   "amount must be greater than 0",
		},
		{
			name: "invalid type returns error",
			req: &domain.CreateTransactionRequest{
				Amount: 100,
				Type:   "transfer",
				Date:   "2026-01-01T00:00:00Z",
			},
			familyID: "",
			wantErr:  true,
			errMsg:   "type must be either 'income' or 'expense'",
		},
		{
			name: "invalid date format returns error",
			req: &domain.CreateTransactionRequest{
				Amount: 100,
				Type:   domain.TransactionTypeIncome,
				Date:   "2026-01-01",
			},
			familyID: "",
			wantErr:  true,
			errMsg:   "invalid date format, use ISO 8601",
		},
		{
			name: "valid income request calls repo",
			req: &domain.CreateTransactionRequest{
				CategoryID: "cat1",
				Amount:     50000,
				Type:       domain.TransactionTypeIncome,
				Date:       "2026-01-01T00:00:00Z",
				Note:       "salary",
			},
			userID:   "user1",
			familyID: "fam1",
			mockCreate: func(req *domain.CreateTransactionRequest, userID, familyID string) (*domain.TransactionDTO, error) {
				return &domain.TransactionDTO{ID: "tx1", Amount: 50000}, nil
			},
			mockGetCategoryByID: func(id string) (*repository.CategoryInfo, error) {
				return &repository.CategoryInfo{ID: "cat1", FamilyID: "fam1", IsDefault: false}, nil
			},
			wantErr: false,
		},
		{
			name: "valid expense request calls repo",
			req: &domain.CreateTransactionRequest{
				CategoryID: "cat1",
				Amount:     20000,
				Type:       domain.TransactionTypeExpense,
				Date:       "2026-03-15T10:00:00Z",
			},
			userID:   "user1",
			familyID: "fam1",
			mockCreate: func(req *domain.CreateTransactionRequest, userID, familyID string) (*domain.TransactionDTO, error) {
				return &domain.TransactionDTO{ID: "tx2", Amount: 20000}, nil
			},
			mockGetCategoryByID: func(id string) (*repository.CategoryInfo, error) {
				return &repository.CategoryInfo{ID: "cat1", FamilyID: "fam1", IsDefault: false}, nil
			},
			wantErr: false,
		},
		{
			name: "category not found returns error",
			req: &domain.CreateTransactionRequest{
				CategoryID: "cat_missing",
				Amount:     100,
				Type:       domain.TransactionTypeIncome,
				Date:       "2026-01-01T00:00:00Z",
			},
			userID:   "user1",
			familyID: "",
			mockGetCategoryByID: func(id string) (*repository.CategoryInfo, error) {
				return nil, nil
			},
			wantErr: true,
			errMsg:  "category not found",
		},
		{
			name: "default category allowed for any family",
			req: &domain.CreateTransactionRequest{
				CategoryID: "cat_default",
				Amount:     100,
				Type:       domain.TransactionTypeIncome,
				Date:       "2026-01-01T00:00:00Z",
			},
			userID:   "user1",
			familyID: "fam1",
			mockCreate: func(req *domain.CreateTransactionRequest, userID, familyID string) (*domain.TransactionDTO, error) {
				return &domain.TransactionDTO{ID: "tx_new"}, nil
			},
			mockGetCategoryByID: func(id string) (*repository.CategoryInfo, error) {
				return &repository.CategoryInfo{ID: "cat_default", FamilyID: "fam99", IsDefault: true}, nil
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &mockTransactionRepo{createFn: tt.mockCreate}
			categoryRepo := &mockCategoryRepo{}
			if tt.mockGetCategoryByID != nil {
				categoryRepo.getByIDFn = tt.mockGetCategoryByID
			}
			svc := NewTransactionService(repo, categoryRepo)

			result, err := svc.CreateTransaction(tt.req, tt.userID, tt.familyID)

			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error but got nil")
				}
				if tt.errMsg != "" && err.Error() != tt.errMsg {
					t.Errorf("expected error %q, got %q", tt.errMsg, err.Error())
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if result == nil {
				t.Error("expected non-nil result")
			}
		})
	}
}

// ---- GetFamilyTransactions (pagination) tests ----

func TestGetFamilyTransactions_Pagination(t *testing.T) {
	tests := []struct {
		name       string
		page       int
		pageSize   int
		wantLimit  int
		wantOffset int
	}{
		{
			name:       "page less than 1 normalized to 1",
			page:       0,
			pageSize:   10,
			wantLimit:  10,
			wantOffset: 0,
		},
		{
			name:       "negative page normalized to 1",
			page:       -5,
			pageSize:   10,
			wantLimit:  10,
			wantOffset: 0,
		},
		{
			name:       "pageSize less than 1 defaults to 20",
			page:       1,
			pageSize:   0,
			wantLimit:  20,
			wantOffset: 0,
		},
		{
			name:       "pageSize greater than 100 defaults to 20",
			page:       1,
			pageSize:   200,
			wantLimit:  20,
			wantOffset: 0,
		},
		{
			name:       "page=2 pageSize=10 gives offset=10",
			page:       2,
			pageSize:   10,
			wantLimit:  10,
			wantOffset: 10,
		},
		{
			name:       "page=3 pageSize=5 gives offset=10",
			page:       3,
			pageSize:   5,
			wantLimit:  5,
			wantOffset: 10,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var capturedLimit, capturedOffset int

			repo := &mockTransactionRepo{
				getByFamilyIDFn: func(familyID string, limit, offset int) ([]*domain.TransactionDTO, error) {
					capturedLimit = limit
					capturedOffset = offset
					return []*domain.TransactionDTO{}, nil
				},
			}
			svc := NewTransactionService(repo, &mockCategoryRepo{})

			_, err := svc.GetFamilyTransactions("fam1", tt.page, tt.pageSize)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if capturedLimit != tt.wantLimit {
				t.Errorf("expected limit %d, got %d", tt.wantLimit, capturedLimit)
			}
			if capturedOffset != tt.wantOffset {
				t.Errorf("expected offset %d, got %d", tt.wantOffset, capturedOffset)
			}
		})
	}
}

// ---- UpdateTransaction tests ----

func TestUpdateTransaction(t *testing.T) {
	tests := []struct {
		name             string
		txID             string
		userID           string
		req              *domain.UpdateTransactionRequest
		mockGetCreatorID func(id string) (string, error)
		mockUpdate       func(id string, req *domain.UpdateTransactionRequest) (*domain.TransactionDTO, error)
		wantErr          bool
		errMsg           string
	}{
		{
			name:   "GetCreatorID error propagates",
			txID:   "tx99",
			userID: "user1",
			req:    &domain.UpdateTransactionRequest{},
			mockGetCreatorID: func(id string) (string, error) {
				return "", errors.New("not found")
			},
			wantErr: true,
			errMsg:  "not found",
		},
		{
			name:   "different owner returns unauthorized",
			txID:   "tx1",
			userID: "user2",
			req:    &domain.UpdateTransactionRequest{},
			mockGetCreatorID: func(id string) (string, error) {
				return "user1", nil
			},
			wantErr: true,
			errMsg:  "unauthorized: you can only update your own transactions",
		},
		{
			name:   "owner can update - dead code at line 88 never executes",
			txID:   "tx1",
			userID: "user1",
			req:    &domain.UpdateTransactionRequest{Note: "updated"},
			mockGetCreatorID: func(id string) (string, error) {
				return "user1", nil
			},
			mockUpdate: func(id string, req *domain.UpdateTransactionRequest) (*domain.TransactionDTO, error) {
				return &domain.TransactionDTO{ID: "tx1", Note: "updated"}, nil
			},
			wantErr: false,
		},
		{
			name:   "invalid type returns error",
			txID:   "tx1",
			userID: "user1",
			req:    &domain.UpdateTransactionRequest{Type: "transfer"},
			mockGetCreatorID: func(id string) (string, error) {
				return "user1", nil
			},
			wantErr: true,
			errMsg:  "type must be either 'income' or 'expense'",
		},
		{
			name:   "valid type income passes",
			txID:   "tx1",
			userID: "user1",
			req:    &domain.UpdateTransactionRequest{Type: domain.TransactionTypeIncome},
			mockGetCreatorID: func(id string) (string, error) {
				return "user1", nil
			},
			mockUpdate: func(id string, req *domain.UpdateTransactionRequest) (*domain.TransactionDTO, error) {
				return &domain.TransactionDTO{ID: "tx1"}, nil
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &mockTransactionRepo{
				getCreatorIDFn: tt.mockGetCreatorID,
				updateFn:       tt.mockUpdate,
			}
			svc := NewTransactionService(repo, &mockCategoryRepo{})

			result, err := svc.UpdateTransaction(tt.txID, tt.userID, tt.req)

			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error but got nil")
				}
				if tt.errMsg != "" && err.Error() != tt.errMsg {
					t.Errorf("expected error %q, got %q", tt.errMsg, err.Error())
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if result == nil {
				t.Error("expected non-nil result")
			}
		})
	}
}

// ---- DeleteTransaction tests ----

func TestDeleteTransaction(t *testing.T) {
	tests := []struct {
		name             string
		txID             string
		userID           string
		mockGetCreatorID func(id string) (string, error)
		mockDelete       func(id string) error
		wantErr          bool
		errMsg           string
	}{
		{
			name:   "GetCreatorID error propagates",
			txID:   "tx99",
			userID: "user1",
			mockGetCreatorID: func(id string) (string, error) {
				return "", errors.New("not found")
			},
			wantErr: true,
			errMsg:  "not found",
		},
		{
			name:   "different owner returns unauthorized",
			txID:   "tx1",
			userID: "user2",
			mockGetCreatorID: func(id string) (string, error) {
				return "user1", nil
			},
			wantErr: true,
			errMsg:  "unauthorized: you can only delete your own transactions",
		},
		{
			name:   "owner can delete",
			txID:   "tx1",
			userID: "user1",
			mockGetCreatorID: func(id string) (string, error) {
				return "user1", nil
			},
			mockDelete: func(id string) error {
				return nil
			},
			wantErr: false,
		},
		{
			name:   "repo delete error propagates",
			txID:   "tx1",
			userID: "user1",
			mockGetCreatorID: func(id string) (string, error) {
				return "user1", nil
			},
			mockDelete: func(id string) error {
				return errors.New("delete failed")
			},
			wantErr: true,
			errMsg:  "delete failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &mockTransactionRepo{
				getCreatorIDFn: tt.mockGetCreatorID,
				deleteFn:       tt.mockDelete,
			}
			svc := NewTransactionService(repo, &mockCategoryRepo{})

			err := svc.DeleteTransaction(tt.txID, tt.userID)

			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error but got nil")
				}
				if tt.errMsg != "" && err.Error() != tt.errMsg {
					t.Errorf("expected error %q, got %q", tt.errMsg, err.Error())
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}
