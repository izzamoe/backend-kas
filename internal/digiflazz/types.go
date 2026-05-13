package digiflazz

import (
	"encoding/json"
	"time"
)

const (
	DefaultBaseURL = "https://api.digiflazz.com/v1"
	DefaultTimeout  = 30 * time.Second
)

type Config struct {
	Username string
	APIKey   string
	BaseURL  string
	Timeout  time.Duration
}

type TransactionStatus string

const (
	StatusSukses  TransactionStatus = "Sukses"
	StatusPending TransactionStatus = "Pending"
	StatusGagal   TransactionStatus = "Gagal"
)

type CekSaldoRequest struct {
	Cmd      string `json:"cmd"`
	Username string `json:"username"`
	Sign     string `json:"sign"`
}

type PriceListRequest struct {
	Cmd      string `json:"cmd"`
	Username string `json:"username"`
	Sign     string `json:"sign"`
	Code     string `json:"code,omitempty"`
	Category string `json:"category,omitempty"`
	Brand    string `json:"brand,omitempty"`
	Type     string `json:"type,omitempty"`
}

type DepositRequest struct {
	Username  string  `json:"username"`
	Amount    float64 `json:"amount"`
	Bank      string  `json:"bank"`
	OwnerName string  `json:"owner_name"`
	Sign      string  `json:"sign"`
}

type TopupRequest struct {
	Username      string  `json:"username"`
	BuyerSKUCode  string  `json:"buyer_sku_code"`
	CustomerNo    string  `json:"customer_no"`
	RefID         string  `json:"ref_id"`
	Sign          string  `json:"sign"`
	Testing       bool    `json:"testing,omitempty"`
	MaxPrice      float64 `json:"max_price,omitempty"`
	CBURL         string  `json:"cb_url,omitempty"`
	AllowDot      bool    `json:"allow_dot,omitempty"`
}

type InqPascaRequest struct {
	Commands      string  `json:"commands"`
	Username      string  `json:"username"`
	BuyerSKUCode  string  `json:"buyer_sku_code"`
	CustomerNo    string  `json:"customer_no"`
	RefID         string  `json:"ref_id"`
	Sign          string  `json:"sign"`
	Testing       bool    `json:"testing,omitempty"`
	Year          int     `json:"year,omitempty"`
	Amount        float64 `json:"amount,omitempty"`
	IdPelanggan2  string  `json:"id_pelanggan2,omitempty"`
}

type PayPascaRequest struct {
	Commands     string `json:"commands"`
	Username     string `json:"username"`
	BuyerSKUCode string `json:"buyer_sku_code"`
	CustomerNo   string `json:"customer_no"`
	RefID        string `json:"ref_id"`
	Sign         string `json:"sign"`
	Testing      bool   `json:"testing,omitempty"`
}

type StatusPascaRequest struct {
	Commands     string `json:"commands"`
	Username     string `json:"username"`
	BuyerSKUCode string `json:"buyer_sku_code"`
	CustomerNo   string `json:"customer_no"`
	RefID        string `json:"ref_id"`
	Sign         string `json:"sign"`
}

type InquiryPLNRequest struct {
	Username   string `json:"username"`
	CustomerNo string `json:"customer_no"`
	Sign       string `json:"sign"`
}

type CekSaldoResponse struct {
	Deposit float64 `json:"deposit"`
}

type PriceListPrepaidItem struct {
	ProductName       string `json:"product_name"`
	Category          string `json:"category"`
	Brand             string `json:"brand"`
	Type              string `json:"type"`
	SellerName        string `json:"seller_name"`
	Price             float64 `json:"price"`
	BuyerSKUCode      string `json:"buyer_sku_code"`
	BuyerProductStatus string `json:"buyer_product_status"`
	SellerProductStatus string `json:"seller_product_status"`
	UnlimitedStock    bool   `json:"unlimited_stock"`
	Stock             int    `json:"stock"`
	Multi             bool   `json:"multi"`
	StartCutOff       string `json:"start_cut_off"`
	EndCutOff         string `json:"end_cut_off"`
	Desc              string `json:"desc"`
}

type PriceListPascaItem struct {
	ProductName        string  `json:"product_name"`
	Category           string  `json:"category"`
	Brand              string  `json:"brand"`
	SellerName         string  `json:"seller_name"`
	Admin              float64 `json:"admin"`
	Commission         float64 `json:"commission"`
	BuyerSKUCode       string  `json:"buyer_sku_code"`
	BuyerProductStatus  string  `json:"buyer_product_status"`
	SellerProductStatus string  `json:"seller_product_status"`
	Desc               string  `json:"desc"`
}

type DepositResponse struct {
	Rc            string  `json:"rc"`
	Bank          string  `json:"bank"`
	PaymentMethod string  `json:"payment_method"`
	AccountNo     string  `json:"account_no"`
	Notes         string  `json:"notes"`
	Amount        float64 `json:"amount"`
}

type TransactionResponse struct {
	RefID          string          `json:"ref_id"`
	CustomerNo     string          `json:"customer_no"`
	CustomerName   string          `json:"customer_name,omitempty"`
	BuyerSKUCode   string          `json:"buyer_sku_code"`
	Message        string          `json:"message"`
	Status         TransactionStatus `json:"status"`
	Rc             string          `json:"rc"`
	Sn             string          `json:"sn,omitempty"`
	BuyerLastSaldo float64         `json:"buyer_last_saldo,omitempty"`
	Price          float64         `json:"price"`
	Admin          float64         `json:"admin,omitempty"`
	SellingPrice   float64         `json:"selling_price,omitempty"`
	Periode        string          `json:"periode,omitempty"`
	Tele           string          `json:"tele,omitempty"`
	Wa             string          `json:"wa,omitempty"`
	Desc           json.RawMessage `json:"desc,omitempty"`
}

type InquiryPLNResponse struct {
	Message       string `json:"message"`
	Status        string `json:"status"`
	Rc            string `json:"rc"`
	CustomerNo    string `json:"customer_no"`
	MeterNo       string `json:"meter_no"`
	SubscriberID  string `json:"subscriber_id"`
	Name          string `json:"name"`
	SegmentPower  string `json:"segment_power"`
}
