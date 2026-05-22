package digiflazz

import "strings"

// MapDigiflazzCategory maps Digiflazz product categories to existing default family category names.
// All Digiflazz mappings are treated as expense categories.
func MapDigiflazzCategory(digiflazzCategory string) (categoryName string, isIncome bool) {
	normalized := strings.ToLower(strings.TrimSpace(digiflazzCategory))

	switch normalized {
	case "pulsa", "data", "voucher", "games", "streaming":
		return "Hiburan", false
	case "pln", "pln token", "internet", "pascabayar":
		return "Rumah & utilitas", false
	case "e-money", "e-wallet":
		return "Lainnya", false
	default:
		return "Lainnya", false
	}
}
