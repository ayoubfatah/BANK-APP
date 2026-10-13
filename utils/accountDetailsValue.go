package utils

import (
	"fmt"
	"strings"
)

const (
	DetailTypeIBAN          = "IBAN"
	DetailTypeAccountNumber = "ACCOUNT_NUMBER"
	DetailTypeRoutingNumber = "ROUTING_NUMBER"
)

type bbanPart struct {
	kind   byte // 'n' = digits, 'a' = uppercase letters
	length int
}

// BBAN structure per country (total IBAN length in the comment)
var bbanSpecs = map[string][]bbanPart{
	"DE": {{'n', 18}},           // 22
	"GB": {{'a', 4}, {'n', 14}}, // 22
	"FR": {{'n', 23}},           // 27
	"ES": {{'n', 20}},           // 24
	"IT": {{'a', 1}, {'n', 22}}, // 27
	"NL": {{'a', 4}, {'n', 10}}, // 18
	"BE": {{'n', 12}},           // 16
	"PT": {{'n', 21}},           // 25
	"IE": {{'a', 4}, {'n', 14}}, // 22
	"AT": {{'n', 16}},           // 20
	"CH": {{'n', 17}},           // 21
	"LU": {{'n', 16}},           // 20
	"SE": {{'n', 20}},           // 24
	"DK": {{'n', 14}},           // 18
	"NO": {{'n', 11}},           // 15
	"FI": {{'n', 14}},           // 18
	"PL": {{'n', 24}},           // 28
	"GR": {{'n', 23}},           // 27
	"MA": {{'n', 24}},           // 28
	"TN": {{'n', 20}},           // 24
	"EG": {{'n', 25}},           // 29
	"AE": {{'n', 19}},           // 23
	"SA": {{'n', 20}},           // 24
	"TR": {{'n', 22}},           // 26
}

func mod97(input string) int {
	remainder := 0

	for _, c := range input {
		if c >= '0' && c <= '9' {
			remainder = (remainder*10 + int(c-'0')) % 97
		} else if c >= 'A' && c <= 'Z' {
			// IBAN letters: A = 10, B = 11, ..., Z = 35.
			value := int(c-'A') + 10
			remainder = (remainder*10 + value/10) % 97
			remainder = (remainder*10 + value%10) % 97
		}
	}

	return remainder
}

func generateBBAN(countryCode string) (string, error) {
	spec, ok := bbanSpecs[countryCode]
	if !ok {
		return "", fmt.Errorf("unsupported country: %s", countryCode)
	}

	var sb strings.Builder
	for _, p := range spec {
		switch p.kind {
		case 'n':
			sb.WriteString(RandomNumbers(p.length))
		case 'a':
			sb.WriteString(strings.ToUpper(RandomString(p.length)))
		}
	}
	return sb.String(), nil
}

func generateTestIBAN(countryCode string) (string, error) {
	countryCode = strings.ToUpper(strings.TrimSpace(countryCode))

	bban, err := generateBBAN(countryCode)
	if err != nil {
		return "", err
	}

	rearranged := bban + countryCode + "00"
	checkDigits := fmt.Sprintf("%02d", 98-mod97(rearranged))

	return countryCode + checkDigits + bban, nil
}

func GenerateAccountDetail(detailType string, countryCode string) (string, error) {
	switch strings.ToUpper(strings.TrimSpace(detailType)) {
	case DetailTypeAccountNumber:
		return RandomNumbers(16), nil

	case DetailTypeIBAN:
		return generateTestIBAN(countryCode)

	case DetailTypeRoutingNumber:
		return "", fmt.Errorf("routing number must come from the banking partner")

	default:
		return "", fmt.Errorf("unsupported account detail type: %s", detailType)
	}
}
