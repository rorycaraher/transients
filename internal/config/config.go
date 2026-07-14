package config

import (
	"fmt"
	"os"
	"time"
)

type Config struct {
	Port    string
	BaseURL string
	DBPath  string

	AdminPasswordHash string
	SessionSecret     string

	R2AccountID       string
	R2AccessKeyID     string
	R2SecretAccessKey string
	R2Bucket          string

	CFAPIToken string
	CFQueueID  string

	PollInterval    time.Duration
	PresignedGetTTL time.Duration
}

func Load() (*Config, error) {
	c := &Config{
		Port:              getenv("PORT", "8080"),
		BaseURL:           os.Getenv("BASE_URL"),
		DBPath:            getenv("DB_PATH", "/var/lib/transients/app.db"),
		AdminPasswordHash: os.Getenv("ADMIN_PASSWORD_HASH"),
		SessionSecret:     os.Getenv("SESSION_SECRET"),
		R2AccountID:       os.Getenv("R2_ACCOUNT_ID"),
		R2AccessKeyID:     os.Getenv("R2_ACCESS_KEY_ID"),
		R2SecretAccessKey: os.Getenv("R2_SECRET_ACCESS_KEY"),
		R2Bucket:          os.Getenv("R2_BUCKET"),
		CFAPIToken:        os.Getenv("CF_API_TOKEN"),
		CFQueueID:         os.Getenv("CF_QUEUE_ID"),
		PollInterval:      10 * time.Second,
		PresignedGetTTL:   4 * time.Hour,
	}

	required := map[string]string{
		"BASE_URL":             c.BaseURL,
		"ADMIN_PASSWORD_HASH":  c.AdminPasswordHash,
		"SESSION_SECRET":       c.SessionSecret,
		"R2_ACCOUNT_ID":        c.R2AccountID,
		"R2_ACCESS_KEY_ID":     c.R2AccessKeyID,
		"R2_SECRET_ACCESS_KEY": c.R2SecretAccessKey,
		"R2_BUCKET":            c.R2Bucket,
		"CF_API_TOKEN":         c.CFAPIToken,
		"CF_QUEUE_ID":          c.CFQueueID,
	}
	for name, val := range required {
		if val == "" {
			return nil, fmt.Errorf("missing required env var %s", name)
		}
	}

	return c, nil
}

func getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
