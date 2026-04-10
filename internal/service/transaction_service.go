package service

import (
	"errors"
	"fmt"
	"kas/internal/domain"
	"kas/internal/repository"
	"time"
)

// TransactionService interface - business logic layer
type TransactionService interface {
	CreateTransaction(req *domain.CreateTransactionRequest, userID, familyID string) (*domain.TransactionDTO, error)
	GetTransaction(id string) (*domain.TransactionDTO, error)
	GetFamilyTransactions(familyID string, page, pageSize int) ([]*domain.TransactionDTO, error)
	UpdateTransaction(id, userID string, req *domain.UpdateTransactionRequest) (*domain.TransactionDTO, error)
	DeleteTransaction(id, userID string) error
	GetFamilyBalance(familyID string) (float64, error)
}

type transactionService struct {
	transactionRepo repository.TransactionRepository
	categoryRepo    repository.CategoryRepository
}

// NewTransactionService creates new transaction service
func NewTransactionService(transactionRepo repository.TransactionRepository, categoryRepo repository.CategoryRepository) TransactionService {
	return &transactionService{
		transactionRepo: transactionRepo,
		categoryRepo:    categoryRepo,
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
	if err != nil {
		return nil, fmt.Errorf("failed to validate category: %w", err)
	}
	if category == nil {
		return nil, errors.New("category not found")
	}
	if !category.IsDefault && category.FamilyID != familyID {
		return nil, errors.New("category does not belong to this family")
	}

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

	// Validate category if being updated
	if req.CategoryID != "" {
		category, err := s.categoryRepo.GetByID(req.CategoryID)
		if err != nil {
			return nil, fmt.Errorf("failed to validate category: %w", err)
		}
		if category == nil {
			return nil, errors.New("category not found")
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
