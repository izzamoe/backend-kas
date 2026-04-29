package service

import (
	"errors"
	"kas/internal/domain"
	"kas/internal/repository"
	"testing"
)

// ---- GetMonthlyReport tests ----

func TestGetMonthlyReport(t *testing.T) {
	tests := []struct {
		name                    string
		reportData              *repository.MonthlyReportData
		wantIncome              float64
		wantExpense             float64
		wantBalance             float64
		wantExpenseBreakLen     int
		wantIncomeBreakLen      int
		wantExpenseBreakAmounts map[string]float64
		wantExpenseBreakCounts  map[string]int
		wantIncomeBreakAmounts  map[string]float64
		wantIncomeBreakCounts   map[string]int
	}{
		{
			name: "empty list returns all zeros and empty breakdowns",
			reportData: &repository.MonthlyReportData{
				TotalIncome:       0,
				TotalExpense:      0,
				ExpenseCategories: []repository.CategoryBreakdownData{},
				IncomeCategories:  []repository.CategoryBreakdownData{},
			},
			wantIncome:          0,
			wantExpense:         0,
			wantBalance:         0,
			wantExpenseBreakLen: 0,
			wantIncomeBreakLen:  0,
		},
		{
			name: "mixed income and expense correct totals and breakdowns",
			reportData: &repository.MonthlyReportData{
				TotalIncome:  150000,
				TotalExpense: 50000,
				ExpenseCategories: []repository.CategoryBreakdownData{
					{CategoryID: "cat1", CategoryName: "Food", Icon: "🍔", Color: "#FF0000", TotalAmount: 30000, Count: 1},
					{CategoryID: "cat2", CategoryName: "Transport", Icon: "🚗", Color: "#0000FF", TotalAmount: 20000, Count: 1},
				},
				IncomeCategories: []repository.CategoryBreakdownData{
					{CategoryID: "cat3", CategoryName: "Salary", Icon: "💼", Color: "#00FF00", TotalAmount: 150000, Count: 1},
				},
			},
			wantIncome:          150000,
			wantExpense:         50000,
			wantBalance:         100000,
			wantExpenseBreakLen: 2,
			wantIncomeBreakLen:  1,
			wantIncomeBreakAmounts: map[string]float64{
				"cat3": 150000,
			},
			wantIncomeBreakCounts: map[string]int{
				"cat3": 1,
			},
		},
		{
			name: "multiple expenses same category grouped correctly",
			reportData: &repository.MonthlyReportData{
				TotalIncome:  0,
				TotalExpense: 35000,
				ExpenseCategories: []repository.CategoryBreakdownData{
					{CategoryID: "cat1", CategoryName: "Food", Icon: "🍔", Color: "#FF0000", TotalAmount: 35000, Count: 3},
				},
				IncomeCategories: []repository.CategoryBreakdownData{},
			},
			wantIncome:          0,
			wantExpense:         35000,
			wantBalance:         -35000,
			wantExpenseBreakLen: 1,
			wantIncomeBreakLen:  0,
			wantExpenseBreakAmounts: map[string]float64{
				"cat1": 35000,
			},
			wantExpenseBreakCounts: map[string]int{
				"cat1": 3,
			},
		},
		{
			name: "expense with nil category not included in breakdown",
			reportData: &repository.MonthlyReportData{
				TotalIncome:       0,
				TotalExpense:      30000,
				ExpenseCategories: []repository.CategoryBreakdownData{},
				IncomeCategories:  []repository.CategoryBreakdownData{},
			},
			wantIncome:          0,
			wantExpense:         30000,
			wantBalance:         -30000,
			wantExpenseBreakLen: 0,
			wantIncomeBreakLen:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &mockTransactionRepo{
				getMonthlyReportDataFn: func(familyID string, year, month int) (*repository.MonthlyReportData, error) {
					return tt.reportData, nil
				},
			}
			svc := NewReportService(repo)

			req := &domain.MonthlyReportRequest{FamilyID: "fam1", Year: 2026, Month: 3}
			report, err := svc.GetMonthlyReport(req)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if report.TotalIncome != tt.wantIncome {
				t.Errorf("TotalIncome: expected %v, got %v", tt.wantIncome, report.TotalIncome)
			}
			if report.TotalExpense != tt.wantExpense {
				t.Errorf("TotalExpense: expected %v, got %v", tt.wantExpense, report.TotalExpense)
			}
			if report.Balance != tt.wantBalance {
				t.Errorf("Balance: expected %v, got %v", tt.wantBalance, report.Balance)
			}
			if len(report.ExpenseBreakdown) != tt.wantExpenseBreakLen {
				t.Errorf("ExpenseBreakdown length: expected %d, got %d", tt.wantExpenseBreakLen, len(report.ExpenseBreakdown))
			}
			if len(report.IncomeBreakdown) != tt.wantIncomeBreakLen {
				t.Errorf("IncomeBreakdown length: expected %d, got %d", tt.wantIncomeBreakLen, len(report.IncomeBreakdown))
			}

			if tt.wantExpenseBreakAmounts != nil {
				breakMap := make(map[string]domain.CategoryBreakdownDTO)
				for _, b := range report.ExpenseBreakdown {
					breakMap[b.CategoryID] = b
				}
				for catID, wantAmt := range tt.wantExpenseBreakAmounts {
					b, ok := breakMap[catID]
					if !ok {
						t.Errorf("expense category %q missing from breakdown", catID)
						continue
					}
					if b.TotalAmount != wantAmt {
						t.Errorf("expense category %q TotalAmount: expected %v, got %v", catID, wantAmt, b.TotalAmount)
					}
				}
			}

			if tt.wantExpenseBreakCounts != nil {
				breakMap := make(map[string]domain.CategoryBreakdownDTO)
				for _, b := range report.ExpenseBreakdown {
					breakMap[b.CategoryID] = b
				}
				for catID, wantCnt := range tt.wantExpenseBreakCounts {
					b, ok := breakMap[catID]
					if !ok {
						t.Errorf("expense category %q missing from breakdown", catID)
						continue
					}
					if b.Count != wantCnt {
						t.Errorf("expense category %q Count: expected %d, got %d", catID, wantCnt, b.Count)
					}
				}
			}

			if tt.wantIncomeBreakAmounts != nil {
				breakMap := make(map[string]domain.CategoryBreakdownDTO)
				for _, b := range report.IncomeBreakdown {
					breakMap[b.CategoryID] = b
				}
				for catID, wantAmt := range tt.wantIncomeBreakAmounts {
					b, ok := breakMap[catID]
					if !ok {
						t.Errorf("income category %q missing from breakdown", catID)
						continue
					}
					if b.TotalAmount != wantAmt {
						t.Errorf("income category %q TotalAmount: expected %v, got %v", catID, wantAmt, b.TotalAmount)
					}
				}
			}

			if tt.wantIncomeBreakCounts != nil {
				breakMap := make(map[string]domain.CategoryBreakdownDTO)
				for _, b := range report.IncomeBreakdown {
					breakMap[b.CategoryID] = b
				}
				for catID, wantCnt := range tt.wantIncomeBreakCounts {
					b, ok := breakMap[catID]
					if !ok {
						t.Errorf("income category %q missing from breakdown", catID)
						continue
					}
					if b.Count != wantCnt {
						t.Errorf("income category %q Count: expected %d, got %d", catID, wantCnt, b.Count)
					}
				}
			}
		})
	}
}

func TestGetMonthlyReportRepoError(t *testing.T) {
	repo := &mockTransactionRepo{
		getMonthlyReportDataFn: func(familyID string, year, month int) (*repository.MonthlyReportData, error) {
			return nil, errors.New("monthly query failed")
		},
	}
	svc := NewReportService(repo)

	_, err := svc.GetMonthlyReport(&domain.MonthlyReportRequest{FamilyID: "fam1", Year: 2026, Month: 3})
	if err == nil || err.Error() != "monthly query failed" {
		t.Fatalf("expected monthly query failed error, got %v", err)
	}
}

// ---- calculatePercentageChange tests ----

func TestCalculatePercentageChange(t *testing.T) {
	tests := []struct {
		name     string
		old      float64
		new      float64
		expected float64
	}{
		{
			name:     "both zero returns 0",
			old:      0,
			new:      0,
			expected: 0,
		},
		{
			name:     "old zero new non-zero returns 100",
			old:      0,
			new:      100,
			expected: 100,
		},
		{
			name:     "decrease 100 to 50 returns -50",
			old:      100,
			new:      50,
			expected: -50,
		},
		{
			name:     "increase 100 to 150 returns 50",
			old:      100,
			new:      150,
			expected: 50,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := calculatePercentageChange(tt.old, tt.new)
			if result != tt.expected {
				t.Errorf("calculatePercentageChange(%v, %v): expected %v, got %v", tt.old, tt.new, tt.expected, result)
			}
		})
	}
}

// ---- GetDashboardSummary tests ----

func TestGetDashboardSummary(t *testing.T) {
	tests := []struct {
		name               string
		getDashboardDataFn func(familyID string, year, month int) (float64, float64, float64, float64, float64, error)
		wantTotalBalance   float64
		wantMonthlyIncome  float64
		wantMonthlyExpense float64
		wantIncomeChange   float64
		wantExpenseChange  float64
	}{
		{
			name: "zero prev month gives 100% change when current non-zero",
			getDashboardDataFn: func(familyID string, year, month int) (float64, float64, float64, float64, float64, error) {
				return 500, 100, 50, 0, 0, nil
			},
			wantTotalBalance:   500,
			wantMonthlyIncome:  100,
			wantMonthlyExpense: 50,
			wantIncomeChange:   100,
			wantExpenseChange:  100,
		},
		{
			name: "income decreased by half",
			getDashboardDataFn: func(familyID string, year, month int) (float64, float64, float64, float64, float64, error) {
				return 1000, 50, 200, 100, 200, nil
			},
			wantTotalBalance:   1000,
			wantMonthlyIncome:  50,
			wantMonthlyExpense: 200,
			wantIncomeChange:   -50,
			wantExpenseChange:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &mockTransactionRepo{
				getDashboardDataFn: tt.getDashboardDataFn,
			}
			svc := NewReportService(repo)

			req := &domain.DashboardSummaryRequest{FamilyID: "fam1", Year: 2026, Month: 3}
			summary, err := svc.GetDashboardSummary(req)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if summary.TotalBalance != tt.wantTotalBalance {
				t.Errorf("TotalBalance: expected %v, got %v", tt.wantTotalBalance, summary.TotalBalance)
			}
			if summary.MonthlyIncome != tt.wantMonthlyIncome {
				t.Errorf("MonthlyIncome: expected %v, got %v", tt.wantMonthlyIncome, summary.MonthlyIncome)
			}
			if summary.MonthlyExpense != tt.wantMonthlyExpense {
				t.Errorf("MonthlyExpense: expected %v, got %v", tt.wantMonthlyExpense, summary.MonthlyExpense)
			}
			if summary.MonthlyIncomeChange != tt.wantIncomeChange {
				t.Errorf("MonthlyIncomeChange: expected %v, got %v", tt.wantIncomeChange, summary.MonthlyIncomeChange)
			}
			if summary.MonthlyExpenseChange != tt.wantExpenseChange {
				t.Errorf("MonthlyExpenseChange: expected %v, got %v", tt.wantExpenseChange, summary.MonthlyExpenseChange)
			}
		})
	}
}

func TestGetDashboardSummaryRepoError(t *testing.T) {
	repo := &mockTransactionRepo{
		getDashboardDataFn: func(familyID string, year, month int) (float64, float64, float64, float64, float64, error) {
			return 0, 0, 0, 0, 0, errors.New("dashboard query failed")
		},
	}
	svc := NewReportService(repo)

	_, err := svc.GetDashboardSummary(&domain.DashboardSummaryRequest{FamilyID: "fam1", Year: 2026, Month: 3})
	if err == nil || err.Error() != "dashboard query failed" {
		t.Fatalf("expected dashboard query failed error, got %v", err)
	}
}
