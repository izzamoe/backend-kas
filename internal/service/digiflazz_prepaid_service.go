package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	digiflazzclient "kas/internal/digiflazz"
	digiflazzdomain "kas/internal/domain/digiflazz"
	"kas/internal/repository"
	"strings"
	"time"
)

func (s *digiflazzOrderService) executePrepaidOrder(ctx context.Context, client digiflazzclient.DigiflazzClient, familyID, createdBy, credentialID, refID string, req digiflazzdomain.CreateOrderRequest, product *digiflazzdomain.ProductDTO) (*digiflazzdomain.OrderDTO, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := validatePrepaidProductAvailable(product, time.Now()); err != nil {
		return nil, err
	}

	customerNo := strings.TrimSpace(req.CustomerNo)
	amount := product.Price + product.Admin

	balanceResp, balanceErr := client.CekSaldo(ctx)
	if balanceErr != nil {
		return nil, fmt.Errorf("failed to check digiflazz balance: %w", balanceErr)
	}
	if balanceResp != nil && balanceResp.Deposit < amount {
		return nil, digiflazzdomain.ErrDigiflazzInsufficientBalance
	}

	topupReq := &digiflazzclient.TopupRequest{
		BuyerSKUCode: product.Code,
		CustomerNo:   customerNo,
		RefID:        refID,
		MaxPrice:     amount,
		AllowDot:     req.AllowDot,
	}
	if req.Amount != nil {
		topupReq.MaxPrice = float64(*req.Amount)
	}
	if req.MaxPrice > 0 {
		topupReq.MaxPrice = req.MaxPrice
	}

	topupResp, topupErr := client.Topup(ctx, topupReq)
	responseDTO, finalStatus := classifyDigiflazzTopupResponse(topupResp, topupErr)
	if topupErr == nil && topupResp != nil && strings.TrimSpace(topupResp.Rc) == "03" {
		finalStatus = digiflazzdomain.OrderStatusPending
	}
	if responseAmount := digiflazzOrderAmount(responseDTO); responseAmount > 0 {
		amount = responseAmount
	}
	responsePayload, err := json.Marshal(responseDTO)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal digiflazz topup response: %w", err)
	}

	orderPrice := responseDTO.Price
	if orderPrice == 0 {
		orderPrice = product.Price
	}
	orderAdmin := responseDTO.Admin
	if orderAdmin == 0 {
		orderAdmin = product.Admin
	}

	order, err := s.orderRepo.Create(repository.CreateDigiflazzOrderParams{
		FamilyID:        familyID,
		UserID:          createdBy,
		CredentialID:    credentialID,
		EventType:       digiflazzdomain.EventTypeTopup,
		RefID:           refID,
		ProductCode:     product.Code,
		ProductName:     product.Name,
		ProductBrand:    product.Brand,
		ProductCategory: product.Category,
		CustomerNo:      customerNo,
		CustomerName:    responseDTO.CustomerName,
		Price:           orderPrice,
		Admin:           orderAdmin,
		Amount:          amount,
		Status:          finalStatus,
		Message:         responseDTO.Message,
		RC:              responseDTO.RC,
		SN:              responseDTO.SN,
		Response:        string(responsePayload),
		IsPrepaid:       true,
	})
	if err != nil {
		return nil, err
	}
	if order == nil {
		return nil, errors.New("failed to create digiflazz order")
	}
	if eventErr := s.createPrepaidOrderEvent(order, order, createdBy, topupReq, topupResp, topupErr); eventErr != nil {
		return nil, eventErr
	}
	return s.finalizeSuccessOrderIfConfigured(order)
}

func (s *digiflazzOrderService) applyPrepaidOrderStatus(familyID, orderID string, status digiflazzdomain.OrderStatus, response *digiflazzdomain.OrderResponseDTO) (*digiflazzdomain.OrderDTO, error) {
	if status == digiflazzdomain.OrderStatusPending {
		return s.UpdateStatus(familyID, orderID, digiflazzdomain.OrderStatusPending, response)
	}

	if status == digiflazzdomain.OrderStatusProcessing {
		return s.UpdateStatus(familyID, orderID, digiflazzdomain.OrderStatusProcessing, response)
	}

	processing, err := s.UpdateStatus(familyID, orderID, digiflazzdomain.OrderStatusProcessing, nil)
	if err != nil {
		return nil, err
	}
	if processing == nil {
		return nil, errors.New("digiflazz order not found")
	}
	return s.UpdateStatus(familyID, orderID, status, response)
}

func (s *digiflazzOrderService) createPrepaidOrderEvent(order, updated *digiflazzdomain.OrderDTO, userID string, topupReq *digiflazzclient.TopupRequest, topupResp *digiflazzclient.TransactionResponse, topupErr error) error {
	if s.eventRepo == nil || order == nil || updated == nil {
		return nil
	}
	payload := map[string]any{
		"request": topupReq,
	}
	if topupResp != nil {
		payload["response"] = topupResp
	}
	if topupErr != nil {
		payload["error"] = topupErr.Error()
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal digiflazz order event payload: %w", err)
	}
	processedAt := time.Now().UTC()
	_, err = s.eventRepo.Create(&repository.DigiflazzEventCreateData{
		OrderID:      order.ID,
		EventType:    digiflazzdomain.EventTypeTopup,
		StatusBefore: order.Status.String(),
		StatusAfter:  updated.Status.String(),
		Source:       "api",
		RC:           updated.RC,
		Message:      updated.Message,
		SN:           updated.SN,
		Payload:      string(data),
		ProcessedAt:  &processedAt,
		CreatedBy:    userID,
	})
	if err != nil {
		return fmt.Errorf("failed to create digiflazz order event: %w", err)
	}
	return nil
}
