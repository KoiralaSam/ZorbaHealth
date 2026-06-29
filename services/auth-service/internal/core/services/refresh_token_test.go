package services

import "testing"

func TestHashRefreshTokenStable(t *testing.T) {
	plain := "test-opaque-token-value"
	h1, err := hashRefreshToken(plain)
	if err != nil {
		t.Fatal(err)
	}
	h2, err := hashRefreshToken(plain)
	if err != nil {
		t.Fatal(err)
	}
	if len(h1) != 32 || len(h2) != 32 {
		t.Fatalf("expected sha256 length 32, got %d and %d", len(h1), len(h2))
	}
	for i := range h1 {
		if h1[i] != h2[i] {
			t.Fatal("hash not stable")
		}
	}
}

func TestNewOpaqueRefreshTokenUnique(t *testing.T) {
	a, _, err := newOpaqueRefreshToken()
	if err != nil {
		t.Fatal(err)
	}
	b, _, err := newOpaqueRefreshToken()
	if err != nil {
		t.Fatal(err)
	}
	if a == b || a == "" {
		t.Fatal("expected unique non-empty tokens")
	}
}
