package handler

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	digiflazzclient "kas/internal/digiflazz"
	digiflazzdomain "kas/internal/domain/digiflazz"
	"kas/internal/repository"
	"kas/internal/service"
	"kas/internal/utils"

	"github.com/pocketbase/pocketbase/core"
)

type digiflazzWebhookPingProbe struct {
	HookID json.RawMessage `json:"hook_id"`
	RefID  json.RawMessage `json:"ref_id"`
}

type DigiflazzWebhookCredentialRepository interface {
	GetSecretByWebhookTokenHash(tokenHash string) (*repository.DigiflazzCredentialRecord, error)
}

type DigiflazzWebhookOrderRepository interface {
	GetByRefID(familyID, refID string) (*digiflazzdomain.OrderDTO, error)
}

type DigiflazzWebhookEventRepository interface {
	Create(data *repository.DigiflazzEventCreateData) (*repository.DigiflazzEventRecord, error)
	ExistsByOrderAndPayloadHash(orderID, payloadHash string) (bool, error)
}

type DigiflazzWebhookHandler struct {
	credentialRepo DigiflazzWebhookCredentialRepository
	orderRepo      DigiflazzWebhookOrderRepository
	eventRepo      DigiflazzWebhookEventRepository
	orderService   service.DigiflazzOrderService
}

func NewDigiflazzWebhookHandler(
	credentialRepo DigiflazzWebhookCredentialRepository,
	orderRepo DigiflazzWebhookOrderRepository,
	eventRepo DigiflazzWebhookEventRepository,
	orderService service.DigiflazzOrderService,
) *DigiflazzWebhookHandler {
	return &DigiflazzWebhookHandler{
		credentialRepo: credentialRepo,
		orderRepo:      orderRepo,
		eventRepo:      eventRepo,
		orderService:   orderService,
	}
}

func (h *DigiflazzWebhookHandler) RegisterRoutes(e *core.ServeEvent) {
	e.Router.POST("/webhooks/digiflazz/{token}", h.Receive)
}

// @Summary Receive Digiflazz webhook
// @Description Receives and processes Digiflazz webhook notifications. The token identifies the family credential. Signature is validated via HMAC-SHA1 over the raw body.
// @Tags digiflazz-webhook
// @Accept json
// @Produce json
// @Param token path string true "Webhook token (identifies the Digiflazz credential)"
// @Param X-Hub-Signature header string false "HMAC-SHA1 signature (format: sha1=<hex>)"
// @Param X-Digiflazz-Event header string false "Digiflazz event type"
// @Success 200 {object} map[string]string "Webhook received"
// @Failure 400 {object} map[string]any "Bad request"
// @Failure 401 {object} map[string]any "Unauthorized - invalid signature"
// @Failure 403 {object} map[string]any "Forbidden"
// @Failure 404 {object} map[string]any "Not found - invalid token"
// @Failure 500 {object} map[string]any "Internal server error"
// @Router /webhooks/digiflazz/{token} [post]
func (h *DigiflazzWebhookHandler) Receive(e *core.RequestEvent) error {
	token := strings.TrimSpace(e.Request.PathValue("token"))
	if token == "" {
		return e.NotFoundError("Webhook not found", nil)
	}

	rawBody, err := io.ReadAll(io.LimitReader(e.Request.Body, 1<<14))
	if err != nil {
		return e.BadRequestError("Invalid webhook body", err)
	}

	eventHeader := strings.TrimSpace(e.Request.Header.Get("X-Digiflazz-Event"))
	source := "webhook"
	if eventHeader != "" {
		source += ":" + eventHeader
	}

	credential, err := h.credentialRepo.GetSecretByWebhookTokenHash(utils.HashString(token))
	if err != nil {
		return e.InternalServerError("Internal server error", err)
	}
	if credential == nil {
		return e.NotFoundError("Webhook not found", nil)
	}

	var pingProbe digiflazzWebhookPingProbe
	if err := json.Unmarshal(rawBody, &pingProbe); err == nil && len(pingProbe.HookID) > 0 && len(pingProbe.RefID) == 0 {
		return h.received(e)
	}

	signatureHeader := strings.TrimSpace(e.Request.Header.Get("X-Hub-Signature"))
	if signatureHeader == "" {
		signatureHeader = strings.TrimSpace(e.Request.Header.Get("X-Digiflazz-Signature"))
	}

	if err := digiflazzclient.VerifyWebhookSignature(credential.WebhookSecret, signatureHeader, rawBody); err != nil {
		return e.UnauthorizedError("Invalid webhook signature", err)
	}

	var payload digiflazzclient.WebhookPayload
	if err := json.Unmarshal(rawBody, &payload); err != nil {
		return e.BadRequestError("Invalid webhook payload", err)
	}

	refID := strings.TrimSpace(payload.RefID)
	if refID == "" {
		return e.BadRequestError("Invalid webhook payload", errors.New("ref_id is required"))
	}

	order, err := h.orderRepo.GetByRefID(credential.FamilyID, refID)
	if err != nil {
		return e.InternalServerError("Internal server error", err)
	}
	if order == nil {
		return h.received(e)
	}
	if order.FamilyID != credential.FamilyID || order.CredentialID != credential.ID {
		return e.ForbiddenError("Access denied", nil)
	}

	payloadHash := utils.HashString(string(rawBody))
	exists, err := h.eventRepo.ExistsByOrderAndPayloadHash(order.ID, payloadHash)
	if err != nil {
		return e.InternalServerError("Internal server error", err)
	}
	if exists {
		return h.received(e)
	}

	redactedPayload, err := utils.RedactWebhookPayload(payload)
	if err != nil {
		return e.InternalServerError("Internal server error", err)
	}

	targetStatus, knownStatus := mapDigiflazzWebhookStatus(payload.Status)
	statusAfter := order.Status.String()
	if knownStatus {
		statusAfter = targetStatus.String()
	}

	processedAt := time.Now().UTC()
	if _, err := h.eventRepo.Create(&repository.DigiflazzEventCreateData{
		OrderID:      order.ID,
		EventType:    digiflazzdomain.EventTypeWebhook,
		StatusBefore: order.Status.String(),
		StatusAfter:  statusAfter,
		Source:       source,
		RC:           payload.Rc,
		Message:      payload.Message,
		SN:           payload.Sn,
		Payload:      redactedPayload,
		PayloadHash:  payloadHash,
		ProcessedAt:  &processedAt,
	}); err != nil {
		return e.InternalServerError("Internal server error", err)
	}

	if knownStatus {
		if _, err := h.applyWebhookStatus(order, targetStatus, webhookPayloadToOrderResponse(payload, targetStatus)); err != nil {
			return mapDigiflazzOrderError(e, err)
		}
	}

	return h.received(e)
}

func (h *DigiflazzWebhookHandler) received(e *core.RequestEvent) error {
	return e.JSON(http.StatusOK, map[string]string{"status": "received"})
}

func (h *DigiflazzWebhookHandler) applyWebhookStatus(order *digiflazzdomain.OrderDTO, target digiflazzdomain.OrderStatus, response *digiflazzdomain.OrderResponseDTO) (*digiflazzdomain.OrderDTO, error) {
	if order == nil || target == "" || order.Status == target || isTerminalDigiflazzOrderStatus(order.Status) {
		return order, nil
	}
	if order.Status == digiflazzdomain.OrderStatusProcessing && target == digiflazzdomain.OrderStatusPending {
		return order, nil
	}
	if order.Status == digiflazzdomain.OrderStatusPending && (target == digiflazzdomain.OrderStatusSuccess || target == digiflazzdomain.OrderStatusFailed) {
		processing, err := h.orderService.UpdateStatus(order.FamilyID, order.ID, digiflazzdomain.OrderStatusProcessing, nil)
		if err != nil {
			return nil, err
		}
		if processing != nil {
			order = processing
		}
	}
	return h.orderService.UpdateStatus(order.FamilyID, order.ID, target, response)
}

func mapDigiflazzWebhookStatus(status digiflazzclient.TransactionStatus) (digiflazzdomain.OrderStatus, bool) {
	switch strings.ToLower(strings.TrimSpace(string(status))) {
	case "success", "sukses":
		return digiflazzdomain.OrderStatusSuccess, true
	case "pending":
		return digiflazzdomain.OrderStatusPending, true
	case "failed", "error", "gagal":
		return digiflazzdomain.OrderStatusFailed, true
	default:
		return "", false
	}
}

func webhookPayloadToOrderResponse(payload digiflazzclient.WebhookPayload, status digiflazzdomain.OrderStatus) *digiflazzdomain.OrderResponseDTO {
	desc := ""
	if len(payload.Desc) > 0 {
		desc = string(payload.Desc)
	}
	return &digiflazzdomain.OrderResponseDTO{
		RefID:          payload.RefID,
		CustomerNo:     payload.CustomerNo,
		CustomerName:   payload.CustomerName,
		BuyerSKUCode:   payload.BuyerSKUCode,
		Message:        payload.Message,
		Status:         status,
		RC:             payload.Rc,
		SN:             payload.Sn,
		BuyerLastSaldo: payload.BuyerLastSaldo,
		Price:          payload.Price,
		SellingPrice:   payload.SellingPrice,
		Tele:           payload.Tele,
		Wa:             payload.Wa,
		Desc:           desc,
	}
}

func isTerminalDigiflazzOrderStatus(status digiflazzdomain.OrderStatus) bool {
	return status == digiflazzdomain.OrderStatusSuccess || status == digiflazzdomain.OrderStatusFailed || status == digiflazzdomain.OrderStatusCancelled
}
