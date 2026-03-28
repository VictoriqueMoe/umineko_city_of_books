package config

import (
	"os"

	"github.com/joho/godotenv"
)

type (
	Config struct {
		DBPath    string
		UploadDir string
		BaseURL   string
	}
)

var (
	Cfg Config
)

func init() {
	_ = godotenv.Load()

	Cfg = Config{
		DBPath:    getEnv("DB_PATH", "truths.db"),
		UploadDir: getEnv("UPLOAD_DIR", "uploads"),
		BaseURL:   getEnv("BASE_URL", "http://localhost:4323"),
	}
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}
