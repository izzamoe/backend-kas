package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	digiflazzclient "kas/internal/digiflazz"
	digiflazzdomain "kas/internal/domain/digiflazz"
	"kas/internal/repository"
)

func (s *digiflazzOrderService) executePostpaidInquiry(ctx context.Context, client digiflazzclient.DigiflazzClient, familyID, createdBy, credentialID, refID string, req digiflazzdomain.CreateOrderRequest, product *digiflazzdomain.ProductDTO) (*digiflazzdomain.OrderDTO, error) { //nolint:funlen // Postpaid inquiry requires multiple validation and persistence steps
	if ctx == nil {
		ctx = context.Background()
	}
	customerNo := strings.TrimSpace(req.CustomerNo)

	var plnCustomerName string
	if strings.EqualFold(strings.TrimSpace(product.Brand), "PLN") {
		plnResp, plnErr := client.InquiryPLN(ctx, &digiflazzclient.InquiryPLNRequest{CustomerNo: customerNo})
		if plnErr == nil && plnResp != nil {
			plnCustomerName = plnResp.Name
		}
	}

	inqReq := &digiflazzclient.InqPascaRequest{
		BuyerSKUCode: product.Code,
		CustomerNo:   customerNo,
		RefID:        refID,
	}
	if req.Year != nil {
		inqReq.Year = *req.Year
	}
	// E-Money inq-pasca requires the chosen denomination in the `amount` field
	// (e.g. buyer_sku_code "emoney" with amount 22500); without it Digiflazz cannot
	// resolve the nominal. pay-pasca later references this inquiry, so it needs no amount.
	if req.Amount != nil {
		inqReq.Amount = float64(*req.Amount)
	}
	if idPelanggan2 := strings.TrimSpace(req.IDPelanggan2); idPelanggan2 != "" {
		inqReq.IdPelanggan2 = idPelanggan2
	}

	inqResp, err := client.InqPasca(ctx, inqReq)
	if err != nil {
		return nil, fmt.Errorf("digiflazz postpaid inquiry failed: %w", err)
	}
	if inqResp == nil {
		return nil, errors.New("digiflazz postpaid inquiry returned empty response")
	}

	status := classifyDigiflazzRCAndStatus(inqResp.Rc, string(inqResp.Status))
	response := transactionResponseToOrderResponse(inqResp, status)
	amount := digiflazzOrderAmount(response)
	if amount <= 0 {
		amount = product.Price + product.Admin
	}
	response.SellingPrice = amount
	responsePayload, err := json.Marshal(response)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal digiflazz inquiry response: %w", err)
	}

	order, err := s.orderRepo.Create(repository.CreateDigiflazzOrderParams{
		FamilyID:        familyID,
		UserID:          createdBy,
		CredentialID:    credentialID,
		EventType:       digiflazzdomain.EventTypeInquiry,
		RefID:           refID,
		ProductCode:     product.Code,
		ProductName:     product.Name,
		ProductBrand:    product.Brand,
		ProductCategory: product.Category,
		CustomerNo:      customerNo,
		CustomerName: func() string {
			if plnCustomerName != "" {
				return plnCustomerName
			}
			return response.CustomerName
		}(),
		Price:     response.Price,
		Admin:     response.Admin,
		Amount:    amount,
		Status:    digiflazzdomain.OrderStatusInquiry,
		Message:   response.Message,
		RC:        response.RC,
		SN:        response.SN,
		Response:  string(responsePayload),
		IsPrepaid: false,
	})
	if err != nil {
		return nil, err
	}
	if order == nil {
		return nil, errors.New("failed to create digiflazz inquiry order")
	}
	if err := s.createPostpaidOrderEvent(order, digiflazzdomain.EventTypeInquiry, "", order.Status.String(), "api", createdBy, map[string]any{"request": inqReq, "response": inqResp}, response); err != nil {
		return nil, err
	}
	return order, nil
}

func (s *digiflazzOrderService) PayPostpaidOrder(ctx context.Context, familyID, userID, orderID string) (*digiflazzdomain.OrderDTO, error) { //nolint:gocyclo // Postpaid payment flow has many validation branches
	if ctx == nil {
		ctx = context.Background()
	}
	if err := s.requireFamilyMember(familyID, userID); err != nil {
		return nil, err
	}
	order, err := s.requireDigiflazzOrder(familyID, orderID)
	if err != nil {
		return nil, err
	}
	if order.Status != digiflazzdomain.OrderStatusInquiry {
		return nil, errors.New("postpaid order must be in inquiry status before pay")
	}

	credential, err := s.credentialRepo.GetActiveSecretByFamilyID(familyID)
	if err != nil {
		return nil, fmt.Errorf("failed to get active digiflazz credential: %w", err)
	}
	if credential == nil {
		return nil, errors.New("active digiflazz credential not found")
	}
	apiKey, err := decryptDigiflazzCredentialAPIKey(credential.APIKeyCiphertext)
	if err != nil {
		return nil, err
	}
	client := s.clientFactory(credential.Username, apiKey, credential.Testing)

	payReq := &digiflazzclient.PayPascaRequest{BuyerSKUCode: order.ProductCode, CustomerNo: order.CustomerNo, RefID: order.RefID}
	payResp, err := client.PayPasca(ctx, payReq)
	if err != nil {
		return nil, fmt.Errorf("digiflazz postpaid pay failed: %w", err)
	}
	if payResp == nil {
		return nil, errors.New("digiflazz postpaid pay returned empty response")
	}
	status := classifyDigiflazzRCAndStatus(payResp.Rc, string(payResp.Status))
	response := transactionResponseToOrderResponse(payResp, status)
	payAmount := digiflazzOrderAmount(response)
	if payAmount > 0 && order.SellingPrice > 0 && math.Abs(payAmount-order.SellingPrice) > 0.000001 {
		eventErr := s.createPostpaidOrderEvent(order, digiflazzdomain.EventTypePay, order.Status.String(), order.Status.String(), "api", userID, map[string]any{"request": payReq, "response": payResp, "error": "amount changed", "inquiry_amount": order.SellingPrice, "pay_amount": payAmount}, response)
		if eventErr != nil {
			return nil, eventErr
		}
		return nil, errors.New("postpaid amount changed since inquiry; please create a fresh inquiry")
	}

	updated, err := s.updatePostpaidOrderStatus(familyID, orderID, status, response)
	if err != nil {
		return nil, err
	}
	if err := s.createPostpaidOrderEvent(updated, digiflazzdomain.EventTypePay, order.Status.String(), updated.Status.String(), "api", userID, map[string]any{"request": payReq, "response": payResp}, response); err != nil {
		return nil, err
	}
	return updated, nil
}

func (s *digiflazzOrderService) CheckPostpaidStatus(ctx context.Context, familyID, userID, orderID string) (*digiflazzdomain.OrderDTO, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := s.requireFamilyMember(familyID, userID); err != nil {
		return nil, err
	}
	order, err := s.requireDigiflazzOrder(familyID, orderID)
	if err != nil {
		return nil, err
	}
	if order.Status != digiflazzdomain.OrderStatusPending && order.Status != digiflazzdomain.OrderStatusProcessing {
		return order, nil
	}

	credential, err := s.credentialRepo.GetActiveSecretByFamilyID(familyID)
	if err != nil {
		return nil, fmt.Errorf("failed to get active digiflazz credential: %w", err)
	}
	if credential == nil {
		return nil, errors.New("active digiflazz credential not found")
	}
	apiKey, err := decryptDigiflazzCredentialAPIKey(credential.APIKeyCiphertext)
	if err != nil {
		return nil, err
	}
	client := s.clientFactory(credential.Username, apiKey, credential.Testing)

	statusReq := &digiflazzclient.StatusPascaRequest{BuyerSKUCode: order.ProductCode, CustomerNo: order.CustomerNo, RefID: order.RefID}
	statusResp, err := client.StatusPasca(ctx, statusReq)
	if err != nil {
		return nil, fmt.Errorf("digiflazz postpaid status check failed: %w", err)
	}
	if statusResp == nil {
		return nil, errors.New("digiflazz postpaid status check returned empty response")
	}
	status := classifyDigiflazzRCAndStatus(statusResp.Rc, string(statusResp.Status))
	response := transactionResponseToOrderResponse(statusResp, status)
	updated, err := s.updatePostpaidOrderStatus(familyID, orderID, status, response)
	if err != nil {
		return nil, err
	}
	if err := s.createPostpaidOrderEvent(updated, digiflazzdomain.EventTypeStatus, order.Status.String(), updated.Status.String(), "api", userID, map[string]any{"request": statusReq, "response": statusResp}, response); err != nil {
		return nil, err
	}
	return updated, nil
}

func (s *digiflazzOrderService) updatePostpaidOrderStatus(familyID, orderID string, status digiflazzdomain.OrderStatus, response *digiflazzdomain.OrderResponseDTO) (*digiflazzdomain.OrderDTO, error) {
	params := repository.UpdateDigiflazzOrderStatusParams{Status: status}
	if response != nil {
		params.Message = response.Message
		params.RC = response.RC
		params.SN = response.SN
		if encoded, err := json.Marshal(response); err == nil {
			params.Response = string(encoded)
		}
	}
	updated, err := s.orderRepo.UpdateStatus(familyID, orderID, params)
	if err != nil {
		return nil, err
	}
	if updated == nil {
		return nil, errors.New("digiflazz order not found")
	}
	return s.finalizeSuccessOrderIfConfigured(updated)
}

func (s *digiflazzOrderService) createPostpaidOrderEvent(order *digiflazzdomain.OrderDTO, eventType digiflazzdomain.EventType, before, after, source, userID string, payload any, response *digiflazzdomain.OrderResponseDTO) error {
	if s.eventRepo == nil || order == nil {
		return nil
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal digiflazz order event payload: %w", err)
	}
	processedAt := time.Now().UTC()
	event := &repository.DigiflazzEventCreateData{
		OrderID:      order.ID,
		EventType:    eventType,
		StatusBefore: before,
		StatusAfter:  after,
		Source:       source,
		Payload:      string(data),
		ProcessedAt:  &processedAt,
		CreatedBy:    userID,
	}
	if response != nil {
		event.RC = response.RC
		event.Message = response.Message
		event.SN = response.SN
	}
	if _, err := s.eventRepo.Create(event); err != nil {
		return fmt.Errorf("failed to create digiflazz order event: %w", err)
	}
	return nil
}
