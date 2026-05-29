package config

import (
	"strings"
	"testing"
)

func TestValidateProductionRequiresStrongJWTSecret(t *testing.T) {
	cfg := defaultValidProductionConfig()
	cfg.JWT.Secret = "short"

	if err := cfg.Validate(); err == nil || !strings.Contains(err.Error(), "jwt.secret") {
		t.Fatalf("Validate() error = %v, want jwt.secret error", err)
	}
}

func TestValidateProductionRejectsWildcardCORS(t *testing.T) {
	cfg := defaultValidProductionConfig()
	cfg.CORS.AllowOrigins = []string{"*"}

	if err := cfg.Validate(); err == nil || !strings.Contains(err.Error(), "cors.allow_origins") {
		t.Fatalf("Validate() error = %v, want cors.allow_origins error", err)
	}
}

func TestValidateProductionRejectsDefaultSeedPasswords(t *testing.T) {
	cfg := defaultValidProductionConfig()
	cfg.Seed.Password = "admin123"

	if err := cfg.Validate(); err == nil || !strings.Contains(err.Error(), "seed.password") {
		t.Fatalf("Validate() error = %v, want seed.password error", err)
	}
}

func TestValidateRequiresPositiveStorageLimits(t *testing.T) {
	cfg := defaultValidProductionConfig()
	cfg.Storage.MaxFileSizeMB = 0

	if err := cfg.Validate(); err == nil || !strings.Contains(err.Error(), "storage.max_file_size") {
		t.Fatalf("Validate() error = %v, want storage.max_file_size error", err)
	}
}

func TestValidateRejectsDisabledRedis(t *testing.T) {
	cfg := defaultValidProductionConfig()
	cfg.Redis.Enable = false

	if err := cfg.Validate(); err == nil || !strings.Contains(err.Error(), "redis.enable") {
		t.Fatalf("Validate() error = %v, want redis.enable error", err)
	}
}

func defaultValidProductionConfig() *Config {
	return &Config{
		App:   AppConfig{Env: "production"},
		DB:    DBConfig{SSLMode: "require"},
		Redis: RedisConfig{Enable: true},
		JWT:   JWTConfig{Secret: "0123456789abcdef0123456789abcdef"},
		Storage: StorageConfig{
			ChunkSizeMB:   5,
			MaxFileSizeMB: 100,
		},
		CORS: CORSConfig{AllowOrigins: []string{"https://admin.example.com"}},
		Seed: SeedConfig{
			Username:     "admin",
			Password:     "changed-admin-password",
			RootPassword: "changed-root-password",
		},
	}
}
