package config

import (
	"os"
	"strconv"
	"strings"
)

type Config struct {
	Port           string
	DBURL          string
	JWTSecret      string
	JWTExpireHours int
	UploadDir      string
	URLPrefix      string
	AllowedOrigins []string
	MaxFileSize    int64
}

func Load() Config {
	return Config{
		Port:           env("PORT", "8080"),
		DBURL:          os.Getenv("DB_URL"),
		JWTSecret:      env("JWT_SECRET", "local-development-only-secret-key-change-before-deploy-123456"),
		JWTExpireHours: 168,
		UploadDir:      env("UPLOAD_DIR", "./uploads"),
		URLPrefix:      "/uploads",
		AllowedOrigins: strings.Split(env("CORS_ALLOWED_ORIGINS", "http://localhost:5173,http://127.0.0.1:5173"), ","),
		MaxFileSize:    int64(envInt("MAX_FILE_SIZE", 5242880)),
	}
}

func env(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

func envInt(k string, def int) int {
	if v := os.Getenv(k); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return def
}
