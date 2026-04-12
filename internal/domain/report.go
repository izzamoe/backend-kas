package domain

// CategoryBreakdownDTO represents expense breakdown per category
type CategoryBreakdownDTO struct {
	CategoryID   string  `json:"category_id"`
	CategoryName string  `json:"category_name"`
	Icon         string  `json:"icon"`
	Color        string  `json:"color"`
	TotalAmount  float64 `json:"total_amount"`
	Count        int     `json:"count"` // Number of transactions
}

// MonthlyReportDTO represents monthly financial report
type MonthlyReportDTO struct {
	FamilyID         string                 `json:"family_id"`
	Year             int                    `json:"year"`
	Month            int                    `json:"month"`
	TotalIncome      float64                `json:"total_income"`
	TotalExpense     float64                `json:"total_expense"`
	Balance          float64                `json:"balance"`           // TotalIncome - TotalExpense
	ExpenseBreakdown []CategoryBreakdownDTO `json:"expense_breakdown"` // Breakdown per category for expenses
	IncomeBreakdown  []CategoryBreakdownDTO `json:"income_breakdown"`  // Breakdown per category for income
}

// MonthlyReportRequest represents request parameters for monthly report
type MonthlyReportRequest struct {
	FamilyID string `json:"family_id"`
	Year     int    `json:"year"`
	Month    int    `json:"month"` // 1-12
}

// DashboardSummaryDTO represents dashboard summary with balance and monthly stats
type DashboardSummaryDTO struct {
	FamilyName           string  `json:"family_name"`
	UserName             string  `json:"user_name"`
	TotalBalance         float64 `json:"total_balance"`          // Overall balance
	MonthlyIncome        float64 `json:"monthly_income"`         // This month's income
	MonthlyIncomeChange  float64 `json:"monthly_income_change"`  // % change from prev month
	MonthlyExpense       float64 `json:"monthly_expense"`        // This month's expense
	MonthlyExpenseChange float64 `json:"monthly_expense_change"` // % change from prev month
}

// DashboardSummaryRequest represents request for dashboard summary
type DashboardSummaryRequest struct {
	FamilyID string `json:"family_id"`
	Year     int    `json:"year"`
	Month    int    `json:"month"` // 1-12
}
