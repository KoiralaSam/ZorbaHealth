package postgres

import (
	"errors"
	"testing"

	domainerrors "github.com/KoiralaSam/ZorbaHealth/services/appointment-service/internal/core/domain/errors"
	"github.com/jackc/pgx/v5/pgconn"
)

func TestMapPGErrorExclusionViolation(t *testing.T) {
	err := &pgconn.PgError{Code: "23P01"}
	mapped := mapPGError(err)
	if !errors.Is(mapped, domainerrors.ErrConflict) {
		t.Fatalf("expected ErrConflict, got %v", mapped)
	}
}

func TestMapPGErrorPassthrough(t *testing.T) {
	orig := errors.New("other")
	if mapPGError(orig) != orig {
		t.Fatal("expected passthrough")
	}
}
