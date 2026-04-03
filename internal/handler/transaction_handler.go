package handler

import (
	"kas/internal/domain"
	"kas/internal/service"
	"net/http"
	"strconv"

	"github.com/pocketbase/pocketbase/core"
)

// TransactionHandler handles HTTP requests for transactions
type TransactionHandler struct {
	service service.TransactionService
}

// NewTransactionHandler creates new transaction handler
func NewTransactionHandler(service service.TransactionService) *TransactionHandler {
	return &TransactionHandler{
		service: service,
	}
}

// RegisterRoutes registers all transaction routes
func (h *TransactionHandler) RegisterRoutes(e *core.ServeEvent) {
	// Custom routes
	e.Router.POST("/api/transactions", h.Create)
	e.Router.GET("/api/transactions/:id", h.GetByID)
	e.Router.GET("/api/families/:familyId/transactions", h.GetByFamily)
	e.Router.PATCH("/api/transactions/:id", h.Update)
	e.Router.DELETE("/api/transactions/:id", h.Delete)
	e.Router.GET("/api/families/:familyId/balance", h.GetBalance)
}

// Create transaction handler
func (h *TransactionHandler) Create(e *core.RequestEvent) error {
	// Parse request body
	var req domain.CreateTransactionRequest
	if err := e.BindBody(&req); err != nil {
		return e.BadRequestError("Invalid request body", err)
	}

	// Get user ID from auth context
	authRecord := e.Auth
	if authRecord == nil {
		return e.UnauthorizedError("Authentication required", nil)
	}

	// Call service
	transaction, err := h.service.CreateTransaction(&req, authRecord.Id)
	if err != nil {
		return e.BadRequestError("Failed to create transaction", err)
	}

	return e.JSON(http.StatusCreated, transaction)
}

// GetByID handler
func (h *TransactionHandler) GetByID(e *core.RequestEvent) error {
	id := e.Request.PathValue("id")

	transaction, err := h.service.GetTransaction(id)
	if err != nil {
		return e.NotFoundError("Transaction not found", err)
	}

	return e.JSON(http.StatusOK, transaction)
}

// GetByFamily handler with pagination
func (h *TransactionHandler) GetByFamily(e *core.RequestEvent) error {
	familyID := e.Request.PathValue("familyId")

	// Get pagination params
	page, _ := strconv.Atoi(e.Request.URL.Query().Get("page"))
	pageSize, _ := strconv.Atoi(e.Request.URL.Query().Get("pageSize"))

	transactions, err := h.service.GetFamilyTransactions(familyID, page, pageSize)
	if err != nil {
		return e.BadRequestError("Failed to get transactions", err)
	}

	return e.JSON(http.StatusOK, map[string]any{
		"items": transactions,
		"page":  page,
	})
}

// Update handler
func (h *TransactionHandler) Update(e *core.RequestEvent) error {
	id := e.Request.PathValue("id")

	// Get user ID from auth
	authRecord := e.Auth
	if authRecord == nil {
		return e.UnauthorizedError("Authentication required", nil)
	}

	var req domain.UpdateTransactionRequest
	if err := e.BindBody(&req); err != nil {
		return e.BadRequestError("Invalid request body", err)
	}

	transaction, err := h.service.UpdateTransaction(id, authRecord.Id, &req)
	if err != nil {
		return e.BadRequestError("Failed to update transaction", err)
	}

	return e.JSON(http.StatusOK, transaction)
}

// Delete handler
func (h *TransactionHandler) Delete(e *core.RequestEvent) error {
	id := e.Request.PathValue("id")

	authRecord := e.Auth
	if authRecord == nil {
		return e.UnauthorizedError("Authentication required", nil)
	}

	if err := h.service.DeleteTransaction(id, authRecord.Id); err != nil {
		return e.BadRequestError("Failed to delete transaction", err)
	}

	return e.NoContent(http.StatusNoContent)
}

// GetBalance handler
func (h *TransactionHandler) GetBalance(e *core.RequestEvent) error {
	familyID := e.Request.PathValue("familyId")

	balance, err := h.service.GetFamilyBalance(familyID)
	if err != nil {
		return e.BadRequestError("Failed to calculate balance", err)
	}

	return e.JSON(http.StatusOK, map[string]any{
		"family_id": familyID,
		"balance":   balance,
	})
}
