package service

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/pocketbase/pocketbase/core"

	digiflazzdomain "kas/internal/domain/digiflazz"
	"kas/internal/repository"
)

type DigiflazzCronService interface {
	RunPriceSync()
	RunOrderPoll()
}

type digiflazzCronService struct {
	app            core.App
	productService DigiflazzProductService
	orderService   DigiflazzOrderService
	credentialRepo repository.DigiflazzCredentialRepository
	orderRepo      repository.DigiflazzOrderRepository
	eventRepo      repository.DigiflazzEventRepository
	priceSyncLock  sync.Mutex
	orderPollLock  sync.Mutex
}

func NewDigiflazzCronService(
	app core.App,
	productService DigiflazzProductService,
	orderService DigiflazzOrderService,
	credentialRepo repository.DigiflazzCredentialRepository,
	orderRepo repository.DigiflazzOrderRepository,
	eventRepo repository.DigiflazzEventRepository,
) DigiflazzCronService {
	return &digiflazzCronService{
		app:            app,
		productService: productService,
		orderService:   orderService,
		credentialRepo: credentialRepo,
		orderRepo:      orderRepo,
		eventRepo:      eventRepo,
	}
}

func (s *digiflazzCronService) RunPriceSync() {
	if !s.priceSyncLock.TryLock() {
		s.logInfo("digiflazz price sync skipped: already running")
		return
	}
	defer s.priceSyncLock.Unlock()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	creds, err := s.credentialRepo.ListAllActive()
	if err != nil {
		s.logError("digiflazz price sync failed: list active credentials", "error", err)
		s.recordCronEvent("", "price_sync", "", fmt.Sprintf("list credentials: %v", err))
		return
	}

	for _, cred := range creds {
		_, syncErr := s.productService.SyncPricelistWithCredential(ctx, cred)
		if syncErr != nil {
			s.logError("digiflazz price sync failed", "credential_id", cred.ID, "error", syncErr)
			s.recordCronEvent("", "price_sync", cred.ID, syncErr.Error())
		}
	}
}

func (s *digiflazzCronService) RunOrderPoll() {
	if !s.orderPollLock.TryLock() {
		s.logInfo("digiflazz order poll skipped: already running")
		return
	}
	defer s.orderPollLock.Unlock()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	createdAfter := time.Now().UTC().Add(-24 * time.Hour)
	orders, err := s.orderRepo.ListPendingForPoll(createdAfter, 0)
	if err != nil {
		s.logError("digiflazz order poll failed: list pending orders", "error", err)
		s.recordCronEvent("", "order_poll", "", fmt.Sprintf("list pending orders: %v", err))
		return
	}

	for _, order := range orders {
		_, updateErr := s.orderService.CheckAndUpdateStatus(ctx, order.ID)
		if updateErr != nil {
			s.logError("digiflazz order poll failed", "order_id", order.ID, "error", updateErr)
			s.recordCronEvent(order.ID, "order_poll", "", updateErr.Error())
		}
	}
}

func (s *digiflazzCronService) recordCronEvent(orderID, source, credentialID, message string) {
	if s.eventRepo == nil {
		return
	}
	payload := map[string]any{
		"source":        source,
		"credential_id": credentialID,
		"message":       message,
	}
	payloadBytes, _ := json.Marshal(payload)
	processedAt := time.Now().UTC()
	_, err := s.eventRepo.Create(&repository.DigiflazzEventCreateData{
		OrderID:     orderID,
		EventType:   digiflazzdomain.EventTypeError,
		StatusAfter: "error",
		Source:      "cron",
		Message:     message,
		Payload:     string(payloadBytes),
		ProcessedAt: &processedAt,
		CreatedBy:   "system",
	})
	if err != nil {
		s.logError("digiflazz cron event recording failed", "error", err)
	}
}

func (s *digiflazzCronService) logInfo(msg string, args ...any) {
	if s.app != nil {
		s.app.Logger().Info(msg, args...)
	}
}

func (s *digiflazzCronService) logError(msg string, args ...any) {
	if s.app != nil {
		s.app.Logger().Error(msg, args...)
	}
}
