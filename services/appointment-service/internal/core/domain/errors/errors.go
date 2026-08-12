package errors

import "errors"

var (
	ErrNotFound            = errors.New("not found")
	ErrForbidden           = errors.New("forbidden")
	ErrUnauthorized        = errors.New("unauthorized")
	ErrInvalidArgument     = errors.New("invalid argument")
	ErrConflict            = errors.New("conflict")
	ErrSlotUnavailable     = errors.New("slot unavailable")
	ErrConsentRequired     = errors.New("hospital consent required")
	ErrNoAvailability      = errors.New("no availability configured")
)
