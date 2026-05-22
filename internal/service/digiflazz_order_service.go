package service

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	digiflazzclient "kas/internal/digiflazz"
	"kas/internal/domain"
	digiflazzdomain "kas/internal/domain/digiflazz"
	"kas/internal/middleware"
	"kas/internal/repository"
	digiflazzmapper "kas/internal/service/digiflazz"
	"kas/internal/utils"
	"math"
	"math/big"
	"net"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/pocketbase/pocketbase/core"
)

const digiflazzRefIDRandomLength = 6

var digiflazzOrderAllowedTransitions = map[digiflazzdomain.OrderStatus]map[digiflazzdomain.OrderStatus]bool{
	digiflazzdomain.OrderStatusInquiry: {
		digiflazzdomain.OrderStatusPending:    true,
		digiflazzdomain.OrderStatusProcessing: true,
		digiflazzdomain.OrderStatusSuccess:    true,
		digiflazzdomain.OrderStatusFailed:     true,
		digiflazzdomain.OrderStatusCancelled:  true,
	},
	digiflazzdomain.OrderStatusPending: {
		digiflazzdomain.OrderStatusProcessing: true,
		digiflazzdomain.OrderStatusCancelled:  true,
	},
	digiflazzdomain.OrderStatusProcessing: {
		digiflazzdomain.OrderStatusSuccess: true,
		digiflazzdomain.OrderStatusFailed:  true,
	},
	digiflazzdomain.OrderStatusSuccess:   {},
	digiflazzdomain.OrderStatusFailed:    {},
	digiflazzdomain.OrderStatusCancelled: {},
}

type DigiflazzOrderService interface {
	CreateOrder(ctx context.Context, familyID, createdBy string, req digiflazzdomain.CreateOrderRequest) (*digiflazzdomain.OrderDTO, error)
	CreatePrepaidOrder(ctx context.Context, req *digiflazzdomain.CreateOrderRequest, userID, familyID string) (*digiflazzdomain.OrderDTO, error)
	CreatePostpaidInquiry(ctx context.Context, req *digiflazzdomain.CreateOrderRequest, userID, familyID string) (*digiflazzdomain.OrderDTO, error)
	PayPostpaidOrder(ctx context.Context, familyID, userID, orderID string) (*digiflazzdomain.OrderDTO, error)
	CheckPostpaidStatus(ctx context.Context, familyID, userID, orderID string) (*digiflazzdomain.OrderDTO, error)
	GetOrder(familyID, id string) (*digiflazzdomain.OrderDTO, error)
	ListFamilyOrders(familyID string, page, pageSize int) ([]*digiflazzdomain.OrderDTO, error)
	UpdateStatus(familyID, id string, status digiflazzdomain.OrderStatus, response *digiflazzdomain.OrderResponseDTO) (*digiflazzdomain.OrderDTO, error)
	FinalizeSuccessOrder(orderID string) (*digiflazzdomain.OrderDTO, error)
	CheckAndUpdateStatus(ctx context.Context, orderID string) (*digiflazzdomain.OrderDTO, error)
}

type DigiflazzOrderServiceDeps struct {
	App             core.App
	CredentialRepo  repository.DigiflazzCredentialRepository
	ProductService  DigiflazzProductService
	EventRepo       repository.DigiflazzEventRepository
	TransactionRepo repository.TransactionRepository
	CategoryRepo    repository.CategoryRepository
	ClientFactory   DigiflazzClientFactory
}

type digiflazzOrderService struct {
	orderRepo       repository.DigiflazzOrderRepository
	app             core.App
	credentialRepo  repository.DigiflazzCredentialRepository
	productService  DigiflazzProductService
	eventRepo       repository.DigiflazzEventRepository
	transactionRepo repository.TransactionRepository
	categoryRepo    repository.CategoryRepository
	clientFactory   DigiflazzClientFactory
}

func NewDigiflazzOrderService(orderRepo repository.DigiflazzOrderRepository, deps ...DigiflazzOrderServiceDeps) DigiflazzOrderService {
	svc := &digiflazzOrderService{orderRepo: orderRepo}
	if len(deps) > 0 {
		svc.app = deps[0].App
		svc.credentialRepo = deps[0].CredentialRepo
		svc.productService = deps[0].ProductService
		svc.eventRepo = deps[0].EventRepo
		svc.transactionRepo = deps[0].TransactionRepo
		svc.categoryRepo = deps[0].CategoryRepo
		svc.clientFactory = deps[0].ClientFactory
	}
	if svc.clientFactory == nil {
		svc.clientFactory = func(username, apiKey string, testing bool) digiflazzclient.DigiflazzClient {
			return digiflazzclient.NewClient(digiflazzclient.Config{Username: username, APIKey: apiKey, Testing: testing})
		}
	}
	return svc
}

func (s *digiflazzOrderService) CreateOrder(ctx context.Context, familyID, createdBy string, req digiflazzdomain.CreateOrderRequest) (*digiflazzdomain.OrderDTO, error) {
	if familyID == "" {
		return nil, errors.New("family id is required")
	}
	if createdBy == "" {
		return nil, errors.New("created by is required")
	}
	if strings.TrimSpace(req.BuyerSKUCode) == "" {
		return nil, errors.New("buyer_sku_code is required")
	}
	if strings.TrimSpace(req.CustomerNo) == "" {
		return nil, errors.New("customer_no is required")
	}
	if s.productService == nil {
		return nil, errors.New("digiflazz product service is required")
	}

	product, err := s.productService.GetProductBySKU(familyID, req.BuyerSKUCode)
	if err != nil {
		return nil, fmt.Errorf("product lookup failed: %w", err)
	}
	if product == nil {
		return nil, digiflazzdomain.ErrProductNotFound
	}

	category := strings.ToLower(strings.TrimSpace(product.Category))
	isEMoney := strings.Contains(category, "e-money") || strings.Contains(category, "emoney")
	isSAMSAT := strings.Contains(category, "samsat")

	if isEMoney {
		if req.Amount == nil || *req.Amount <= 0 || *req.Amount%1000 != 0 {
			return nil, digiflazzdomain.ErrAmountRequired
		}
	}
	if isSAMSAT {
		if strings.TrimSpace(req.IDPelanggan2) == "" {
			return nil, digiflazzdomain.ErrIDPelanggan2Required
		}
	}

	if s.credentialRepo == nil {
		return nil, errors.New("digiflazz credential repository is required")
	}
	cred, err := s.credentialRepo.GetActiveSecretByFamilyID(familyID)
	if err != nil {
		return nil, fmt.Errorf("failed to get credential: %w", err)
	}
	if cred == nil {
		return nil, errors.New("no active digiflazz credential found for family")
	}

	key, err := credentialEncryptionKey()
	if err != nil {
		return nil, err
	}
	rawAPIKey, err := utils.Decrypt(cred.APIKeyCiphertext, key)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt api key: %w", err)
	}

	refID := uuid.New().String()
	client := s.clientFactory(cred.Username, rawAPIKey, cred.Testing)

	if product.IsPrepaid {
		return s.executePrepaidOrder(ctx, client, familyID, createdBy, cred.ID, refID, req, product)
	}
	return s.executePostpaidInquiry(ctx, client, familyID, createdBy, cred.ID, refID, req, product)
}

func (s *digiflazzOrderService) CreatePrepaidOrder(ctx context.Context, req *digiflazzdomain.CreateOrderRequest, userID, familyID string) (*digiflazzdomain.OrderDTO, error) {
	if req == nil {
		return nil, errors.New("order request is required")
	}
	if err := s.requireFamilyMember(familyID, userID); err != nil {
		return nil, err
	}
	return s.CreateOrder(ctx, familyID, userID, *req)
}

func (s *digiflazzOrderService) CreatePostpaidInquiry(ctx context.Context, req *digiflazzdomain.CreateOrderRequest, userID, familyID string) (*digiflazzdomain.OrderDTO, error) {
	if req == nil {
		return nil, errors.New("order request is required")
	}
	if err := s.requireFamilyMember(familyID, userID); err != nil {
		return nil, err
	}
	return s.CreateOrder(ctx, familyID, userID, *req)
}

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
	}
	if req.Amount != nil {
		topupReq.MaxPrice = float64(*req.Amount)
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

func (s *digiflazzOrderService) executePostpaidInquiry(ctx context.Context, client digiflazzclient.DigiflazzClient, familyID, createdBy, credentialID, refID string, req digiflazzdomain.CreateOrderRequest, product *digiflazzdomain.ProductDTO) (*digiflazzdomain.OrderDTO, error) {
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
		CustomerNo: customerNo,
		CustomerName: func() string {
			if plnCustomerName != "" {
				return plnCustomerName
			}
			return response.CustomerName
		}(),
		Price:  response.Price,
		Admin:  response.Admin,
		Amount: amount,
		Status: digiflazzdomain.OrderStatusInquiry,
		Message:         response.Message,
		RC:              response.RC,
		SN:              response.SN,
		Response:        string(responsePayload),
		IsPrepaid:       false,
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

func (s *digiflazzOrderService) PayPostpaidOrder(ctx context.Context, familyID, userID, orderID string) (*digiflazzdomain.OrderDTO, error) {
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

func (s *digiflazzOrderService) GetOrder(familyID, id string) (*digiflazzdomain.OrderDTO, error) {
	if familyID == "" {
		return nil, errors.New("family id is required")
	}
	if id == "" {
		return nil, errors.New("order id is required")
	}
	return s.orderRepo.GetByID(familyID, id)
}

func (s *digiflazzOrderService) ListFamilyOrders(familyID string, page, pageSize int) ([]*digiflazzdomain.OrderDTO, error) {
	if familyID == "" {
		return nil, errors.New("family id is required")
	}
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	offset := (page - 1) * pageSize
	return s.orderRepo.ListByFamily(familyID, pageSize, offset)
}

func (s *digiflazzOrderService) UpdateStatus(familyID, id string, status digiflazzdomain.OrderStatus, response *digiflazzdomain.OrderResponseDTO) (*digiflazzdomain.OrderDTO, error) {
	if familyID == "" {
		return nil, errors.New("family id is required")
	}
	if id == "" {
		return nil, errors.New("order id is required")
	}
	if !isKnownDigiflazzOrderStatus(status) {
		return nil, fmt.Errorf("invalid digiflazz order status: %s", status)
	}

	current, err := s.orderRepo.GetByID(familyID, id)
	if err != nil {
		return nil, err
	}
	if current == nil {
		return nil, errors.New("digiflazz order not found")
	}
	if !canTransitionDigiflazzOrder(current.Status, status) {
		return nil, fmt.Errorf("invalid digiflazz order status transition: %s -> %s", current.Status, status)
	}

	params := repository.UpdateDigiflazzOrderStatusParams{Status: status}
	if response != nil {
		params.Message = response.Message
		params.RC = response.RC
		params.SN = response.SN
		if encoded, err := json.Marshal(response); err == nil {
			params.Response = string(encoded)
		}
	}

	updated, err := s.orderRepo.UpdateStatus(familyID, id, params)
	if err != nil {
		return nil, err
	}
	return s.finalizeSuccessOrderIfConfigured(updated)
}

type digiflazzOrderFinalizationRepository interface {
	GetByIDAny(id string) (*digiflazzdomain.OrderDTO, error)
	LinkTransactionIfEmpty(familyID, id, transactionID string) (*digiflazzdomain.OrderDTO, error)
}

type digiflazzCategoryLookupRepository interface {
	FindByFamilyNameAndType(familyID, name, txType string) (*repository.CategoryInfo, error)
}

type digiflazzTransactionLookupRepository interface {
	FindByFamilyUserCategoryAmountNote(familyID, userID, categoryID string, amount float64, note string) (*domain.TransactionDTO, error)
}

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

func (s *digiflazzOrderService) FinalizeSuccessOrder(orderID string) (*digiflazzdomain.OrderDTO, error) {
	orderID = strings.TrimSpace(orderID)
	if orderID == "" {
		return nil, errors.New("order id is required")
	}
	orderRepo, ok := s.orderRepo.(digiflazzOrderFinalizationRepository)
	if !ok {
		return nil, errors.New("digiflazz order repository does not support finalization")
	}
	order, err := orderRepo.GetByIDAny(orderID)
	if err != nil {
		return nil, err
	}
	if order == nil {
		return nil, errors.New("digiflazz order not found")
	}
	return s.finalizeSuccessOrder(order)
}

func (s *digiflazzOrderService) finalizeSuccessOrderIfConfigured(order *digiflazzdomain.OrderDTO) (*digiflazzdomain.OrderDTO, error) {
	if order == nil || order.Status != digiflazzdomain.OrderStatusSuccess || order.TransactionID != "" || !s.hasFinalizationDependencies() {
		return order, nil
	}
	return s.finalizeSuccessOrder(order)
}

func (s *digiflazzOrderService) hasFinalizationDependencies() bool {
	if s.transactionRepo == nil || s.categoryRepo == nil || s.orderRepo == nil {
		return false
	}
	if _, ok := s.orderRepo.(digiflazzOrderFinalizationRepository); !ok {
		return false
	}
	if _, ok := s.categoryRepo.(digiflazzCategoryLookupRepository); !ok {
		return false
	}
	if _, ok := s.transactionRepo.(digiflazzTransactionLookupRepository); !ok {
		return false
	}
	return true
}

func (s *digiflazzOrderService) finalizeSuccessOrder(order *digiflazzdomain.OrderDTO) (*digiflazzdomain.OrderDTO, error) {
	if order == nil {
		return nil, errors.New("digiflazz order is required")
	}
	if order.Status != digiflazzdomain.OrderStatusSuccess || order.TransactionID != "" {
		return order, nil
	}
	if !s.hasFinalizationDependencies() {
		return nil, errors.New("digiflazz order finalization dependencies are required")
	}

	orderRepo := s.orderRepo.(digiflazzOrderFinalizationRepository)
	transactionLookup := s.transactionRepo.(digiflazzTransactionLookupRepository)

	latest, err := orderRepo.GetByIDAny(order.ID)
	if err != nil {
		return nil, err
	}
	if latest == nil {
		return nil, errors.New("digiflazz order not found")
	}
	if latest.Status != digiflazzdomain.OrderStatusSuccess || latest.TransactionID != "" {
		return latest, nil
	}

	category, err := s.resolveDigiflazzExpenseCategory(latest)
	if err != nil {
		return nil, err
	}
	amount := latest.SellingPrice
	if amount <= 0 {
		return nil, errors.New("digiflazz order selling price is required")
	}
	userID := strings.TrimSpace(latest.CreatedBy)
	if userID == "" {
		return nil, errors.New("digiflazz order creator is required")
	}
	description := digiflazzExpenseDescription(latest)

	existing, err := transactionLookup.FindByFamilyUserCategoryAmountNote(latest.FamilyID, userID, category.ID, amount, description)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return s.linkFinalizedTransaction(orderRepo, latest, existing.ID)
	}

	transaction, err := s.transactionRepo.Create(&domain.CreateTransactionRequest{
		CategoryID: category.ID,
		Type:       domain.TransactionTypeExpense,
		Amount:     amount,
		Note:       description,
		Date:       time.Now().UTC().Format(time.RFC3339),
	}, userID, latest.FamilyID)
	if err != nil {
		return nil, err
	}
	if transaction == nil || transaction.ID == "" {
		return nil, errors.New("failed to create digiflazz expense transaction")
	}

	linked, err := s.linkFinalizedTransaction(orderRepo, latest, transaction.ID)
	if err != nil {
		return s.recoverFinalizedOrder(orderRepo, transactionLookup, latest, category.ID, amount, description, err)
	}
	return linked, nil
}

func (s *digiflazzOrderService) resolveDigiflazzExpenseCategory(order *digiflazzdomain.OrderDTO) (*repository.CategoryInfo, error) {
	categoryRepo := s.categoryRepo.(digiflazzCategoryLookupRepository)
	categoryName := mappedDigiflazzExpenseCategoryName(order)
	category, err := categoryRepo.FindByFamilyNameAndType(order.FamilyID, categoryName, string(domain.TransactionTypeExpense))
	if err != nil {
		return nil, err
	}
	if category == nil && categoryName != "Lainnya" {
		category, err = categoryRepo.FindByFamilyNameAndType(order.FamilyID, "Lainnya", string(domain.TransactionTypeExpense))
		if err != nil {
			return nil, err
		}
	}
	if category == nil {
		return nil, fmt.Errorf("expense category %q not found for family", categoryName)
	}
	return category, nil
}

func mappedDigiflazzExpenseCategoryName(order *digiflazzdomain.OrderDTO) string {
	categoryName, _ := digiflazzmapper.MapDigiflazzCategory(order.ProductCategory)
	if categoryName != "" && categoryName != "Lainnya" {
		return categoryName
	}
	brandName, _ := digiflazzmapper.MapDigiflazzCategory(order.ProductBrand)
	if brandName != "" && brandName != "Lainnya" {
		return brandName
	}
	if categoryName != "" {
		return categoryName
	}
	return "Lainnya"
}

func digiflazzExpenseDescription(order *digiflazzdomain.OrderDTO) string {
	productName := strings.TrimSpace(order.ProductName)
	if productName == "" {
		productName = strings.TrimSpace(order.ProductCode)
	}
	return fmt.Sprintf("Pembelian %s - %s", productName, strings.TrimSpace(order.CustomerNo))
}

func (s *digiflazzOrderService) linkFinalizedTransaction(orderRepo digiflazzOrderFinalizationRepository, order *digiflazzdomain.OrderDTO, transactionID string) (*digiflazzdomain.OrderDTO, error) {
	linked, err := orderRepo.LinkTransactionIfEmpty(order.FamilyID, order.ID, transactionID)
	if err != nil {
		return nil, err
	}
	if linked == nil {
		return nil, errors.New("digiflazz order not found")
	}
	return linked, nil
}

func (s *digiflazzOrderService) recoverFinalizedOrder(orderRepo digiflazzOrderFinalizationRepository, transactionLookup digiflazzTransactionLookupRepository, order *digiflazzdomain.OrderDTO, categoryID string, amount float64, description string, linkErr error) (*digiflazzdomain.OrderDTO, error) {
	latest, err := orderRepo.GetByIDAny(order.ID)
	if err != nil {
		return nil, err
	}
	if latest != nil && latest.TransactionID != "" {
		return latest, nil
	}
	existing, findErr := transactionLookup.FindByFamilyUserCategoryAmountNote(order.FamilyID, order.CreatedBy, categoryID, amount, description)
	if findErr != nil {
		return nil, findErr
	}
	if existing != nil {
		linked, retryErr := s.linkFinalizedTransaction(orderRepo, order, existing.ID)
		if retryErr == nil {
			return linked, nil
		}
	}
	return nil, linkErr
}

func (s *digiflazzOrderService) requireFamilyMember(familyID, userID string) error {
	if strings.TrimSpace(familyID) == "" {
		return errors.New("family id is required")
	}
	if strings.TrimSpace(userID) == "" {
		return errors.New("user id is required")
	}
	if s.app == nil {
		return nil
	}
	role, err := middleware.GetFamilyRole(s.app, familyID, userID)
	if err != nil {
		return err
	}
	if role == "" {
		return errors.New("unauthorized: user is not a member of this family")
	}
	return nil
}

func (s *digiflazzOrderService) requireDigiflazzOrder(familyID, orderID string) (*digiflazzdomain.OrderDTO, error) {
	if strings.TrimSpace(orderID) == "" {
		return nil, errors.New("order id is required")
	}
	order, err := s.orderRepo.GetByID(familyID, orderID)
	if err != nil {
		return nil, err
	}
	if order == nil {
		return nil, errors.New("digiflazz order not found")
	}
	return order, nil
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

func isDigiflazzPLNProduct(product *digiflazzdomain.ProductDTO) bool {
	if product == nil {
		return false
	}
	value := strings.ToLower(product.Code + " " + product.Name + " " + product.Brand + " " + product.Category)
	return strings.Contains(value, "pln") || strings.Contains(value, "listrik")
}

func digiflazzOrderAmount(response *digiflazzdomain.OrderResponseDTO) float64 {
	if response == nil {
		return 0
	}
	if response.SellingPrice > 0 {
		return response.SellingPrice
	}
	return response.Price + response.Admin
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

func canTransitionDigiflazzOrder(from, to digiflazzdomain.OrderStatus) bool {
	allowed, ok := digiflazzOrderAllowedTransitions[from]
	if !ok {
		return false
	}
	return allowed[to]
}

func isKnownDigiflazzOrderStatus(status digiflazzdomain.OrderStatus) bool {
	_, ok := digiflazzOrderAllowedTransitions[status]
	return ok
}

func validatePrepaidProductAvailable(product *digiflazzdomain.ProductDTO, now time.Time) error {
	if product == nil {
		return errors.New("product snapshot is required")
	}
	if strings.TrimSpace(product.Code) == "" {
		return fmt.Errorf("%w: product code is required", digiflazzdomain.ErrDigiflazzInvalidProduct)
	}
	status := strings.ToLower(strings.TrimSpace(product.Status))
	if status != "active" && status != "true" && status != "1" && status != "yes" {
		return fmt.Errorf("%w: product %s is inactive", digiflazzdomain.ErrDigiflazzProductUnavailable, product.Code)
	}
	if isWithinDigiflazzCutoff(product.StartCutOff, product.EndCutOff, now) {
		return fmt.Errorf("%w: product %s is in cutoff window", digiflazzdomain.ErrDigiflazzProductUnavailable, product.Code)
	}
	return nil
}

func isWithinDigiflazzCutoff(start, end string, now time.Time) bool {
	start = strings.TrimSpace(start)
	end = strings.TrimSpace(end)
	if start == "" || end == "" {
		return false
	}
	startTime, err := parseDigiflazzClock(start)
	if err != nil {
		return false
	}
	endTime, err := parseDigiflazzClock(end)
	if err != nil {
		return false
	}
	current := now.Hour()*60 + now.Minute()
	if startTime <= endTime {
		return current >= startTime && current <= endTime
	}
	return current >= startTime || current <= endTime
}

func parseDigiflazzClock(value string) (int, error) {
	parsed, err := time.Parse("15:04", value)
	if err != nil {
		return 0, err
	}
	return parsed.Hour()*60 + parsed.Minute(), nil
}

func decryptDigiflazzCredentialAPIKey(ciphertext string) (string, error) {
	key, err := credentialEncryptionKey()
	if err != nil {
		return "", err
	}
	apiKey, err := utils.Decrypt(ciphertext, key)
	if err != nil {
		return "", fmt.Errorf("failed to decrypt api key: %w", err)
	}
	return apiKey, nil
}

func classifyDigiflazzTopupResponse(resp *digiflazzclient.TransactionResponse, err error) (*digiflazzdomain.OrderResponseDTO, digiflazzdomain.OrderStatus) {
	if err != nil {
		status := digiflazzdomain.OrderStatusProcessing
		message := "digiflazz topup is processing"
		if !isTimeoutLike(err) {
			message = err.Error()
		}
		return &digiflazzdomain.OrderResponseDTO{Message: message, Status: status}, status
	}
	if resp == nil {
		status := digiflazzdomain.OrderStatusProcessing
		return &digiflazzdomain.OrderResponseDTO{Message: "empty digiflazz topup response", Status: status}, status
	}
	status := classifyDigiflazzRCAndStatus(resp.Rc, string(resp.Status))
	return transactionResponseToOrderResponse(resp, status), status
}

func classifyDigiflazzRCAndStatus(rc, status string) digiflazzdomain.OrderStatus {
	normalizedStatus := strings.ToLower(strings.TrimSpace(status))
	switch normalizedStatus {
	case "sukses", "success", "successful":
		return digiflazzdomain.OrderStatusSuccess
	case "pending", "process", "processing", "diproses":
		return digiflazzdomain.OrderStatusProcessing
	case "gagal", "failed", "failure":
		return digiflazzdomain.OrderStatusFailed
	}

	switch strings.TrimSpace(rc) {
	case "00":
		return digiflazzdomain.OrderStatusSuccess
	case "03", "99":
		return digiflazzdomain.OrderStatusProcessing
	case "", "01", "10":
		return digiflazzdomain.OrderStatusProcessing
	case "02", "04", "06", "07", "09", "40", "41", "42", "43", "44", "45", "47", "49", "50", "51", "52", "53", "54", "55", "56", "57", "58", "59", "60", "61", "62", "63", "64", "65", "66", "67", "68", "69", "70", "71", "72", "73", "74", "80", "81", "82", "83", "84", "85", "86", "87", "88":
		return digiflazzdomain.OrderStatusFailed
	default:
		return digiflazzdomain.OrderStatusProcessing
	}
}

func transactionResponseToOrderResponse(resp *digiflazzclient.TransactionResponse, status digiflazzdomain.OrderStatus) *digiflazzdomain.OrderResponseDTO {
	desc := ""
	if len(resp.Desc) > 0 {
		desc = string(resp.Desc)
	}
	return &digiflazzdomain.OrderResponseDTO{
		RefID:          resp.RefID,
		CustomerNo:     resp.CustomerNo,
		CustomerName:   resp.CustomerName,
		BuyerSKUCode:   resp.BuyerSKUCode,
		Message:        resp.Message,
		Status:         status,
		RC:             resp.Rc,
		SN:             resp.Sn,
		BuyerLastSaldo: resp.BuyerLastSaldo,
		Price:          resp.Price,
		Admin:          resp.Admin,
		SellingPrice:   resp.SellingPrice,
		Periode:        resp.Periode,
		Tele:           resp.Tele,
		Wa:             resp.Wa,
		Desc:           desc,
	}
}

func isTimeoutLike(err error) bool {
	if errors.Is(err, context.DeadlineExceeded) || os.IsTimeout(err) {
		return true
	}
	var netErr net.Error
	return errors.As(err, &netErr) && netErr.Timeout()
}

func generateDigiflazzOrderRefID(familyID string, now time.Time) (string, error) {
	familyShort := familyID
	if len(familyShort) > 6 {
		familyShort = familyShort[:6]
	}
	familyShort = strings.ToUpper(familyShort)

	randomPart, err := randomDigiflazzOrderRefPart(digiflazzRefIDRandomLength)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("DFZ-%s-%d-%s", familyShort, now.Unix(), randomPart), nil
}

func randomDigiflazzOrderRefPart(length int) (string, error) {
	const alphabet = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	buf := make([]byte, length)
	max := big.NewInt(int64(len(alphabet)))
	for i := range buf {
		n, err := rand.Int(rand.Reader, max)
		if err != nil {
			return "", fmt.Errorf("failed to generate ref_id random part: %w", err)
		}
		buf[i] = alphabet[n.Int64()]
	}

	return string(buf), nil
}
