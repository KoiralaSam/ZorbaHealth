package errors

import "errors"

var (
	ErrPatientIDRequired = errors.New("patient_id required")
	ErrQueryRequired     = errors.New("query required")
	ErrInvalidPatientID  = errors.New("invalid patient_id: ")
	ErrEmbedQueryFailed  = errors.New("embed query: ")

	ErrHospitalIDRequired = errors.New("hospital_id required")
	ErrInvalidHospitalID  = errors.New("invalid hospital_id: ")

	ErrFetchChunksFailed = errors.New("fetch chunks: ")
	ErrNoRecordsFound    = errors.New("no records found")

	ErrSourceFileRequired = errors.New("source_file required")
	ErrTextRequired        = errors.New("text required")
	ErrEmbedChunkFailed    = errors.New("embed chunk ")
	ErrStoreChunkFailed    = errors.New("store chunk ")

	ErrSessionIDRequired = errors.New("session_id required")
	ErrRoleRequired      = errors.New("role required")
	ErrContentRequired   = errors.New("content required")
	ErrEmbedFailed       = errors.New("embed: ")

	ErrLoadTurnsFailed = errors.New("load turns: ")

	ErrResourceTypeRequired = errors.New("resource_type required")
	ErrListResourcesFailed  = errors.New("list resources: ")

	// OpenAI adapter errors
	ErrEmbedEmptyInput     = errors.New("embed: empty input")
	ErrEmbedEmptyResponse  = errors.New("embed: empty response")

	ErrSummarizeNoChunksProvided = errors.New("summarize: no chunks provided")
	ErrSummarizeAllChunksEmpty   = errors.New("summarize: all chunks empty")
	ErrSummarizeFailed           = errors.New("summarize: ")
)

