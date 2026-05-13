package digiflazz

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type DigiflazzClient interface {
	CekSaldo(ctx context.Context) (*CekSaldoResponse, error)
	DaftarHargaPrabayar(ctx context.Context, opts *PriceListRequest) ([]PriceListPrepaidItem, error)
	DaftarHargaPascabayar(ctx context.Context, opts *PriceListRequest) ([]PriceListPascaItem, error)
	Deposit(ctx context.Context, req *DepositRequest) (*DepositResponse, error)
	Topup(ctx context.Context, req *TopupRequest) (*TransactionResponse, error)
	InqPasca(ctx context.Context, req *InqPascaRequest) (*TransactionResponse, error)
	PayPasca(ctx context.Context, req *PayPascaRequest) (*TransactionResponse, error)
	StatusPasca(ctx context.Context, req *StatusPascaRequest) (*TransactionResponse, error)
	InquiryPLN(ctx context.Context, req *InquiryPLNRequest) (*InquiryPLNResponse, error)
}

type DigiflazzError struct {
	StatusCode int
	Body       string
}

func (e *DigiflazzError) Error() string {
	return fmt.Sprintf("digiflazz error: status=%d body=%s", e.StatusCode, e.Body)
}

type client struct {
	cfg  Config
	http *http.Client
}

// NewClient constructs a DigiflazzClient. Panics if Username or APIKey is empty.
func NewClient(cfg Config) DigiflazzClient {
	if cfg.Username == "" || cfg.APIKey == "" {
		panic("digiflazz: Username and APIKey must not be empty")
	}
	if cfg.BaseURL == "" {
		cfg.BaseURL = DefaultBaseURL
	}
	if cfg.Timeout == 0 {
		cfg.Timeout = DefaultTimeout
	}
	return &client{
		cfg:  cfg,
		http: &http.Client{Timeout: cfg.Timeout},
	}
}

type envelope struct {
	Data json.RawMessage `json:"data"`
}

func (c *client) post(ctx context.Context, path string, body any, out any) error {
	b, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("digiflazz: marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.cfg.BaseURL+path, bytes.NewReader(b))
	if err != nil {
		return fmt.Errorf("digiflazz: create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("digiflazz: do request: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("digiflazz: read response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return &DigiflazzError{StatusCode: resp.StatusCode, Body: string(bodyBytes)}
	}

	if out == nil {
		return nil
	}

	var env envelope
	if err := json.Unmarshal(bodyBytes, &env); err != nil {
		return fmt.Errorf("digiflazz: unmarshal envelope: %w", err)
	}
	if len(env.Data) == 0 {
		return nil
	}
	if err := json.Unmarshal(env.Data, out); err != nil {
		return fmt.Errorf("digiflazz: unmarshal data: %w", err)
	}
	return nil
}

func (c *client) CekSaldo(ctx context.Context) (*CekSaldoResponse, error) {
	req := CekSaldoRequest{
		Cmd:      "deposit",
		Username: c.cfg.Username,
		Sign:     signDepo(c.cfg.Username, c.cfg.APIKey),
	}
	var resp CekSaldoResponse
	if err := c.post(ctx, "/cek-saldo", req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (c *client) DaftarHargaPrabayar(ctx context.Context, opts *PriceListRequest) ([]PriceListPrepaidItem, error) {
	if opts == nil {
		opts = &PriceListRequest{}
	}
	opts.Cmd = "prepaid"
	opts.Username = c.cfg.Username
	opts.Sign = signPricelist(c.cfg.Username, c.cfg.APIKey)

	var items []PriceListPrepaidItem
	if err := c.post(ctx, "/price-list", opts, &items); err != nil {
		return nil, err
	}
	return items, nil
}

func (c *client) DaftarHargaPascabayar(ctx context.Context, opts *PriceListRequest) ([]PriceListPascaItem, error) {
	if opts == nil {
		opts = &PriceListRequest{}
	}
	opts.Cmd = "pasca"
	opts.Username = c.cfg.Username
	opts.Sign = signPricelist(c.cfg.Username, c.cfg.APIKey)

	var items []PriceListPascaItem
	if err := c.post(ctx, "/price-list", opts, &items); err != nil {
		return nil, err
	}
	return items, nil
}

func (c *client) Deposit(ctx context.Context, req *DepositRequest) (*DepositResponse, error) {
	req.Username = c.cfg.Username
	req.Sign = signDeposit(c.cfg.Username, c.cfg.APIKey)

	var resp DepositResponse
	if err := c.post(ctx, "/deposit", req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (c *client) Topup(ctx context.Context, req *TopupRequest) (*TransactionResponse, error) {
	req.Username = c.cfg.Username
	req.Sign = signTransaction(c.cfg.Username, c.cfg.APIKey, req.RefID)
	if c.cfg.Testing {
		req.Testing = true
	}

	var resp TransactionResponse
	if err := c.post(ctx, "/transaction", req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (c *client) InqPasca(ctx context.Context, req *InqPascaRequest) (*TransactionResponse, error) {
	req.Commands = "inq-pasca"
	req.Username = c.cfg.Username
	req.Sign = signTransaction(c.cfg.Username, c.cfg.APIKey, req.RefID)
	if c.cfg.Testing {
		req.Testing = true
	}

	var resp TransactionResponse
	if err := c.post(ctx, "/transaction", req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (c *client) PayPasca(ctx context.Context, req *PayPascaRequest) (*TransactionResponse, error) {
	req.Commands = "pay-pasca"
	req.Username = c.cfg.Username
	req.Sign = signTransaction(c.cfg.Username, c.cfg.APIKey, req.RefID)
	if c.cfg.Testing {
		req.Testing = true
	}

	var resp TransactionResponse
	if err := c.post(ctx, "/transaction", req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (c *client) StatusPasca(ctx context.Context, req *StatusPascaRequest) (*TransactionResponse, error) {
	req.Commands = "status-pasca"
	req.Username = c.cfg.Username
	req.Sign = signTransaction(c.cfg.Username, c.cfg.APIKey, req.RefID)

	var resp TransactionResponse
	if err := c.post(ctx, "/transaction", req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (c *client) InquiryPLN(ctx context.Context, req *InquiryPLNRequest) (*InquiryPLNResponse, error) {
	req.Username = c.cfg.Username
	req.Sign = signInquiryPLN(c.cfg.Username, c.cfg.APIKey, req.CustomerNo)

	var resp InquiryPLNResponse
	if err := c.post(ctx, "/inquiry-pln", req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}
