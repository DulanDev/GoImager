package config

import (
	"net/url"
	"testing"
)

func TestIsAllowedWildcard(t *testing.T) {
	cfg := defaults()
	if !cfg.IsAllowed("anything.com") {
		t.Fatal("wildcard should allow all")
	}
}

func TestIsAllowedExplicit(t *testing.T) {
	cfg := Config{Allowed: []string{"cdn.example.com", "images.example.com"}}
	if !cfg.IsAllowed("cdn.example.com") {
		t.Fatal("cdn should be allowed")
	}
	if !cfg.IsAllowed("images.example.com") {
		t.Fatal("images should be allowed")
	}
	if cfg.IsAllowed("evil.com") {
		t.Fatal("evil.com should be blocked")
	}
}

func TestIsAllowedURL(t *testing.T) {
	cfg := Config{Allowed: []string{"cdn.example.com"}}
	u, _ := url.Parse("https://cdn.example.com/path/img.png")
	if !cfg.IsAllowedURL(u) {
		t.Fatal("url host should match")
	}
}

func TestIsAllowedEmpty(t *testing.T) {
	cfg := Config{Allowed: nil}
	if cfg.IsAllowed("anything.com") {
		t.Fatal("nil allowlist should block all")
	}
	cfg2 := Config{Allowed: []string{}}
	if cfg2.IsAllowed("anything.com") {
		t.Fatal("empty allowlist should block all")
	}
}

func TestIsAllowedHostWithPort(t *testing.T) {
	cfg := Config{Allowed: []string{"cdn.example.com"}}
	if cfg.IsAllowed("cdn.example.com:8080") {
		t.Fatal("host with port should not raw-match bare host entry")
	}
}

func TestIsAllowedURLWithPort(t *testing.T) {
	cfg := Config{Allowed: []string{"cdn.example.com"}}
	u, _ := url.Parse("https://cdn.example.com:8080/path/img.png")
	if !cfg.IsAllowedURL(u) {
		t.Fatal("IsAllowedURL should strip port via Hostname()")
	}
}

func TestIsAllowedURLNilSafe(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("IsAllowedURL(nil) panicked: %v", r)
		}
	}()
	cfg := Config{Allowed: []string{"cdn.example.com"}}
	_ = cfg.IsAllowedURL(nil)
}

func TestIsAllowedURLEmptyHost(t *testing.T) {
	cfg := Config{Allowed: []string{"cdn.example.com"}}
	u, _ := url.Parse("/relative/path/img.png")
	if cfg.IsAllowedURL(u) {
		t.Fatal("url with no host should be blocked")
	}
}

// TestEnvOverridesAvifRpm ensures env vars OPTIMIZER_AVIF_PATH and
// RATE_LIMIT_RPM override their respective config defaults.
func TestEnvOverridesAvifRpm(t *testing.T) {
	t.Setenv("OPTIMIZER_AVIF_PATH", "/custom/avifenc")
	t.Setenv("RATE_LIMIT_RPM", "120")
	cfg := defaults()
	applyEnv(&cfg)
	if cfg.Optimizer.AvifPath != "/custom/avifenc" {
		t.Errorf("avif path = %s", cfg.Optimizer.AvifPath)
	}
	if cfg.RateLimit.RPM != 120 {
		t.Errorf("rpm = %d", cfg.RateLimit.RPM)
	}
}