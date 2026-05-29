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
	SSLMode      string `mapstructure:"sslmode"`
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
	Driver        string             `mapstructure:"driver"` // local | pg | oss
	ChunkSizeMB   int                `mapstructure:"chunk_size"`
	MaxFileSizeMB int64              `mapstructure:"max_file_size"`
	Local         LocalStorageConfig `mapstructure:"local"`
	OSS           OSSConfig          `mapstructure:"oss"`
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
	Level  string        `mapstructure:"level"`
	Format string        `mapstructure:"format"` // console | json
	Output string        `mapstructure:"output"` // stdout | file | both
	Access LogFileConfig `mapstructure:"access"`
	App    LogFileConfig `mapstructure:"app"`
	Error  LogFileConfig `mapstructure:"error"`
}

// LogFileConfig holds per-file log rotation settings.
type LogFileConfig struct {
	Enabled    bool   `mapstructure:"enabled"`
	Path       string `mapstructure:"path"`
	MaxSize    int    `mapstructure:"max_size"`    // MB
	MaxBackups int    `mapstructure:"max_backups"` // file count
	MaxAge     int    `mapstructure:"max_age"`     // days
	Compress   bool   `mapstructure:"compress"`
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
	Username     string `mapstructure:"username"`
	Password     string `mapstructure:"password"`
	RootPassword string `mapstructure:"root_password"`
	RootEmail    string `mapstructure:"root_email"`
	Email        string `mapstructure:"email"`
	DisplayName  string `mapstructure:"display_name"`
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
	if err := cfg.Validate(); err != nil {
		return nil, err
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
	v.SetDefault("db.sslmode", "disable")
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
	v.SetDefault("jwt.access_expire", 3600)    // 1 hour
	v.SetDefault("jwt.refresh_expire", 604800) // 7 days

	// Storage
	v.SetDefault("storage.driver", "local")
	v.SetDefault("storage.chunk_size", 5)      // MB
	v.SetDefault("storage.max_file_size", 100) // MB
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
	v.SetDefault("log.output", "both")

	// Log - Access
	v.SetDefault("log.access.enabled", true)
	v.SetDefault("log.access.path", "logs/access.log")
	v.SetDefault("log.access.max_size", 200)
	v.SetDefault("log.access.max_backups", 7)
	v.SetDefault("log.access.max_age", 7)
	v.SetDefault("log.access.compress", true)

	// Log - App
	v.SetDefault("log.app.enabled", true)
	v.SetDefault("log.app.path", "logs/app.log")
	v.SetDefault("log.app.max_size", 100)
	v.SetDefault("log.app.max_backups", 7)
	v.SetDefault("log.app.max_age", 7)
	v.SetDefault("log.app.compress", true)

	// Log - Error
	v.SetDefault("log.error.enabled", true)
	v.SetDefault("log.error.path", "logs/error.log")
	v.SetDefault("log.error.max_size", 50)
	v.SetDefault("log.error.max_backups", 30)
	v.SetDefault("log.error.max_age", 30)
	v.SetDefault("log.error.compress", true)

	// CORS
	v.SetDefault("cors.allow_origins", []string{"http://localhost:3000"})
	v.SetDefault("cors.allow_methods", []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"})
	v.SetDefault("cors.allow_headers", []string{"Origin", "Content-Type", "Authorization"})

	// Rate Limit
	v.SetDefault("rate_limit.requests_per_minute", 60)

	// Seed
	v.SetDefault("seed.username", "admin")
	v.SetDefault("seed.password", "admin123")
	v.SetDefault("seed.root_password", "root123456")
	v.SetDefault("seed.root_email", "root@niko-admin.local")
	v.SetDefault("seed.email", "admin@example.com")
	v.SetDefault("seed.display_name", "管理员")

	// Explicit env bindings for important toggles.
	_ = v.BindEnv("redis.enable", "NIKO_REDIS_ENABLE")
	_ = v.BindEnv("seed.root_password", "NIKO_SEED_ROOT_PASSWORD")
	_ = v.BindEnv("seed.root_email", "NIKO_SEED_ROOT_EMAIL")

	// AutomaticEnv + SetEnvPrefix("NIKO") + SetEnvKeyReplacer(".", "_")
	// 已自动处理所有环境变量映射，例如 NIKO_DB_HOST → db.host
}

// Validate 校验配置是否满足运行安全要求，生产环境会启用更严格的检查。
func (c *Config) Validate() error {
	if c == nil {
		return fmt.Errorf("config is nil")
	}

	if c.Storage.ChunkSizeMB <= 0 {
		return fmt.Errorf("storage.chunk_size must be greater than 0")
	}
	if c.Storage.MaxFileSizeMB <= 0 {
		return fmt.Errorf("storage.max_file_size must be greater than 0")
	}
	if c.JWT.Secret == "" {
		return fmt.Errorf("jwt.secret must be configured")
	}

	if !c.Redis.Enable {
		return fmt.Errorf("redis.enable must be true because redis is required for jwt refresh tokens and access token blacklist")
	}

	if !isProduction(c.App.Env) {
		return nil
	}

	if len(c.JWT.Secret) < 32 {
		return fmt.Errorf("jwt.secret must be at least 32 characters in production")
	}
	for _, origin := range c.CORS.AllowOrigins {
		if origin == "*" {
			return fmt.Errorf("cors.allow_origins cannot contain wildcard in production")
		}
	}
	if strings.EqualFold(c.DB.SSLMode, "disable") || c.DB.SSLMode == "" {
		return fmt.Errorf("db.sslmode must not be disable in production")
	}
	if c.Seed.Password == "admin123" {
		return fmt.Errorf("seed.password must not use default value in production")
	}
	if c.Seed.RootPassword == "root123456" {
		return fmt.Errorf("seed.root_password must not use default value in production")
	}

	return nil
}

// isProduction 判断当前运行环境是否为生产环境。
func isProduction(env string) bool {
	env = strings.ToLower(strings.TrimSpace(env))
	return env == "prod" || env == "production"
}
