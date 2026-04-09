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
	data, err := s.transactionRepo.GetMonthlyReportData(req.FamilyID, req.Year, req.Month)
	if err != nil {
		return nil, err
	}

	breakdown := make([]domain.CategoryBreakdownDTO, len(data.Categories))
	for i, c := range data.Categories {
		breakdown[i] = domain.CategoryBreakdownDTO{
			CategoryID:   c.CategoryID,
			CategoryName: c.CategoryName,
			Icon:         c.Icon,
			Color:        c.Color,
			TotalAmount:  c.TotalAmount,
			Count:        c.Count,
		}
	}

	return &domain.MonthlyReportDTO{
		FamilyID:          req.FamilyID,
		Year:              req.Year,
		Month:             req.Month,
		TotalIncome:       data.TotalIncome,
		TotalExpense:      data.TotalExpense,
		Balance:           data.TotalIncome - data.TotalExpense,
		CategoryBreakdown: breakdown,
	}, nil
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
