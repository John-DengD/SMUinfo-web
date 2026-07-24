package httpx

import (
	"strconv"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func TestJWTRoundTrip(t *testing.T) {
	m := NewJWT("test-secret-32-bytes-xxxxxxxxxxxxxx", 168)
	tok, err := m.Generate(42, "ADMIN")
	if err != nil {
		t.Fatal(err)
	}
	uid, role, err := m.Parse(tok)
	if err != nil || uid != 42 || role != "ADMIN" {
		t.Fatalf("uid=%d role=%s err=%v", uid, role, err)
	}
}

func TestJWTWrongSecret(t *testing.T) {
	signer := NewJWT("secret-a-32-bytes-xxxxxxxxxxxxxxxxx", 168)
	tok, err := signer.Generate(42, "ADMIN")
	if err != nil {
		t.Fatal(err)
	}
	verifier := NewJWT("secret-b-32-bytes-xxxxxxxxxxxxxxxxx", 168)
	if _, _, err := verifier.Parse(tok); err == nil {
		t.Fatal("expected parse to fail for token signed with a different secret")
	}
}

func TestJWTExpired(t *testing.T) {
	secret := []byte("test-secret-32-bytes-xxxxxxxxxxxxxx")
	now := time.Now()
	tok, err := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub":  strconv.FormatInt(42, 10),
		"role": "ADMIN",
		"iat":  now.Add(-2 * time.Hour).Unix(),
		"exp":  now.Add(-1 * time.Hour).Unix(),
	}).SignedString(secret)
	if err != nil {
		t.Fatal(err)
	}
	m := NewJWT("test-secret-32-bytes-xxxxxxxxxxxxxx", 168)
	if _, _, err := m.Parse(tok); err == nil {
		t.Fatal("expected parse to fail for expired token")
	}
}
