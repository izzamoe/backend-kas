package service

import (
	"context"
	"errors"
	"fmt"
	"strings"

	digiflazzclient "kas/internal/digiflazz"
	digiflazzdomain "kas/internal/domain/digiflazz"
)

// CheckAndUpdateStatus polls the Digiflazz API for the current status of a pending or
// processing order. This is called by the cron job and does not require a family-member
// authorization check (system-level operation).
func (s *digiflazzOrderService) CheckAndUpdateStatus(ctx context.Context, orderID string) (*digiflazzdomain.OrderDTO, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	orderID = strings.TrimSpace(orderID)
	if orderID == "" {
		return nil, errors.New("order id is required")
	}
	finalizationRepo, ok := s.orderRepo.(digiflazzOrderFinalizationRepository)
	if !ok {
		return nil, errors.New("order repository does not support system-level get")
	}
	order, err := finalizationRepo.GetByIDAny(orderID)
	if err != nil {
		return nil, fmt.Errorf("digiflazz_order_svc: get order: %w", err)
	}
	if order == nil {
		return nil, nil
	}
	if order.Status != digiflazzdomain.OrderStatusPending && order.Status != digiflazzdomain.OrderStatusProcessing {
		return order, nil
	}
	if s.credentialRepo == nil {
		return nil, errors.New("credential repository is required for status check")
	}
	cred, err := s.credentialRepo.GetActiveSecretByFamilyID(order.FamilyID)
	if err != nil {
		return nil, fmt.Errorf("digiflazz_order_svc: get credential: %w", err)
	}
	if cred == nil {
		return nil, fmt.Errorf("digiflazz_order_svc: no active credential for family %s", order.FamilyID)
	}
	apiKey, err := decryptDigiflazzCredentialAPIKey(cred.APIKeyCiphertext)
	if err != nil {
		return nil, err
	}
	client := s.clientFactory(cred.Username, apiKey, cred.Testing)
	if order.EventType == digiflazzdomain.EventTypeTopup {
		return s.checkAndUpdatePrepaidStatus(ctx, order, client)
	}
	return s.checkAndUpdatePostpaidStatus(ctx, order, client)
}

func (s *digiflazzOrderService) checkAndUpdatePrepaidStatus(ctx context.Context, order *digiflazzdomain.OrderDTO, client digiflazzclient.DigiflazzClient) (*digiflazzdomain.OrderDTO, error) {
	topupReq := &digiflazzclient.TopupRequest{
		BuyerSKUCode: order.ProductCode,
		CustomerNo:   order.CustomerNo,
		RefID:        order.RefID,
	}
	topupResp, topupErr := client.Topup(ctx, topupReq)
	responseDTO, newStatus := classifyDigiflazzTopupResponse(topupResp, topupErr)
	if order.Status == digiflazzdomain.OrderStatusProcessing && newStatus == digiflazzdomain.OrderStatusProcessing {
		return order, nil
	}
	if order.Status == digiflazzdomain.OrderStatusPending {
		return s.applyPrepaidOrderStatus(order.FamilyID, order.ID, newStatus, responseDTO)
	}
	return s.UpdateStatus(order.FamilyID, order.ID, newStatus, responseDTO)
}

func (s *digiflazzOrderService) checkAndUpdatePostpaidStatus(ctx context.Context, order *digiflazzdomain.OrderDTO, client digiflazzclient.DigiflazzClient) (*digiflazzdomain.OrderDTO, error) {
	statusReq := &digiflazzclient.StatusPascaRequest{
		BuyerSKUCode: order.ProductCode,
		CustomerNo:   order.CustomerNo,
		RefID:        order.RefID,
	}
	statusResp, err := client.StatusPasca(ctx, statusReq)
	if err != nil {
		return nil, fmt.Errorf("digiflazz postpaid status check failed: %w", err)
	}
	if statusResp == nil {
		return nil, errors.New("digiflazz postpaid status check returned empty response")
	}
	newStatus := classifyDigiflazzRCAndStatus(statusResp.Rc, string(statusResp.Status))
	response := transactionResponseToOrderResponse(statusResp, newStatus)
	if order.Status == digiflazzdomain.OrderStatusProcessing && newStatus == digiflazzdomain.OrderStatusProcessing {
		return order, nil
	}
	return s.updatePostpaidOrderStatus(order.FamilyID, order.ID, newStatus, response)
}
