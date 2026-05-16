// Package config provides configuration loading with priority:
// environment variables > .env > config.yaml > defaults.
package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/viper"
)

// Config holds all configuration for the application.
type Config struct {
	App       AppConfig       `mapstructure:"app"`
	DB        DBConfig        `mapstructure:"db"`
	Redis     RedisConfig     `mapstructure:"redis"`
	JWT       JWTConfig       `mapstructure:"jwt"`
	Storage   StorageConfig   `mapstructure:"storage"`
	Log       LogConfig       `mapstructure:"log"`
	CORS      CORSConfig      `mapstructure:"cors"`
	RateLimit RateLimitConfig `mapstructure:"rate_limit"`
	Seed      SeedConfig      `mapstructure:"seed"`
	Proxy     ProxyConfig     `mapstructure:"proxy"`
}

// AppConfig holds application-level settings.
type AppConfig struct {
	Name string `mapstructure:"name"`
	Port int    `mapstructure:"port"`
	Env  string `mapstructure:"env"`
}

// DBConfig holds database connection settings.
type DBConfig struct {
	Host         string `mapstructure:"host"`
	Port         int    `mapstructure:"port"`
	User         string `mapstructure:"user"`
	Password     string `mapstructure:"password"`
	Name         string `mapstructure:"name"`
	MaxOpenConns int    `mapstructure:"max_open_conns"`
	MaxIdleConns int    `mapstructure:"max_idle_conns"`
}

// RedisConfig holds Redis connection settings.
type RedisConfig struct {
	Enable   bool   `mapstructure:"enable"`
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	Password string `mapstructure:"password"`
	DB       int    `mapstructure:"db"`
}

// JWTConfig holds JWT authentication settings.
type JWTConfig struct {
	Secret           string `mapstructure:"secret"`
	Issuer           string `mapstructure:"issuer"`
	Audience         string `mapstructure:"audience"`
	AccessExpireSec  int    `mapstructure:"access_expire"`
	RefreshExpireSec int    `mapstructure:"refresh_expire"`
}

// StorageConfig holds file storage settings.
type StorageConfig struct {
	Driver      string               `mapstructure:"driver"` // local | pg | oss
	ChunkSizeMB int                  `mapstructure:"chunk_size"`
	Local       LocalStorageConfig   `mapstructure:"local"`
	OSS         OSSConfig            `mapstructure:"oss"`
}

// LocalStorageConfig holds local filesystem storage settings.
type LocalStorageConfig struct {
	UploadDir string `mapstructure:"upload_dir"`
	PublicURL string `mapstructure:"public_url"`
}

// OSSConfig holds object storage service settings.
type OSSConfig struct {
	Endpoint  string `mapstructure:"endpoint"`
	AccessKey string `mapstructure:"access_key"`
	SecretKey string `mapstructure:"secret_key"`
	Bucket    string `mapstructure:"bucket"`
	UseSSL    bool   `mapstructure:"use_ssl"`
}

// LogConfig holds logging settings.
type LogConfig struct {
	Level  string `mapstructure:"level"`
	Format string `mapstructure:"format"` // console | json
}

// CORSConfig holds CORS settings.
type CORSConfig struct {
	AllowOrigins []string `mapstructure:"allow_origins"`
	AllowMethods []string `mapstructure:"allow_methods"`
	AllowHeaders []string `mapstructure:"allow_headers"`
}

// RateLimitConfig holds rate limiting settings.
type RateLimitConfig struct {
	RequestsPerMinute int `mapstructure:"requests_per_minute"`
}

// SeedConfig holds default seed data settings.
type SeedConfig struct {
	Username    string `mapstructure:"username"`
	Password    string `mapstructure:"password"`
	Email       string `mapstructure:"email"`
	DisplayName string `mapstructure:"display_name"`
}

// ProxyConfig holds trusted proxy settings for secure header validation.
type ProxyConfig struct {
	TrustedProxies []string `mapstructure:"trusted_proxies"`
}

// Load reads configuration from files and environment variables.
// Priority: env > .env > config.yaml > defaults.
func Load() (*Config, error) {
	// Load .env into OS environment (doesn't overwrite existing vars).
	loadDotEnv(".env")

	v := viper.New()
	setDefaults(v)

	// Determine the environment (default to "development").
	env := os.Getenv("NIKO_APP_ENV")
	if env == "" {
		env = "development"
	}

	// Merge environment-specific config file (lowest priority).
	configPath := fmt.Sprintf("./configs/config.%s.yaml", env)
	if _, err := os.Stat(configPath); err == nil {
		v.SetConfigFile(configPath)
		_ = v.MergeInConfig()
	}

	// Map environment variables with NIKO_ prefix.
	v.SetEnvPrefix("NIKO")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("unmarshal config: %w", err)
	}

	return &cfg, nil
}

// loadDotEnv reads a .env file and sets variables in the OS environment.
// Existing OS environment variables are not overwritten.
func loadDotEnv(path string) {
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		// 去除行内注释
		if idx := strings.Index(value, "#"); idx != -1 {
			value = strings.TrimSpace(value[:idx])
		}
		value = strings.Trim(value, "\"'")
		if os.Getenv(key) == "" {
			os.Setenv(key, value)
		}
	}
}

// setDefaults registers default configuration values and binds environment variables.
func setDefaults(v *viper.Viper) {
	// App
	v.SetDefault("app.name", "niko-admin")
	v.SetDefault("app.port", 8080)
	v.SetDefault("app.env", "development")

	// DB
	v.SetDefault("db.host", "localhost")
	v.SetDefault("db.port", 5432)
	v.SetDefault("db.user", "postgres")
	v.SetDefault("db.password", "")
	v.SetDefault("db.name", "niko_admin")
	v.SetDefault("db.max_open_conns", 25)
	v.SetDefault("db.max_idle_conns", 5)

	// Redis
	v.SetDefault("redis.enable", true)
	v.SetDefault("redis.host", "localhost")
	v.SetDefault("redis.port", 6379)
	v.SetDefault("redis.password", "")
	v.SetDefault("redis.db", 0)

	// JWT
	v.SetDefault("jwt.secret", "")
	v.SetDefault("jwt.issuer", "niko-admin")
	v.SetDefault("jwt.audience", "niko-admin")
	v.SetDefault("jwt.access_expire", 3600)   // 1 hour
	v.SetDefault("jwt.refresh_expire", 604800) // 7 days

	// Storage
	v.SetDefault("storage.driver", "local")
	v.SetDefault("storage.chunk_size", 5) // MB
	v.SetDefault("storage.local.upload_dir", "./uploads")
	v.SetDefault("storage.local.public_url", "/uploads")
	v.SetDefault("storage.oss.endpoint", "")
	v.SetDefault("storage.oss.access_key", "")
	v.SetDefault("storage.oss.secret_key", "")
	v.SetDefault("storage.oss.bucket", "")
	v.SetDefault("storage.oss.use_ssl", true)

	// Log
	v.SetDefault("log.level", "info")
	v.SetDefault("log.format", "console")

	// CORS
	v.SetDefault("cors.allow_origins", []string{"http://localhost:3000"})
	v.SetDefault("cors.allow_methods", []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"})
	v.SetDefault("cors.allow_headers", []string{"Origin", "Content-Type", "Authorization"})

	// Rate Limit
	v.SetDefault("rate_limit.requests_per_minute", 60)

	// Seed
	v.SetDefault("seed.username", "admin")
	v.SetDefault("seed.password", "admin123")
	v.SetDefault("seed.email", "admin@example.com")
	v.SetDefault("seed.display_name", "管理员")

	// Explicit env bindings for important toggles.
	_ = v.BindEnv("redis.enable", "NIKO_REDIS_ENABLE")

	// AutomaticEnv + SetEnvPrefix("NIKO") + SetEnvKeyReplacer(".", "_")
	// 已自动处理所有环境变量映射，例如 NIKO_DB_HOST → db.host
}
