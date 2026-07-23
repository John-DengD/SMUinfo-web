package config

import (
	"testing"
)

func TestLoadDefaults(t *testing.T) {
	t.Setenv("DB_URL", "postgres://x")
	t.Setenv("JWT_SECRET", "s")
	c := Load()
	if c.Port != "8080" {
		t.Fatalf("want 8080 got %s", c.Port)
	}
	if c.UploadDir != "./uploads" {
		t.Fatalf("upload dir default")
	}
	if c.MaxFileSize != 5242880 {
		t.Fatalf("max file size default")
	}
}
