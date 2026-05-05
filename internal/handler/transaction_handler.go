package handler

import (
	"kas/internal/domain"
	"kas/internal/middleware"
	"kas/internal/service"
	"net/http"
	"strconv"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/hook"
)

// TransactionHandler handles HTTP requests for transactions
type TransactionHandler struct {
	service       service.TransactionService
	requireAuth   *hook.Handler[*core.RequestEvent]
	requireFamily *hook.Handler[*core.RequestEvent]
}

// NewTransactionHandler creates new transaction handler
func NewTransactionHandler(
	service service.TransactionService,
	requireAuth func(*core.RequestEvent) error,
	requireFamily func(*core.RequestEvent) error,
) *TransactionHandler {
	return &TransactionHandler{
		service:       service,
		requireAuth:   &hook.Handler[*core.RequestEvent]{Func: requireAuth},
		requireFamily: &hook.Handler[*core.RequestEvent]{Func: requireFamily},
	}
}

// RegisterRoutes registers all transaction routes
func (h *TransactionHandler) RegisterRoutes(e *core.ServeEvent) {
	e.Router.POST("/api/transactions", h.Create).Bind(h.requireAuth).Bind(h.requireFamily)
	e.Router.GET("/api/transactions", h.List).Bind(h.requireAuth).Bind(h.requireFamily)
	e.Router.GET("/api/transactions/{id}", h.GetByID).Bind(h.requireAuth).Bind(h.requireFamily)
	e.Router.GET("/api/families/{familyId}/transactions", h.GetByFamily).Bind(h.requireAuth).Bind(h.requireFamily)
	e.Router.PATCH("/api/transactions/{id}", h.Update).Bind(h.requireAuth)
	e.Router.DELETE("/api/transactions/{id}", h.Delete).Bind(h.requireAuth)
	e.Router.GET("/api/families/{familyId}/balance", h.GetBalance).Bind(h.requireAuth).Bind(h.requireFamily)
}

// List handler returns authenticated user's family transactions for a date range.
func (h *TransactionHandler) List(e *core.RequestEvent) error {
	familyID, ok := middleware.GetFamilyIDFromContext(e.Request.Context())
	if !ok {
		return e.InternalServerError("Family context not found", nil)
	}

	query := e.Request.URL.Query()
	startDate := query.Get("start")
	endDate := query.Get("end")
	if startDate == "" {
		return e.BadRequestError("start is required", nil)
	}
	if endDate == "" {
		return e.BadRequestError("end is required", nil)
	}

	page, _ := strconv.Atoi(query.Get("page"))
	perPage, _ := strconv.Atoi(query.Get("perPage"))
	if perPage == 0 {
		perPage, _ = strconv.Atoi(query.Get("pageSize"))
	}

	transactions, err := h.service.GetTransactionsByDateRange(familyID, startDate, endDate, page, perPage)
	if err != nil {
		return e.BadRequestError("Failed to get transactions", err)
	}

	return e.JSON(http.StatusOK, transactions)
}

// Create transaction handler
func (h *TransactionHandler) Create(e *core.RequestEvent) error {
	familyID, ok := middleware.GetFamilyIDFromContext(e.Request.Context())
	if !ok {
		return e.InternalServerError("Family context not found", nil)
	}

	var req domain.CreateTransactionRequest
	if err := e.BindBody(&req); err != nil {
		return e.BadRequestError("Invalid request body", err)
	}

	transaction, err := h.service.CreateTransaction(&req, e.Auth.Id, familyID)
	if err != nil {
		return e.BadRequestError("Failed to create transaction", err)
	}

	return e.JSON(http.StatusCreated, transaction)
}

// GetByID handler
func (h *TransactionHandler) GetByID(e *core.RequestEvent) error {
	id := e.Request.PathValue("id")
	familyID, ok := middleware.GetFamilyIDFromContext(e.Request.Context())
	if !ok {
		return e.InternalServerError("Family context not found", nil)
	}

	transaction, err := h.service.GetTransaction(id)
	if err != nil {
		return e.NotFoundError("Transaction not found", err)
	}
	if transaction.FamilyID != familyID {
		return e.ForbiddenError("Cannot access another family's transaction", nil)
	}

	return e.JSON(http.StatusOK, transaction)
}

// GetByFamily handler with pagination
func (h *TransactionHandler) GetByFamily(e *core.RequestEvent) error {
	familyID := e.Request.PathValue("familyId")
	authFamilyID, ok := middleware.GetFamilyIDFromContext(e.Request.Context())
	if !ok {
		return e.InternalServerError("Family context not found", nil)
	}
	if familyID != authFamilyID {
		return e.ForbiddenError("Cannot access another family's transactions", nil)
	}

	// Get pagination params
	page, _ := strconv.Atoi(e.Request.URL.Query().Get("page"))
	pageSize, _ := strconv.Atoi(e.Request.URL.Query().Get("pageSize"))

	// Normalize here so response matches what was actually queried
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	transactions, err := h.service.GetFamilyTransactions(familyID, page, pageSize)
	if err != nil {
		return e.BadRequestError("Failed to get transactions", err)
	}

	return e.JSON(http.StatusOK, map[string]any{
		"items":    transactions,
		"page":     page,
		"pageSize": pageSize,
	})
}

// Update handler
func (h *TransactionHandler) Update(e *core.RequestEvent) error {
	id := e.Request.PathValue("id")

	var req domain.UpdateTransactionRequest
	if err := e.BindBody(&req); err != nil {
		return e.BadRequestError("Invalid request body", err)
	}

	transaction, err := h.service.UpdateTransaction(id, e.Auth.Id, &req)
	if err != nil {
		return e.BadRequestError("Failed to update transaction", err)
	}

	return e.JSON(http.StatusOK, transaction)
}

// Delete handler
func (h *TransactionHandler) Delete(e *core.RequestEvent) error {
	id := e.Request.PathValue("id")

	if err := h.service.DeleteTransaction(id, e.Auth.Id); err != nil {
		return e.BadRequestError("Failed to delete transaction", err)
	}

	return e.NoContent(http.StatusNoContent)
}

// GetBalance handler
func (h *TransactionHandler) GetBalance(e *core.RequestEvent) error {
	familyID := e.Request.PathValue("familyId")
	authFamilyID, ok := middleware.GetFamilyIDFromContext(e.Request.Context())
	if !ok {
		return e.InternalServerError("Family context not found", nil)
	}
	if familyID != authFamilyID {
		return e.ForbiddenError("Cannot access another family's balance", nil)
	}

	balance, err := h.service.GetFamilyBalance(familyID)
	if err != nil {
		return e.BadRequestError("Failed to calculate balance", err)
	}

	return e.JSON(http.StatusOK, map[string]any{
		"family_id": familyID,
		"balance":   balance,
	})
}
