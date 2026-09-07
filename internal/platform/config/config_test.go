package config

import "testing"

func TestLoadRequiresMongoDBConfiguration(t *testing.T) {
	t.Setenv("MONGODB_URI", "")
	t.Setenv("MONGODB_DATABASE", "")
	t.Setenv("JWT_SIGNING_KEY", "01234567890123456789012345678901")
	if _, err := Load(); err == nil { t.Fatal("expected configuration error") }
}

func TestLoadRequiresStrongJWTSigningKey(t *testing.T) {
	t.Setenv("MONGODB_URI", "mongodb://localhost:27017")
	t.Setenv("MONGODB_DATABASE", "tuma254_test")
	t.Setenv("JWT_SIGNING_KEY", "short")
	if _, err := Load(); err == nil { t.Fatal("expected configuration error") }
}

func TestLoad(t *testing.T) {
	t.Setenv("MONGODB_URI", "mongodb://localhost:27017")
	t.Setenv("MONGODB_DATABASE", "tuma254_test")
	t.Setenv("JWT_SIGNING_KEY", "01234567890123456789012345678901")
	cfg, err := Load()
	if err != nil { t.Fatalf("unexpected error: %v", err) }
	if cfg.MongoDBDatabase != "tuma254_test" { t.Fatalf("unexpected database: %s", cfg.MongoDBDatabase) }
	if cfg.AccessTokenTTL <= 0 || cfg.RefreshTokenTTL <= 0 { t.Fatal("expected positive token TTLs") }
}


func TestProductionRejectsUnsafeDefaults(t *testing.T) {
	t.Setenv("APP_ENV", "production")
	t.Setenv("MONGODB_URI", "mongodb://localhost:27017")
	t.Setenv("MONGODB_DATABASE", "tuma254")
	t.Setenv("JWT_SIGNING_KEY", "example-development-secret-key-0123456789")
	if _, err := Load(); err == nil {
		t.Fatal("expected unsafe production configuration to be rejected")
	}
}
