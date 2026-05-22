package handler

import (
	"kas/internal/domain"
	"kas/internal/service"
	"net/http"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/hook"
)

// FamilyHandler handles HTTP requests for family management endpoints.
type FamilyHandler struct {
	service     service.FamilyService
	requireAuth *hook.Handler[*core.RequestEvent]
}

// NewFamilyHandler creates a new FamilyHandler wired with the given service and auth middleware.
func NewFamilyHandler(service service.FamilyService, requireAuth func(*core.RequestEvent) error) *FamilyHandler {
	return &FamilyHandler{
		service:     service,
		requireAuth: &hook.Handler[*core.RequestEvent]{Func: requireAuth},
	}
}

// RegisterRoutes registers all family-related HTTP routes with their required middleware.
func (h *FamilyHandler) RegisterRoutes(e *core.ServeEvent) {
	e.Router.POST("/api/families", h.Create).Bind(h.requireAuth)
	e.Router.POST("/api/families/join", h.Join).Bind(h.requireAuth)
	e.Router.POST("/api/families/leave", h.Leave).Bind(h.requireAuth)
}

// Create handles POST /api/families — creates a new family for the authenticated user.
// @Summary Create a new family
// @Description Creates a new family and assigns the authenticated user as the owner.
// @Tags families
// @Accept json
// @Produce json
// @Param body body domain.CreateFamilyRequest true "Family creation payload"
// @Success 201 {object} domain.CreateFamilyResponse
// @Failure 400 {object} map[string]any "Bad request"
// @Failure 401 {object} map[string]any "Unauthorized"
// @Router /api/families [post]
// @Security BearerAuth
func (h *FamilyHandler) Create(e *core.RequestEvent) error {
	var req domain.CreateFamilyRequest
	if err := e.BindBody(&req); err != nil {
		return e.BadRequestError("Invalid request body", err)
	}

	response, err := h.service.CreateFamily(&req, e.Auth.Id)
	if err != nil {
		return e.BadRequestError("Failed to create family", err)
	}

	return e.JSON(http.StatusCreated, response)
}

// Join handles POST /api/families/join — adds the authenticated user to an existing family via invite code.
// @Summary Join an existing family
// @Description Adds the authenticated user to an existing family using an invite code.
// @Tags families
// @Accept json
// @Produce json
// @Param body body domain.JoinFamilyRequest true "Join family payload"
// @Success 200 {object} domain.FamilyDTO
// @Failure 400 {object} map[string]any "Bad request"
// @Failure 401 {object} map[string]any "Unauthorized"
// @Failure 404 {object} map[string]any "Family not found (invalid invite code)"
// @Router /api/families/join [post]
// @Security BearerAuth
func (h *FamilyHandler) Join(e *core.RequestEvent) error {
	var req domain.JoinFamilyRequest
	if err := e.BindBody(&req); err != nil {
		return e.BadRequestError("Invalid request body", err)
	}

	family, err := h.service.JoinFamily(&req, e.Auth.Id)
	if err != nil {
		if err.Error() == "invalid invite code" {
			return e.NotFoundError("Family not found", err)
		}

		return e.BadRequestError("Failed to join family", err)
	}

	return e.JSON(http.StatusOK, family)
}

// Leave handles POST /api/families/leave — removes the authenticated user from their current family.
// @Summary Leave current family
// @Description Removes the authenticated user from their current family.
// @Tags families
// @Accept json
// @Produce json
// @Success 204 "No content"
// @Failure 400 {object} map[string]any "Bad request"
// @Failure 401 {object} map[string]any "Unauthorized"
// @Router /api/families/leave [post]
// @Security BearerAuth
func (h *FamilyHandler) Leave(e *core.RequestEvent) error {
	if err := h.service.LeaveFamily(e.Auth.Id); err != nil {
		return e.BadRequestError("Failed to leave family", err)
	}

	return e.NoContent(http.StatusNoContent)
}
