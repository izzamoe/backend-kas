package service

import (
	"errors"
	"fmt"
	"kas/internal/domain"
	digiflazzdomain "kas/internal/domain/digiflazz"
	"kas/internal/repository"
	digiflazzmapper "kas/internal/service/digiflazz"
	"strings"
	"time"
)

// digiflazzOrderFinalizationRepository extends the order repository with
// cross-family lookup and transaction-linking capabilities needed for finalization.
type digiflazzOrderFinalizationRepository interface {
	GetByIDAny(id string) (*digiflazzdomain.OrderDTO, error)
	LinkTransactionIfEmpty(familyID, id, transactionID string) (*digiflazzdomain.OrderDTO, error)
}

type digiflazzCategoryLookupRepository interface {
	FindByFamilyNameAndType(familyID, name, txType string) (*repository.CategoryInfo, error)
}

type digiflazzTransactionLookupRepository interface {
	FindByFamilyUserCategoryAmountNote(familyID, userID, categoryID string, amount float64, note string) (*domain.TransactionDTO, error)
}

func (s *digiflazzOrderService) FinalizeSuccessOrder(orderID string) (*digiflazzdomain.OrderDTO, error) {
	orderID = strings.TrimSpace(orderID)
	if orderID == "" {
		return nil, errors.New("order id is required")
	}
	orderRepo, ok := s.orderRepo.(digiflazzOrderFinalizationRepository)
	if !ok {
		return nil, errors.New("digiflazz order repository does not support finalization")
	}
	order, err := orderRepo.GetByIDAny(orderID)
	if err != nil {
		return nil, err
	}
	if order == nil {
		return nil, errors.New("digiflazz order not found")
	}
	return s.finalizeSuccessOrder(order)
}

func (s *digiflazzOrderService) finalizeSuccessOrderIfConfigured(order *digiflazzdomain.OrderDTO) (*digiflazzdomain.OrderDTO, error) {
	if order == nil || order.Status != digiflazzdomain.OrderStatusSuccess || order.TransactionID != "" || !s.hasFinalizationDependencies() {
		return order, nil
	}
	return s.finalizeSuccessOrder(order)
}

func (s *digiflazzOrderService) hasFinalizationDependencies() bool {
	if s.transactionRepo == nil || s.categoryRepo == nil || s.orderRepo == nil {
		return false
	}
	if _, ok := s.orderRepo.(digiflazzOrderFinalizationRepository); !ok {
		return false
	}
	if _, ok := s.categoryRepo.(digiflazzCategoryLookupRepository); !ok {
		return false
	}
	if _, ok := s.transactionRepo.(digiflazzTransactionLookupRepository); !ok {
		return false
	}
	return true
}

func (s *digiflazzOrderService) finalizeSuccessOrder(order *digiflazzdomain.OrderDTO) (*digiflazzdomain.OrderDTO, error) {
	if order == nil {
		return nil, errors.New("digiflazz order is required")
	}
	if order.Status != digiflazzdomain.OrderStatusSuccess || order.TransactionID != "" {
		return order, nil
	}
	if !s.hasFinalizationDependencies() {
		return nil, errors.New("digiflazz order finalization dependencies are required")
	}

	orderRepo := s.orderRepo.(digiflazzOrderFinalizationRepository)
	transactionLookup := s.transactionRepo.(digiflazzTransactionLookupRepository)

	latest, err := orderRepo.GetByIDAny(order.ID)
	if err != nil {
		return nil, err
	}
	if latest == nil {
		return nil, errors.New("digiflazz order not found")
	}
	if latest.Status != digiflazzdomain.OrderStatusSuccess || latest.TransactionID != "" {
		return latest, nil
	}

	category, err := s.resolveDigiflazzExpenseCategory(latest)
	if err != nil {
		return nil, err
	}
	amount := latest.SellingPrice
	if amount <= 0 {
		return nil, errors.New("digiflazz order selling price is required")
	}
	userID := strings.TrimSpace(latest.CreatedBy)
	if userID == "" {
		return nil, errors.New("digiflazz order creator is required")
	}
	description := digiflazzExpenseDescription(latest)

	existing, err := transactionLookup.FindByFamilyUserCategoryAmountNote(latest.FamilyID, userID, category.ID, amount, description)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return s.linkFinalizedTransaction(orderRepo, latest, existing.ID)
	}

	transaction, err := s.transactionRepo.Create(&domain.CreateTransactionRequest{
		CategoryID: category.ID,
		Type:       domain.TransactionTypeExpense,
		Amount:     amount,
		Note:       description,
		Date:       time.Now().UTC().Format(time.RFC3339),
	}, userID, latest.FamilyID)
	if err != nil {
		return nil, err
	}
	if transaction == nil || transaction.ID == "" {
		return nil, errors.New("failed to create digiflazz expense transaction")
	}

	linked, err := s.linkFinalizedTransaction(orderRepo, latest, transaction.ID)
	if err != nil {
		return s.recoverFinalizedOrder(orderRepo, transactionLookup, latest, category.ID, amount, description, err)
	}
	return linked, nil
}

func (s *digiflazzOrderService) resolveDigiflazzExpenseCategory(order *digiflazzdomain.OrderDTO) (*repository.CategoryInfo, error) {
	categoryRepo := s.categoryRepo.(digiflazzCategoryLookupRepository)
	categoryName := mappedDigiflazzExpenseCategoryName(order)
	category, err := categoryRepo.FindByFamilyNameAndType(order.FamilyID, categoryName, string(domain.TransactionTypeExpense))
	if err != nil {
		return nil, err
	}
	if category == nil && categoryName != "Lainnya" {
		category, err = categoryRepo.FindByFamilyNameAndType(order.FamilyID, "Lainnya", string(domain.TransactionTypeExpense))
		if err != nil {
			return nil, err
		}
	}
	if category == nil {
		return nil, fmt.Errorf("expense category %q not found for family", categoryName)
	}
	return category, nil
}

func mappedDigiflazzExpenseCategoryName(order *digiflazzdomain.OrderDTO) string {
	categoryName, _ := digiflazzmapper.MapDigiflazzCategory(order.ProductCategory)
	if categoryName != "" && categoryName != "Lainnya" {
		return categoryName
	}
	brandName, _ := digiflazzmapper.MapDigiflazzCategory(order.ProductBrand)
	if brandName != "" && brandName != "Lainnya" {
		return brandName
	}
	if categoryName != "" {
		return categoryName
	}
	return "Lainnya"
}

func digiflazzExpenseDescription(order *digiflazzdomain.OrderDTO) string {
	productName := strings.TrimSpace(order.ProductName)
	if productName == "" {
		productName = strings.TrimSpace(order.ProductCode)
	}
	return fmt.Sprintf("Pembelian %s - %s", productName, strings.TrimSpace(order.CustomerNo))
}

func (s *digiflazzOrderService) linkFinalizedTransaction(orderRepo digiflazzOrderFinalizationRepository, order *digiflazzdomain.OrderDTO, transactionID string) (*digiflazzdomain.OrderDTO, error) {
	linked, err := orderRepo.LinkTransactionIfEmpty(order.FamilyID, order.ID, transactionID)
	if err != nil {
		return nil, err
	}
	if linked == nil {
		return nil, errors.New("digiflazz order not found")
	}
	return linked, nil
}

func (s *digiflazzOrderService) recoverFinalizedOrder(orderRepo digiflazzOrderFinalizationRepository, transactionLookup digiflazzTransactionLookupRepository, order *digiflazzdomain.OrderDTO, categoryID string, amount float64, description string, linkErr error) (*digiflazzdomain.OrderDTO, error) {
	latest, err := orderRepo.GetByIDAny(order.ID)
	if err != nil {
		return nil, err
	}
	if latest != nil && latest.TransactionID != "" {
		return latest, nil
	}
	existing, findErr := transactionLookup.FindByFamilyUserCategoryAmountNote(order.FamilyID, order.CreatedBy, categoryID, amount, description)
	if findErr != nil {
		return nil, findErr
	}
	if existing != nil {
		linked, retryErr := s.linkFinalizedTransaction(orderRepo, order, existing.ID)
		if retryErr == nil {
			return linked, nil
		}
	}
	return nil, linkErr
}
