package errors

import "errors"

var (
	ErrNoLocationFound    = errors.New("no location found for session")
	ErrNoActiveConnection = errors.New("no active websocket connection for patient")
	ErrInvalidCoordinates = errors.New("invalid coordinates")
	ErrUnauthorized       = errors.New("unauthorized")
)
