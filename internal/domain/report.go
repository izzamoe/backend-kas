package domain

// CategoryBreakdownDTO represents expense breakdown per category
// @Description CategoryBreakdownDTO represents a single category's financial breakdown
type CategoryBreakdownDTO struct {
	CategoryID   string  `json:"category_id" example:"cat1"`
	CategoryName string  `json:"category_name" example:"Makanan"`
	Icon         string  `json:"icon" example:"🍔"`
	Color        string  `json:"color" example:"#FF5733"`
	TotalAmount  float64 `json:"total_amount" example:"800000"`
	Count        int     `json:"count" example:"12"` // Number of transactions
}

// MonthlyReportDTO represents monthly financial report
// @Description MonthlyReportDTO represents the monthly financial report response
type MonthlyReportDTO struct {
	FamilyID         string                 `json:"family_id" example:"abc123"`
	Year             int                    `json:"year" example:"2026"`
	Month            int                    `json:"month" example:"4"`
	TotalIncome      float64                `json:"total_income" example:"5000000"`
	TotalExpense     float64                `json:"total_expense" example:"1800000"`
	Balance          float64                `json:"balance" example:"3200000"` // TotalIncome - TotalExpense
	ExpenseBreakdown []CategoryBreakdownDTO `json:"expense_breakdown"`         // Breakdown per category for expenses
	IncomeBreakdown  []CategoryBreakdownDTO `json:"income_breakdown"`          // Breakdown per category for income
}

// MonthlyReportRequest represents request parameters for monthly report
type MonthlyReportRequest struct {
	FamilyID string `json:"family_id"`
	Year     int    `json:"year"`
	Month    int    `json:"month"` // 1-12
}

// DashboardSummaryDTO represents dashboard summary with balance and monthly stats
// @Description DashboardSummaryDTO represents the dashboard summary response
type DashboardSummaryDTO struct {
	FamilyName           string  `json:"family_name" example:"Keluarga Bahagia"`
	UserName             string  `json:"user_name" example:"Budi"`
	TotalBalance         float64 `json:"total_balance" example:"12000000"`      // Overall balance
	MonthlyIncome        float64 `json:"monthly_income" example:"5000000"`      // This month's income
	MonthlyIncomeChange  float64 `json:"monthly_income_change" example:"10.5"`  // % change from prev month
	MonthlyExpense       float64 `json:"monthly_expense" example:"1800000"`     // This month's expense
	MonthlyExpenseChange float64 `json:"monthly_expense_change" example:"-5.2"` // % change from prev month
}

// DashboardSummaryRequest represents request for dashboard summary
type DashboardSummaryRequest struct {
	FamilyID string `json:"family_id"`
	Year     int    `json:"year"`
	Month    int    `json:"month"` // 1-12
}
