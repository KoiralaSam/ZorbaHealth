package errors

import "errors"

var (
	ErrEmptyText           = errors.New("text must not be empty")
	ErrTextTooLong         = errors.New("text exceeds maximum allowed length")
	ErrUnsupportedLanguage = errors.New("language code is not supported")
	ErrTranslationFailed   = errors.New("translation provider returned an error")
	ErrProviderUnavailable = errors.New("translation provider is unavailable")
	ErrInvalidLanguageCode = errors.New("language code must be a valid ISO 639-1 code")
)
