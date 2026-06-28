package digiflazz

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

const (
	testUsername = "testuser"
	testAPIKey   = "testapikey"
)

func testClient(server *httptest.Server) DigiflazzClient {
	return NewClient(Config{
		Username: testUsername,
		APIKey:   testAPIKey,
		BaseURL:  server.URL,
		Timeout:  5 * time.Second,
	})
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}

func assertBaseRequest(t *testing.T, r *http.Request) {
	t.Helper()
	if r.Method != http.MethodPost {
		t.Errorf("method: got %s want POST", r.Method)
	}
	if ct := r.Header.Get("Content-Type"); ct != "application/json" {
		t.Errorf("Content-Type: got %q want application/json", ct)
	}
}

func assertIsDFError(t *testing.T, err error, wantCode int) {
	t.Helper()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var dfErr *DigiflazzError
	if !errors.As(err, &dfErr) {
		t.Fatalf("expected *DigiflazzError, got %T: %v", err, err)
	}
	if dfErr.StatusCode != wantCode {
		t.Errorf("StatusCode: got %d want %d", dfErr.StatusCode, wantCode)
	}
}

func TestCekSaldo_Happy(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertBaseRequest(t, r)
		var req CekSaldoRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		if req.Username != testUsername {
			t.Errorf("username: got %q want %q", req.Username, testUsername)
		}
		if req.Sign == "" {
			t.Error("expected non-empty sign")
		}
		writeJSON(w, map[string]any{"data": map[string]any{"deposit": 75000.0}})
	}))
	defer srv.Close()

	resp, err := testClient(srv).CekSaldo(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Deposit != 75000 {
		t.Errorf("deposit: got %v want 75000", resp.Deposit)
	}
}

func TestCekSaldo_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "internal server error", http.StatusInternalServerError)
	}))
	defer srv.Close()

	_, err := testClient(srv).CekSaldo(context.Background())
	assertIsDFError(t, err, 500)
}

func TestDaftarHargaPrabayar_Happy(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertBaseRequest(t, r)
		var req PriceListRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		if req.Username != testUsername {
			t.Errorf("username: got %q want %q", req.Username, testUsername)
		}
		if req.Cmd != "prepaid" {
			t.Errorf("cmd: got %q want prepaid", req.Cmd)
		}
		if req.Sign == "" {
			t.Error("expected non-empty sign")
		}
		items := []PriceListPrepaidItem{
			{ProductName: "Pulsa 10K", Category: "Pulsa", Brand: "Telkomsel", Price: 10500},
		}
		writeJSON(w, map[string]any{"data": items})
	}))
	defer srv.Close()

	items, err := testClient(srv).DaftarHargaPrabayar(context.Background(), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("len: got %d want 1", len(items))
	}
	if items[0].ProductName != "Pulsa 10K" {
		t.Errorf("product_name: got %q want Pulsa 10K", items[0].ProductName)
	}
	if items[0].Price != 10500 {
		t.Errorf("price: got %v want 10500", items[0].Price)
	}
}

func TestDaftarHargaPrabayar_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "internal server error", http.StatusInternalServerError)
	}))
	defer srv.Close()

	_, err := testClient(srv).DaftarHargaPrabayar(context.Background(), nil)
	assertIsDFError(t, err, 500)
}

func TestDaftarHargaPascabayar_Happy(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertBaseRequest(t, r)
		var req PriceListRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		if req.Username != testUsername {
			t.Errorf("username: got %q want %q", req.Username, testUsername)
		}
		if req.Cmd != "pasca" {
			t.Errorf("cmd: got %q want pasca", req.Cmd)
		}
		if req.Sign == "" {
			t.Error("expected non-empty sign")
		}
		items := []PriceListPascaItem{
			{ProductName: "PLN 100K", Category: "PLN", Brand: "PLN", Admin: 2500},
		}
		writeJSON(w, map[string]any{"data": items})
	}))
	defer srv.Close()

	items, err := testClient(srv).DaftarHargaPascabayar(context.Background(), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("len: got %d want 1", len(items))
	}
	if items[0].ProductName != "PLN 100K" {
		t.Errorf("product_name: got %q want PLN 100K", items[0].ProductName)
	}
}

func TestDaftarHargaPascabayar_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "internal server error", http.StatusInternalServerError)
	}))
	defer srv.Close()

	_, err := testClient(srv).DaftarHargaPascabayar(context.Background(), nil)
	assertIsDFError(t, err, 500)
}

func TestDeposit_Happy(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertBaseRequest(t, r)
		var req DepositRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		if req.Username != testUsername {
			t.Errorf("username: got %q want %q", req.Username, testUsername)
		}
		if req.Sign == "" {
			t.Error("expected non-empty sign")
		}
		writeJSON(w, map[string]any{"data": DepositResponse{
			Rc:            "00",
			Bank:          "BCA",
			PaymentMethod: "transfer",
			AccountNo:     "1234567890",
			Amount:        500000,
		}})
	}))
	defer srv.Close()

	req := &DepositRequest{Amount: 500000, Bank: "BCA", OwnerName: "Test User"}
	resp, err := testClient(srv).Deposit(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Bank != "BCA" {
		t.Errorf("bank: got %q want BCA", resp.Bank)
	}
	if resp.Amount != 500000 {
		t.Errorf("amount: got %v want 500000", resp.Amount)
	}
	if resp.Rc != "00" {
		t.Errorf("rc: got %q want 00", resp.Rc)
	}
}

func TestDeposit_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "internal server error", http.StatusInternalServerError)
	}))
	defer srv.Close()

	req := &DepositRequest{Amount: 100000, Bank: "BNI", OwnerName: "Test"}
	_, err := testClient(srv).Deposit(context.Background(), req)
	assertIsDFError(t, err, 500)
}

func TestTestWebhookPing_Happy(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertBaseRequest(t, r)
		if r.URL.Path != "/report/hooks/hook-123/pings" {
			t.Fatalf("path: got %q want /report/hooks/hook-123/pings", r.URL.Path)
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err == nil {
			t.Fatalf("expected empty request body, got JSON: %+v", body)
		}
		writeJSON(w, WebhookPingResponse{
			Sed:    "ping-sed",
			HookID: "hook-123",
			Hook: WebhookPingHook{
				URL:    "https://example.test/webhooks/digiflazz/token",
				Secret: "secret",
				Type:   "application/json",
				Status: 1,
			},
		})
	}))
	defer srv.Close()

	resp, err := testClient(srv).TestWebhookPing(context.Background(), "hook-123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.HookID != "hook-123" || resp.Hook.Status != 1 || resp.Hook.URL == "" {
		t.Fatalf("unexpected ping response: %+v", resp)
	}
}

func TestTestWebhookPing_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "internal server error", http.StatusInternalServerError)
	}))
	defer srv.Close()

	_, err := testClient(srv).TestWebhookPing(context.Background(), "hook-123")
	assertIsDFError(t, err, 500)
}

func TestTopup_Happy(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertBaseRequest(t, r)
		var req TopupRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		if req.Username != testUsername {
			t.Errorf("username: got %q want %q", req.Username, testUsername)
		}
		if req.Sign == "" {
			t.Error("expected non-empty sign")
		}
		writeJSON(w, map[string]any{"data": TransactionResponse{
			RefID:        req.RefID,
			CustomerNo:   req.CustomerNo,
			BuyerSKUCode: req.BuyerSKUCode,
			Status:       StatusSukses,
			Message:      "Sukses",
			Rc:           "00",
			Price:        12000,
		}})
	}))
	defer srv.Close()

	req := &TopupRequest{BuyerSKUCode: "xld10", CustomerNo: "08123456789", RefID: "ref001"}
	resp, err := testClient(srv).Topup(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Status != StatusSukses {
		t.Errorf("status: got %q want %q", resp.Status, StatusSukses)
	}
	if resp.RefID != "ref001" {
		t.Errorf("ref_id: got %q want ref001", resp.RefID)
	}
}

func TestTopup_GlobalTestingMode(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertBaseRequest(t, r)
		var req TopupRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		if !req.Testing {
			t.Errorf("expected Testing=true when global testing mode is enabled, got false")
		}
		writeJSON(w, map[string]any{"data": TransactionResponse{RefID: req.RefID, Status: StatusSukses, Rc: "00"}})
	}))
	defer srv.Close()

	client := NewClient(Config{
		Username: testUsername,
		APIKey:   testAPIKey,
		BaseURL:  srv.URL,
		Timeout:  5 * time.Second,
		Testing:  true,
	})

	_, err := client.Topup(context.Background(), &TopupRequest{BuyerSKUCode: "xld10", CustomerNo: "08123456789", RefID: "global-test-001"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestTopup_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "internal server error", http.StatusInternalServerError)
	}))
	defer srv.Close()

	req := &TopupRequest{BuyerSKUCode: "xld10", CustomerNo: "08123456789", RefID: "ref002"}
	_, err := testClient(srv).Topup(context.Background(), req)
	assertIsDFError(t, err, 500)
}

func TestInqPasca_Happy(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertBaseRequest(t, r)
		var req InqPascaRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		if req.Username != testUsername {
			t.Errorf("username: got %q want %q", req.Username, testUsername)
		}
		if req.Commands != "inq-pasca" {
			t.Errorf("commands: got %q want inq-pasca", req.Commands)
		}
		if req.Sign == "" {
			t.Error("expected non-empty sign")
		}
		writeJSON(w, map[string]any{"data": TransactionResponse{
			RefID:        req.RefID,
			CustomerNo:   req.CustomerNo,
			BuyerSKUCode: req.BuyerSKUCode,
			Status:       StatusSukses,
			Rc:           "00",
			Price:        150000,
		}})
	}))
	defer srv.Close()

	req := &InqPascaRequest{BuyerSKUCode: "pln", CustomerNo: "12345678901", RefID: "inq001"}
	resp, err := testClient(srv).InqPasca(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Status != StatusSukses {
		t.Errorf("status: got %q want %q", resp.Status, StatusSukses)
	}
	if resp.Price != 150000 {
		t.Errorf("price: got %v want 150000", resp.Price)
	}
}

func TestInqPasca_GlobalTestingMode(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertBaseRequest(t, r)
		var req InqPascaRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		if !req.Testing {
			t.Errorf("expected Testing=true when global testing mode is enabled, got false")
		}
		writeJSON(w, map[string]any{"data": TransactionResponse{RefID: req.RefID, Status: StatusSukses, Rc: "00"}})
	}))
	defer srv.Close()

	client := NewClient(Config{
		Username: testUsername,
		APIKey:   testAPIKey,
		BaseURL:  srv.URL,
		Timeout:  5 * time.Second,
		Testing:  true,
	})

	_, err := client.InqPasca(context.Background(), &InqPascaRequest{BuyerSKUCode: "pln", CustomerNo: "12345678901", RefID: "global-test-002"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestInqPasca_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "internal server error", http.StatusInternalServerError)
	}))
	defer srv.Close()

	req := &InqPascaRequest{BuyerSKUCode: "pln", CustomerNo: "12345678901", RefID: "inq002"}
	_, err := testClient(srv).InqPasca(context.Background(), req)
	assertIsDFError(t, err, 500)
}

func TestPayPasca_Happy(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertBaseRequest(t, r)
		var req PayPascaRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		if req.Username != testUsername {
			t.Errorf("username: got %q want %q", req.Username, testUsername)
		}
		if req.Commands != "pay-pasca" {
			t.Errorf("commands: got %q want pay-pasca", req.Commands)
		}
		if req.Sign == "" {
			t.Error("expected non-empty sign")
		}
		writeJSON(w, map[string]any{"data": TransactionResponse{
			RefID:      req.RefID,
			CustomerNo: req.CustomerNo,
			Status:     StatusSukses,
			Rc:         "00",
			Price:      150000,
		}})
	}))
	defer srv.Close()

	req := &PayPascaRequest{BuyerSKUCode: "pln", CustomerNo: "12345678901", RefID: "pay001"}
	resp, err := testClient(srv).PayPasca(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Status != StatusSukses {
		t.Errorf("status: got %q want %q", resp.Status, StatusSukses)
	}
	if resp.RefID != "pay001" {
		t.Errorf("ref_id: got %q want pay001", resp.RefID)
	}
}

func TestPayPasca_GlobalTestingMode(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertBaseRequest(t, r)
		var req PayPascaRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		if !req.Testing {
			t.Errorf("expected Testing=true when global testing mode is enabled, got false")
		}
		writeJSON(w, map[string]any{"data": TransactionResponse{RefID: req.RefID, CustomerNo: req.CustomerNo, Status: StatusSukses, Rc: "00"}})
	}))
	defer srv.Close()

	client := NewClient(Config{
		Username: testUsername,
		APIKey:   testAPIKey,
		BaseURL:  srv.URL,
		Timeout:  5 * time.Second,
		Testing:  true,
	})

	_, err := client.PayPasca(context.Background(), &PayPascaRequest{BuyerSKUCode: "pln", CustomerNo: "12345678901", RefID: "global-test-003"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestPayPasca_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "internal server error", http.StatusInternalServerError)
	}))
	defer srv.Close()

	req := &PayPascaRequest{BuyerSKUCode: "pln", CustomerNo: "12345678901", RefID: "pay002"}
	_, err := testClient(srv).PayPasca(context.Background(), req)
	assertIsDFError(t, err, 500)
}

func TestStatusPasca_Happy(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertBaseRequest(t, r)
		var req StatusPascaRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		if req.Username != testUsername {
			t.Errorf("username: got %q want %q", req.Username, testUsername)
		}
		if req.Commands != "status-pasca" {
			t.Errorf("commands: got %q want status-pasca", req.Commands)
		}
		if req.Sign == "" {
			t.Error("expected non-empty sign")
		}
		writeJSON(w, map[string]any{"data": TransactionResponse{
			RefID:      req.RefID,
			CustomerNo: req.CustomerNo,
			Status:     StatusSukses,
			Rc:         "00",
		}})
	}))
	defer srv.Close()

	req := &StatusPascaRequest{BuyerSKUCode: "pln", CustomerNo: "12345678901", RefID: "stat001"}
	resp, err := testClient(srv).StatusPasca(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.RefID != "stat001" {
		t.Errorf("ref_id: got %q want stat001", resp.RefID)
	}
	if resp.Status != StatusSukses {
		t.Errorf("status: got %q want %q", resp.Status, StatusSukses)
	}
}

func TestStatusPasca_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "internal server error", http.StatusInternalServerError)
	}))
	defer srv.Close()

	req := &StatusPascaRequest{BuyerSKUCode: "pln", CustomerNo: "12345678901", RefID: "stat002"}
	_, err := testClient(srv).StatusPasca(context.Background(), req)
	assertIsDFError(t, err, 500)
}

func TestInquiryPLN_Happy(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertBaseRequest(t, r)
		var req InquiryPLNRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		if req.Username != testUsername {
			t.Errorf("username: got %q want %q", req.Username, testUsername)
		}
		if req.Sign == "" {
			t.Error("expected non-empty sign")
		}
		writeJSON(w, map[string]any{"data": InquiryPLNResponse{
			CustomerNo:   req.CustomerNo,
			MeterNo:      "56789",
			Name:         "Test Customer",
			SegmentPower: "900VA",
			Status:       "sukses",
			Rc:           "00",
		}})
	}))
	defer srv.Close()

	req := &InquiryPLNRequest{CustomerNo: "12345678901"}
	resp, err := testClient(srv).InquiryPLN(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Name != "Test Customer" {
		t.Errorf("name: got %q want Test Customer", resp.Name)
	}
	if resp.SegmentPower != "900VA" {
		t.Errorf("segment_power: got %q want 900VA", resp.SegmentPower)
	}
	if resp.MeterNo != "56789" {
		t.Errorf("meter_no: got %q want 56789", resp.MeterNo)
	}
}

func TestInquiryPLN_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "internal server error", http.StatusInternalServerError)
	}))
	defer srv.Close()

	req := &InquiryPLNRequest{CustomerNo: "12345678901"}
	_, err := testClient(srv).InquiryPLN(context.Background(), req)
	assertIsDFError(t, err, 500)
}

func TestClient_NonTwoXXResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "internal server error", http.StatusInternalServerError)
	}))
	defer srv.Close()

	_, err := testClient(srv).CekSaldo(context.Background())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var dfErr *DigiflazzError
	if !errors.As(err, &dfErr) {
		t.Fatalf("expected *DigiflazzError, got %T: %v", err, err)
	}
	if dfErr.StatusCode != http.StatusInternalServerError {
		t.Errorf("StatusCode: got %d want 500", dfErr.StatusCode)
	}
}

func TestClient_MalformedJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data": "not-an-object"}`))
	}))
	defer srv.Close()

	_, err := testClient(srv).CekSaldo(context.Background())
	if err == nil {
		t.Fatal("expected error for malformed JSON data, got nil")
	}
}

func TestClient_ContextCancellation(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(10 * time.Millisecond)
		writeJSON(w, map[string]any{"data": map[string]any{"deposit": 0}})
	}))
	defer srv.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := testClient(srv).CekSaldo(ctx)
	if err == nil {
		t.Fatal("expected error for canceled context, got nil")
	}
}

func TestClient_SignatureInRequest(t *testing.T) {
	const refID = "sig-test-ref"
	expectedSign := signTransaction(testUsername, testAPIKey, refID)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req TopupRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		if req.Sign != expectedSign {
			t.Errorf("sign: got %q want %q", req.Sign, expectedSign)
		}
		writeJSON(w, map[string]any{"data": TransactionResponse{
			RefID:  refID,
			Status: StatusSukses,
			Rc:     "00",
		}})
	}))
	defer srv.Close()

	req := &TopupRequest{BuyerSKUCode: "xld10", CustomerNo: "08123456789", RefID: refID}
	_, err := testClient(srv).Topup(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
