package main

import (
	"log"
	"time"

	"github.com/joho/godotenv"
)

func main() {
	startedAt := time.Now()
	loadEnv()

	cfg, err := LoadConfig()
	if err != nil {
		log.Fatalf("[test] config failed: %v", err)
	}

	log.Printf("[test] start mode reset_only=%v seed_only=%v run_newman_only=%v skip_newman=%v newman_suite=%s", cfg.ResetOnly, cfg.SeedOnly, cfg.RunNewmanOnly, cfg.SkipNewman, cfg.NewmanSuite)

	if cfg.EnsureDB {
		runStep("ensure database", func() error {
			return EnsureTestDatabaseExists(cfg)
		})
	}

	db, err := OpenTestDB(cfg)
	if err != nil {
		log.Fatalf("[test] open database failed: %v", err)
	}
	defer CloseDB(db)

	if !cfg.RunNewmanOnly {
		runStep("rebuild schema", func() error {
			return RebuildTestSchema(db, cfg)
		})
	}

	if cfg.ResetOnly || (!cfg.RunNewmanOnly && !cfg.SeedOnly) {
		runStep("reset data", func() error {
			return ResetTestDatabase(db, cfg)
		})
	}

	if cfg.SeedOnly || !cfg.RunNewmanOnly {
		runStep("seed admin", func() error {
			return SeedAdminUser(db, cfg)
		})

		if cfg.SeedActiveBank {
			runStep("seed active bank", func() error {
				return SeedActiveBank(db, cfg)
			})
		}
	}

	if !cfg.ResetOnly && !cfg.SeedOnly {
		runStep("assert prepared state", func() error {
			return AssertPreparedTestState(db, cfg)
		})
	}

	if cfg.ResetOnly || cfg.SeedOnly || cfg.SkipNewman {
		log.Printf("[test] newman skipped total=%s", time.Since(startedAt).Round(time.Millisecond))
		return
	}

	runStep("newman", func() error {
		return RunNewman(cfg)
	})

	log.Printf("[test] completed total=%s", time.Since(startedAt).Round(time.Millisecond))
}

func runStep(name string, fn func() error) {
	startedAt := time.Now()
	log.Printf("[test] step=%s status=start", name)
	if err := fn(); err != nil {
		log.Fatalf("[test] step=%s status=failed elapsed=%s error=%v", name, time.Since(startedAt).Round(time.Millisecond), err)
	}
	log.Printf("[test] step=%s status=ok elapsed=%s", name, time.Since(startedAt).Round(time.Millisecond))
}

func loadEnv() {
	if err := godotenv.Load(); err != nil {
		log.Println("[test] root .env not found, existing environment is used")
	}

	if err := godotenv.Overload("test/.env.test"); err != nil {
		log.Println("[test] test/.env.test not found, existing environment is used")
	}
}
