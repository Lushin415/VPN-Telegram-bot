package services

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"testing"
)

func TestCheckYooKassaSignature(t *testing.T) {
	secret := "testsecret"
	body := []byte(`{"test":"data"}`)

	h := hmac.New(sha256.New, []byte(secret))
	h.Write(body)
	calc := hex.EncodeToString(h.Sum(nil))

	tests := []struct {
		desc        string
		authHeader  string
		yoomoneyHdr string
		want        bool
	}{
		{"valid Authorization", "HMAC " + calc, "", true},
		{"valid Authorization SHA256", "HMAC-SHA256 " + calc, "", true},
		{"valid Yoomoney header", "", calc, true},
		{"wrong signature", "HMAC wrong", "", false},
		{"wrong yoomoney", "", "wrong", false},
		{"both empty", "", "", false},
	}

	for _, tt := range tests {
		if got := checkYooKassaSignature(secret, body, tt.authHeader, tt.yoomoneyHdr); got != tt.want {
			t.Errorf("%s: got %v, want %v", tt.desc, got, tt.want)
		}
	}
}
