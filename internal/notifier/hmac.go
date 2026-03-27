package notifier

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
)

// Sign returns a hex-encoded HMAC-SHA256 signature of payload using secret.
func Sign(payload []byte, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	return hex.EncodeToString(mac.Sum(nil))
}

// Verify checks that signature matches the HMAC-SHA256 of payload using secret.
// Uses constant-time comparison to prevent timing attacks.
func Verify(payload []byte, secret string, signature string) bool {
	expected := Sign(payload, secret)
	expectedBytes, err := hex.DecodeString(expected)
	if err != nil {
		return false
	}
	sigBytes, err := hex.DecodeString(signature)
	if err != nil {
		return false
	}
	return hmac.Equal(expectedBytes, sigBytes)
}
