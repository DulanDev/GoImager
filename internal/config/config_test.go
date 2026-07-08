package config

import (
	"os"
	"path/filepath"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestDefaults(t *testing.T) {
	cfg := defaults()
	if cfg.Server.Port != "8080" {
		t.Errorf("port = %s", cfg.Server.Port)
	}
	if cfg.Server.MaxFileSizeMB != 20 {
		t.Errorf("max = %d", cfg.Server.MaxFileSizeMB)
	}
	if cfg.Quality.Default != 85 {
		t.Errorf("quality = %d", cfg.Quality.Default)
	}
}

func TestYamlOverride(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "goimager.yaml")
	if err := os.WriteFile(path, []byte("server:\n  port: 9090\n  max_file_size_mb: 5\n"), 0644); err != nil {
		t.Fatal(err)
	}
	cfg := defaults()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if err := yaml.Unmarshal(b, &cfg); err != nil {
		t.Fatalf("yaml: %v", err)
	}
	if cfg.Server.Port != "9090" {
		t.Errorf("port = %s", cfg.Server.Port)
	}
	if cfg.Server.MaxFileSizeMB != 5 {
		t.Errorf("max = %d", cfg.Server.MaxFileSizeMB)
	}
}

func TestEnvOverrides(t *testing.T) {
	t.Setenv("PORT", "7777")
	t.Setenv("MAX_FILE_SIZE_MB", "7")
	t.Setenv("DEFAULT_QUALITY", "70")
	t.Setenv("API_KEY", "secret")
	t.Setenv("ALLOWED_DOMAINS", "a.com,b.com")
	cfg := defaults()
	applyEnv(&cfg)
	if cfg.Server.Port != "7777" {
		t.Errorf("port = %s", cfg.Server.Port)
	}
	if cfg.Server.MaxFileSizeMB != 7 {
		t.Errorf("max = %d", cfg.Server.MaxFileSizeMB)
	}
	if cfg.Quality.Default != 70 {
		t.Errorf("quality = %d", cfg.Quality.Default)
	}
	if cfg.Auth.APIKey != "secret" {
		t.Errorf("apikey = %s", cfg.Auth.APIKey)
	}
	if len(cfg.Allowed) != 2 || cfg.Allowed[0] != "a.com" {
		t.Errorf("allowed = %v", cfg.Allowed)
	}
}