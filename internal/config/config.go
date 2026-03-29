package config

import (
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type (
	Config struct {
		DBPath         string
		UploadDir      string
		BaseURL        string
		LogLevel       string
		MaxBodySize    int
		MaxImageSize   int64
		MaxVideoSize   int64
		MaxGeneralSize int64
	}
)

var (
	Cfg Config
)

func init() {
	_ = godotenv.Load()

	Cfg = Config{
		DBPath:         getEnv("DB_PATH", "truths.db"),
		UploadDir:      getEnv("UPLOAD_DIR", "uploads"),
		BaseURL:        getEnv("BASE_URL", "http://localhost:4323"),
		LogLevel:       getEnv("LOG_LEVEL", "info"),
		MaxBodySize:    getEnvInt("MAX_BODY_SIZE", 50*1024*1024),
		MaxImageSize:   getEnvInt64("MAX_IMAGE_SIZE", 10*1024*1024),
		MaxVideoSize:   getEnvInt64("MAX_VIDEO_SIZE", 100*1024*1024),
		MaxGeneralSize: getEnvInt64("MAX_GENERAL_SIZE", 50*1024*1024),
	}
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if value, ok := os.LookupEnv(key); ok {
		if parsed, err := strconv.Atoi(value); err == nil {
			return parsed
		}
	}
	return fallback
}

func getEnvInt64(key string, fallback int64) int64 {
	if value, ok := os.LookupEnv(key); ok {
		if parsed, err := strconv.ParseInt(value, 10, 64); err == nil {
			return parsed
		}
	}
	return fallback
}
