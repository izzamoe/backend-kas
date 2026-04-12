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

	toBreakdownDTO := func(src []repository.CategoryBreakdownData) []domain.CategoryBreakdownDTO {
		out := make([]domain.CategoryBreakdownDTO, len(src))
		for i, c := range src {
			out[i] = domain.CategoryBreakdownDTO{
				CategoryID:   c.CategoryID,
				CategoryName: c.CategoryName,
				Icon:         c.Icon,
				Color:        c.Color,
				TotalAmount:  c.TotalAmount,
				Count:        c.Count,
			}
		}
		return out
	}

	return &domain.MonthlyReportDTO{
		FamilyID:         req.FamilyID,
		Year:             req.Year,
		Month:            req.Month,
		TotalIncome:      data.TotalIncome,
		TotalExpense:     data.TotalExpense,
		Balance:          data.TotalIncome - data.TotalExpense,
		ExpenseBreakdown: toBreakdownDTO(data.ExpenseCategories),
		IncomeBreakdown:  toBreakdownDTO(data.IncomeCategories),
	}, nil
}

// GetDashboardSummary generates dashboard summary with total balance and monthly stats
func (s *reportService) GetDashboardSummary(req *domain.DashboardSummaryRequest) (*domain.DashboardSummaryDTO, error) {
	totalBalance, monthlyIncome, monthlyExpense, prevIncome, prevExpense, err := s.transactionRepo.GetDashboardData(req.FamilyID, req.Year, req.Month)
	if err != nil {
		return nil, err
	}

	incomeChange := calculatePercentageChange(prevIncome, monthlyIncome)
	expenseChange := calculatePercentageChange(prevExpense, monthlyExpense)

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
