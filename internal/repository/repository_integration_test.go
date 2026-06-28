package repository

import (
	"errors"
	"testing"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"

	"kas/internal/domain"
	_ "kas/migrations"
)

func setupRepositoryTestApp(t *testing.T) *tests.TestApp {
	t.Helper()

	app, err := tests.NewTestApp()
	if err != nil {
		t.Fatalf("failed to create test app: %v", err)
	}
	t.Cleanup(app.Cleanup)

	return app
}

func createTestRecord(t *testing.T, app core.App, collectionName string, values map[string]any) *core.Record {
	t.Helper()

	collection, err := app.FindCollectionByNameOrId(collectionName)
	if err != nil {
		t.Fatalf("failed to find collection %s: %v", collectionName, err)
	}
	record := core.NewRecord(collection)
	for key, value := range values {
		record.Set(key, value)
	}
	if err := app.Save(record); err != nil {
		t.Fatalf("failed to save %s record: %v", collectionName, err)
	}
	return record
}

func createTestUser(t *testing.T, app core.App) *core.Record {
	t.Helper()

	collection, err := app.FindCollectionByNameOrId("users")
	if err != nil {
		t.Fatalf("failed to find users collection: %v", err)
	}
	user := core.NewRecord(collection)
	user.Set("email", "user@example.com")
	user.Set("verified", true)
	user.Set("name", "User Test")
	user.SetPassword("password123456")
	if err := app.Save(user); err != nil {
		t.Fatalf("failed to save user record: %v", err)
	}
	return user
}

func createRepositoryFixtures(t *testing.T, app core.App) (familyID, userID, categoryID string) {
	t.Helper()

	family := createTestRecord(t, app, "families", map[string]any{
		"name":        "Keluarga Test",
		"invite_code": "INVITE01",
	})
	user := createTestUser(t, app)
	category := createTestRecord(t, app, "categories", map[string]any{
		"family_id":  family.Id,
		"name":       "Food",
		"icon":       "🍔",
		"color":      "#ff0000",
		"type":       "expense",
		"is_default": false,
		"is_master":  false,
	})

	return family.Id, user.Id, category.Id
}

func TestDateRange(t *testing.T) {
	t.Run("middle of year", func(t *testing.T) {
		start, end := dateRange(2026, 3)
		if start != "2026-03-01" || end != "2026-04-01" {
			t.Fatalf("unexpected range: %s - %s", start, end)
		}
	})

	t.Run("december rolls over to next year", func(t *testing.T) {
		start, end := dateRange(2026, 12)
		if start != "2026-12-01" || end != "2027-01-01" {
			t.Fatalf("unexpected range: %s - %s", start, end)
		}
	})
}

func TestFamilyRepositoryIntegration(t *testing.T) {
	app := setupRepositoryTestApp(t)
	repo := NewFamilyRepository(app)

	created, err := repo.Create(app, "Keluarga Repo", "REPO1234")
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}
	if created.ID == "" || created.Name != "Keluarga Repo" || created.InviteCode != "REPO1234" {
		t.Fatalf("unexpected created family: %+v", created)
	}

	found, err := repo.FindByInviteCode("REPO1234")
	if err != nil {
		t.Fatalf("FindByInviteCode returned error: %v", err)
	}
	if found == nil || found.ID != created.ID || found.Name != "Keluarga Repo" {
		t.Fatalf("unexpected found family: %+v", found)
	}

	missing, err := repo.FindByInviteCode("MISSING")
	if err != nil {
		t.Fatalf("FindByInviteCode missing returned error: %v", err)
	}
	if missing != nil {
		t.Fatalf("expected nil missing family, got %+v", missing)
	}
}

func TestFamilyMemberRepositoryIntegration(t *testing.T) {
	app := setupRepositoryTestApp(t)
	familyID, userID, _ := createRepositoryFixtures(t, app)
	repo := NewFamilyMemberRepository(app)

	missing, err := repo.GetByUserID(userID)
	if err != nil {
		t.Fatalf("GetByUserID missing returned error: %v", err)
	}
	if missing != nil {
		t.Fatalf("expected missing membership nil, got %+v", missing)
	}

	if err := repo.CreateMember(app, familyID, userID, "owner"); err != nil {
		t.Fatalf("CreateMember returned error: %v", err)
	}
	membership, err := repo.GetByUserID(userID)
	if err != nil {
		t.Fatalf("GetByUserID returned error: %v", err)
	}
	if membership == nil || membership.FamilyID != familyID || membership.UserID != userID || membership.Role != "owner" {
		t.Fatalf("unexpected membership: %+v", membership)
	}

	familyName, err := repo.GetFamilyName(familyID)
	if err != nil {
		t.Fatalf("GetFamilyName returned error: %v", err)
	}
	if familyName != "Keluarga Test" {
		t.Fatalf("expected family name Keluarga Test, got %q", familyName)
	}

	if err := repo.DeleteMember(userID); err != nil {
		t.Fatalf("DeleteMember returned error: %v", err)
	}
	membership, err = repo.GetByUserID(userID)
	if err != nil {
		t.Fatalf("GetByUserID after delete returned error: %v", err)
	}
	if membership != nil {
		t.Fatalf("expected membership deleted, got %+v", membership)
	}

	if err := repo.DeleteMember(userID); err == nil || err.Error() != "not a member of any family" {
		t.Fatalf("expected not a member error, got %v", err)
	}
}

func TestCategoryRepositoryIntegration(t *testing.T) {
	app := setupRepositoryTestApp(t)
	familyID, _, categoryID := createRepositoryFixtures(t, app)
	repo := NewCategoryRepository(app)

	category, err := repo.GetByID(categoryID)
	if err != nil {
		t.Fatalf("GetByID returned error: %v", err)
	}
	if category == nil || category.ID != categoryID || category.FamilyID != familyID || category.Name != "Food" || category.IsDefault {
		t.Fatalf("unexpected category: %+v", category)
	}

	_, err = repo.GetByID("missingcategory")
	if !errors.Is(err, domain.ErrCategoryNotFound) {
		t.Fatalf("expected ErrCategoryNotFound for missing category, got %v", err)
	}
}

func TestTransactionRepositoryIntegration(t *testing.T) {
	app := setupRepositoryTestApp(t)
	familyID, userID, categoryID := createRepositoryFixtures(t, app)
	repo := NewTransactionRepository(app)

	income, err := repo.Create(&domain.CreateTransactionRequest{
		CategoryID: categoryID,
		Type:       domain.TransactionTypeIncome,
		Amount:     250000,
		Note:       "salary",
		Date:       "2026-03-05T09:00:00Z",
	}, userID, familyID)
	if err != nil {
		t.Fatalf("Create income returned error: %v", err)
	}
	expense, err := repo.Create(&domain.CreateTransactionRequest{
		CategoryID: categoryID,
		Type:       domain.TransactionTypeExpense,
		Amount:     50000,
		Note:       "lunch",
		Date:       "2026-03-10T12:00:00Z",
	}, userID, familyID)
	if err != nil {
		t.Fatalf("Create expense returned error: %v", err)
	}
	_, err = repo.Create(&domain.CreateTransactionRequest{
		CategoryID: categoryID,
		Type:       domain.TransactionTypeExpense,
		Amount:     20000,
		Note:       "april",
		Date:       "2026-04-01T12:00:00Z",
	}, userID, familyID)
	if err != nil {
		t.Fatalf("Create april expense returned error: %v", err)
	}

	found, err := repo.GetByID(income.ID)
	if err != nil {
		t.Fatalf("GetByID returned error: %v", err)
	}
	if found.ID != income.ID || found.Amount != 250000 || found.Type != domain.TransactionTypeIncome {
		t.Fatalf("unexpected found transaction: %+v", found)
	}

	creatorID, err := repo.GetCreatorID(income.ID)
	if err != nil {
		t.Fatalf("GetCreatorID returned error: %v", err)
	}
	if creatorID != userID {
		t.Fatalf("expected creator %s, got %s", userID, creatorID)
	}

	page, err := repo.GetByFamilyID(familyID, 2, 0)
	if err != nil {
		t.Fatalf("GetByFamilyID returned error: %v", err)
	}
	if len(page) != 2 {
		t.Fatalf("expected 2 paged transactions, got %d", len(page))
	}

	rangedTransactions, totalItems, err := repo.GetByFamilyDateRange(familyID, "2026-03-01", "2026-04-01", 1, 0)
	if err != nil {
		t.Fatalf("GetByFamilyDateRange returned error: %v", err)
	}
	if len(rangedTransactions) != 1 || totalItems != 2 {
		t.Fatalf("expected 1 ranged transaction and total 2, got len=%d total=%d", len(rangedTransactions), totalItems)
	}
	if rangedTransactions[0].Category == nil || rangedTransactions[0].Creator == nil || rangedTransactions[0].Family == nil {
		t.Fatalf("expected expanded relations, got %+v", rangedTransactions[0])
	}

	updated, err := repo.Update(expense.ID, &domain.UpdateTransactionRequest{Amount: 75000, Note: "updated lunch"})
	if err != nil {
		t.Fatalf("Update returned error: %v", err)
	}
	if updated.Amount != 75000 || updated.Note != "updated lunch" {
		t.Fatalf("unexpected updated transaction: %+v", updated)
	}

	balance, err := repo.GetTotalByFamily(familyID)
	if err != nil {
		t.Fatalf("GetTotalByFamily returned error: %v", err)
	}
	if balance != 155000 {
		t.Fatalf("expected balance 155000, got %v", balance)
	}

	incomeTotal, expenseTotal, err := repo.GetMonthlyStats(familyID, 2026, 3)
	if err != nil {
		t.Fatalf("GetMonthlyStats returned error: %v", err)
	}
	if incomeTotal != 250000 || expenseTotal != 75000 {
		t.Fatalf("unexpected monthly stats: income=%v expense=%v", incomeTotal, expenseTotal)
	}

	report, err := repo.GetMonthlyReportData(familyID, 2026, 3)
	if err != nil {
		t.Fatalf("GetMonthlyReportData returned error: %v", err)
	}
	if report.TotalIncome != 250000 || report.TotalExpense != 75000 || len(report.ExpenseCategories) != 1 {
		t.Fatalf("unexpected monthly report: %+v", report)
	}

	totalBalance, monthlyIncome, monthlyExpense, prevIncome, prevExpense, err := repo.GetDashboardData(familyID, 2026, 3)
	if err != nil {
		t.Fatalf("GetDashboardData returned error: %v", err)
	}
	if totalBalance != 155000 || monthlyIncome != 250000 || monthlyExpense != 75000 || prevIncome != 0 || prevExpense != 0 {
		t.Fatalf("unexpected dashboard data: balance=%v income=%v expense=%v prevIncome=%v prevExpense=%v", totalBalance, monthlyIncome, monthlyExpense, prevIncome, prevExpense)
	}

	if err := repo.Delete(expense.ID); err != nil {
		t.Fatalf("Delete returned error: %v", err)
	}
	if _, err := repo.GetByID(expense.ID); err == nil {
		t.Fatal("expected GetByID deleted transaction to fail")
	}
}

func TestTransactionRepositoryAdditionalBranches(t *testing.T) {
	app := setupRepositoryTestApp(t)
	familyID, userID, categoryID := createRepositoryFixtures(t, app)
	secondCategory := createTestRecord(t, app, "categories", map[string]any{
		"family_id":  familyID,
		"name":       "Salary",
		"icon":       "💼",
		"color":      "#00ff00",
		"type":       "income",
		"is_default": false,
		"is_master":  false,
	})
	repo := NewTransactionRepository(app)

	decemberIncome, err := repo.Create(&domain.CreateTransactionRequest{
		CategoryID: secondCategory.Id,
		Type:       domain.TransactionTypeIncome,
		Amount:     100000,
		Note:       "december salary",
		Date:       "2025-12-15T09:00:00Z",
	}, userID, familyID)
	if err != nil {
		t.Fatalf("Create december income returned error: %v", err)
	}
	if _, err := repo.Create(&domain.CreateTransactionRequest{
		CategoryID: categoryID,
		Type:       domain.TransactionTypeExpense,
		Amount:     25000,
		Note:       "january lunch",
		Date:       "2026-01-10T12:00:00Z",
	}, userID, familyID); err != nil {
		t.Fatalf("Create january expense returned error: %v", err)
	}

	updated, err := repo.Update(decemberIncome.ID, &domain.UpdateTransactionRequest{
		CategoryID: categoryID,
		Type:       domain.TransactionTypeExpense,
		Amount:     30000,
		Note:       "updated all fields",
		Date:       "2026-01-05T10:00:00Z",
	})
	if err != nil {
		t.Fatalf("Update all fields returned error: %v", err)
	}
	if updated.Type != domain.TransactionTypeExpense || updated.Amount != 30000 || updated.Note != "updated all fields" {
		t.Fatalf("unexpected updated all-fields transaction: %+v", updated)
	}

	totalBalance, monthlyIncome, monthlyExpense, prevIncome, prevExpense, err := repo.GetDashboardData(familyID, 2026, 1)
	if err != nil {
		t.Fatalf("GetDashboardData January returned error: %v", err)
	}
	if totalBalance != -55000 || monthlyIncome != 0 || monthlyExpense != 55000 || prevIncome != 0 || prevExpense != 0 {
		t.Fatalf("unexpected January dashboard data: balance=%v income=%v expense=%v prevIncome=%v prevExpense=%v", totalBalance, monthlyIncome, monthlyExpense, prevIncome, prevExpense)
	}

	emptyIncome, emptyExpense, err := repo.GetMonthlyStats(familyID, 2026, 2)
	if err != nil {
		t.Fatalf("GetMonthlyStats empty month returned error: %v", err)
	}
	if emptyIncome != 0 || emptyExpense != 0 {
		t.Fatalf("expected empty month stats to be zero, got income=%v expense=%v", emptyIncome, emptyExpense)
	}

	emptyReport, err := repo.GetMonthlyReportData(familyID, 2026, 2)
	if err != nil {
		t.Fatalf("GetMonthlyReportData empty month returned error: %v", err)
	}
	if emptyReport.TotalIncome != 0 || emptyReport.TotalExpense != 0 || len(emptyReport.IncomeCategories) != 0 || len(emptyReport.ExpenseCategories) != 0 {
		t.Fatalf("expected empty monthly report, got %+v", emptyReport)
	}

	if _, err := repo.Update("missingrecordid", &domain.UpdateTransactionRequest{Note: "nope"}); err == nil {
		t.Fatal("expected Update missing record to fail")
	}
	if err := repo.Delete("missingrecordid"); err == nil {
		t.Fatal("expected Delete missing record to fail")
	}
	if _, err := repo.GetCreatorID("missingrecordid"); err == nil {
		t.Fatal("expected GetCreatorID missing record to fail")
	}
}

func TestFamilyAndMemberRepositoryErrorBranches(t *testing.T) {
	app := setupRepositoryTestApp(t)
	familyRepo := NewFamilyRepository(app)
	memberRepo := NewFamilyMemberRepository(app)

	if _, err := memberRepo.GetFamilyName("missingfamily"); err == nil {
		t.Fatal("expected GetFamilyName missing family to fail")
	}
	if err := memberRepo.CreateMember(app, "missingfamily", "missinguser", "member"); err == nil {
		t.Fatal("expected CreateMember with missing relations to fail")
	}
	if _, err := familyRepo.Create(app, "", ""); err == nil {
		t.Fatal("expected Create with blank required fields to fail")
	}
}

func TestCategorySeedMasterCategoriesIntegration(t *testing.T) {
	app := setupRepositoryTestApp(t)
	familyID, _, _ := createRepositoryFixtures(t, app)
	repo := NewCategoryRepository(app)

	createTestRecord(t, app, "categories", map[string]any{
		"family_id":  "",
		"name":       "Master Income",
		"icon":       "💰",
		"color":      "#00ff00",
		"type":       "income",
		"is_default": true,
		"is_master":  true,
	})

	if err := repo.SeedMasterCategories(app, familyID); err != nil {
		t.Fatalf("SeedMasterCategories returned error: %v", err)
	}
	records, err := app.FindRecordsByFilter("categories", "family_id = {:familyID} && is_default = true && is_master = false", "name", -1, 0, map[string]any{"familyID": familyID})
	if err != nil {
		t.Fatalf("failed to query seeded categories: %v", err)
	}
	if len(records) == 0 {
		t.Fatal("expected seeded default category")
	}
}

func TestCategorySeedMasterCategoriesNoMasters(t *testing.T) {
	app := setupRepositoryTestApp(t)
	familyID, _, _ := createRepositoryFixtures(t, app)
	repo := NewCategoryRepository(app)
	masterRecords, err := app.FindRecordsByFilter("categories", "is_master = true && family_id = ''", "", -1, 0, nil)
	if err != nil {
		t.Fatalf("failed to query master categories: %v", err)
	}
	for _, record := range masterRecords {
		if err := app.Delete(record); err != nil {
			t.Fatalf("failed to delete seeded master category: %v", err)
		}
	}

	if err := repo.SeedMasterCategories(app, familyID); err == nil || err.Error() != "no master categories found in database" {
		t.Fatalf("expected no master categories error, got %v", err)
	}
}
