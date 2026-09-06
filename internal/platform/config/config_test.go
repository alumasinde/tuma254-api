package config

import "testing"

func TestLoadRequiresMongoDBConfiguration(t *testing.T) {
	t.Setenv("MONGODB_URI", "")
	t.Setenv("MONGODB_DATABASE", "")

	if _, err := Load(); err == nil {
		t.Fatal("expected configuration error")
	}
}

func TestLoad(t *testing.T) {
	t.Setenv("MONGODB_URI", "mongodb://localhost:27017")
	t.Setenv("MONGODB_DATABASE", "tuma254_test")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.MongoDBDatabase != "tuma254_test" {
		t.Fatalf("unexpected database: %s", cfg.MongoDBDatabase)
	}
}
