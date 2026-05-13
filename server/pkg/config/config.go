package config

import (
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

type ServerConfig struct {
	Port int    `yaml:"port"`
	Host string `yaml:"host"`
}

type AppConfig struct {
	Server   ServerConfig `yaml:"server"`
	Database struct {
		DSN string `yaml:"dsn"`
	} `yaml:"database"`
	Redis struct {
		Addr     string `yaml:"addr"`
		Password string `yaml:"password"`
		DB       int    `yaml:"db"`
	} `yaml:"redis"`
	MinIO struct {
		Endpoint  string `yaml:"endpoint"`
		AccessKey string `yaml:"access_key"`
		SecretKey string `yaml:"secret_key"`
		UseSSL    bool   `yaml:"use_ssl"`
		Bucket    string `yaml:"bucket"`
	} `yaml:"minio"`
	Log struct {
		Level  string `yaml:"level"`
		Format string `yaml:"format"`
	} `yaml:"log"`
	JWT struct {
		AccessSecret  string `yaml:"access_secret"`
		RefreshSecret string `yaml:"refresh_secret"`
		AccessTTL     int    `yaml:"access_ttl_sec"`
		RefreshTTL    int    `yaml:"refresh_ttl_sec"`
	} `yaml:"jwt"`
	SMTP struct {
		Host     string `yaml:"host"`
		Port     int    `yaml:"port"`
		Username string `yaml:"username"`
		Password string `yaml:"password"`
		From     string `yaml:"from"`
	} `yaml:"smtp"`
}

// Host extracts the hostname from a PostgreSQL DSN.
func (c *AppConfig) DBHost() string {
	dsn := c.Database.DSN
	for _, part := range strings.Fields(dsn) {
		if strings.HasPrefix(part, "host=") {
			return part[5:]
		}
	}
	return "localhost"
}

// RedisHost extracts the hostname from the Redis address.
func (c *AppConfig) RedisHost() string {
	if idx := strings.LastIndex(c.Redis.Addr, ":"); idx > 0 {
		return c.Redis.Addr[:idx]
	}
	return c.Redis.Addr
}

// MinIOHost extracts the hostname from the MinIO endpoint.
func (c *AppConfig) MinIOHost() string {
	if idx := strings.LastIndex(c.MinIO.Endpoint, ":"); idx > 0 {
		return c.MinIO.Endpoint[:idx]
	}
	return c.MinIO.Endpoint
}

func Load(path string) (*AppConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	cfg := &AppConfig{}
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, err
	}
	// Override sensitive values from environment variables
	applyEnvOverrides(cfg)
	return cfg, nil
}

func applyEnvOverrides(cfg *AppConfig) {
	// Env vars take precedence over config file values for security-critical settings
	if v := os.Getenv("DB_PASSWORD"); v != "" {
		// Replace password in DSN: host=... user=... password=OLD → password=ENV
		cfg.Database.DSN = overrideDSNPassword(cfg.Database.DSN, v)
	}
	if v := os.Getenv("JWT_ACCESS_SECRET"); v != "" {
		cfg.JWT.AccessSecret = v
	}
	if v := os.Getenv("JWT_REFRESH_SECRET"); v != "" {
		cfg.JWT.RefreshSecret = v
	}
	if v := os.Getenv("MINIO_SECRET_KEY"); v != "" {
		cfg.MinIO.SecretKey = v
	}
	if v := os.Getenv("MINIO_ACCESS_KEY"); v != "" {
		cfg.MinIO.AccessKey = v
	}
	if v := os.Getenv("REDIS_PASSWORD"); v != "" {
		cfg.Redis.Password = v
	}
	if v := os.Getenv("SMTP_PASSWORD"); v != "" {
		cfg.SMTP.Password = v
	}
}

func overrideDSNPassword(dsn, newPass string) string {
	fields := strings.Fields(dsn)
	for i, f := range fields {
		if strings.HasPrefix(f, "password=") {
			fields[i] = "password=" + newPass
		}
	}
	return strings.Join(fields, " ")
}
