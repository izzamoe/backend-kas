package service

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

// TestMain stubs the default rate fetcher so the package's unit tests never
// reach the live Frankfurter API.
func TestMain(m *testing.M) {
	defaultRateFetcher = func(context.Context) (float64, error) {
		return 0, errors.New("rate fetching disabled in tests")
	}
	os.Exit(m.Run())
}

func TestFetchUSDIDRRate(t *testing.T) {
	t.Run("parses rate from response", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"date":"2026-07-25","base":"USD","quote":"IDR","rate":17931}`))
		}))
		defer srv.Close()

		rate, err := fetchRateFrom(t, srv.URL)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if rate != 17931 {
			t.Fatalf("rate = %v, want 17931", rate)
		}
	})

	t.Run("errors on non-2xx", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer srv.Close()

		if _, err := fetchRateFrom(t, srv.URL); err == nil {
			t.Fatal("expected error for 500 response")
		}
	})

	t.Run("errors on zero rate", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte(`{"base":"USD","quote":"IDR","rate":0}`))
		}))
		defer srv.Close()

		if _, err := fetchRateFrom(t, srv.URL); err == nil {
			t.Fatal("expected error for zero rate")
		}
	})
}

// fetchRateFrom points fetchUSDIDRRate's client at a test server by rewriting
// the outbound host, so the real request/parse path is exercised.
func fetchRateFrom(t *testing.T, baseURL string) (float64, error) {
	t.Helper()

	original := exchangeRateHTTPClient
	t.Cleanup(func() { exchangeRateHTTPClient = original })

	exchangeRateHTTPClient = &http.Client{
		Transport: rewriteHostTransport{target: baseURL},
		Timeout:   exchangeRateTimeout,
	}

	return fetchUSDIDRRate(context.Background())
}

type rewriteHostTransport struct {
	target string
}

func (rt rewriteHostTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	redirected := req.Clone(req.Context())

	target, err := http.NewRequest(http.MethodGet, rt.target, http.NoBody)
	if err != nil {
		return nil, err
	}
	redirected.URL.Scheme = target.URL.Scheme
	redirected.URL.Host = target.URL.Host
	redirected.Host = target.URL.Host

	return http.DefaultTransport.RoundTrip(redirected)
}

func TestConvertToUSD(t *testing.T) {
	t.Run("divides amount by rate", func(t *testing.T) {
		svc := &transactionService{
			fetchRate: func(context.Context) (float64, error) { return 17931, nil },
		}

		amountUSD, rate := svc.convertToUSD(179310)
		if rate != 17931 {
			t.Fatalf("rate = %v, want 17931", rate)
		}
		if amountUSD != 10 {
			t.Fatalf("amountUSD = %v, want 10", amountUSD)
		}
	})

	t.Run("falls back to zero when fetch fails", func(t *testing.T) {
		svc := &transactionService{
			fetchRate: func(context.Context) (float64, error) { return 0, errors.New("boom") },
		}

		amountUSD, rate := svc.convertToUSD(179310)
		if amountUSD != 0 || rate != 0 {
			t.Fatalf("got (%v, %v), want (0, 0)", amountUSD, rate)
		}
	})

	t.Run("falls back to zero on non-positive rate", func(t *testing.T) {
		svc := &transactionService{
			fetchRate: func(context.Context) (float64, error) { return 0, nil },
		}

		amountUSD, rate := svc.convertToUSD(179310)
		if amountUSD != 0 || rate != 0 {
			t.Fatalf("got (%v, %v), want (0, 0)", amountUSD, rate)
		}
	})

	t.Run("falls back to zero when fetcher is nil", func(t *testing.T) {
		svc := &transactionService{}

		amountUSD, rate := svc.convertToUSD(179310)
		if amountUSD != 0 || rate != 0 {
			t.Fatalf("got (%v, %v), want (0, 0)", amountUSD, rate)
		}
	})
}
