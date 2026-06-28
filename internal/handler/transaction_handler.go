package handler

import (
	"net/http"
	"strconv"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/hook"

	"kas/internal/domain"
	"kas/internal/middleware"
	"kas/internal/service"
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
//
//	@Summary		List transactions by date range
//	@Description	Returns authenticated user's family transactions for a date range with pagination
//	@Tags			transactions
//	@Accept			json
//	@Produce		json
//	@Param			start	query	string	true	"Start date (ISO 8601 format, e.g. 2026-01-01)"
//	@Param			end		query	string	true	"End date (ISO 8601 format, e.g. 2026-01-31)"
//	@Param			page	query	int		false	"Page number (default: 1)"
//	@Param			perPage	query	int		false	"Items per page (default: 20)"
//	@Security		BearerAuth
//	@Success		200	{object}	domain.TransactionListResponse
//	@Failure		400	{object}	map[string]interface{}
//	@Failure		401	{object}	map[string]interface{}
//	@Failure		403	{object}	map[string]interface{}
//	@Failure		500	{object}	map[string]interface{}
//	@Router			/api/transactions [get]
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
//
//	@Summary		Create a new transaction
//	@Description	Creates a new transaction for the authenticated user's family
//	@Tags			transactions
//	@Accept			json
//	@Produce		json
//	@Param			body	body		domain.CreateTransactionRequest	true	"Transaction data"
//	@Security		BearerAuth
//	@Success		201	{object}	domain.TransactionDTO
//	@Failure		400	{object}	map[string]interface{}
//	@Failure		401	{object}	map[string]interface{}
//	@Failure		403	{object}	map[string]interface{}
//	@Failure		500	{object}	map[string]interface{}
//	@Router			/api/transactions [post]
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
//
//	@Summary		Get transaction by ID
//	@Description	Returns a single transaction by ID for the authenticated user's family
//	@Tags			transactions
//	@Accept			json
//	@Produce		json
//	@Param			id	path	string	true	"Transaction ID"
//	@Security		BearerAuth
//	@Success		200	{object}	domain.TransactionDTO
//	@Failure		400	{object}	map[string]interface{}
//	@Failure		401	{object}	map[string]interface{}
//	@Failure		403	{object}	map[string]interface{}
//	@Failure		404	{object}	map[string]interface{}
//	@Failure		500	{object}	map[string]interface{}
//	@Router			/api/transactions/{id} [get]
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
//
//	@Summary		Get family transactions
//	@Description	Returns paginated transactions for a specific family
//	@Tags			transactions
//	@Accept			json
//	@Produce		json
//	@Param			familyId	path	string	true	"Family ID"
//	@Param			page		query	int		false	"Page number (default: 1)"
//	@Param			pageSize	query	int		false	"Page size (default: 20, max: 100)"
//	@Security		BearerAuth
//	@Success		200	{object}	domain.FamilyTransactionListResponse
//	@Failure		400	{object}	map[string]interface{}
//	@Failure		401	{object}	map[string]interface{}
//	@Failure		403	{object}	map[string]interface{}
//	@Failure		500	{object}	map[string]interface{}
//	@Router			/api/families/{familyId}/transactions [get]
func (h *TransactionHandler) GetByFamily(e *core.RequestEvent) error {
	familyID := e.Request.PathValue("familyId")
	authFamilyID, ok := middleware.GetFamilyIDFromContext(e.Request.Context())
	if !ok {
		return e.InternalServerError("Family context not found", nil)
	}
	if familyID != authFamilyID {
		return e.ForbiddenError("Cannot access another family's transactions", nil)
	}

	page, pageSize := ParsePagination(e.Request.URL.Query())

	transactions, err := h.service.GetFamilyTransactions(familyID, page, pageSize)
	if err != nil {
		return e.BadRequestError("Failed to get transactions", err)
	}

	return e.JSON(http.StatusOK, map[string]any{
		"items":     transactions,
		"page":      page,
		"page_size": pageSize,
	})
}

// Update handler
//
//	@Summary		Update a transaction
//	@Description	Updates an existing transaction (must be the creator)
//	@Tags			transactions
//	@Accept			json
//	@Produce		json
//	@Param			id		path	string						true	"Transaction ID"
//	@Param			body	body	domain.UpdateTransactionRequest	true	"Update data"
//	@Security		BearerAuth
//	@Success		200	{object}	domain.TransactionDTO
//	@Failure		400	{object}	map[string]interface{}
//	@Failure		401	{object}	map[string]interface{}
//	@Failure		500	{object}	map[string]interface{}
//	@Router			/api/transactions/{id} [patch]
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
//
//	@Summary		Delete a transaction
//	@Description	Deletes an existing transaction (must be the creator)
//	@Tags			transactions
//	@Accept			json
//	@Produce		json
//	@Param			id	path	string	true	"Transaction ID"
//	@Security		BearerAuth
//	@Success		204	"No Content"
//	@Failure		400	{object}	map[string]interface{}
//	@Failure		401	{object}	map[string]interface{}
//	@Failure		500	{object}	map[string]interface{}
//	@Router			/api/transactions/{id} [delete]
func (h *TransactionHandler) Delete(e *core.RequestEvent) error {
	id := e.Request.PathValue("id")

	if err := h.service.DeleteTransaction(id, e.Auth.Id); err != nil {
		return e.BadRequestError("Failed to delete transaction", err)
	}

	return e.NoContent(http.StatusNoContent)
}

// GetBalance handler
//
//	@Summary		Get family balance
//	@Description	Returns the current balance for a specific family
//	@Tags			transactions
//	@Accept			json
//	@Produce		json
//	@Param			familyId	path	string	true	"Family ID"
//	@Security		BearerAuth
//	@Success		200	{object}	domain.BalanceResponse
//	@Failure		400	{object}	map[string]interface{}
//	@Failure		401	{object}	map[string]interface{}
//	@Failure		403	{object}	map[string]interface{}
//	@Failure		500	{object}	map[string]interface{}
//	@Router			/api/families/{familyId}/balance [get]
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
