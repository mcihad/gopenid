package auth

import (
	"testing"
	"time"
)

func TestVerifyTOTP(t *testing.T) {
	// RFC 6238 test secret: "12345678901234567890" encoded as base32.
	secret := "GEZDGNBVGY3TQOJQGEZDGNBVGY3TQOJQ"
	if !VerifyTOTP(secret, "94287082"[2:], time.Unix(59, 0)) {
		t.Fatal("expected TOTP code to verify")
	}
	if VerifyTOTP(secret, "000000", time.Unix(59, 0)) {
		t.Fatal("unexpected TOTP verification success")
	}
}
