package auth

import "testing"

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
