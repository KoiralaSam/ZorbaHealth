package errors

import "errors"

var (
	ErrHospitalNotFound   = errors.New("hospital not found")
	ErrPatientNotFound    = errors.New("patient not found")
	ErrInvalidPeriod      = errors.New("invalid period — use 7d, 30d, or 90d")
	ErrInvalidGranularity = errors.New("invalid granularity — use day or week")
	ErrUnauthorized       = errors.New("unauthorized")
)
