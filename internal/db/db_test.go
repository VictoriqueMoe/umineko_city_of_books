package db

import (
	"testing"
)

func TestOpenAndMigrate(t *testing.T) {
	database, err := Open(":memory:")
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	defer database.Close()

	if err := Migrate(database); err != nil {
		t.Fatalf("failed to run migrations: %v", err)
	}

	var count int
	err = database.QueryRow("SELECT COUNT(*) FROM users").Scan(&count)
	if err != nil {
		t.Fatalf("failed to query users table: %v", err)
	}
	if count != 0 {
		t.Errorf("expected 0 users, got %d", count)
	}
}

func TestMigrateCreatesAllTables(t *testing.T) {
	database, err := Open(":memory:")
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	defer database.Close()

	if err := Migrate(database); err != nil {
		t.Fatalf("failed to run migrations: %v", err)
	}

	tables := []string{"users", "sessions", "theories", "theory_evidence", "responses", "response_evidence", "theory_votes", "response_votes"}
	for _, table := range tables {
		var name string
		err := database.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name=?", table).Scan(&name)
		if err != nil {
			t.Errorf("table %q not found: %v", table, err)
		}
	}
}
