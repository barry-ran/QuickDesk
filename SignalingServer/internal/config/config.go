package config

import (
	"fmt"
	"log"
	"strings"

	"github.com/spf13/viper"
)

type IceConfig struct {
	TurnURLs      []string
	AuthSecret    string // shared secret with coturn (use-auth-secret mode)
	CredentialTTL int    // TURN credential TTL in seconds
	StunURLs      []string
}

type Config struct {
	Server   ServerConfig   `mapstructure:"server"`
	Database DatabaseConfig `mapstructure:"database"`
	Redis    RedisConfig    `mapstructure:"redis"`
	Ice      IceConfig
}

type ServerConfig struct {
	Host string `mapstructure:"host"`
	Port int    `mapstructure:"port"`
}

type DatabaseConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	User     string `mapstructure:"user"`
	Password string `mapstructure:"password"`
	DBName   string `mapstructure:"dbname"`
}

type RedisConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	Password string `mapstructure:"password"`
}

func Load() *Config {
	viper.SetConfigName(".env")
	viper.SetConfigType("env")
	viper.AddConfigPath(".")
	viper.AutomaticEnv()

	// Set defaults
	viper.SetDefault("SERVER_HOST", "0.0.0.0")
	viper.SetDefault("SERVER_PORT", 8000)
	viper.SetDefault("DB_HOST", "localhost")
	viper.SetDefault("DB_PORT", 5432)
	viper.SetDefault("DB_USER", "quickdesk")
	viper.SetDefault("DB_PASSWORD", "quickdesk123")
	viper.SetDefault("DB_NAME", "quickdesk")
	viper.SetDefault("REDIS_HOST", "localhost")
	viper.SetDefault("REDIS_PORT", 6379)
	viper.SetDefault("REDIS_PASSWORD", "")

	viper.SetDefault("TURN_URLS", "")
	viper.SetDefault("TURN_AUTH_SECRET", "")
	viper.SetDefault("TURN_CREDENTIAL_TTL", 86400)
	viper.SetDefault("STUN_URLS", "")

	// Read config file (optional, will use defaults if not exists)
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			log.Println("Config file not found, using defaults and environment variables")
		} else {
			log.Fatalf("Error reading config file: %v", err)
		}
	}

	cfg := &Config{
		Server: ServerConfig{
			Host: viper.GetString("SERVER_HOST"),
			Port: viper.GetInt("SERVER_PORT"),
		},
		Database: DatabaseConfig{
			Host:     viper.GetString("DB_HOST"),
			Port:     viper.GetInt("DB_PORT"),
			User:     viper.GetString("DB_USER"),
			Password: viper.GetString("DB_PASSWORD"),
			DBName:   viper.GetString("DB_NAME"),
		},
		Redis: RedisConfig{
			Host:     viper.GetString("REDIS_HOST"),
			Port:     viper.GetInt("REDIS_PORT"),
			Password: viper.GetString("REDIS_PASSWORD"),
		},
	}

	cfg.Ice = parseIceConfig()

	log.Printf("Loaded config: Server=%s:%d, DB=%s:%d/%s, ICE TURN=%d STUN=%d TTL=%ds",
		cfg.Server.Host, cfg.Server.Port,
		cfg.Database.Host, cfg.Database.Port, cfg.Database.DBName,
		len(cfg.Ice.TurnURLs), len(cfg.Ice.StunURLs), cfg.Ice.CredentialTTL)

	return cfg
}

func (c *DatabaseConfig) DSN() string {
	return fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		c.Host, c.Port, c.User, c.Password, c.DBName)
}

func (c *RedisConfig) Addr() string {
	return fmt.Sprintf("%s:%d", c.Host, c.Port)
}

func parseIceConfig() IceConfig {
	ice := IceConfig{
		CredentialTTL: viper.GetInt("TURN_CREDENTIAL_TTL"),
		AuthSecret:    viper.GetString("TURN_AUTH_SECRET"),
	}
	if urls := viper.GetString("TURN_URLS"); urls != "" {
		ice.TurnURLs = splitAndTrim(urls)
	}
	if urls := viper.GetString("STUN_URLS"); urls != "" {
		ice.StunURLs = splitAndTrim(urls)
	}
	return ice
}

func splitAndTrim(s string) []string {
	parts := strings.Split(s, ",")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			result = append(result, p)
		}
	}
	return result
}
