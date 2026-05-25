package digiflazz

import "time"

type OrderListResponse struct {
	Items    []*OrderDTO `json:"items"`
	Page     int         `json:"page"`
	PageSize int         `json:"page_size"`
}

type OrderDTO struct {
	ID              string      `json:"id"`
	FamilyID        string      `json:"family_id"`
	CreatedBy       string      `json:"created_by,omitempty"`
	CredentialID    string      `json:"credential_id"`
	EventType       EventType   `json:"event_type"`
	ProductCode     string      `json:"product_code"`
	ProductName     string      `json:"product_name,omitempty"`
	ProductBrand    string      `json:"product_brand,omitempty"`
	ProductCategory string      `json:"product_category,omitempty"`
	CustomerNo      string      `json:"customer_no"`
	CustomerName    string      `json:"customer_name,omitempty"`
	RefID           string      `json:"ref_id"`
	Status          OrderStatus `json:"status" enums:"inquiry,pending,processing,success,failed,cancelled"`
	Message         string      `json:"message,omitempty"`
	RC              string      `json:"rc,omitempty"`
	SN              string      `json:"sn,omitempty"`
	Price           float64     `json:"price,omitempty"`
	Admin           float64     `json:"admin,omitempty"`
	SellingPrice    float64     `json:"selling_price,omitempty"`
	BuyerLastSaldo  float64     `json:"buyer_last_saldo,omitempty"`
	Periode         string      `json:"periode,omitempty"`
	Tele            string      `json:"tele,omitempty"`
	Wa              string      `json:"wa,omitempty"`
	Desc            string      `json:"desc,omitempty"`
	TransactionID   string      `json:"transaction_id,omitempty"`
	RequestedAt     time.Time   `json:"requested_at"`
	ProcessedAt     *time.Time  `json:"processed_at,omitempty"`
	CreatedAt       time.Time   `json:"created_at"`
	UpdatedAt       time.Time   `json:"updated_at"`
}

type CreateOrderRequest struct {
	BuyerSKUCode string  `json:"buyer_sku_code"`
	CustomerNo   string  `json:"customer_no"`
	Amount       *int    `json:"amount,omitempty"`
	IDPelanggan2 string  `json:"id_pelanggan2,omitempty"`
	Year         *int    `json:"year,omitempty"`
	MaxPrice     float64 `json:"max_price,omitempty"`
	AllowDot     bool    `json:"allow_dot,omitempty"`
}

type PLNInquiryResult struct {
	CustomerNo   string `json:"customer_no"`
	MeterNo      string `json:"meter_no"`
	SubscriberID string `json:"subscriber_id"`
	Name         string `json:"name"`
	SegmentPower string `json:"segment_power"`
}

type TopupRequest struct {
	Username     string  `json:"username"`
	BuyerSKUCode string  `json:"buyer_sku_code"`
	CustomerNo   string  `json:"customer_no"`
	RefID        string  `json:"ref_id"`
	Testing      bool    `json:"testing,omitempty"`
	MaxPrice     float64 `json:"max_price,omitempty"`
	CBURL        string  `json:"cb_url,omitempty"`
	AllowDot     bool    `json:"allow_dot,omitempty"`
}

type InquiryRequest struct {
	Username     string  `json:"username"`
	BuyerSKUCode string  `json:"buyer_sku_code"`
	CustomerNo   string  `json:"customer_no"`
	RefID        string  `json:"ref_id"`
	Testing      bool    `json:"testing,omitempty"`
	Year         int     `json:"year,omitempty"`
	Amount       float64 `json:"amount,omitempty"`
	CustomerID2  string  `json:"customer_id2,omitempty"`
}

type PayRequest struct {
	Username     string `json:"username"`
	BuyerSKUCode string `json:"buyer_sku_code"`
	CustomerNo   string `json:"customer_no"`
	RefID        string `json:"ref_id"`
	Testing      bool   `json:"testing,omitempty"`
}

type StatusCheckRequest struct {
	Username     string `json:"username"`
	BuyerSKUCode string `json:"buyer_sku_code"`
	CustomerNo   string `json:"customer_no"`
	RefID        string `json:"ref_id"`
}

type DepositRequestDTO struct {
	FamilyID  string  `json:"family_id"`
	Amount    float64 `json:"amount"`
	Bank      string  `json:"bank"`
	OwnerName string  `json:"owner_name"`
}

type OrderResponseDTO struct {
	RefID          string      `json:"ref_id"`
	CustomerNo     string      `json:"customer_no"`
	CustomerName   string      `json:"customer_name,omitempty"`
	BuyerSKUCode   string      `json:"buyer_sku_code"`
	Message        string      `json:"message"`
	Status         OrderStatus `json:"status"`
	RC             string      `json:"rc"`
	SN             string      `json:"sn,omitempty"`
	BuyerLastSaldo float64     `json:"buyer_last_saldo,omitempty"`
	Price          float64     `json:"price"`
	Admin          float64     `json:"admin,omitempty"`
	SellingPrice   float64     `json:"selling_price,omitempty"`
	Periode        string      `json:"periode,omitempty"`
	Tele           string      `json:"tele,omitempty"`
	Wa             string      `json:"wa,omitempty"`
	Desc           string      `json:"desc,omitempty"`
}
