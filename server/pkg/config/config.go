package config

import (
	"os"

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
	JWT struct {
		AccessSecret  string `yaml:"access_secret"`
		RefreshSecret string `yaml:"refresh_secret"`
		AccessTTL     int    `yaml:"access_ttl_sec"`
		RefreshTTL    int    `yaml:"refresh_ttl_sec"`
	} `yaml:"jwt"`
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
	return cfg, nil
}
