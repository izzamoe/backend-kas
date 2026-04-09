package repository

import (
	"fmt"
	"kas/generated"
	"kas/internal/domain"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

// TransactionRepository interface - abstraction layer
type TransactionRepository interface {
	Create(req *domain.CreateTransactionRequest, userID string) (*domain.TransactionDTO, error)
	GetByID(id string) (*domain.TransactionDTO, error)
	GetByFamilyID(familyID string, limit, offset int) ([]*domain.TransactionDTO, error)
	GetByFamilyAndMonth(familyID string, year, month int) ([]*domain.TransactionDTO, error)
	Update(id string, req *domain.UpdateTransactionRequest) (*domain.TransactionDTO, error)
	Delete(id string) error
	GetTotalByFamily(familyID string) (float64, error)
	GetMonthlyStats(familyID string, year, month int) (income, expense float64, err error)
}

// transactionRepo adalah implementasi concrete
type transactionRepo struct {
	app *pocketbase.PocketBase
}

// NewTransactionRepository creates new transaction repository
func NewTransactionRepository(app *pocketbase.PocketBase) TransactionRepository {
	return &transactionRepo{
		app: app,
	}
}

// Create transaction - menggunakan generated proxy
func (r *transactionRepo) Create(req *domain.CreateTransactionRequest, userID string) (*domain.TransactionDTO, error) {
	collection, err := r.app.FindCollectionByNameOrId("transactions")
	if err != nil {
		return nil, err
	}

	record := core.NewRecord(collection)

	// Set data menggunakan Set method dari Record
	record.Set("family_id", req.FamilyID)
	record.Set("created_by", userID)
	record.Set("category_id", req.CategoryID)
	record.Set("type", string(req.Type))
	record.Set("amount", req.Amount)
	record.Set("note", req.Note)
	record.Set("date", req.Date)

	if err := r.app.Save(record); err != nil {
		return nil, err
	}

	// Convert ke DTO menggunakan generated proxy
	return r.recordToDTO(record)
}

// GetByID - menggunakan generated proxy untuk type-safe access
func (r *transactionRepo) GetByID(id string) (*domain.TransactionDTO, error) {
	record, err := r.app.FindRecordById("transactions", id)
	if err != nil {
		return nil, err
	}

	// Expand relations manually
	expandFields := []string{"category_id", "created_by", "family_id"}
	r.app.ExpandRecord(record, expandFields, nil)

	return r.recordToDTO(record)
}

// GetByFamilyID with pagination
func (r *transactionRepo) GetByFamilyID(familyID string, limit, offset int) ([]*domain.TransactionDTO, error) {
	records, err := r.app.FindRecordsByFilter(
		"transactions",
		"family_id = {:familyID}",
		"-created",
		limit,
		offset,
		map[string]any{"familyID": familyID},
	)
	if err != nil {
		return nil, err
	}

	// Expand relations for all records
	expandFields := []string{"category_id", "created_by", "family_id"}
	r.app.ExpandRecords(records, expandFields, nil)

	dtos := make([]*domain.TransactionDTO, 0, len(records))
	for _, record := range records {
		dto, err := r.recordToDTO(record)
		if err != nil {
			return nil, fmt.Errorf("failed to convert record %s: %w", record.Id, err)
		}
		dtos = append(dtos, dto)
	}

	return dtos, nil
}

// GetByFamilyAndMonth gets all transactions for a specific month (OPTIMIZED)
func (r *transactionRepo) GetByFamilyAndMonth(familyID string, year, month int) ([]*domain.TransactionDTO, error) {
	// Create date range: YYYY-MM-01 to YYYY-MM-31
	startDate := fmt.Sprintf("%04d-%02d-01", year, month)

	// Calculate next month for upper bound
	nextMonth := month + 1
	nextYear := year
	if nextMonth > 12 {
		nextMonth = 1
		nextYear++
	}
	endDate := fmt.Sprintf("%04d-%02d-01", nextYear, nextMonth)

	// Use PocketBase filter with date comparison
	// date >= "2026-03-01" AND date < "2026-04-01"
	records, err := r.app.FindRecordsByFilter(
		"transactions",
		"family_id = {:familyID} && date >= {:startDate} && date < {:endDate}",
		"-date", // Sort by date descending
		-1,      // No limit
		0,
		map[string]any{
			"familyID":  familyID,
			"startDate": startDate,
			"endDate":   endDate,
		},
	)
	if err != nil {
		return nil, err
	}

	// Expand relations for all records
	expandFields := []string{"category_id", "created_by", "family_id"}
	r.app.ExpandRecords(records, expandFields, nil)

	dtos := make([]*domain.TransactionDTO, 0, len(records))
	for _, record := range records {
		dto, err := r.recordToDTO(record)
		if err != nil {
			return nil, fmt.Errorf("failed to convert record %s: %w", record.Id, err)
		}
		dtos = append(dtos, dto)
	}

	return dtos, nil
}

// Update transaction
func (r *transactionRepo) Update(id string, req *domain.UpdateTransactionRequest) (*domain.TransactionDTO, error) {
	record, err := r.app.FindRecordById("transactions", id)
	if err != nil {
		return nil, err
	}

	if req.CategoryID != "" {
		record.Set("category_id", req.CategoryID)
	}
	if req.Type != "" {
		record.Set("type", string(req.Type))
	}
	if req.Amount > 0 {
		record.Set("amount", req.Amount)
	}
	if req.Note != "" {
		record.Set("note", req.Note)
	}
	if req.Date != "" {
		record.Set("date", req.Date)
	}

	if err := r.app.Save(record); err != nil {
		return nil, err
	}

	return r.recordToDTO(record)
}

// Delete transaction
func (r *transactionRepo) Delete(id string) error {
	record, err := r.app.FindRecordById("transactions", id)
	if err != nil {
		return err
	}

	return r.app.Delete(record)
}

// GetTotalByFamily calculates total for a family (OPTIMIZED - using SQL aggregation)
func (r *transactionRepo) GetTotalByFamily(familyID string) (float64, error) {
	// Use raw SQL for aggregation - much faster than fetching all records
	var totalIncome, totalExpense float64

	// Query total income
	incomeQuery := "SELECT COALESCE(SUM(amount), 0) FROM transactions WHERE family_id = {:familyID} AND type = 'income'"
	err := r.app.DB().NewQuery(incomeQuery).Bind(map[string]any{"familyID": familyID}).Row(&totalIncome)
	if err != nil {
		return 0, err
	}

	// Query total expense
	expenseQuery := "SELECT COALESCE(SUM(amount), 0) FROM transactions WHERE family_id = {:familyID} AND type = 'expense'"
	err = r.app.DB().NewQuery(expenseQuery).Bind(map[string]any{"familyID": familyID}).Row(&totalExpense)
	if err != nil {
		return 0, err
	}

	return totalIncome - totalExpense, nil
}

// GetMonthlyStats calculates monthly income and expense (OPTIMIZED - using SQL aggregation)
func (r *transactionRepo) GetMonthlyStats(familyID string, year, month int) (income, expense float64, err error) {
	// Create date range: YYYY-MM-01 to YYYY-MM-31
	startDate := fmt.Sprintf("%04d-%02d-01", year, month)

	// Calculate next month for upper bound
	nextMonth := month + 1
	nextYear := year
	if nextMonth > 12 {
		nextMonth = 1
		nextYear++
	}
	endDate := fmt.Sprintf("%04d-%02d-01", nextYear, nextMonth)

	// Single query with conditional aggregation
	query := `
		SELECT 
			COALESCE(SUM(CASE WHEN type = 'income' THEN amount ELSE 0 END), 0) as total_income,
			COALESCE(SUM(CASE WHEN type = 'expense' THEN amount ELSE 0 END), 0) as total_expense
		FROM transactions 
		WHERE family_id = {:familyID} 
		AND date >= {:startDate} 
		AND date < {:endDate}
	`

	err = r.app.DB().NewQuery(query).Bind(map[string]any{
		"familyID":  familyID,
		"startDate": startDate,
		"endDate":   endDate,
	}).Row(&income, &expense)

	return income, expense, err
}

// recordToDTO converts Record to DTO using generated proxy
// Ini contoh penggunaan generated type-safe proxy!
func (r *transactionRepo) recordToDTO(record *core.Record) (*domain.TransactionDTO, error) {
	// Gunakan generated proxy untuk type-safe access
	proxy, err := generated.WrapRecord[generated.Transactions](record)
	if err != nil {
		return nil, fmt.Errorf("failed to wrap transaction record: %w", err)
	}

	// Convert enum TypeSelectType to string using proxy
	typeStr := "income"
	if proxy.Type() == generated.Expense {
		typeStr = "expense"
	}

	dto := &domain.TransactionDTO{
		ID:        proxy.Id,
		Type:      domain.TransactionType(typeStr),
		Amount:    proxy.Amount(),
		Note:      proxy.Note(),
		Date:      proxy.Date().Time(),
		CreatedAt: proxy.Created().Time(),
		UpdatedAt: proxy.Updated().Time(),
	}

	// Handle expanded family relation (nil check)
	if family := proxy.FamilyId(); family != nil {
		dto.FamilyID = family.Id
		dto.Family = &domain.FamilyExpand{
			ID:         family.Id,
			Name:       family.Name(),
			InviteCode: family.InviteCode(),
		}
	}

	// Handle expanded creator relation (nil check)
	if creator := proxy.CreatedBy(); creator != nil {
		dto.CreatedBy = creator.Id
		dto.Creator = &domain.UserExpand{
			ID:     creator.Id,
			Name:   creator.Name(),
			Avatar: creator.Avatar(),
		}
	}

	// Handle expanded category relation (nil check)
	if category := proxy.CategoryId(); category != nil {
		dto.CategoryID = category.Id
		dto.Category = &domain.CategoryExpand{
			ID:        category.Id,
			Name:      category.Name(),
			Icon:      category.Icon(),
			Color:     category.Color(),
			IsDefault: category.IsDefault(),
		}
	}

	return dto, nil
}
