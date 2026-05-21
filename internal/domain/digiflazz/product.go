package digiflazz

type ProductDTO struct {
	Code           string  `json:"code"`
	Name           string  `json:"name"`
	Category       string  `json:"category"`
	Brand          string  `json:"brand"`
	Type           string  `json:"type"`
	SellerName     string  `json:"seller_name"`
	Price          float64 `json:"price"`
	Admin          float64 `json:"admin,omitempty"`
	Commission     float64 `json:"commission,omitempty"`
	Status         string  `json:"status"`
	Stock          int     `json:"stock,omitempty"`
	UnlimitedStock bool    `json:"unlimited_stock,omitempty"`
	Multi          bool    `json:"multi,omitempty"`
	StartCutOff    string  `json:"start_cut_off,omitempty"`
	EndCutOff      string  `json:"end_cut_off,omitempty"`
	Description    string  `json:"description,omitempty"`
	LastUpdated    string  `json:"last_updated,omitempty"`
	FamilyID       string  `json:"family_id,omitempty"`
	CredentialID   string  `json:"credential_id,omitempty"`
	IsPrepaid      bool    `json:"is_prepaid"`
}

type ProductListRequest struct {
	Category string `json:"category,omitempty"`
	Brand    string `json:"brand,omitempty"`
	Type     string `json:"type,omitempty"`
	Code     string `json:"code,omitempty"`
	Page     int    `json:"page,omitempty"`
	PerPage  int    `json:"per_page,omitempty"`
}

type ProductSearchRequest struct {
	Query    string `json:"query,omitempty"`
	Category string `json:"category,omitempty"`
	Brand    string `json:"brand,omitempty"`
	Type     string `json:"type,omitempty"`
	Status   string `json:"status,omitempty"`
	Limit    int    `json:"limit,omitempty"`
}

type ProductListResponse struct {
	Items []*ProductDTO `json:"items"`
	Limit int           `json:"limit"`
}
