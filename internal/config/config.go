package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
	"gopkg.in/yaml.v3"
)

type Config struct {
	Server    Server     `yaml:"server"`
	Quality   Quality    `yaml:"quality"`
	Auth      Auth       `yaml:"auth"`
	RateLimit RateLimit  `yaml:"rate_limit"`
	Optimizer Optimizer  `yaml:"optimizer"`
	Logging   Logging    `yaml:"logging"`
	Allowed   []string   `yaml:"allowed_domains"`
}

type Server struct {
	Port          string `yaml:"port"`
	MaxFileSizeMB int    `yaml:"max_file_size_mb"`
	MaxDimension  int    `yaml:"max_dimension"`
}

type Quality struct {
	Default int `yaml:"default"`
}

type Auth struct {
	APIKey string `yaml:"api_key"`
}

type RateLimit struct {
	RPS int `yaml:"rps"`
	RPM int `yaml:"rpm"`
}

type Optimizer struct {
	PngquantPath string `yaml:"pngquant_path"`
	MozjpegPath  string `yaml:"mozjpeg_path"`
	CwebpPath    string `yaml:"cwebp_path"`
	AvifPath     string `yaml:"avif_path"`
}

type Logging struct {
	Level  string `yaml:"level"`
	Format string `yaml:"format"`
}

func defaults() Config {
	return Config{
		Server:    Server{Port: "8080", MaxFileSizeMB: 20, MaxDimension: 10000},
		Quality:   Quality{Default: 85},
		Auth:      Auth{APIKey: ""},
		RateLimit: RateLimit{RPS: 100},
		Optimizer: Optimizer{PngquantPath: "pngquant", MozjpegPath: "cjpeg", CwebpPath: "cwebp", AvifPath: "avifenc"},
		Logging:   Logging{Level: "info", Format: "json"},
		Allowed:   []string{"*"},
	}
}

func yamlPaths() []string {
	candidates := []string{"goimager.yaml"}
	if home, err := os.UserHomeDir(); err == nil {
		candidates = append(candidates, filepath.Join(home, ".goimager.yaml"))
	}
	candidates = append(candidates, "/etc/goimager/config.yaml")
	return candidates
}

func Load() (Config, error) {
	cfg := defaults()

	for _, p := range yamlPaths() {
		if b, err := os.ReadFile(p); err == nil {
			if err := yaml.Unmarshal(b, &cfg); err != nil {
				return cfg, fmt.Errorf("parse yaml %s: %w", p, err)
			}
			break
		}
	}

	_ = godotenv.Load()
	applyEnv(&cfg)
	return cfg, nil
}

func applyEnv(c *Config) {
	envStr("PORT", &c.Server.Port)
	envInt("MAX_FILE_SIZE_MB", &c.Server.MaxFileSizeMB)
	envInt("MAX_DIMENSION", &c.Server.MaxDimension)
	envInt("DEFAULT_QUALITY", &c.Quality.Default)
	envStr("API_KEY", &c.Auth.APIKey)
	envInt("RATE_LIMIT_RPS", &c.RateLimit.RPS)
	envInt("RATE_LIMIT_RPM", &c.RateLimit.RPM)
	envStr("LOG_LEVEL", &c.Logging.Level)
	envStr("LOG_FORMAT", &c.Logging.Format)
	envStr("OPTIMIZER_PNGQUANT_PATH", &c.Optimizer.PngquantPath)
	envStr("OPTIMIZER_MOZJPEG_PATH", &c.Optimizer.MozjpegPath)
	envStr("OPTIMIZER_CWEBP_PATH", &c.Optimizer.CwebpPath)
envStr("OPTIMIZER_AVIF_PATH", &c.Optimizer.AvifPath)
	if v, ok := os.LookupEnv("ALLOWED_DOMAINS"); ok {
		if v == "*" {
			c.Allowed = []string{"*"}
		} else if v != "" {
			c.Allowed = strings.Split(v, ",")
			for i := range c.Allowed {
				c.Allowed[i] = strings.TrimSpace(c.Allowed[i])
			}
		}
	}
}

func envStr(key string, dst *string) {
	if v, ok := os.LookupEnv(key); ok {
		*dst = v
	}
}

func envInt(key string, dst *int) {
	if v, ok := os.LookupEnv(key); ok {
		if n, err := strconv.Atoi(v); err == nil {
			*dst = n
		}
	}
}