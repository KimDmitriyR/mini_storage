package config

import (
	"fmt"
	"os"
	"strconv"
)

const bytesInMegabyte = 1024 * 1024

type Config struct {
	Port            string
	StorageDir      string
	MaxUploadSizeMB int64
	DatabasePath    string
}

func Load() (Config, error) {
	cfg := Config{
		Port:            getEnv("PORT", "8080"),
		StorageDir:      getEnv("STORAGE_DIR", "storage"),
		MaxUploadSizeMB: int64(getEnvAsInt("MAX_UPLOAD_SIZE_MB", 10)),
		DatabasePath:    getEnv("DATABASE_PATH", "storage/metadata.db"),
	}

	if cfg.MaxUploadSizeMB <= 0 {
		return Config{}, fmt.Errorf("MAX_UPLOAD_SIZE_MB must be greater than zero")
	}

	return cfg, nil
}

func (c Config) Address() string {
	return ":" + c.Port
}

func (c Config) MaxUploadSizeBytes() int64 {
	return c.MaxUploadSizeMB * bytesInMegabyte
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok && value != "" {
		return value
	}

	return fallback
}

func getEnvAsInt(key string, fallback int) int {
	value := getEnv(key, "")
	if value == "" {
		return fallback
	}

	parsedValue, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}

	return parsedValue
}
