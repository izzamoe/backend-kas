package handler

import (
	"net/http"
	"strconv"

	digiflazzdomain "kas/internal/domain/digiflazz"
	"kas/internal/middleware"
	"kas/internal/service"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/hook"
)

// DigiflazzProductHandler handles HTTP requests for Digiflazz product listing and sync.
type DigiflazzProductHandler struct {
	service            service.DigiflazzProductService
	requireAuth        *hook.Handler[*core.RequestEvent]
	requireFamily      *hook.Handler[*core.RequestEvent]
	requireFamilyOwner *hook.Handler[*core.RequestEvent]
}

// NewDigiflazzProductHandler creates a new DigiflazzProductHandler.
func NewDigiflazzProductHandler(
	svc service.DigiflazzProductService,
	requireAuth func(*core.RequestEvent) error,
	requireFamily func(*core.RequestEvent) error,
	requireFamilyOwner func(*core.RequestEvent) error,
) *DigiflazzProductHandler {
	return &DigiflazzProductHandler{
		service:            svc,
		requireAuth:        &hook.Handler[*core.RequestEvent]{Func: requireAuth},
		requireFamily:      &hook.Handler[*core.RequestEvent]{Func: requireFamily},
		requireFamilyOwner: &hook.Handler[*core.RequestEvent]{Func: requireFamilyOwner},
	}
}

// RegisterRoutes registers all Digiflazz product routes.
func (h *DigiflazzProductHandler) RegisterRoutes(e *core.ServeEvent) {
	e.Router.GET("/api/digiflazz/products", h.Search).Bind(h.requireAuth).Bind(h.requireFamily)
	e.Router.POST("/api/digiflazz/products/sync", h.Sync).Bind(h.requireAuth).Bind(h.requireFamily).Bind(h.requireFamilyOwner)
}

// Sync triggers a manual Digiflazz pricelist sync for the family. Owner only.
//
//	@Summary		Sync Digiflazz products
//	@Description	Triggers an immediate pricelist sync from Digiflazz for the authenticated family. Owner only.
//	@Tags			digiflazz-products
//	@Produce		json
//	@Success		200	{object}	service.SyncResult
//	@Failure		401	{object}	map[string]any
//	@Failure		403	{object}	map[string]any
//	@Failure		500	{object}	map[string]any
//	@Security		BearerAuth
//	@Router			/api/digiflazz/products/sync [post]
func (h *DigiflazzProductHandler) Sync(e *core.RequestEvent) error {
	familyID, ok := middleware.GetFamilyIDFromContext(e.Request.Context())
	if !ok {
		return e.InternalServerError("Family context not found", nil)
	}

	result, err := h.service.SyncForFamily(e.Request.Context(), familyID)
	if err != nil {
		return e.InternalServerError("Failed to sync products", err)
	}

	return e.JSON(http.StatusOK, result)
}

// Search lists/searches Digiflazz products. Accessible by all family members.
//
//	@Summary		Search Digiflazz products
//	@Description	Returns a filtered list of available Digiflazz products. Accessible by all family members.
//	@Tags			digiflazz-products
//	@Produce		json
//	@Param			query		query		string	false	"Free-text search on product name or SKU code"
//	@Param			category	query		string	false	"Filter by product category (e.g. Pulsa, Data, PLN)"
//	@Param			brand		query		string	false	"Filter by brand (e.g. Telkomsel, XL)"
//	@Param			type		query		string	false	"Filter by order type"	Enums(prepaid, postpaid)
//	@Param			status		query		string	false	"Filter by product status"	Enums(active, inactive)
//	@Param			per_page	query		int		false	"Number of results to return (1–200, default 50)"
//	@Success		200			{object}	digiflazz.ProductListResponse
//	@Failure		401			{object}	map[string]any
//	@Failure		500			{object}	map[string]any
//	@Security		BearerAuth
//	@Router			/api/digiflazz/products [get]
func (h *DigiflazzProductHandler) Search(e *core.RequestEvent) error {
	query := e.Request.URL.Query()

	limit, _ := strconv.Atoi(query.Get("per_page"))
	if limit <= 0 || limit > 200 {
		limit = 50
	}

	req := &digiflazzdomain.ProductSearchRequest{
		Query:    query.Get("query"),
		Category: query.Get("category"),
		Brand:    query.Get("brand"),
		Type:     query.Get("type"),
		Status:   query.Get("status"),
		Limit:    limit,
	}

	familyID, ok := middleware.GetFamilyIDFromContext(e.Request.Context())
	if !ok {
		return e.InternalServerError("Family context not found", nil)
	}

	products, err := h.service.SearchProducts(familyID, req)
	if err != nil {
		return e.InternalServerError("Failed to search products", err)
	}

	return e.JSON(http.StatusOK, map[string]any{
		"items": products,
		"limit": limit,
	})
}
