package handler

import (
	"kas/internal/domain"
	"kas/internal/middleware"
	"kas/internal/repository"
	"kas/internal/service"
	"strconv"
	"time"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/hook"
)

// ReportHandler handles HTTP requests for reports
type ReportHandler struct {
	reportService    service.ReportService
	familyMemberRepo repository.FamilyMemberRepository
	requireAuth      *hook.Handler[*core.RequestEvent]
	requireFamily    *hook.Handler[*core.RequestEvent]
}

// NewReportHandler creates a new report handler
func NewReportHandler(reportService service.ReportService, familyMemberRepo repository.FamilyMemberRepository, requireAuth func(*core.RequestEvent) error, requireFamily func(*core.RequestEvent) error) *ReportHandler {
	return &ReportHandler{
		reportService:    reportService,
		familyMemberRepo: familyMemberRepo,
		requireAuth:      &hook.Handler[*core.RequestEvent]{Func: requireAuth},
		requireFamily:    &hook.Handler[*core.RequestEvent]{Func: requireFamily},
	}
}

// RegisterRoutes registers all report routes
func (h *ReportHandler) RegisterRoutes(e *core.ServeEvent) {
	// GET /api/reports/monthly?year=2026&month=3
	e.Router.GET("/api/reports/monthly", h.GetMonthlyReport).Bind(h.requireAuth).Bind(h.requireFamily)

	// GET /api/reports/summary?year=2026&month=3
	e.Router.GET("/api/reports/summary", h.GetDashboardSummary).Bind(h.requireAuth).Bind(h.requireFamily)
}

// GetMonthlyReport handles GET /api/reports/monthly
// @Summary Get monthly financial report
// @Description Returns a monthly financial report with income and expense breakdown per category for the authenticated user's family.
// @Tags reports
// @Accept json
// @Produce json
// @Param year query int true "Year (1900-2100)"
// @Param month query int true "Month (1-12)"
// @Success 200 {object} domain.MonthlyReportDTO
// @Failure 400 {object} map[string]any "Bad request - missing or invalid parameters"
// @Failure 401 {object} map[string]any "Unauthorized"
// @Failure 500 {object} map[string]any "Internal server error"
// @Router /api/reports/monthly [get]
// @Security BearerAuth
func (h *ReportHandler) GetMonthlyReport(e *core.RequestEvent) error {
	familyID, ok := middleware.GetFamilyIDFromContext(e.Request.Context())
	if !ok {
		return e.InternalServerError("family_id not found in context", nil)
	}

	// Get query parameters
	yearStr := e.Request.URL.Query().Get("year")
	monthStr := e.Request.URL.Query().Get("month")

	// Validate required parameters
	if yearStr == "" {
		return e.BadRequestError("year is required", nil)
	}
	if monthStr == "" {
		return e.BadRequestError("month is required", nil)
	}

	// Parse year and month
	year, err := strconv.Atoi(yearStr)
	if err != nil || year < 1900 || year > 2100 {
		return e.BadRequestError("invalid year", err)
	}

	month, err := strconv.Atoi(monthStr)
	if err != nil || month < 1 || month > 12 {
		return e.BadRequestError("invalid month (must be 1-12)", err)
	}

	// Create request
	req := &domain.MonthlyReportRequest{
		FamilyID: familyID,
		Year:     year,
		Month:    month,
	}

	// Get report from service
	report, err := h.reportService.GetMonthlyReport(req)
	if err != nil {
		e.App.Logger().Error("failed to generate monthly report", "family_id", familyID, "year", year, "month", month, "error", err)
		return e.InternalServerError("Failed to generate report", err)
	}

	// Return JSON response
	return e.JSON(200, report)
}

// GetDashboardSummary handles GET /api/reports/summary
// @Summary Get dashboard summary
// @Description Returns a dashboard summary with total balance, monthly income/expense, and percentage change vs previous month. Defaults to current year/month if not provided.
// @Tags reports
// @Accept json
// @Produce json
// @Param year query int false "Year (1900-2100) — default: current year"
// @Param month query int false "Month (1-12) — default: current month"
// @Success 200 {object} domain.DashboardSummaryDTO
// @Failure 400 {object} map[string]any "Bad request - invalid parameters"
// @Failure 401 {object} map[string]any "Unauthorized"
// @Failure 500 {object} map[string]any "Internal server error"
// @Router /api/reports/summary [get]
// @Security BearerAuth
func (h *ReportHandler) GetDashboardSummary(e *core.RequestEvent) error {
	familyID, ok := middleware.GetFamilyIDFromContext(e.Request.Context())
	if !ok {
		return e.InternalServerError("family_id not found in context", nil)
	}

	yearStr := e.Request.URL.Query().Get("year")
	monthStr := e.Request.URL.Query().Get("month")

	now := time.Now()

	if yearStr == "" {
		yearStr = strconv.Itoa(now.Year())
	}
	if monthStr == "" {
		monthStr = strconv.Itoa(int(now.Month()))
	}

	year, err := strconv.Atoi(yearStr)
	if err != nil || year < 1900 || year > 2100 {
		return e.BadRequestError("invalid year", err)
	}

	month, err := strconv.Atoi(monthStr)
	if err != nil || month < 1 || month > 12 {
		return e.BadRequestError("invalid month (must be 1-12)", err)
	}

	// Create request
	req := &domain.DashboardSummaryRequest{
		FamilyID: familyID,
		Year:     year,
		Month:    month,
	}

	// Get summary from service
	summary, err := h.reportService.GetDashboardSummary(req)
	if err != nil {
		e.App.Logger().Error("failed to generate dashboard summary", "family_id", familyID, "year", year, "month", month, "error", err)
		return e.InternalServerError("Failed to generate summary", err)
	}

	summary.UserName = e.Auth.GetString("name")

	familyName, err := h.familyMemberRepo.GetFamilyName(familyID)
	if err != nil {
		e.App.Logger().Error("failed to fetch family name", "family_id", familyID, "error", err)
		return e.InternalServerError("Failed to fetch family name", err)
	}
	summary.FamilyName = familyName

	// Return JSON response
	return e.JSON(200, summary)
}
