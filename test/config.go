package main

import (
	"errors"
	"flag"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type Config struct {
	TestDatabaseURL    string
	PrimaryDatabaseURL string

	TestActiveBankPath string
	SeedActiveBank     bool

	NewmanBin         string
	NewmanCollection  string
	NewmanEnvironment string
	NewmanReportDir   string

	TestAdminUsername string
	TestAdminEmail    string
	TestAdminPassword string

	EnsureDB      bool
	ResetOnly     bool
	SeedOnly      bool
	RunNewmanOnly bool
	SkipNewman    bool

	AllowSameDB bool

	AssertPreparedState bool
	MinActiveBankWords  int64
	MinMetaLangRows     int64
	MinMetaBaseRows     int64
	MinSynonymRows      int64
}

func LoadConfig() (Config, error) {
	var cfg Config

	flag.BoolVar(&cfg.ResetOnly, "reset-only", false, "сбросить тестовую базу данных и завершить работу")
	flag.BoolVar(&cfg.SeedOnly, "seed-only", false, "создать тестового администратора и завершить работу")
	flag.BoolVar(&cfg.RunNewmanOnly, "run-newman-only", false, "пропустить сброс и подготовку данных, запустить только Newman")
	flag.BoolVar(&cfg.SkipNewman, "skip-newman", false, "пропустить запуск Newman после сброса и подготовки данных")
	flag.Parse()

	cfg.TestDatabaseURL = strings.TrimSpace(os.Getenv("TEST_DATABASE_URL"))
	cfg.PrimaryDatabaseURL = strings.TrimSpace(os.Getenv("DATABASE_URL"))
	cfg.TestActiveBankPath = strings.TrimSpace(os.Getenv("TEST_ACTIVE_BANK_PATH"))

	cfg.NewmanBin = strings.TrimSpace(os.Getenv("NEWMAN_BIN"))
	cfg.NewmanCollection = strings.TrimSpace(os.Getenv("NEWMAN_COLLECTION"))
	cfg.NewmanEnvironment = strings.TrimSpace(os.Getenv("NEWMAN_ENVIRONMENT"))
	cfg.NewmanReportDir = strings.TrimSpace(os.Getenv("NEWMAN_REPORT_DIR"))

	cfg.TestAdminUsername = strings.TrimSpace(os.Getenv("TEST_ADMIN_USERNAME"))
	cfg.TestAdminEmail = strings.TrimSpace(os.Getenv("TEST_ADMIN_EMAIL"))
	cfg.TestAdminPassword = strings.TrimSpace(os.Getenv("TEST_ADMIN_PASSWORD"))

	cfg.EnsureDB = parseBoolEnv("TEST_ENSURE_DB")
	cfg.AllowSameDB = parseBoolEnv("TEST_ALLOW_SAME_DB")
	cfg.SeedActiveBank = parseBoolEnvDefault("TEST_SEED_ACTIVE_BANK", true)
	cfg.AssertPreparedState = parseBoolEnvDefault("TEST_ASSERT_PREPARED_STATE", true)
	cfg.MinActiveBankWords = parseInt64EnvDefault("TEST_MIN_ACTIVE_BANK_WORDS", 1)
	cfg.MinMetaLangRows = parseInt64EnvDefault("TEST_MIN_META_LANG_ROWS", 3)
	cfg.MinMetaBaseRows = parseInt64EnvDefault("TEST_MIN_META_BASE_ROWS", 1)
	cfg.MinSynonymRows = parseInt64EnvDefault("TEST_MIN_SYNONYM_ROWS", 1)

	if cfg.TestDatabaseURL == "" {
		return cfg, errors.New("переменная окружения TEST_DATABASE_URL не задана")
	}

	if cfg.TestAdminUsername == "" {
		cfg.TestAdminUsername = "admin"
	}
	if cfg.TestAdminEmail == "" {
		cfg.TestAdminEmail = "admin@example.com"
	}
	if cfg.TestAdminPassword == "" {
		cfg.TestAdminPassword = "admin"
	}

	if cfg.TestActiveBankPath == "" {
		cfg.TestActiveBankPath = filepath.ToSlash(filepath.Join("test", "fixtures", "active_bank_test.jsonl"))
	}

	if cfg.NewmanCollection == "" {
		cfg.NewmanCollection = filepath.ToSlash(filepath.Join("test", "Rugen_Lexicon_Handlers_detailed_asserts.postman_collection.json"))
	}
	if cfg.NewmanEnvironment == "" {
		cfg.NewmanEnvironment = filepath.ToSlash(filepath.Join("test", "Rugen_local.postman_environment.json"))
	}
	if cfg.NewmanReportDir == "" {
		cfg.NewmanReportDir = filepath.ToSlash(filepath.Join("test", "reports"))
	}

	return cfg, nil
}

func parseBoolEnv(name string) bool {
	v := strings.TrimSpace(strings.ToLower(os.Getenv(name)))
	return v == "1" || v == "true" || v == "yes" || v == "on"
}

func parseBoolEnvDefault(name string, fallback bool) bool {
	v := strings.TrimSpace(strings.ToLower(os.Getenv(name)))
	if v == "" {
		return fallback
	}
	return v == "1" || v == "true" || v == "yes" || v == "on"
}

func parseInt64EnvDefault(name string, fallback int64) int64 {
	v := strings.TrimSpace(os.Getenv(name))
	if v == "" {
		return fallback
	}
	parsed, err := strconv.ParseInt(v, 10, 64)
	if err != nil || parsed < 0 {
		return fallback
	}
	return parsed
}
