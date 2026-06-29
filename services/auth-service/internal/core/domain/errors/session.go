package errors

import "errors"

var (
	ErrInvalidRefreshToken = errors.New("invalid refresh token")
	ErrRefreshTokenReuse     = errors.New("refresh token reuse detected")
	ErrSessionRevoked        = errors.New("session revoked")
)
