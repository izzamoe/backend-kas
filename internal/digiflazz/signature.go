package digiflazz

import (
	"crypto/md5"
	"fmt"
)

func signDepo(username, apiKey string) string {
	return md5Hex(username + apiKey + "depo")
}

func signPricelist(username, apiKey string) string {
	return md5Hex(username + apiKey + "pricelist")
}

func signDeposit(username, apiKey string) string {
	return md5Hex(username + apiKey + "deposit")
}

func signTransaction(username, apiKey, refID string) string {
	return md5Hex(username + apiKey + refID)
}

func signInquiryPLN(username, apiKey, customerNo string) string {
	return md5Hex(username + apiKey + customerNo)
}

func md5Hex(input string) string {
	sum := md5.Sum([]byte(input))
	return fmt.Sprintf("%x", sum)
}
