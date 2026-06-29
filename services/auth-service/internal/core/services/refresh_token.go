package services

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
)

func newOpaqueRefreshToken() (plain string, hash []byte, err error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", nil, err
	}
	plain = base64.RawURLEncoding.EncodeToString(b)
	sum := sha256.Sum256([]byte(plain))
	return plain, sum[:], nil
}

func hashRefreshToken(plain string) ([]byte, error) {
	if plain == "" {
		return nil, fmt.Errorf("empty refresh token")
	}
	sum := sha256.Sum256([]byte(plain))
	return sum[:], nil
}
