package repository

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	digiflazzdomain "kas/internal/domain/digiflazz"

	"kas/generated"

	"github.com/pocketbase/pocketbase/core"
)

type CreateDigiflazzOrderParams struct {
	FamilyID        string
	UserID          string
	CredentialID    string
	EventType       digiflazzdomain.EventType
	RefID           string
	ProductCode     string
	ProductName     string
	ProductBrand    string
	ProductCategory string
	CustomerNo      string
	CustomerName    string
	Price           float64
	Admin           float64
	Amount          float64
	Status          digiflazzdomain.OrderStatus
	Message         string
	RC              string
	SN              string
	Response        string
	IsPrepaid       bool
	Note            string
}

type UpdateDigiflazzOrderStatusParams struct {
	Status   digiflazzdomain.OrderStatus
	Message  string
	RC       string
	SN       string
	Response string
}

type DigiflazzOrderRepository interface {
	Create(params CreateDigiflazzOrderParams) (*digiflazzdomain.OrderDTO, error)
	GetByID(familyID, id string) (*digiflazzdomain.OrderDTO, error)
	GetByRefID(familyID, refID string) (*digiflazzdomain.OrderDTO, error)
	UpdateStatus(familyID, id string, params UpdateDigiflazzOrderStatusParams) (*digiflazzdomain.OrderDTO, error)
	ListByFamily(familyID string, limit, offset int) ([]*digiflazzdomain.OrderDTO, error)
	ListPendingForPoll(createdAfter time.Time, limit int) ([]*digiflazzdomain.OrderDTO, error)
}

type digiflazzOrderRepo struct {
	app core.App
}

func NewDigiflazzOrderRepository(app core.App) DigiflazzOrderRepository {
	return &digiflazzOrderRepo{app: app}
}

func (r *digiflazzOrderRepo) Create(params CreateDigiflazzOrderParams) (*digiflazzdomain.OrderDTO, error) {
	existing, err := r.GetByRefID(params.FamilyID, params.RefID)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return existing, nil
	}

	proxy, err := generated.NewProxy[generated.DigiflazzOrders](r.app)
	if err != nil {
		return nil, fmt.Errorf("failed to create digiflazz order proxy: %w", err)
	}

	proxy.Record.Set("family_id", params.FamilyID)
	proxy.Record.Set("created_by", params.UserID)
	proxy.SetRefId(params.RefID)
	proxy.SetBuyerSkuCode(params.ProductCode)
	proxy.SetCustomerNo(params.CustomerNo)
	proxy.SetProductName(params.ProductName)
	proxy.SetCategory(params.ProductCategory)
	proxy.SetPrice(params.Price)
	proxy.SetAdmin(params.Admin)
	proxy.SetTotal(params.Amount)
	proxy.SetIsPrepaid(params.IsPrepaid)

	if err := validateDigiflazzOrderStatus(params.Status); err != nil {
		return nil, err
	}
	proxy.Record.Set("status", params.Status.String())
	proxy.SetMessage(params.Message)
	proxy.SetRc(params.RC)
	proxy.SetSn(params.SN)
	proxy.SetResponse(params.Response)

	payload, err := marshalDigiflazzOrderSnapshot(params)
	if err != nil {
		return nil, err
	}
	proxy.SetPayload(payload)

	if err := r.app.Save(proxy.Record); err != nil {
		if duplicate, findErr := r.GetByRefID(params.FamilyID, params.RefID); findErr == nil && duplicate != nil {
			return duplicate, nil
		}
		return nil, fmt.Errorf("failed to save digiflazz order: %w", err)
	}

	return r.recordToDTO(proxy.Record)
}

func (r *digiflazzOrderRepo) GetByID(familyID, id string) (*digiflazzdomain.OrderDTO, error) {
	record, err := r.app.FindFirstRecordByFilter(
		"digiflazz_orders",
		"id = {:id} && family_id = {:familyID}",
		map[string]any{"id": id, "familyID": familyID},
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, digiflazzdomain.ErrOrderNotFound
		}
		return nil, fmt.Errorf("failed to find digiflazz order: %w", err)
	}

	return r.recordToDTO(record)
}

func (r *digiflazzOrderRepo) GetByIDAny(id string) (*digiflazzdomain.OrderDTO, error) {
	record, err := r.app.FindFirstRecordByFilter(
		"digiflazz_orders",
		"id = {:id}",
		map[string]any{"id": id},
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, digiflazzdomain.ErrOrderNotFound
		}
		return nil, fmt.Errorf("failed to find digiflazz order: %w", err)
	}

	return r.recordToDTO(record)
}

func (r *digiflazzOrderRepo) GetByRefID(familyID, refID string) (*digiflazzdomain.OrderDTO, error) {
	record, err := r.app.FindFirstRecordByFilter(
		"digiflazz_orders",
		"family_id = {:familyID} && ref_id = {:refID}",
		map[string]any{"familyID": familyID, "refID": refID},
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to find digiflazz order by ref_id: %w", err)
	}

	return r.recordToDTO(record)
}

func (r *digiflazzOrderRepo) UpdateStatus(familyID, id string, params UpdateDigiflazzOrderStatusParams) (*digiflazzdomain.OrderDTO, error) {
	record, err := r.app.FindFirstRecordByFilter(
		"digiflazz_orders",
		"id = {:id} && family_id = {:familyID}",
		map[string]any{"id": id, "familyID": familyID},
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, digiflazzdomain.ErrOrderNotFound
		}
		return nil, fmt.Errorf("failed to find digiflazz order for status update: %w", err)
	}

	proxy, err := generated.WrapRecord[generated.DigiflazzOrders](record)
	if err != nil {
		return nil, fmt.Errorf("failed to wrap digiflazz order record: %w", err)
	}

	if err := validateDigiflazzOrderStatus(params.Status); err != nil {
		return nil, err
	}
	proxy.Record.Set("status", params.Status.String())
	if params.Message != "" {
		proxy.SetMessage(params.Message)
	}
	if params.RC != "" {
		proxy.SetRc(params.RC)
	}
	if params.SN != "" {
		proxy.SetSn(params.SN)
	}
	if params.Response != "" {
		proxy.SetResponse(params.Response)
	}

	if err := r.app.Save(proxy.Record); err != nil {
		return nil, fmt.Errorf("failed to update digiflazz order status: %w", err)
	}

	return r.recordToDTO(proxy.Record)
}

func (r *digiflazzOrderRepo) LinkTransactionIfEmpty(familyID, id, transactionID string) (*digiflazzdomain.OrderDTO, error) {
	record, err := r.app.FindFirstRecordByFilter(
		"digiflazz_orders",
		"id = {:id} && family_id = {:familyID}",
		map[string]any{"id": id, "familyID": familyID},
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, digiflazzdomain.ErrOrderNotFound
		}
		return nil, fmt.Errorf("failed to find digiflazz order for transaction link: %w", err)
	}

	if record.GetString("transaction_id") != "" {
		return r.recordToDTO(record)
	}
	record.Set("transaction_id", transactionID)
	if err := r.app.Save(record); err != nil {
		return nil, fmt.Errorf("failed to link digiflazz order transaction: %w", err)
	}

	return r.recordToDTO(record)
}

func (r *digiflazzOrderRepo) ListPendingForPoll(createdAfter time.Time, limit int) ([]*digiflazzdomain.OrderDTO, error) {
	if limit <= 0 {
		limit = 100
	}
	records, err := r.app.FindRecordsByFilter(
		"digiflazz_orders",
		"(status = 'pending' || status = 'processing') && created >= {:createdAfter}",
		"created",
		limit,
		0,
		map[string]any{"createdAfter": createdAfter},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to list pending digiflazz orders for poll: %w", err)
	}
	orders := make([]*digiflazzdomain.OrderDTO, 0, len(records))
	for _, record := range records {
		order, err := r.recordToDTO(record)
		if err != nil {
			return nil, fmt.Errorf("failed to convert digiflazz order %s: %w", record.Id, err)
		}
		orders = append(orders, order)
	}
	return orders, nil
}

func (r *digiflazzOrderRepo) ListByFamily(familyID string, limit, offset int) ([]*digiflazzdomain.OrderDTO, error) {
	records, err := r.app.FindRecordsByFilter(
		"digiflazz_orders",
		"family_id = {:familyID}",
		"-created",
		limit,
		offset,
		map[string]any{"familyID": familyID},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to list digiflazz orders: %w", err)
	}

	orders := make([]*digiflazzdomain.OrderDTO, 0, len(records))
	for _, record := range records {
		order, err := r.recordToDTO(record)
		if err != nil {
			return nil, fmt.Errorf("failed to convert digiflazz order %s: %w", record.Id, err)
		}
		orders = append(orders, order)
	}

	return orders, nil
}

type digiflazzOrderSnapshotPayload struct {
	CredentialID    string                    `json:"credential_id,omitempty"`
	EventType       digiflazzdomain.EventType `json:"event_type,omitempty"`
	ProductBrand    string                    `json:"product_brand,omitempty"`
	ProductCategory string                    `json:"product_category,omitempty"`
	CustomerName    string                    `json:"customer_name,omitempty"`
	Amount          float64                   `json:"amount,omitempty"`
	Note            string                    `json:"note,omitempty"`
}

func marshalDigiflazzOrderSnapshot(params CreateDigiflazzOrderParams) (string, error) {
	payload := digiflazzOrderSnapshotPayload{
		CredentialID:    params.CredentialID,
		EventType:       params.EventType,
		ProductBrand:    params.ProductBrand,
		ProductCategory: params.ProductCategory,
		CustomerName:    params.CustomerName,
		Amount:          params.Amount,
		Note:            params.Note,
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("failed to marshal digiflazz order snapshot: %w", err)
	}
	return string(data), nil
}

func (r *digiflazzOrderRepo) recordToDTO(record *core.Record) (*digiflazzdomain.OrderDTO, error) {
	proxy, err := generated.WrapRecord[generated.DigiflazzOrders](record)
	if err != nil {
		return nil, fmt.Errorf("failed to wrap digiflazz order record: %w", err)
	}

	status, err := domainDigiflazzStatus(record.GetString("status"))
	if err != nil {
		return nil, err
	}

	createdAt := proxy.Created().Time()
	updatedAt := proxy.Updated().Time()
	dto := &digiflazzdomain.OrderDTO{
		ID:              proxy.Id,
		FamilyID:        proxy.Record.GetString("family_id"),
		CreatedBy:       proxy.Record.GetString("created_by"),
		ProductCode:     proxy.BuyerSkuCode(),
		ProductName:     proxy.ProductName(),
		ProductCategory: proxy.Category(),
		CustomerNo:      proxy.CustomerNo(),
		RefID:           proxy.RefId(),
		Status:          status,
		Message:         proxy.Message(),
		RC:              proxy.Rc(),
		SN:              proxy.Sn(),
		Price:           proxy.Price(),
		Admin:           proxy.Admin(),
		SellingPrice:    proxy.Total(),
		TransactionID:   proxy.TransactionId(),
		RequestedAt:     createdAt,
		CreatedAt:       createdAt,
		UpdatedAt:       updatedAt,
	}

	if status == digiflazzdomain.OrderStatusSuccess || status == digiflazzdomain.OrderStatusFailed || status == digiflazzdomain.OrderStatusCancelled {
		dto.ProcessedAt = &updatedAt
	}

	var response digiflazzdomain.OrderResponseDTO
	if raw := proxy.Response(); raw != "" {
		if err := json.Unmarshal([]byte(raw), &response); err == nil {
			if dto.Message == "" {
				dto.Message = response.Message
			}
			if dto.RC == "" {
				dto.RC = response.RC
			}
			if dto.SN == "" {
				dto.SN = response.SN
			}
			dto.BuyerLastSaldo = response.BuyerLastSaldo
			dto.Periode = response.Periode
			dto.Tele = response.Tele
			dto.Wa = response.Wa
			dto.Desc = response.Desc
			if response.Price > 0 {
				dto.Price = response.Price
			}
			if response.Admin > 0 {
				dto.Admin = response.Admin
			}
			if response.SellingPrice > 0 {
				dto.SellingPrice = response.SellingPrice
			}
		}
	}

	var payload digiflazzOrderSnapshotPayload
	if raw := proxy.Payload(); raw != "" {
		if err := json.Unmarshal([]byte(raw), &payload); err == nil {
			dto.CredentialID = payload.CredentialID
			dto.EventType = payload.EventType
			dto.ProductBrand = payload.ProductBrand
			if dto.ProductCategory == "" {
				dto.ProductCategory = payload.ProductCategory
			}
			dto.CustomerName = payload.CustomerName
			if dto.SellingPrice == 0 {
				dto.SellingPrice = payload.Amount
			}
		}
	}
	applyDigiflazzOrderResponseSnapshot(dto, proxy.Response())

	return dto, nil
}

func validateDigiflazzOrderStatus(status digiflazzdomain.OrderStatus) error {
	switch status {
	case digiflazzdomain.OrderStatusInquiry:
		return nil
	case digiflazzdomain.OrderStatusPending:
		return nil
	case digiflazzdomain.OrderStatusProcessing:
		return nil
	case digiflazzdomain.OrderStatusSuccess:
		return nil
	case digiflazzdomain.OrderStatusFailed:
		return nil
	case digiflazzdomain.OrderStatusCancelled:
		return nil
	default:
		return fmt.Errorf("invalid digiflazz order status: %s", status)
	}
}

func domainDigiflazzStatus(status string) (digiflazzdomain.OrderStatus, error) {
	switch status {
	case digiflazzdomain.OrderStatusInquiry.String():
		return digiflazzdomain.OrderStatusInquiry, nil
	case digiflazzdomain.OrderStatusPending.String():
		return digiflazzdomain.OrderStatusPending, nil
	case digiflazzdomain.OrderStatusProcessing.String():
		return digiflazzdomain.OrderStatusProcessing, nil
	case digiflazzdomain.OrderStatusSuccess.String():
		return digiflazzdomain.OrderStatusSuccess, nil
	case digiflazzdomain.OrderStatusFailed.String():
		return digiflazzdomain.OrderStatusFailed, nil
	case digiflazzdomain.OrderStatusCancelled.String():
		return digiflazzdomain.OrderStatusCancelled, nil
	default:
		return "", fmt.Errorf("invalid digiflazz order status: %s", status)
	}
}

func applyDigiflazzOrderResponseSnapshot(dto *digiflazzdomain.OrderDTO, raw string) {
	if dto == nil || raw == "" {
		return
	}
	var resp digiflazzdomain.OrderResponseDTO
	if err := json.Unmarshal([]byte(raw), &resp); err != nil {
		return
	}
	if dto.CustomerName == "" {
		dto.CustomerName = resp.CustomerName
	}
	if dto.Message == "" {
		dto.Message = resp.Message
	}
	if dto.RC == "" {
		dto.RC = resp.RC
	}
	if dto.SN == "" {
		dto.SN = resp.SN
	}
	if dto.Price == 0 {
		dto.Price = resp.Price
	}
	if dto.Admin == 0 {
		dto.Admin = resp.Admin
	}
	if dto.SellingPrice == 0 {
		dto.SellingPrice = resp.SellingPrice
	}
	dto.BuyerLastSaldo = resp.BuyerLastSaldo
	dto.Periode = resp.Periode
	dto.Tele = resp.Tele
	dto.Wa = resp.Wa
	dto.Desc = resp.Desc
}
