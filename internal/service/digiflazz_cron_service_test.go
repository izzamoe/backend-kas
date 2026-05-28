package service

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	digiflazzdomain "kas/internal/domain/digiflazz"
	"kas/internal/repository"
)

type fakeCronProductService struct {
	syncPricelistWithCredentialFn func(ctx context.Context, credential *repository.DigiflazzCredentialRecord) (*SyncResult, error)
}

func (f *fakeCronProductService) SyncForFamily(_ context.Context, _ string) (*SyncResult, error) {
	return nil, nil
}
func (f *fakeCronProductService) SyncPricelistWithCredential(ctx context.Context, credential *repository.DigiflazzCredentialRecord) (*SyncResult, error) {
	if f.syncPricelistWithCredentialFn != nil {
		return f.syncPricelistWithCredentialFn(ctx, credential)
	}
	return nil, nil
}
func (f *fakeCronProductService) SearchProducts(familyID string, req *digiflazzdomain.ProductSearchRequest) ([]*digiflazzdomain.ProductDTO, error) {
	return nil, nil
}
func (f *fakeCronProductService) GetProductBySKU(familyID, sku string) (*digiflazzdomain.ProductDTO, error) {
	return nil, nil
}

type fakeCronOrderService struct {
	checkAndUpdateStatusFn func(ctx context.Context, orderID string) (*digiflazzdomain.OrderDTO, error)
}

func (f *fakeCronOrderService) CreateOrder(ctx context.Context, familyID, createdBy string, req digiflazzdomain.CreateOrderRequest) (*digiflazzdomain.OrderDTO, error) {
	return nil, nil
}
func (f *fakeCronOrderService) CreatePrepaidOrder(ctx context.Context, req *digiflazzdomain.CreateOrderRequest, userID, familyID string) (*digiflazzdomain.OrderDTO, error) {
	return nil, nil
}
func (f *fakeCronOrderService) CreatePostpaidInquiry(ctx context.Context, req *digiflazzdomain.CreateOrderRequest, userID, familyID string) (*digiflazzdomain.OrderDTO, error) {
	return nil, nil
}
func (f *fakeCronOrderService) PayPostpaidOrder(ctx context.Context, familyID, userID, orderID string) (*digiflazzdomain.OrderDTO, error) {
	return nil, nil
}
func (f *fakeCronOrderService) CheckPostpaidStatus(ctx context.Context, familyID, userID, orderID string) (*digiflazzdomain.OrderDTO, error) {
	return nil, nil
}
func (f *fakeCronOrderService) GetOrder(familyID, id string) (*digiflazzdomain.OrderDTO, error) {
	return nil, nil
}
func (f *fakeCronOrderService) ListFamilyOrders(familyID string, page, pageSize int) ([]*digiflazzdomain.OrderDTO, error) {
	return nil, nil
}
func (f *fakeCronOrderService) UpdateStatus(familyID, id string, status digiflazzdomain.OrderStatus, response *digiflazzdomain.OrderResponseDTO) (*digiflazzdomain.OrderDTO, error) {
	return nil, nil
}
func (f *fakeCronOrderService) FinalizeSuccessOrder(orderID string) (*digiflazzdomain.OrderDTO, error) {
	return nil, nil
}
func (f *fakeCronOrderService) CheckAndUpdateStatus(ctx context.Context, orderID string) (*digiflazzdomain.OrderDTO, error) {
	if f.checkAndUpdateStatusFn != nil {
		return f.checkAndUpdateStatusFn(ctx, orderID)
	}
	return nil, nil
}
func (f *fakeCronOrderService) InquiryPLN(_ context.Context, _, _ string) (*digiflazzdomain.PLNInquiryResult, error) {
	return nil, nil
}

type fakeCronCredentialRepo struct {
	listAllActiveFn func() ([]*repository.DigiflazzCredentialRecord, error)
}

func (f *fakeCronCredentialRepo) Create(data *repository.DigiflazzCredentialCreateData) (*digiflazzdomain.CredentialDTO, error) {
	return nil, nil
}
func (f *fakeCronCredentialRepo) GetByID(id string) (*digiflazzdomain.CredentialDTO, error) {
	return nil, nil
}
func (f *fakeCronCredentialRepo) GetSecretByID(id string) (*repository.DigiflazzCredentialRecord, error) {
	return nil, nil
}
func (f *fakeCronCredentialRepo) GetByFamilyID(familyID string) (*digiflazzdomain.CredentialDTO, error) {
	return nil, nil
}
func (f *fakeCronCredentialRepo) GetSecretByFamilyID(familyID string) (*repository.DigiflazzCredentialRecord, error) {
	return nil, nil
}
func (f *fakeCronCredentialRepo) GetActiveByFamilyID(familyID string) (*digiflazzdomain.CredentialDTO, error) {
	return nil, nil
}
func (f *fakeCronCredentialRepo) GetActiveSecretByFamilyID(familyID string) (*repository.DigiflazzCredentialRecord, error) {
	return nil, nil
}
func (f *fakeCronCredentialRepo) ListByFamilyID(familyID string, limit, offset int) ([]*digiflazzdomain.CredentialDTO, error) {
	return nil, nil
}
func (f *fakeCronCredentialRepo) CountByFamilyID(familyID string) (int, error) {
	return 0, nil
}
func (f *fakeCronCredentialRepo) Update(id string, data *repository.DigiflazzCredentialUpdateData) (*digiflazzdomain.CredentialDTO, error) {
	return nil, nil
}
func (f *fakeCronCredentialRepo) Disable(id string) (*digiflazzdomain.CredentialDTO, error) {
	return nil, nil
}
func (f *fakeCronCredentialRepo) Delete(id string) error {
	return nil
}
func (f *fakeCronCredentialRepo) GetSecretByWebhookTokenHash(tokenHash string) (*repository.DigiflazzCredentialRecord, error) {
	return nil, nil
}
func (f *fakeCronCredentialRepo) ListAllActive() ([]*repository.DigiflazzCredentialRecord, error) {
	if f.listAllActiveFn != nil {
		return f.listAllActiveFn()
	}
	return nil, nil
}

type fakeCronOrderRepo struct {
	listPendingForPollFn func(createdAfter time.Time, limit int) ([]*digiflazzdomain.OrderDTO, error)
}

func (f *fakeCronOrderRepo) Create(params repository.CreateDigiflazzOrderParams) (*digiflazzdomain.OrderDTO, error) {
	return nil, nil
}
func (f *fakeCronOrderRepo) GetByID(familyID, id string) (*digiflazzdomain.OrderDTO, error) {
	return nil, nil
}
func (f *fakeCronOrderRepo) GetByRefID(familyID, refID string) (*digiflazzdomain.OrderDTO, error) {
	return nil, nil
}
func (f *fakeCronOrderRepo) UpdateStatus(familyID, id string, params repository.UpdateDigiflazzOrderStatusParams) (*digiflazzdomain.OrderDTO, error) {
	return nil, nil
}
func (f *fakeCronOrderRepo) ListByFamily(familyID string, limit, offset int) ([]*digiflazzdomain.OrderDTO, error) {
	return nil, nil
}
func (f *fakeCronOrderRepo) ListPendingForPoll(createdAfter time.Time, limit int) ([]*digiflazzdomain.OrderDTO, error) {
	if f.listPendingForPollFn != nil {
		return f.listPendingForPollFn(createdAfter, limit)
	}
	return nil, nil
}

type fakeCronEventRepo struct {
	createFn    func(data *repository.DigiflazzEventCreateData) (*repository.DigiflazzEventRecord, error)
	createCalls int
	createMutex sync.Mutex
}

func (f *fakeCronEventRepo) Create(data *repository.DigiflazzEventCreateData) (*repository.DigiflazzEventRecord, error) {
	f.createMutex.Lock()
	defer f.createMutex.Unlock()
	f.createCalls++
	if f.createFn != nil {
		return f.createFn(data)
	}
	return nil, nil
}
func (f *fakeCronEventRepo) GetByID(id string) (*repository.DigiflazzEventRecord, error) {
	return nil, nil
}
func (f *fakeCronEventRepo) GetByFamilyAndID(familyID, id string) (*repository.DigiflazzEventRecord, error) {
	return nil, nil
}
func (f *fakeCronEventRepo) ListByFamilyID(familyID string, limit, offset int) ([]*repository.DigiflazzEventRecord, error) {
	return nil, nil
}
func (f *fakeCronEventRepo) ExistsByOrderAndPayloadHash(orderID, payloadHash string) (bool, error) {
	return false, nil
}

func TestDigiflazzCronPriceSyncCallsServiceForEachCredential(t *testing.T) {
	var syncedIDs []string
	productSvc := &fakeCronProductService{
		syncPricelistWithCredentialFn: func(ctx context.Context, credential *repository.DigiflazzCredentialRecord) (*SyncResult, error) {
			syncedIDs = append(syncedIDs, credential.ID)
			return &SyncResult{TotalUpserted: 1}, nil
		},
	}
	credRepo := &fakeCronCredentialRepo{
		listAllActiveFn: func() ([]*repository.DigiflazzCredentialRecord, error) {
			return []*repository.DigiflazzCredentialRecord{
				{ID: "cred1", FamilyID: "fam1"},
				{ID: "cred2", FamilyID: "fam2"},
			}, nil
		},
	}
	svc := NewDigiflazzCronService(nil, productSvc, nil, credRepo, nil, nil)
	svc.RunPriceSync()

	if len(syncedIDs) != 2 {
		t.Fatalf("expected 2 credentials synced, got %d", len(syncedIDs))
	}
	if syncedIDs[0] != "cred1" || syncedIDs[1] != "cred2" {
		t.Fatalf("unexpected synced IDs: %v", syncedIDs)
	}
}

func TestDigiflazzCronPriceSyncFailSoftContinuesOnError(t *testing.T) {
	var syncedIDs []string
	productSvc := &fakeCronProductService{
		syncPricelistWithCredentialFn: func(ctx context.Context, credential *repository.DigiflazzCredentialRecord) (*SyncResult, error) {
			if credential.ID == "cred1" {
				return nil, errors.New("sync failed")
			}
			syncedIDs = append(syncedIDs, credential.ID)
			return &SyncResult{TotalUpserted: 1}, nil
		},
	}
	eventRepo := &fakeCronEventRepo{}
	credRepo := &fakeCronCredentialRepo{
		listAllActiveFn: func() ([]*repository.DigiflazzCredentialRecord, error) {
			return []*repository.DigiflazzCredentialRecord{
				{ID: "cred1", FamilyID: "fam1"},
				{ID: "cred2", FamilyID: "fam2"},
			}, nil
		},
	}
	svc := NewDigiflazzCronService(nil, productSvc, nil, credRepo, nil, eventRepo)
	svc.RunPriceSync()

	if len(syncedIDs) != 1 || syncedIDs[0] != "cred2" {
		t.Fatalf("expected only cred2 synced, got %v", syncedIDs)
	}
	if eventRepo.createCalls != 1 {
		t.Fatalf("expected 1 event recorded for failure, got %d", eventRepo.createCalls)
	}
}

func TestDigiflazzCronOrderPollFindsAndProcessesPendingOrders(t *testing.T) {
	var checkedIDs []string
	orderSvc := &fakeCronOrderService{
		checkAndUpdateStatusFn: func(ctx context.Context, orderID string) (*digiflazzdomain.OrderDTO, error) {
			checkedIDs = append(checkedIDs, orderID)
			return &digiflazzdomain.OrderDTO{ID: orderID, Status: digiflazzdomain.OrderStatusSuccess}, nil
		},
	}
	orderRepo := &fakeCronOrderRepo{
		listPendingForPollFn: func(createdAfter time.Time, limit int) ([]*digiflazzdomain.OrderDTO, error) {
			return []*digiflazzdomain.OrderDTO{
				{ID: "order1", FamilyID: "fam1"},
				{ID: "order2", FamilyID: "fam2"},
			}, nil
		},
	}
	svc := NewDigiflazzCronService(nil, nil, orderSvc, nil, orderRepo, nil)
	svc.RunOrderPoll()

	if len(checkedIDs) != 2 {
		t.Fatalf("expected 2 orders checked, got %d", len(checkedIDs))
	}
	if checkedIDs[0] != "order1" || checkedIDs[1] != "order2" {
		t.Fatalf("unexpected checked IDs: %v", checkedIDs)
	}
}

func TestDigiflazzCronOrderPollFailSoftContinuesOnError(t *testing.T) {
	var checkedIDs []string
	orderSvc := &fakeCronOrderService{
		checkAndUpdateStatusFn: func(ctx context.Context, orderID string) (*digiflazzdomain.OrderDTO, error) {
			if orderID == "order1" {
				return nil, errors.New("check failed")
			}
			checkedIDs = append(checkedIDs, orderID)
			return &digiflazzdomain.OrderDTO{ID: orderID, Status: digiflazzdomain.OrderStatusSuccess}, nil
		},
	}
	eventRepo := &fakeCronEventRepo{}
	orderRepo := &fakeCronOrderRepo{
		listPendingForPollFn: func(createdAfter time.Time, limit int) ([]*digiflazzdomain.OrderDTO, error) {
			return []*digiflazzdomain.OrderDTO{
				{ID: "order1", FamilyID: "fam1"},
				{ID: "order2", FamilyID: "fam2"},
			}, nil
		},
	}
	svc := NewDigiflazzCronService(nil, nil, orderSvc, nil, orderRepo, eventRepo)
	svc.RunOrderPoll()

	if len(checkedIDs) != 1 || checkedIDs[0] != "order2" {
		t.Fatalf("expected only order2 checked, got %v", checkedIDs)
	}
	if eventRepo.createCalls != 1 {
		t.Fatalf("expected 1 event recorded for failure, got %d", eventRepo.createCalls)
	}
}

func TestDigiflazzCronPriceSyncLockPreventsOverlap(t *testing.T) {
	var callCount int
	var mu sync.Mutex
	productSvc := &fakeCronProductService{
		syncPricelistWithCredentialFn: func(ctx context.Context, credential *repository.DigiflazzCredentialRecord) (*SyncResult, error) {
			mu.Lock()
			callCount++
			mu.Unlock()
			time.Sleep(50 * time.Millisecond)
			return &SyncResult{TotalUpserted: 1}, nil
		},
	}
	credRepo := &fakeCronCredentialRepo{
		listAllActiveFn: func() ([]*repository.DigiflazzCredentialRecord, error) {
			return []*repository.DigiflazzCredentialRecord{{ID: "cred1"}}, nil
		},
	}
	svc := NewDigiflazzCronService(nil, productSvc, nil, credRepo, nil, nil)

	var wg sync.WaitGroup
	wg.Add(2)
	go func() { defer wg.Done(); svc.RunPriceSync() }()
	go func() { defer wg.Done(); svc.RunPriceSync() }()
	wg.Wait()

	mu.Lock()
	count := callCount
	mu.Unlock()
	if count != 1 {
		t.Fatalf("expected 1 sync call due to lock, got %d", count)
	}
}

func TestDigiflazzCronOrderPollLockPreventsOverlap(t *testing.T) {
	var callCount int
	var mu sync.Mutex
	orderSvc := &fakeCronOrderService{
		checkAndUpdateStatusFn: func(ctx context.Context, orderID string) (*digiflazzdomain.OrderDTO, error) {
			mu.Lock()
			callCount++
			mu.Unlock()
			time.Sleep(50 * time.Millisecond)
			return &digiflazzdomain.OrderDTO{ID: orderID}, nil
		},
	}
	orderRepo := &fakeCronOrderRepo{
		listPendingForPollFn: func(createdAfter time.Time, limit int) ([]*digiflazzdomain.OrderDTO, error) {
			return []*digiflazzdomain.OrderDTO{{ID: "order1"}}, nil
		},
	}
	svc := NewDigiflazzCronService(nil, nil, orderSvc, nil, orderRepo, nil)

	var wg sync.WaitGroup
	wg.Add(2)
	go func() { defer wg.Done(); svc.RunOrderPoll() }()
	go func() { defer wg.Done(); svc.RunOrderPoll() }()
	wg.Wait()

	mu.Lock()
	count := callCount
	mu.Unlock()
	if count != 1 {
		t.Fatalf("expected 1 poll call due to lock, got %d", count)
	}
}
