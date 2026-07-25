package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

const (
	// exchangeRateURL mengembalikan kurs USD/IDR terbaru.
	// Contoh response: {"date":"2026-07-25","base":"USD","quote":"IDR","rate":17931}
	exchangeRateURL = "https://api.frankfurter.dev/v2/rate/USD/IDR"

	exchangeRateTimeout = 5 * time.Second
)

// rateFetcher mengambil kurs USD/IDR. Dibuat sebagai tipe fungsi supaya
// gampang di-stub di test tanpa memanggil API beneran.
type rateFetcher func(ctx context.Context) (float64, error)

var exchangeRateHTTPClient = &http.Client{Timeout: exchangeRateTimeout}

// defaultRateFetcher dipakai NewTransactionService. Di-override di test
// supaya unit test nggak nembak API beneran.
var defaultRateFetcher rateFetcher = fetchUSDIDRRate

type exchangeRateResponse struct {
	Date  string  `json:"date"`
	Base  string  `json:"base"`
	Quote string  `json:"quote"`
	Rate  float64 `json:"rate"`
}

// fetchUSDIDRRate mengambil kurs USD/IDR dari Frankfurter.
func fetchUSDIDRRate(ctx context.Context) (float64, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, exchangeRateURL, http.NoBody)
	if err != nil {
		return 0, fmt.Errorf("exchange rate: create request: %w", err)
	}

	resp, err := exchangeRateHTTPClient.Do(req)
	if err != nil {
		return 0, fmt.Errorf("exchange rate: do request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return 0, fmt.Errorf("exchange rate: unexpected status %d", resp.StatusCode)
	}

	var out exchangeRateResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return 0, fmt.Errorf("exchange rate: decode response: %w", err)
	}
	if out.Rate <= 0 {
		return 0, fmt.Errorf("exchange rate: invalid rate %v", out.Rate)
	}

	return out.Rate, nil
}

// convertToUSD mengembalikan nilai amount dalam USD beserta kurs yang dipakai.
// Kalau kurs gagal diambil, dua-duanya 0 (fallback, bukan error) supaya
// transaksi tetap bisa dibuat walau API kurs lagi down.
func (s *transactionService) convertToUSD(amount float64) (amountUSD, rate float64) {
	if s.fetchRate == nil {
		return 0, 0
	}

	ctx, cancel := context.WithTimeout(context.Background(), exchangeRateTimeout)
	defer cancel()

	rate, err := s.fetchRate(ctx)
	if err != nil || rate <= 0 {
		return 0, 0
	}

	return amount / rate, rate
}
