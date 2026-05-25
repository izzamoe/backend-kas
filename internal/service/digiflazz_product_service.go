package service

import (
	"context"
	"fmt"

	digiflazzclient "kas/internal/digiflazz"
	digiflazzdomain "kas/internal/domain/digiflazz"
	"kas/internal/repository"
	"kas/internal/utils"

	"github.com/pocketbase/pocketbase/core"
)

type SyncResult struct {
	PrepaidUpserted  int      `json:"prepaid_upserted"`
	PostpaidUpserted int      `json:"postpaid_upserted"`
	TotalUpserted    int      `json:"total_upserted"`
	Errors           []string `json:"errors,omitempty"`
}

type DigiflazzProductService interface {
	SyncPricelistWithCredential(ctx context.Context, credential *repository.DigiflazzCredentialRecord) (*SyncResult, error)
	SyncForFamily(ctx context.Context, familyID string) (*SyncResult, error)
	SearchProducts(familyID string, req *digiflazzdomain.ProductSearchRequest) ([]*digiflazzdomain.ProductDTO, error)
	GetProductBySKU(familyID, sku string) (*digiflazzdomain.ProductDTO, error)
}

type digiflazzProductService struct {
	app            core.App
	productRepo    repository.DigiflazzProductRepository
	credentialRepo repository.DigiflazzCredentialRepository
	clientFactory  DigiflazzClientFactory
}

func NewDigiflazzProductService(app core.App, productRepo repository.DigiflazzProductRepository, credentialRepo repository.DigiflazzCredentialRepository, clientFactory DigiflazzClientFactory) DigiflazzProductService {
	if clientFactory == nil {
		clientFactory = func(username, apiKey string, testing bool) digiflazzclient.DigiflazzClient {
			return digiflazzclient.NewClient(digiflazzclient.Config{
				Username: username,
				APIKey:   apiKey,
				Testing:  testing,
			})
		}
	}
	return &digiflazzProductService{
		app:            app,
		productRepo:    productRepo,
		credentialRepo: credentialRepo,
		clientFactory:  clientFactory,
	}
}

func (s *digiflazzProductService) doSyncWithClient(ctx context.Context, client digiflazzclient.DigiflazzClient, familyID, credentialID string) *SyncResult {
	result := &SyncResult{}

	prepaidItems, prepaidErr := client.DaftarHargaPrabayar(ctx, nil)
	if prepaidErr != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("prepaid fetch: %v", prepaidErr))
	} else {
		for _, item := range prepaidItems {
			if _, upsertErr := s.productRepo.Upsert(prepaidToUpsertInput(item, familyID, credentialID)); upsertErr != nil {
				result.Errors = append(result.Errors, fmt.Sprintf("upsert prepaid %s: %v", item.BuyerSKUCode, upsertErr))
			} else {
				result.PrepaidUpserted++
			}
		}
	}

	postpaidItems, postpaidErr := client.DaftarHargaPascabayar(ctx, nil)
	if postpaidErr != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("postpaid fetch: %v", postpaidErr))
	} else {
		for _, item := range postpaidItems {
			if _, upsertErr := s.productRepo.Upsert(postpaidToUpsertInput(item, familyID, credentialID)); upsertErr != nil {
				result.Errors = append(result.Errors, fmt.Sprintf("upsert postpaid %s: %v", item.BuyerSKUCode, upsertErr))
			} else {
				result.PostpaidUpserted++
			}
		}
	}

	result.TotalUpserted = result.PrepaidUpserted + result.PostpaidUpserted
	return result
}

func (s *digiflazzProductService) SyncPricelistWithCredential(ctx context.Context, credential *repository.DigiflazzCredentialRecord) (*SyncResult, error) {
	if credential == nil {
		return nil, fmt.Errorf("digiflazz_product_svc: credential is required")
	}
	familyID := credential.FamilyID
	credentialID := credential.ID
	key, err := credentialEncryptionKey()
	if err != nil {
		return nil, fmt.Errorf("digiflazz_product_svc: encryption key: %w", err)
	}
	rawAPIKey, err := utils.Decrypt(credential.APIKeyCiphertext, key)
	if err != nil {
		return nil, fmt.Errorf("digiflazz_product_svc: decrypt api key: %w", err)
	}
	client := s.clientFactory(credential.Username, rawAPIKey, credential.Testing)
	return s.doSyncWithClient(ctx, client, familyID, credentialID), nil
}

func (s *digiflazzProductService) SyncForFamily(ctx context.Context, familyID string) (*SyncResult, error) {
	if familyID == "" {
		return nil, fmt.Errorf("digiflazz_product_svc: familyID is required for SyncForFamily")
	}
	credential, err := s.credentialRepo.GetActiveSecretByFamilyID(familyID)
	if err != nil {
		return nil, fmt.Errorf("digiflazz_product_svc: get active credential: %w", err)
	}
	if credential == nil {
		return nil, fmt.Errorf("digiflazz_product_svc: no active credential found for family")
	}
	return s.SyncPricelistWithCredential(ctx, credential)
}

func (s *digiflazzProductService) SearchProducts(familyID string, req *digiflazzdomain.ProductSearchRequest) ([]*digiflazzdomain.ProductDTO, error) {
	if familyID == "" {
		return nil, fmt.Errorf("digiflazz_product_svc: familyID is required for SearchProducts")
	}
	if req == nil {
		req = &digiflazzdomain.ProductSearchRequest{}
	}
	return s.productRepo.Search(familyID, req)
}

func (s *digiflazzProductService) GetProductBySKU(familyID, sku string) (*digiflazzdomain.ProductDTO, error) {
	if familyID == "" {
		return nil, fmt.Errorf("digiflazz_product_svc: familyID is required for GetProductBySKU")
	}
	if sku == "" {
		return nil, fmt.Errorf("digiflazz_product_svc: sku must not be empty")
	}
	return s.productRepo.GetBySKU(familyID, sku)
}

func prepaidToUpsertInput(item digiflazzclient.PriceListPrepaidItem, familyID, credentialID string) *repository.UpsertProductInput {
	return &repository.UpsertProductInput{
		FamilyID:            familyID,
		CredentialID:        credentialID,
		ProductName:         item.ProductName,
		Category:            item.Category,
		Brand:               item.Brand,
		Type:                item.Type,
		BuyerSKUCode:        item.BuyerSKUCode,
		Price:               item.Price,
		BuyerProductStatus:  string(item.BuyerProductStatus),
		SellerProductStatus: string(item.SellerProductStatus),
		Stock:               float64(item.Stock),
		Multi:               item.Multi,
		Desc:                item.Desc,
		Provider:            item.SellerName,
		IsPrepaid:           true,
	}
}

func postpaidToUpsertInput(item digiflazzclient.PriceListPascaItem, familyID, credentialID string) *repository.UpsertProductInput {
	return &repository.UpsertProductInput{
		FamilyID:            familyID,
		CredentialID:        credentialID,
		ProductName:         item.ProductName,
		Category:            item.Category,
		Brand:               item.Brand,
		Type:                "postpaid",
		BuyerSKUCode:        item.BuyerSKUCode,
		Price:               item.Admin,
		Admin:               item.Admin,
		BuyerProductStatus:  string(item.BuyerProductStatus),
		SellerProductStatus: string(item.SellerProductStatus),
		Desc:                item.Desc,
		Provider:            item.SellerName,
		IsPrepaid:           false,
	}
}
