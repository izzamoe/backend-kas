package service

import (
	"kas/internal/domain"
	"kas/internal/repository"
)

// ReportService interface - business logic for reports
type ReportService interface {
	GetMonthlyReport(req *domain.MonthlyReportRequest) (*domain.MonthlyReportDTO, error)
	GetDashboardSummary(req *domain.DashboardSummaryRequest) (*domain.DashboardSummaryDTO, error)
}

// reportService is the concrete implementation
type reportService struct {
	transactionRepo repository.TransactionRepository
}

// NewReportService creates a new report service
func NewReportService(transactionRepo repository.TransactionRepository) ReportService {
	return &reportService{
		transactionRepo: transactionRepo,
	}
}

// GetMonthlyReport generates a monthly financial report
func (s *reportService) GetMonthlyReport(req *domain.MonthlyReportRequest) (*domain.MonthlyReportDTO, error) {
	// Get all transactions for the month
	transactions, err := s.transactionRepo.GetByFamilyAndMonth(req.FamilyID, req.Year, req.Month)
	if err != nil {
		return nil, err
	}

	// Initialize report
	report := &domain.MonthlyReportDTO{
		FamilyID:          req.FamilyID,
		Year:              req.Year,
		Month:             req.Month,
		TotalIncome:       0,
		TotalExpense:      0,
		Balance:           0,
		CategoryBreakdown: []domain.CategoryBreakdownDTO{},
	}

	// Map for grouping by category (key: category_id)
	categoryMap := make(map[string]*domain.CategoryBreakdownDTO)

	// Process each transaction
	for _, tx := range transactions {
		if tx.Type == domain.TransactionTypeIncome {
			// Add to total income
			report.TotalIncome += tx.Amount
		} else {
			// Add to total expense
			report.TotalExpense += tx.Amount

			// Group by category for expense breakdown
			if tx.Category != nil {
				categoryID := tx.Category.ID
				if breakdown, exists := categoryMap[categoryID]; exists {
					// Category already exists, update totals
					breakdown.TotalAmount += tx.Amount
					breakdown.Count++
				} else {
					// New category, create entry
					categoryMap[categoryID] = &domain.CategoryBreakdownDTO{
						CategoryID:   categoryID,
						CategoryName: tx.Category.Name,
						Icon:         tx.Category.Icon,
						Color:        tx.Category.Color,
						TotalAmount:  tx.Amount,
						Count:        1,
					}
				}
			}
		}
	}

	// Calculate balance
	report.Balance = report.TotalIncome - report.TotalExpense

	// Convert map to slice
	for _, breakdown := range categoryMap {
		report.CategoryBreakdown = append(report.CategoryBreakdown, *breakdown)
	}

	return report, nil
}

// GetDashboardSummary generates dashboard summary with total balance and monthly stats (OPTIMIZED)
func (s *reportService) GetDashboardSummary(req *domain.DashboardSummaryRequest) (*domain.DashboardSummaryDTO, error) {
	// Get total balance using optimized SQL aggregation
	totalBalance, err := s.transactionRepo.GetTotalByFamily(req.FamilyID)
	if err != nil {
		return nil, err
	}

	// Get current month stats using optimized SQL aggregation
	monthlyIncome, monthlyExpense, err := s.transactionRepo.GetMonthlyStats(req.FamilyID, req.Year, req.Month)
	if err != nil {
		return nil, err
	}

	// Calculate previous month
	prevYear, prevMonth := req.Year, req.Month-1
	if prevMonth == 0 {
		prevMonth = 12
		prevYear--
	}

	// Get previous month stats using optimized SQL aggregation
	prevMonthIncome, prevMonthExpense, err := s.transactionRepo.GetMonthlyStats(req.FamilyID, prevYear, prevMonth)
	if err != nil {
		return nil, err
	}

	// Calculate percentage changes
	incomeChange := calculatePercentageChange(prevMonthIncome, monthlyIncome)
	expenseChange := calculatePercentageChange(prevMonthExpense, monthlyExpense)

	return &domain.DashboardSummaryDTO{
		TotalBalance:         totalBalance,
		MonthlyIncome:        monthlyIncome,
		MonthlyIncomeChange:  incomeChange,
		MonthlyExpense:       monthlyExpense,
		MonthlyExpenseChange: expenseChange,
	}, nil
}

// calculatePercentageChange calculates percentage change between two values
func calculatePercentageChange(oldValue, newValue float64) float64 {
	if oldValue == 0 {
		if newValue == 0 {
			return 0
		}
		return 100 // If old is 0 and new is not, 100% increase
	}
	return ((newValue - oldValue) / oldValue) * 100
}
