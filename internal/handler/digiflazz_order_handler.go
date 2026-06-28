package handler

import (
	"errors"
	"net/http"
	"strings"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/hook"

	digiflazzdomain "kas/internal/domain/digiflazz"
	"kas/internal/middleware"
	"kas/internal/service"
)

type DigiflazzOrderHandler struct {
	service       service.DigiflazzOrderService
	requireAuth   *hook.Handler[*core.RequestEvent]
	requireFamily *hook.Handler[*core.RequestEvent]
}

func NewDigiflazzOrderHandler(
	svc service.DigiflazzOrderService,
	requireAuth func(*core.RequestEvent) error,
	requireFamily func(*core.RequestEvent) error,
) *DigiflazzOrderHandler {
	return &DigiflazzOrderHandler{
		service:       svc,
		requireAuth:   &hook.Handler[*core.RequestEvent]{Func: requireAuth},
		requireFamily: &hook.Handler[*core.RequestEvent]{Func: requireFamily},
	}
}

func (h *DigiflazzOrderHandler) RegisterRoutes(e *core.ServeEvent) {
	e.Router.GET("/api/digiflazz/orders", h.List).Bind(h.requireAuth).Bind(h.requireFamily)
	e.Router.GET("/api/digiflazz/orders/{id}", h.Get).Bind(h.requireAuth).Bind(h.requireFamily)
	e.Router.POST("/api/digiflazz/orders", h.Create).Bind(h.requireAuth).Bind(h.requireFamily)
	e.Router.POST("/api/digiflazz/orders/{id}/pay", h.Pay).Bind(h.requireAuth).Bind(h.requireFamily)
	e.Router.GET("/api/digiflazz/pln/inquiry", h.InquiryPLN).Bind(h.requireAuth).Bind(h.requireFamily)
}

// @Summary List Digiflazz orders
// @Description Returns a paginated list of Digiflazz orders for the authenticated user's family.
// @Tags digiflazz-orders
// @Accept json
// @Produce json
// @Param page query int false "Page number (default: 1)"
// @Param page_size query int false "Page size, 1-100 (default: 20)"
// @Success 200 {object} digiflazz.OrderListResponse
// @Failure 401 {object} map[string]any "Unauthorized"
// @Failure 403 {object} map[string]any "Forbidden"
// @Failure 500 {object} map[string]any "Internal server error"
// @Router /api/digiflazz/orders [get]
// @Security BearerAuth
func (h *DigiflazzOrderHandler) List(e *core.RequestEvent) error {
	familyID, ok := middleware.GetFamilyIDFromContext(e.Request.Context())
	if !ok {
		return e.InternalServerError("Family context not found", nil)
	}

	page, pageSize := ParsePagination(e.Request.URL.Query())

	orders, err := h.service.ListFamilyOrders(familyID, page, pageSize)
	if err != nil {
		return mapDigiflazzOrderError(e, err)
	}
	return e.JSON(http.StatusOK, map[string]any{
		"items":     orders,
		"page":      page,
		"page_size": pageSize,
	})
}

// @Summary Get a Digiflazz order
// @Description Returns a single Digiflazz order by ID for the authenticated user's family.
// @Tags digiflazz-orders
// @Accept json
// @Produce json
// @Param id path string true "Order ID"
// @Success 200 {object} digiflazz.OrderDTO
// @Failure 401 {object} map[string]any "Unauthorized"
// @Failure 403 {object} map[string]any "Forbidden"
// @Failure 404 {object} map[string]any "Not found"
// @Failure 500 {object} map[string]any "Internal server error"
// @Router /api/digiflazz/orders/{id} [get]
// @Security BearerAuth
func (h *DigiflazzOrderHandler) Get(e *core.RequestEvent) error {
	familyID, ok := middleware.GetFamilyIDFromContext(e.Request.Context())
	if !ok {
		return e.InternalServerError("Family context not found", nil)
	}

	id := e.Request.PathValue("id")
	order, err := h.service.GetOrder(familyID, id)
	if err != nil {
		return mapDigiflazzOrderError(e, err)
	}
	if order == nil {
		return e.NotFoundError("Digiflazz order not found", nil)
	}
	return e.JSON(http.StatusOK, order)
}

// @Summary Create a Digiflazz order
// @Description Creates a prepaid order or postpaid inquiry. Set order_type to "prepaid" (default) or "postpaid" for a postpaid inquiry.
// @Tags digiflazz-orders
// @Accept json
// @Produce json
// @Param request body digiflazz.CreateOrderRequest true "Order request"
// @Success 201 {object} digiflazz.OrderDTO
// @Failure 400 {object} map[string]any "Bad request"
// @Failure 401 {object} map[string]any "Unauthorized"
// @Failure 403 {object} map[string]any "Forbidden"
// @Failure 500 {object} map[string]any "Internal server error"
// @Router /api/digiflazz/orders [post]
// @Security BearerAuth
func (h *DigiflazzOrderHandler) Create(e *core.RequestEvent) error {
	familyID, ok := middleware.GetFamilyIDFromContext(e.Request.Context())
	if !ok {
		return e.InternalServerError("Family context not found", nil)
	}

	var req digiflazzdomain.CreateOrderRequest
	if err := e.BindBody(&req); err != nil {
		return e.BadRequestError("Invalid request body", err)
	}
	order, err := h.service.CreateOrder(e.Request.Context(), familyID, e.Auth.Id, req)
	if err != nil {
		return mapDigiflazzOrderError(e, err)
	}
	return e.JSON(http.StatusCreated, order)
}

// @Summary Pay a postpaid inquiry order
// @Description Executes payment for an existing postpaid inquiry order.
// @Tags digiflazz-orders
// @Accept json
// @Produce json
// @Param id path string true "Order ID"
// @Success 200 {object} digiflazz.OrderDTO
// @Failure 400 {object} map[string]any "Bad request"
// @Failure 401 {object} map[string]any "Unauthorized"
// @Failure 403 {object} map[string]any "Forbidden"
// @Failure 404 {object} map[string]any "Not found"
// @Failure 500 {object} map[string]any "Internal server error"
// @Router /api/digiflazz/orders/{id}/pay [post]
// @Security BearerAuth
func (h *DigiflazzOrderHandler) Pay(e *core.RequestEvent) error {
	familyID, ok := middleware.GetFamilyIDFromContext(e.Request.Context())
	if !ok {
		return e.InternalServerError("Family context not found", nil)
	}

	order, err := h.service.PayPostpaidOrder(e.Request.Context(), familyID, e.Auth.Id, e.Request.PathValue("id"))
	if err != nil {
		return mapDigiflazzOrderError(e, err)
	}
	return e.JSON(http.StatusOK, order)
}

// @Summary PLN customer inquiry
// @Description Validates a PLN meter number and returns customer details before purchasing a PLN token.
// @Tags digiflazz-orders
// @Accept json
// @Produce json
// @Param customer_no query string true "PLN meter number or subscriber ID"
// @Success 200 {object} digiflazz.PLNInquiryResult
// @Failure 400 {object} map[string]any "Bad request"
// @Failure 401 {object} map[string]any "Unauthorized"
// @Failure 403 {object} map[string]any "Forbidden"
// @Failure 500 {object} map[string]any "Internal server error"
// @Router /api/digiflazz/pln/inquiry [get]
// @Security BearerAuth
func (h *DigiflazzOrderHandler) InquiryPLN(e *core.RequestEvent) error {
	familyID, ok := middleware.GetFamilyIDFromContext(e.Request.Context())
	if !ok {
		return e.InternalServerError("Family context not found", nil)
	}

	customerNo := strings.TrimSpace(e.Request.URL.Query().Get("customer_no"))
	if customerNo == "" {
		return e.BadRequestError("customer_no query parameter is required", nil)
	}

	result, err := h.service.InquiryPLN(e.Request.Context(), familyID, customerNo)
	if err != nil {
		return mapDigiflazzOrderError(e, err)
	}
	return e.JSON(http.StatusOK, result)
}

func mapDigiflazzOrderError(e *core.RequestEvent, err error) error { //nolint:gocyclo // Error mapping requires exhaustive error type dispatch
	if err == nil {
		return nil
	}
	if errors.Is(err, digiflazzdomain.ErrProductNotFound) {
		return e.BadRequestError("product not found for your account", err)
	}
	if errors.Is(err, digiflazzdomain.ErrAmountRequired) {
		return e.BadRequestError("amount is required and must be a multiple of 1000 for E-Money products", err)
	}
	if errors.Is(err, digiflazzdomain.ErrIDPelanggan2Required) {
		return e.BadRequestError("id_pelanggan2 (NIK) is required for SAMSAT products", err)
	}
	if errors.Is(err, digiflazzdomain.ErrOrderNotFound) {
		return e.NotFoundError("Digiflazz order not found", err)
	}
	lower := strings.ToLower(err.Error())
	if strings.Contains(lower, "unauthorized") || strings.Contains(lower, "not a member") {
		return e.ForbiddenError("Access denied", errors.New(err.Error()))
	}
	if strings.Contains(lower, "not found") {
		return e.NotFoundError("Digiflazz order not found", err)
	}
	if strings.Contains(lower, "required") || strings.Contains(lower, "invalid") || strings.Contains(lower, "must be") || strings.Contains(lower, "amount changed") || strings.Contains(lower, "fresh inquiry") || strings.Contains(lower, "insufficient") || strings.Contains(lower, "unavailable") || strings.Contains(lower, "inactive") || strings.Contains(lower, "cutoff") {
		return e.BadRequestError(err.Error(), err)
	}
	return e.InternalServerError("Internal server error", err)
}
