package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/url"
	"strings"

	"bd_back_for_translate_app/models"

	"github.com/jackc/pgx/v4/pgxpool"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func OpenTestDB(cfg Config) (*gorm.DB, error) {
	testDBName, err := databaseNameFromURL(cfg.TestDatabaseURL)
	if err != nil {
		return nil, err
	}
	if err := guardTestDatabaseName(testDBName, cfg); err != nil {
		return nil, err
	}

	db, err := gorm.Open(postgres.Open(cfg.TestDatabaseURL), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}
	if err := sqlDB.Ping(); err != nil {
		return nil, err
	}

	log.Printf("[тестовый контур] подключение к тестовой базе данных выполнено: %s", testDBName)
	return db, nil
}

func CloseDB(db *gorm.DB) {
	if db == nil {
		return
	}
	sqlDB, err := db.DB()
	if err == nil {
		_ = sqlDB.Close()
	}
}

func EnsureTestDatabaseExists(cfg Config) error {
	testDBName, err := databaseNameFromURL(cfg.TestDatabaseURL)
	if err != nil {
		return err
	}
	if err := guardTestDatabaseName(testDBName, cfg); err != nil {
		return err
	}

	adminURL, err := replaceDBName(cfg.TestDatabaseURL, "postgres")
	if err != nil {
		return err
	}

	ctx := context.Background()

	pool, err := pgxpool.Connect(ctx, adminURL)
	if err != nil {
		return err
	}
	defer pool.Close()

	var exists bool
	if err := pool.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM pg_database WHERE datname = $1)", testDBName).Scan(&exists); err != nil {
		return err
	}
	if exists {
		log.Printf("[тестовый контур] тестовая база данных уже существует: %s", testDBName)
		return nil
	}

	query := fmt.Sprintf(`CREATE DATABASE "%s"`, escapeIdentifier(testDBName))
	if _, err := pool.Exec(ctx, query); err != nil {
		return err
	}

	log.Printf("[тестовый контур] тестовая база данных создана: %s", testDBName)
	return nil
}

func RebuildTestSchema(db *gorm.DB, cfg Config) error {
	testDBName, err := databaseNameFromURL(cfg.TestDatabaseURL)
	if err != nil {
		return err
	}
	if err := guardTestDatabaseName(testDBName, cfg); err != nil {
		return err
	}

	if err := DropTestSchema(db, cfg); err != nil {
		return err
	}
	if err := AutoMigrateTestSchema(db); err != nil {
		return fmt.Errorf("ошибка автоматической миграции тестовой схемы: %w", err)
	}
	return nil
}

func DropTestSchema(db *gorm.DB, cfg Config) error {
	testDBName, err := databaseNameFromURL(cfg.TestDatabaseURL)
	if err != nil {
		return err
	}
	if err := guardTestDatabaseName(testDBName, cfg); err != nil {
		return err
	}

	query := `
DROP TABLE IF EXISTS
	audit_log,
	sync_events,
	attempts,
	user_word_progress,
	words_meta_lang,
	words_meta_base,
	word_synonyms,
	words,
	categories,
	admin_users,
	users,
	content_snapshot_versions
CASCADE;
`
	if err := db.Exec(query).Error; err != nil {
		return fmt.Errorf("не удалось удалить устаревшую тестовую схему: %w", err)
	}

	log.Printf("[тестовый контур] устаревшие тестовые таблицы удалены в базе %s", testDBName)
	return nil
}

func AutoMigrateTestSchema(db *gorm.DB) error {
	return db.AutoMigrate(
		&models.Category{},
		&models.Word{},
		&models.WordMetaLang{},
		&models.WordMetaBase{},
		&models.WordSynonym{},
		&models.ContentSnapshotVersion{},
		&models.User{},
		&models.Attempt{},
		&models.UserWordProgress{},
		&models.SyncEvent{},
		&models.AdminUser{},
		&models.AuditLog{},
	)
}

func databaseNameFromURL(raw string) (string, error) {
	u, err := url.Parse(raw)
	if err != nil {
		return "", err
	}
	name := strings.TrimPrefix(u.Path, "/")
	if name == "" {
		return "", errors.New("в строке подключения не указано имя базы данных")
	}
	return name, nil
}

func replaceDBName(raw, newName string) (string, error) {
	u, err := url.Parse(raw)
	if err != nil {
		return "", err
	}
	u.Path = "/" + newName
	return u.String(), nil
}

func guardTestDatabaseName(testDBName string, cfg Config) error {
	if testDBName == "" {
		return errors.New("имя тестовой базы данных не указано")
	}
	if !strings.Contains(strings.ToLower(testDBName), "test") {
		return fmt.Errorf("отказ от использования нетестовой базы данных: %s", testDBName)
	}
	if cfg.PrimaryDatabaseURL != "" {
		primaryName, err := databaseNameFromURL(cfg.PrimaryDatabaseURL)
		if err == nil && primaryName == testDBName {
			if cfg.AllowSameDB && strings.Contains(strings.ToLower(testDBName), "test") {
				return nil
			}
			return fmt.Errorf("TEST_DATABASE_URL указывает на ту же базу данных, что и DATABASE_URL: %s", testDBName)
		}
	}
	return nil
}

func escapeIdentifier(s string) string {
	return strings.ReplaceAll(s, `"`, `""`)
}
