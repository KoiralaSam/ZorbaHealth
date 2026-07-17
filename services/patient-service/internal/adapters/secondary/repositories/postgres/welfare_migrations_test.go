package postgres_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWelfareMigrationsAreDeterministic(t *testing.T) {
	root := filepath.Join("..", "..", "..", "..", "..", "..", "..", "migrations")
	// services/patient-service/internal/adapters/secondary/repositories/postgres -> repo root
	up16 := filepath.Join(root, "000016_welfare_checks.up.sql")
	down16 := filepath.Join(root, "000016_welfare_checks.down.sql")
	up17 := filepath.Join(root, "000017_ensure_welfare_checks.up.sql")
	down17 := filepath.Join(root, "000017_ensure_welfare_checks.down.sql")
	for _, path := range []string{up16, down16, up17, down17} {
		body, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		text := string(body)
		if strings.TrimSpace(text) == "" {
			t.Fatalf("%s is empty", path)
		}
	}
	upBody, _ := os.ReadFile(up16)
	if !strings.Contains(string(upBody), "welfare_check_requests") {
		t.Fatal("000016 up missing requests table")
	}
	if !strings.Contains(string(upBody), "'missed'") {
		t.Fatal("000016 up missing missed status")
	}
	if !strings.Contains(string(upBody), "WELFARE_CHECK_ANSWERED") {
		t.Fatal("000016 up missing answered audit event")
	}
}
