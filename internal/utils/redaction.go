package utils

import (
	"encoding/json"
	"strings"
)

var webhookSensitiveFields = []string{
	"token",
	"password",
	"api_key",
	"secret",
	"signature",
	"pin",
	"otp",
	"authorization",
	"customer_no",
	"customer_name",
	"sn",
	"tele",
	"wa",
}

// RedactJSON replaces values of sensitive fields at any nesting level with "***".
func RedactJSON(input []byte, sensitiveFields []string) ([]byte, error) {
	var data any
	if err := json.Unmarshal(input, &data); err != nil {
		return nil, err
	}

	sensitive := make(map[string]struct{}, len(sensitiveFields))
	for _, field := range sensitiveFields {
		sensitive[field] = struct{}{}
	}

	redactValue(data, sensitive)
	return json.Marshal(data)
}

// RedactString masks all but the last keepLast characters.
func RedactString(s string, keepLast int) string {
	runes := []rune(s)
	if keepLast <= 0 {
		return strings.Repeat("*", len(runes))
	}
	if keepLast >= len(runes) {
		return s
	}
	return strings.Repeat("*", len(runes)-keepLast) + string(runes[len(runes)-keepLast:])
}

// RedactWebhookPayload marshals and redacts a webhook payload before it is stored.
func RedactWebhookPayload(payload any) (string, error) {
	data, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	redacted, err := RedactJSON(data, webhookSensitiveFields)
	if err != nil {
		return "", err
	}
	return string(redacted), nil
}

func redactValue(value any, sensitive map[string]struct{}) {
	switch v := value.(type) {
	case map[string]any:
		for key, child := range v {
			if _, ok := sensitive[key]; ok {
				v[key] = "***"
				continue
			}
			redactValue(child, sensitive)
		}
	case []any:
		for _, item := range v {
			redactValue(item, sensitive)
		}
	}
}
