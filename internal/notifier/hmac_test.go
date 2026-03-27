package notifier

import (
	"testing"
)

func TestSign(t *testing.T) {
	// known vector computed with: echo -n '{"type":"contact.resolved","phone":"+1234567890"}' | openssl dgst -sha256 -hmac "mysecret"
	// We verify by checking that Sign produces a consistent 64-char hex string and matches itself.
	payload := []byte(`{"type":"contact.resolved","phone":"+1234567890"}`)
	secret := "mysecret"
	sig1 := Sign(payload, secret)
	sig2 := Sign(payload, secret)

	if len(sig1) != 64 {
		t.Errorf("Sign() produced signature of length %d, want 64", len(sig1))
	}
	if sig1 != sig2 {
		t.Error("Sign() is not deterministic")
	}

	// Different secrets must produce different signatures.
	sigOther := Sign(payload, "othersecret")
	if sig1 == sigOther {
		t.Error("Sign() with different secrets must produce different signatures")
	}
}

func TestVerify(t *testing.T) {
	payload := []byte(`{"type":"contact.created","phone":"+5511999999999"}`)
	secret := "topsecret"

	sig := Sign(payload, secret)

	t.Run("valid signature", func(t *testing.T) {
		if !Verify(payload, secret, sig) {
			t.Error("Verify() should return true for a valid signature")
		}
	})

	t.Run("wrong secret", func(t *testing.T) {
		if Verify(payload, "wrongsecret", sig) {
			t.Error("Verify() should return false for wrong secret")
		}
	})

	t.Run("tampered payload", func(t *testing.T) {
		if Verify([]byte(`{"type":"contact.created","phone":"+9999999999"}`), secret, sig) {
			t.Error("Verify() should return false for tampered payload")
		}
	})

	t.Run("invalid signature hex", func(t *testing.T) {
		if Verify(payload, secret, "not-valid-hex!!") {
			t.Error("Verify() should return false for invalid hex signature")
		}
	})

	t.Run("empty signature", func(t *testing.T) {
		if Verify(payload, secret, "") {
			t.Error("Verify() should return false for empty signature")
		}
	})
}

func TestSignVerifyRoundtrip(t *testing.T) {
	cases := []struct {
		payload []byte
		secret  string
	}{
		{[]byte("hello world"), "secret1"},
		{[]byte(`{"key":"value"}`), "another-secret"},
		{[]byte(""), "empty-payload"},
		{[]byte("data"), ""},
	}

	for _, c := range cases {
		sig := Sign(c.payload, c.secret)
		if !Verify(c.payload, c.secret, sig) {
			t.Errorf("roundtrip failed for payload=%q secret=%q", c.payload, c.secret)
		}
	}
}
