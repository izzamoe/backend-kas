package domain

import "time"

// Domain models - business logic representation
// Berbeda dari generated proxies, ini pure business model

type TransactionType string

const (
	TransactionTypeIncome  TransactionType = "income"
	TransactionTypeExpense TransactionType = "expense"
)

// TransactionDTO adalah Data Transfer Object untuk API
type TransactionDTO struct {
	ID         string          `json:"id"`
	FamilyID   string          `json:"family_id"`
	CreatedBy  string          `json:"created_by"`
	CategoryID string          `json:"category_id"`
	Type       TransactionType `json:"type"`
	Amount     float64         `json:"amount"`
	// AmountUSD adalah Amount dikonversi ke USD memakai ExchangeRate saat transaksi dibuat.
	AmountUSD float64 `json:"amount_usd"`
	// ExchangeRate adalah kurs USD/IDR yang dipakai (0 kalau gagal ambil kurs).
	ExchangeRate float64   `json:"exchange_rate"`
	Note         string    `json:"note"`
	Date         time.Time `json:"date"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`

	// Expanded fields (optional, populated when expand is used)
	Family   *FamilyExpand   `json:"family,omitempty"`
	Category *CategoryExpand `json:"category,omitempty"`
	Creator  *UserExpand     `json:"creator,omitempty"`
}

// Expanded relation structs
type FamilyExpand struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	InviteCode string `json:"invite_code,omitempty"`
}

type CategoryExpand struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Icon      string `json:"icon"`
	Color     string `json:"color"`
	IsDefault bool   `json:"is_default"`
}

type UserExpand struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Avatar string `json:"avatar,omitempty"`
}

// CreateTransactionRequest untuk request body
type CreateTransactionRequest struct {
	CategoryID string          `json:"category_id" validate:"required"`
	Type       TransactionType `json:"type" validate:"required,oneof=income expense" enums:"income,expense"`
	Amount     float64         `json:"amount" validate:"required,gt=0"`
	// AmountUSD dan ExchangeRate diisi oleh service dari kurs live, bukan dari client.
	AmountUSD    float64 `json:"-"`
	ExchangeRate float64 `json:"-"`
	Note         string  `json:"note"`
	Date         string  `json:"date" validate:"required"` // ISO format
}

// UpdateTransactionRequest untuk update
type UpdateTransactionRequest struct {
	CategoryID string          `json:"category_id,omitempty"`
	Type       TransactionType `json:"type,omitempty" validate:"omitempty,oneof=income expense" enums:"income,expense"`
	Amount     float64         `json:"amount,omitempty"`
	// AmountUSD dan ExchangeRate dihitung ulang oleh service saat Amount berubah,
	// bukan dikirim client.
	AmountUSD    float64 `json:"-"`
	ExchangeRate float64 `json:"-"`
	Note         string  `json:"note,omitempty"`
	Date         string  `json:"date,omitempty"`
}

// TransactionListResponse wraps paginated transaction list responses.
type TransactionListResponse struct {
	Items      []*TransactionDTO `json:"items"`
	Page       int               `json:"page"`
	PerPage    int               `json:"perPage"`
	TotalItems int               `json:"totalItems"`
	TotalPages int               `json:"totalPages"`
}

// FamilyTransactionListResponse wraps paginated family transaction list responses.
type FamilyTransactionListResponse struct {
	Items    []*TransactionDTO `json:"items"`
	Page     int               `json:"page"`
	PageSize int               `json:"pageSize"`
}

// BalanceResponse represents the balance for a family.
type BalanceResponse struct {
	FamilyID string  `json:"family_id"`
	Balance  float64 `json:"balance"`
}
