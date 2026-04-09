package repository

import (
	"fmt"
	"kas/generated"
	"kas/internal/domain"

	"github.com/pocketbase/pocketbase/core"
)

// defaultExpandFields is shared across all methods to avoid repeated allocation
var defaultExpandFields = []string{"category_id", "created_by", "family_id"}

// dateRange returns start and end date strings for a given year/month
func dateRange(year, month int) (startDate, endDate string) {
	startDate = fmt.Sprintf("%04d-%02d-01", year, month)
	nextMonth := month + 1
	nextYear := year
	if nextMonth > 12 {
		nextMonth = 1
		nextYear++
	}
	endDate = fmt.Sprintf("%04d-%02d-01", nextYear, nextMonth)
	return
}

// MonthlyReportData holds pre-aggregated monthly report data from SQL
type MonthlyReportData struct {
	TotalIncome  float64
	TotalExpense float64
	Categories   []CategoryBreakdownData
}

// CategoryBreakdownData holds per-category aggregation from SQL
type CategoryBreakdownData struct {
	CategoryID   string
	CategoryName string
	Icon         string
	Color        string
	TotalAmount  float64
	Count        int
}

// TransactionRepository interface - abstraction layer
type TransactionRepository interface {
	Create(req *domain.CreateTransactionRequest, userID string) (*domain.TransactionDTO, error)
	GetByID(id string) (*domain.TransactionDTO, error)
	GetCreatorID(id string) (string, error)
	GetByFamilyID(familyID string, limit, offset int) ([]*domain.TransactionDTO, error)
	GetByFamilyAndMonth(familyID string, year, month int) ([]*domain.TransactionDTO, error)
	Update(id string, req *domain.UpdateTransactionRequest) (*domain.TransactionDTO, error)
	Delete(id string) error
	GetTotalByFamily(familyID string) (float64, error)
	GetMonthlyStats(familyID string, year, month int) (income, expense float64, err error)
	GetMonthlyReportData(familyID string, year, month int) (*MonthlyReportData, error)
	GetDashboardData(familyID string, year, month int) (totalBalance, monthlyIncome, monthlyExpense, prevIncome, prevExpense float64, err error)
}

// transactionRepo adalah implementasi concrete
type transactionRepo struct {
	app core.App
}

// NewTransactionRepository creates new transaction repository
func NewTransactionRepository(app core.App) TransactionRepository {
	return &transactionRepo{
		app: app,
	}
}

// Create transaction - menggunakan generated proxy
func (r *transactionRepo) Create(req *domain.CreateTransactionRequest, userID string) (*domain.TransactionDTO, error) {
	collection, err := r.app.FindCachedCollectionByNameOrId("transactions")
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
	r.app.ExpandRecord(record, defaultExpandFields, nil)

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
	r.app.ExpandRecords(records, defaultExpandFields, nil)

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

func (r *transactionRepo) GetByFamilyAndMonth(familyID string, year, month int) ([]*domain.TransactionDTO, error) {
	startDate, endDate := dateRange(year, month)

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
	r.app.ExpandRecords(records, defaultExpandFields, nil)

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

// GetCreatorID returns just the creator's user ID for a transaction (lightweight, no expand)
func (r *transactionRepo) GetCreatorID(id string) (string, error) {
	record, err := r.app.FindRecordById("transactions", id)
	if err != nil {
		return "", err
	}
	return record.GetString("created_by"), nil
}

// GetTotalByFamily calculates total for a family using single CASE WHEN query
func (r *transactionRepo) GetTotalByFamily(familyID string) (float64, error) {
	var balance float64

	query := `
		SELECT 
			COALESCE(SUM(CASE WHEN type = 'income' THEN amount ELSE 0 END), 0) -
			COALESCE(SUM(CASE WHEN type = 'expense' THEN amount ELSE 0 END), 0)
		FROM transactions 
		WHERE family_id = {:familyID}
	`

	err := r.app.DB().NewQuery(query).Bind(map[string]any{"familyID": familyID}).Row(&balance)
	if err != nil {
		return 0, err
	}

	return balance, nil
}

func (r *transactionRepo) GetMonthlyStats(familyID string, year, month int) (income, expense float64, err error) {
	startDate, endDate := dateRange(year, month)

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

func (r *transactionRepo) GetMonthlyReportData(familyID string, year, month int) (*MonthlyReportData, error) {
	startDate, endDate := dateRange(year, month)

	var totalIncome, totalExpense float64
	totalsQuery := `
		SELECT
			COALESCE(SUM(CASE WHEN type = 'income' THEN amount ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN type = 'expense' THEN amount ELSE 0 END), 0)
		FROM transactions
		WHERE family_id = {:familyID}
		AND date >= {:startDate}
		AND date < {:endDate}
	`
	err := r.app.DB().NewQuery(totalsQuery).Bind(map[string]any{
		"familyID":  familyID,
		"startDate": startDate,
		"endDate":   endDate,
	}).Row(&totalIncome, &totalExpense)
	if err != nil {
		return nil, fmt.Errorf("failed to get monthly totals: %w", err)
	}

	type categoryRow struct {
		CategoryID   string  `db:"category_id"`
		CategoryName string  `db:"name"`
		Icon         string  `db:"icon"`
		Color        string  `db:"color"`
		TotalAmount  float64 `db:"total_amount"`
		Count        int     `db:"count"`
	}

	var categories []categoryRow
	breakdownQuery := `
		SELECT
			t.category_id,
			c.name,
			c.icon,
			c.color,
			SUM(t.amount) as total_amount,
			COUNT(*) as count
		FROM transactions t
		JOIN categories c ON t.category_id = c.id
		WHERE t.family_id = {:familyID}
		AND t.type = 'expense'
		AND t.date >= {:startDate}
		AND t.date < {:endDate}
		GROUP BY t.category_id, c.name, c.icon, c.color
	`
	err = r.app.DB().NewQuery(breakdownQuery).Bind(map[string]any{
		"familyID":  familyID,
		"startDate": startDate,
		"endDate":   endDate,
	}).All(&categories)
	if err != nil {
		return nil, fmt.Errorf("failed to get category breakdown: %w", err)
	}

	result := &MonthlyReportData{
		TotalIncome:  totalIncome,
		TotalExpense: totalExpense,
		Categories:   make([]CategoryBreakdownData, len(categories)),
	}
	for i, c := range categories {
		result.Categories[i] = CategoryBreakdownData{
			CategoryID:   c.CategoryID,
			CategoryName: c.CategoryName,
			Icon:         c.Icon,
			Color:        c.Color,
			TotalAmount:  c.TotalAmount,
			Count:        c.Count,
		}
	}

	return result, nil
}

func (r *transactionRepo) GetDashboardData(familyID string, year, month int) (totalBalance, monthlyIncome, monthlyExpense, prevIncome, prevExpense float64, err error) {
	startDate, endDate := dateRange(year, month)

	prevYear, prevMon := year, month-1
	if prevMon == 0 {
		prevMon = 12
		prevYear--
	}
	prevStartDate, prevEndDate := dateRange(prevYear, prevMon)

	query := `
		SELECT
			COALESCE(SUM(CASE WHEN type = 'income' THEN amount ELSE 0 END), 0) -
			COALESCE(SUM(CASE WHEN type = 'expense' THEN amount ELSE 0 END), 0) as total_balance,
			COALESCE(SUM(CASE WHEN type = 'income' AND date >= {:startDate} AND date < {:endDate} THEN amount ELSE 0 END), 0) as monthly_income,
			COALESCE(SUM(CASE WHEN type = 'expense' AND date >= {:startDate} AND date < {:endDate} THEN amount ELSE 0 END), 0) as monthly_expense,
			COALESCE(SUM(CASE WHEN type = 'income' AND date >= {:prevStartDate} AND date < {:prevEndDate} THEN amount ELSE 0 END), 0) as prev_income,
			COALESCE(SUM(CASE WHEN type = 'expense' AND date >= {:prevStartDate} AND date < {:prevEndDate} THEN amount ELSE 0 END), 0) as prev_expense
		FROM transactions
		WHERE family_id = {:familyID}
	`

	err = r.app.DB().NewQuery(query).Bind(map[string]any{
		"familyID":      familyID,
		"startDate":     startDate,
		"endDate":       endDate,
		"prevStartDate": prevStartDate,
		"prevEndDate":   prevEndDate,
	}).Row(&totalBalance, &monthlyIncome, &monthlyExpense, &prevIncome, &prevExpense)

	return
}

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
