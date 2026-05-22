package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	digiflazzdomain "kas/internal/domain/digiflazz"
	"kas/internal/repository"
)

type DigiflazzCronService interface {
	RunPriceSync()
	RunOrderPoll()
}

type digiflazzCronService struct {
	productService   DigiflazzProductService
	orderService     DigiflazzOrderService
	credentialRepo   repository.DigiflazzCredentialRepository
	orderRepo        repository.DigiflazzOrderRepository
	eventRepo        repository.DigiflazzEventRepository
	priceSyncLock    sync.Mutex
	orderPollLock    sync.Mutex
}

func NewDigiflazzCronService(
	productService DigiflazzProductService,
	orderService DigiflazzOrderService,
	credentialRepo repository.DigiflazzCredentialRepository,
	orderRepo repository.DigiflazzOrderRepository,
	eventRepo repository.DigiflazzEventRepository,
) DigiflazzCronService {
	return &digiflazzCronService{
		productService: productService,
		orderService:   orderService,
		credentialRepo: credentialRepo,
		orderRepo:      orderRepo,
		eventRepo:      eventRepo,
	}
}

func (s *digiflazzCronService) RunPriceSync() {
	if !s.priceSyncLock.TryLock() {
		log.Println("digiflazz price sync skipped: already running")
		return
	}
	defer s.priceSyncLock.Unlock()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	creds, err := s.credentialRepo.ListAllActive()
	if err != nil {
		log.Printf("digiflazz price sync failed: list active credentials: %v", err)
		s.recordCronEvent("", "price_sync", "", fmt.Sprintf("list credentials: %v", err))
		return
	}

	for _, cred := range creds {
		_, syncErr := s.productService.SyncPricelistWithCredential(ctx, cred)
		if syncErr != nil {
			log.Printf("digiflazz price sync failed for credential %s: %v", cred.ID, syncErr)
			s.recordCronEvent("", "price_sync", cred.ID, syncErr.Error())
		}
	}
}

func (s *digiflazzCronService) RunOrderPoll() {
	if !s.orderPollLock.TryLock() {
		log.Println("digiflazz order poll skipped: already running")
		return
	}
	defer s.orderPollLock.Unlock()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	createdAfter := time.Now().UTC().Add(-24 * time.Hour)
	orders, err := s.orderRepo.ListPendingForPoll(createdAfter, 0)
	if err != nil {
		log.Printf("digiflazz order poll failed: list pending orders: %v", err)
		s.recordCronEvent("", "order_poll", "", fmt.Sprintf("list pending orders: %v", err))
		return
	}

	for _, order := range orders {
		_, updateErr := s.orderService.CheckAndUpdateStatus(ctx, order.ID)
		if updateErr != nil {
			log.Printf("digiflazz order poll failed for order %s: %v", order.ID, updateErr)
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
		log.Printf("digiflazz cron event recording failed: %v", err)
	}
}
