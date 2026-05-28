package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	digiflazzclient "kas/internal/digiflazz"
	digiflazzdomain "kas/internal/domain/digiflazz"
	"kas/internal/middleware"
	"kas/internal/repository"
	"kas/internal/utils"
	"strings"
	"time"

	"github.com/pocketbase/pocketbase/core"
)

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
	InquiryPLN(ctx context.Context, familyID, customerNo string) (*digiflazzdomain.PLNInquiryResult, error)
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
	if errors.Is(err, digiflazzdomain.ErrProductNotFound) {
		return nil, digiflazzdomain.ErrProductNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("product lookup failed: %w", err)
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

	refID, err := generateDigiflazzOrderRefID(familyID, time.Now())
	if err != nil {
		return nil, fmt.Errorf("failed to generate ref id: %w", err)
	}
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
