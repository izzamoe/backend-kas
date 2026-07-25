package service

import (
	"errors"
	"fmt"
	"time"

	"kas/internal/domain"
	"kas/internal/repository"
)

// TransactionService interface - business logic layer
type TransactionService interface {
	CreateTransaction(req *domain.CreateTransactionRequest, userID, familyID string) (*domain.TransactionDTO, error)
	GetTransaction(id string) (*domain.TransactionDTO, error)
	GetFamilyTransactions(familyID string, page, pageSize int) ([]*domain.TransactionDTO, error)
	GetTransactionsByDateRange(familyID, startDate, endDate string, page, perPage int) (*domain.TransactionListResponse, error)
	UpdateTransaction(id, userID string, req *domain.UpdateTransactionRequest) (*domain.TransactionDTO, error)
	DeleteTransaction(id, userID string) error
	GetFamilyBalance(familyID string) (float64, error)
}

type transactionService struct {
	transactionRepo repository.TransactionRepository
	categoryRepo    repository.CategoryRepository
	fetchRate       rateFetcher
}

// NewTransactionService creates new transaction service
func NewTransactionService(transactionRepo repository.TransactionRepository, categoryRepo repository.CategoryRepository) TransactionService {
	return &transactionService{
		transactionRepo: transactionRepo,
		categoryRepo:    categoryRepo,
		fetchRate:       defaultRateFetcher,
	}
}

// CreateTransaction dengan business logic validation
func (s *transactionService) CreateTransaction(req *domain.CreateTransactionRequest, userID, familyID string) (*domain.TransactionDTO, error) {
	// Business validation
	if req.Amount <= 0 {
		return nil, errors.New("amount must be greater than 0")
	}

	// Validate type
	if req.Type != domain.TransactionTypeIncome && req.Type != domain.TransactionTypeExpense {
		return nil, errors.New("type must be either 'income' or 'expense'")
	}

	// Parse date
	_, err := time.Parse(time.RFC3339, req.Date)
	if err != nil {
		return nil, errors.New("invalid date format, use ISO 8601")
	}

	// Validate category belongs to family (or is a default category)
	category, err := s.categoryRepo.GetByID(req.CategoryID)
	if errors.Is(err, domain.ErrCategoryNotFound) {
		return nil, domain.ErrCategoryNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to validate category: %w", err)
	}
	if !category.IsDefault && category.FamilyID != familyID {
		return nil, errors.New("category does not belong to this family")
	}

	// Kurs diambil dari API live; kalau gagal, dua-duanya jadi 0 dan
	// transaksi tetap dibuat.
	req.AmountUSD, req.ExchangeRate = s.convertToUSD(req.Amount)

	return s.transactionRepo.Create(req, userID, familyID)
}

// GetTransaction by ID
func (s *transactionService) GetTransaction(id string) (*domain.TransactionDTO, error) {
	return s.transactionRepo.GetByID(id)
}

// GetFamilyTransactions with pagination
func (s *transactionService) GetFamilyTransactions(familyID string, page, pageSize int) ([]*domain.TransactionDTO, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20 // default
	}

	offset := (page - 1) * pageSize
	return s.transactionRepo.GetByFamilyID(familyID, pageSize, offset)
}

// GetTransactionsByDateRange returns auth-scoped transactions for an inclusive date range.
func (s *transactionService) GetTransactionsByDateRange(familyID, startDate, endDate string, page, perPage int) (*domain.TransactionListResponse, error) {
	start, err := time.Parse(time.DateOnly, startDate)
	if err != nil {
		return nil, errors.New("invalid start date format, use YYYY-MM-DD")
	}

	end, err := time.Parse(time.DateOnly, endDate)
	if err != nil {
		return nil, errors.New("invalid end date format, use YYYY-MM-DD")
	}
	if end.Before(start) {
		return nil, errors.New("end date must be greater than or equal to start date")
	}

	if page < 1 {
		page = 1
	}
	if perPage < 1 || perPage > 100 {
		perPage = 20
	}

	offset := (page - 1) * perPage
	endExclusive := end.AddDate(0, 0, 1).Format(time.DateOnly)
	transactions, totalItems, err := s.transactionRepo.GetByFamilyDateRange(
		familyID,
		start.Format(time.DateOnly),
		endExclusive,
		perPage,
		offset,
	)
	if err != nil {
		return nil, err
	}

	totalPages := 0
	if totalItems > 0 {
		totalPages = (totalItems + perPage - 1) / perPage
	}

	return &domain.TransactionListResponse{
		Items:      transactions,
		Page:       page,
		PerPage:    perPage,
		TotalItems: totalItems,
		TotalPages: totalPages,
	}, nil
}

// UpdateTransaction with authorization
func (s *transactionService) UpdateTransaction(id, userID string, req *domain.UpdateTransactionRequest) (*domain.TransactionDTO, error) {
	creatorID, err := s.transactionRepo.GetCreatorID(id)
	if err != nil {
		return nil, err
	}

	if creatorID != userID {
		return nil, errors.New("unauthorized: you can only update your own transactions")
	}

	// Validate amount if provided (0 means "not updating amount")
	if req.Amount < 0 {
		return nil, errors.New("amount must be greater than 0")
	}

	// Validate type if provided
	if req.Type != "" && req.Type != domain.TransactionTypeIncome && req.Type != domain.TransactionTypeExpense {
		return nil, errors.New("type must be either 'income' or 'expense'")
	}

	// Amount berubah -> nilai USD ikut dihitung ulang dengan kurs terbaru
	if req.Amount > 0 {
		req.AmountUSD, req.ExchangeRate = s.convertToUSD(req.Amount)
	}

	// Validate category if being updated
	if req.CategoryID != "" {
		_, err := s.categoryRepo.GetByID(req.CategoryID)
		if errors.Is(err, domain.ErrCategoryNotFound) {
			return nil, domain.ErrCategoryNotFound
		}
		if err != nil {
			return nil, fmt.Errorf("failed to validate category: %w", err)
		}
	}

	return s.transactionRepo.Update(id, req)
}

// DeleteTransaction with authorization
func (s *transactionService) DeleteTransaction(id, userID string) error {
	creatorID, err := s.transactionRepo.GetCreatorID(id)
	if err != nil {
		return err
	}

	if creatorID != userID {
		return errors.New("unauthorized: you can only delete your own transactions")
	}

	return s.transactionRepo.Delete(id)
}

// GetFamilyBalance calculates total balance
func (s *transactionService) GetFamilyBalance(familyID string) (float64, error) {
	return s.transactionRepo.GetTotalByFamily(familyID)
}
