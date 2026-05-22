package utils

import (
	"encoding/json"
	"testing"
)

func TestRedactJSON(t *testing.T) {
	input := []byte(`{
		"token":"abc123",
		"profile":{
			"password":"p@ss",
			"details":[
				{"api_key":"key-1","name":"alice"},
				{"name":"bob","nested":{"secret":"hide"}}
			]
		}
	}`)

	redacted, err := RedactJSON(input, []string{"token", "password", "api_key", "secret"})
	if err != nil {
		t.Fatalf("RedactJSON() error = %v", err)
	}

	var got map[string]any
	if err := json.Unmarshal(redacted, &got); err != nil {
		t.Fatalf("redacted JSON invalid: %v", err)
	}

	if got["token"] != "***" {
		t.Fatalf("token not redacted: %#v", got["token"])
	}

	profile := got["profile"].(map[string]any)
	if profile["password"] != "***" {
		t.Fatalf("password not redacted: %#v", profile["password"])
	}

	details := profile["details"].([]any)
	first := details[0].(map[string]any)
	if first["api_key"] != "***" {
		t.Fatalf("api_key not redacted: %#v", first["api_key"])
	}
	second := details[1].(map[string]any)
	nested := second["nested"].(map[string]any)
	if nested["secret"] != "***" {
		t.Fatalf("secret not redacted: %#v", nested["secret"])
	}
}

func TestRedactString(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		keepLast int
		want     string
	}{
		{name: "mask all but last two", input: "abc123", keepLast: 2, want: "****23"},
		{name: "keep all", input: "abc", keepLast: 5, want: "abc"},
		{name: "mask all", input: "abc", keepLast: 0, want: "***"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := RedactString(tt.input, tt.keepLast); got != tt.want {
				t.Fatalf("RedactString() = %q, want %q", got, tt.want)
			}
		})
	}
}
