package service

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"net"
	"os"
	"strings"
	"time"

	digiflazzclient "kas/internal/digiflazz"
	digiflazzdomain "kas/internal/domain/digiflazz"
	"kas/internal/utils"
)

const digiflazzRefIDRandomLength = 6

var digiflazzOrderAllowedTransitions = map[digiflazzdomain.OrderStatus]map[digiflazzdomain.OrderStatus]bool{
	digiflazzdomain.OrderStatusInquiry: {
		digiflazzdomain.OrderStatusPending:    true,
		digiflazzdomain.OrderStatusProcessing: true,
		digiflazzdomain.OrderStatusSuccess:    true,
		digiflazzdomain.OrderStatusFailed:     true,
		digiflazzdomain.OrderStatusCancelled:  true,
	},
	digiflazzdomain.OrderStatusPending: {
		digiflazzdomain.OrderStatusProcessing: true,
		digiflazzdomain.OrderStatusCancelled:  true,
	},
	digiflazzdomain.OrderStatusProcessing: {
		digiflazzdomain.OrderStatusSuccess: true,
		digiflazzdomain.OrderStatusFailed:  true,
	},
	digiflazzdomain.OrderStatusSuccess:   {},
	digiflazzdomain.OrderStatusFailed:    {},
	digiflazzdomain.OrderStatusCancelled: {},
}

func canTransitionDigiflazzOrder(from, to digiflazzdomain.OrderStatus) bool {
	allowed, ok := digiflazzOrderAllowedTransitions[from]
	if !ok {
		return false
	}
	return allowed[to]
}

func isKnownDigiflazzOrderStatus(status digiflazzdomain.OrderStatus) bool {
	_, ok := digiflazzOrderAllowedTransitions[status]
	return ok
}

func validatePrepaidProductAvailable(product *digiflazzdomain.ProductDTO, now time.Time) error {
	if product == nil {
		return errors.New("product snapshot is required")
	}
	if strings.TrimSpace(product.Code) == "" {
		return fmt.Errorf("%w: product code is required", digiflazzdomain.ErrDigiflazzInvalidProduct)
	}
	status := strings.ToLower(strings.TrimSpace(product.Status))
	if status != "active" && status != "true" && status != "1" && status != "yes" {
		return fmt.Errorf("%w: product %s is inactive", digiflazzdomain.ErrDigiflazzProductUnavailable, product.Code)
	}
	if isWithinDigiflazzCutoff(product.StartCutOff, product.EndCutOff, now) {
		return fmt.Errorf("%w: product %s is in cutoff window", digiflazzdomain.ErrDigiflazzProductUnavailable, product.Code)
	}
	return nil
}

func isWithinDigiflazzCutoff(start, end string, now time.Time) bool {
	start = strings.TrimSpace(start)
	end = strings.TrimSpace(end)
	if start == "" || end == "" {
		return false
	}
	startTime, err := parseDigiflazzClock(start)
	if err != nil {
		return false
	}
	endTime, err := parseDigiflazzClock(end)
	if err != nil {
		return false
	}
	current := now.Hour()*60 + now.Minute()
	if startTime <= endTime {
		return current >= startTime && current <= endTime
	}
	return current >= startTime || current <= endTime
}

func parseDigiflazzClock(value string) (int, error) {
	parsed, err := time.Parse("15:04", value)
	if err != nil {
		return 0, err
	}
	return parsed.Hour()*60 + parsed.Minute(), nil
}

func decryptDigiflazzCredentialAPIKey(ciphertext string) (string, error) {
	key, err := credentialEncryptionKey()
	if err != nil {
		return "", err
	}
	apiKey, err := utils.Decrypt(ciphertext, key)
	if err != nil {
		return "", fmt.Errorf("failed to decrypt api key: %w", err)
	}
	return apiKey, nil
}

func classifyDigiflazzTopupResponse(resp *digiflazzclient.TransactionResponse, err error) (*digiflazzdomain.OrderResponseDTO, digiflazzdomain.OrderStatus) {
	if err != nil {
		// For a Digiflazz API error, classify by response code so this path stays
		// consistent with classifyDigiflazzRCAndStatus: indeterminate codes (timeout
		// rc 01, pending rc 03/99) stay processing and are reconciled later by the
		// status-check cron / webhook, while definitive rejections (invalid sign, SKU
		// inactive, insufficient balance, duplicate ref_id, ...) become failed instead
		// of being polled forever. A timeout must NOT be marked failed: the transaction
		// may still complete at the biller.
		if apiErr, ok := errors.AsType[*digiflazzdomain.DigiflazzAPIError](err); ok {
			status := classifyDigiflazzRCAndStatus(apiErr.RC, "")
			message := apiErr.Message
			if message == "" {
				message = err.Error()
			}
			return &digiflazzdomain.OrderResponseDTO{Message: message, RC: apiErr.RC, Status: status}, status
		}
		// Transport/parse error (timeout, connection reset, ...): the transaction state
		// at the biller is unknown, so keep it processing rather than risk recording a
		// successful topup as failed.
		message := "digiflazz topup is processing"
		if !isTimeoutLike(err) {
			message = err.Error()
		}
		return &digiflazzdomain.OrderResponseDTO{Message: message, Status: digiflazzdomain.OrderStatusProcessing}, digiflazzdomain.OrderStatusProcessing
	}
	if resp == nil {
		status := digiflazzdomain.OrderStatusProcessing
		return &digiflazzdomain.OrderResponseDTO{Message: "empty digiflazz topup response", Status: status}, status
	}
	status := classifyDigiflazzRCAndStatus(resp.Rc, string(resp.Status))
	return transactionResponseToOrderResponse(resp, status), status
}

func classifyDigiflazzRCAndStatus(rc, status string) digiflazzdomain.OrderStatus {
	normalizedStatus := strings.ToLower(strings.TrimSpace(status))
	switch normalizedStatus {
	case "sukses", "success", "successful":
		return digiflazzdomain.OrderStatusSuccess
	case "pending", "process", "processing", "diproses":
		return digiflazzdomain.OrderStatusProcessing
	case "gagal", "failed", "failure":
		return digiflazzdomain.OrderStatusFailed
	}

	switch strings.TrimSpace(rc) {
	case "00":
		return digiflazzdomain.OrderStatusSuccess
	case "03", "99":
		return digiflazzdomain.OrderStatusProcessing
	// Timeouts leave the transaction in an indeterminate state at the biller and may
	// still complete: rc 01 (timeout) and rc 70 ("Timeout Dari Biller", which forms a
	// transaction) must stay processing and be reconciled later, never failed.
	case "", "01", "10", "70":
		return digiflazzdomain.OrderStatusProcessing
	case "02", "04", "06", "07", "09", "40", "41", "42", "43", "44", "45", "47", "49", "50", "51", "52", "53", "54", "55", "56", "57", "58", "59", "60", "61", "62", "63", "64", "65", "66", "67", "68", "69", "71", "72", "73", "74", "80", "81", "82", "83", "84", "85", "86", "87", "88":
		return digiflazzdomain.OrderStatusFailed
	default:
		return digiflazzdomain.OrderStatusProcessing
	}
}

func transactionResponseToOrderResponse(resp *digiflazzclient.TransactionResponse, status digiflazzdomain.OrderStatus) *digiflazzdomain.OrderResponseDTO {
	desc := ""
	if len(resp.Desc) > 0 {
		desc = string(resp.Desc)
	}
	return &digiflazzdomain.OrderResponseDTO{
		RefID:          resp.RefID,
		CustomerNo:     resp.CustomerNo,
		CustomerName:   resp.CustomerName,
		BuyerSKUCode:   resp.BuyerSKUCode,
		Message:        resp.Message,
		Status:         status,
		RC:             resp.Rc,
		SN:             resp.Sn,
		BuyerLastSaldo: resp.BuyerLastSaldo,
		Price:          resp.Price,
		Admin:          resp.Admin,
		SellingPrice:   resp.SellingPrice,
		Periode:        resp.Periode,
		Tele:           resp.Tele,
		Wa:             resp.Wa,
		Desc:           desc,
	}
}

func digiflazzOrderAmount(response *digiflazzdomain.OrderResponseDTO) float64 {
	if response == nil {
		return 0
	}
	if response.SellingPrice > 0 {
		return response.SellingPrice
	}
	return response.Price + response.Admin
}

func isTimeoutLike(err error) bool {
	if errors.Is(err, context.DeadlineExceeded) || os.IsTimeout(err) {
		return true
	}
	var netErr net.Error
	return errors.As(err, &netErr) && netErr.Timeout()
}

func generateDigiflazzOrderRefID(familyID string, now time.Time) (string, error) {
	familyShort := familyID
	if len(familyShort) > 6 {
		familyShort = familyShort[:6]
	}
	familyShort = strings.ToUpper(familyShort)

	randomPart, err := randomDigiflazzOrderRefPart(digiflazzRefIDRandomLength)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("DFZ-%s-%d-%s", familyShort, now.Unix(), randomPart), nil
}

func randomDigiflazzOrderRefPart(length int) (string, error) {
	const alphabet = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	buf := make([]byte, length)
	max := big.NewInt(int64(len(alphabet)))
	for i := range buf {
		n, err := rand.Int(rand.Reader, max)
		if err != nil {
			return "", fmt.Errorf("failed to generate ref_id random part: %w", err)
		}
		buf[i] = alphabet[n.Int64()]
	}

	return string(buf), nil
}

// inquiryPLN performs a standalone PLN customer lookup using the family's active credential.
func (s *digiflazzOrderService) InquiryPLN(ctx context.Context, familyID, customerNo string) (*digiflazzdomain.PLNInquiryResult, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	customerNo = strings.TrimSpace(customerNo)
	if customerNo == "" {
		return nil, errors.New("customer_no is required")
	}
	if strings.TrimSpace(familyID) == "" {
		return nil, errors.New("family_id is required")
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
	apiKey, err := decryptDigiflazzCredentialAPIKey(cred.APIKeyCiphertext)
	if err != nil {
		return nil, err
	}
	client := s.clientFactory(cred.Username, apiKey, cred.Testing)
	resp, err := client.InquiryPLN(ctx, &digiflazzclient.InquiryPLNRequest{CustomerNo: customerNo})
	if err != nil {
		return nil, fmt.Errorf("pln inquiry failed: %w", err)
	}
	if resp == nil {
		return nil, errors.New("empty pln inquiry response")
	}
	return &digiflazzdomain.PLNInquiryResult{
		CustomerNo:   resp.CustomerNo,
		MeterNo:      resp.MeterNo,
		SubscriberID: resp.SubscriberID,
		Name:         resp.Name,
		SegmentPower: resp.SegmentPower,
	}, nil
}
