package main

import (
	"path/filepath"
	"testing"
)

func TestDatabaseNameFromURL(t *testing.T) {
	name, err := databaseNameFromURL("postgres://user:pass@localhost:5432/rugen_test?sslmode=disable")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if name != "rugen_test" {
		t.Fatalf("expected rugen_test, got %s", name)
	}
}

func TestGuardRejectsNonTestDatabaseName(t *testing.T) {
	err := guardTestDatabaseName("language_db", Config{})
	if err == nil {
		t.Fatalf("expected production-like database name to be rejected")
	}
}

func TestGuardAllowsTestDatabaseName(t *testing.T) {
	err := guardTestDatabaseName("language_db_test", Config{})
	if err != nil {
		t.Fatalf("expected test database name to be allowed: %v", err)
	}
}

func TestGuardRejectsSamePrimaryDatabaseUnlessExplicitlyAllowed(t *testing.T) {
	cfg := Config{PrimaryDatabaseURL: "postgres://user:pass@localhost:5432/language_db_test?sslmode=disable"}
	err := guardTestDatabaseName("language_db_test", cfg)
	if err == nil {
		t.Fatalf("expected same primary and test database to be rejected")
	}
}

func TestResolveActiveBankFixturePathUsesConfiguredExistingFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "active_bank_test.jsonl")
	writeErr := writeSmallFile(path, []byte("{}\n"))
	if writeErr != nil {
		t.Fatalf("failed to create fixture: %v", writeErr)
	}
	resolved, err := resolveActiveBankFixturePath(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resolved != path {
		t.Fatalf("expected %s, got %s", path, resolved)
	}
}
