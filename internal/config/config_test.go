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

func TestYamlAllowedDomainsScalar(t *testing.T) {
	cfg := defaults()
	if err := yaml.Unmarshal([]byte("allowed_domains: \"*\"\n"), &cfg); err != nil {
		t.Fatalf("yaml scalar: %v", err)
	}
	if len(cfg.Allowed) != 1 || cfg.Allowed[0] != "*" {
		t.Errorf("scalar '*' should normalize to [\"*\"]; got %v", cfg.Allowed)
	}
}

func TestYamlAllowedDomainsScalarCSV(t *testing.T) {
	cfg := defaults()
	if err := yaml.Unmarshal([]byte("allowed_domains: \"a.com, b.com\"\n"), &cfg); err != nil {
		t.Fatalf("yaml csv: %v", err)
	}
	if len(cfg.Allowed) != 2 || cfg.Allowed[0] != "a.com" || cfg.Allowed[1] != "b.com" {
		t.Errorf("csv should split+trim; got %v", cfg.Allowed)
	}
}

func TestYamlAllowedDomainsList(t *testing.T) {
	cfg := defaults()
	if err := yaml.Unmarshal([]byte("allowed_domains:\n  - \"a.com\"\n  - \"b.com\"\n"), &cfg); err != nil {
		t.Fatalf("yaml list: %v", err)
	}
	if len(cfg.Allowed) != 2 || cfg.Allowed[0] != "a.com" || cfg.Allowed[1] != "b.com" {
		t.Errorf("list should pass through; got %v", cfg.Allowed)
	}
}

func TestYamlSigningKeys(t *testing.T) {
	cfg := defaults()
	if err := yaml.Unmarshal([]byte("auth:\n  signing_key: \"k\"\n"), &cfg); err != nil {
		t.Fatalf("yaml: %v", err)
	}
	if got := cfg.Auth.SigningKeysList(); len(got) != 1 || got[0] != "k" {
		t.Errorf("signing key from yaml = %v", got)
	}
}

func TestEnvOverrides(t *testing.T) {
	t.Setenv("PORT", "7777")
	t.Setenv("MAX_FILE_SIZE_MB", "7")
	t.Setenv("DEFAULT_QUALITY", "70")
	t.Setenv("API_KEY", "secret")
	t.Setenv("SIGNING_KEY", "k1")
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
	if cfg.Auth.SigningKey != "k1" {
		t.Errorf("signing_key = %s", cfg.Auth.SigningKey)
	}
	if got := cfg.Auth.SigningKeysList(); len(got) != 1 || got[0] != "k1" {
		t.Errorf("signing_keys_list = %v", got)
	}

	t.Setenv("SIGNING_KEYS", "k2,k3")
	applyEnv(&cfg)
	if got := cfg.Auth.SigningKeysList(); len(got) != 2 || got[0] != "k2" || got[1] != "k3" {
		t.Errorf("signing_keys_list after env list = %v", got)
	}
	if len(cfg.Allowed) != 2 || cfg.Allowed[0] != "a.com" {
		t.Errorf("allowed = %v", cfg.Allowed)
	}
}