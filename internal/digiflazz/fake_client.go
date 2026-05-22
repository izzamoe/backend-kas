package digiflazz

import (
	"context"
	"sync"
	"time"
)

type CallRecord struct {
	Method   string
	Request  any
	Response any
	Error    error
	At       time.Time
}

type fakeResponse struct {
	resp any
	err  error
}

type FakeClient struct {
	mu        sync.Mutex
	responses map[string]fakeResponse
	history   []CallRecord
	counts    map[string]int
}

var _ DigiflazzClient = (*FakeClient)(nil)

func NewFakeClient() *FakeClient {
	return &FakeClient{
		responses: make(map[string]fakeResponse),
		counts:    make(map[string]int),
	}
}

func (f *FakeClient) Reset() {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.responses = make(map[string]fakeResponse)
	f.history = nil
	f.counts = make(map[string]int)
}

func (f *FakeClient) SetResponse(method string, resp any, err error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.responses == nil {
		f.responses = make(map[string]fakeResponse)
	}
	f.responses[method] = fakeResponse{resp: resp, err: err}
}

func (f *FakeClient) CallCount(method string) int {
	f.mu.Lock()
	defer f.mu.Unlock()

	return f.counts[method]
}

func (f *FakeClient) History() []CallRecord {
	f.mu.Lock()
	defer f.mu.Unlock()

	out := make([]CallRecord, len(f.history))
	copy(out, f.history)
	return out
}

func (f *FakeClient) record(method string, req any) (any, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.responses == nil {
		f.responses = make(map[string]fakeResponse)
	}
	if f.counts == nil {
		f.counts = make(map[string]int)
	}

	f.counts[method]++
	configured := f.responses[method]
	call := CallRecord{
		Method:   method,
		Request:  req,
		Response: configured.resp,
		Error:    configured.err,
		At:       time.Now(),
	}
	f.history = append(f.history, call)

	return configured.resp, configured.err
}

func castResponse[T any](resp any) (*T, error) {
	if resp == nil {
		return nil, nil
	}
	if typed, ok := resp.(*T); ok {
		return typed, nil
	}
	if typed, ok := resp.(T); ok {
		return &typed, nil
	}
	return nil, nil
}

func castSliceResponse[T any](resp any) ([]T, error) {
	if resp == nil {
		return nil, nil
	}
	if typed, ok := resp.([]T); ok {
		return typed, nil
	}
	return nil, nil
}

func (f *FakeClient) CekSaldo(ctx context.Context) (*CekSaldoResponse, error) {
	resp, err := f.record("CekSaldo", nil)
	if err != nil {
		return nil, err
	}
	return castResponse[CekSaldoResponse](resp)
}

func (f *FakeClient) DaftarHargaPrabayar(ctx context.Context, opts *PriceListRequest) ([]PriceListPrepaidItem, error) {
	resp, err := f.record("DaftarHargaPrabayar", opts)
	if err != nil {
		return nil, err
	}
	return castSliceResponse[PriceListPrepaidItem](resp)
}

func (f *FakeClient) DaftarHargaPascabayar(ctx context.Context, opts *PriceListRequest) ([]PriceListPascaItem, error) {
	resp, err := f.record("DaftarHargaPascabayar", opts)
	if err != nil {
		return nil, err
	}
	return castSliceResponse[PriceListPascaItem](resp)
}

func (f *FakeClient) Deposit(ctx context.Context, req *DepositRequest) (*DepositResponse, error) {
	resp, err := f.record("Deposit", req)
	if err != nil {
		return nil, err
	}
	return castResponse[DepositResponse](resp)
}

func (f *FakeClient) Topup(ctx context.Context, req *TopupRequest) (*TransactionResponse, error) {
	resp, err := f.record("Topup", req)
	if err != nil {
		return nil, err
	}
	return castResponse[TransactionResponse](resp)
}

func (f *FakeClient) InqPasca(ctx context.Context, req *InqPascaRequest) (*TransactionResponse, error) {
	resp, err := f.record("InqPasca", req)
	if err != nil {
		return nil, err
	}
	return castResponse[TransactionResponse](resp)
}

func (f *FakeClient) PayPasca(ctx context.Context, req *PayPascaRequest) (*TransactionResponse, error) {
	resp, err := f.record("PayPasca", req)
	if err != nil {
		return nil, err
	}
	return castResponse[TransactionResponse](resp)
}

func (f *FakeClient) StatusPasca(ctx context.Context, req *StatusPascaRequest) (*TransactionResponse, error) {
	resp, err := f.record("StatusPasca", req)
	if err != nil {
		return nil, err
	}
	return castResponse[TransactionResponse](resp)
}

func (f *FakeClient) InquiryPLN(ctx context.Context, req *InquiryPLNRequest) (*InquiryPLNResponse, error) {
	resp, err := f.record("InquiryPLN", req)
	if err != nil {
		return nil, err
	}
	return castResponse[InquiryPLNResponse](resp)
}
