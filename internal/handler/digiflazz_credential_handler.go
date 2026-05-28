package handler

import (
	"errors"
	"net/http"
	"strings"

	digiflazzdomain "kas/internal/domain/digiflazz"
	"kas/internal/middleware"
	"kas/internal/service"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/hook"
)

// DigiflazzCredentialHandler handles HTTP requests for Digiflazz credential management.
type DigiflazzCredentialHandler struct {
	service            service.DigiflazzCredentialService
	requireAuth        *hook.Handler[*core.RequestEvent]
	requireFamily      *hook.Handler[*core.RequestEvent]
	requireFamilyOwner *hook.Handler[*core.RequestEvent]
}

type DepositRequest struct {
	Amount float64 `json:"amount"`
	Bank   string  `json:"bank"`
}

// NewDigiflazzCredentialHandler creates a new DigiflazzCredentialHandler.
func NewDigiflazzCredentialHandler(
	svc service.DigiflazzCredentialService,
	requireAuth func(*core.RequestEvent) error,
	requireFamily func(*core.RequestEvent) error,
	requireFamilyOwner func(*core.RequestEvent) error,
) *DigiflazzCredentialHandler {
	return &DigiflazzCredentialHandler{
		service:            svc,
		requireAuth:        &hook.Handler[*core.RequestEvent]{Func: requireAuth},
		requireFamily:      &hook.Handler[*core.RequestEvent]{Func: requireFamily},
		requireFamilyOwner: &hook.Handler[*core.RequestEvent]{Func: requireFamilyOwner},
	}
}

// RegisterRoutes registers all Digiflazz credential routes.
// All credential endpoints are owner-only; the service layer enforces ownership.
func (h *DigiflazzCredentialHandler) RegisterRoutes(e *core.ServeEvent) {
	e.Router.POST("/api/digiflazz/credential", h.Upsert).Bind(h.requireAuth).Bind(h.requireFamily).Bind(h.requireFamilyOwner)
	e.Router.DELETE("/api/digiflazz/credential", h.Delete).Bind(h.requireAuth).Bind(h.requireFamily).Bind(h.requireFamilyOwner)
	e.Router.POST("/api/digiflazz/credential/rotate", h.RotateToken).Bind(h.requireAuth).Bind(h.requireFamily).Bind(h.requireFamilyOwner)
	e.Router.POST("/api/digiflazz/credential/test-webhook", h.TestWebhook).Bind(h.requireAuth).Bind(h.requireFamily).Bind(h.requireFamilyOwner)
	e.Router.GET("/api/digiflazz/credential", h.Get).Bind(h.requireAuth).Bind(h.requireFamily)
	e.Router.GET("/api/digiflazz/credential/balance", h.CheckBalance).Bind(h.requireAuth).Bind(h.requireFamily)
	e.Router.POST("/api/digiflazz/deposit", h.Deposit).Bind(h.requireAuth).Bind(h.requireFamily)
}

// Get returns the safe credential metadata for the authenticated user's family.
//
//	@Summary		Get Digiflazz credential
//	@Description	Returns credential metadata (no plaintext secrets) for the authenticated family. Owner-only.
//	@Tags			digiflazz-credentials
//	@Produce		json
//	@Success		200	{object}	digiflazz.CredentialDTO
//	@Failure		401	{object}	map[string]any
//	@Failure		403	{object}	map[string]any
//	@Failure		404	{object}	map[string]any
//	@Failure		500	{object}	map[string]any
//	@Security		BearerAuth
//	@Router			/api/digiflazz/credential [get]
func (h *DigiflazzCredentialHandler) Get(e *core.RequestEvent) error {
	familyID, ok := middleware.GetFamilyIDFromContext(e.Request.Context())
	if !ok {
		return e.InternalServerError("Family context not found", nil)
	}

	dto, err := h.service.GetCredential(e.Request.Context(), familyID, e.Auth.Id)
	if err != nil {
		return mapCredentialError(e, err)
	}

	return e.JSON(http.StatusOK, dto)
}

// Upsert creates or updates the Digiflazz credential for the family.
//
//	@Summary		Create or update Digiflazz credential
//	@Description	Creates or updates the Digiflazz credential for the authenticated family. Owner-only. The family_id is always taken from the auth context and cannot be overridden via request body.
//	@Tags			digiflazz-credentials
//	@Accept			json
//	@Produce		json
//	@Param			body	body		digiflazz.UpsertCredentialRequest	true	"Credential fields — api_key is encrypted at rest; api_key_last4 is shown in responses"
//	@Success		200		{object}	digiflazz.UpsertCredentialResult
//	@Failure		400		{object}	map[string]any
//	@Failure		401		{object}	map[string]any
//	@Failure		403		{object}	map[string]any
//	@Failure		500		{object}	map[string]any
//	@Security		BearerAuth
//	@Router			/api/digiflazz/credential [post]
func (h *DigiflazzCredentialHandler) Upsert(e *core.RequestEvent) error {
	familyID, ok := middleware.GetFamilyIDFromContext(e.Request.Context())
	if !ok {
		return e.InternalServerError("Family context not found", nil)
	}

	var req digiflazzdomain.UpsertCredentialRequest
	if err := e.BindBody(&req); err != nil {
		return e.BadRequestError("Invalid request body", err)
	}

	result, err := h.service.UpsertCredential(e.Request.Context(), familyID, e.Auth.Id, req)
	if err != nil {
		return mapCredentialError(e, err)
	}

	resp := map[string]any{
		"credential":     result.Credential,
		"sync_initiated": result.SyncInitiated,
	}
	if result.RawWebhookToken != "" {
		baseURL := strings.TrimSuffix(e.App.Settings().Meta.AppURL, "/")
		resp["webhook_url"] = baseURL + "/webhooks/digiflazz/" + result.RawWebhookToken
	}

	return e.JSON(http.StatusOK, resp)
}

// Delete deletes the Digiflazz credential for the family.
//
//	@Summary		Delete Digiflazz credential
//	@Description	Permanently removes the family's Digiflazz credential. Owner-only.
//	@Tags			digiflazz-credentials
//	@Success		204
//	@Failure		401	{object}	map[string]any
//	@Failure		403	{object}	map[string]any
//	@Failure		404	{object}	map[string]any
//	@Failure		500	{object}	map[string]any
//	@Security		BearerAuth
//	@Router			/api/digiflazz/credential [delete]
func (h *DigiflazzCredentialHandler) Delete(e *core.RequestEvent) error {
	familyID, ok := middleware.GetFamilyIDFromContext(e.Request.Context())
	if !ok {
		return e.InternalServerError("Family context not found", nil)
	}

	if err := h.service.DeleteCredential(e.Request.Context(), familyID, e.Auth.Id); err != nil {
		return mapCredentialError(e, err)
	}

	return e.NoContent(http.StatusNoContent)
}

// RotateToken rotates the webhook token for the family's Digiflazz credential.
//
//	@Summary		Rotate webhook token
//	@Description	Generates a new webhook token for the family's Digiflazz credential. The new token is returned once — store it securely. Owner-only.
//	@Tags			digiflazz-credentials
//	@Produce		json
//	@Success		200	{object}	digiflazz.RotateWebhookTokenResponse
//	@Failure		401	{object}	map[string]any
//	@Failure		403	{object}	map[string]any
//	@Failure		404	{object}	map[string]any
//	@Failure		500	{object}	map[string]any
//	@Security		BearerAuth
//	@Router			/api/digiflazz/credential/rotate [post]
func (h *DigiflazzCredentialHandler) RotateToken(e *core.RequestEvent) error {
	familyID, ok := middleware.GetFamilyIDFromContext(e.Request.Context())
	if !ok {
		return e.InternalServerError("Family context not found", nil)
	}

	resp, err := h.service.RotateWebhookToken(e.Request.Context(), familyID, e.Auth.Id)
	if err != nil {
		return mapCredentialError(e, err)
	}

	baseURL := strings.TrimSuffix(e.App.Settings().Meta.AppURL, "/")
	webhookURL := baseURL + "/webhooks/digiflazz/" + resp.Token
	return e.JSON(http.StatusOK, map[string]any{
		"credential":  resp.Credential,
		"token":       resp.Token,
		"webhook_url": webhookURL,
	})
}

// TestWebhook triggers Digiflazz's dashboard webhook ping endpoint for the active credential.
//
//	@Summary		Test Digiflazz webhook
//	@Description	Calls Digiflazz POST /v1/report/hooks/{webhookID}/pings using the active family credential. Owner-only.
//	@Tags			digiflazz-credentials
//	@Produce		json
//	@Success		200	{object}	digiflazz.WebhookTestResponse
//	@Failure		400	{object}	map[string]any
//	@Failure		401	{object}	map[string]any
//	@Failure		403	{object}	map[string]any
//	@Failure		404	{object}	map[string]any
//	@Failure		500	{object}	map[string]any
//	@Security		BearerAuth
//	@Router			/api/digiflazz/credential/test-webhook [post]
func (h *DigiflazzCredentialHandler) TestWebhook(e *core.RequestEvent) error {
	familyID, ok := middleware.GetFamilyIDFromContext(e.Request.Context())
	if !ok {
		return e.InternalServerError("Family context not found", nil)
	}

	resp, err := h.service.TestWebhook(e.Request.Context(), familyID, e.Auth.Id)
	if err != nil {
		return mapCredentialError(e, err)
	}

	return e.JSON(http.StatusOK, resp)
}

// CheckBalance checks the Digiflazz deposit balance for the family.
//
//	@Summary		Get Digiflazz balance
//	@Description	Fetches the current deposit balance from Digiflazz for the authenticated family. Owner-only.
//	@Tags			digiflazz-credentials
//	@Produce		json
//	@Success		200	{object}	digiflazz.BalanceResponse
//	@Failure		401	{object}	map[string]any
//	@Failure		403	{object}	map[string]any
//	@Failure		404	{object}	map[string]any
//	@Failure		500	{object}	map[string]any
//	@Security		BearerAuth
//	@Router			/api/digiflazz/credential/balance [get]
func (h *DigiflazzCredentialHandler) CheckBalance(e *core.RequestEvent) error {
	familyID, ok := middleware.GetFamilyIDFromContext(e.Request.Context())
	if !ok {
		return e.InternalServerError("Family context not found", nil)
	}

	resp, err := h.service.CheckBalance(e.Request.Context(), familyID, e.Auth.Id)
	if err != nil {
		return mapCredentialError(e, err)
	}

	return e.JSON(http.StatusOK, resp)
}

// Deposit creates a Digiflazz deposit request for the family owner.
//
//	@Summary		Create deposit request
//	@Description	Initiates a deposit to the family's Digiflazz account. Returns bank transfer details. Owner-only.
//	@Tags			digiflazz-credentials
//	@Accept			json
//	@Produce		json
//	@Param			body	body		handler.DepositRequest				true	"Deposit amount and destination bank"
//	@Success		200		{object}	digiflazz.DigiflazzDepositResponse
//	@Failure		400		{object}	map[string]any
//	@Failure		401		{object}	map[string]any
//	@Failure		403		{object}	map[string]any
//	@Failure		500		{object}	map[string]any
//	@Security		BearerAuth
//	@Router			/api/digiflazz/deposit [post]
func (h *DigiflazzCredentialHandler) Deposit(e *core.RequestEvent) error {
	familyID, ok := middleware.GetFamilyIDFromContext(e.Request.Context())
	if !ok {
		return e.InternalServerError("Family context not found", nil)
	}

	var req DepositRequest
	if err := e.BindBody(&req); err != nil {
		return e.BadRequestError("Invalid request body", err)
	}

	resp, err := h.service.Deposit(e.Request.Context(), familyID, e.Auth.Id, req.Amount, req.Bank)
	if err != nil {
		return mapCredentialError(e, err)
	}

	return e.JSON(http.StatusOK, resp)
}

// mapCredentialError converts service errors to appropriate HTTP errors.
func mapCredentialError(e *core.RequestEvent, err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, digiflazzdomain.ErrCredentialNotFound) {
		return e.NotFoundError("Credential not found", err)
	}
	msg := err.Error()
	lower := strings.ToLower(msg)

	if strings.Contains(lower, "unauthorized") || strings.Contains(lower, "only family owner") {
		return e.ForbiddenError("Access denied", errors.New(msg))
	}
	if strings.Contains(lower, "not found") {
		return e.NotFoundError("Credential not found", err)
	}
	if strings.Contains(lower, "already exists") || strings.Contains(lower, "is required") || strings.Contains(lower, "validation failed") {
		return e.BadRequestError(msg, err)
	}
	return e.InternalServerError("Internal server error", err)
}
