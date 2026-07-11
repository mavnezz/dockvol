package config

import (
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/ilyakaznacheev/cleanenv"
	"github.com/joho/godotenv"

	env_utils "dockvol-backend/internal/util/env"
	"dockvol-backend/internal/util/logger"
)

var log = logger.GetLogger()

type EnvVariables struct {
	EnvMode env_utils.EnvMode `env:"ENV_MODE" required:"true"`

	DataFolder    string
	TempFolder    string
	SecretKeyPath string
	DatabasePath  string

	// oauth
	GitHubClientID     string `env:"GITHUB_CLIENT_ID"`
	GitHubClientSecret string `env:"GITHUB_CLIENT_SECRET"`
	GoogleClientID     string `env:"GOOGLE_CLIENT_ID"`
	GoogleClientSecret string `env:"GOOGLE_CLIENT_SECRET"`

	// Cloudflare Turnstile
	CloudflareTurnstileSecretKey string `env:"CLOUDFLARE_TURNSTILE_SECRET_KEY"`
	CloudflareTurnstileSiteKey   string `env:"CLOUDFLARE_TURNSTILE_SITE_KEY"`

	// SMTP configuration (optional)
	SMTPHost               string `env:"SMTP_HOST"`
	SMTPPort               int    `env:"SMTP_PORT"`
	SMTPUser               string `env:"SMTP_USER"`
	SMTPPassword           string `env:"SMTP_PASSWORD"`
	SMTPFrom               string `env:"SMTP_FROM"`
	SMTPInsecureSkipVerify bool   `env:"SMTP_INSECURE_SKIP_VERIFY"`

	// Application URL (optional) - used for email links
	DockVolURL string `env:"DOCKVOL_URL"`
}

var env EnvVariables

var initEnv = sync.OnceFunc(loadEnvVariables)

func GetEnv() *EnvVariables {
	initEnv()
	return &env
}

func loadEnvVariables() {
	cwd, err := os.Getwd()
	if err != nil {
		log.Warn("could not get current working directory", "error", err)
		cwd = "."
	}

	backendRoot := cwd
	for {
		if _, err := os.Stat(filepath.Join(backendRoot, "go.mod")); err == nil {
			break
		}

		parent := filepath.Dir(backendRoot)
		if parent == backendRoot {
			break
		}

		backendRoot = parent
	}

	envPath := filepath.Join(filepath.Dir(backendRoot), ".env")

	if err := godotenv.Load(envPath); err != nil {
		log.Info("no .env file found, using process environment", "path", envPath)
	} else {
		log.Info("loaded .env", "path", envPath)
	}

	// Empty values for non-string fields (e.g. SMTP_PORT=) crash cleanenv's
	// strconv parsing. Drop them so cleanenv falls back to the Go zero value.
	unsetEmptyEnvVars()

	if err := cleanenv.ReadEnv(&env); err != nil {
		log.Error("Configuration could not be loaded", "error", err)
		os.Exit(1)
	}

	if env.SMTPHost != "" && env.SMTPPort <= 0 {
		log.Error("SMTP_PORT must be a positive integer when SMTP_HOST is set", "value", env.SMTPPort)
		os.Exit(1)
	}

	if env.EnvMode != env_utils.EnvModeDevelopment && env.EnvMode != env_utils.EnvModeProduction {
		log.Error("ENV_MODE is invalid", "mode", env.EnvMode)
		os.Exit(1)
	}

	dataRoot := filepath.Join(filepath.Dir(backendRoot), "dockvol-data")
	env.DataFolder = filepath.Join(dataRoot, "backups")
	env.TempFolder = filepath.Join(dataRoot, "temp")
	env.SecretKeyPath = filepath.Join(dataRoot, "secret.key")
	env.DatabasePath = filepath.Join(dataRoot, "dockvol.db")

	log.Info("Environment variables loaded successfully!")
}

func unsetEmptyEnvVars() {
	for _, kv := range os.Environ() {
		key, value, ok := strings.Cut(kv, "=")
		if !ok {
			continue
		}

		if value == "" {
			_ = os.Unsetenv(key)
		}
	}
}
