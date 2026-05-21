package repository

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"kas/generated"
	digiflazzdomain "kas/internal/domain/digiflazz"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

type UpsertProductInput struct {
	FamilyID            string
	CredentialID        string
	ProductName         string
	Category            string
	Brand               string
	Type                string
	BuyerSKUCode        string
	Price               float64
	Admin               float64
	BuyerProductStatus  string
	SellerProductStatus string
	Stock               float64
	Multi               bool
	Desc                string
	Provider            string
	IsPrepaid           bool
}

type DigiflazzProductRepository interface {
	Upsert(input *UpsertProductInput) (*digiflazzdomain.ProductDTO, error)
	Search(familyID string, req *digiflazzdomain.ProductSearchRequest) ([]*digiflazzdomain.ProductDTO, error)
	GetBySKU(familyID, sku string) (*digiflazzdomain.ProductDTO, error)
	DeleteByFamilyID(familyID string) error
}

type digiflazzProductRepo struct {
	app core.App
}

func NewDigiflazzProductRepository(app core.App) DigiflazzProductRepository {
	return &digiflazzProductRepo{app: app}
}

func (r *digiflazzProductRepo) Upsert(input *UpsertProductInput) (*digiflazzdomain.ProductDTO, error) {
	if input.FamilyID == "" {
		return nil, fmt.Errorf("digiflazz_product_repo: familyID is required")
	}

	var record *core.Record

	existing, err := r.app.FindFirstRecordByFilter(
		"digiflazz_products",
		"family_id = {:fid} && buyer_sku_code = {:sku}",
		dbx.Params{"fid": input.FamilyID, "sku": input.BuyerSKUCode},
	)
	switch {
	case err == nil && existing != nil:
		record = existing
	case err == nil && existing == nil:
		col, colErr := r.app.FindCachedCollectionByNameOrId("digiflazz_products")
		if colErr != nil {
			return nil, fmt.Errorf("digiflazz_product_repo: find collection: %w", colErr)
		}
		record = core.NewRecord(col)
	case errors.Is(err, sql.ErrNoRows):
		col, colErr := r.app.FindCachedCollectionByNameOrId("digiflazz_products")
		if colErr != nil {
			return nil, fmt.Errorf("digiflazz_product_repo: find collection: %w", colErr)
		}
		record = core.NewRecord(col)
	default:
		return nil, fmt.Errorf("digiflazz_product_repo: look up %s: %w", input.BuyerSKUCode, err)
	}

	record.Set("product_name", input.ProductName)
	record.Set("category", input.Category)
	record.Set("brand", input.Brand)
	record.Set("type", input.Type)
	record.Set("buyer_sku_code", input.BuyerSKUCode)
	record.Set("price", input.Price)
	record.Set("admin", input.Admin)
	record.Set("buyer_product_status", input.BuyerProductStatus)
	record.Set("seller_product_status", input.SellerProductStatus)
	record.Set("stock", input.Stock)
	record.Set("multi", input.Multi)
	record.Set("desc", input.Desc)
	record.Set("provider", input.Provider)
	record.Set("is_prepaid", input.IsPrepaid)
	record.Set("family_id", input.FamilyID)
	record.Set("credential_id", input.CredentialID)

	if saveErr := r.app.Save(record); saveErr != nil {
		return nil, fmt.Errorf("digiflazz_product_repo: save %s: %w", input.BuyerSKUCode, saveErr)
	}

	return r.recordToDTO(record)
}

func (r *digiflazzProductRepo) Search(familyID string, req *digiflazzdomain.ProductSearchRequest) ([]*digiflazzdomain.ProductDTO, error) {
	if familyID == "" {
		return nil, fmt.Errorf("digiflazz_product_repo: familyID is required for Search")
	}

	if req == nil {
		req = &digiflazzdomain.ProductSearchRequest{}
	}

	parts := make([]string, 0, 6)
	params := make(map[string]any, 6)

	parts = append(parts, "family_id = {:fid}")
	params["fid"] = familyID

	if req.Category != "" {
		parts = append(parts, "category = {:category}")
		params["category"] = req.Category
	}
	if req.Brand != "" {
		parts = append(parts, "brand = {:brand}")
		params["brand"] = req.Brand
	}
	if req.Type != "" {
		parts = append(parts, "type = {:type}")
		params["type"] = req.Type
	}
	if req.Status != "" {
		parts = append(parts, "buyer_product_status = {:status}")
		params["status"] = req.Status
	}
	if req.Query != "" {
		parts = append(parts, "(product_name ~ {:query} || buyer_sku_code ~ {:query})")
		params["query"] = req.Query
	}

	filter := strings.Join(parts, " && ")

	limit := req.Limit
	if limit <= 0 || limit > 200 {
		limit = 50
	}

	records, fetchErr := r.app.FindRecordsByFilter(
		"digiflazz_products",
		filter,
		"product_name",
		limit,
		0,
		params,
	)
	if fetchErr != nil {
		return nil, fmt.Errorf("digiflazz_product_repo: search: %w", fetchErr)
	}

	dtos := make([]*digiflazzdomain.ProductDTO, 0, len(records))
	for _, rec := range records {
		dto, dtoErr := r.recordToDTO(rec)
		if dtoErr != nil {
			return nil, dtoErr
		}
		dtos = append(dtos, dto)
	}

	return dtos, nil
}

func (r *digiflazzProductRepo) GetBySKU(familyID, sku string) (*digiflazzdomain.ProductDTO, error) {
	if familyID == "" {
		return nil, fmt.Errorf("digiflazz_product_repo: familyID is required for GetBySKU")
	}

	record, err := r.app.FindFirstRecordByFilter(
		"digiflazz_products",
		"family_id = {:fid} && buyer_sku_code = {:sku}",
		dbx.Params{"fid": familyID, "sku": sku},
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("digiflazz_product_repo: get by SKU %s: %w", sku, err)
	}
	if record == nil {
		return nil, nil
	}
	return r.recordToDTO(record)
}

func (r *digiflazzProductRepo) DeleteByFamilyID(familyID string) error {
	if familyID == "" {
		return fmt.Errorf("digiflazz_product_repo: familyID is required for DeleteByFamilyID")
	}
	records, err := r.app.FindRecordsByFilter(
		"digiflazz_products",
		"family_id = {:fid}",
		"",
		-1,
		0,
		dbx.Params{"fid": familyID},
	)
	if err != nil {
		return fmt.Errorf("digiflazz_product_repo: find for delete: %w", err)
	}
	for _, rec := range records {
		if err := r.app.Delete(rec); err != nil {
			return fmt.Errorf("digiflazz_product_repo: delete record %s: %w", rec.Id, err)
		}
	}
	return nil
}

func (r *digiflazzProductRepo) recordToDTO(record *core.Record) (*digiflazzdomain.ProductDTO, error) {
	proxy, err := generated.WrapRecord[generated.DigiflazzProducts](record)
	if err != nil {
		return nil, fmt.Errorf("digiflazz_product_repo: wrap record: %w", err)
	}

	return &digiflazzdomain.ProductDTO{
		Code:         proxy.BuyerSkuCode(),
		Name:         proxy.ProductName(),
		Category:     proxy.Category(),
		Brand:        proxy.Brand(),
		Type:         proxy.Type(),
		SellerName:   proxy.Provider(),
		Price:        proxy.Price(),
		Admin:        proxy.Admin(),
		Status:       proxy.BuyerProductStatus(),
		Stock:        int(proxy.Stock()),
		Multi:        proxy.Multi(),
		Description:  proxy.Desc(),
		FamilyID:     record.GetString("family_id"),
		CredentialID: record.GetString("credential_id"),
		IsPrepaid:    proxy.IsPrepaid(),
	}, nil
}
