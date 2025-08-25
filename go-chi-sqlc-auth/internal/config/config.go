package config

import (
	"fmt"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	Port int
	DB   DBConfig
	JWT  JWTConfig
}

type DBConfig struct {
	Host        string
	Port        int
	User        string
	Password    string
	Name        string
	SSLMode     string // disable|require|verify-ca|verify-full
	SSLRootCert string
	SSLCert     string
	SSLKey      string
}

type JWTConfig struct {
	Secret         string
	ExpiresInHours int
}

func Load() (*Config, error) {
	_ = godotenv.Load()

	cfg := &Config{}

	cfg.Port = getInt("PORT", 8080)

	cfg.DB = DBConfig{
		Host:        getStr("DB_HOST", "localhost"),
		Port:        getInt("DB_PORT", 5432),
		User:        getStr("DB_USER", "postgres"),
		Password:    getStr("DB_PASSWORD", "postgres"),
		Name:        getStr("DB_NAME", "go_chi_auth"),
		SSLMode:     getStr("DB_SSLMODE", "disable"),
		SSLRootCert: getStr("DB_SSLROOTCERT", ""),
		SSLCert:     getStr("DB_SSLCERT", ""),
		SSLKey:      getStr("DB_SSLKEY", ""),
	}

	cfg.JWT = JWTConfig{
		Secret:         getStr("JWT_SECRET", "change_me_super_secret"),
		ExpiresInHours: getInt("JWT_EXPIRES_IN_HOURS", 24),
	}

	return cfg, nil
}

func (c DBConfig) DSN() string {
	// Build a pgx connection string
	base := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		c.Host, c.Port, c.User, c.Password, c.Name, c.SSLMode,
	)
	if c.SSLMode == "verify-ca" || c.SSLMode == "verify-full" {
		if c.SSLRootCert != "" {
			base += fmt.Sprintf(" sslrootcert=%s", c.SSLRootCert)
		}
		if c.SSLCert != "" {
			base += fmt.Sprintf(" sslcert=%s", c.SSLCert)
		}
		if c.SSLKey != "" {
			base += fmt.Sprintf(" sslkey=%s", c.SSLKey)
		}
	}
	return base
}

func getStr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func getInt(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			return i
		}
	}
	return def
}
