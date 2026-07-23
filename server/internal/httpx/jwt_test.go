package httpx

import "testing"

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
