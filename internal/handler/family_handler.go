package handler

import (
	"kas/internal/domain"
	"kas/internal/service"
	"net/http"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/hook"
)

type FamilyHandler struct {
	service     service.FamilyService
	requireAuth *hook.Handler[*core.RequestEvent]
}

func NewFamilyHandler(service service.FamilyService, requireAuth func(*core.RequestEvent) error) *FamilyHandler {
	return &FamilyHandler{
		service:     service,
		requireAuth: &hook.Handler[*core.RequestEvent]{Func: requireAuth},
	}
}

func (h *FamilyHandler) RegisterRoutes(e *core.ServeEvent) {
	e.Router.POST("/api/families", h.Create).Bind(h.requireAuth)
	e.Router.POST("/api/families/join", h.Join).Bind(h.requireAuth)
	e.Router.POST("/api/families/leave", h.Leave).Bind(h.requireAuth)
}

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

func (h *FamilyHandler) Leave(e *core.RequestEvent) error {
	if err := h.service.LeaveFamily(e.Auth.Id); err != nil {
		return e.BadRequestError("Failed to leave family", err)
	}

	return e.NoContent(http.StatusNoContent)
}
