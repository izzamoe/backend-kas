package service

import (
	"kas/internal/domain"
	"kas/internal/repository"
	"testing"
)

// ---- GetMonthlyReport tests ----

func TestGetMonthlyReport(t *testing.T) {
	tests := []struct {
		name             string
		reportData       *repository.MonthlyReportData
		wantIncome       float64
		wantExpense      float64
		wantBalance      float64
		wantBreakLen     int
		wantBreakAmounts map[string]float64
		wantBreakCounts  map[string]int
	}{
		{
			name: "empty list returns all zeros and empty breakdown",
			reportData: &repository.MonthlyReportData{
				TotalIncome:  0,
				TotalExpense: 0,
				Categories:   []repository.CategoryBreakdownData{},
			},
			wantIncome:   0,
			wantExpense:  0,
			wantBalance:  0,
			wantBreakLen: 0,
		},
		{
			name: "mixed income and expense correct totals",
			reportData: &repository.MonthlyReportData{
				TotalIncome:  150000,
				TotalExpense: 50000,
				Categories: []repository.CategoryBreakdownData{
					{CategoryID: "cat1", CategoryName: "Food", Icon: "🍔", Color: "#FF0000", TotalAmount: 30000, Count: 1},
					{CategoryID: "cat2", CategoryName: "Transport", Icon: "🚗", Color: "#0000FF", TotalAmount: 20000, Count: 1},
				},
			},
			wantIncome:   150000,
			wantExpense:  50000,
			wantBalance:  100000,
			wantBreakLen: 2,
		},
		{
			name: "multiple expenses same category grouped correctly",
			reportData: &repository.MonthlyReportData{
				TotalIncome:  0,
				TotalExpense: 35000,
				Categories: []repository.CategoryBreakdownData{
					{CategoryID: "cat1", CategoryName: "Food", Icon: "🍔", Color: "#FF0000", TotalAmount: 35000, Count: 3},
				},
			},
			wantIncome:   0,
			wantExpense:  35000,
			wantBalance:  -35000,
			wantBreakLen: 1,
			wantBreakAmounts: map[string]float64{
				"cat1": 35000,
			},
			wantBreakCounts: map[string]int{
				"cat1": 3,
			},
		},
		{
			name: "expense with nil category not included in breakdown",
			reportData: &repository.MonthlyReportData{
				TotalIncome:  0,
				TotalExpense: 30000,
				Categories:   []repository.CategoryBreakdownData{},
			},
			wantIncome:   0,
			wantExpense:  30000,
			wantBalance:  -30000,
			wantBreakLen: 0,
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
			if len(report.CategoryBreakdown) != tt.wantBreakLen {
				t.Errorf("CategoryBreakdown length: expected %d, got %d", tt.wantBreakLen, len(report.CategoryBreakdown))
			}

			if tt.wantBreakAmounts != nil {
				breakMap := make(map[string]domain.CategoryBreakdownDTO)
				for _, b := range report.CategoryBreakdown {
					breakMap[b.CategoryID] = b
				}

				for catID, wantAmt := range tt.wantBreakAmounts {
					b, ok := breakMap[catID]
					if !ok {
						t.Errorf("category %q missing from breakdown", catID)
						continue
					}
					if b.TotalAmount != wantAmt {
						t.Errorf("category %q TotalAmount: expected %v, got %v", catID, wantAmt, b.TotalAmount)
					}
				}
			}

			if tt.wantBreakCounts != nil {
				breakMap := make(map[string]domain.CategoryBreakdownDTO)
				for _, b := range report.CategoryBreakdown {
					breakMap[b.CategoryID] = b
				}

				for catID, wantCnt := range tt.wantBreakCounts {
					b, ok := breakMap[catID]
					if !ok {
						t.Errorf("category %q missing from breakdown", catID)
						continue
					}
					if b.Count != wantCnt {
						t.Errorf("category %q Count: expected %d, got %d", catID, wantCnt, b.Count)
					}
				}
			}
		})
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
		totalBalance       float64
		currentIncome      float64
		currentExpense     float64
		prevIncome         float64
		prevExpense        float64
		wantTotalBalance   float64
		wantMonthlyIncome  float64
		wantMonthlyExpense float64
		wantIncomeChange   float64
		wantExpenseChange  float64
	}{
		{
			name:               "zero prev month gives 100% change when current non-zero",
			totalBalance:       500,
			currentIncome:      100,
			currentExpense:     50,
			prevIncome:         0,
			prevExpense:        0,
			wantTotalBalance:   500,
			wantMonthlyIncome:  100,
			wantMonthlyExpense: 50,
			wantIncomeChange:   100,
			wantExpenseChange:  100,
		},
		{
			name:               "income decreased by half",
			totalBalance:       1000,
			currentIncome:      50,
			currentExpense:     200,
			prevIncome:         100,
			prevExpense:        200,
			wantTotalBalance:   1000,
			wantMonthlyIncome:  50,
			wantMonthlyExpense: 200,
			wantIncomeChange:   -50,
			wantExpenseChange:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			callCount := 0
			repo := &mockTransactionRepo{
				getTotalByFamilyFn: func(familyID string) (float64, error) {
					return tt.totalBalance, nil
				},
				getMonthlyStatsFn: func(familyID string, year, month int) (float64, float64, error) {
					callCount++
					if callCount == 1 {
						// current month
						return tt.currentIncome, tt.currentExpense, nil
					}
					// previous month
					return tt.prevIncome, tt.prevExpense, nil
				},
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
