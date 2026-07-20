package config

import (
	"flag"
	"fmt"
	"log"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"
)

var envFile string

func init() {
	flag.StringVar(&envFile, "env", "", "Path to env config file (default: .env in current directory)")
}

type IceConfig struct {
	TurnURLs      []string
	AuthSecret    string // shared secret with coturn (use-auth-secret mode)
	CredentialTTL int    // TURN credential TTL in seconds
	StunURLs      []string
	MaxRateKbps   int // max bitrate (kbps) for TURN relayed connections; 0 means no cap
}

type AdminConfig struct {
	User     string
	Password string
}

type SecurityConfig struct {
	APIKey         string   // API Key for client authentication; empty means disabled
	AllowedOrigins []string // Allowed Origin domains for WebClient; empty means disabled
}

type SmsConfig struct {
	AccessKeyID     string
	AccessKeySecret string
	SignName        string
	TemplateCode    string
	Enabled         bool // true when all required fields are configured
}

type Config struct {
	Server   ServerConfig   `mapstructure:"server"`
	Database DatabaseConfig `mapstructure:"database"`
	Redis    RedisConfig    `mapstructure:"redis"`
	Ice      IceConfig
	Admin    AdminConfig
	Security SecurityConfig
	Sms      SmsConfig
	Runtime  RuntimeConfig
}

type RuntimeConfig struct {
	RunAutoMigrate bool
	PprofAddr      string
	LogDir         string
	LogMaxSizeMB   int
	LogMaxBackups  int
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
	if !flag.Parsed() {
		flag.Parse()
	}

	if envFile != "" {
		viper.SetConfigFile(envFile)
		viper.SetConfigType("env")
		log.Printf("Using config file: %s", envFile)
	} else {
		viper.SetConfigName(".env")
		viper.SetConfigType("env")
		viper.AddConfigPath(".")
	}
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
	viper.SetDefault("TURN_MAX_RATE_KBPS", 0)

	viper.SetDefault("ADMIN_USER", "admin")
	viper.SetDefault("ADMIN_PASSWORD", "admin")

	viper.SetDefault("API_KEY", "")
	viper.SetDefault("ALLOWED_ORIGINS", "")

	// Aliyun SMS (optional – leave empty to disable phone features)
	viper.SetDefault("ALIYUN_SMS_ACCESS_KEY_ID", "")
	viper.SetDefault("ALIYUN_SMS_ACCESS_KEY_SECRET", "")
	viper.SetDefault("ALIYUN_SMS_SIGN_NAME", "")
	viper.SetDefault("ALIYUN_SMS_TEMPLATE_CODE", "")

	viper.SetDefault("RUN_AUTO_MIGRATE", false)
	viper.SetDefault("PPROF_ADDR", "")
	viper.SetDefault("LOG_DIR", "logs")
	viper.SetDefault("LOG_MAX_SIZE_MB", 50)
	viper.SetDefault("LOG_MAX_BACKUPS", 5)

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			log.Println("Config file not found, using defaults and environment variables")
		} else {
			log.Fatalf("Error reading config file: %v", err)
		}
	} else {
		absPath, _ := filepath.Abs(viper.ConfigFileUsed())
		log.Printf("Loaded config from: %s", absPath)
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
	cfg.Admin = AdminConfig{
		User:     viper.GetString("ADMIN_USER"),
		Password: viper.GetString("ADMIN_PASSWORD"),
	}
	var allowedOrigins []string
	if origins := viper.GetString("ALLOWED_ORIGINS"); origins != "" {
		allowedOrigins = splitAndTrim(origins)
	}
	cfg.Security = SecurityConfig{
		APIKey:         viper.GetString("API_KEY"),
		AllowedOrigins: allowedOrigins,
	}

	// Aliyun SMS – enabled only when all four fields are set
	smsKeyID := viper.GetString("ALIYUN_SMS_ACCESS_KEY_ID")
	smsKeySecret := viper.GetString("ALIYUN_SMS_ACCESS_KEY_SECRET")
	smsSignName := viper.GetString("ALIYUN_SMS_SIGN_NAME")
	smsTemplate := viper.GetString("ALIYUN_SMS_TEMPLATE_CODE")
	smsEnabled := smsKeyID != "" && smsKeySecret != "" && smsSignName != "" && smsTemplate != ""
	cfg.Sms = SmsConfig{
		AccessKeyID:     smsKeyID,
		AccessKeySecret: smsKeySecret,
		SignName:        smsSignName,
		TemplateCode:    smsTemplate,
		Enabled:         smsEnabled,
	}
	cfg.Runtime = RuntimeConfig{
		RunAutoMigrate: viper.GetBool("RUN_AUTO_MIGRATE"),
		PprofAddr:      viper.GetString("PPROF_ADDR"),
		LogDir:         viper.GetString("LOG_DIR"),
		LogMaxSizeMB:   viper.GetInt("LOG_MAX_SIZE_MB"),
		LogMaxBackups:  viper.GetInt("LOG_MAX_BACKUPS"),
	}

	apiKeyStatus := "disabled"
	if cfg.Security.APIKey != "" {
		apiKeyStatus = "enabled"
	}
	log.Printf("Loaded config: Server=%s:%d, DB=%s:%d/%s, ICE TURN=%d STUN=%d TTL=%ds MaxRate=%dkbps, APIKey=%s, AllowedOrigins=%d, SMS=%v, AutoMigrate=%v, Pprof=%v, LogDir=%s",
		cfg.Server.Host, cfg.Server.Port,
		cfg.Database.Host, cfg.Database.Port, cfg.Database.DBName,
		len(cfg.Ice.TurnURLs), len(cfg.Ice.StunURLs), cfg.Ice.CredentialTTL, cfg.Ice.MaxRateKbps,
		apiKeyStatus, len(cfg.Security.AllowedOrigins), cfg.Sms.Enabled, cfg.Runtime.RunAutoMigrate, cfg.Runtime.PprofAddr != "", cfg.Runtime.LogDir)

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
		MaxRateKbps:   viper.GetInt("TURN_MAX_RATE_KBPS"),
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
