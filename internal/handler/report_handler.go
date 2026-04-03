package handler

import (
	"kas/internal/domain"
	"kas/internal/middleware"
	"kas/internal/repository"
	"kas/internal/service"
	"log"
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
		log.Printf("Error generating report: %v", err)
		return e.InternalServerError("Failed to generate report", err)
	}

	// Return JSON response
	return e.JSON(200, report)
}

// GetDashboardSummary handles GET /api/reports/summary
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
		log.Printf("Error generating dashboard summary: %v", err)
		return e.InternalServerError("Failed to generate summary", err)
	}

	summary.UserName = e.Auth.GetString("name")

	familyName, err := h.familyMemberRepo.GetFamilyName(familyID)
	if err != nil {
		log.Printf("Error fetching family name: %v", err)
		return e.InternalServerError("Failed to fetch family name", err)
	}
	summary.FamilyName = familyName

	// Return JSON response
	return e.JSON(200, summary)
}
