package auth

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/John-DengD/smu-deal/server/internal/httpx"
)

func TestValidateStudentNo(t *testing.T) {
	if _, err := validateStudentNo("12345"); err == nil {
		t.Fatal("short should fail")
	}
	if _, err := validateStudentNo("123456789012"); err == nil {
		t.Fatal("fake blocklist should fail")
	}
	if v, err := validateStudentNo(" 202312345678 "); err != nil || v != "202312345678" {
		t.Fatalf("trim/ok got %q %v", v, err)
	}
}

func TestValidateName(t *testing.T) {
	if _, err := validateName("张"); err == nil {
		t.Fatal("too short")
	}
	if _, err := validateName("张3"); err == nil {
		t.Fatal("digit not allowed")
	}
	if v, err := validateName("  张 三  "); err != nil || v != "张 三" {
		t.Fatalf("got %q %v", v, err)
	}
}

// A 30-CJK-char password is 30 runes (within the 6-64 rune range) but 90 bytes,
// exceeding bcrypt's 72-byte limit. It must return a clean business error rather
// than reaching bcrypt.GenerateFromPassword (which would 500). The guard runs
// before any Querier call, so a nil Querier is safe here.
func TestRegisterLongMultibytePassword(t *testing.T) {
	svc := NewService(nil, httpx.NewJWT("test-secret-32-bytes-xxxxxxxxxxxxxx", 168))
	pw := strings.Repeat("密", 30)
	if len(pw) <= 72 {
		t.Fatalf("test precondition: expected >72 bytes, got %d", len(pw))
	}
	_, err := svc.Register(context.Background(), RegisterReq{
		Name:      "张三",
		StudentNo: "202312345678",
		Password:  pw,
	})
	var be httpx.BizError
	if !errors.As(err, &be) {
		t.Fatalf("expected BizError, got %v", err)
	}
	if be.Msg != "密码长度需为 6-64 位" {
		t.Fatalf("unexpected message: %q", be.Msg)
	}
}
