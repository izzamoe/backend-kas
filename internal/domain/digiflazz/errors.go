package digiflazz

import (
	"errors"
	"fmt"
	"strings"
)

var (
	ErrDigiflazzOrderPending          = errors.New("digiflazz order is pending")
	ErrDigiflazzOrderProcessing       = errors.New("digiflazz order is still processing")
	ErrDigiflazzOrderSuccess          = errors.New("digiflazz order completed successfully")
	ErrDigiflazzOrderFailed           = errors.New("digiflazz order failed")
	ErrDigiflazzOrderCancelled        = errors.New("digiflazz order was canceled")
	ErrDigiflazzTimeout               = errors.New("transaksi timeout")
	ErrDigiflazzTransactionFailed     = errors.New("transaksi gagal")
	ErrDigiflazzTransactionPending    = errors.New("transaksi pending")
	ErrDigiflazzPayloadError          = errors.New("payload error")
	ErrDigiflazzInvalidSignature      = errors.New("signature tidak valid")
	ErrDigiflazzInvalidUsername       = errors.New("username tidak valid")
	ErrDigiflazzSKUInactive           = errors.New("SKU tidak ditemukan atau non-aktif")
	ErrDigiflazzInsufficientBalance   = errors.New("saldo tidak cukup")
	ErrDigiflazzIPNotAllowed          = errors.New("IP tidak dikenali")
	ErrDigiflazzRefIDNotUnique        = errors.New("ref id tidak unik")
	ErrDigiflazzDuplicateReferenceID  = ErrDigiflazzRefIDNotUnique
	ErrDigiflazzTransactionNotFound   = errors.New("transaksi tidak ditemukan")
	ErrDigiflazzRateLimit             = errors.New("limitasi pengecekan pricelist")
	ErrDigiflazzAccountBlocked        = errors.New("akun diblokir")
	ErrDigiflazzAccountNotVerified    = errors.New("akun belum diverifikasi")
	ErrDigiflazzCutoff                = errors.New("sedang cut off")
	ErrDigiflazzUnknownError          = errors.New("error tidak diketahui")
	ErrDigiflazzUnknownResponseCode   = ErrDigiflazzUnknownError
	ErrDigiflazzInvalidCustomerNumber = errors.New("digiflazz customer number is invalid")
	ErrDigiflazzInvalidProduct        = errors.New("digiflazz product code is invalid")
	ErrDigiflazzProductUnavailable    = errors.New("digiflazz product is unavailable")
	ErrDigiflazzTransactionRejected   = errors.New("digiflazz transaction was rejected")
	ErrDigiflazzSystemBusy            = errors.New("digiflazz system is busy")
	ErrProductNotFound                = errors.New("product not found for your account")
	ErrAmountRequired                 = errors.New("amount is required and must be a multiple of 1000 for E-Money products")
	ErrIDPelanggan2Required           = errors.New("id_pelanggan2 (NIK) is required for SAMSAT products")
	ErrOrderNotFound                  = errors.New("digiflazz order not found")
	ErrCredentialNotFound             = errors.New("digiflazz credential not found")
)

type DigiflazzAPIError struct {
	RC      string
	Message string
	Err     error
}

func (e *DigiflazzAPIError) Error() string {
	return fmt.Sprintf("digiflazz api error: rc=%s message=%s", e.RC, e.Message)
}

func (e *DigiflazzAPIError) Unwrap() error { return e.Err }

func MapDigiflazzRC(rc, message string) error { //nolint:gocyclo // RC code mapping requires exhaustive switch
	msg := strings.TrimSpace(message)
	switch strings.TrimSpace(rc) {
	case "00":
		return nil
	case "01":
		return &DigiflazzAPIError{RC: rc, Message: msg, Err: ErrDigiflazzTimeout}
	case "02":
		return &DigiflazzAPIError{RC: rc, Message: msg, Err: ErrDigiflazzTransactionFailed}
	case "03":
		return &DigiflazzAPIError{RC: rc, Message: msg, Err: ErrDigiflazzTransactionPending}
	case "40":
		return &DigiflazzAPIError{RC: rc, Message: msg, Err: ErrDigiflazzPayloadError}
	case "41":
		return &DigiflazzAPIError{RC: rc, Message: msg, Err: ErrDigiflazzInvalidSignature}
	case "42":
		return &DigiflazzAPIError{RC: rc, Message: msg, Err: ErrDigiflazzInvalidUsername}
	case "43":
		return &DigiflazzAPIError{RC: rc, Message: msg, Err: ErrDigiflazzSKUInactive}
	case "44":
		return &DigiflazzAPIError{RC: rc, Message: msg, Err: ErrDigiflazzInsufficientBalance}
	case "45":
		return &DigiflazzAPIError{RC: rc, Message: msg, Err: ErrDigiflazzIPNotAllowed}
	case "47":
		return &DigiflazzAPIError{RC: rc, Message: msg, Err: ErrDigiflazzTransactionRejected}
	case "49":
		return &DigiflazzAPIError{RC: rc, Message: msg, Err: ErrDigiflazzRefIDNotUnique}
	case "50":
		return &DigiflazzAPIError{RC: rc, Message: msg, Err: ErrDigiflazzTransactionNotFound}
	case "51", "52", "54":
		return &DigiflazzAPIError{RC: rc, Message: msg, Err: ErrDigiflazzInvalidCustomerNumber}
	case "53", "55":
		return &DigiflazzAPIError{RC: rc, Message: msg, Err: ErrDigiflazzProductUnavailable}
	case "58", "66":
		return &DigiflazzAPIError{RC: rc, Message: msg, Err: ErrDigiflazzCutoff}
	case "70":
		// "Timeout Dari Biller": the transaction is formed but its outcome is unknown,
		// so it is treated like rc 01 (indeterminate, reconciled later), never failed.
		return &DigiflazzAPIError{RC: rc, Message: msg, Err: ErrDigiflazzTimeout}
	case "80", "81", "88":
		return &DigiflazzAPIError{RC: rc, Message: msg, Err: ErrDigiflazzAccountBlocked}
	case "82":
		return &DigiflazzAPIError{RC: rc, Message: msg, Err: ErrDigiflazzAccountNotVerified}
	case "83", "85", "86":
		return &DigiflazzAPIError{RC: rc, Message: msg, Err: ErrDigiflazzRateLimit}
	case "84", "87":
		return &DigiflazzAPIError{RC: rc, Message: msg, Err: ErrAmountRequired}
	default:
		return &DigiflazzAPIError{RC: rc, Message: msg, Err: ErrDigiflazzUnknownError}
	}
}
