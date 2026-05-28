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
