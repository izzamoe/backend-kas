package digiflazz

import "testing"

func TestMapDigiflazzCategory(t *testing.T) {
	tests := []struct {
		name             string
		input            string
		wantCategoryName string
		wantIsIncome     bool
	}{
		{name: "pulsa", input: "Pulsa", wantCategoryName: "Hiburan", wantIsIncome: false},
		{name: "data", input: "Data", wantCategoryName: "Hiburan", wantIsIncome: false},
		{name: "voucher", input: "Voucher", wantCategoryName: "Hiburan", wantIsIncome: false},
		{name: "games", input: "Games", wantCategoryName: "Hiburan", wantIsIncome: false},
		{name: "streaming", input: "Streaming", wantCategoryName: "Hiburan", wantIsIncome: false},
		{name: "pln", input: "PLN", wantCategoryName: "Rumah & utilitas", wantIsIncome: false},
		{name: "pln token", input: "PLN Token", wantCategoryName: "Rumah & utilitas", wantIsIncome: false},
		{name: "internet", input: "Internet", wantCategoryName: "Rumah & utilitas", wantIsIncome: false},
		{name: "e-money", input: "E-Money", wantCategoryName: "Lainnya", wantIsIncome: false},
		{name: "e-wallet", input: "E-Wallet", wantCategoryName: "Lainnya", wantIsIncome: false},
		{name: "pascabayar", input: "Pascabayar", wantCategoryName: "Rumah & utilitas", wantIsIncome: false},
		{name: "fallback", input: "Unknown Category", wantCategoryName: "Lainnya", wantIsIncome: false},
		{name: "case insensitive and trim", input: "  PuLsA  ", wantCategoryName: "Hiburan", wantIsIncome: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotCategoryName, gotIsIncome := MapDigiflazzCategory(tt.input)
			if gotCategoryName != tt.wantCategoryName {
				t.Fatalf("expected category %q, got %q", tt.wantCategoryName, gotCategoryName)
			}
			if gotIsIncome != tt.wantIsIncome {
				t.Fatalf("expected isIncome %v, got %v", tt.wantIsIncome, gotIsIncome)
			}
		})
	}
}
