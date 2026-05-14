package main

import (
	"log"

	"github.com/joho/godotenv"
)

func main() {
	loadEnv()

	cfg, err := LoadConfig()
	if err != nil {
		log.Fatalf("[тестовый контур] ошибка конфигурации: %v", err)
	}

	if cfg.EnsureDB {
		if err := EnsureTestDatabaseExists(cfg); err != nil {
			log.Fatalf("[тестовый контур] ошибка подготовки тестовой базы данных: %v", err)
		}
	}

	db, err := OpenTestDB(cfg)
	if err != nil {
		log.Fatalf("[тестовый контур] ошибка подключения к тестовой базе данных: %v", err)
	}
	defer CloseDB(db)

	if !cfg.RunNewmanOnly {
		if err := RebuildTestSchema(db, cfg); err != nil {
			log.Fatalf("[тестовый контур] ошибка пересоздания тестовой схемы: %v", err)
		}
		log.Println("[тестовый контур] тестовая схема пересоздана")
	}

	if cfg.ResetOnly || (!cfg.RunNewmanOnly && !cfg.SeedOnly) {
		if err := ResetTestDatabase(db, cfg); err != nil {
			log.Fatalf("[тестовый контур] ошибка сброса тестовых данных: %v", err)
		}
		log.Println("[тестовый контур] тестовые данные сброшены")
	}

	if cfg.SeedOnly || (!cfg.RunNewmanOnly) {
		if err := SeedAdminUser(db, cfg); err != nil {
			log.Fatalf("[тестовый контур] ошибка создания тестового администратора: %v", err)
		}
		log.Println("[тестовый контур] тестовый администратор создан")

		if cfg.SeedActiveBank {
			if err := SeedActiveBank(db, cfg); err != nil {
				log.Fatalf("[тестовый контур] ошибка подготовки active_bank: %v", err)
			}
			log.Println("[тестовый контур] active_bank подготовлен")
		}
	}

	if !cfg.ResetOnly && !cfg.SeedOnly {
		if err := AssertPreparedTestState(db, cfg); err != nil {
			log.Fatalf("[тестовый контур] ошибка проверки подготовленного состояния: %v", err)
		}
	}

	if cfg.ResetOnly || cfg.SeedOnly || cfg.SkipNewman {
		log.Println("[тестовый контур] запуск Newman пропущен")
		return
	}

	if err := RunNewman(cfg); err != nil {
		log.Fatalf("[тестовый контур] ошибка Newman: %v", err)
	}

	log.Println("[тестовый контур] тестовый сценарий завершён")
}

func loadEnv() {
	if err := godotenv.Load(); err != nil {
		log.Println("[тестовый контур] корневой .env не найден, используются переменные окружения")
	}

	if err := godotenv.Overload("test/.env.test"); err != nil {
		log.Println("[тестовый контур] test/.env.test не найден, используются уже загруженные переменные окружения")
	}
}
